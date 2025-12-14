// Package tui provides the terminal user interface for k1s.
// This file contains all data loading commands that fetch information
// from the Kubernetes cluster asynchronously using tea.Cmd.
package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/internal/adapters/repository"
)

// loadInitialData fetches the initial data required for the application startup.
// It retrieves the list of namespaces and nodes from the cluster.
// This is used when the application starts without a specific namespace flag.
// Returns a loadedMsg with namespaces and nodes, or an error if namespace listing fails.
func (m *Model) loadInitialData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		namespaces, err := m.k8sClient.ListNamespaces(ctx)
		if err != nil {
			return loadedMsg{err: err}
		}

		nodes, _ := repository.ListNodes(ctx, m.k8sClient.Clientset())

		return loadedMsg{
			namespaces: namespaces,
			nodes:      nodes,
		}
	}
}

// loadInitialDataWithResources fetches initial data along with namespace resources.
// This is used when the application starts with the -n flag to go directly to resources view.
// It retrieves namespaces, nodes, pods, configmaps, and secrets for the specified namespace.
// Returns an initialResourcesLoadedMsg with all data, or an error if critical operations fail.
func (m *Model) loadInitialDataWithResources() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		namespaces, err := m.k8sClient.ListNamespaces(ctx)
		if err != nil {
			return initialResourcesLoadedMsg{err: err}
		}

		nodes, _ := repository.ListNodes(ctx, m.k8sClient.Clientset())

		// Load resources for the specified namespace
		pods, err := repository.ListAllPods(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace())
		if err != nil {
			return initialResourcesLoadedMsg{err: err}
		}
		configmaps, _ := repository.ListConfigMaps(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace())
		secrets, _ := repository.ListSecrets(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace())

		return initialResourcesLoadedMsg{
			namespaces: namespaces,
			nodes:      nodes,
			pods:       pods,
			configmaps: configmaps,
			secrets:    secrets,
		}
	}
}

// loadWorkloads fetches all workloads of the currently selected resource type.
// The resource type (Deployments, StatefulSets, DaemonSets, Jobs, CronJobs)
// is determined by the navigator's current selection.
// Also refreshes the namespace list for the selector.
// Returns a loadedMsg with workloads and namespaces.
func (m *Model) loadWorkloads() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		workloads, err := repository.ListWorkloads(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace(), m.navigator.ResourceType())
		if err != nil {
			return loadedMsg{err: err}
		}

		namespaces, _ := m.k8sClient.ListNamespaces(ctx)

		return loadedMsg{
			workloads:  workloads,
			namespaces: namespaces,
		}
	}
}

// loadPods fetches all pods belonging to a specific workload.
// It uses label selectors to find pods managed by the workload.
// Also loads ConfigMaps and Secrets for the namespace to populate the resources view.
// Returns a resourcesLoadedMsg with pods, configmaps, and secrets.
func (m *Model) loadPods(workload *repository.WorkloadInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pods, err := repository.GetWorkloadPods(ctx, m.k8sClient.Clientset(), *workload)
		if err != nil {
			return resourcesLoadedMsg{err: err}
		}
		// Also load ConfigMaps and Secrets
		configmaps, _ := repository.ListConfigMaps(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace())
		secrets, _ := repository.ListSecrets(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace())
		return resourcesLoadedMsg{pods: pods, configmaps: configmaps, secrets: secrets}
	}
}

// loadAllResources fetches all pods, configmaps, and secrets in the current namespace.
// When no pods are found, it also tries to find the first scalable workload
// (Deployment, StatefulSet, or Argo Rollout) to enable scale controls.
// This allows users to scale up workloads even when no pods are running.
// Returns a resourcesLoadedMsg with all resources and optional workload for scaling.
func (m *Model) loadAllResources() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		ns := m.k8sClient.Namespace()
		pods, err := repository.ListAllPods(ctx, m.k8sClient.Clientset(), ns)
		if err != nil {
			return resourcesLoadedMsg{err: err}
		}
		configmaps, _ := repository.ListConfigMaps(ctx, m.k8sClient.Clientset(), ns)
		secrets, _ := repository.ListSecrets(ctx, m.k8sClient.Clientset(), ns)

		// Fetch first scalable workload for scale controls when pods = 0
		var workload *repository.WorkloadInfo
		if len(pods) == 0 {
			// Try deployments first
			deployments, _ := repository.ListWorkloads(ctx, m.k8sClient.Clientset(), ns, repository.ResourceDeployments)
			if len(deployments) > 0 {
				workload = &deployments[0]
			} else {
				// Try statefulsets
				statefulsets, _ := repository.ListWorkloads(ctx, m.k8sClient.Clientset(), ns, repository.ResourceStatefulSets)
				if len(statefulsets) > 0 {
					workload = &statefulsets[0]
				}
			}
			// Try Argo Rollouts via dynamic client
			if workload == nil && m.k8sClient.DynamicClient() != nil {
				rollouts, _ := repository.ListRollouts(ctx, m.k8sClient.DynamicClient(), ns)
				if len(rollouts) > 0 {
					workload = &rollouts[0]
				}
			}
		}

		return resourcesLoadedMsg{pods: pods, configmaps: configmaps, secrets: secrets, workload: workload}
	}
}

// loadConfigMapData fetches the full data of a specific ConfigMap.
// This is called when user selects a ConfigMap to view its contents.
// Returns a configMapDataMsg with the ConfigMap data including all keys and values.
func (m *Model) loadConfigMapData(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := repository.GetConfigMap(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace(), name)
		if err != nil {
			return configMapDataMsg{err: err}
		}
		return configMapDataMsg{data: data}
	}
}

// loadSecretData fetches the full data of a specific Secret.
// This is called when user selects a Secret or Docker Registry secret to view.
// The secret data is automatically base64 decoded for display.
// Returns a secretDataMsg with the decoded secret data.
func (m *Model) loadSecretData(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := repository.GetSecret(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace(), name)
		if err != nil {
			return secretDataMsg{err: err}
		}
		return secretDataMsg{data: data}
	}
}

// loadPodsByNode fetches all pods running on a specific node.
// This is used when user selects a node in the namespace/nodes view.
// Returns a nodePodLoadedMsg with the node name and list of pods on that node.
func (m *Model) loadPodsByNode(nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pods, err := repository.ListPodsByNode(ctx, m.k8sClient.Clientset(), nodeName)
		if err != nil {
			return nodePodLoadedMsg{nodeName: nodeName, err: err}
		}
		return nodePodLoadedMsg{nodeName: nodeName, pods: pods}
	}
}

// loadDashboardData fetches all data required for the pod dashboard view.
// This includes: refreshed pod status, container logs, events, metrics,
// related resources (services, ingresses, Istio resources), debug helpers,
// and node information.
// Returns a dashboardDataMsg with all dashboard components.
func (m *Model) loadDashboardData(pod *repository.PodInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Refresh pod info for real-time status updates
		updatedPod, _ := repository.GetPod(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name)
		if updatedPod == nil {
			updatedPod = pod
		}

		logs, _ := repository.GetAllContainerLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, 200)
		events, _ := repository.GetPodEvents(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name)
		metrics, _ := repository.GetPodMetrics(ctx, m.k8sClient.MetricsClient(), pod.Namespace, pod.Name)
		related, _ := repository.GetRelatedResources(ctx, m.k8sClient.Clientset(), m.k8sClient.DynamicClient(), *updatedPod)

		helpers := repository.AnalyzePodIssues(updatedPod, events)

		// Get node info for the pod's node
		var node *repository.NodeInfo
		if updatedPod.Node != "" {
			node, _ = repository.GetNode(ctx, m.k8sClient.Clientset(), updatedPod.Node)
		}

		return dashboardDataMsg{
			pod:     updatedPod,
			logs:    logs,
			events:  events,
			metrics: metrics,
			related: related,
			helpers: helpers,
			node:    node,
		}
	}
}

// loadLogsForState fetches logs based on the current dashboard state.
// It handles three scenarios:
// - Previous logs: fetches logs from a previous container instance (crashed/restarted)
// - Specific container: fetches logs from a selected container in multi-container pods
// - All containers: fetches logs from all containers when no specific one is selected
// Returns a logsUpdatedMsg with the fetched log lines.
func (m *Model) loadLogsForState(pod *repository.PodInfo, container string, previous bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var logs []repository.LogLine
		var err error

		if previous {
			// Get previous logs for specific container or first container
			targetContainer := container
			if targetContainer == "" && len(pod.Containers) > 0 {
				targetContainer = pod.Containers[0].Name
			}
			if targetContainer != "" {
				logs, err = repository.GetPreviousLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, targetContainer, 200)
			}
		} else if container != "" {
			// Get logs for specific container
			opts := repository.LogOptions{
				Container:  container,
				TailLines:  200,
				Timestamps: true,
			}
			logs, err = repository.GetPodLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, opts)
		} else {
			// Get all container logs
			logs, err = repository.GetAllContainerLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, 200)
		}

		if err != nil {
			return logsUpdatedMsg{logs: []repository.LogLine{{Content: "Error fetching logs: " + err.Error(), IsError: true}}}
		}

		return logsUpdatedMsg{logs: logs}
	}
}

// filteredNodes returns the list of nodes filtered by the current search query.
// If no search query is set, returns all nodes.
// The search is case-insensitive and matches against node names.
func (m Model) filteredNodes() []repository.NodeInfo {
	if m.nodeSearchQuery == "" {
		return m.nodes
	}
	query := strings.ToLower(m.nodeSearchQuery)
	var filtered []repository.NodeInfo
	for _, node := range m.nodes {
		if strings.Contains(strings.ToLower(node.Name), query) {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

// tickCmd creates a command that sends a tickMsg after the configured refresh interval.
// This is used for automatic dashboard refresh to keep logs and status up to date.
// The interval is configured in the application config (default: 5 seconds).
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(m.config.RefreshInterval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// clearStatusAfter creates a command that clears the status message after a duration.
// This is used to show temporary status messages (success/error) that auto-dismiss.
// Returns a clearStatusMsg after the specified duration.
func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

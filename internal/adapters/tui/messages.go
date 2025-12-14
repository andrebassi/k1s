// Package tui provides the terminal user interface for k1s.
// This file contains all message types used for communication between
// bubbletea components via the tea.Msg interface.
package tui

import (
	"time"

	"github.com/andrebassi/k1s/internal/adapters/repository"
)

// loadedMsg is sent when initial data loading completes.
// Contains namespace list, node list, and optionally workload list.
// Used during application startup and namespace/resource refresh.
type loadedMsg struct {
	workloads  []repository.WorkloadInfo // Workloads for current view (Deployments, StatefulSets, etc.)
	namespaces []string                  // Available namespaces in the cluster
	nodes      []repository.NodeInfo     // Cluster nodes with status and resource info
	err        error                     // Error if data loading failed
}

// resourcesLoadedMsg is sent when namespace resources are loaded.
// Contains pods, configmaps, and secrets for the selected namespace.
// Also includes the first scalable workload when no pods exist (for scale-up feature).
type resourcesLoadedMsg struct {
	pods       []repository.PodInfo       // Pods in the namespace (all or filtered by workload)
	configmaps []repository.ConfigMapInfo // ConfigMaps in the namespace
	secrets    []repository.SecretInfo    // Secrets in the namespace
	workload   *repository.WorkloadInfo   // First scalable workload for scale controls when pods=0
	err        error                      // Error if resource loading failed
}

// dashboardDataMsg is sent when pod dashboard data is ready.
// Contains all information needed to render the 4-panel pod debugging dashboard:
// logs, events, metrics, related resources, debug helpers, and node info.
type dashboardDataMsg struct {
	pod     *repository.PodInfo         // Updated pod information with current status
	logs    []repository.LogLine        // Container logs (last N lines from all containers)
	events  []repository.EventInfo      // Pod events (warnings and normal events)
	metrics *repository.PodMetrics      // CPU/Memory usage metrics from metrics-server
	related *repository.RelatedResources // Related Services, Ingresses, VirtualServices, Gateways
	helpers []repository.DebugHelper    // Debug hints based on pod state analysis
	node    *repository.NodeInfo        // Node information where pod is running
}

// logsUpdatedMsg is sent when container logs are refreshed.
// Used for log refresh operations (specific container, previous logs, time filter).
type logsUpdatedMsg struct {
	logs []repository.LogLine // Updated log lines
}

// podDeletedMsg is sent when a pod deletion operation completes.
// Contains the result of the delete operation (success or error).
type podDeletedMsg struct {
	namespace string // Namespace where the pod was deleted
	podName   string // Name of the deleted pod
	err       error  // Error if deletion failed (nil on success)
}

// workloadActionMsg is sent when a workload action (scale/restart) completes.
// Contains the result of the operation and details about the workload affected.
type workloadActionMsg struct {
	action       string                  // Action performed: "scale" or "restart"
	workloadName string                  // Name of the workload
	namespace    string                  // Namespace of the workload
	resourceType repository.ResourceType // Type: Deployment, StatefulSet, etc.
	replicas     int32                   // New replica count (only for scale action)
	err          error                   // Error if action failed (nil on success)
}

// tickMsg is sent periodically for automatic dashboard refresh.
// The time value indicates when the tick was generated.
type tickMsg time.Time

// clearStatusMsg is sent to clear the status message after a delay.
// Used to auto-dismiss success/error messages in the status bar.
type clearStatusMsg struct{}

// configMapDataMsg is sent when a ConfigMap's data is fetched.
// Contains the full ConfigMap data with all keys and values.
type configMapDataMsg struct {
	data *repository.ConfigMapData // ConfigMap data including all keys and values
	err  error                     // Error if fetch failed
}

// secretDataMsg is sent when a Secret's data is fetched.
// Contains the decoded (base64) secret data with all keys and values.
type secretDataMsg struct {
	data *repository.SecretData // Secret data (decoded) including all keys and values
	err  error                  // Error if fetch failed
}

// nodePodLoadedMsg is sent when pods for a specific node are loaded.
// Used when user selects a node to see all pods running on that node.
type nodePodLoadedMsg struct {
	nodeName string               // Name of the node
	pods     []repository.PodInfo // Pods running on the node
	err      error                // Error if loading failed
}

// initialResourcesLoadedMsg is sent when initial data with resources is loaded.
// Used when application starts with -n flag to go directly to resources view.
// Contains both cluster-level data (namespaces, nodes) and namespace resources.
type initialResourcesLoadedMsg struct {
	namespaces []string                   // Available namespaces in the cluster
	nodes      []repository.NodeInfo      // Cluster nodes with status info
	pods       []repository.PodInfo       // Pods in the specified namespace
	configmaps []repository.ConfigMapInfo // ConfigMaps in the namespace
	secrets    []repository.SecretInfo    // Secrets in the namespace
	err        error                      // Error if loading failed
}

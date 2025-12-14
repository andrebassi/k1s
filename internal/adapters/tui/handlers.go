// Package tui provides the terminal user interface for k1s.
// This file contains navigation and input handlers for user interactions.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/internal/adapters/tui/component"
)

// handleBack handles the escape/back action for navigation.
// Returns to the previous view or mode based on current state:
// - From Dashboard: Returns to Navigator in Resources mode
// - From Resources mode: Returns to Namespace selection
// - From ResourceType mode: Returns to Namespace selection
// - From Namespace mode: No action (root level)
func (m *Model) handleBack() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewDashboard:
		m.view = ViewNavigator
		m.pod = nil
		// Always go back to pods list
		m.navigator.SetMode(component.ModeResources)
		return m, nil

	case ViewNavigator:
		switch m.navigator.Mode() {
		case component.ModeResources:
			// Go back to namespace selection
			m.navigator.SetMode(component.ModeNamespace)
			m.workload = nil
			m.selectedNode = "" // Clear node filter
			return m, nil
		case component.ModeNamespace:
			// Stay in namespace selection (no back action)
			return m, nil
		case component.ModeResourceType:
			m.navigator.SetMode(component.ModeNamespace)
			return m, nil
		}
	}
	return m, nil
}

// handleEnter handles the enter key action based on current view and selection.
// Behavior varies by view and mode:
//
// Navigator View:
//   - ModeWorkloads: Loads pods for selected workload
//   - ModeResources/Pods: Opens pod dashboard with logs, events, metrics
//   - ModeResources/ConfigMaps: Loads ConfigMap data for viewing
//   - ModeResources/Secrets: Loads Secret data for viewing
//   - ModeResources/DockerRegistry: Loads Docker Registry secret for viewing
//   - ModeNamespace (nodes active): Loads pods running on selected node
//   - ModeNamespace (default): Selects namespace and loads resources
//   - ModeResourceType: Selects resource type and loads workloads
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewNavigator:
		switch m.navigator.Mode() {
		case component.ModeWorkloads:
			workload := m.navigator.SelectedWorkload()
			if workload != nil {
				m.workload = workload
				m.loading = true
				return m, m.loadPods(workload)
			}

		case component.ModeResources:
			switch m.navigator.Section() {
			case component.SectionPods:
				pod := m.navigator.SelectedPod()
				if pod != nil {
					m.pod = pod
					m.view = ViewDashboard
					m.dashboard.SetPod(pod)
					// Set breadcrumb: namespace > pods > podname
					workloadName := ""
					if m.workload != nil {
						workloadName = m.workload.Name
					}
					m.dashboard.SetBreadcrumb(
						m.k8sClient.Namespace(),
						"pods",
						workloadName,
						pod.Name,
					)
					m.dashboard.SetContext(m.k8sClient.Context())
					m.dashboard.SetNamespace(m.k8sClient.Namespace())
					m.loading = true
					return m, tea.Batch(
						m.loadDashboardData(pod),
						m.tickCmd(),
					)
				}
			case component.SectionConfigMaps:
				cm := m.navigator.SelectedConfigMap()
				if cm != nil {
					m.loading = true
					return m, m.loadConfigMapData(cm.Name)
				}
			case component.SectionSecrets:
				secret := m.navigator.SelectedSecret()
				if secret != nil {
					m.loading = true
					m.isDockerRegistrySecret = false
					return m, m.loadSecretData(secret.Name)
				}
			case component.SectionDockerRegistry:
				secret := m.navigator.SelectedDockerRegistrySecret()
				if secret != nil {
					m.loading = true
					m.isDockerRegistrySecret = true
					return m, m.loadSecretData(secret.Name)
				}
			}

		case component.ModeNamespace:
			// If nodes panel is active, load pods for selected node
			if m.nodesPanelActive {
				filteredNodes := m.filteredNodes()
				if len(filteredNodes) > 0 && m.nodeCursor < len(filteredNodes) {
					node := filteredNodes[m.nodeCursor]
					m.loading = true
					m.nodeSearching = false
					m.nodeSearchQuery = ""
					return m, m.loadPodsByNode(node.Name)
				}
			}
			// Check if namespace is not Active (e.g., Terminating)
			// If so, show delete confirmation instead of entering
			nsInfo := m.navigator.SelectedNamespaceInfo()
			if nsInfo != nil && nsInfo.Status != "Active" {
				m.confirmDialog.Show(
					fmt.Sprintf("Force delete namespace '%s'?", nsInfo.Name),
					"This will remove all resources and finalizers.",
					"delete_namespace",
					nsInfo,
				)
				return m, nil
			}
			// Otherwise, select namespace and load resources
			ns := m.navigator.SelectedNamespace()
			if ns != "" {
				m.k8sClient.SetNamespace(ns)
				m.config.SetLastNamespace(ns)
				m.selectedNode = "" // Clear node filter
				m.loading = true
				// Load all resources (pods, configmaps, secrets)
				return m, m.loadAllResources()
			}

		case component.ModeResourceType:
			rt := m.navigator.SelectedResourceType()
			m.navigator.SetResourceType(rt)
			m.config.SetLastResourceType(string(rt))
			m.navigator.SetMode(component.ModeWorkloads)
			m.loading = true
			return m, m.loadWorkloads()
		}
	}
	return m, nil
}

// refresh triggers a data refresh for the current view.
// - Navigator view: Reloads workloads for the current namespace and resource type
// - Dashboard view: Reloads pod dashboard data (logs, events, metrics)
func (m *Model) refresh() tea.Cmd {
	switch m.view {
	case ViewNavigator:
		m.loading = true
		return m.loadWorkloads()
	case ViewDashboard:
		if m.pod != nil {
			m.loading = true
			return m.loadDashboardData(m.pod)
		}
	}
	return nil
}

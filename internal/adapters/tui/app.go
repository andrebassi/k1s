// Package app provides the main application logic for k1s.
//
// This package implements the bubbletea Model interface, managing
// the application state, view transitions, and message handling.
// It coordinates between the Kubernetes client, UI components,
// and user interactions.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/configs"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/component"
	"github.com/andrebassi/k1s/internal/adapters/tui/keys"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
	"github.com/andrebassi/k1s/internal/adapters/tui/view"
)

// ViewState represents the current view mode of the application.
type ViewState int

// Application view states.
const (
	ViewNavigator ViewState = iota // Resource navigation view (namespaces, pods, etc.)
	ViewDashboard                  // Pod debugging dashboard (logs, events, metrics)
)

// Model is the main application state implementing tea.Model.
// It holds all UI components, Kubernetes client, and application state.
type Model struct {
	k8sClient          *repository.Client
	config             *configs.Config
	navigator          component.Navigator
	dashboard          view.Dashboard
	help               component.HelpPanel
	spinner            spinner.Model
	workloadActionMenu component.WorkloadActionMenu
	confirmDialog      component.ConfirmDialog
	configMapViewer        component.ConfigMapViewer
	secretViewer           component.SecretViewer
	dockerRegistryViewer   component.DockerRegistryViewer
	isDockerRegistrySecret bool // Track if we're viewing a docker registry secret
	view                   ViewState
	width              int
	height             int
	loading            bool
	err                error
	keys               keys.KeyMap
	workload           *repository.WorkloadInfo
	pod                *repository.PodInfo
	nodes              []repository.NodeInfo
	nodeCursor         int
	selectedNode       string // Node name for filtering pods
	nodesPanelActive   bool   // True when nodes panel is focused (right side)
	statusMsg          string // Status message for navigator view
	nodeSearching      bool   // True when searching nodes
	nodeSearchQuery    string // Node search query

	// State tracking for reactive log fetching
	lastShowPrevious bool
	lastLogContainer string

	// Flag to indicate we should load resources on init (when -n flag used)
	startWithResources bool
}

// Options configures the application initialization.
type Options struct {
	Namespace string // Initial namespace to select (empty for interactive selection)
}

// New creates a new application model with default options.
func New() (*Model, error) {
	return NewWithOptions(Options{})
}

// NewWithOptions creates a new application model with the specified options.
// If a namespace is provided, the app starts directly in the resources view.
func NewWithOptions(opts Options) (*Model, error) {
	client, err := repository.NewClient()
	if err != nil {
		return nil, err
	}

	cfg, err := configs.Load()
	if err != nil {
		cfg = configs.DefaultConfig()
	}

	// Use provided namespace or fall back to config
	initialNamespace := cfg.LastNamespace
	startInResources := false
	if opts.Namespace != "" {
		initialNamespace = opts.Namespace
		startInResources = true
	}
	client.SetNamespace(initialNamespace)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = style.SpinnerStyle

	navigator := component.NewNavigator()
	if startInResources {
		navigator.SetMode(component.ModeResources)
	}

	return &Model{
		k8sClient:          client,
		config:             cfg,
		navigator:          navigator,
		dashboard:          view.NewDashboard(),
		help:               component.NewHelpPanel(),
		spinner:            s,
		workloadActionMenu: component.NewWorkloadActionMenu(),
		confirmDialog:        component.NewConfirmDialog(),
		configMapViewer:      component.NewConfigMapViewer(),
		secretViewer:         component.NewSecretViewer(),
		dockerRegistryViewer: component.NewDockerRegistryViewer(),
		view:                 ViewNavigator,
		loading:            true,
		keys:               keys.DefaultKeyMap(),
		startWithResources: startInResources,
	}, nil
}

func (m Model) Init() tea.Cmd {
	if m.startWithResources {
		// When -n flag is used, load resources directly
		return tea.Batch(
			m.spinner.Tick,
			m.loadInitialDataWithResources(),
		)
	}
	return tea.Batch(
		m.spinner.Tick,
		m.loadInitialData(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.navigator.SetSize(msg.Width, msg.Height-3) // -2 for border, -1 for status bar
		m.dashboard.SetSize(msg.Width, msg.Height-3) // -2 for border, -1 for status bar
		m.help.SetSize(msg.Width, msg.Height)
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.navigator.SetWorkloads(msg.workloads)
		m.navigator.SetNamespaces(msg.namespaces)
		m.nodes = msg.nodes
		// Start with namespace selection if no workloads loaded (initial start)
		if len(msg.workloads) == 0 && len(msg.namespaces) > 0 {
			m.navigator.SetMode(component.ModeNamespace)
		}
		return m, nil

	case resourcesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.navigator.SetPods(msg.pods)
		m.navigator.SetConfigMaps(msg.configmaps)
		m.navigator.SetSecrets(msg.secrets)
		m.navigator.SetMode(component.ModeResources)
		// Pass workload info for scale controls when no pods
		// Use msg.workload (from namespace load) or m.workload (from workload selection)
		workload := msg.workload
		if workload == nil {
			workload = m.workload
		}
		m.navigator.SetScaleWorkload(workload)
		return m, nil

	case initialResourcesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.navigator.SetNamespaces(msg.namespaces)
		m.nodes = msg.nodes
		m.navigator.SetPods(msg.pods)
		m.navigator.SetConfigMaps(msg.configmaps)
		m.navigator.SetSecrets(msg.secrets)
		m.navigator.SetMode(component.ModeResources)
		return m, nil

	case configMapDataMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = "Error loading ConfigMap: " + msg.err.Error()
			return m, nil
		}
		m.configMapViewer.SetSize(m.width, m.height)
		m.configMapViewer.SetNamespaces(m.navigator.GetActiveNamespaceNames())
		m.configMapViewer.Show(msg.data, m.k8sClient.Namespace())
		return m, nil

	case component.ConfigMapViewerClosed:
		// ConfigMap viewer was closed, nothing special to do
		return m, nil

	case secretDataMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = "Error loading Secret: " + msg.err.Error()
			return m, nil
		}
		// Show appropriate viewer based on secret type
		if m.isDockerRegistrySecret {
			m.dockerRegistryViewer.SetSize(m.width, m.height)
			m.dockerRegistryViewer.SetNamespaces(m.navigator.GetActiveNamespaceNames())
			m.dockerRegistryViewer.Show(msg.data, m.k8sClient.Namespace())
		} else {
			m.secretViewer.SetSize(m.width, m.height)
			m.secretViewer.SetNamespaces(m.navigator.GetActiveNamespaceNames())
			m.secretViewer.Show(msg.data, m.k8sClient.Namespace())
		}
		return m, nil

	case component.DockerRegistryViewerClosed:
		// Docker Registry viewer was closed
		return m, nil

	case component.SecretViewerClosed:
		// Secret viewer was closed, nothing special to do
		return m, nil

	case component.SecretCopyRequest:
		// Handle secret copy request
		if msg.AllNamespaces {
			// Filter out source namespace and start progress flow
			var namespaces []string
			for _, ns := range msg.Namespaces {
				if ns != msg.SourceNamespace {
					namespaces = append(namespaces, ns)
				}
			}
			if len(namespaces) == 0 {
				if len(msg.Namespaces) == 0 {
					m.statusMsg = "Error: namespace list is empty"
				} else {
					m.statusMsg = "No other namespaces to copy to"
				}
				return m, clearStatusAfter(3 * time.Second)
			}
			// Start with first namespace
			first := namespaces[0]
			remaining := namespaces[1:]
			m.statusMsg = fmt.Sprintf("Copying to %s...", first)
			return m, m.copySecretToSingleNamespace(msg.SourceNamespace, msg.SecretName, first, remaining, 0, 0)
		} else {
			m.statusMsg = fmt.Sprintf("Copying to %s...", msg.TargetNamespace)
			return m, m.copySecretToSingleNamespace(msg.SourceNamespace, msg.SecretName, msg.TargetNamespace, nil, 0, 0)
		}

	case component.SecretCopyProgress:
		// Continue copying to next namespace
		statusText := fmt.Sprintf("Copying to %s... (%d done)", msg.CurrentNamespace, msg.SuccessCount+msg.ErrorCount)
		m.statusMsg = statusText
		m.secretViewer.SetStatusMsg(statusText)
		return m, m.copySecretToSingleNamespace(msg.SourceNamespace, msg.SecretName, msg.CurrentNamespace, msg.Remaining, msg.SuccessCount, msg.ErrorCount)

	case component.SecretCopyResult:
		// Show result
		var statusText string
		if msg.Success {
			statusText = msg.Message
		} else if msg.Err != nil {
			statusText = "Error: " + msg.Err.Error()
		} else {
			statusText = msg.Message
		}
		m.statusMsg = statusText
		m.secretViewer.SetStatusMsg(statusText)
		// Clear status after showing result
		return m, clearStatusAfter(3 * time.Second)

	case component.ConfigMapCopyProgress:
		// Continue copying to next namespace
		statusText := fmt.Sprintf("Copying to %s... (%d done)", msg.CurrentNamespace, msg.SuccessCount+msg.ErrorCount)
		m.statusMsg = statusText
		m.configMapViewer.SetStatusMsg(statusText)
		return m, m.copyConfigMapToSingleNamespace(msg.SourceNamespace, msg.ConfigMapName, msg.CurrentNamespace, msg.Remaining, msg.SuccessCount, msg.ErrorCount)

	case component.ConfigMapCopyResult:
		// Show result
		var statusText string
		if msg.Success {
			statusText = msg.Message
		} else if msg.Err != nil {
			statusText = "Error: " + msg.Err.Error()
		} else {
			statusText = msg.Message
		}
		m.statusMsg = statusText
		m.configMapViewer.SetStatusMsg(statusText)
		// Clear status after showing result
		return m, clearStatusAfter(3 * time.Second)

	case component.DockerRegistryCopyProgress:
		// Continue copying to next namespace
		statusText := fmt.Sprintf("Copying to %s... (%d done)", msg.CurrentNamespace, msg.SuccessCount+msg.ErrorCount)
		m.statusMsg = statusText
		m.dockerRegistryViewer.SetStatusMsg(statusText)
		return m, m.copyDockerRegistryToSingleNamespace(msg.SourceNamespace, msg.SecretName, msg.CurrentNamespace, msg.Remaining, msg.SuccessCount, msg.ErrorCount)

	case component.DockerRegistryCopyResult:
		// Show result
		var statusText string
		if msg.Success {
			statusText = msg.Message
		} else if msg.Err != nil {
			statusText = "Error: " + msg.Err.Error()
		} else {
			statusText = msg.Message
		}
		m.statusMsg = statusText
		m.dockerRegistryViewer.SetStatusMsg(statusText)
		// Clear status after showing result
		return m, clearStatusAfter(3 * time.Second)

	case nodePodLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = "Error loading pods: " + msg.err.Error()
			return m, nil
		}
		m.selectedNode = msg.nodeName
		m.navigator.SetPods(msg.pods)
		m.navigator.SetConfigMaps(nil) // Clear configmaps for node view
		m.navigator.SetSecrets(nil)    // Clear secrets for node view
		m.navigator.SetMode(component.ModeResources)
		return m, nil

	case dashboardDataMsg:
		m.loading = false
		// Update pod info for real-time status
		if msg.pod != nil {
			m.pod = msg.pod
			m.dashboard.SetPod(msg.pod)
		}
		m.dashboard.SetLogs(msg.logs)
		m.dashboard.SetEvents(msg.events)
		m.dashboard.SetMetrics(msg.metrics)
		m.dashboard.SetRelated(msg.related)
		m.dashboard.SetHelpers(msg.helpers)
		m.dashboard.SetNode(msg.node)
		// Pass workload info to navigator for scale controls when no pods
		if msg.related != nil && msg.related.Owner != nil && msg.related.Owner.WorkloadKind != "" {
			// Convert Owner info to WorkloadInfo for Navigator
			var resourceType repository.ResourceType
			switch msg.related.Owner.WorkloadKind {
			case "Deployment":
				resourceType = repository.ResourceDeployments
			case "StatefulSet":
				resourceType = repository.ResourceStatefulSets
			case "DaemonSet":
				resourceType = repository.ResourceDaemonSets
			case "Rollout":
				resourceType = repository.ResourceRollouts
			}
			m.navigator.SetScaleWorkload(&repository.WorkloadInfo{
				Name:      msg.related.Owner.WorkloadName,
				Namespace: m.k8sClient.Namespace(),
				Type:      resourceType,
				Replicas:  msg.related.Owner.Replicas,
			})
		}
		return m, nil

	case logsUpdatedMsg:
		m.dashboard.SetLogs(msg.logs)
		return m, nil

	case view.DeletePodRequest:
		return m, m.deletePod(msg.Namespace, msg.PodName)

	case podDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Go back to pods list after deletion
			m.view = ViewNavigator
			m.pod = nil
			m.navigator.SetMode(component.ModeResources)
			return m, m.loadAllResources()
		}
		return m, nil

	case namespaceDeletedMsg:
		if msg.err != nil {
			m.statusMsg = "Failed to delete namespace: " + msg.err.Error()
		} else {
			m.statusMsg = fmt.Sprintf("Namespace %s deleted", msg.namespace)
			// Refresh namespace list
			return m, tea.Batch(m.loadInitialData(), clearStatusAfter(3*time.Second))
		}
		return m, clearStatusAfter(5 * time.Second)

	case component.WorkloadActionMenuResult:
		workload := m.navigator.SelectedWorkload()
		if workload == nil {
			return m, nil
		}
		switch msg.Item.Action {
		case "scale":
			m.loading = true
			return m, m.scaleWorkload(workload, msg.Item.Replicas)
		case "copy":
			err := component.CopyToClipboard(msg.Item.Command)
			if err == nil {
				m.statusMsg = "Copied: " + msg.Item.Label
			} else {
				m.statusMsg = "Copy failed: " + err.Error()
			}
		}
		return m, nil

	case component.ConfirmResult:
		// Handle workload restart at app level
		if msg.Confirmed && msg.Action == "restart" {
			if workload, ok := msg.Data.(*repository.WorkloadInfo); ok {
				m.loading = true
				m.statusMsg = "Restarting..."
				return m, m.restartWorkload(workload)
			}
		}
		// Handle namespace force delete
		if msg.Confirmed && msg.Action == "delete_namespace" {
			if nsInfo, ok := msg.Data.(*repository.NamespaceInfo); ok {
				m.statusMsg = fmt.Sprintf("Deleting namespace %s...", nsInfo.Name)
				return m, m.forceDeleteNamespace(nsInfo.Name)
			}
		}
		// Forward other confirm results (exec, port-forward, delete) to dashboard
		if m.view == ViewDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case view.ExecFinishedMsg:
		// Forward exec finished to dashboard
		if m.view == ViewDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case view.DescribeOutputMsg:
		// Forward describe output to dashboard
		if m.view == ViewDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case workloadActionMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = "Error: " + msg.err.Error()
			return m, clearStatusAfter(5 * time.Second)
		}
		switch msg.action {
		case "scale":
			m.statusMsg = fmt.Sprintf("Scaled %s to %d replicas", msg.workloadName, msg.replicas)
		case "restart":
			m.statusMsg = fmt.Sprintf("Restart initiated for %s", msg.workloadName)
		}
		// Refresh based on current view
		if m.view == ViewNavigator && m.navigator.Mode() == component.ModeResources {
			// Stay on resources view and reload
			return m, tea.Batch(m.loadAllResources(), clearStatusAfter(3*time.Second))
		}
		// Refresh workloads list for other views
		return m, tea.Batch(m.loadWorkloads(), clearStatusAfter(3*time.Second))

	case clearStatusMsg:
		m.statusMsg = ""
		m.secretViewer.SetStatusMsg("")
		m.configMapViewer.SetStatusMsg("")
		m.dockerRegistryViewer.SetStatusMsg("")
		return m, nil

	case view.ScaleRequestMsg:
		// Handle scale request from dashboard
		var resourceType repository.ResourceType
		switch msg.WorkloadKind {
		case "Deployment":
			resourceType = repository.ResourceDeployments
		case "StatefulSet":
			resourceType = repository.ResourceStatefulSets
		case "DaemonSet":
			resourceType = repository.ResourceDaemonSets
		case "Rollout":
			resourceType = repository.ResourceRollouts
		}
		workload := &repository.WorkloadInfo{
			Name:      msg.WorkloadName,
			Namespace: msg.Namespace,
			Type:      resourceType,
			Replicas:  msg.NewReplicas,
		}
		m.statusMsg = fmt.Sprintf("Scaling %s to %d...", msg.WorkloadName, msg.NewReplicas)
		return m, m.scaleWorkload(workload, msg.NewReplicas)

	case tickMsg:
		if m.view == ViewDashboard && m.pod != nil {
			return m, tea.Batch(
				m.loadDashboardData(m.pod),
				m.tickCmd(),
			)
		}
		// Refresh resources list in real-time when viewing resources
		if m.view == ViewNavigator && m.navigator.Mode() == component.ModeResources {
			// If viewing pods by node, refresh with node filter
			if m.selectedNode != "" {
				return m, tea.Batch(
					m.loadPodsByNode(m.selectedNode),
					m.tickCmd(),
				)
			}
			return m, tea.Batch(
				m.loadAllResources(),
				m.tickCmd(),
			)
		}
		return m, m.tickCmd()

	case tea.KeyMsg:
		// Confirm dialog takes highest priority
		if m.confirmDialog.IsVisible() {
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		// Workload action menu takes priority
		if m.workloadActionMenu.IsVisible() {
			m.workloadActionMenu, cmd = m.workloadActionMenu.Update(msg)
			return m, cmd
		}

		// Help overlay takes priority
		if m.help.IsVisible() {
			if msg.String() == "?" || msg.String() == "esc" {
				m.help.Hide()
				return m, nil
			}
			return m, nil
		}

		// ConfigMap viewer takes priority
		if m.configMapViewer.IsVisible() {
			m.configMapViewer, cmd = m.configMapViewer.Update(msg)
			// Check for pending copy request
			if req := m.configMapViewer.GetPendingRequest(); req != nil {
				// Handle the copy request directly
				if req.AllNamespaces {
					var namespaces []string
					for _, ns := range req.Namespaces {
						if ns != req.SourceNamespace {
							namespaces = append(namespaces, ns)
						}
					}
					if len(namespaces) == 0 {
						statusText := "No other namespaces to copy to"
						m.statusMsg = statusText
						m.configMapViewer.SetStatusMsg(statusText)
						return m, clearStatusAfter(3 * time.Second)
					}
					first := namespaces[0]
					remaining := namespaces[1:]
					statusText := fmt.Sprintf("Copying to %s...", first)
					m.statusMsg = statusText
					m.configMapViewer.SetStatusMsg(statusText)
					return m, m.copyConfigMapToSingleNamespace(req.SourceNamespace, req.ConfigMapName, first, remaining, 0, 0)
				}
				statusText := fmt.Sprintf("Copying to %s...", req.TargetNamespace)
				m.statusMsg = statusText
				m.configMapViewer.SetStatusMsg(statusText)
				return m, m.copyConfigMapToSingleNamespace(req.SourceNamespace, req.ConfigMapName, req.TargetNamespace, nil, 0, 0)
			}
			return m, cmd
		}

		// Docker Registry viewer takes priority
		if m.dockerRegistryViewer.IsVisible() {
			m.dockerRegistryViewer, cmd = m.dockerRegistryViewer.Update(msg)
			// Check for pending copy request
			if req := m.dockerRegistryViewer.GetPendingRequest(); req != nil {
				// Handle the copy request directly
				if req.AllNamespaces {
					var namespaces []string
					for _, ns := range req.Namespaces {
						if ns != req.SourceNamespace {
							namespaces = append(namespaces, ns)
						}
					}
					if len(namespaces) == 0 {
						statusText := "No other namespaces to copy to"
						m.statusMsg = statusText
						m.dockerRegistryViewer.SetStatusMsg(statusText)
						return m, clearStatusAfter(3 * time.Second)
					}
					first := namespaces[0]
					remaining := namespaces[1:]
					statusText := fmt.Sprintf("Copying to %s...", first)
					m.statusMsg = statusText
					m.dockerRegistryViewer.SetStatusMsg(statusText)
					return m, m.copyDockerRegistryToSingleNamespace(req.SourceNamespace, req.SecretName, first, remaining, 0, 0)
				}
				statusText := fmt.Sprintf("Copying to %s...", req.TargetNamespace)
				m.statusMsg = statusText
				m.dockerRegistryViewer.SetStatusMsg(statusText)
				return m, m.copyDockerRegistryToSingleNamespace(req.SourceNamespace, req.SecretName, req.TargetNamespace, nil, 0, 0)
			}
			return m, cmd
		}

		// Secret viewer takes priority
		if m.secretViewer.IsVisible() {
			m.secretViewer, cmd = m.secretViewer.Update(msg)
			// Check for pending copy request
			if req := m.secretViewer.GetPendingRequest(); req != nil {
				// Handle the copy request directly
				if req.AllNamespaces {
					var namespaces []string
					for _, ns := range req.Namespaces {
						if ns != req.SourceNamespace {
							namespaces = append(namespaces, ns)
						}
					}
					if len(namespaces) == 0 {
						statusText := "No other namespaces to copy to"
						m.statusMsg = statusText
						m.secretViewer.SetStatusMsg(statusText)
						return m, clearStatusAfter(3 * time.Second)
					}
					first := namespaces[0]
					remaining := namespaces[1:]
					statusText := fmt.Sprintf("Copying to %s...", first)
					m.statusMsg = statusText
					m.secretViewer.SetStatusMsg(statusText)
					return m, m.copySecretToSingleNamespace(req.SourceNamespace, req.SecretName, first, remaining, 0, 0)
				}
				statusText := fmt.Sprintf("Copying to %s...", req.TargetNamespace)
				m.statusMsg = statusText
				m.secretViewer.SetStatusMsg(statusText)
				return m, m.copySecretToSingleNamespace(req.SourceNamespace, req.SecretName, req.TargetNamespace, nil, 0, 0)
			}
			return m, cmd
		}

		// When navigator is searching, handle keys appropriately
		if m.view == ViewNavigator && m.navigator.IsSearching() {
			if msg.String() == "ctrl+c" {
				m.saveConfig()
				return m, tea.Quit
			}
			// Tab or Enter: exit search mode, keep filter, allow navigation
			if msg.Type == tea.KeyTab || msg.String() == "tab" || msg.Type == tea.KeyEnter || msg.String() == "enter" {
				m.navigator, cmd = m.navigator.Update(msg)
				return m, cmd
			}
			// Pass all other keys to navigator for typing
			m.navigator, cmd = m.navigator.Update(msg)
			return m, cmd
		}

		// When dashboard is in fullscreen logs/events mode, pass search-related keys directly
		if m.view == ViewDashboard && (m.dashboard.IsFullscreenLogs() || m.dashboard.IsFullscreenEvents()) {
			key := msg.String()
			// Pass / to start search, OR all keys if already searching
			if key == "/" || m.dashboard.IsLogsSearching() || m.dashboard.IsEventsSearching() {
				m.dashboard, cmd = m.dashboard.Update(msg)
				return m, cmd
			}
		}

		// Handle node search mode
		if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace && m.nodesPanelActive && m.nodeSearching {
			switch msg.String() {
			case "esc":
				// Esc clears search or exits search mode
				if m.nodeSearchQuery != "" {
					m.nodeSearchQuery = ""
					m.nodeCursor = 0
				} else {
					m.nodeSearching = false
				}
				return m, nil
			case "tab", "enter":
				// Tab/Enter: exit search mode, keep filter, allow navigation
				m.nodeSearching = false
				return m, nil
			case "backspace":
				if len(m.nodeSearchQuery) > 0 {
					m.nodeSearchQuery = m.nodeSearchQuery[:len(m.nodeSearchQuery)-1]
					m.nodeCursor = 0
				}
				return m, nil
			default:
				k := msg.String()
				if len(k) == 1 {
					m.nodeSearchQuery += k
					m.nodeCursor = 0
				}
				return m, nil
			}
		}

		// Start node search with / key
		if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace && m.nodesPanelActive && !m.nodeSearching {
			if msg.String() == "/" {
				m.nodeSearching = true
				m.nodeSearchQuery = ""
				return m, nil
			}
			// Clear node filter with c key
			if msg.String() == "c" && m.nodeSearchQuery != "" {
				m.nodeSearchQuery = ""
				m.nodeCursor = 0
				return m, nil
			}
		}

		// Normal key handling when not searching
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.saveConfig()
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.Toggle()
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m, m.refresh()

		case key.Matches(msg, m.keys.Namespace):
			if m.view == ViewNavigator {
				m.navigator.SetMode(component.ModeNamespace)
				m.nodesPanelActive = false
				return m, nil
			}

		case key.Matches(msg, m.keys.NextPanel):
			// In namespace mode, switch between namespace and node panels
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace {
				m.nodesPanelActive = !m.nodesPanelActive
				return m, nil
			}

		case key.Matches(msg, m.keys.PrevPanel):
			// In namespace mode, switch between namespace and node panels
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace {
				m.nodesPanelActive = !m.nodesPanelActive
				return m, nil
			}

		case msg.String() == "left":
			// In namespace mode, switch to namespace panel (left)
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace {
				m.nodesPanelActive = false
				return m, nil
			}

		case msg.String() == "right":
			// In namespace mode, switch to node panel (right)
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace && len(m.nodes) > 0 {
				m.nodesPanelActive = true
				return m, nil
			}

		case msg.String() == "d":
			// In namespace mode, delete Terminating namespaces
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace && !m.nodesPanelActive {
				nsInfo := m.navigator.SelectedNamespaceInfo()
				if nsInfo != nil && nsInfo.Status != "Active" {
					// Show confirmation dialog for namespace deletion
					m.confirmDialog.Show(
						fmt.Sprintf("Force delete namespace '%s'?", nsInfo.Name),
						"This will remove all resources and finalizers.",
						"delete_namespace",
						nsInfo,
					)
					return m, nil
				}
			}

		case key.Matches(msg, m.keys.Up):
			// Handle node panel navigation
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace && m.nodesPanelActive {
				if m.nodeCursor > 0 {
					m.nodeCursor--
				}
				return m, nil
			}

		case key.Matches(msg, m.keys.Down):
			// Handle node panel navigation
			if m.view == ViewNavigator && m.navigator.Mode() == component.ModeNamespace && m.nodesPanelActive {
				filteredNodes := m.filteredNodes()
				if m.nodeCursor < len(filteredNodes)-1 {
					m.nodeCursor++
				}
				return m, nil
			}

		case key.Matches(msg, m.keys.Back):
			// Don't handle back if dashboard has active overlay or is searching - let dashboard handle esc
			if m.view == ViewDashboard && (m.dashboard.IsLogsSearching() || m.dashboard.HasActiveOverlay()) {
				break // Fall through to dashboard update
			}
			// If dashboard is fullscreen, just close fullscreen instead of going back
			if m.view == ViewDashboard && m.dashboard.IsFullscreen() {
				m.dashboard.CloseFullscreen()
				return m, nil
			}
			return m.handleBack()

		case key.Matches(msg, m.keys.Enter):
			// Don't handle enter if dashboard has active overlay - let dashboard handle it
			if m.view == ViewDashboard && m.dashboard.HasActiveOverlay() {
				break // Fall through to dashboard update
			}
			// Let dashboard handle Enter for fullscreen toggle
			if m.view == ViewDashboard {
				break // Fall through to dashboard update
			}
			return m.handleEnter()
		}
	}

	switch m.view {
	case ViewNavigator:
		if !m.navigator.IsSearching() {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if key.Matches(msg, m.keys.ResourceType) {
					m.navigator.SetMode(component.ModeResourceType)
					return m, nil
				}
				// Scale action (only for scalable resource types)
				if key.Matches(msg, m.keys.Scale) && m.navigator.Mode() == component.ModeWorkloads {
					workload := m.navigator.SelectedWorkload()
					if workload != nil {
						rt := m.navigator.ResourceType()
						if rt == repository.ResourceDeployments || rt == repository.ResourceStatefulSets {
							items := component.ScaleActions(
								m.k8sClient.Namespace(),
								workload.Name,
								string(rt),
								workload.Replicas,
							)
							m.workloadActionMenu.Show("Scale "+workload.Name, items)
							return m, nil
						}
					}
				}
				// Restart action
				if key.Matches(msg, m.keys.Restart) && m.navigator.Mode() == component.ModeWorkloads {
					workload := m.navigator.SelectedWorkload()
					if workload != nil {
						rt := m.navigator.ResourceType()
						if rt == repository.ResourceDeployments || rt == repository.ResourceStatefulSets || rt == repository.ResourceDaemonSets {
							m.confirmDialog.Show(
								"Restart "+string(rt),
								"Are you sure you want to restart '"+workload.Name+"'?",
								"restart",
								workload,
							)
							return m, nil
						}
					}
				}
				// Scale up ('s') in resources view when no pods but workload exists
				if msg.String() == "s" && m.navigator.Mode() == component.ModeResources && m.navigator.HasWorkload() {
					workload := m.navigator.GetScaleWorkload()
					if workload != nil {
						newReplicas := int32(1) // Scale to 1 when no pods
						m.statusMsg = fmt.Sprintf("Scaling %s to %d...", workload.Name, newReplicas)
						return m, m.scaleWorkload(workload, newReplicas)
					}
				}
				// Scale down ('d') in resources view when no pods but workload exists
				if msg.String() == "d" && m.navigator.Mode() == component.ModeResources && m.navigator.HasWorkload() {
					workload := m.navigator.GetScaleWorkload()
					if workload != nil && workload.Replicas > 0 {
						newReplicas := workload.Replicas - 1
						m.statusMsg = fmt.Sprintf("Scaling %s to %d...", workload.Name, newReplicas)
						return m, m.scaleWorkload(workload, newReplicas)
					}
				}
			}
		}
		m.navigator, cmd = m.navigator.Update(msg)
		cmds = append(cmds, cmd)

	case ViewDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
		cmds = append(cmds, cmd)

		// Check if log state changed and needs refresh
		if m.pod != nil {
			currentShowPrevious := m.dashboard.LogsShowPrevious()
			currentContainer := m.dashboard.LogsSelectedContainer()

			if currentShowPrevious != m.lastShowPrevious || currentContainer != m.lastLogContainer {
				m.lastShowPrevious = currentShowPrevious
				m.lastLogContainer = currentContainer
				cmds = append(cmds, m.loadLogsForState(m.pod, currentContainer, currentShowPrevious))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

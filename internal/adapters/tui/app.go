// Package app provides the main application logic for k1s.
//
// This package implements the bubbletea Model interface, managing
// the application state, view transitions, and message handling.
// It coordinates between the Kubernetes client, UI components,
// and user interactions.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	configMapViewer    component.ConfigMapViewer
	secretViewer       component.SecretViewer
	view               ViewState
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

// loadedMsg is sent when initial data loading completes.
type loadedMsg struct {
	workloads  []repository.WorkloadInfo
	namespaces []string
	nodes      []repository.NodeInfo
	err        error
}

// resourcesLoadedMsg is sent when namespace resources are loaded.
type resourcesLoadedMsg struct {
	pods       []repository.PodInfo
	configmaps []repository.ConfigMapInfo
	secrets    []repository.SecretInfo
	workload   *repository.WorkloadInfo // First scalable workload for scale controls
	err        error
}

// dashboardDataMsg is sent when pod dashboard data is ready.
type dashboardDataMsg struct {
	pod     *repository.PodInfo
	logs    []repository.LogLine
	events  []repository.EventInfo
	metrics *repository.PodMetrics
	related *repository.RelatedResources
	helpers []repository.DebugHelper
	node    *repository.NodeInfo
}

type logsUpdatedMsg struct {
	logs []repository.LogLine
}

type podDeletedMsg struct {
	namespace string
	podName   string
	err       error
}

type workloadActionMsg struct {
	action       string
	workloadName string
	namespace    string
	resourceType repository.ResourceType
	replicas     int32
	err          error
}

type tickMsg time.Time

type clearStatusMsg struct{}

type configMapDataMsg struct {
	data *repository.ConfigMapData
	err  error
}

type secretDataMsg struct {
	data *repository.SecretData
	err  error
}

type nodePodLoadedMsg struct {
	nodeName string
	pods     []repository.PodInfo
	err      error
}

type initialResourcesLoadedMsg struct {
	namespaces []string
	nodes      []repository.NodeInfo
	pods       []repository.PodInfo
	configmaps []repository.ConfigMapInfo
	secrets    []repository.SecretInfo
	err        error
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
		confirmDialog:      component.NewConfirmDialog(),
		configMapViewer:    component.NewConfigMapViewer(),
		secretViewer:       component.NewSecretViewer(),
		view:               ViewNavigator,
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
		m.configMapViewer.SetNamespaces(m.navigator.GetNamespaces())
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
		m.secretViewer.SetSize(m.width, m.height)
		m.secretViewer.SetNamespaces(m.navigator.GetNamespaces())
		m.secretViewer.Show(msg.data, m.k8sClient.Namespace())
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

func (m Model) View() string {
	if m.err != nil {
		return style.StatusError.Render("Error: " + m.err.Error())
	}

	if m.loading {
		// Center loading spinner
		loadingMsg := m.spinner.View() + " Loading..."
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loadingMsg)
	}

	// Calculate dimensions for content box
	contentHeight := m.height - 2 // 2 for border top/bottom
	contentWidth := m.width - 2   // 2 for border left/right

	var content string
	switch m.view {
	case ViewNavigator:
		// In namespace mode, show nodes panel on the right
		if m.navigator.Mode() == component.ModeNamespace && len(m.nodes) > 0 {
			leftWidth := contentWidth / 2
			rightWidth := contentWidth - leftWidth - 3 // 3 for separator

			// Set panel active state for indicator
			m.navigator.SetPanelActive(!m.nodesPanelActive)

			leftContent := m.navigator.View()
			rightContent := m.renderNodesPanel(rightWidth, m.height-3) // Match navigator height

			// Join panels side by side
			content = lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().Width(leftWidth).Render(leftContent),
				lipgloss.NewStyle().Foreground(style.Surface).Render(" │ "),
				lipgloss.NewStyle().Width(rightWidth).Render(rightContent),
			)
		} else {
			m.navigator.SetPanelActive(true)
			content = m.navigator.View()
		}
	case ViewDashboard:
		content = m.dashboard.View()
	}

	// Render confirm dialog as overlay (highest priority)
	if m.confirmDialog.IsVisible() {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.confirmDialog.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Render workload action menu as overlay
	if m.workloadActionMenu.IsVisible() {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.workloadActionMenu.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	if m.help.IsVisible() {
		// Render floating help modal centered on screen
		helpModal := m.help.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			helpModal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Render ConfigMap viewer as overlay
	if m.configMapViewer.IsVisible() {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Left,
			lipgloss.Top,
			m.configMapViewer.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Render Secret viewer as overlay
	if m.secretViewer.IsVisible() {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Left,
			lipgloss.Top,
			m.secretViewer.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Reserve 1 line for status bar
	boxHeight := contentHeight - 1

	// Create bordered box for content
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Surface).
		Width(contentWidth).
		Height(boxHeight)

	boxedContent := boxStyle.Render(content)

	// Status bar at bottom (always reserve space, same width as box)
	statusStyle := lipgloss.NewStyle().
		Foreground(style.Warning).
		Bold(true).
		Padding(0, 2).
		Width(contentWidth + 2) // +2 for border
	statusBar := statusStyle.Render(m.statusMsg)

	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusBar)
}

func (m Model) renderNodesPanel(width, height int) string {
	var b strings.Builder

	// Get filtered nodes
	nodes := m.filteredNodes()

	// Header
	iconStyle := lipgloss.NewStyle().Foreground(style.Primary).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(style.Text).Bold(true)

	if m.nodesPanelActive {
		b.WriteString(iconStyle.Render("● "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(titleStyle.Render("SELECT NODE"))
	b.WriteString("\n")

	// Show search bar or filter indicator on separate line (same style as namespace)
	if m.nodeSearching {
		searchStyle := lipgloss.NewStyle().
			Foreground(style.Text).
			Background(style.Surface).
			Padding(0, 1)
		b.WriteString(searchStyle.Render("/ " + m.nodeSearchQuery + "_"))
		b.WriteString("\n\n")
	} else if m.nodeSearchQuery != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(style.Secondary).
			Bold(true)
		clearHint := style.HelpDescStyle.Render(" (c to clear)")
		b.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s", m.nodeSearchQuery)))
		b.WriteString(clearHint)
		b.WriteString("\n\n")
	} else {
		// Empty line to maintain consistent table position
		b.WriteString("\n\n")
	}

	// Table header
	header := fmt.Sprintf("  %-3s %-40s %-8s %s", "#", "NODE", "STATUS", "PODS")
	b.WriteString(style.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Calculate visible window for scrolling
	maxVisible := height - 8 // Account for header and footer (same as namespace)
	if maxVisible < 5 {
		maxVisible = 5
	}

	startIdx := 0
	endIdx := len(nodes)
	if len(nodes) > maxVisible {
		startIdx = m.nodeCursor - maxVisible/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + maxVisible
		if endIdx > len(nodes) {
			endIdx = len(nodes)
			startIdx = endIdx - maxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	for i := startIdx; i < endIdx; i++ {
		node := nodes[i]
		idx := fmt.Sprintf("%d", i+1)

		statusStyle := style.StatusRunning
		if node.Status != "Ready" {
			statusStyle = style.StatusError
		}

		cursor := "  "
		nodeName := style.Truncate(node.Name, 40)
		// Pad status to 8 chars before styling to maintain alignment
		statusPadded := fmt.Sprintf("%-8s", node.Status)
		if m.nodesPanelActive && i == m.nodeCursor {
			cursor = style.CursorStyle.Render("> ")
			rowStyle := lipgloss.NewStyle().Background(style.Surface)
			row := fmt.Sprintf("%s%-3s %-40s %s %d",
				cursor,
				idx,
				nodeName,
				statusStyle.Render(statusPadded),
				node.PodCount,
			)
			b.WriteString(rowStyle.Render(row))
		} else {
			b.WriteString(fmt.Sprintf("%s%-3s %-40s %s %d",
				cursor,
				idx,
				nodeName,
				statusStyle.Render(statusPadded),
				node.PodCount,
			))
		}
		b.WriteString("\n")
	}

	return b.String()
}

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
					return m, m.loadSecretData(secret.Name)
				}
			case component.SectionDockerRegistry:
				secret := m.navigator.SelectedDockerRegistrySecret()
				if secret != nil {
					m.loading = true
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
			// Otherwise, select namespace
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

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(m.config.RefreshInterval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (m *Model) saveConfig() {
	_ = m.config.Save()
}

func (m *Model) deletePod(namespace, podName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.DeletePod(ctx, namespace, podName)
		return podDeletedMsg{
			namespace: namespace,
			podName:   podName,
			err:       err,
		}
	}
}

func (m *Model) scaleWorkload(workload *repository.WorkloadInfo, replicas int32) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.ScaleWorkload(ctx, workload.Namespace, workload.Name, workload.Type, replicas)
		return workloadActionMsg{
			action:       "scale",
			workloadName: workload.Name,
			namespace:    workload.Namespace,
			resourceType: workload.Type,
			replicas:     replicas,
			err:          err,
		}
	}
}

func (m *Model) restartWorkload(workload *repository.WorkloadInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.RestartWorkload(ctx, workload.Namespace, workload.Name, workload.Type)
		return workloadActionMsg{
			action:       "restart",
			workloadName: workload.Name,
			namespace:    workload.Namespace,
			resourceType: workload.Type,
			err:          err,
		}
	}
}

func (m *Model) copySecretToSingleNamespace(sourceNs, secretName, targetNs string, remaining []string, successCount, errorCount int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Small delay so user can see the namespace name
		time.Sleep(300 * time.Millisecond)

		// Copy to current namespace
		err := repository.CopySecretToNamespace(ctx, m.k8sClient.Clientset(), sourceNs, secretName, targetNs)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}

		// If no more remaining, return final result
		if len(remaining) == 0 {
			if errorCount > 0 {
				return component.SecretCopyResult{
					Success: false,
					Message: fmt.Sprintf("Copied to %d namespaces, %d failed", successCount, errorCount),
				}
			}
			if successCount == 1 {
				return component.SecretCopyResult{
					Success: true,
					Message: fmt.Sprintf("Copied to %s", targetNs),
				}
			}
			return component.SecretCopyResult{
				Success: true,
				Message: fmt.Sprintf("Copied to %d namespaces", successCount),
			}
		}

		// Send progress for next namespace
		next := remaining[0]
		newRemaining := remaining[1:]
		return component.SecretCopyProgress{
			SecretName:       secretName,
			SourceNamespace:  sourceNs,
			CurrentNamespace: next,
			Remaining:        newRemaining,
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
		}
	}
}

func (m *Model) copyConfigMapToSingleNamespace(sourceNs, configMapName, targetNs string, remaining []string, successCount, errorCount int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Small delay so user can see the namespace name
		time.Sleep(300 * time.Millisecond)

		// Copy to current namespace
		err := repository.CopyConfigMapToNamespace(ctx, m.k8sClient.Clientset(), sourceNs, configMapName, targetNs)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}

		// If no more remaining, return final result
		if len(remaining) == 0 {
			if errorCount > 0 {
				return component.ConfigMapCopyResult{
					Success: false,
					Message: fmt.Sprintf("Copied to %d namespaces, %d failed", successCount, errorCount),
				}
			}
			if successCount == 1 {
				return component.ConfigMapCopyResult{
					Success: true,
					Message: fmt.Sprintf("Copied to %s", targetNs),
				}
			}
			return component.ConfigMapCopyResult{
				Success: true,
				Message: fmt.Sprintf("Copied to %d namespaces", successCount),
			}
		}

		// Send progress for next namespace
		next := remaining[0]
		newRemaining := remaining[1:]
		return component.ConfigMapCopyProgress{
			ConfigMapName:    configMapName,
			SourceNamespace:  sourceNs,
			CurrentNamespace: next,
			Remaining:        newRemaining,
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
		}
	}
}

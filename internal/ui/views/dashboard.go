package views

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k8sdebug/internal/k8s"
	"github.com/andrebassi/k8sdebug/internal/ui/components"
	"github.com/andrebassi/k8sdebug/internal/ui/keys"
	"github.com/andrebassi/k8sdebug/internal/ui/styles"
)

type PanelFocus int

const (
	FocusLogs PanelFocus = iota
	FocusEvents
	FocusMetrics
	FocusManifest
)

type Dashboard struct {
	pod           *k8s.PodInfo
	related       *k8s.RelatedResources
	logs          components.LogsPanel
	events        components.EventsPanel
	metrics       components.MetricsPanel
	manifest      components.ManifestPanel
	breadcrumb    components.Breadcrumb
	help          components.HelpPanel
	actionMenu    components.ActionMenu
	podActionMenu components.PodActionMenu
	confirmDialog components.ConfirmDialog
	resultViewer  components.ResultViewer
	focus         PanelFocus
	fullscreen    bool
	width         int
	height        int
	keys          keys.KeyMap
	statusMsg     string // Temporary status message (e.g., "Copied!")
	namespace     string // Current namespace for kubectl commands
	context       string // Current context for kubectl commands
	pendingAction *components.PodActionItem // Action waiting for confirmation
}

func NewDashboard() Dashboard {
	return Dashboard{
		logs:          components.NewLogsPanel(),
		events:        components.NewEventsPanel(),
		metrics:       components.NewMetricsPanel(),
		manifest:      components.NewManifestPanel(),
		breadcrumb:    components.NewBreadcrumb(),
		help:          components.NewHelpPanel(),
		actionMenu:    components.NewActionMenu(),
		podActionMenu: components.NewPodActionMenu(),
		confirmDialog: components.NewConfirmDialog(),
		resultViewer:  components.NewResultViewer(),
		focus:         FocusLogs,
		keys:          keys.DefaultKeyMap(),
	}
}

func (d Dashboard) Init() tea.Cmd {
	return nil
}

// DeletePodRequest is sent to app.go to request pod deletion
type DeletePodRequest struct {
	Namespace string
	PodName   string
}

// ExecFinishedMsg is sent when an external command finishes
type ExecFinishedMsg struct {
	Err error
}

// DescribeOutputMsg contains the output of kubectl describe
type DescribeOutputMsg struct {
	Title   string
	Content string
	Err     error
}

func (d Dashboard) Update(msg tea.Msg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle ExecFinishedMsg (after external command returns)
	if result, ok := msg.(ExecFinishedMsg); ok {
		if result.Err != nil {
			d.statusMsg = "Command failed: " + result.Err.Error()
		} else {
			d.statusMsg = "Command completed"
		}
		return d, nil
	}

	// Handle DescribeOutputMsg (display describe output in result viewer)
	if result, ok := msg.(DescribeOutputMsg); ok {
		if result.Err != nil {
			d.statusMsg = "Describe failed: " + result.Err.Error()
		} else {
			d.resultViewer.Show(result.Title, result.Content, d.width-4, d.height-4)
		}
		return d, nil
	}

	// Handle ActionMenuResult (copy commands)
	if result, ok := msg.(components.ActionMenuResult); ok {
		if result.Copied && result.Err == nil {
			d.statusMsg = "Copied: " + result.Item.Label
		} else if result.Err != nil {
			d.statusMsg = "Copy failed: " + result.Err.Error()
		}
		return d, nil
	}

	// Handle ResultViewerCopiedMsg (copy from result viewer)
	if result, ok := msg.(components.ResultViewerCopiedMsg); ok {
		if result.Err == nil {
			contentLen := len(result.Content)
			d.statusMsg = fmt.Sprintf("Copied %d chars to clipboard: %s", contentLen, result.Title)
		} else {
			d.statusMsg = "Copy failed: " + result.Err.Error()
		}
		return d, nil
	}

	// Handle PodActionMenuResult
	if result, ok := msg.(components.PodActionMenuResult); ok {
		switch result.Item.Action {
		case "delete":
			// Show confirmation dialog
			d.confirmDialog.Show(
				"Delete Pod",
				"Are you sure you want to delete pod '"+d.pod.Name+"'?",
				"delete",
				d.pod,
			)
			return d, nil
		case "exec":
			// Show confirmation before exec
			d.pendingAction = &result.Item
			d.confirmDialog.Show(
				"Exec into Pod",
				"Open shell in '"+d.pod.Name+"'?\nThis will suspend the UI until you exit the shell.",
				"exec",
				d.pod,
			)
			return d, nil
		case "port-forward":
			// Show confirmation before port-forward
			d.pendingAction = &result.Item
			d.confirmDialog.Show(
				"Port Forward",
				"Start port forwarding for '"+d.pod.Name+"'?\nPress Ctrl+C in terminal to stop and return.",
				"port-forward",
				d.pod,
			)
			return d, nil
		case "describe":
			// Run describe command and capture output
			d.statusMsg = "Loading describe..."
			cmdStr := result.Item.Command
			podName := d.pod.Name
			return d, func() tea.Msg {
				c := exec.Command("sh", "-c", cmdStr)
				output, err := c.CombinedOutput()
				if err != nil {
					return DescribeOutputMsg{Err: err}
				}
				return DescribeOutputMsg{
					Title:   "Pod: " + podName,
					Content: string(output),
				}
			}
		case "copy":
			// Copy the command to clipboard
			err := components.CopyToClipboard(result.Item.Command)
			if err == nil {
				d.statusMsg = "Copied: " + result.Item.Label
			} else {
				d.statusMsg = "Copy failed: " + err.Error()
			}
			return d, nil
		}
		return d, nil
	}

	// Handle ConfirmResult
	if result, ok := msg.(components.ConfirmResult); ok {
		if result.Confirmed {
			switch result.Action {
			case "delete":
				if pod, ok := result.Data.(*k8s.PodInfo); ok {
					d.statusMsg = "Deleting pod..."
					return d, func() tea.Msg {
						return DeletePodRequest{
							Namespace: pod.Namespace,
							PodName:   pod.Name,
						}
					}
				}
			case "exec", "port-forward":
				// Execute the pending action
				if d.pendingAction != nil {
					cmdStr := d.pendingAction.Command
					d.pendingAction = nil
					c := exec.Command("sh", "-c", cmdStr)
					return d, tea.ExecProcess(c, func(err error) tea.Msg {
						if err != nil {
							return ExecFinishedMsg{Err: err}
						}
						return ExecFinishedMsg{}
					})
				}
			}
		} else {
			// Cancelled - clear pending action
			d.pendingAction = nil
		}
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Confirm dialog takes highest priority
		if d.confirmDialog.IsVisible() {
			d.confirmDialog, cmd = d.confirmDialog.Update(msg)
			return d, cmd
		}

		// Result viewer takes priority (for describe output etc)
		if d.resultViewer.IsVisible() {
			d.resultViewer, cmd = d.resultViewer.Update(msg)
			return d, cmd
		}

		// Pod action menu takes priority
		if d.podActionMenu.IsVisible() {
			d.podActionMenu, cmd = d.podActionMenu.Update(msg)
			return d, cmd
		}

		// Action menu (copy commands) takes priority
		if d.actionMenu.IsVisible() {
			d.actionMenu, cmd = d.actionMenu.Update(msg)
			return d, cmd
		}

		if d.help.IsVisible() {
			if msg.String() == "?" || msg.String() == "esc" {
				d.help.Hide()
				return d, nil
			}
			return d, nil
		}

		// When logs panel is searching, pass all keys to it (except esc/enter handled above)
		if d.focus == FocusLogs && d.logs.IsSearching() {
			d.logs, cmd = d.logs.Update(msg)
			return d, cmd
		}

		// Clear status message on any key press
		d.statusMsg = ""

		switch {
		case key.Matches(msg, d.keys.PodActions):
			if d.pod != nil {
				var containers []string
				for _, c := range d.pod.Containers {
					containers = append(containers, c.Name)
				}
				items := components.PodActions(d.namespace, d.pod.Name, containers)
				d.podActionMenu.Show("Pod Actions", items)
			}
			return d, nil

		case key.Matches(msg, d.keys.CopyCommands):
			if d.pod != nil {
				var containers []string
				for _, c := range d.pod.Containers {
					containers = append(containers, c.Name)
				}
				selectedContainer := d.logs.SelectedContainer()
				items := components.KubectlCommands(d.namespace, d.pod.Name, selectedContainer, containers)
				d.actionMenu.Show("Copy kubectl command", items)
			}
			return d, nil

		case key.Matches(msg, d.keys.Help):
			d.help.Toggle()
			return d, nil

		case key.Matches(msg, d.keys.NextPanel):
			d.nextPanel()
			return d, nil

		case key.Matches(msg, d.keys.PrevPanel):
			d.prevPanel()
			return d, nil

		case key.Matches(msg, d.keys.Panel1):
			d.focus = FocusLogs
			return d, nil

		case key.Matches(msg, d.keys.Panel2):
			d.focus = FocusEvents
			return d, nil

		case key.Matches(msg, d.keys.Panel3):
			d.focus = FocusMetrics
			return d, nil

		case key.Matches(msg, d.keys.Panel4):
			d.focus = FocusManifest
			return d, nil

		case key.Matches(msg, d.keys.ToggleFullView):
			d.fullscreen = !d.fullscreen
			return d, nil

		// Arrow key navigation between panels (2x2 grid)
		// Layout: Logs(0) | Events(1)
		//         Metrics(2) | Manifest(3)
		// Note: When FocusMetrics is active, arrows are passed to MetricsPanel for inner box navigation
		case msg.String() == "left":
			if d.focus == FocusMetrics {
				// Let MetricsPanel handle left/right for inner box navigation
				break
			}
			switch d.focus {
			case FocusEvents:
				d.focus = FocusLogs
			case FocusManifest:
				d.focus = FocusMetrics
			case FocusLogs:
				d.focus = FocusEvents // wrap
			}
			return d, nil

		case msg.String() == "right":
			if d.focus == FocusMetrics {
				// Let MetricsPanel handle left/right for inner box navigation
				break
			}
			switch d.focus {
			case FocusLogs:
				d.focus = FocusEvents
			case FocusEvents:
				d.focus = FocusLogs // wrap
			case FocusManifest:
				d.focus = FocusMetrics // wrap
			}
			return d, nil

		case msg.String() == "up":
			if d.focus == FocusMetrics {
				// Let MetricsPanel handle up/down for scrolling
				break
			}
			switch d.focus {
			case FocusManifest:
				d.focus = FocusEvents
			case FocusLogs:
				d.focus = FocusMetrics // wrap
			case FocusEvents:
				d.focus = FocusManifest // wrap
			}
			return d, nil

		case msg.String() == "down":
			if d.focus == FocusMetrics {
				// Let MetricsPanel handle up/down for scrolling
				break
			}
			switch d.focus {
			case FocusLogs:
				d.focus = FocusMetrics
			case FocusEvents:
				d.focus = FocusManifest
			case FocusManifest:
				d.focus = FocusEvents // wrap
			}
			return d, nil

		case key.Matches(msg, d.keys.Enter):
			// Enter on Logs panel: if fullscreen, copy logs; otherwise toggle fullscreen
			if d.focus == FocusLogs {
				if d.fullscreen {
					// In fullscreen, pass Enter to logs panel for copy
					d.logs, cmd = d.logs.Update(msg)
					return d, cmd
				}
				d.fullscreen = !d.fullscreen
				return d, nil
			}
			// Enter on Events panel toggles fullscreen
			if d.focus == FocusEvents {
				d.fullscreen = !d.fullscreen
				return d, nil
			}
			// Enter on Resource Usage panel shows detailed resource info
			if d.focus == FocusMetrics && d.pod != nil {
				content := d.renderDetailedResources()
				d.resultViewer.Show("Resource Details: "+d.pod.Name, content, d.width-4, d.height-4)
				return d, nil
			}
			// Enter on Pod Details panel shows kubectl describe
			if d.focus == FocusManifest && d.pod != nil {
				d.statusMsg = "Loading describe..."
				podName := d.pod.Name
				namespace := d.namespace
				return d, func() tea.Msg {
					cmdStr := "kubectl describe pod " + podName + " -n " + namespace
					c := exec.Command("sh", "-c", cmdStr)
					output, err := c.CombinedOutput()
					if err != nil {
						return DescribeOutputMsg{Err: err}
					}
					return DescribeOutputMsg{
						Title:   "Pod: " + podName,
						Content: string(output),
					}
				}
			}
		}
	}

	switch d.focus {
	case FocusLogs:
		d.logs, cmd = d.logs.Update(msg)
		cmds = append(cmds, cmd)
	case FocusEvents:
		d.events, cmd = d.events.Update(msg)
		cmds = append(cmds, cmd)
	case FocusMetrics:
		d.metrics, cmd = d.metrics.Update(msg)
		cmds = append(cmds, cmd)
	case FocusManifest:
		d.manifest, cmd = d.manifest.Update(msg)
		cmds = append(cmds, cmd)
	}

	return d, tea.Batch(cmds...)
}

func (d *Dashboard) nextPanel() {
	d.focus = (d.focus + 1) % 4
}

func (d *Dashboard) prevPanel() {
	d.focus = (d.focus + 3) % 4
}

func (d Dashboard) View() string {
	if d.pod == nil {
		return styles.PanelStyle.Render("No pod selected")
	}

	var b strings.Builder

	// Show breadcrumb with optional status message
	breadcrumbView := d.breadcrumb.View()
	if d.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(styles.Success).
			Bold(true)
		breadcrumbView = breadcrumbView + "  " + statusStyle.Render(d.statusMsg)
	}
	b.WriteString(breadcrumbView)
	b.WriteString("\n")

	if d.fullscreen {
		// Render only the focused panel in fullscreen
		b.WriteString(d.renderFullscreenPanel())
	} else {
		// Normal 4-panel layout
		topRow := d.renderTopRow()
		bottomRow := d.renderBottomRow()

		b.WriteString(topRow)
		b.WriteString("\n")
		b.WriteString(bottomRow)
	}

	content := b.String()

	// Render confirm dialog as overlay (highest priority)
	if d.confirmDialog.IsVisible() {
		return d.renderFloatingDialog(d.confirmDialog.View())
	}

	// Render result viewer as overlay (for describe output etc)
	if d.resultViewer.IsVisible() {
		return d.renderFloatingDialog(d.resultViewer.View())
	}

	// Render pod action menu as overlay
	if d.podActionMenu.IsVisible() {
		return d.renderFloatingDialog(d.podActionMenu.View())
	}

	// Render action menu as overlay if visible
	if d.actionMenu.IsVisible() {
		return d.renderFloatingDialog(d.actionMenu.View())
	}

	if d.help.IsVisible() {
		return d.renderFloatingDialog(d.help.View())
	}

	return content
}

func (d Dashboard) renderFullscreenPanel() string {
	panelWidth := d.width - 4
	panelHeight := d.height - 8

	var content string
	switch d.focus {
	case FocusLogs:
		d.logs.SetSize(panelWidth, panelHeight)
		content = d.logs.View()
	case FocusEvents:
		d.events.SetSize(panelWidth, panelHeight)
		content = d.events.View()
	case FocusMetrics:
		d.metrics.SetSize(panelWidth, panelHeight)
		content = d.metrics.View()
	case FocusManifest:
		d.manifest.SetSize(panelWidth, panelHeight)
		content = d.manifest.View()
	}

	return d.wrapPanel(content, panelWidth, panelHeight, true)
}

func (d Dashboard) renderTopRow() string {
	halfWidth := (d.width - 1) / 2
	panelHeight := (d.height - 4) / 2

	d.logs.SetSize(halfWidth-4, panelHeight-2)
	d.events.SetSize(halfWidth-4, panelHeight-2)

	logsView := d.wrapPanel(d.logs.View(), halfWidth-2, panelHeight, d.focus == FocusLogs)
	eventsView := d.wrapPanel(d.events.View(), halfWidth-2, panelHeight, d.focus == FocusEvents)

	return lipgloss.JoinHorizontal(lipgloss.Top, logsView, eventsView)
}

func (d Dashboard) renderBottomRow() string {
	halfWidth := (d.width - 1) / 2
	panelHeight := (d.height - 4) / 2

	d.metrics.SetSize(halfWidth-4, panelHeight-2)
	d.manifest.SetSize(halfWidth-4, panelHeight-2)

	metricsView := d.wrapPanel(d.metrics.View(), halfWidth-2, panelHeight, d.focus == FocusMetrics)
	manifestView := d.wrapPanel(d.manifest.View(), halfWidth-2, panelHeight, d.focus == FocusManifest)

	return lipgloss.JoinHorizontal(lipgloss.Top, metricsView, manifestView)
}

func (d Dashboard) wrapPanel(content string, width, height int, active bool) string {
	style := styles.PanelStyle
	if active {
		style = styles.ActivePanelStyle
	}

	return style.
		Width(width).
		Height(height).
		Render(content)
}

func (d Dashboard) renderFloatingDialog(dialogContent string) string {
	return lipgloss.Place(
		d.width,
		d.height-4,
		lipgloss.Center,
		lipgloss.Center,
		dialogContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Background),
	)
}

func (d *Dashboard) SetPod(pod *k8s.PodInfo) {
	d.pod = pod
	d.manifest.SetPod(pod)
	d.metrics.SetPod(pod)

	// Extract container names for logs panel
	var containerNames []string
	for _, c := range pod.Containers {
		containerNames = append(containerNames, c.Name)
	}
	d.logs.SetContainers(containerNames)
}

func (d *Dashboard) SetLogs(logs []k8s.LogLine) {
	d.logs.SetLogs(logs)
}

func (d *Dashboard) SetEvents(events []k8s.EventInfo) {
	d.events.SetEvents(events)
}

func (d *Dashboard) SetMetrics(metrics *k8s.PodMetrics) {
	d.metrics.SetMetrics(metrics)
}

func (d *Dashboard) SetRelated(related *k8s.RelatedResources) {
	d.related = related
	d.manifest.SetRelated(related)
}

func (d *Dashboard) SetNode(node *k8s.NodeInfo) {
	d.metrics.SetNode(node)
}

func (d *Dashboard) SetHelpers(helpers []k8s.DebugHelper) {
	d.manifest.SetHelpers(helpers)
}

func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.breadcrumb.SetWidth(width)
	d.help.SetSize(width, height)
}

func (d *Dashboard) SetBreadcrumb(items ...string) {
	d.breadcrumb.SetItems(items...)
}

func (d *Dashboard) SetContext(ctx string) {
	d.context = ctx
}

func (d *Dashboard) SetNamespace(ns string) {
	d.namespace = ns
}

func (d Dashboard) Focus() PanelFocus {
	return d.focus
}

func (d Dashboard) HelpVisible() bool {
	return d.help.IsVisible()
}

func (d Dashboard) ShortHelp() string {
	return d.help.ShortHelp()
}

// Logs panel state getters for app to react to
func (d Dashboard) LogsSelectedContainer() string {
	return d.logs.SelectedContainer()
}

func (d Dashboard) LogsShowPrevious() bool {
	return d.logs.ShowPrevious()
}

func (d *Dashboard) GetPod() *k8s.PodInfo {
	return d.pod
}

func (d Dashboard) IsLogsSearching() bool {
	return d.logs.IsSearching()
}

func (d Dashboard) HasActiveOverlay() bool {
	return d.resultViewer.IsVisible() ||
		d.confirmDialog.IsVisible() ||
		d.podActionMenu.IsVisible() ||
		d.actionMenu.IsVisible() ||
		d.help.IsVisible()
}

func (d Dashboard) IsFullscreen() bool {
	return d.fullscreen
}

func (d *Dashboard) CloseFullscreen() {
	d.fullscreen = false
}

func (d Dashboard) renderDetailedResources() string {
	if d.pod == nil {
		return "No pod selected"
	}

	var b strings.Builder

	// Pod-level info
	b.WriteString(styles.SubtitleStyle.Render("Pod Info"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "QoS Class:", d.pod.QoSClass))
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "Service Account:", d.pod.ServiceAccount))
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "Restart Policy:", d.pod.RestartPolicy))
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "DNS Policy:", d.pod.DNSPolicy))
	b.WriteString(fmt.Sprintf("  %-22s %ds\n", "Termination Grace:", d.pod.TerminationGracePeriod))
	if d.pod.PriorityClassName != "" {
		b.WriteString(fmt.Sprintf("  %-22s %s\n", "Priority Class:", d.pod.PriorityClassName))
	}
	if d.pod.Priority != nil {
		b.WriteString(fmt.Sprintf("  %-22s %d\n", "Priority:", *d.pod.Priority))
	}
	b.WriteString("\n")

	// Network info
	b.WriteString(styles.SubtitleStyle.Render("Network"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "Pod IP:", d.pod.IP))
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "Host IP:", d.pod.HostIP))
	b.WriteString(fmt.Sprintf("  %-22s %s\n", "Node:", d.pod.Node))
	if d.pod.StartTime != "" {
		b.WriteString(fmt.Sprintf("  %-22s %s\n", "Started:", d.pod.StartTime))
	}
	b.WriteString("\n")

	// Services (right after Network)
	if d.related != nil && len(d.related.Services) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Services"))
		b.WriteString("\n")
		for _, svc := range d.related.Services {
			typeStyle := styles.StatusMuted
			if svc.Type == "LoadBalancer" {
				typeStyle = styles.StatusRunning
			} else if svc.Type == "NodePort" {
				typeStyle = styles.LogContainer
			}
			b.WriteString(fmt.Sprintf("  • %s\n", styles.LogContainer.Render(svc.Name)))
			b.WriteString(fmt.Sprintf("    Type:       %s\n", typeStyle.Render(svc.Type)))
			b.WriteString(fmt.Sprintf("    ClusterIP:  %s\n", svc.ClusterIP))
			b.WriteString(fmt.Sprintf("    Ports:      %s\n", svc.Ports))
			b.WriteString(fmt.Sprintf("    Endpoints:  %d\n", svc.Endpoints))
		}
		b.WriteString("\n")
	}

	// Ingresses (after Services) - with detailed routing info
	if d.related != nil && len(d.related.Ingresses) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Ingresses"))
		b.WriteString("\n")
		for _, ing := range d.related.Ingresses {
			// Ingress name and class
			classInfo := ""
			if ing.Class != "" {
				classInfo = fmt.Sprintf(" (%s)", ing.Class)
			}
			b.WriteString(fmt.Sprintf("  • %s%s\n", styles.LogContainer.Render(ing.Name), styles.StatusMuted.Render(classInfo)))

			// TLS info
			if ing.TLS {
				tlsInfo := styles.StatusRunning.Render("TLS enabled")
				if len(ing.TLSSecrets) > 0 {
					tlsInfo += fmt.Sprintf(" [%s]", strings.Join(ing.TLSSecrets, ", "))
				}
				b.WriteString(fmt.Sprintf("    %s\n", tlsInfo))
			}

			// Hosts
			if len(ing.Hosts) > 0 {
				b.WriteString(fmt.Sprintf("    Hosts:      %s\n", strings.Join(ing.Hosts, ", ")))
			}

			// Rules with paths (routing details)
			for _, rule := range ing.Rules {
				if len(rule.Paths) > 0 {
					for _, path := range rule.Paths {
						serviceStyle := lipgloss.NewStyle().Foreground(styles.Secondary)
						routeInfo := fmt.Sprintf("%s → %s:%s",
							path.Path,
							serviceStyle.Render(path.ServiceName),
							path.ServicePort)
						if path.PathType != "" && path.PathType != "Prefix" {
							routeInfo += fmt.Sprintf(" [%s]", path.PathType)
						}
						b.WriteString(fmt.Sprintf("    Route:      %s\n", routeInfo))
					}
				}
			}

			// Important annotations for debugging
			if len(ing.Annotations) > 0 {
				b.WriteString("    Annotations:\n")
				for k, v := range ing.Annotations {
					// Shorten annotation key for display
					shortKey := k
					if strings.Contains(k, "nginx.ingress.kubernetes.io/") {
						shortKey = strings.Replace(k, "nginx.ingress.kubernetes.io/", "nginx/", 1)
					} else if strings.Contains(k, "traefik.ingress.kubernetes.io/") {
						shortKey = strings.Replace(k, "traefik.ingress.kubernetes.io/", "traefik/", 1)
					} else if strings.Contains(k, "cert-manager.io/") {
						shortKey = strings.Replace(k, "cert-manager.io/", "cert/", 1)
					}
					b.WriteString(fmt.Sprintf("      %s: %s\n", styles.StatusMuted.Render(shortKey), v))
				}
			}
		}
		b.WriteString("\n")
	}

	// VirtualServices (Istio) - after Ingresses
	if d.related != nil && len(d.related.VirtualServices) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("VirtualServices (Istio)"))
		b.WriteString("\n")
		for _, vs := range d.related.VirtualServices {
			b.WriteString(fmt.Sprintf("  • %s\n", styles.LogContainer.Render(vs.Name)))
			if len(vs.Hosts) > 0 {
				b.WriteString(fmt.Sprintf("    Hosts:     %s\n", strings.Join(vs.Hosts, ", ")))
			}
			if len(vs.Gateways) > 0 {
				b.WriteString(fmt.Sprintf("    Gateways:  %s\n", strings.Join(vs.Gateways, ", ")))
			}
			for _, route := range vs.Routes {
				destStyle := lipgloss.NewStyle().Foreground(styles.Secondary)
				routeInfo := fmt.Sprintf("%s → %s:%d",
					route.Match,
					destStyle.Render(route.Destination),
					route.Port)
				if route.Weight > 0 && route.Weight < 100 {
					routeInfo += fmt.Sprintf(" (weight: %d%%)", route.Weight)
				}
				b.WriteString(fmt.Sprintf("    Route:     %s\n", routeInfo))
			}
		}
		b.WriteString("\n")
	}

	// Gateways (Istio) - show details of referenced gateways
	if d.related != nil && len(d.related.Gateways) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Gateways (Istio)"))
		b.WriteString("\n")
		for _, gw := range d.related.Gateways {
			gwRef := gw.Name
			if gw.Namespace != "" && gw.Namespace != d.pod.Namespace {
				gwRef = gw.Namespace + "/" + gw.Name
			}
			b.WriteString(fmt.Sprintf("  • %s\n", styles.LogContainer.Render(gwRef)))
			for _, srv := range gw.Servers {
				protocolStyle := styles.StatusMuted
				if srv.Protocol == "HTTPS" || srv.TLS != "" {
					protocolStyle = styles.StatusRunning
				}
				portInfo := fmt.Sprintf("%d/%s", srv.Port, protocolStyle.Render(srv.Protocol))
				if srv.TLS != "" {
					portInfo += fmt.Sprintf(" [TLS: %s]", srv.TLS)
				}
				b.WriteString(fmt.Sprintf("    Port:      %s\n", portInfo))
				if len(srv.Hosts) > 0 {
					b.WriteString(fmt.Sprintf("    Hosts:     %s\n", strings.Join(srv.Hosts, ", ")))
				}
			}
		}
		b.WriteString("\n")
	}

	// Node Selector
	if len(d.pod.NodeSelector) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Node Selector"))
		b.WriteString("\n")
		for k, v := range d.pod.NodeSelector {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
		b.WriteString("\n")
	}

	// Tolerations
	if len(d.pod.Tolerations) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Tolerations"))
		b.WriteString("\n")
		for _, t := range d.pod.Tolerations {
			if t.Key == "" {
				b.WriteString("  • (all taints)\n")
			} else {
				tolStr := fmt.Sprintf("  • %s", t.Key)
				if t.Value != "" {
					tolStr += "=" + t.Value
				}
				if t.Effect != "" {
					tolStr += " :" + t.Effect
				}
				b.WriteString(tolStr + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Init Containers
	if len(d.pod.InitContainers) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Init Containers"))
		b.WriteString("\n")
		for _, c := range d.pod.InitContainers {
			stateStyle := styles.GetStatusStyle(c.State)
			state := c.State
			if c.Reason != "" {
				state += " (" + c.Reason + ")"
			}
			b.WriteString(fmt.Sprintf("  • %s: %s\n", c.Name, stateStyle.Render(state)))
			b.WriteString(fmt.Sprintf("    Image: %s\n", c.Image))
		}
		b.WriteString("\n")
	}

	// Container details
	for _, c := range d.pod.Containers {
		b.WriteString(styles.LogContainer.Render("Container: " + c.Name))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %-20s %s\n", "Image:", c.Image))
		b.WriteString(fmt.Sprintf("  %-20s %s\n", "Pull Policy:", c.ImagePullPolicy))
		stateStyle := styles.GetStatusStyle(c.State)
		b.WriteString(fmt.Sprintf("  %-20s %s\n", "State:", stateStyle.Render(c.State)))
		if c.StartedAt != "" {
			b.WriteString(fmt.Sprintf("  %-20s %s\n", "Started:", c.StartedAt))
		}
		if c.ExitCode != nil {
			b.WriteString(fmt.Sprintf("  %-20s %d\n", "Exit Code:", *c.ExitCode))
		}
		b.WriteString(fmt.Sprintf("  %-20s %d\n", "Restarts:", c.RestartCount))
		b.WriteString(fmt.Sprintf("  %-20s %d\n", "Env Vars:", c.EnvVarCount))
		b.WriteString("\n")

		// Resources
		b.WriteString(styles.SubtitleStyle.Render("  Resources"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    %-18s %s\n", "CPU Request:", formatResource(c.Resources.CPURequest)))
		b.WriteString(fmt.Sprintf("    %-18s %s\n", "CPU Limit:", formatResource(c.Resources.CPULimit)))
		b.WriteString(fmt.Sprintf("    %-18s %s\n", "Mem Request:", formatResource(c.Resources.MemoryRequest)))
		b.WriteString(fmt.Sprintf("    %-18s %s\n", "Mem Limit:", formatResource(c.Resources.MemoryLimit)))
		b.WriteString("\n")

		// Ports
		if len(c.Ports) > 0 {
			b.WriteString(styles.SubtitleStyle.Render("  Ports"))
			b.WriteString("\n")
			for _, p := range c.Ports {
				portStr := fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol)
				if p.Name != "" {
					portStr = p.Name + ": " + portStr
				}
				b.WriteString("    • " + portStr + "\n")
			}
			b.WriteString("\n")
		}

		// Probes
		b.WriteString(styles.SubtitleStyle.Render("  Probes"))
		b.WriteString("\n")
		if c.LivenessProbe != nil {
			b.WriteString("    Liveness:   " + formatProbe(c.LivenessProbe) + "\n")
		} else {
			b.WriteString("    Liveness:   " + styles.StatusMuted.Render("not configured") + "\n")
		}
		if c.ReadinessProbe != nil {
			b.WriteString("    Readiness:  " + formatProbe(c.ReadinessProbe) + "\n")
		} else {
			b.WriteString("    Readiness:  " + styles.StatusMuted.Render("not configured") + "\n")
		}
		if c.StartupProbe != nil {
			b.WriteString("    Startup:    " + formatProbe(c.StartupProbe) + "\n")
		}
		b.WriteString("\n")

		// Security Context
		b.WriteString(styles.SubtitleStyle.Render("  Security Context"))
		b.WriteString("\n")
		if c.SecurityContext != nil {
			hasContent := false
			if c.SecurityContext.RunAsUser != nil {
				b.WriteString(fmt.Sprintf("    %-18s %d\n", "Run As User:", *c.SecurityContext.RunAsUser))
				hasContent = true
			}
			if c.SecurityContext.RunAsGroup != nil {
				b.WriteString(fmt.Sprintf("    %-18s %d\n", "Run As Group:", *c.SecurityContext.RunAsGroup))
				hasContent = true
			}
			if c.SecurityContext.RunAsNonRoot != nil && *c.SecurityContext.RunAsNonRoot {
				b.WriteString(fmt.Sprintf("    %-18s %s\n", "Run As Non-Root:", styles.StatusRunning.Render("yes")))
				hasContent = true
			}
			if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
				b.WriteString(fmt.Sprintf("    %-18s %s\n", "Privileged:", styles.StatusError.Render("YES")))
				hasContent = true
			}
			if c.SecurityContext.ReadOnlyRoot != nil && *c.SecurityContext.ReadOnlyRoot {
				b.WriteString(fmt.Sprintf("    %-18s %s\n", "Read-Only Root:", styles.StatusRunning.Render("yes")))
				hasContent = true
			}
			if !hasContent {
				b.WriteString("    (default)\n")
			}
		} else {
			b.WriteString("    (default)\n")
		}
		b.WriteString("\n")

		// Volume Mounts
		if len(c.VolumeMounts) > 0 {
			b.WriteString(styles.SubtitleStyle.Render("  Volume Mounts"))
			b.WriteString("\n")
			for _, vm := range c.VolumeMounts {
				ro := ""
				if vm.ReadOnly {
					ro = " (ro)"
				}
				b.WriteString(fmt.Sprintf("    • %s → %s%s\n", vm.Name, vm.MountPath, ro))
			}
			b.WriteString("\n")
		}
	}

	// Volumes
	if len(d.pod.Volumes) > 0 {
		b.WriteString(styles.SubtitleStyle.Render("Volumes"))
		b.WriteString("\n")
		for _, v := range d.pod.Volumes {
			if v.Source != "" {
				b.WriteString(fmt.Sprintf("  • %s (%s: %s)\n", v.Name, v.Type, v.Source))
			} else {
				b.WriteString(fmt.Sprintf("  • %s (%s)\n", v.Name, v.Type))
			}
		}
		b.WriteString("\n")
	}

	// Related ConfigMaps and Secrets
	if d.related != nil {
		// ConfigMaps used
		if len(d.related.ConfigMaps) > 0 {
			b.WriteString(styles.SubtitleStyle.Render("ConfigMaps Used"))
			b.WriteString("\n")
			for _, cm := range d.related.ConfigMaps {
				b.WriteString(fmt.Sprintf("  • %s\n", cm))
			}
			b.WriteString("\n")
		}

		// Secrets used
		if len(d.related.Secrets) > 0 {
			b.WriteString(styles.SubtitleStyle.Render("Secrets Used"))
			b.WriteString("\n")
			for _, s := range d.related.Secrets {
				b.WriteString(fmt.Sprintf("  • %s\n", s))
			}
		}
	}

	return b.String()
}

func formatResource(v string) string {
	if v == "" || v == "0" {
		return styles.StatusMuted.Render("not set")
	}
	return v
}

func formatProbe(p *k8s.ProbeInfo) string {
	if p == nil {
		return "not configured"
	}

	var result string
	switch p.Type {
	case "HTTP":
		scheme := p.Scheme
		if scheme == "" {
			scheme = "HTTP"
		}
		result = scheme + " " + p.Path + " :" + formatInt32(p.Port)
	case "TCP":
		result = "TCP :" + formatInt32(p.Port)
	case "Exec":
		result = "Exec: " + strings.Join(p.Command, " ")
	case "gRPC":
		result = "gRPC :" + formatInt32(p.Port)
	default:
		result = p.Type
	}

	result += " (delay:" + formatInt32(p.InitialDelay) + "s"
	result += " period:" + formatInt32(p.Period) + "s"
	result += " fail:" + formatInt32(p.FailureThreshold) + ")"

	return result
}

func formatInt32(v int32) string {
	return fmt.Sprintf("%d", v)
}

func formatInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}

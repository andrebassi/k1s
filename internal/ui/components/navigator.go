package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k8sdebug/internal/k8s"
	"github.com/andrebassi/k8sdebug/internal/ui/keys"
	"github.com/andrebassi/k8sdebug/internal/ui/styles"
)

type NavigatorMode int

const (
	ModeWorkloads NavigatorMode = iota
	ModePods
	ModeNamespace
	ModeResourceType
)

type Navigator struct {
	workloads    []k8s.WorkloadInfo
	pods         []k8s.PodInfo
	namespaces   []string
	cursor       int
	mode         NavigatorMode
	width        int
	height       int
	searchInput  textinput.Model
	searching    bool
	searchQuery  string
	resourceType k8s.ResourceType
	keys         keys.KeyMap
}

func NewNavigator() Navigator {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 50
	ti.Width = 30

	return Navigator{
		resourceType: k8s.ResourceDeployments,
		searchInput:  ti,
		keys:         keys.DefaultKeyMap(),
	}
}

func (n Navigator) Init() tea.Cmd {
	return nil
}

func (n Navigator) Update(msg tea.Msg) (Navigator, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When searching, only handle search-specific keys
		if n.searching {
			switch msg.String() {
			case "enter", "esc":
				n.searching = false
				n.searchQuery = n.searchInput.Value()
				n.cursor = 0 // Reset cursor after filter
			default:
				n.searchInput, cmd = n.searchInput.Update(msg)
				// Live filter as user types
				n.searchQuery = n.searchInput.Value()
				n.cursor = 0
			}
			return n, cmd
		}

		// Normal navigation mode
		switch {
		case key.Matches(msg, n.keys.Up):
			n.moveUp()
		case key.Matches(msg, n.keys.Down):
			n.moveDown()
		case key.Matches(msg, n.keys.Home):
			n.cursor = 0
		case key.Matches(msg, n.keys.End):
			n.cursor = n.maxItems() - 1
			if n.cursor < 0 {
				n.cursor = 0
			}
		case key.Matches(msg, n.keys.PageUp):
			n.pageUp()
		case key.Matches(msg, n.keys.PageDown):
			n.pageDown()
		case key.Matches(msg, n.keys.Search):
			n.searching = true
			n.searchInput.SetValue(n.searchQuery)
			n.searchInput.Focus()
			return n, textinput.Blink
		case key.Matches(msg, n.keys.Clear):
			n.ClearSearch()
		}
	}

	return n, nil
}

func (n *Navigator) moveUp() {
	if n.cursor > 0 {
		n.cursor--
	}
}

func (n *Navigator) moveDown() {
	max := n.maxItems() - 1
	if n.cursor < max {
		n.cursor++
	}
}

func (n *Navigator) pageUp() {
	n.cursor -= 10
	if n.cursor < 0 {
		n.cursor = 0
	}
}

func (n *Navigator) pageDown() {
	max := n.maxItems() - 1
	n.cursor += 10
	if n.cursor > max {
		n.cursor = max
	}
	if n.cursor < 0 {
		n.cursor = 0
	}
}

func (n Navigator) maxItems() int {
	switch n.mode {
	case ModeWorkloads:
		return len(n.filteredWorkloads())
	case ModePods:
		return len(n.filteredPods())
	case ModeNamespace:
		return len(n.filteredNamespaces())
	case ModeResourceType:
		return len(k8s.AllResourceTypes)
	}
	return 0
}

func (n Navigator) View() string {
	var b strings.Builder

	// Title with mode indicator
	b.WriteString(n.renderHeader())
	b.WriteString("\n")

	// Search bar or filter indicator
	if n.searching {
		searchStyle := lipgloss.NewStyle().
			Foreground(styles.Text).
			Background(styles.Surface).
			Padding(0, 1)
		b.WriteString(searchStyle.Render("/ " + n.searchInput.View()))
		b.WriteString("\n\n")
	} else if n.searchQuery != "" {
		filterStyle := lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true)
		clearHint := styles.HelpDescStyle.Render(" (c to clear)")
		b.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s", n.searchQuery)))
		b.WriteString(clearHint)
		b.WriteString("\n\n")
	} else {
		b.WriteString("\n")
	}

	// Content based on mode
	switch n.mode {
	case ModeWorkloads:
		b.WriteString(n.renderWorkloads())
	case ModePods:
		b.WriteString(n.renderPods())
	case ModeNamespace:
		b.WriteString(n.renderNamespaces())
	case ModeResourceType:
		b.WriteString(n.renderResourceTypes())
	}

	return b.String()
}

func (n Navigator) renderHeader() string {
	var icon, title string

	switch n.mode {
	case ModeWorkloads:
		icon = "◈"
		title = strings.ToUpper(string(n.resourceType))
	case ModePods:
		icon = "●"
		title = "PODS"
	case ModeNamespace:
		icon = "◉"
		title = "SELECT NAMESPACE"
	case ModeResourceType:
		icon = "◆"
		title = "SELECT RESOURCE TYPE"
	}

	iconStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Text).Bold(true)

	return iconStyle.Render(icon) + " " + titleStyle.Render(title)
}

func (n Navigator) renderWorkloads() string {
	workloads := n.filteredWorkloads()
	if len(workloads) == 0 {
		if n.searchQuery != "" {
			return styles.StatusMuted.Render("  No workloads match filter")
		}
		return styles.StatusMuted.Render("  No workloads found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("  %-32s %-10s %-15s %-8s", "NAME", "READY", "STATUS", "AGE")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Items
	visible := n.visibleRange(len(workloads))
	for i := visible.start; i < visible.end; i++ {
		w := workloads[i]
		b.WriteString(n.renderWorkloadRow(w, i == n.cursor))
		b.WriteString("\n")
	}

	// Scroll indicator
	b.WriteString(n.renderScrollIndicator(visible, len(workloads)))
	return b.String()
}

func (n Navigator) renderWorkloadRow(w k8s.WorkloadInfo, selected bool) string {
	cursor := "  "
	if selected {
		cursor = styles.CursorStyle.Render("> ")
	}

	name := styles.Truncate(w.Name, 32)
	statusStyle := styles.GetStatusStyle(w.Status)

	if selected {
		rowStyle := lipgloss.NewStyle().Background(styles.Surface)
		return rowStyle.Render(fmt.Sprintf("%s%-32s %-10s %-15s %-8s",
			cursor, name, w.Ready, statusStyle.Render(w.Status), w.Age))
	}

	return fmt.Sprintf("%s%-32s %-10s %-15s %-8s",
		cursor, name, w.Ready, statusStyle.Render(w.Status), w.Age)
}

func (n Navigator) renderPods() string {
	pods := n.filteredPods()
	if len(pods) == 0 {
		if n.searchQuery != "" {
			return styles.StatusMuted.Render("  No pods match filter")
		}
		return styles.StatusMuted.Render("  No pods found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("  %-38s %-8s %-18s %-8s %-6s", "NAME", "READY", "STATUS", "RESTARTS", "AGE")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Items
	visible := n.visibleRange(len(pods))
	for i := visible.start; i < visible.end; i++ {
		p := pods[i]
		b.WriteString(n.renderPodRow(p, i == n.cursor))
		b.WriteString("\n")
	}

	// Scroll indicator
	b.WriteString(n.renderScrollIndicator(visible, len(pods)))
	return b.String()
}

func (n Navigator) renderPodRow(p k8s.PodInfo, selected bool) string {
	cursor := "  "
	if selected {
		cursor = styles.CursorStyle.Render("> ")
	}

	name := styles.Truncate(p.Name, 38)
	statusStyle := styles.GetStatusStyle(p.Status)

	restarts := fmt.Sprintf("%d", p.Restarts)
	if p.Restarts > 0 {
		restarts = styles.StatusError.Render(restarts)
	}

	if selected {
		rowStyle := lipgloss.NewStyle().Background(styles.Surface)
		return rowStyle.Render(fmt.Sprintf("%s%-38s %-8s %-18s %-8s %-6s",
			cursor, name, p.Ready, statusStyle.Render(p.Status), restarts, p.Age))
	}

	return fmt.Sprintf("%s%-38s %-8s %-18s %-8s %-6s",
		cursor, name, p.Ready, statusStyle.Render(p.Status), restarts, p.Age)
}

func (n Navigator) renderNamespaces() string {
	namespaces := n.filteredNamespaces()
	if len(namespaces) == 0 {
		return styles.StatusMuted.Render("  No namespaces found")
	}

	var b strings.Builder

	// Table header
	header := fmt.Sprintf("  %-4s %-40s %-10s", "#", "NAMESPACE", "STATUS")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	visible := n.visibleRange(len(namespaces))

	for i := visible.start; i < visible.end; i++ {
		ns := namespaces[i]
		idx := fmt.Sprintf("%d", i+1)
		status := styles.StatusRunning.Render("Active")

		cursor := "  "
		if i == n.cursor {
			cursor = styles.CursorStyle.Render("> ")
			rowStyle := lipgloss.NewStyle().Background(styles.Surface)
			row := fmt.Sprintf("%s%-4s %-40s %-10s", cursor, idx, ns, status)
			b.WriteString(rowStyle.Render(row))
		} else {
			b.WriteString(fmt.Sprintf("%s%-4s %-40s %s", cursor, idx, ns, status))
		}
		b.WriteString("\n")
	}

	b.WriteString(n.renderScrollIndicator(visible, len(namespaces)))
	return b.String()
}

func (n Navigator) renderResourceTypes() string {
	var b strings.Builder

	// Table header
	header := fmt.Sprintf("  %-4s %-20s %-30s", "#", "TYPE", "DESCRIPTION")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	descriptions := map[k8s.ResourceType]string{
		k8s.ResourceDeployments:  "Manages ReplicaSets and Pods",
		k8s.ResourceStatefulSets: "Stateful workloads with identity",
		k8s.ResourceDaemonSets:   "Runs on every node",
		k8s.ResourceJobs:         "One-time batch tasks",
		k8s.ResourceCronJobs:     "Scheduled batch tasks",
	}

	for i, rt := range k8s.AllResourceTypes {
		idx := fmt.Sprintf("%d", i+1)
		desc := descriptions[rt]
		if desc == "" {
			desc = "-"
		}

		cursor := "  "
		if i == n.cursor {
			cursor = styles.CursorStyle.Render("> ")
			rowStyle := lipgloss.NewStyle().Background(styles.Surface)
			row := fmt.Sprintf("%s%-4s %-20s %-30s", cursor, idx, string(rt), desc)
			b.WriteString(rowStyle.Render(row))
		} else {
			b.WriteString(fmt.Sprintf("%s%-4s %-20s %-30s", cursor, idx, string(rt), desc))
		}
		b.WriteString("\n")
	}

	return b.String()
}

type visibleRange struct {
	start, end int
}

func (n Navigator) visibleRange(total int) visibleRange {
	maxVisible := n.height - 8
	if maxVisible < 5 {
		maxVisible = 15
	}

	start := 0
	end := total

	if total > maxVisible {
		start = n.cursor - maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > total {
			end = total
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
	}

	return visibleRange{start, end}
}

func (n Navigator) renderScrollIndicator(visible visibleRange, total int) string {
	if total == 0 {
		return ""
	}
	if visible.start > 0 || visible.end < total {
		percent := 0
		if total > 0 {
			percent = (n.cursor + 1) * 100 / total
		}
		return styles.StatusMuted.Render(fmt.Sprintf("\n  %d/%d (%d%%)", n.cursor+1, total, percent))
	}
	return styles.StatusMuted.Render(fmt.Sprintf("\n  %d items", total))
}

func (n Navigator) filteredWorkloads() []k8s.WorkloadInfo {
	if n.searchQuery == "" {
		return n.workloads
	}

	query := strings.ToLower(n.searchQuery)
	var filtered []k8s.WorkloadInfo
	for _, w := range n.workloads {
		if strings.Contains(strings.ToLower(w.Name), query) ||
			strings.Contains(strings.ToLower(w.Status), query) {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

func (n Navigator) filteredPods() []k8s.PodInfo {
	if n.searchQuery == "" {
		return n.pods
	}

	query := strings.ToLower(n.searchQuery)
	var filtered []k8s.PodInfo
	for _, p := range n.pods {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Status), query) ||
			strings.Contains(strings.ToLower(p.Node), query) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (n Navigator) filteredNamespaces() []string {
	if n.searchQuery == "" {
		return n.namespaces
	}

	query := strings.ToLower(n.searchQuery)
	var filtered []string
	for _, ns := range n.namespaces {
		if strings.Contains(strings.ToLower(ns), query) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

func (n *Navigator) SetWorkloads(workloads []k8s.WorkloadInfo) {
	n.workloads = workloads
	if n.cursor >= len(n.filteredWorkloads()) {
		n.cursor = 0
	}
}

func (n *Navigator) SetPods(pods []k8s.PodInfo) {
	n.pods = pods
	// Keep cursor in bounds but don't reset to 0 (for real-time refresh)
	if n.cursor >= len(pods) {
		n.cursor = len(pods) - 1
	}
	if n.cursor < 0 {
		n.cursor = 0
	}
}

func (n *Navigator) SetNamespaces(namespaces []string) {
	n.namespaces = namespaces
}

func (n *Navigator) SetResourceType(rt k8s.ResourceType) {
	n.resourceType = rt
}

func (n *Navigator) SetMode(mode NavigatorMode) {
	n.mode = mode
	n.cursor = 0
	n.ClearSearch()
}

func (n *Navigator) SetSize(width, height int) {
	n.width = width
	n.height = height
}

func (n Navigator) SelectedWorkload() *k8s.WorkloadInfo {
	workloads := n.filteredWorkloads()
	if n.cursor >= 0 && n.cursor < len(workloads) {
		return &workloads[n.cursor]
	}
	return nil
}

func (n Navigator) SelectedPod() *k8s.PodInfo {
	pods := n.filteredPods()
	if n.cursor >= 0 && n.cursor < len(pods) {
		return &pods[n.cursor]
	}
	return nil
}

func (n Navigator) SelectedNamespace() string {
	namespaces := n.filteredNamespaces()
	if n.cursor >= 0 && n.cursor < len(namespaces) {
		return namespaces[n.cursor]
	}
	return ""
}

func (n Navigator) SelectedResourceType() k8s.ResourceType {
	if n.cursor >= 0 && n.cursor < len(k8s.AllResourceTypes) {
		return k8s.AllResourceTypes[n.cursor]
	}
	return k8s.ResourceDeployments
}

func (n Navigator) Mode() NavigatorMode {
	return n.mode
}

func (n Navigator) IsSearching() bool {
	return n.searching
}

func (n Navigator) HasFilter() bool {
	return n.searchQuery != ""
}

func (n Navigator) ResourceType() k8s.ResourceType {
	return n.resourceType
}

func (n *Navigator) ClearSearch() {
	n.searchQuery = ""
	n.searchInput.SetValue("")
	n.searching = false
	n.cursor = 0
}

func (n *Navigator) CloseSearch() {
	n.searching = false
	n.searchQuery = n.searchInput.Value()
}

func (n Navigator) Render(width int) string {
	return lipgloss.NewStyle().Width(width).Render(n.View())
}

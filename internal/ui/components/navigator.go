package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/k8s"
	"github.com/andrebassi/k1s/internal/ui/keys"
	"github.com/andrebassi/k1s/internal/ui/styles"
)

type NavigatorMode int

const (
	ModeWorkloads NavigatorMode = iota
	ModeResources
	ModeNamespace
	ModeResourceType
)

type PodViewSection int

const (
	SectionPods PodViewSection = iota
	SectionConfigMaps
	SectionSecrets
	SectionDockerRegistry
)

type Navigator struct {
	workloads    []k8s.WorkloadInfo
	pods         []k8s.PodInfo
	configmaps   []k8s.ConfigMapInfo
	secrets      []k8s.SecretInfo
	namespaces   []string
	cursor       int
	section      PodViewSection // Current section in pods view
	sectionCursors [4]int       // Cursor for each section (Pods, ConfigMaps, Secrets, DockerRegistry)
	mode         NavigatorMode
	width        int
	height       int
	searchInput  textinput.Model
	searching    bool
	searchQuery  string
	resourceType k8s.ResourceType
	keys         keys.KeyMap
	panelActive  bool           // Whether this panel is active (for namespace mode with nodes)
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
		case key.Matches(msg, n.keys.NextPanel):
			// Tab to next section in pods view
			if n.mode == ModeResources {
				n.nextSection()
			}
		case key.Matches(msg, n.keys.PrevPanel):
			// Shift+Tab to previous section in pods view
			if n.mode == ModeResources {
				n.prevSection()
			}
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
	if n.mode == ModeResources {
		// Move within current section, or jump to previous section
		if n.sectionCursors[n.section] > 0 {
			n.sectionCursors[n.section]--
		} else {
			// At top of section, go to previous section (at its last item)
			n.prevSection()
			// Move cursor to last item of new section
			max := n.sectionMaxItems() - 1
			if max >= 0 {
				n.sectionCursors[n.section] = max
			}
		}
	} else {
		if n.cursor > 0 {
			n.cursor--
		}
	}
}

func (n *Navigator) moveDown() {
	if n.mode == ModeResources {
		// Move within current section, or jump to next section
		max := n.sectionMaxItems() - 1
		if n.sectionCursors[n.section] < max {
			n.sectionCursors[n.section]++
		} else {
			// At bottom of section, go to next section (at its first item)
			n.nextSection()
			// Move cursor to first item of new section
			n.sectionCursors[n.section] = 0
		}
	} else {
		max := n.maxItems() - 1
		if n.cursor < max {
			n.cursor++
		}
	}
}

func (n *Navigator) pageUp() {
	if n.mode == ModeResources {
		n.sectionCursors[n.section] -= 10
		if n.sectionCursors[n.section] < 0 {
			n.sectionCursors[n.section] = 0
		}
	} else {
		n.cursor -= 10
		if n.cursor < 0 {
			n.cursor = 0
		}
	}
}

func (n *Navigator) pageDown() {
	if n.mode == ModeResources {
		max := n.sectionMaxItems() - 1
		n.sectionCursors[n.section] += 10
		if n.sectionCursors[n.section] > max {
			n.sectionCursors[n.section] = max
		}
		if n.sectionCursors[n.section] < 0 {
			n.sectionCursors[n.section] = 0
		}
	} else {
		max := n.maxItems() - 1
		n.cursor += 10
		if n.cursor > max {
			n.cursor = max
		}
		if n.cursor < 0 {
			n.cursor = 0
		}
	}
}

func (n *Navigator) nextSection() {
	n.section = (n.section + 1) % 4
}

func (n *Navigator) prevSection() {
	n.section = (n.section + 3) % 4
}

func (n Navigator) sectionMaxItems() int {
	switch n.section {
	case SectionPods:
		return len(n.filteredPods())
	case SectionConfigMaps:
		return len(n.configmaps)
	case SectionSecrets:
		return len(n.filteredSecrets())
	case SectionDockerRegistry:
		return len(n.dockerRegistrySecrets())
	}
	return 0
}

// filteredSecrets returns secrets excluding docker registry type
func (n Navigator) filteredSecrets() []k8s.SecretInfo {
	var filtered []k8s.SecretInfo
	for _, s := range n.secrets {
		if s.Type != "kubernetes.io/dockerconfigjson" && s.Type != "kubernetes.io/dockercfg" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// dockerRegistrySecrets returns only docker registry secrets
func (n Navigator) dockerRegistrySecrets() []k8s.SecretInfo {
	var filtered []k8s.SecretInfo
	for _, s := range n.secrets {
		if s.Type == "kubernetes.io/dockerconfigjson" || s.Type == "kubernetes.io/dockercfg" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (n Navigator) maxItems() int {
	switch n.mode {
	case ModeWorkloads:
		return len(n.filteredWorkloads())
	case ModeResources:
		return n.sectionMaxItems()
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
	case ModeResources:
		b.WriteString(n.renderResources())
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
	case ModeResources:
		// No header for resources view - sections have their own headers
		return ""
	case ModeNamespace:
		icon = "◉"
		title = "SELECT NAMESPACE"
	case ModeResourceType:
		icon = "◆"
		title = "SELECT RESOURCE TYPE"
	}

	iconStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Text).Bold(true)

	// Only show active indicator if panel is active
	if n.mode == ModeNamespace && !n.panelActive {
		return "  " + titleStyle.Render(title)
	}

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

func (n Navigator) renderResources() string {
	var b strings.Builder

	// Calculate height for each section
	totalHeight := n.height - 8 // Reserve space for headers
	podsHeight := totalHeight * 40 / 100      // 40%
	cmHeight := totalHeight * 20 / 100        // 20%
	secretsHeight := totalHeight * 20 / 100   // 20%
	dockerHeight := totalHeight * 20 / 100    // 20%

	// PODS Section
	sectionActive := n.section == SectionPods
	b.WriteString(n.renderSectionHeader("PODS", len(n.pods), sectionActive))
	b.WriteString("\n")
	b.WriteString(n.renderPodsTable(podsHeight, sectionActive))
	b.WriteString("\n\n")

	// CONFIGMAPS Section
	sectionActive = n.section == SectionConfigMaps
	b.WriteString(n.renderSectionHeader("ConfigMaps", len(n.configmaps), sectionActive))
	b.WriteString("\n")
	b.WriteString(n.renderConfigMapsTable(cmHeight, sectionActive))
	b.WriteString("\n\n")

	// SECRETS Section (excluding docker registry)
	filteredSecrets := n.filteredSecrets()
	sectionActive = n.section == SectionSecrets
	b.WriteString(n.renderSectionHeader("Secrets", len(filteredSecrets), sectionActive))
	b.WriteString("\n")
	b.WriteString(n.renderFilteredSecretsTable(secretsHeight, sectionActive, filteredSecrets))
	b.WriteString("\n\n")

	// DOCKER REGISTRY Section
	dockerSecrets := n.dockerRegistrySecrets()
	sectionActive = n.section == SectionDockerRegistry
	b.WriteString(n.renderSectionHeader("Docker Registry", len(dockerSecrets), sectionActive))
	b.WriteString("\n")
	b.WriteString(n.renderDockerRegistryTable(dockerHeight, sectionActive, dockerSecrets))

	return b.String()
}

func (n Navigator) renderSectionHeader(title string, count int, active bool) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	titleWithCount := titleStyle.Render(fmt.Sprintf("%s (%d)", title, count))
	if active {
		indicator := styles.StatusRunning.Render("●")
		return indicator + " " + titleWithCount
	}
	return "  " + titleWithCount
}

func (n Navigator) renderPodsTable(maxRows int, active bool) string {
	pods := n.filteredPods()
	if len(pods) == 0 {
		return styles.StatusMuted.Render("  No pods found")
	}

	var b strings.Builder
	header := fmt.Sprintf("  %-38s %-8s %-10s %-8s %-6s", "NAME", "READY", "STATUS", "RESTARTS", "AGE")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	cursor := n.sectionCursors[SectionPods]
	visibleRows := maxRows - 1 // Reserve for header

	// Calculate visible window
	startIdx, endIdx := n.calculateVisibleWindow(cursor, len(pods), visibleRows)

	// Show "more above" indicator
	if startIdx > 0 {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... %d more above", startIdx)))
		b.WriteString("\n")
		visibleRows--
		endIdx = startIdx + visibleRows
		if endIdx > len(pods) {
			endIdx = len(pods)
		}
	}

	for i := startIdx; i < endIdx; i++ {
		selected := active && i == cursor
		b.WriteString(n.renderPodRow(pods[i], selected))
		b.WriteString("\n")
	}

	// Show "more below" indicator
	if endIdx < len(pods) {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... and %d more", len(pods)-endIdx)))
	}

	return b.String()
}

func (n Navigator) renderConfigMapsTable(maxRows int, active bool) string {
	if len(n.configmaps) == 0 {
		return styles.StatusMuted.Render("  No configmaps found")
	}

	var b strings.Builder
	header := fmt.Sprintf("  %-40s %-8s %-6s", "NAME", "KEYS", "AGE")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	cursor := n.sectionCursors[SectionConfigMaps]
	visibleRows := maxRows - 1

	startIdx, endIdx := n.calculateVisibleWindow(cursor, len(n.configmaps), visibleRows)

	if startIdx > 0 {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... %d more above", startIdx)))
		b.WriteString("\n")
		visibleRows--
		endIdx = startIdx + visibleRows
		if endIdx > len(n.configmaps) {
			endIdx = len(n.configmaps)
		}
	}

	for i := startIdx; i < endIdx; i++ {
		selected := active && i == cursor
		b.WriteString(n.renderConfigMapRow(n.configmaps[i], selected))
		b.WriteString("\n")
	}

	if endIdx < len(n.configmaps) {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... and %d more", len(n.configmaps)-endIdx)))
	}

	return b.String()
}

func (n Navigator) renderSecretsTable(maxRows int, active bool) string {
	return n.renderFilteredSecretsTable(maxRows, active, n.secrets)
}

func (n Navigator) renderFilteredSecretsTable(maxRows int, active bool, secrets []k8s.SecretInfo) string {
	if len(secrets) == 0 {
		return styles.StatusMuted.Render("  No secrets found")
	}

	var b strings.Builder
	header := fmt.Sprintf("  %-40s %-30s %-8s %-6s", "NAME", "TYPE", "KEYS", "AGE")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	cursor := n.sectionCursors[SectionSecrets]
	visibleRows := maxRows - 1

	startIdx, endIdx := n.calculateVisibleWindow(cursor, len(secrets), visibleRows)

	if startIdx > 0 {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... %d more above", startIdx)))
		b.WriteString("\n")
		visibleRows--
		endIdx = startIdx + visibleRows
		if endIdx > len(secrets) {
			endIdx = len(secrets)
		}
	}

	for i := startIdx; i < endIdx; i++ {
		selected := active && i == cursor
		b.WriteString(n.renderSecretRow(secrets[i], selected))
		b.WriteString("\n")
	}

	if endIdx < len(secrets) {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... and %d more", len(secrets)-endIdx)))
	}

	return b.String()
}

func (n Navigator) renderDockerRegistryTable(maxRows int, active bool, secrets []k8s.SecretInfo) string {
	if len(secrets) == 0 {
		return styles.StatusMuted.Render("  No docker registry secrets found")
	}

	var b strings.Builder
	header := fmt.Sprintf("  %-50s %-6s", "NAME", "AGE")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	cursor := n.sectionCursors[SectionDockerRegistry]
	visibleRows := maxRows - 1

	startIdx, endIdx := n.calculateVisibleWindow(cursor, len(secrets), visibleRows)

	if startIdx > 0 {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... %d more above", startIdx)))
		b.WriteString("\n")
		visibleRows--
		endIdx = startIdx + visibleRows
		if endIdx > len(secrets) {
			endIdx = len(secrets)
		}
	}

	for i := startIdx; i < endIdx; i++ {
		selected := active && i == cursor
		b.WriteString(n.renderDockerRegistryRow(secrets[i], selected))
		b.WriteString("\n")
	}

	if endIdx < len(secrets) {
		b.WriteString(styles.StatusMuted.Render(fmt.Sprintf("  ... and %d more", len(secrets)-endIdx)))
	}

	return b.String()
}

func (n Navigator) renderDockerRegistryRow(s k8s.SecretInfo, selected bool) string {
	cursorStr := "  "
	if selected {
		cursorStr = styles.CursorStyle.Render("> ")
	}

	name := styles.Truncate(s.Name, 50)

	if selected {
		rowStyle := lipgloss.NewStyle().Background(styles.Surface)
		return rowStyle.Render(fmt.Sprintf("%s%-50s %-6s", cursorStr, name, s.Age))
	}
	return fmt.Sprintf("%s%-50s %-6s", cursorStr, name, s.Age)
}

func (n Navigator) renderConfigMapRow(cm k8s.ConfigMapInfo, selected bool) string {
	cursorStr := "  "
	if selected {
		cursorStr = styles.CursorStyle.Render("> ")
	}

	name := styles.Truncate(cm.Name, 40)

	if selected {
		rowStyle := lipgloss.NewStyle().Background(styles.Surface)
		return rowStyle.Render(fmt.Sprintf("%s%-40s %-8d %-6s", cursorStr, name, cm.Keys, cm.Age))
	}
	return fmt.Sprintf("%s%-40s %-8d %-6s", cursorStr, name, cm.Keys, cm.Age)
}

func (n Navigator) renderSecretRow(s k8s.SecretInfo, selected bool) string {
	cursorStr := "  "
	if selected {
		cursorStr = styles.CursorStyle.Render("> ")
	}

	name := styles.Truncate(s.Name, 40)
	secretType := styles.Truncate(s.Type, 30)

	if selected {
		rowStyle := lipgloss.NewStyle().Background(styles.Surface)
		return rowStyle.Render(fmt.Sprintf("%s%-40s %-30s %-8d %-6s", cursorStr, name, secretType, s.Keys, s.Age))
	}
	return fmt.Sprintf("%s%-40s %-30s %-8d %-6s", cursorStr, name, secretType, s.Keys, s.Age)
}

func (n Navigator) renderPodRow(p k8s.PodInfo, selected bool) string {
	cursor := "  "
	if selected {
		cursor = styles.CursorStyle.Render("> ")
	}

	name := styles.Truncate(p.Name, 38)
	statusStyle := styles.GetStatusStyle(p.Status)

	// Pad values before styling to maintain alignment
	statusPadded := fmt.Sprintf("%-10s", p.Status)
	restartsPadded := fmt.Sprintf("%-8d", p.Restarts)

	styledStatus := statusStyle.Render(statusPadded)
	styledRestarts := restartsPadded
	if p.Restarts > 0 {
		styledRestarts = styles.StatusError.Render(restartsPadded)
	}

	if selected {
		rowStyle := lipgloss.NewStyle().Background(styles.Surface)
		return rowStyle.Render(fmt.Sprintf("%s%-38s %-8s %s %s %-6s",
			cursor, name, p.Ready, styledStatus, styledRestarts, p.Age))
	}

	return fmt.Sprintf("%s%-38s %-8s %s %s %-6s",
		cursor, name, p.Ready, styledStatus, styledRestarts, p.Age)
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

// calculateVisibleWindow calculates start and end indices to keep cursor visible in a scrollable list
func (n Navigator) calculateVisibleWindow(cursor, total, visibleRows int) (startIdx, endIdx int) {
	if total <= visibleRows {
		return 0, total
	}

	// Keep cursor in the middle when possible
	halfVisible := visibleRows / 2
	startIdx = cursor - halfVisible
	if startIdx < 0 {
		startIdx = 0
	}

	endIdx = startIdx + visibleRows
	if endIdx > total {
		endIdx = total
		startIdx = endIdx - visibleRows
		if startIdx < 0 {
			startIdx = 0
		}
	}

	return startIdx, endIdx
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
	if n.sectionCursors[SectionPods] >= len(pods) {
		n.sectionCursors[SectionPods] = len(pods) - 1
	}
	if n.sectionCursors[SectionPods] < 0 {
		n.sectionCursors[SectionPods] = 0
	}
}

func (n *Navigator) SetConfigMaps(cms []k8s.ConfigMapInfo) {
	n.configmaps = cms
	if n.sectionCursors[SectionConfigMaps] >= len(cms) {
		n.sectionCursors[SectionConfigMaps] = len(cms) - 1
	}
	if n.sectionCursors[SectionConfigMaps] < 0 {
		n.sectionCursors[SectionConfigMaps] = 0
	}
}

func (n *Navigator) SetSecrets(secrets []k8s.SecretInfo) {
	n.secrets = secrets

	// Handle regular secrets cursor
	filteredCount := len(n.filteredSecrets())
	if n.sectionCursors[SectionSecrets] >= filteredCount {
		n.sectionCursors[SectionSecrets] = filteredCount - 1
	}
	if n.sectionCursors[SectionSecrets] < 0 {
		n.sectionCursors[SectionSecrets] = 0
	}

	// Handle docker registry secrets cursor
	dockerCount := len(n.dockerRegistrySecrets())
	if n.sectionCursors[SectionDockerRegistry] >= dockerCount {
		n.sectionCursors[SectionDockerRegistry] = dockerCount - 1
	}
	if n.sectionCursors[SectionDockerRegistry] < 0 {
		n.sectionCursors[SectionDockerRegistry] = 0
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

func (n *Navigator) SetPanelActive(active bool) {
	n.panelActive = active
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
	cursor := n.sectionCursors[SectionPods]
	if cursor >= 0 && cursor < len(pods) {
		return &pods[cursor]
	}
	return nil
}

func (n Navigator) SelectedConfigMap() *k8s.ConfigMapInfo {
	cursor := n.sectionCursors[SectionConfigMaps]
	if cursor >= 0 && cursor < len(n.configmaps) {
		return &n.configmaps[cursor]
	}
	return nil
}

func (n Navigator) SelectedSecret() *k8s.SecretInfo {
	secrets := n.filteredSecrets()
	cursor := n.sectionCursors[SectionSecrets]
	if cursor >= 0 && cursor < len(secrets) {
		return &secrets[cursor]
	}
	return nil
}

func (n Navigator) SelectedDockerRegistrySecret() *k8s.SecretInfo {
	secrets := n.dockerRegistrySecrets()
	cursor := n.sectionCursors[SectionDockerRegistry]
	if cursor >= 0 && cursor < len(secrets) {
		return &secrets[cursor]
	}
	return nil
}

func (n Navigator) Section() PodViewSection {
	return n.section
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

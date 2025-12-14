package component

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// ConfigMapViewerMode represents the current mode of the viewer
type ConfigMapViewerMode int

const (
	ConfigMapViewerModeNormal    ConfigMapViewerMode = iota // Normal key/value viewing
	ConfigMapViewerModeAction                               // Action menu
	ConfigMapViewerModeNamespace                            // Namespace selector
)

// ConfigMapViewer displays ConfigMap data in a modal with key selection
type ConfigMapViewer struct {
	configmap  *repository.ConfigMapData
	namespace  string
	visible    bool
	scroll     int
	width      int
	height     int
	lines      []string    // Pre-rendered lines for scrolling
	sortedKeys []string    // Sorted keys for selection
	keyCursor  int         // Currently selected key index
	keyLineMap map[int]int // Maps key index to first line index
	copied     bool        // Show "copied" feedback

	// Action menu and namespace selector
	mode           ConfigMapViewerMode
	actionCursor   int      // Action menu cursor
	namespaces     []string // Available namespaces
	nsCursor       int      // Namespace selector cursor
	nsScroll       int      // Namespace scroll offset
	nsSearchQuery  string   // Namespace filter
	statusMsg      string   // Status message (success/error)
	pendingRequest *ConfigMapCopyRequest // Pending copy request
}

// ConfigMapViewerClosed is sent when the viewer is closed
type ConfigMapViewerClosed struct{}

// ConfigMapValueCopied is sent when a value is copied to clipboard
type ConfigMapValueCopied struct {
	Key string
}

// ConfigMapCopyRequest is sent when user wants to copy configmap to namespace(s)
type ConfigMapCopyRequest struct {
	ConfigMapName   string
	SourceNamespace string
	TargetNamespace string   // Single namespace or empty for all
	AllNamespaces   bool     // If true, copy to all namespaces
	Namespaces      []string // List of all namespaces (when AllNamespaces is true)
}

// ConfigMapCopyResult is sent when configmap copy operation completes
type ConfigMapCopyResult struct {
	Success bool
	Message string
	Err     error
}

// ConfigMapCopyProgress is sent during multi-namespace copy to show progress
type ConfigMapCopyProgress struct {
	ConfigMapName    string
	SourceNamespace  string
	CurrentNamespace string   // Namespace being copied to
	Remaining        []string // Remaining namespaces to copy
	SuccessCount     int
	ErrorCount       int
}

func NewConfigMapViewer() ConfigMapViewer {
	return ConfigMapViewer{
		keyLineMap: make(map[int]int),
	}
}

func (v ConfigMapViewer) Init() tea.Cmd {
	return nil
}

func (v ConfigMapViewer) Update(msg tea.Msg) (ConfigMapViewer, tea.Cmd) {
	if !v.visible {
		return v, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle different modes
		switch v.mode {
		case ConfigMapViewerModeAction:
			return v.updateActionMenu(msg)
		case ConfigMapViewerModeNamespace:
			return v.updateNamespaceSelector(msg)
		default:
			return v.updateNormal(msg)
		}
	}

	return v, nil
}

func (v ConfigMapViewer) updateNormal(msg tea.KeyMsg) (ConfigMapViewer, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		v.visible = false
		v.copied = false
		return v, func() tea.Msg { return ConfigMapViewerClosed{} }
	case "a":
		// Open action menu
		v.mode = ConfigMapViewerModeAction
		v.actionCursor = 0
		return v, nil
	case "up", "k":
		v.copied = false
		if v.keyCursor > 0 {
			v.keyCursor--
			v.scrollToKey()
		}
	case "down", "j":
		v.copied = false
		if v.keyCursor < len(v.sortedKeys)-1 {
			v.keyCursor++
			v.scrollToKey()
		}
	case "enter":
		// Copy selected key's value to clipboard
		if v.keyCursor >= 0 && v.keyCursor < len(v.sortedKeys) && v.configmap != nil {
			key := v.sortedKeys[v.keyCursor]
			value := v.configmap.Data[key]
			if err := copyToClipboard(value); err == nil {
				v.copied = true
				return v, func() tea.Msg { return ConfigMapValueCopied{Key: key} }
			}
		}
	case "pgup", "ctrl+u":
		v.copied = false
		v.keyCursor -= 5
		if v.keyCursor < 0 {
			v.keyCursor = 0
		}
		v.scrollToKey()
	case "pgdown", "ctrl+d":
		v.copied = false
		v.keyCursor += 5
		if v.keyCursor >= len(v.sortedKeys) {
			v.keyCursor = len(v.sortedKeys) - 1
		}
		if v.keyCursor < 0 {
			v.keyCursor = 0
		}
		v.scrollToKey()
	case "g", "home":
		v.copied = false
		v.keyCursor = 0
		v.scrollToKey()
	case "G", "end":
		v.copied = false
		v.keyCursor = len(v.sortedKeys) - 1
		if v.keyCursor < 0 {
			v.keyCursor = 0
		}
		v.scrollToKey()
	}
	return v, nil
}

func (v ConfigMapViewer) updateActionMenu(msg tea.KeyMsg) (ConfigMapViewer, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		v.mode = ConfigMapViewerModeNormal
		return v, nil
	case "up", "k":
		if v.actionCursor > 0 {
			v.actionCursor--
		}
	case "down", "j":
		if v.actionCursor < 1 { // 2 options: 0 and 1
			v.actionCursor++
		}
	case "enter", "1", "2":
		idx := v.actionCursor
		if msg.String() == "1" {
			idx = 0
		} else if msg.String() == "2" {
			idx = 1
		}

		if idx == 0 {
			// Copy to specific namespace - show namespace selector
			v.mode = ConfigMapViewerModeNamespace
			v.nsCursor = 0
			v.nsScroll = 0
			v.nsSearchQuery = ""
			return v, nil
		} else if idx == 1 {
			// Copy to all namespaces - return the request directly
			v.mode = ConfigMapViewerModeNormal
			req := ConfigMapCopyRequest{
				ConfigMapName:   "",
				SourceNamespace: v.namespace,
				AllNamespaces:   true,
				Namespaces:      make([]string, len(v.namespaces)),
			}
			if v.configmap != nil {
				req.ConfigMapName = v.configmap.Name
			}
			copy(req.Namespaces, v.namespaces)
			v.pendingRequest = &req
			return v, nil
		}
	}
	return v, nil
}

func (v ConfigMapViewer) updateNamespaceSelector(msg tea.KeyMsg) (ConfigMapViewer, tea.Cmd) {
	filtered := v.filteredNamespaces()

	switch msg.String() {
	case "esc":
		// Go back to action menu
		v.mode = ConfigMapViewerModeAction
		v.nsSearchQuery = ""
		return v, nil
	case "up", "k":
		if v.nsCursor > 0 {
			v.nsCursor--
			v.adjustNsScroll(filtered)
		}
	case "down", "j":
		if v.nsCursor < len(filtered)-1 {
			v.nsCursor++
			v.adjustNsScroll(filtered)
		}
	case "enter":
		if v.nsCursor >= 0 && v.nsCursor < len(filtered) && v.configmap != nil {
			targetNs := filtered[v.nsCursor]
			// Don't copy to same namespace
			if targetNs == v.namespace {
				return v, nil
			}
			v.mode = ConfigMapViewerModeNormal
			v.nsSearchQuery = ""
			req := ConfigMapCopyRequest{
				ConfigMapName:   v.configmap.Name,
				SourceNamespace: v.namespace,
				TargetNamespace: targetNs,
				AllNamespaces:   false,
			}
			v.pendingRequest = &req
			return v, nil
		}
	case "backspace":
		if len(v.nsSearchQuery) > 0 {
			v.nsSearchQuery = v.nsSearchQuery[:len(v.nsSearchQuery)-1]
			v.nsCursor = 0
			v.nsScroll = 0
		}
	default:
		// Type to filter
		k := msg.String()
		if len(k) == 1 && k >= " " && k <= "~" {
			v.nsSearchQuery += k
			v.nsCursor = 0
			v.nsScroll = 0
		}
	}
	return v, nil
}

func (v ConfigMapViewer) filteredNamespaces() []string {
	if v.nsSearchQuery == "" {
		return v.namespaces
	}
	var filtered []string
	query := strings.ToLower(v.nsSearchQuery)
	for _, ns := range v.namespaces {
		if strings.Contains(strings.ToLower(ns), query) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

func (v *ConfigMapViewer) adjustNsScroll(filtered []string) {
	maxVisible := 15
	if v.nsCursor < v.nsScroll {
		v.nsScroll = v.nsCursor
	} else if v.nsCursor >= v.nsScroll+maxVisible {
		v.nsScroll = v.nsCursor - maxVisible + 1
	}
}

func (v *ConfigMapViewer) scrollToKey() {
	if lineIdx, ok := v.keyLineMap[v.keyCursor]; ok {
		maxLines := v.maxVisibleLines()
		// Scroll to make the selected key visible
		if lineIdx < v.scroll {
			v.scroll = lineIdx
		} else if lineIdx >= v.scroll+maxLines {
			v.scroll = lineIdx - maxLines + 1
		}
	}
}

func (v ConfigMapViewer) maxVisibleLines() int {
	maxLines := v.height - 10
	if maxLines < 5 {
		maxLines = 5
	}
	return maxLines
}

func (v *ConfigMapViewer) buildLines() {
	v.lines = []string{}
	v.sortedKeys = []string{}
	v.keyLineMap = make(map[int]int)

	if v.configmap == nil || len(v.configmap.Data) == 0 {
		v.lines = append(v.lines, style.StatusMuted.Render("No data in this ConfigMap"))
		return
	}

	// Sort keys
	for k := range v.configmap.Data {
		v.sortedKeys = append(v.sortedKeys, k)
	}
	sort.Strings(v.sortedKeys)

	maxValueWidth := v.width - 16
	if maxValueWidth < 40 {
		maxValueWidth = 40
	}

	for i, key := range v.sortedKeys {
		// Record the line index where this key starts
		v.keyLineMap[i] = len(v.lines)

		// Key header (will be highlighted based on selection in View)
		v.lines = append(v.lines, key)

		// Value with word wrapping
		value := v.configmap.Data[key]
		if value == "" {
			v.lines = append(v.lines, "  (empty)")
		} else {
			// Split by newlines first
			valueLines := strings.Split(value, "\n")
			for _, line := range valueLines {
				// Wrap long lines
				wrapped := v.wrapText(line, maxValueWidth)
				for _, wl := range wrapped {
					v.lines = append(v.lines, "  "+wl)
				}
			}
		}

		// Add spacing between keys (except last)
		if i < len(v.sortedKeys)-1 {
			v.lines = append(v.lines, "")
		}
	}
}

func (v ConfigMapViewer) wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	for len(text) > maxWidth {
		breakPoint := maxWidth
		for i := maxWidth; i > maxWidth/2; i-- {
			if i < len(text) && (text[i] == ' ' || text[i] == ',' || text[i] == ';' || text[i] == ':') {
				breakPoint = i + 1
				break
			}
		}
		if breakPoint > len(text) {
			breakPoint = len(text)
		}
		lines = append(lines, text[:breakPoint])
		text = text[breakPoint:]
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return lines
}

func (v ConfigMapViewer) View() string {
	if !v.visible || v.configmap == nil {
		return ""
	}

	// Handle different modes
	switch v.mode {
	case ConfigMapViewerModeAction:
		return v.renderActionMenu()
	case ConfigMapViewerModeNamespace:
		return v.renderNamespaceSelector()
	}

	var header strings.Builder
	var content strings.Builder

	// Breadcrumb with info - outside the box, same line
	separatorStyle := lipgloss.NewStyle().Foreground(style.TextMuted)
	itemStyle := lipgloss.NewStyle().Foreground(style.Primary)
	infoStyle := lipgloss.NewStyle().Foreground(style.Secondary)

	breadcrumb := itemStyle.Render(v.namespace) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("configmaps") +
		separatorStyle.Render(" > ") +
		itemStyle.Render(v.configmap.Name) +
		separatorStyle.Render(" - ") +
		infoStyle.Render(fmt.Sprintf("[%s] [%d keys]", v.configmap.Age, len(v.configmap.Data)))
	header.WriteString(breadcrumb)
	header.WriteString("\n")

	// Styles for rendering
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Primary)
	selectedKeyStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Text).Background(style.Primary)
	valueStyle := lipgloss.NewStyle().Foreground(style.Text)

	// Render visible lines inside box
	maxLines := v.maxVisibleLines()
	endIdx := v.scroll + maxLines
	if endIdx > len(v.lines) {
		endIdx = len(v.lines)
	}

	// Find which key each line belongs to
	lineToKey := make(map[int]int)
	for keyIdx, lineIdx := range v.keyLineMap {
		// Mark all lines belonging to this key
		nextKeyLine := len(v.lines)
		for nextIdx, nextLine := range v.keyLineMap {
			if nextIdx > keyIdx && nextLine < nextKeyLine {
				nextKeyLine = nextLine
			}
		}
		for l := lineIdx; l < nextKeyLine; l++ {
			lineToKey[l] = keyIdx
		}
	}

	for i := v.scroll; i < endIdx; i++ {
		line := v.lines[i]
		keyIdx := lineToKey[i]
		isSelected := keyIdx == v.keyCursor

		// Check if this is the key header line
		isKeyHeader := false
		if keyLine, ok := v.keyLineMap[keyIdx]; ok && keyLine == i {
			isKeyHeader = true
		}

		if isKeyHeader {
			if isSelected {
				content.WriteString(selectedKeyStyle.Render("> " + line))
			} else {
				content.WriteString("  " + keyStyle.Render(line))
			}
		} else if strings.HasPrefix(line, "  ") {
			if isSelected {
				content.WriteString(valueStyle.Render(line))
			} else {
				content.WriteString(valueStyle.Render(line))
			}
		} else {
			content.WriteString(line)
		}
		content.WriteString("\n")
	}

	// Fill remaining space
	renderedLines := endIdx - v.scroll
	for i := renderedLines; i < maxLines; i++ {
		content.WriteString("\n")
	}

	// Box style matching Logs panel
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Surface).
		Padding(0, 1).
		Width(v.width - 10).
		Height(v.height - 10)

	boxedContent := boxStyle.Render(content.String())

	// Footer with help and copied indicator
	var footer string
	copiedIndicator := ""
	if v.copied {
		copiedIndicator = style.StatusRunning.Render(" [Copied!]")
	}

	// Add status message if present
	statusIndicator := ""
	if v.statusMsg != "" {
		statusIndicator = " " + style.StatusRunning.Render(v.statusMsg)
	}

	if len(v.sortedKeys) > 0 {
		keyInfo := fmt.Sprintf("[%d/%d]", v.keyCursor+1, len(v.sortedKeys))
		footer = style.StatusMuted.Render(fmt.Sprintf("%s ↑↓:select  Enter:copy  a:actions  Esc:close", keyInfo)) + copiedIndicator + statusIndicator
	} else {
		footer = style.StatusMuted.Render("a:actions  Esc:close") + statusIndicator
	}

	return header.String() + boxedContent + "\n" + footer
}

func (v ConfigMapViewer) renderActionMenu() string {
	var b strings.Builder

	// Header
	separatorStyle := lipgloss.NewStyle().Foreground(style.TextMuted)
	itemStyle := lipgloss.NewStyle().Foreground(style.Primary)

	breadcrumb := itemStyle.Render(v.namespace) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("configmaps") +
		separatorStyle.Render(" > ") +
		itemStyle.Render(v.configmap.Name) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("Actions")
	b.WriteString(breadcrumb)
	b.WriteString("\n\n")

	// Action menu items
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Text).Background(style.Primary)
	normalStyle := lipgloss.NewStyle().Foreground(style.Text)

	actions := []string{
		"[1] Copy to namespace...",
		"[2] Copy to all namespaces",
	}

	for i, action := range actions {
		if i == v.actionCursor {
			b.WriteString(selectedStyle.Render("> " + action))
		} else {
			b.WriteString(normalStyle.Render("  " + action))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(style.StatusMuted.Render("↑↓:select  Enter/1/2:choose  Esc:back"))

	return b.String()
}

func (v ConfigMapViewer) renderNamespaceSelector() string {
	var b strings.Builder

	// Header
	separatorStyle := lipgloss.NewStyle().Foreground(style.TextMuted)
	itemStyle := lipgloss.NewStyle().Foreground(style.Primary)

	breadcrumb := itemStyle.Render(v.namespace) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("configmaps") +
		separatorStyle.Render(" > ") +
		itemStyle.Render(v.configmap.Name) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("Select Namespace")
	b.WriteString(breadcrumb)
	b.WriteString("\n\n")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Text).Background(style.Primary)
	normalStyle := lipgloss.NewStyle().Foreground(style.Text)
	currentNsStyle := lipgloss.NewStyle().Foreground(style.Secondary)

	// Search bar
	if v.nsSearchQuery != "" {
		filterStyle := lipgloss.NewStyle().Foreground(style.Secondary)
		b.WriteString(filterStyle.Render("Filter: " + v.nsSearchQuery))
	} else {
		b.WriteString(style.StatusMuted.Render("Type to filter..."))
	}
	b.WriteString("\n\n")

	filtered := v.filteredNamespaces()

	if len(filtered) == 0 {
		b.WriteString(style.StatusMuted.Render("No namespaces match filter"))
		b.WriteString("\n")
	} else {
		maxVisible := 15
		startIdx := v.nsScroll
		endIdx := startIdx + maxVisible
		if endIdx > len(filtered) {
			endIdx = len(filtered)
		}

		for i := startIdx; i < endIdx; i++ {
			ns := filtered[i]
			prefix := "  "
			suffix := ""

			// Mark current namespace
			if ns == v.namespace {
				suffix = currentNsStyle.Render(" (current)")
			}

			if i == v.nsCursor {
				b.WriteString(selectedStyle.Render("> " + ns))
				b.WriteString(suffix)
			} else {
				b.WriteString(prefix + normalStyle.Render(ns) + suffix)
			}
			b.WriteString("\n")
		}

		// Show scroll indicator
		if len(filtered) > maxVisible {
			b.WriteString(style.StatusMuted.Render(fmt.Sprintf("\n[%d/%d]", v.nsCursor+1, len(filtered))))
		}
	}

	b.WriteString("\n")
	b.WriteString(style.StatusMuted.Render("↑↓:select  Enter:copy  Esc:back"))

	return b.String()
}

func (v *ConfigMapViewer) Show(cm *repository.ConfigMapData, namespace string) {
	v.configmap = cm
	v.namespace = namespace
	v.scroll = 0
	v.keyCursor = 0
	v.copied = false
	v.mode = ConfigMapViewerModeNormal
	v.statusMsg = ""
	v.buildLines()
	v.visible = true
}

func (v *ConfigMapViewer) Hide() {
	v.visible = false
	v.copied = false
	v.mode = ConfigMapViewerModeNormal
	v.statusMsg = ""
}

func (v ConfigMapViewer) IsVisible() bool {
	return v.visible
}

func (v *ConfigMapViewer) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.configmap != nil {
		v.buildLines()
	}
}

func (v *ConfigMapViewer) SetNamespaces(namespaces []string) {
	v.namespaces = namespaces
}

// GetPendingRequest returns any pending copy request and clears it
func (v *ConfigMapViewer) GetPendingRequest() *ConfigMapCopyRequest {
	req := v.pendingRequest
	v.pendingRequest = nil
	return req
}

// SetStatusMsg sets the status message shown in the footer
func (v *ConfigMapViewer) SetStatusMsg(msg string) {
	v.statusMsg = msg
}

// copyToClipboard copies text to system clipboard
func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := pipe.Write([]byte(text)); err != nil {
		return err
	}
	if err := pipe.Close(); err != nil {
		return err
	}
	return cmd.Wait()
}

package component

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// SecretViewerMode represents the current mode of the viewer
type SecretViewerMode int

const (
	SecretViewerModeNormal    SecretViewerMode = iota // Normal key/value viewing
	SecretViewerModeAction                            // Action menu
	SecretViewerModeNamespace                         // Namespace selector
)

// SecretViewer displays Secret data in a modal with decoded values and key selection
type SecretViewer struct {
	secret      *repository.SecretData
	namespace   string
	visible     bool
	scroll      int
	width       int
	height      int
	lines       []string    // Pre-rendered lines for scrolling
	sortedKeys  []string    // Sorted keys for selection
	keyCursor   int         // Currently selected key index
	keyLineMap  map[int]int // Maps key index to first line index
	copied      bool        // Show "copied" feedback

	// Action menu and namespace selector
	mode           SecretViewerMode
	actionCursor   int      // Action menu cursor
	namespaces     []string // Available namespaces
	nsCursor       int      // Namespace selector cursor
	nsScroll       int      // Namespace scroll offset
	nsSearchQuery  string   // Namespace filter
	statusMsg      string   // Status message (success/error)
	pendingRequest *SecretCopyRequest // Pending copy request
}

// SecretViewerClosed is sent when the viewer is closed
type SecretViewerClosed struct{}

// SecretValueCopied is sent when a value is copied to clipboard
type SecretValueCopied struct {
	Key string
}

// SecretCopyRequest is sent when user wants to copy secret to namespace(s)
type SecretCopyRequest struct {
	SecretName      string
	SourceNamespace string
	TargetNamespace string   // Single namespace or empty for all
	AllNamespaces   bool     // If true, copy to all namespaces
	Namespaces      []string // List of all namespaces (when AllNamespaces is true)
}

// SecretCopyResult is sent when secret copy operation completes
type SecretCopyResult struct {
	Success   bool
	Message   string
	Err       error
}

// SecretCopyProgress is sent during multi-namespace copy to show progress
type SecretCopyProgress struct {
	SecretName       string
	SourceNamespace  string
	CurrentNamespace string   // Namespace being copied to
	Remaining        []string // Remaining namespaces to copy
	SuccessCount     int
	ErrorCount       int
}

func NewSecretViewer() SecretViewer {
	return SecretViewer{
		keyLineMap: make(map[int]int),
	}
}

func (v SecretViewer) Init() tea.Cmd {
	return nil
}

func (v SecretViewer) Update(msg tea.Msg) (SecretViewer, tea.Cmd) {
	if !v.visible {
		return v, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle different modes
		switch v.mode {
		case SecretViewerModeAction:
			return v.updateActionMenu(msg)
		case SecretViewerModeNamespace:
			return v.updateNamespaceSelector(msg)
		default:
			return v.updateNormal(msg)
		}
	}

	return v, nil
}

func (v SecretViewer) updateNormal(msg tea.KeyMsg) (SecretViewer, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		v.visible = false
		v.copied = false
		return v, func() tea.Msg { return SecretViewerClosed{} }
	case "a":
		// Open action menu
		v.mode = SecretViewerModeAction
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
		if v.keyCursor >= 0 && v.keyCursor < len(v.sortedKeys) && v.secret != nil {
			key := v.sortedKeys[v.keyCursor]
			value := v.secret.Data[key]
			if err := copyToClipboard(value); err == nil {
				v.copied = true
				return v, func() tea.Msg { return SecretValueCopied{Key: key} }
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

func (v SecretViewer) updateActionMenu(msg tea.KeyMsg) (SecretViewer, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		v.mode = SecretViewerModeNormal
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
			v.mode = SecretViewerModeNamespace
			v.nsCursor = 0
			v.nsScroll = 0
			v.nsSearchQuery = ""
			return v, nil
		} else if idx == 1 {
			// Copy to all namespaces - return the request directly as a message
			v.mode = SecretViewerModeNormal
			req := SecretCopyRequest{
				SecretName:      "",
				SourceNamespace: v.namespace,
				AllNamespaces:   true,
				Namespaces:      make([]string, len(v.namespaces)),
			}
			if v.secret != nil {
				req.SecretName = v.secret.Name
			}
			copy(req.Namespaces, v.namespaces)
			// Return the request - will be handled in app.go
			v.pendingRequest = &req
			return v, nil
		}
	}
	return v, nil
}

func (v SecretViewer) updateNamespaceSelector(msg tea.KeyMsg) (SecretViewer, tea.Cmd) {
	filtered := v.filteredNamespaces()

	switch msg.String() {
	case "esc":
		// Go back to action menu
		v.mode = SecretViewerModeAction
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
		if v.nsCursor >= 0 && v.nsCursor < len(filtered) && v.secret != nil {
			targetNs := filtered[v.nsCursor]
			// Don't copy to same namespace
			if targetNs == v.namespace {
				return v, nil
			}
			v.mode = SecretViewerModeNormal
			v.nsSearchQuery = ""
			return v, func() tea.Msg {
				return SecretCopyRequest{
					SecretName:      v.secret.Name,
					SourceNamespace: v.namespace,
					TargetNamespace: targetNs,
					AllNamespaces:   false,
				}
			}
		}
	case "backspace":
		if len(v.nsSearchQuery) > 0 {
			v.nsSearchQuery = v.nsSearchQuery[:len(v.nsSearchQuery)-1]
			v.nsCursor = 0
			v.nsScroll = 0
		}
	default:
		// Type to filter
		if len(msg.String()) == 1 {
			v.nsSearchQuery += msg.String()
			v.nsCursor = 0
			v.nsScroll = 0
		}
	}
	return v, nil
}

func (v *SecretViewer) filteredNamespaces() []string {
	if v.nsSearchQuery == "" {
		return v.namespaces
	}
	query := strings.ToLower(v.nsSearchQuery)
	var filtered []string
	for _, ns := range v.namespaces {
		if strings.Contains(strings.ToLower(ns), query) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

func (v *SecretViewer) adjustNsScroll(filtered []string) {
	maxVisible := v.height - 16
	if maxVisible < 5 {
		maxVisible = 5
	}
	if v.nsCursor < v.nsScroll {
		v.nsScroll = v.nsCursor
	} else if v.nsCursor >= v.nsScroll+maxVisible {
		v.nsScroll = v.nsCursor - maxVisible + 1
	}
}

func (v *SecretViewer) scrollToKey() {
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

func (v SecretViewer) maxVisibleLines() int {
	maxLines := v.height - 10
	if maxLines < 5 {
		maxLines = 5
	}
	return maxLines
}

func (v *SecretViewer) buildLines() {
	v.lines = []string{}
	v.sortedKeys = []string{}
	v.keyLineMap = make(map[int]int)

	if v.secret == nil || len(v.secret.Data) == 0 {
		v.lines = append(v.lines, style.StatusMuted.Render("No data in this Secret"))
		return
	}

	// Sort keys
	for k := range v.secret.Data {
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

		// Value with word wrapping (decoded from base64)
		value := v.secret.Data[key]
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

func (v SecretViewer) wrapText(text string, maxWidth int) []string {
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

func (v SecretViewer) View() string {
	if !v.visible || v.secret == nil {
		return ""
	}

	var header strings.Builder
	var content strings.Builder

	// Breadcrumb with info - outside the box, same line
	separatorStyle := lipgloss.NewStyle().Foreground(style.TextMuted)
	itemStyle := lipgloss.NewStyle().Foreground(style.Primary)
	infoStyle := lipgloss.NewStyle().Foreground(style.Secondary)

	breadcrumb := itemStyle.Render(v.namespace) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("secrets") +
		separatorStyle.Render(" > ") +
		itemStyle.Render(v.secret.Name) +
		separatorStyle.Render(" - ") +
		infoStyle.Render(fmt.Sprintf("[%s] [%s] [%d keys]", v.secret.Age, v.secret.Type, len(v.secret.Data)))
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

	// Box style matching other viewers
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
		footer = style.StatusMuted.Render("a:actions  Esc:close")
	}

	result := header.String() + boxedContent + "\n" + footer

	// Render overlay for action menu
	if v.mode == SecretViewerModeAction {
		overlay := v.renderActionMenu()
		result = v.overlayContent(result, overlay)
	}

	// Render overlay for namespace selector
	if v.mode == SecretViewerModeNamespace {
		overlay := v.renderNamespaceSelector()
		result = v.overlayContent(result, overlay)
	}

	return result
}

func (v SecretViewer) renderActionMenu() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Primary)
	itemStyle := lipgloss.NewStyle().Foreground(style.Text)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Text).Background(style.Primary)
	shortcutStyle := lipgloss.NewStyle().Foreground(style.Secondary)

	b.WriteString(titleStyle.Render("Secret Actions"))
	b.WriteString("\n\n")

	actions := []string{
		"Copy to namespace...",
		"Copy to all namespaces",
	}

	for i, action := range actions {
		shortcut := fmt.Sprintf("[%d] ", i+1)
		if i == v.actionCursor {
			b.WriteString(selectedStyle.Render("> " + shortcut + action))
		} else {
			b.WriteString("  " + shortcutStyle.Render(shortcut) + itemStyle.Render(action))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(style.StatusMuted.Render("↑↓:select  Enter:confirm  Esc:cancel"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Primary).
		Padding(1, 2).
		Width(40)

	return boxStyle.Render(b.String())
}

func (v SecretViewer) renderNamespaceSelector() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Primary)
	itemStyle := lipgloss.NewStyle().Foreground(style.Text)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Text).Background(style.Primary)
	currentNsStyle := lipgloss.NewStyle().Foreground(style.TextMuted)

	b.WriteString(titleStyle.Render("Select Target Namespace"))
	b.WriteString("\n")

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
				b.WriteString(prefix + itemStyle.Render(ns) + suffix)
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

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Primary).
		Padding(1, 2).
		Width(50).
		MaxHeight(25)

	return boxStyle.Render(b.String())
}

func (v SecretViewer) overlayContent(base, overlay string) string {
	// Center the overlay on the screen
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(style.Background),
	)
}

func (v *SecretViewer) Show(secret *repository.SecretData, namespace string) {
	v.secret = secret
	v.namespace = namespace
	v.scroll = 0
	v.keyCursor = 0
	v.copied = false
	v.mode = SecretViewerModeNormal
	v.statusMsg = ""
	v.buildLines()
	v.visible = true
}

func (v *SecretViewer) Hide() {
	v.visible = false
	v.copied = false
	v.mode = SecretViewerModeNormal
	v.statusMsg = ""
}

func (v SecretViewer) IsVisible() bool {
	return v.visible
}

func (v *SecretViewer) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.secret != nil {
		v.buildLines()
	}
}

func (v *SecretViewer) SetNamespaces(namespaces []string) {
	v.namespaces = namespaces
}

func (v SecretViewer) GetSecret() *repository.SecretData {
	return v.secret
}

func (v SecretViewer) GetNamespace() string {
	return v.namespace
}

// GetPendingRequest returns any pending copy request and clears it
func (v *SecretViewer) GetPendingRequest() *SecretCopyRequest {
	req := v.pendingRequest
	v.pendingRequest = nil
	return req
}

// SetStatusMsg sets the status message shown in the footer
func (v *SecretViewer) SetStatusMsg(msg string) {
	v.statusMsg = msg
}

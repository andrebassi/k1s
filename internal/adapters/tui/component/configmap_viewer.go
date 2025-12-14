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

// ConfigMapViewer displays ConfigMap data in a modal with key selection
type ConfigMapViewer struct {
	configmap   *repository.ConfigMapData
	namespace   string
	visible     bool
	scroll      int
	width       int
	height      int
	lines       []string   // Pre-rendered lines for scrolling
	sortedKeys  []string   // Sorted keys for selection
	keyCursor   int        // Currently selected key index
	keyLineMap  map[int]int // Maps key index to first line index
	copied      bool       // Show "copied" feedback
}

// ConfigMapViewerClosed is sent when the viewer is closed
type ConfigMapViewerClosed struct{}

// ConfigMapValueCopied is sent when a value is copied to clipboard
type ConfigMapValueCopied struct {
	Key string
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
		switch msg.String() {
		case "esc", "q":
			v.visible = false
			v.copied = false
			return v, func() tea.Msg { return ConfigMapViewerClosed{} }
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
	}

	return v, nil
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

	if len(v.sortedKeys) > 0 {
		keyInfo := fmt.Sprintf("[%d/%d]", v.keyCursor+1, len(v.sortedKeys))
		footer = style.StatusMuted.Render(fmt.Sprintf("%s ↑↓:select  Enter:copy  Esc:close", keyInfo)) + copiedIndicator
	} else {
		footer = style.StatusMuted.Render("Esc:close")
	}

	return header.String() + boxedContent + "\n" + footer
}

func (v *ConfigMapViewer) Show(cm *repository.ConfigMapData, namespace string) {
	v.configmap = cm
	v.namespace = namespace
	v.scroll = 0
	v.keyCursor = 0
	v.copied = false
	v.buildLines()
	v.visible = true
}

func (v *ConfigMapViewer) Hide() {
	v.visible = false
	v.copied = false
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

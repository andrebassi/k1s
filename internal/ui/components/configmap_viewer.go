package components

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k8sdebug/internal/k8s"
	"github.com/andrebassi/k8sdebug/internal/ui/styles"
)

// ConfigMapViewer displays ConfigMap data in a modal
type ConfigMapViewer struct {
	configmap *k8s.ConfigMapData
	namespace string
	visible   bool
	scroll    int
	width     int
	height    int
	lines     []string // Pre-rendered lines for scrolling
}

// ConfigMapViewerClosed is sent when the viewer is closed
type ConfigMapViewerClosed struct{}

func NewConfigMapViewer() ConfigMapViewer {
	return ConfigMapViewer{}
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
			return v, func() tea.Msg { return ConfigMapViewerClosed{} }
		case "up", "k":
			if v.scroll > 0 {
				v.scroll--
			}
		case "down", "j":
			maxScroll := len(v.lines) - v.maxVisibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if v.scroll < maxScroll {
				v.scroll++
			}
		case "pgup", "ctrl+u":
			v.scroll -= 10
			if v.scroll < 0 {
				v.scroll = 0
			}
		case "pgdown", "ctrl+d":
			maxScroll := len(v.lines) - v.maxVisibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			v.scroll += 10
			if v.scroll > maxScroll {
				v.scroll = maxScroll
			}
		case "g", "home":
			v.scroll = 0
		case "G", "end":
			maxScroll := len(v.lines) - v.maxVisibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			v.scroll = maxScroll
		}
	}

	return v, nil
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

	if v.configmap == nil || len(v.configmap.Data) == 0 {
		v.lines = append(v.lines, styles.StatusMuted.Render("No data in this ConfigMap"))
		return
	}

	// Sort keys
	keys := make([]string, 0, len(v.configmap.Data))
	for k := range v.configmap.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	valueStyle := lipgloss.NewStyle().Foreground(styles.Text)

	maxValueWidth := v.width - 16
	if maxValueWidth < 40 {
		maxValueWidth = 40
	}

	for i, key := range keys {
		// Key header
		v.lines = append(v.lines, keyStyle.Render(key))

		// Value with word wrapping
		value := v.configmap.Data[key]
		if value == "" {
			v.lines = append(v.lines, valueStyle.Render("  (empty)"))
		} else {
			// Split by newlines first
			valueLines := strings.Split(value, "\n")
			for _, line := range valueLines {
				// Wrap long lines
				wrapped := v.wrapText(line, maxValueWidth)
				for _, wl := range wrapped {
					v.lines = append(v.lines, valueStyle.Render("  "+wl))
				}
			}
		}

		// Add spacing between keys (except last)
		if i < len(keys)-1 {
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
	separatorStyle := lipgloss.NewStyle().Foreground(styles.TextMuted)
	itemStyle := lipgloss.NewStyle().Foreground(styles.Primary)
	infoStyle := lipgloss.NewStyle().Foreground(styles.Secondary)

	breadcrumb := itemStyle.Render(v.namespace) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("configmaps") +
		separatorStyle.Render(" > ") +
		itemStyle.Render(v.configmap.Name) +
		separatorStyle.Render(" - ") +
		infoStyle.Render(fmt.Sprintf("[%s] [%d keys]", v.configmap.Age, len(v.configmap.Data)))
	header.WriteString(breadcrumb)
	header.WriteString("\n")

	// Render visible lines inside box
	maxLines := v.maxVisibleLines()
	endIdx := v.scroll + maxLines
	if endIdx > len(v.lines) {
		endIdx = len(v.lines)
	}

	for i := v.scroll; i < endIdx; i++ {
		content.WriteString(v.lines[i])
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
		BorderForeground(styles.Surface).
		Padding(0, 1).
		Width(v.width - 10).
		Height(v.height - 10)

	boxedContent := boxStyle.Render(content.String())

	// Footer with scroll info - outside the box
	var footer string
	if len(v.lines) > maxLines {
		scrollPct := 0
		maxScroll := len(v.lines) - maxLines
		if maxScroll > 0 {
			scrollPct = (v.scroll * 100) / maxScroll
		}
		footer = styles.StatusMuted.Render(fmt.Sprintf("[%d%%] ↑↓:scroll  PgUp/Dn:fast  g/G:top/end  Esc:close", scrollPct))
	} else {
		footer = styles.StatusMuted.Render("Esc:close")
	}

	return header.String() + boxedContent + "\n" + footer
}

func (v *ConfigMapViewer) Show(cm *k8s.ConfigMapData, namespace string) {
	v.configmap = cm
	v.namespace = namespace
	v.scroll = 0
	v.buildLines()
	v.visible = true
}

func (v *ConfigMapViewer) Hide() {
	v.visible = false
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

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

// HPAViewer displays HPA details in a modal
type HPAViewer struct {
	hpa       *repository.HPAData
	namespace string
	visible   bool
	scroll    int
	width     int
	height    int
	lines     []string
	copied    bool // Show "copied" feedback
}

// HPAViewerClosed is sent when the viewer is closed
type HPAViewerClosed struct{}

func NewHPAViewer() HPAViewer {
	return HPAViewer{}
}

func (v HPAViewer) Init() tea.Cmd {
	return nil
}

func (v HPAViewer) Update(msg tea.Msg) (HPAViewer, tea.Cmd) {
	if !v.visible {
		return v, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			v.visible = false
			v.copied = false
			return v, func() tea.Msg { return HPAViewerClosed{} }
		case "enter":
			// Copy HPA summary to clipboard
			if v.hpa != nil {
				summary := v.buildClipboardContent()
				if err := CopyToClipboard(summary); err == nil {
					v.copied = true
				}
			}
		case "up", "k":
			v.copied = false
			if v.scroll > 0 {
				v.scroll--
			}
		case "down", "j":
			v.copied = false
			maxScroll := len(v.lines) - v.maxVisibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if v.scroll < maxScroll {
				v.scroll++
			}
		case "pgup", "ctrl+u":
			v.copied = false
			v.scroll -= 10
			if v.scroll < 0 {
				v.scroll = 0
			}
		case "pgdown", "ctrl+d":
			v.copied = false
			maxScroll := len(v.lines) - v.maxVisibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			v.scroll += 10
			if v.scroll > maxScroll {
				v.scroll = maxScroll
			}
		case "g", "home":
			v.copied = false
			v.scroll = 0
		case "G", "end":
			v.copied = false
			maxScroll := len(v.lines) - v.maxVisibleLines()
			if maxScroll < 0 {
				maxScroll = 0
			}
			v.scroll = maxScroll
		}
	}

	return v, nil
}

// buildClipboardContent creates a text summary of the HPA for clipboard
func (v HPAViewer) buildClipboardContent() string {
	if v.hpa == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("HPA: %s\n", v.hpa.Name))
	b.WriteString(fmt.Sprintf("Namespace: %s\n", v.hpa.Namespace))
	b.WriteString(fmt.Sprintf("Reference: %s\n", v.hpa.Reference))
	b.WriteString(fmt.Sprintf("Age: %s\n", v.hpa.Age))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Replicas: %d/%d (min: %d, max: %d)\n",
		v.hpa.CurrentReplicas, v.hpa.DesiredReplicas, v.hpa.MinReplicas, v.hpa.MaxReplicas))
	b.WriteString("\n")

	if len(v.hpa.Metrics) > 0 {
		b.WriteString("Metrics:\n")
		for _, m := range v.hpa.Metrics {
			b.WriteString(fmt.Sprintf("  - %s (%s): %s / %s\n", m.Name, m.Type, m.Current, m.Target))
		}
		b.WriteString("\n")
	}

	if len(v.hpa.Conditions) > 0 {
		b.WriteString("Conditions:\n")
		for _, c := range v.hpa.Conditions {
			b.WriteString(fmt.Sprintf("  - %s: %s", c.Type, c.Status))
			if c.Reason != "" {
				b.WriteString(fmt.Sprintf(" (%s)", c.Reason))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (v HPAViewer) maxVisibleLines() int {
	maxLines := v.height - 10
	if maxLines < 5 {
		maxLines = 5
	}
	return maxLines
}

func (v *HPAViewer) buildLines() {
	v.lines = []string{}

	if v.hpa == nil {
		v.lines = append(v.lines, style.StatusMuted.Render("No HPA data"))
		return
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Primary)
	valueStyle := lipgloss.NewStyle().Foreground(style.Text)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(style.Secondary).Underline(true)

	// Basic Info
	v.lines = append(v.lines, headerStyle.Render("Basic Information"))
	v.lines = append(v.lines, "")
	v.lines = append(v.lines, labelStyle.Render("Name:            ")+valueStyle.Render(v.hpa.Name))
	v.lines = append(v.lines, labelStyle.Render("Namespace:       ")+valueStyle.Render(v.hpa.Namespace))
	v.lines = append(v.lines, labelStyle.Render("Age:             ")+valueStyle.Render(v.hpa.Age))
	v.lines = append(v.lines, labelStyle.Render("Reference:       ")+valueStyle.Render(v.hpa.Reference))
	v.lines = append(v.lines, "")

	// Replicas
	v.lines = append(v.lines, headerStyle.Render("Replicas"))
	v.lines = append(v.lines, "")
	v.lines = append(v.lines, labelStyle.Render("Min Replicas:    ")+valueStyle.Render(fmt.Sprintf("%d", v.hpa.MinReplicas)))
	v.lines = append(v.lines, labelStyle.Render("Max Replicas:    ")+valueStyle.Render(fmt.Sprintf("%d", v.hpa.MaxReplicas)))
	v.lines = append(v.lines, labelStyle.Render("Current:         ")+valueStyle.Render(fmt.Sprintf("%d", v.hpa.CurrentReplicas)))
	v.lines = append(v.lines, labelStyle.Render("Desired:         ")+valueStyle.Render(fmt.Sprintf("%d", v.hpa.DesiredReplicas)))
	v.lines = append(v.lines, "")

	// Metrics
	if len(v.hpa.Metrics) > 0 {
		v.lines = append(v.lines, headerStyle.Render("Metrics"))
		v.lines = append(v.lines, "")
		for i, metric := range v.hpa.Metrics {
			v.lines = append(v.lines, labelStyle.Render(fmt.Sprintf("  [%d] Type:     ", i+1))+valueStyle.Render(metric.Type))
			v.lines = append(v.lines, labelStyle.Render("      Name:     ")+valueStyle.Render(metric.Name))
			v.lines = append(v.lines, labelStyle.Render("      Current:  ")+valueStyle.Render(metric.Current))
			v.lines = append(v.lines, labelStyle.Render("      Target:   ")+valueStyle.Render(metric.Target))
			v.lines = append(v.lines, "")
		}
	}

	// Conditions
	if len(v.hpa.Conditions) > 0 {
		v.lines = append(v.lines, headerStyle.Render("Conditions"))
		v.lines = append(v.lines, "")
		for _, cond := range v.hpa.Conditions {
			var statusStyled lipgloss.Style
			if cond.Status == "True" {
				statusStyled = style.StatusRunning
			} else {
				statusStyled = style.StatusError
			}
			v.lines = append(v.lines, labelStyle.Render("  Type:    ")+valueStyle.Render(cond.Type))
			v.lines = append(v.lines, labelStyle.Render("  Status:  ")+statusStyled.Render(cond.Status))
			if cond.Reason != "" {
				v.lines = append(v.lines, labelStyle.Render("  Reason:  ")+valueStyle.Render(cond.Reason))
			}
			if cond.Message != "" {
				// Wrap message if too long
				wrapped := v.wrapText(cond.Message, v.width-30)
				for j, line := range wrapped {
					if j == 0 {
						v.lines = append(v.lines, labelStyle.Render("  Message: ")+valueStyle.Render(line))
					} else {
						v.lines = append(v.lines, "           "+valueStyle.Render(line))
					}
				}
			}
			v.lines = append(v.lines, "")
		}
	}

	// Labels
	if len(v.hpa.Labels) > 0 {
		v.lines = append(v.lines, headerStyle.Render("Labels"))
		v.lines = append(v.lines, "")
		keys := make([]string, 0, len(v.hpa.Labels))
		for k := range v.hpa.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v.lines = append(v.lines, labelStyle.Render("  "+k+": ")+valueStyle.Render(v.hpa.Labels[k]))
		}
		v.lines = append(v.lines, "")
	}

	// Annotations
	if len(v.hpa.Annotations) > 0 {
		v.lines = append(v.lines, headerStyle.Render("Annotations"))
		v.lines = append(v.lines, "")
		keys := make([]string, 0, len(v.hpa.Annotations))
		for k := range v.hpa.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			val := v.hpa.Annotations[k]
			// Truncate long values
			if len(val) > 60 {
				val = val[:57] + "..."
			}
			v.lines = append(v.lines, labelStyle.Render("  "+k+":"))
			v.lines = append(v.lines, "    "+valueStyle.Render(val))
		}
	}
}

func (v HPAViewer) wrapText(text string, maxWidth int) []string {
	if maxWidth < 20 {
		maxWidth = 20
	}
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

func (v HPAViewer) View() string {
	if !v.visible || v.hpa == nil {
		return ""
	}

	var header strings.Builder
	var content strings.Builder

	// Breadcrumb
	separatorStyle := lipgloss.NewStyle().Foreground(style.TextMuted)
	itemStyle := lipgloss.NewStyle().Foreground(style.Primary)
	infoStyle := lipgloss.NewStyle().Foreground(style.Secondary)

	breadcrumb := itemStyle.Render(v.namespace) +
		separatorStyle.Render(" > ") +
		itemStyle.Render("hpa") +
		separatorStyle.Render(" > ") +
		itemStyle.Render(v.hpa.Name) +
		separatorStyle.Render(" - ") +
		infoStyle.Render(fmt.Sprintf("[%s] [%d/%d replicas]", v.hpa.Age, v.hpa.CurrentReplicas, v.hpa.MaxReplicas))
	header.WriteString(breadcrumb)
	header.WriteString("\n")

	// Render visible lines
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

	// Box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Surface).
		Padding(0, 1).
		Width(v.width - 10).
		Height(v.height - 10)

	boxedContent := boxStyle.Render(content.String())

	// Footer with copied indicator
	scrollInfo := ""
	if len(v.lines) > maxLines {
		scrollInfo = fmt.Sprintf("[%d/%d] ", v.scroll+1, len(v.lines)-maxLines+1)
	}

	copiedIndicator := ""
	if v.copied {
		copiedIndicator = style.StatusRunning.Render(" [Copied!]")
	}

	footer := style.StatusMuted.Render(scrollInfo + "↑↓:scroll  Enter:copy  Esc:close") + copiedIndicator

	return header.String() + boxedContent + "\n" + footer
}

func (v *HPAViewer) Show(hpa *repository.HPAData, namespace string) {
	v.hpa = hpa
	v.namespace = namespace
	v.scroll = 0
	v.copied = false
	v.buildLines()
	v.visible = true
}

func (v *HPAViewer) Hide() {
	v.visible = false
	v.copied = false
}

func (v HPAViewer) IsVisible() bool {
	return v.visible
}

func (v *HPAViewer) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.hpa != nil {
		v.buildLines()
	}
}

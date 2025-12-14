package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// MetricsPanel displays resource usage metrics in a two-column layout.
// Left column shows container CPU/memory usage, right column shows node information.
// Supports independent scrolling of each column with arrow key navigation.
type MetricsPanel struct {
	metrics          *repository.PodMetrics
	pod              *repository.PodInfo
	node             *repository.NodeInfo
	viewport         viewport.Model
	ready            bool
	width            int
	height           int
	available        bool
	leftScrollOffset int      // Scroll offset for container resources (left box)
	rightScrollOffset int     // Scroll offset for node info (right box)
	leftContentLines []string // Cached content lines for left box
	rightContentLines []string // Cached content lines for right box
	focusedBox       int      // 0 = left (Container Resources), 1 = right (Node Info)
}

func NewMetricsPanel() MetricsPanel {
	return MetricsPanel{}
}

func (m MetricsPanel) Init() tea.Cmd {
	return nil
}

func (m MetricsPanel) Update(msg tea.Msg) (MetricsPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			// Switch to left box (Container Resources)
			m.focusedBox = 0
			m.updateContent()
			return m, nil
		case "right":
			// Switch to right box (Node Info) if node exists
			if m.node != nil {
				m.focusedBox = 1
				m.updateContent()
			}
			return m, nil
		case "up", "k": // Scroll up
			if m.focusedBox == 0 {
				if m.leftScrollOffset > 0 {
					m.leftScrollOffset--
					m.updateContent()
				}
			} else {
				if m.rightScrollOffset > 0 {
					m.rightScrollOffset--
					m.updateContent()
				}
			}
			return m, nil
		case "down", "j": // Scroll down
			if m.focusedBox == 0 {
				maxScroll := len(m.leftContentLines) - m.getVisibleLines()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.leftScrollOffset < maxScroll {
					m.leftScrollOffset++
					m.updateContent()
				}
			} else {
				maxScroll := len(m.rightContentLines) - m.getVisibleLines()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.rightScrollOffset < maxScroll {
					m.rightScrollOffset++
					m.updateContent()
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *MetricsPanel) getVisibleLines() int {
	// Account for box borders and title
	return m.height - 6
}

func (m MetricsPanel) View() string {
	if !m.ready {
		return style.PanelStyle.Render("Loading metrics...")
	}

	var header strings.Builder
	header.WriteString(style.PanelTitleStyle.Render("Resource Usage"))
	header.WriteString("\n")

	content := header.String() + m.viewport.View()

	// Add metrics-server hint at bottom right if not available
	if !m.available {
		hint := style.StatusMuted.Render("Metrics Server not available")
		hintLen := 28
		padding := m.width - hintLen
		if padding > 0 {
			content += "\n\n" + strings.Repeat(" ", padding) + hint
		}
	}

	return content
}

func (m *MetricsPanel) SetMetrics(metrics *repository.PodMetrics) {
	m.metrics = metrics
	m.available = metrics != nil
	m.updateContent()
}

func (m *MetricsPanel) SetPod(pod *repository.PodInfo) {
	// Only reset scroll/focus if pod actually changed
	podChanged := m.pod == nil || pod == nil ||
		m.pod.Name != pod.Name || m.pod.Namespace != pod.Namespace

	m.pod = pod

	if podChanged {
		m.leftScrollOffset = 0
		m.rightScrollOffset = 0
		m.focusedBox = 0
	}
	m.updateContent()
}

func (m *MetricsPanel) SetNode(node *repository.NodeInfo) {
	m.node = node
	m.updateContent()
}

func (m *MetricsPanel) SetSize(width, height int) {
	m.width = width
	m.height = height - 2

	if !m.ready {
		m.viewport = viewport.New(width, m.height)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = m.height
	}

	m.updateContent()
}

func (m *MetricsPanel) updateContent() {
	if !m.ready {
		return
	}

	if m.pod == nil {
		m.viewport.SetContent(style.StatusMuted.Render("No pod selected"))
		return
	}

	// Build left column (container resources)
	var leftCol strings.Builder
	for _, c := range m.pod.Containers {
		leftCol.WriteString(style.LogContainer.Render(fmt.Sprintf("Container: %s\n", c.Name)))
		leftCol.WriteString("\n")

		// Resources table
		leftCol.WriteString(fmt.Sprintf("  %-14s %s\n", "CPU Request:", formatResourceValue(c.Resources.CPURequest)))
		leftCol.WriteString(fmt.Sprintf("  %-14s %s\n", "CPU Limit:", formatResourceValue(c.Resources.CPULimit)))
		leftCol.WriteString(fmt.Sprintf("  %-14s %s\n", "Mem Request:", formatResourceValue(c.Resources.MemoryRequest)))
		leftCol.WriteString(fmt.Sprintf("  %-14s %s\n", "Mem Limit:", formatResourceValue(c.Resources.MemoryLimit)))

		// Usage metrics (real-time from metrics-server)
		if m.metrics != nil {
			for _, cm := range m.metrics.Containers {
				if cm.Name == c.Name {
					leftCol.WriteString(fmt.Sprintf("  %-14s %s\n", "CPU Usage:", style.StatusRunning.Render(cm.CPUUsage)))
					leftCol.WriteString(fmt.Sprintf("  %-14s %s\n", "Mem Usage:", style.StatusRunning.Render(cm.MemoryUsage)))
					break
				}
			}
		}
		leftCol.WriteString("\n")
	}

	if m.metrics == nil && m.available {
		leftCol.WriteString(style.StatusMuted.Render("Waiting for metrics..."))
	}

	// Build right column (node info) - without title, we add it later
	var rightCol strings.Builder
	// Calculate max value width for truncation (colWidth - label(12) - padding(4))
	maxValueWidth := (m.width-3)/2 - 16
	if maxValueWidth < 10 {
		maxValueWidth = 10
	}
	// Helper to truncate string
	truncate := func(s string, max int) string {
		if len(s) > max {
			return s[:max-3] + "..."
		}
		return s
	}
	if m.node != nil {
		rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "Name:", truncate(m.node.Name, maxValueWidth)))

		statusStyle := style.StatusRunning
		if m.node.Status != "Ready" {
			statusStyle = style.StatusError
		}
		rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "Status:", statusStyle.Render(m.node.Status)))
		rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "Roles:", truncate(m.node.Roles, maxValueWidth)))
		rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "Version:", truncate(m.node.Version, maxValueWidth)))
		rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "Age:", m.node.Age))
		rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "IP:", m.node.InternalIP))
		rightCol.WriteString(fmt.Sprintf("%-12s %d\n", "Pods:", m.node.PodCount))
		if m.node.CPU != "" {
			rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "CPU:", m.node.CPU))
		}
		if m.node.Memory != "" {
			rightCol.WriteString(fmt.Sprintf("%-12s %s\n", "Memory:", m.node.Memory))
		}
	} else if m.pod != nil && m.pod.Node != "" {
		rightCol.WriteString(fmt.Sprintf("%s\n", truncate(m.pod.Node, maxValueWidth+12)))
	}

	// Combine columns side by side if we have node info
	if rightCol.Len() > 0 {
		// Calculate column widths
		colWidth := (m.width - 3) / 2 // -3 for separator and spacing
		visibleLines := m.height - 5

		// Cache content lines for both columns
		m.leftContentLines = strings.Split(leftCol.String(), "\n")
		m.rightContentLines = strings.Split(rightCol.String(), "\n")

		// Build LEFT column content
		var leftContent strings.Builder
		leftTitleStyle := style.SubtitleStyle
		if m.focusedBox == 0 {
			leftTitleStyle = lipgloss.NewStyle().Foreground(style.Success).Bold(true).Italic(true)
		}
		leftTitle := "Container Resources"
		if m.leftScrollOffset > 0 {
			leftTitle += " ▲"
		}
		leftContent.WriteString(leftTitleStyle.Render(leftTitle))
		leftContent.WriteString("\n\n")

		leftStartLine := m.leftScrollOffset
		leftEndLine := leftStartLine + visibleLines
		if leftEndLine > len(m.leftContentLines) {
			leftEndLine = len(m.leftContentLines)
		}

		for i := leftStartLine; i < leftEndLine && i < len(m.leftContentLines); i++ {
			leftContent.WriteString(m.leftContentLines[i])
			leftContent.WriteString("\n")
		}

		if leftEndLine < len(m.leftContentLines) {
			leftContent.WriteString(style.StatusMuted.Render("▼ more..."))
		}

		// Build RIGHT column content
		var rightContent strings.Builder
		rightTitleStyle := style.SubtitleStyle
		if m.focusedBox == 1 {
			rightTitleStyle = lipgloss.NewStyle().Foreground(style.Success).Bold(true).Italic(true)
		}
		rightTitle := "Node Info"
		if m.rightScrollOffset > 0 {
			rightTitle += " ▲"
		}
		rightContent.WriteString(rightTitleStyle.Render(rightTitle))
		rightContent.WriteString("\n\n")

		rightStartLine := m.rightScrollOffset
		rightEndLine := rightStartLine + visibleLines
		if rightEndLine > len(m.rightContentLines) {
			rightEndLine = len(m.rightContentLines)
		}

		for i := rightStartLine; i < rightEndLine && i < len(m.rightContentLines); i++ {
			rightContent.WriteString(m.rightContentLines[i])
			rightContent.WriteString("\n")
		}

		if rightEndLine < len(m.rightContentLines) {
			rightContent.WriteString(style.StatusMuted.Render("▼ more..."))
		}

		// Style columns
		leftColStyle := lipgloss.NewStyle().Width(colWidth).Padding(0, 1)
		rightColStyle := lipgloss.NewStyle().Width(colWidth).Padding(0, 1)

		leftColRendered := leftColStyle.Render(leftContent.String())
		rightColRendered := rightColStyle.Render(rightContent.String())

		// Create full-height separator line
		separatorHeight := m.height
		if separatorHeight < 1 {
			separatorHeight = 1
		}
		var separatorLines strings.Builder
		for i := 0; i < separatorHeight; i++ {
			separatorLines.WriteString("│")
			if i < separatorHeight-1 {
				separatorLines.WriteString("\n")
			}
		}
		separator := lipgloss.NewStyle().
			Foreground(style.Surface).
			Render(separatorLines.String())

		combined := lipgloss.JoinHorizontal(lipgloss.Top, leftColRendered, separator, rightColRendered)
		m.viewport.SetContent(combined)
	} else {
		m.viewport.SetContent(leftCol.String())
	}
}

// stripAnsi removes ANSI escape codes for accurate length calculation
func stripAnsi(s string) string {
	result := s
	for {
		start := strings.Index(result, "\x1b[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

func (m MetricsPanel) checkResourceIssues() []string {
	if m.pod == nil {
		return nil
	}

	var issues []string

	for _, c := range m.pod.Containers {
		if c.Resources.MemoryLimit == "" || c.Resources.MemoryLimit == "0" {
			issues = append(issues, fmt.Sprintf("Container '%s' has no memory limit", c.Name))
		}
		if c.Resources.CPULimit == "" || c.Resources.CPULimit == "0" {
			issues = append(issues, fmt.Sprintf("Container '%s' has no CPU limit", c.Name))
		}
		if c.Resources.MemoryRequest == "" || c.Resources.MemoryRequest == "0" {
			issues = append(issues, fmt.Sprintf("Container '%s' has no memory request", c.Name))
		}
	}

	return issues
}

func formatResourceValue(v string) string {
	if v == "" || v == "0" {
		return style.StatusMuted.Render("not set")
	}
	return v
}

func (m MetricsPanel) IsAvailable() bool {
	return m.available
}

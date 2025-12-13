package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k8sdebug/internal/k8s"
	"github.com/andrebassi/k8sdebug/internal/ui/styles"
)

type MetricsPanel struct {
	metrics   *k8s.PodMetrics
	pod       *k8s.PodInfo
	viewport  viewport.Model
	ready     bool
	width     int
	height    int
	available bool
}

func NewMetricsPanel() MetricsPanel {
	return MetricsPanel{}
}

func (m MetricsPanel) Init() tea.Cmd {
	return nil
}

func (m MetricsPanel) Update(msg tea.Msg) (MetricsPanel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m MetricsPanel) View() string {
	if !m.ready {
		return styles.PanelStyle.Render("Loading metrics...")
	}

	var header strings.Builder
	header.WriteString(styles.PanelTitleStyle.Render("Resource Usage"))
	if !m.available {
		header.WriteString(styles.SubtitleStyle.Render(" (metrics-server not available)"))
	}
	header.WriteString("\n")

	return header.String() + m.viewport.View()
}

func (m *MetricsPanel) SetMetrics(metrics *k8s.PodMetrics) {
	m.metrics = metrics
	m.available = metrics != nil
	m.updateContent()
}

func (m *MetricsPanel) SetPod(pod *k8s.PodInfo) {
	m.pod = pod
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

	var content strings.Builder

	if m.pod == nil {
		content.WriteString(styles.StatusMuted.Render("No pod selected"))
		m.viewport.SetContent(content.String())
		return
	}

	for _, c := range m.pod.Containers {
		content.WriteString(styles.LogContainer.Render(fmt.Sprintf("Container: %s\n", c.Name)))
		content.WriteString("\n")

		// Resources table
		content.WriteString(fmt.Sprintf("  %-14s %s\n", "CPU Request:", formatResourceValue(c.Resources.CPURequest)))
		content.WriteString(fmt.Sprintf("  %-14s %s\n", "CPU Limit:", formatResourceValue(c.Resources.CPULimit)))
		content.WriteString(fmt.Sprintf("  %-14s %s\n", "Mem Request:", formatResourceValue(c.Resources.MemoryRequest)))
		content.WriteString(fmt.Sprintf("  %-14s %s\n", "Mem Limit:", formatResourceValue(c.Resources.MemoryLimit)))

		// Usage metrics (real-time from metrics-server)
		if m.metrics != nil {
			for _, cm := range m.metrics.Containers {
				if cm.Name == c.Name {
					content.WriteString("\n")
					content.WriteString(fmt.Sprintf("  %-14s %s\n", "CPU Usage:", styles.StatusRunning.Render(cm.CPUUsage)))
					content.WriteString(fmt.Sprintf("  %-14s %s\n", "Mem Usage:", styles.StatusRunning.Render(cm.MemoryUsage)))
					break
				}
			}
		}
		content.WriteString("\n")
	}

	if m.metrics == nil && m.available {
		content.WriteString(styles.StatusMuted.Render("Waiting for metrics..."))
	}

	m.viewport.SetContent(content.String())
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
		return styles.StatusMuted.Render("not set")
	}
	return v
}

func (m MetricsPanel) IsAvailable() bool {
	return m.available
}

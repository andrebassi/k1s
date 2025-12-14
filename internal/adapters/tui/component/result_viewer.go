package component

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// ResultViewerCopiedMsg is sent when content is copied to clipboard
type ResultViewerCopiedMsg struct {
	Title   string
	Err     error
	Content string // The content that was copied
}

// ResultViewer displays command output in a scrollable viewport
type ResultViewer struct {
	title      string
	content    string // Store content for clipboard copy
	viewport   viewport.Model
	visible    bool
	ready      bool
	width      int
	height     int
	copyStatus string // Status message after copy
}

func NewResultViewer() ResultViewer {
	return ResultViewer{}
}

func (r ResultViewer) Init() tea.Cmd {
	return nil
}

func (r ResultViewer) Update(msg tea.Msg) (ResultViewer, tea.Cmd) {
	if !r.visible {
		return r, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			r.visible = false
			return r, nil
		case "enter":
			// Copy content to clipboard (strip ANSI codes for clean markdown)
			content := stripAnsiCodes(r.content)
			err := CopyToClipboard(content)
			if err == nil {
				r.copyStatus = "Copied to clipboard!"
			} else {
				r.copyStatus = "Copy failed: " + err.Error()
			}
			return r, nil
		case "g":
			r.viewport.GotoTop()
			return r, nil
		case "G":
			r.viewport.GotoBottom()
			return r, nil
		}
	}

	r.viewport, cmd = r.viewport.Update(msg)
	return r, cmd
}

func (r ResultViewer) View() string {
	if !r.visible {
		return ""
	}

	var b strings.Builder

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(style.Primary).
		Padding(0, 1).
		Width(r.width - 4)
	b.WriteString(titleStyle.Render(r.title))
	b.WriteString("\n")

	// Content viewport
	b.WriteString(r.viewport.View())
	b.WriteString("\n")

	// Footer with scroll info and hints
	footerStyle := lipgloss.NewStyle().
		Foreground(style.Muted).
		Padding(0, 1).
		Width(r.width - 4)

	scrollInfo := ""
	if r.viewport.TotalLineCount() > r.viewport.Height {
		percent := int(float64(r.viewport.YOffset) / float64(r.viewport.TotalLineCount()-r.viewport.Height) * 100)
		scrollInfo = lipgloss.NewStyle().Foreground(style.Secondary).Render(
			" | " + strconv.Itoa(percent) + "%",
		)
	}

	footer := "j/k scroll • g/G top/bottom • enter copy • q/esc close" + scrollInfo
	if r.copyStatus != "" {
		footer = footer + " - " + lipgloss.NewStyle().Foreground(style.Success).Bold(true).Render(r.copyStatus)
	}
	b.WriteString(footerStyle.Render(footer))

	// Wrap in a box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Primary).
		Background(style.Background)

	return boxStyle.Render(b.String())
}

func (r *ResultViewer) Show(title, content string, width, height int) {
	r.title = title
	r.content = content // Store content for clipboard copy
	r.width = width
	r.height = height
	r.visible = true
	r.copyStatus = "" // Clear previous copy status

	// Initialize viewport
	viewportHeight := max(height-6, 5)
	viewportWidth := max(width-6, 20)

	r.viewport = viewport.New(viewportWidth, viewportHeight)
	r.viewport.SetContent(content)
	r.ready = true
}

func (r *ResultViewer) Hide() {
	r.visible = false
}

func (r ResultViewer) IsVisible() bool {
	return r.visible
}

func (r *ResultViewer) SetSize(width, height int) {
	r.width = width
	r.height = height
	if r.ready {
		r.viewport.Width = width - 6
		r.viewport.Height = height - 6
	}
}

// stripAnsiCodes removes ANSI escape sequences and formats content as clean markdown
func stripAnsiCodes(text string) string {
	// Remove all ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cleaned := ansiRegex.ReplaceAllString(text, "")

	// Convert section headers to markdown format
	lines := strings.Split(cleaned, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the start
		if len(result) == 0 && trimmed == "" {
			continue
		}

		// Detect section headers (lines that start with a word and no leading spaces in original)
		if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// Check if it looks like a section header
			if strings.HasSuffix(trimmed, ":") || isKnownSection(trimmed) {
				// Add markdown header
				if len(result) > 0 {
					result = append(result, "") // Add blank line before header
				}
				result = append(result, "## "+trimmed)
				continue
			}
		}

		// Detect container headers
		if strings.HasPrefix(trimmed, "Container:") {
			if len(result) > 0 {
				result = append(result, "")
			}
			result = append(result, "### "+trimmed)
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// isKnownSection checks if the line is a known section header
func isKnownSection(line string) bool {
	sections := []string{
		"Pod Info", "Network", "Services", "Ingresses", "VirtualServices",
		"Gateways", "Tolerations", "Node Selector", "Volumes", "ConfigMaps Used",
		"Secrets Used", "Resources", "Ports", "Probes", "Security Context",
		"Volume Mounts", "Environment Variables",
	}
	for _, s := range sections {
		if strings.HasPrefix(line, s) {
			return true
		}
	}
	return false
}

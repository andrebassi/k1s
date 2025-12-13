package components

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k8sdebug/internal/ui/styles"
)

// ResultViewer displays command output in a scrollable viewport
type ResultViewer struct {
	title    string
	viewport viewport.Model
	visible  bool
	ready    bool
	width    int
	height   int
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
		Foreground(styles.Primary).
		Background(styles.Surface).
		Padding(0, 1).
		Width(r.width - 4)
	b.WriteString(titleStyle.Render(r.title))
	b.WriteString("\n")

	// Content viewport
	b.WriteString(r.viewport.View())
	b.WriteString("\n")

	// Footer with scroll info and hints
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Background(styles.Surface).
		Padding(0, 1).
		Width(r.width - 4)

	scrollInfo := ""
	if r.viewport.TotalLineCount() > r.viewport.Height {
		percent := int(float64(r.viewport.YOffset) / float64(r.viewport.TotalLineCount()-r.viewport.Height) * 100)
		scrollInfo = lipgloss.NewStyle().Foreground(styles.Secondary).Render(
			" | " + strconv.Itoa(percent) + "%",
		)
	}

	footer := "j/k scroll • g/G top/bottom • q/esc close" + scrollInfo
	b.WriteString(footerStyle.Render(footer))

	// Wrap in a box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Background(styles.Background)

	return boxStyle.Render(b.String())
}

func (r *ResultViewer) Show(title, content string, width, height int) {
	r.title = title
	r.width = width
	r.height = height
	r.visible = true

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

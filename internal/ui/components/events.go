package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/k8s"
	"github.com/andrebassi/k1s/internal/ui/styles"
)

type EventsPanel struct {
	events      []k8s.EventInfo
	viewport    viewport.Model
	ready       bool
	width       int
	height      int
	cursor      int
	showAll     bool
	copyStatus  string
	searching   bool
	searchInput textinput.Model
	filter      string
}

func NewEventsPanel() EventsPanel {
	ti := textinput.New()
	ti.Placeholder = "Search events..."
	ti.CharLimit = 100
	ti.Width = 30

	return EventsPanel{
		searchInput: ti,
	}
}

func (e EventsPanel) Init() tea.Cmd {
	return nil
}

func (e EventsPanel) Update(msg tea.Msg) (EventsPanel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode
		if e.searching {
			switch msg.String() {
			case "esc":
				// Esc behavior:
				// 1. If filter has content: clear filter
				// 2. If filter empty: exit search mode
				if e.filter != "" {
					e.filter = ""
					e.searchInput.SetValue("")
					e.updateContent()
					return e, nil
				}
				e.searching = false
				e.searchInput.Blur()
				return e, nil
			case "enter":
				e.searching = false
				e.searchInput.Blur()
				e.filter = e.searchInput.Value()
				e.updateContent()
				return e, nil
			default:
				e.searchInput, cmd = e.searchInput.Update(msg)
				// Live search as you type
				e.filter = e.searchInput.Value()
				e.updateContent()
				return e, cmd
			}
		}

		// Normal mode
		switch msg.String() {
		case "enter":
			// Copy events to clipboard
			content := e.getPlainTextEvents()
			err := CopyToClipboard(content)
			if err == nil {
				e.copyStatus = "Copied to clipboard!"
			} else {
				e.copyStatus = "Copy failed: " + err.Error()
			}
			return e, nil
		case "/":
			e.searching = true
			e.searchInput.Focus()
			return e, textinput.Blink
		case "w":
			e.showAll = !e.showAll
			e.updateContent()
		case "j", "down":
			if e.cursor < len(e.getDisplayedEvents())-1 {
				e.cursor++
			}
		case "k", "up":
			if e.cursor > 0 {
				e.cursor--
			}
		}
	}

	e.viewport, cmd = e.viewport.Update(msg)
	return e, cmd
}

func (e EventsPanel) View() string {
	if !e.ready {
		return styles.PanelStyle.Render("Loading events...")
	}

	var header strings.Builder
	header.WriteString(styles.PanelTitleStyle.Render("Events"))

	warningCount := e.warningCount()
	if warningCount > 0 {
		header.WriteString(styles.EventWarning.Render(fmt.Sprintf(" [%d warnings]", warningCount)))
	}

	if !e.showAll {
		header.WriteString(styles.SubtitleStyle.Render(" (warnings only, press 'w' for all)"))
	}

	// Show search input or filter indicator
	if e.searching {
		header.WriteString("  ")
		header.WriteString(e.searchInput.View())
	} else if e.filter != "" {
		filterStyle := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
		header.WriteString(filterStyle.Render(fmt.Sprintf("  [filter: %s]", e.filter)))
	}
	header.WriteString("\n")

	result := header.String() + e.viewport.View()

	// Show copy status at bottom right
	if e.copyStatus != "" {
		padding := e.width - len(e.copyStatus) - 4
		if padding < 0 {
			padding = 0
		}
		statusMsg := lipgloss.NewStyle().Foreground(styles.Success).Bold(true).Render(e.copyStatus)
		result += strings.Repeat(" ", padding) + statusMsg
	}

	return result
}

func (e *EventsPanel) SetEvents(events []k8s.EventInfo) {
	e.events = events
	e.cursor = 0
	e.copyStatus = "" // Clear copy status when events update
	e.updateContent()
}

func (e *EventsPanel) SetSize(width, height int) {
	e.width = width
	e.height = height - 2

	if !e.ready {
		e.viewport = viewport.New(width, e.height)
		e.ready = true
	} else {
		e.viewport.Width = width
		e.viewport.Height = e.height
	}

	e.updateContent()
}

func (e *EventsPanel) updateContent() {
	if !e.ready {
		return
	}

	var content strings.Builder
	events := e.getDisplayedEvents()

	if len(events) == 0 {
		content.WriteString(styles.StatusMuted.Render("No events found"))
	} else {
		for i, event := range events {
			line := e.formatEvent(event, i == e.cursor)
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	e.viewport.SetContent(content.String())
}

func (e EventsPanel) getDisplayedEvents() []k8s.EventInfo {
	var filtered []k8s.EventInfo

	// First filter by warning type if not showing all
	for _, event := range e.events {
		if e.showAll || event.Type == "Warning" {
			filtered = append(filtered, event)
		}
	}

	// Then filter by search term
	if e.filter != "" {
		filter := strings.ToLower(e.filter)
		var searchFiltered []k8s.EventInfo
		for _, event := range filtered {
			if strings.Contains(strings.ToLower(event.Message), filter) ||
				strings.Contains(strings.ToLower(event.Reason), filter) ||
				strings.Contains(strings.ToLower(event.Type), filter) {
				searchFiltered = append(searchFiltered, event)
			}
		}
		filtered = searchFiltered
	}

	return filtered
}

func (e EventsPanel) formatEvent(event k8s.EventInfo, selected bool) string {
	var b strings.Builder

	typeStyle := styles.EventNormal
	if event.Type == "Warning" {
		typeStyle = styles.EventWarning
	}

	prefix := "  "
	if selected {
		prefix = "> "
		b.WriteString(styles.CursorStyle.Render(prefix))
	} else {
		b.WriteString(prefix)
	}

	b.WriteString(typeStyle.Render(fmt.Sprintf("%-8s", event.Type)))
	b.WriteString(" ")
	b.WriteString(styles.LogTimestamp.Render(fmt.Sprintf("%-6s", event.Age)))
	b.WriteString(" ")
	b.WriteString(styles.LogContainer.Render(fmt.Sprintf("%-20s", styles.Truncate(event.Reason, 20))))
	b.WriteString(" ")

	maxMsgLen := e.width - 40
	if maxMsgLen < 20 {
		maxMsgLen = 20
	}
	msg := styles.Truncate(event.Message, maxMsgLen)
	b.WriteString(styles.LogNormal.Render(msg))

	return b.String()
}

func (e EventsPanel) warningCount() int {
	count := 0
	for _, event := range e.events {
		if event.Type == "Warning" {
			count++
		}
	}
	return count
}

func (e EventsPanel) SelectedEvent() *k8s.EventInfo {
	events := e.getDisplayedEvents()
	if e.cursor >= 0 && e.cursor < len(events) {
		return &events[e.cursor]
	}
	return nil
}

func (e EventsPanel) EventCount() int {
	return len(e.events)
}

func (e EventsPanel) WarningCount() int {
	return e.warningCount()
}

func (e EventsPanel) IsSearching() bool {
	return e.searching
}

func (e *EventsPanel) ClearSearch() {
	e.searching = false
	e.filter = ""
	e.searchInput.SetValue("")
	e.searchInput.Blur()
	e.updateContent()
}

// getPlainTextEvents returns events as plain text without ANSI codes
func (e EventsPanel) getPlainTextEvents() string {
	var content strings.Builder
	events := e.getDisplayedEvents()

	for _, event := range events {
		content.WriteString(fmt.Sprintf("%-8s %-6s %-20s %s\n",
			event.Type,
			event.Age,
			event.Reason,
			event.Message))
	}

	return content.String()
}

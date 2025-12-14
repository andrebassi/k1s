package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// EventsPanel displays Kubernetes events with filtering capabilities.
// Features include: warning-only filter, text search, and clipboard copy.
type EventsPanel struct {
	events      []repository.EventInfo
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

// NewEventsPanel creates a new events panel with default settings.
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
				// Esc: clear filter or exit search mode
				if e.filter != "" {
					e.filter = ""
					e.searchInput.SetValue("")
					e.updateContent()
					return e, nil
				}
				e.searching = false
				e.searchInput.Blur()
				return e, nil
			case "tab", "enter":
				// Tab/Enter: exit search mode, keep filter
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
		return style.PanelStyle.Render("Loading events...")
	}

	var header strings.Builder
	header.WriteString(style.PanelTitleStyle.Render("Events"))

	warningCount := e.warningCount()
	if warningCount > 0 {
		header.WriteString(style.EventWarning.Render(fmt.Sprintf(" [%d warnings]", warningCount)))
	}

	if !e.showAll {
		header.WriteString(style.SubtitleStyle.Render(" (warnings only, press 'w' for all)"))
	}

	// Show search input or filter indicator
	if e.searching {
		header.WriteString("  ")
		header.WriteString(e.searchInput.View())
	} else if e.filter != "" {
		filterStyle := lipgloss.NewStyle().Foreground(style.Warning).Bold(true)
		header.WriteString(filterStyle.Render(fmt.Sprintf("  [filter: %s]", e.filter)))
	}
	header.WriteString("\n")

	result := header.String() + e.viewport.View()

	// Show "No events found" at bottom right
	events := e.getDisplayedEvents()
	if len(events) == 0 {
		hint := style.StatusMuted.Render("No events found")
		hintLen := 15
		padding := e.width - hintLen
		if padding > 0 {
			result += "\n" + strings.Repeat(" ", padding) + hint
		}
	}

	// Show copy status at bottom right
	if e.copyStatus != "" {
		padding := e.width - len(e.copyStatus) - 4
		if padding < 0 {
			padding = 0
		}
		statusMsg := lipgloss.NewStyle().Foreground(style.Success).Bold(true).Render(e.copyStatus)
		result += strings.Repeat(" ", padding) + statusMsg
	}

	return result
}

func (e *EventsPanel) SetEvents(events []repository.EventInfo) {
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

	for i, event := range events {
		line := e.formatEvent(event, i == e.cursor)
		content.WriteString(line)
		content.WriteString("\n")
	}

	e.viewport.SetContent(content.String())
}

func (e EventsPanel) getDisplayedEvents() []repository.EventInfo {
	var filtered []repository.EventInfo

	// First filter by warning type if not showing all
	for _, event := range e.events {
		if e.showAll || event.Type == "Warning" {
			filtered = append(filtered, event)
		}
	}

	// Then filter by search term
	if e.filter != "" {
		filter := strings.ToLower(e.filter)
		var searchFiltered []repository.EventInfo
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

func (e EventsPanel) formatEvent(event repository.EventInfo, selected bool) string {
	var b strings.Builder

	typeStyle := style.EventNormal
	if event.Type == "Warning" {
		typeStyle = style.EventWarning
	}

	prefix := "  "
	if selected {
		prefix = "> "
		b.WriteString(style.CursorStyle.Render(prefix))
	} else {
		b.WriteString(prefix)
	}

	b.WriteString(typeStyle.Render(fmt.Sprintf("%-8s", event.Type)))
	b.WriteString(" ")
	b.WriteString(style.LogTimestamp.Render(fmt.Sprintf("%-6s", event.Age)))
	b.WriteString(" ")
	b.WriteString(style.LogContainer.Render(fmt.Sprintf("%-20s", style.Truncate(event.Reason, 20))))
	b.WriteString(" ")

	maxMsgLen := e.width - 40
	if maxMsgLen < 20 {
		maxMsgLen = 20
	}
	msg := style.Truncate(event.Message, maxMsgLen)
	b.WriteString(style.LogNormal.Render(msg))

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

func (e EventsPanel) SelectedEvent() *repository.EventInfo {
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

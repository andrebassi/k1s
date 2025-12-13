package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k8sdebug/internal/k8s"
	"github.com/andrebassi/k8sdebug/internal/ui/styles"
)

type EventsPanel struct {
	events    []k8s.EventInfo
	viewport  viewport.Model
	ready     bool
	width     int
	height    int
	cursor    int
	showAll   bool
}

func NewEventsPanel() EventsPanel {
	return EventsPanel{}
}

func (e EventsPanel) Init() tea.Cmd {
	return nil
}

func (e EventsPanel) Update(msg tea.Msg) (EventsPanel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
	header.WriteString("\n")

	return header.String() + e.viewport.View()
}

func (e *EventsPanel) SetEvents(events []k8s.EventInfo) {
	e.events = events
	e.cursor = 0
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
	if e.showAll {
		return e.events
	}

	var warnings []k8s.EventInfo
	for _, event := range e.events {
		if event.Type == "Warning" {
			warnings = append(warnings, event)
		}
	}
	return warnings
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

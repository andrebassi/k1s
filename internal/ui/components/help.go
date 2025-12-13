package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/ui/styles"
)

type HelpEntry struct {
	Key  string
	Desc string
}

type HelpPanel struct {
	entries [][]HelpEntry
	width   int
	height  int
	visible bool
}

func NewHelpPanel() HelpPanel {
	return HelpPanel{
		entries: defaultHelpEntries(),
	}
}

func defaultHelpEntries() [][]HelpEntry {
	return [][]HelpEntry{
		{
			{Key: "↑/k", Desc: "move up"},
			{Key: "↓/j", Desc: "move down"},
			{Key: "g", Desc: "first item"},
			{Key: "G", Desc: "last item"},
		},
		{
			{Key: "PgUp", Desc: "page up"},
			{Key: "PgDn", Desc: "page down"},
			{Key: "enter", Desc: "select"},
			{Key: "esc", Desc: "back"},
		},
		{
			{Key: "/", Desc: "search/filter"},
			{Key: "c", Desc: "clear filter"},
			{Key: "r", Desc: "refresh"},
		},
		{
			{Key: "n", Desc: "change namespace"},
			{Key: "t", Desc: "change resource type"},
		},
		{
			{Key: "tab", Desc: "next panel"},
			{Key: "S-tab", Desc: "prev panel"},
			{Key: "1-4", Desc: "focus panel"},
		},
		{
			{Key: "f", Desc: "follow logs"},
			{Key: "e", Desc: "next error"},
			{Key: "w", Desc: "wrap lines"},
			{Key: "v", Desc: "fullscreen"},
		},
		{
			{Key: "?", Desc: "toggle help"},
			{Key: "q", Desc: "quit"},
		},
	}
}

func (h *HelpPanel) SetSize(width, height int) {
	h.width = width
	h.height = height
}

func (h *HelpPanel) Toggle() {
	h.visible = !h.visible
}

func (h *HelpPanel) Show() {
	h.visible = true
}

func (h *HelpPanel) Hide() {
	h.visible = false
}

func (h HelpPanel) IsVisible() bool {
	return h.visible
}

func (h HelpPanel) View() string {
	if !h.visible {
		return ""
	}

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		MarginBottom(1).
		Render("Keyboard Shortcuts")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Render entries in two columns
	for _, group := range h.entries {
		for _, entry := range group {
			key := styles.HelpKeyStyle.Render(styles.PadRight(entry.Key, 8))
			desc := styles.HelpDescStyle.Render(entry.Desc)
			b.WriteString("  ")
			b.WriteString(key)
			b.WriteString(" ")
			b.WriteString(desc)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true).
		Render("Press ? or esc to close")
	b.WriteString(footer)

	content := b.String()

	// Modal style with background
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Background(styles.Background).
		Padding(1, 3).
		MarginTop(1).
		MarginBottom(1)

	return modalStyle.Render(content)
}

func (h HelpPanel) ShortHelp() string {
	shortcuts := []HelpEntry{
		{Key: "↑↓/jk", Desc: "nav"},
		{Key: "enter", Desc: "select"},
		{Key: "esc", Desc: "back"},
		{Key: "/", Desc: "filter"},
		{Key: "?", Desc: "help"},
	}

	var parts []string
	for _, s := range shortcuts {
		parts = append(parts,
			styles.HelpKeyStyle.Render(s.Key)+
				styles.HelpSeparator.Render(":")+
				styles.HelpDescStyle.Render(s.Desc))
	}

	return strings.Join(parts, styles.HelpSeparator.Render(" • "))
}

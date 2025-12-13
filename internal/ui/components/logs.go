package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/k8s"
	"github.com/andrebassi/k1s/internal/ui/styles"
)

type TimeFilter int

const (
	TimeFilterAll TimeFilter = iota
	TimeFilter5Min
	TimeFilter15Min
	TimeFilter1Hour
	TimeFilter6Hours
)

var timeFilterLabels = map[TimeFilter]string{
	TimeFilterAll:    "All",
	TimeFilter5Min:   "5m",
	TimeFilter15Min:  "15m",
	TimeFilter1Hour:  "1h",
	TimeFilter6Hours: "6h",
}

type LogsPanel struct {
	logs         []k8s.LogLine
	viewport     viewport.Model
	ready        bool
	width        int
	height       int
	following    bool
	filter       string
	containers   []string // list of container names
	containerIdx int      // -1 = all, 0+ = specific container
	showPrevious bool     // show previous container logs
	searching    bool     // true when search input is active
	searchInput  textinput.Model
	timeFilter   TimeFilter
	copyStatus   string // Status message after copy
}

func NewLogsPanel() LogsPanel {
	ti := textinput.New()
	ti.Placeholder = "Search logs..."
	ti.CharLimit = 100
	ti.Width = 30

	return LogsPanel{
		following:    true,
		containerIdx: -1, // -1 means all containers
		searchInput:  ti,
	}
}

func (l LogsPanel) Init() tea.Cmd {
	return nil
}

func (l LogsPanel) Update(msg tea.Msg) (LogsPanel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode
		if l.searching {
			switch msg.String() {
			case "esc":
				// Esc: clear filter or exit search mode
				if l.filter != "" {
					l.filter = ""
					l.searchInput.SetValue("")
					l.updateContent()
					return l, nil
				}
				l.searching = false
				l.searchInput.Blur()
				return l, nil
			case "tab", "enter":
				// Tab/Enter: exit search mode, keep filter
				l.searching = false
				l.searchInput.Blur()
				l.filter = l.searchInput.Value()
				l.updateContent()
				return l, nil
			default:
				l.searchInput, cmd = l.searchInput.Update(msg)
				// Live search as you type
				l.filter = l.searchInput.Value()
				l.updateContent()
				return l, cmd
			}
		}

		// Normal mode
		switch msg.String() {
		case "enter":
			// Copy logs to clipboard
			content := l.getPlainTextLogs()
			err := CopyToClipboard(content)
			if err == nil {
				l.copyStatus = "Copied to clipboard!"
			} else {
				l.copyStatus = "Copy failed: " + err.Error()
			}
			return l, nil
		case "/":
			l.searching = true
			l.searchInput.Focus()
			return l, textinput.Blink
		case "c":
			// Clear filter
			l.filter = ""
			l.searchInput.SetValue("")
			l.updateContent()
			return l, nil
		case "f":
			l.following = !l.following
			if l.following {
				l.viewport.GotoBottom()
			}
		case "e":
			l.jumpToNextError()
		case "g":
			l.viewport.GotoTop()
		case "G":
			l.viewport.GotoBottom()
		case "[":
			l.prevContainer()
		case "]":
			l.nextContainer()
		case "P":
			l.showPrevious = !l.showPrevious
			// Note: actual previous logs fetch handled by dashboard
		case "T":
			l.cycleTimeFilter()
			l.updateContent()
			return l, nil
		}
	}

	l.viewport, cmd = l.viewport.Update(msg)
	return l, cmd
}

func (l LogsPanel) View() string {
	if !l.ready {
		return styles.PanelStyle.Render("Loading logs...")
	}

	var header strings.Builder
	header.WriteString(styles.PanelTitleStyle.Render("Logs"))

	// Show container indicator
	if len(l.containers) > 0 {
		containerName := "all"
		if l.containerIdx >= 0 && l.containerIdx < len(l.containers) {
			containerName = l.containers[l.containerIdx]
		}
		header.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf(" [%s]", containerName)))

		// Show navigation hint if multiple containers
		if len(l.containers) > 1 {
			header.WriteString(styles.HelpDescStyle.Render(fmt.Sprintf(" (%d/%d)", l.containerIdx+2, len(l.containers)+1)))
		}
	}

	if l.showPrevious {
		header.WriteString(styles.EventWarning.Render(" [Previous]"))
	}
	if l.following && !l.showPrevious {
		header.WriteString(styles.StatusRunning.Render(" [Following]"))
	}

	// Show time filter indicator
	if l.timeFilter != TimeFilterAll {
		header.WriteString(styles.HelpKeyStyle.Render(fmt.Sprintf(" [%s]", timeFilterLabels[l.timeFilter])))
	}

	// Show filter indicator
	if l.filter != "" && !l.searching {
		header.WriteString(styles.HelpKeyStyle.Render(fmt.Sprintf(" /%s", l.filter)))
		header.WriteString(styles.HelpDescStyle.Render(" (c:clear)"))
	}

	header.WriteString("\n")

	// Show search input if searching
	if l.searching {
		header.WriteString(styles.HelpKeyStyle.Render("/"))
		header.WriteString(l.searchInput.View())
		header.WriteString("\n")
	}

	result := header.String() + l.viewport.View()

	// Show copy status at bottom right
	if l.copyStatus != "" {
		padding := l.width - len(l.copyStatus) - 4
		if padding < 0 {
			padding = 0
		}
		statusMsg := lipgloss.NewStyle().Foreground(styles.Success).Bold(true).Render(l.copyStatus)
		result += strings.Repeat(" ", padding) + statusMsg
	}

	return result
}

func (l *LogsPanel) SetLogs(logs []k8s.LogLine) {
	l.logs = logs
	l.copyStatus = "" // Clear copy status when logs update
	l.updateContent()
}

func (l *LogsPanel) SetSize(width, height int) {
	l.width = width
	l.height = height - 2

	if !l.ready {
		l.viewport = viewport.New(width, l.height)
		l.ready = true
	} else {
		l.viewport.Width = width
		l.viewport.Height = l.height
	}

	l.updateContent()
}

func (l *LogsPanel) SetContainers(containers []string) {
	l.containers = containers
	l.containerIdx = -1 // reset to "all" when containers change
}

func (l *LogsPanel) nextContainer() {
	if len(l.containers) == 0 {
		return
	}
	// Cycle: -1 (all) -> 0 -> 1 -> ... -> len-1 -> -1
	l.containerIdx++
	if l.containerIdx >= len(l.containers) {
		l.containerIdx = -1
	}
	l.updateContent()
}

func (l *LogsPanel) prevContainer() {
	if len(l.containers) == 0 {
		return
	}
	// Cycle: -1 (all) <- 0 <- 1 <- ... <- len-1 <- -1
	l.containerIdx--
	if l.containerIdx < -1 {
		l.containerIdx = len(l.containers) - 1
	}
	l.updateContent()
}

func (l LogsPanel) SelectedContainer() string {
	if l.containerIdx >= 0 && l.containerIdx < len(l.containers) {
		return l.containers[l.containerIdx]
	}
	return "" // empty means all
}

func (l LogsPanel) ShowPrevious() bool {
	return l.showPrevious
}

func (l *LogsPanel) cycleTimeFilter() {
	l.timeFilter = (l.timeFilter + 1) % 5
}

func (l LogsPanel) getTimeFilterDuration() time.Duration {
	switch l.timeFilter {
	case TimeFilter5Min:
		return 5 * time.Minute
	case TimeFilter15Min:
		return 15 * time.Minute
	case TimeFilter1Hour:
		return time.Hour
	case TimeFilter6Hours:
		return 6 * time.Hour
	default:
		return 0 // No time filter
	}
}

func (l *LogsPanel) SetFilter(filter string) {
	l.filter = filter
	l.updateContent()
}

func (l *LogsPanel) ToggleFollow() {
	l.following = !l.following
	if l.following {
		l.viewport.GotoBottom()
	}
}

func (l *LogsPanel) updateContent() {
	if !l.ready {
		return
	}

	var content strings.Builder
	filteredLogs := l.getFilteredLogs()

	for _, log := range filteredLogs {
		line := l.formatLogLine(log)
		content.WriteString(line)
		content.WriteString("\n")
	}

	l.viewport.SetContent(content.String())

	if l.following {
		l.viewport.GotoBottom()
	}
}

func (l LogsPanel) getFilteredLogs() []k8s.LogLine {
	var filtered []k8s.LogLine
	now := time.Now()
	timeDuration := l.getTimeFilterDuration()

	// First filter by container if specific container selected
	selectedContainer := l.SelectedContainer()
	for _, log := range l.logs {
		if selectedContainer != "" && log.Container != selectedContainer {
			continue
		}
		filtered = append(filtered, log)
	}

	// Then filter by time if set
	if timeDuration > 0 {
		cutoff := now.Add(-timeDuration)
		var timeFiltered []k8s.LogLine
		for _, log := range filtered {
			if !log.Timestamp.IsZero() && log.Timestamp.After(cutoff) {
				timeFiltered = append(timeFiltered, log)
			}
		}
		filtered = timeFiltered
	}

	// Then filter by text filter if set
	if l.filter != "" {
		filter := strings.ToLower(l.filter)
		var textFiltered []k8s.LogLine
		for _, log := range filtered {
			if strings.Contains(strings.ToLower(log.Content), filter) {
				textFiltered = append(textFiltered, log)
			}
		}
		filtered = textFiltered
	}

	return filtered
}

func (l LogsPanel) formatLogLine(log k8s.LogLine) string {
	var b strings.Builder

	if !log.Timestamp.IsZero() {
		ts := log.Timestamp.Format("15:04:05")
		b.WriteString(styles.LogTimestamp.Render(ts))
		b.WriteString(" ")
	}

	// Show container name when viewing all containers
	if log.Container != "" && l.containerIdx == -1 && len(l.containers) > 1 {
		b.WriteString(styles.LogContainer.Render(fmt.Sprintf("[%s]", log.Container)))
		b.WriteString(" ")
	}

	if log.IsError {
		b.WriteString(styles.LogError.Render(log.Content))
	} else {
		b.WriteString(styles.LogNormal.Render(log.Content))
	}

	return b.String()
}

func (l *LogsPanel) jumpToNextError() {
	content := l.viewport.View()
	lines := strings.Split(content, "\n")
	currentLine := l.viewport.YOffset

	for i := currentLine + 1; i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), "error") ||
			strings.Contains(strings.ToLower(lines[i]), "fatal") ||
			strings.Contains(strings.ToLower(lines[i]), "panic") {
			l.viewport.SetYOffset(i)
			return
		}
	}

	for i := 0; i < currentLine; i++ {
		if strings.Contains(strings.ToLower(lines[i]), "error") ||
			strings.Contains(strings.ToLower(lines[i]), "fatal") ||
			strings.Contains(strings.ToLower(lines[i]), "panic") {
			l.viewport.SetYOffset(i)
			return
		}
	}
}

func (l LogsPanel) IsFollowing() bool {
	return l.following
}

func (l LogsPanel) LogCount() int {
	return len(l.logs)
}

func (l LogsPanel) ErrorCount() int {
	count := 0
	for _, log := range l.logs {
		if log.IsError {
			count++
		}
	}
	return count
}

func (l LogsPanel) IsSearching() bool {
	return l.searching
}

func (l *LogsPanel) ClearSearch() {
	l.searching = false
	l.filter = ""
	l.searchInput.SetValue("")
	l.searchInput.Blur()
	l.updateContent()
}

func (l LogsPanel) Filter() string {
	return l.filter
}

// getPlainTextLogs returns logs as plain text without ANSI codes
func (l LogsPanel) getPlainTextLogs() string {
	var content strings.Builder
	filteredLogs := l.getFilteredLogs()

	for _, log := range filteredLogs {
		if !log.Timestamp.IsZero() {
			ts := log.Timestamp.Format("15:04:05")
			content.WriteString(ts)
			content.WriteString(" ")
		}

		// Show container name when viewing all containers
		if log.Container != "" && l.containerIdx == -1 && len(l.containers) > 1 {
			content.WriteString(fmt.Sprintf("[%s] ", log.Container))
		}

		content.WriteString(log.Content)
		content.WriteString("\n")
	}

	return content.String()
}

// stripLogsAnsi removes ANSI escape sequences from text
func stripLogsAnsi(text string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(text, "")
}

package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - optimized for readability on dark terminals
	Primary     = lipgloss.Color("#3B82F6") // Dark blue
	Secondary   = lipgloss.Color("#22D3EE") // Bright cyan - good contrast
	Success     = lipgloss.Color("#4ADE80") // Bright green - very readable
	Warning     = lipgloss.Color("#FBBF24") // Amber - warm and visible
	Error       = lipgloss.Color("#F87171") // Soft red - not too harsh
	Muted       = lipgloss.Color("#9CA3AF") // Gray - subtle but readable
	Background  = lipgloss.Color("#111827") // Dark background
	Surface     = lipgloss.Color("#4B5563") // Lighter surface for borders
	Text        = lipgloss.Color("#F3F4F6") // Off-white - less eye strain
	TextMuted   = lipgloss.Color("#D1D5DB") // Light gray - readable muted text
	Accent      = lipgloss.Color("#F472B6") // Pink accent for special items

	// Base styles
	BaseStyle = lipgloss.NewStyle()

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Italic(true)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Surface).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	// List styles
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(Text)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color("#1F2937")).
				Background(Primary).
				Bold(true)

	CursorStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	// Status styles
	StatusRunning = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	StatusPending = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	StatusError = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	StatusMuted = lipgloss.NewStyle().
			Foreground(Muted)

	// Log styles
	LogTimestamp = lipgloss.NewStyle().
			Foreground(Muted)

	LogContainer = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	LogError = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	LogNormal = lipgloss.NewStyle().
			Foreground(Text)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Primary).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(Surface)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(Text)

	// Help styles
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	HelpSeparator = lipgloss.NewStyle().
			Foreground(Surface)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Foreground(Secondary).
				Background(lipgloss.Color("#1F2937")).
				Bold(true)

	// Breadcrumb
	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	BreadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true)

	// Event type styles
	EventWarning = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	EventNormal = lipgloss.NewStyle().
			Foreground(Success)

	// Spinner
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Primary)

	// Credit style
	CreditStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Search input style
	SearchStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Surface).
			Padding(0, 1)
)

func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "Running", "Completed", "Active", "Ready":
		return StatusRunning
	case "Pending", "Progressing", "ContainerCreating":
		return StatusPending
	case "Failed", "Error", "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "OOMKilled", "NotReady", "Terminating":
		return StatusError
	default:
		return StatusMuted
	}
}

func RenderWithWidth(s lipgloss.Style, content string, width int) string {
	return s.Width(width).Render(content)
}

func Truncate(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func PadRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + spaces(width-len(s))
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

// Credit returns the credit line
func Credit() string {
	heart := lipgloss.NewStyle().Foreground(Error).Render("♥")
	return CreditStyle.Render("built with " + heart + " by ") +
		lipgloss.NewStyle().Foreground(Primary).Bold(true).Render("doganarif")
}

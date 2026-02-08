// Package styles provides lipgloss styles for the TUI.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	Primary     = lipgloss.Color("#7C3AED") // Purple
	Secondary   = lipgloss.Color("#06B6D4") // Cyan
	Accent      = lipgloss.Color("#F59E0B") // Amber

	// Status colors
	Success     = lipgloss.Color("#10B981") // Green
	Warning     = lipgloss.Color("#F59E0B") // Amber
	Error       = lipgloss.Color("#EF4444") // Red
	Info        = lipgloss.Color("#3B82F6") // Blue

	// Neutral colors
	Text        = lipgloss.Color("#F9FAFB") // Light gray
	TextMuted   = lipgloss.Color("#9CA3AF") // Muted gray
	Background  = lipgloss.Color("#111827") // Dark background
	Surface     = lipgloss.Color("#1F2937") // Card background
	Border      = lipgloss.Color("#374151") // Border color
)

// Theme holds all the styles for the application.
type Theme struct {
	// Layout
	App           lipgloss.Style
	Header        lipgloss.Style
	Footer        lipgloss.Style
	Content       lipgloss.Style
	Sidebar       lipgloss.Style

	// Session list
	SessionItem         lipgloss.Style
	SessionItemSelected lipgloss.Style
	SessionItemActive   lipgloss.Style
	SessionName         lipgloss.Style
	SessionProfile      lipgloss.Style
	SessionRegion       lipgloss.Style
	SessionStatus       lipgloss.Style

	// Status badges
	StatusActive    lipgloss.Style
	StatusInactive  lipgloss.Style
	StatusExpiring  lipgloss.Style
	StatusError     lipgloss.Style
	StatusPending   lipgloss.Style

	// General UI
	Title           lipgloss.Style
	Subtitle        lipgloss.Style
	Label           lipgloss.Style
	Value           lipgloss.Style
	HelpKey         lipgloss.Style
	HelpDesc        lipgloss.Style
	ErrorText       lipgloss.Style
	SuccessText     lipgloss.Style
	WarningText     lipgloss.Style
	InfoText        lipgloss.Style

	// Input
	Input           lipgloss.Style
	InputFocused    lipgloss.Style
	InputPlaceholder lipgloss.Style

	// Buttons
	Button          lipgloss.Style
	ButtonFocused   lipgloss.Style
	ButtonDisabled  lipgloss.Style

	// Dialog
	Dialog          lipgloss.Style
	DialogTitle     lipgloss.Style
}

// DefaultTheme returns the default theme.
func DefaultTheme() *Theme {
	return &Theme{
		// Layout
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(Border).
			Padding(0, 1).
			MarginBottom(1),

		Footer: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(Border).
			Padding(0, 1).
			MarginTop(1).
			Foreground(TextMuted),

		Content: lipgloss.NewStyle().
			Padding(0, 1),

		Sidebar: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true).
			BorderForeground(Border).
			Padding(1, 2).
			Width(30),

		// Session list
		SessionItem: lipgloss.NewStyle().
			Padding(0, 2).
			MarginBottom(0),

		SessionItemSelected: lipgloss.NewStyle().
			Padding(0, 2).
			MarginBottom(0).
			Background(Surface).
			Foreground(Text).
			Bold(true),

		SessionItemActive: lipgloss.NewStyle().
			Padding(0, 2).
			MarginBottom(0).
			Foreground(Success),

		SessionName: lipgloss.NewStyle().
			Bold(true).
			Foreground(Text),

		SessionProfile: lipgloss.NewStyle().
			Foreground(TextMuted),

		SessionRegion: lipgloss.NewStyle().
			Foreground(Secondary),

		SessionStatus: lipgloss.NewStyle().
			Foreground(TextMuted),

		// Status badges
		StatusActive: lipgloss.NewStyle().
			Foreground(Success).
			Bold(true),

		StatusInactive: lipgloss.NewStyle().
			Foreground(TextMuted),

		StatusExpiring: lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true),

		StatusError: lipgloss.NewStyle().
			Foreground(Error).
			Bold(true),

		StatusPending: lipgloss.NewStyle().
			Foreground(Info).
			Bold(true),

		// General UI
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(TextMuted).
			MarginBottom(1),

		Label: lipgloss.NewStyle().
			Foreground(TextMuted).
			Width(15),

		Value: lipgloss.NewStyle().
			Foreground(Text),

		HelpKey: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(TextMuted),

		ErrorText: lipgloss.NewStyle().
			Foreground(Error),

		SuccessText: lipgloss.NewStyle().
			Foreground(Success),

		WarningText: lipgloss.NewStyle().
			Foreground(Warning),

		InfoText: lipgloss.NewStyle().
			Foreground(Info),

		// Input
		Input: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1),

		InputFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1),

		InputPlaceholder: lipgloss.NewStyle().
			Foreground(TextMuted),

		// Buttons
		Button: lipgloss.NewStyle().
			Padding(0, 2).
			Background(Surface).
			Foreground(Text),

		ButtonFocused: lipgloss.NewStyle().
			Padding(0, 2).
			Background(Primary).
			Foreground(Text).
			Bold(true),

		ButtonDisabled: lipgloss.NewStyle().
			Padding(0, 2).
			Background(Surface).
			Foreground(TextMuted),

		// Dialog
		Dialog: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2).
			Width(60),

		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1),
	}
}

// StatusStyle returns the appropriate style for a session status.
func (t *Theme) StatusStyle(status string) lipgloss.Style {
	switch status {
	case "active":
		return t.StatusActive
	case "expiring":
		return t.StatusExpiring
	case "error":
		return t.StatusError
	case "pending":
		return t.StatusPending
	default:
		return t.StatusInactive
	}
}

// StatusIcon returns an icon for a session status.
func StatusIcon(status string) string {
	switch status {
	case "active":
		return "●"
	case "expiring":
		return "◐"
	case "error":
		return "✗"
	case "pending":
		return "○"
	default:
		return "○"
	}
}

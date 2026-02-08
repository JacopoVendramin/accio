// Package components provides reusable TUI components.
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// KeyBinding represents a key binding with its description.
type KeyBinding struct {
	Key  string
	Desc string
}

// HelpBar renders a help bar with key bindings.
type HelpBar struct {
	bindings []KeyBinding
	theme    *styles.Theme
}

// NewHelpBar creates a new help bar.
func NewHelpBar(theme *styles.Theme) *HelpBar {
	return &HelpBar{
		theme: theme,
	}
}

// SetBindings sets the key bindings to display.
func (h *HelpBar) SetBindings(bindings []KeyBinding) {
	h.bindings = bindings
}

// View renders the help bar.
func (h *HelpBar) View() string {
	if len(h.bindings) == 0 {
		return ""
	}

	var parts []string
	for _, b := range h.bindings {
		key := h.theme.HelpKey.Render(b.Key)
		desc := h.theme.HelpDesc.Render(b.Desc)
		parts = append(parts, key+" "+desc)
	}

	return strings.Join(parts, "  ")
}

// DefaultSessionListBindings returns the default bindings for the session list.
func DefaultSessionListBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "start/stop"},
		{Key: "n", Desc: "new"},
		{Key: "d", Desc: "delete"},
		{Key: "r", Desc: "refresh"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

// DefaultDetailViewBindings returns the default bindings for the detail view.
func DefaultDetailViewBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "esc", Desc: "back"},
		{Key: "s", Desc: "start"},
		{Key: "x", Desc: "stop"},
		{Key: "e", Desc: "edit"},
		{Key: "c", Desc: "copy profile"},
		{Key: "q", Desc: "quit"},
	}
}

// Spinner renders a simple spinner animation.
type Spinner struct {
	frames  []string
	current int
}

// NewSpinner creates a new spinner.
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Next advances the spinner and returns the current frame.
func (s *Spinner) Next() string {
	frame := s.frames[s.current]
	s.current = (s.current + 1) % len(s.frames)
	return frame
}

// StatusBadge renders a status badge.
func StatusBadge(status string, theme *styles.Theme) string {
	icon := styles.StatusIcon(status)
	style := theme.StatusStyle(status)
	return style.Render(icon + " " + status)
}

// ProgressBar renders a simple progress bar.
func ProgressBar(percent float64, width int, theme *styles.Theme) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	filled := int(float64(width) * percent)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	var style lipgloss.Style
	if percent > 0.5 {
		style = theme.SuccessText
	} else if percent > 0.2 {
		style = theme.WarningText
	} else {
		style = theme.ErrorText
	}

	return style.Render(bar)
}

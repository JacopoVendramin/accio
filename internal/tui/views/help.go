package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// HelpKeyMap defines key bindings for the help view.
type HelpKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Back  key.Binding
}

// DefaultHelpKeyMap returns the default key bindings.
func DefaultHelpKeyMap() HelpKeyMap {
	return HelpKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "q", "?"),
			key.WithHelp("esc", "back"),
		),
	}
}

// HelpSection represents a section of help content.
type HelpSection struct {
	Title    string
	Bindings []HelpBinding
}

// HelpBinding represents a key binding with description.
type HelpBinding struct {
	Key  string
	Desc string
}

// HelpView shows help information.
type HelpView struct {
	sections []HelpSection
	scroll   int
	theme    *styles.Theme
	keyMap   HelpKeyMap
	width    int
	height   int

	// Callbacks
	onBack func() tea.Cmd
}

// NewHelpView creates a new help view.
func NewHelpView(theme *styles.Theme) *HelpView {
	return &HelpView{
		sections: defaultHelpSections(),
		theme:    theme,
		keyMap:   DefaultHelpKeyMap(),
	}
}

// defaultHelpSections returns the default help content.
func defaultHelpSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Session List",
			Bindings: []HelpBinding{
				{Key: "↑/↓ or j/k", Desc: "Navigate sessions"},
				{Key: "enter", Desc: "Start/Stop selected session"},
				{Key: "n", Desc: "Create new session"},
				{Key: "d", Desc: "Delete selected session"},
				{Key: "e", Desc: "Edit selected session"},
				{Key: "v", Desc: "View session details"},
				{Key: "r", Desc: "Refresh session list"},
				{Key: "s", Desc: "Open settings"},
				{Key: "?", Desc: "Show this help"},
				{Key: "q", Desc: "Quit application"},
			},
		},
		{
			Title: "Session Detail",
			Bindings: []HelpBinding{
				{Key: "s", Desc: "Start session"},
				{Key: "x", Desc: "Stop session"},
				{Key: "e", Desc: "Edit session"},
				{Key: "c", Desc: "Copy profile name to clipboard"},
				{Key: "esc", Desc: "Go back to list"},
			},
		},
		{
			Title: "Create Wizard",
			Bindings: []HelpBinding{
				{Key: "enter/tab", Desc: "Next step"},
				{Key: "shift+tab", Desc: "Previous step"},
				{Key: "↑/↓", Desc: "Select option"},
				{Key: "esc", Desc: "Cancel creation"},
			},
		},
		{
			Title: "Dialogs",
			Bindings: []HelpBinding{
				{Key: "enter", Desc: "Confirm/Submit"},
				{Key: "esc", Desc: "Cancel"},
				{Key: "←/→", Desc: "Switch buttons"},
				{Key: "y/n", Desc: "Quick confirm/cancel"},
			},
		},
		{
			Title: "Session Types",
			Bindings: []HelpBinding{
				{Key: "[IAM]", Desc: "IAM User with static credentials"},
				{Key: "[SSO]", Desc: "AWS SSO / Identity Center"},
				{Key: "[Role]", Desc: "IAM Role (assumed from another session)"},
				{Key: "[SAML]", Desc: "SAML Federation"},
			},
		},
		{
			Title: "Status Indicators",
			Bindings: []HelpBinding{
				{Key: "●", Desc: "Active - session is running"},
				{Key: "◐", Desc: "Expiring - credentials expire soon"},
				{Key: "○", Desc: "Inactive - session is stopped"},
				{Key: "✗", Desc: "Error - session has an error"},
			},
		},
		{
			Title: "AWS CLI Integration",
			Bindings: []HelpBinding{
				{Key: "credential_process", Desc: "AWS CLI calls accio for credentials"},
				{Key: "--profile", Desc: "Use: aws --profile <name> ..."},
			},
		},
	}
}

// SetSize sets the view size.
func (v *HelpView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnBack sets the callback for back action.
func (v *HelpView) SetOnBack(fn func() tea.Cmd) {
	v.onBack = fn
}

// Update handles input for the help view.
func (v *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Back):
			if v.onBack != nil {
				return v, v.onBack()
			}
		case key.Matches(msg, v.keyMap.Up):
			if v.scroll > 0 {
				v.scroll--
			}
		case key.Matches(msg, v.keyMap.Down):
			v.scroll++
		}
	}

	return v, nil
}

// View renders the help view.
func (v *HelpView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("My-Leapp Help"))
	b.WriteString("\n")
	b.WriteString(v.theme.Subtitle.Render("AWS Credentials Manager"))
	b.WriteString("\n\n")

	for _, section := range v.sections {
		b.WriteString(v.theme.SessionName.Render(section.Title))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", len(section.Title)))
		b.WriteString("\n")

		for _, binding := range section.Bindings {
			keyStr := v.theme.HelpKey.Render(padRight(binding.Key, 20))
			descStr := v.theme.HelpDesc.Render(binding.Desc)
			b.WriteString("  " + keyStr + descStr + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(v.theme.Footer.Render("Press esc or ? to close help"))

	return b.String()
}

// Init initializes the view.
func (v *HelpView) Init() tea.Cmd {
	return nil
}

// padRight pads a string to the specified width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

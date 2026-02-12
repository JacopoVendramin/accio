package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// AddMenuOption represents a menu option type.
type AddMenuOption int

const (
	AddMenuOptionSSOIntegration AddMenuOption = iota
	AddMenuOptionIAMUserSession
)

// AddMenuKeyMap defines key bindings for the add menu.
type AddMenuKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

// DefaultAddMenuKeyMap returns the default key bindings.
func DefaultAddMenuKeyMap() AddMenuKeyMap {
	return AddMenuKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "select"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// AddMenuView is a menu for selecting what to add.
type AddMenuView struct {
	theme  *styles.Theme
	keyMap AddMenuKeyMap
	width  int
	height int
	cursor int

	// Callbacks
	onSelect func(AddMenuOption) tea.Cmd
	onCancel func() tea.Cmd
}

// NewAddMenuView creates a new add menu view.
func NewAddMenuView(theme *styles.Theme) *AddMenuView {
	return &AddMenuView{
		theme:  theme,
		keyMap: DefaultAddMenuKeyMap(),
		cursor: 0,
	}
}

// SetSize sets the view size.
func (v *AddMenuView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSelect sets the callback for option selection.
func (v *AddMenuView) SetOnSelect(fn func(AddMenuOption) tea.Cmd) {
	v.onSelect = fn
}

// SetOnCancel sets the callback for cancellation.
func (v *AddMenuView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// Reset resets the menu to initial state.
func (v *AddMenuView) Reset() {
	v.cursor = 0
}

// Update handles input for the menu.
func (v *AddMenuView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		case key.Matches(msg, v.keyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keyMap.Down):
			if v.cursor < 1 { // We have 2 options (0 and 1)
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Select):
			if v.onSelect != nil {
				return v, v.onSelect(AddMenuOption(v.cursor))
			}
		}
	}
	return v, nil
}

// View renders the menu.
func (v *AddMenuView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("What would you like to add?"))
	b.WriteString("\n\n")

	options := []struct {
		title string
		desc  string
	}{
		{
			title: "AWS SSO Integration",
			desc:  "Connect to AWS Identity Center and sync multiple sessions",
		},
		{
			title: "IAM User Session",
			desc:  "Add a single session using access key credentials",
		},
	}

	for i, opt := range options {
		cursor := "  "
		titleStyle := v.theme.SessionItem
		descStyle := v.theme.Subtitle

		if i == v.cursor {
			cursor = "▶ "
			titleStyle = v.theme.SessionItemSelected
		}

		b.WriteString(titleStyle.Render(cursor + opt.title))
		b.WriteString("\n")
		b.WriteString(descStyle.Render("  " + opt.desc))
		b.WriteString("\n\n")
	}

	b.WriteString("\n")
	b.WriteString(v.theme.Footer.Render("↑/↓: navigate • enter: select • esc: cancel"))

	return b.String()
}

// Init initializes the view.
func (v *AddMenuView) Init() tea.Cmd {
	return nil
}

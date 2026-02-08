package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// MFADialogKeyMap defines key bindings for the MFA dialog.
type MFADialogKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
}

// DefaultMFADialogKeyMap returns the default key bindings.
func DefaultMFADialogKeyMap() MFADialogKeyMap {
	return MFADialogKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// MFADialogView shows a dialog for entering MFA token.
type MFADialogView struct {
	session  *session.Session
	input    textinput.Model
	theme    *styles.Theme
	keyMap   MFADialogKeyMap
	width    int
	height   int
	err      string

	// Callbacks
	onSubmit func(*session.Session, string) tea.Cmd
	onCancel func() tea.Cmd
}

// NewMFADialogView creates a new MFA dialog view.
func NewMFADialogView(theme *styles.Theme) *MFADialogView {
	ti := textinput.New()
	ti.Placeholder = "123456"
	ti.CharLimit = 6
	ti.Width = 20
	ti.Focus()

	return &MFADialogView{
		input:  ti,
		theme:  theme,
		keyMap: DefaultMFADialogKeyMap(),
	}
}

// SetSession sets the session requiring MFA.
func (v *MFADialogView) SetSession(sess *session.Session) {
	v.session = sess
	v.input.SetValue("")
	v.err = ""
	v.input.Focus()
}

// SetSize sets the view size.
func (v *MFADialogView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSubmit sets the callback for submit action.
func (v *MFADialogView) SetOnSubmit(fn func(*session.Session, string) tea.Cmd) {
	v.onSubmit = fn
}

// SetOnCancel sets the callback for cancel action.
func (v *MFADialogView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// SetError sets an error message.
func (v *MFADialogView) SetError(err string) {
	v.err = err
}

// Update handles input for the MFA dialog.
func (v *MFADialogView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		case key.Matches(msg, v.keyMap.Submit):
			token := v.input.Value()
			if len(token) != 6 {
				v.err = "MFA token must be 6 digits"
				return v, nil
			}
			if v.onSubmit != nil && v.session != nil {
				return v, v.onSubmit(v.session, token)
			}
		}
	}

	v.input, cmd = v.input.Update(msg)
	return v, cmd
}

// View renders the MFA dialog.
func (v *MFADialogView) View() string {
	var b strings.Builder

	// Dialog box
	b.WriteString(v.theme.DialogTitle.Render("MFA Token Required"))
	b.WriteString("\n\n")

	if v.session != nil {
		b.WriteString(v.theme.Label.Render("Session: "))
		b.WriteString(v.theme.Value.Render(v.session.Name))
		b.WriteString("\n")

		if v.session.Config.IAMUser != nil && v.session.Config.IAMUser.MFASerial != "" {
			b.WriteString(v.theme.Label.Render("MFA Device: "))
			b.WriteString(v.theme.Value.Render(v.session.Config.IAMUser.MFASerial))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(v.theme.Label.Render("Enter MFA Token:"))
	b.WriteString("\n")
	b.WriteString(v.input.View())

	if v.err != "" {
		b.WriteString("\n\n")
		b.WriteString(v.theme.ErrorText.Render(v.err))
	}

	b.WriteString("\n\n")
	b.WriteString(v.theme.Footer.Render("enter: submit • esc: cancel"))

	return v.theme.Dialog.Render(b.String())
}

// Init initializes the view.
func (v *MFADialogView) Init() tea.Cmd {
	return textinput.Blink
}

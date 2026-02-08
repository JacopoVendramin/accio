package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// ConfirmDialogKeyMap defines key bindings for the confirm dialog.
type ConfirmDialogKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
	Left    key.Binding
	Right   key.Binding
}

// DefaultConfirmDialogKeyMap returns the default key bindings.
func DefaultConfirmDialogKeyMap() ConfirmDialogKeyMap {
	return ConfirmDialogKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "n"),
			key.WithHelp("esc/n", "cancel"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
		),
	}
}

// ConfirmDialogView shows a confirmation dialog.
type ConfirmDialogView struct {
	title       string
	message     string
	confirmText string
	cancelText  string
	selected    int // 0 = cancel, 1 = confirm
	theme       *styles.Theme
	keyMap      ConfirmDialogKeyMap
	width       int
	height      int
	data        interface{} // arbitrary data to pass to callbacks

	// Callbacks
	onConfirm func(interface{}) tea.Cmd
	onCancel  func() tea.Cmd
}

// NewConfirmDialogView creates a new confirmation dialog view.
func NewConfirmDialogView(theme *styles.Theme) *ConfirmDialogView {
	return &ConfirmDialogView{
		title:       "Confirm",
		confirmText: "Yes",
		cancelText:  "No",
		selected:    0, // Cancel selected by default (safer)
		theme:       theme,
		keyMap:      DefaultConfirmDialogKeyMap(),
	}
}

// SetContent sets the dialog content.
func (v *ConfirmDialogView) SetContent(title, message string) {
	v.title = title
	v.message = message
	v.selected = 0
}

// SetButtons sets the button text.
func (v *ConfirmDialogView) SetButtons(confirmText, cancelText string) {
	v.confirmText = confirmText
	v.cancelText = cancelText
}

// SetData sets arbitrary data to pass to callbacks.
func (v *ConfirmDialogView) SetData(data interface{}) {
	v.data = data
}

// SetSize sets the view size.
func (v *ConfirmDialogView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnConfirm sets the callback for confirm action.
func (v *ConfirmDialogView) SetOnConfirm(fn func(interface{}) tea.Cmd) {
	v.onConfirm = fn
}

// SetOnCancel sets the callback for cancel action.
func (v *ConfirmDialogView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// Update handles input for the confirm dialog.
func (v *ConfirmDialogView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		case key.Matches(msg, v.keyMap.Confirm):
			if v.selected == 1 {
				if v.onConfirm != nil {
					return v, v.onConfirm(v.data)
				}
			} else {
				if v.onCancel != nil {
					return v, v.onCancel()
				}
			}
		case key.Matches(msg, v.keyMap.Left):
			v.selected = 0
		case key.Matches(msg, v.keyMap.Right):
			v.selected = 1
		case msg.String() == "y":
			if v.onConfirm != nil {
				return v, v.onConfirm(v.data)
			}
		case msg.String() == "n":
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		}
	}

	return v, nil
}

// View renders the confirm dialog.
func (v *ConfirmDialogView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.DialogTitle.Render(v.title))
	b.WriteString("\n\n")
	b.WriteString(v.theme.Value.Render(v.message))
	b.WriteString("\n\n")

	// Render buttons
	cancelStyle := v.theme.Button
	confirmStyle := v.theme.Button

	if v.selected == 0 {
		cancelStyle = v.theme.ButtonFocused
	} else {
		confirmStyle = v.theme.ButtonFocused
	}

	buttons := fmt.Sprintf("%s  %s",
		cancelStyle.Render(" "+v.cancelText+" "),
		confirmStyle.Render(" "+v.confirmText+" "),
	)
	b.WriteString(buttons)

	b.WriteString("\n\n")
	b.WriteString(v.theme.Footer.Render("←/→: select • enter: confirm • esc: cancel"))

	return v.theme.Dialog.Render(b.String())
}

// Init initializes the view.
func (v *ConfirmDialogView) Init() tea.Cmd {
	return nil
}

package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/tui/components"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// EditIntegrationField represents a field being edited.
type EditIntegrationField int

const (
	EditIntegrationFieldName EditIntegrationField = iota
	EditIntegrationFieldStartURL
	EditIntegrationFieldRegion
)

// EditIntegrationKeyMap defines key bindings for the edit integration view.
type EditIntegrationKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Edit   key.Binding
	Save   key.Binding
	Cancel key.Binding
}

// DefaultEditIntegrationKeyMap returns the default key bindings.
func DefaultEditIntegrationKeyMap() EditIntegrationKeyMap {
	return EditIntegrationKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Edit: key.NewBinding(
			key.WithKeys("enter", "e"),
			key.WithHelp("enter", "edit"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// EditableIntegrationField represents a field that can be edited.
type EditableIntegrationField struct {
	Field       EditIntegrationField
	Label       string
	Value       string
	Placeholder string
	Editable    bool
}

// EditIntegrationView shows a form for editing an integration.
type EditIntegrationView struct {
	integration *integration.Integration
	fields      []EditableIntegrationField
	cursor      int
	editing     bool
	input       textinput.Model
	helpBar     *components.HelpBar
	theme       *styles.Theme
	keyMap      EditIntegrationKeyMap
	width       int
	height      int
	changed     bool

	// Callbacks
	onSave   func(*integration.Integration) tea.Cmd
	onCancel func() tea.Cmd
}

// NewEditIntegrationView creates a new edit integration view.
func NewEditIntegrationView(theme *styles.Theme) *EditIntegrationView {
	ti := textinput.New()
	ti.CharLimit = 256

	helpBar := components.NewHelpBar(theme)
	helpBar.SetBindings([]components.KeyBinding{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "edit"},
		{Key: "ctrl+s", Desc: "save"},
		{Key: "esc", Desc: "cancel"},
	})

	return &EditIntegrationView{
		input:   ti,
		helpBar: helpBar,
		theme:   theme,
		keyMap:  DefaultEditIntegrationKeyMap(),
	}
}

// SetIntegration sets the integration to edit.
func (v *EditIntegrationView) SetIntegration(integ *integration.Integration) {
	v.integration = integ
	v.fields = v.buildFields(integ)
	v.cursor = 0
	v.editing = false
	v.changed = false
}

// buildFields creates editable fields for the integration.
func (v *EditIntegrationView) buildFields(integ *integration.Integration) []EditableIntegrationField {
	fields := []EditableIntegrationField{
		{
			Field:       EditIntegrationFieldName,
			Label:       "Name",
			Value:       integ.Name,
			Placeholder: "Integration name",
			Editable:    true,
		},
	}

	// Add type-specific fields
	if integ.Type == integration.IntegrationTypeAWSSSO && integ.Config.AWSSSO != nil {
		fields = append(fields,
			EditableIntegrationField{
				Field:       EditIntegrationFieldStartURL,
				Label:       "Start URL",
				Value:       integ.Config.AWSSSO.StartURL,
				Placeholder: "https://xxxxx.awsapps.com/start",
				Editable:    true,
			},
			EditableIntegrationField{
				Field:       EditIntegrationFieldRegion,
				Label:       "Region",
				Value:       integ.Config.AWSSSO.Region,
				Placeholder: "us-east-1",
				Editable:    true,
			},
		)
	}

	return fields
}

// SetSize sets the view size.
func (v *EditIntegrationView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSave sets the callback for save action.
func (v *EditIntegrationView) SetOnSave(fn func(*integration.Integration) tea.Cmd) {
	v.onSave = fn
}

// SetOnCancel sets the callback for cancel action.
func (v *EditIntegrationView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// Update handles input for the edit integration view.
func (v *EditIntegrationView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if v.editing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				v.editing = false
				v.input.Blur()
				return v, nil
			case "enter":
				// Save the edited value
				v.fields[v.cursor].Value = v.input.Value()
				v.changed = true
				v.editing = false
				v.input.Blur()
				return v, nil
			}
		}
		var cmd tea.Cmd
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}

	// Not editing - handle navigation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		case key.Matches(msg, v.keyMap.Save):
			if v.changed {
				v.applyChanges()
				if v.onSave != nil {
					return v, v.onSave(v.integration)
				}
			}
		case key.Matches(msg, v.keyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keyMap.Down):
			if v.cursor < len(v.fields)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Edit):
			if v.fields[v.cursor].Editable {
				v.editing = true
				v.input.SetValue(v.fields[v.cursor].Value)
				v.input.Placeholder = v.fields[v.cursor].Placeholder
				v.input.Focus()
				return v, textinput.Blink
			}
		}
	}

	return v, nil
}

// applyChanges applies the edited values to the integration.
func (v *EditIntegrationView) applyChanges() {
	for _, field := range v.fields {
		switch field.Field {
		case EditIntegrationFieldName:
			v.integration.Name = field.Value
		case EditIntegrationFieldStartURL:
			if v.integration.Config.AWSSSO != nil {
				v.integration.Config.AWSSSO.StartURL = field.Value
			}
		case EditIntegrationFieldRegion:
			if v.integration.Config.AWSSSO != nil {
				v.integration.Config.AWSSSO.Region = field.Value
			}
		}
	}
}

// View renders the edit integration view.
func (v *EditIntegrationView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Edit Integration"))
	b.WriteString("\n\n")

	if v.integration == nil {
		b.WriteString(v.theme.ErrorText.Render("No integration selected"))
		b.WriteString("\n\n")
		b.WriteString(v.theme.Footer.Render(v.helpBar.View()))
		return b.String()
	}

	// Show fields
	for i, field := range v.fields {
		cursor := "  "
		if i == v.cursor && !v.editing {
			cursor = "▶ "
		}

		label := v.theme.Label.Render(field.Label + ":")
		value := field.Value
		if value == "" {
			value = v.theme.Subtitle.Render("<empty>")
		} else {
			value = v.theme.Value.Render(value)
		}

		if v.editing && i == v.cursor {
			b.WriteString(cursor + label + "\n")
			b.WriteString("  " + v.input.View() + "\n\n")
		} else {
			b.WriteString(cursor + label + " " + value + "\n\n")
		}
	}

	// Status message
	if v.changed {
		b.WriteString(v.theme.WarningText.Render("⚠ Unsaved changes - Press Ctrl+S to save"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(v.theme.Footer.Render(v.helpBar.View()))

	return b.String()
}

// Init initializes the view.
func (v *EditIntegrationView) Init() tea.Cmd {
	return nil
}

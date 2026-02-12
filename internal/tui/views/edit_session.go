package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/tui/components"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// EditField represents a field being edited.
type EditField int

const (
	EditFieldName EditField = iota
	EditFieldProfile
	EditFieldRegion
	EditFieldMFASerial
	EditFieldSSOStartURL
	EditFieldSSOAccountID
	EditFieldSSORoleName
	EditFieldRoleARN
	EditFieldRoleExternalID
)

// EditSessionKeyMap defines key bindings for the edit view.
type EditSessionKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Edit   key.Binding
	Save   key.Binding
	Cancel key.Binding
}

// DefaultEditSessionKeyMap returns the default key bindings.
func DefaultEditSessionKeyMap() EditSessionKeyMap {
	return EditSessionKeyMap{
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

// EditableField represents a field that can be edited.
type EditableField struct {
	Field       EditField
	Label       string
	Value       string
	Placeholder string
	Editable    bool
}

// EditSessionView shows a form for editing a session.
type EditSessionView struct {
	session *session.Session
	fields  []EditableField
	cursor  int
	editing bool
	input   textinput.Model
	helpBar *components.HelpBar
	theme   *styles.Theme
	keyMap  EditSessionKeyMap
	width   int
	height  int
	changed bool

	// Callbacks
	onSave   func(*session.Session) tea.Cmd
	onCancel func() tea.Cmd
}

// NewEditSessionView creates a new edit session view.
func NewEditSessionView(theme *styles.Theme) *EditSessionView {
	ti := textinput.New()
	ti.CharLimit = 256

	helpBar := components.NewHelpBar(theme)
	helpBar.SetBindings([]components.KeyBinding{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "edit"},
		{Key: "ctrl+s", Desc: "save"},
		{Key: "esc", Desc: "cancel"},
	})

	return &EditSessionView{
		input:   ti,
		helpBar: helpBar,
		theme:   theme,
		keyMap:  DefaultEditSessionKeyMap(),
	}
}

// SetSession sets the session to edit.
func (v *EditSessionView) SetSession(sess *session.Session) {
	v.session = sess
	v.fields = v.buildFields(sess)
	v.cursor = 0
	v.editing = false
	v.changed = false
}

// buildFields creates editable fields for the session.
func (v *EditSessionView) buildFields(sess *session.Session) []EditableField {
	fields := []EditableField{
		{
			Field:       EditFieldName,
			Label:       "Name",
			Value:       sess.Name,
			Placeholder: "Session name",
			Editable:    true,
		},
		{
			Field:       EditFieldProfile,
			Label:       "Profile Name",
			Value:       sess.ProfileName,
			Placeholder: "AWS profile name",
			Editable:    true,
		},
		{
			Field:       EditFieldRegion,
			Label:       "Region",
			Value:       sess.Region,
			Placeholder: "AWS region",
			Editable:    true,
		},
	}

	// Add type-specific fields
	switch sess.Type {
	case session.SessionTypeIAMUser:
		if sess.Config.IAMUser != nil {
			fields = append(fields, EditableField{
				Field:       EditFieldMFASerial,
				Label:       "MFA Serial",
				Value:       sess.Config.IAMUser.MFASerial,
				Placeholder: "arn:aws:iam::123456789012:mfa/user (optional)",
				Editable:    true,
			})
		}
	case session.SessionTypeAWSSSO:
		if sess.Config.AWSSSO != nil {
			fields = append(fields,
				EditableField{
					Field:       EditFieldSSOStartURL,
					Label:       "SSO Start URL",
					Value:       sess.Config.AWSSSO.StartURL,
					Placeholder: "https://my-sso.awsapps.com/start",
					Editable:    true,
				},
				EditableField{
					Field:       EditFieldSSOAccountID,
					Label:       "Account ID",
					Value:       sess.Config.AWSSSO.AccountID,
					Placeholder: "123456789012",
					Editable:    true,
				},
				EditableField{
					Field:       EditFieldSSORoleName,
					Label:       "Role Name",
					Value:       sess.Config.AWSSSO.RoleName,
					Placeholder: "MyRole",
					Editable:    true,
				},
			)
		}
	case session.SessionTypeIAMRole:
		if sess.Config.IAMRole != nil {
			fields = append(fields,
				EditableField{
					Field:       EditFieldRoleARN,
					Label:       "Role ARN",
					Value:       sess.Config.IAMRole.RoleARN,
					Placeholder: "arn:aws:iam::123456789012:role/MyRole",
					Editable:    true,
				},
				EditableField{
					Field:       EditFieldRoleExternalID,
					Label:       "External ID",
					Value:       sess.Config.IAMRole.ExternalID,
					Placeholder: "external-id (optional)",
					Editable:    true,
				},
			)
		}
	}

	return fields
}

// SetSize sets the view size.
func (v *EditSessionView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSave sets the callback for save action.
func (v *EditSessionView) SetOnSave(fn func(*session.Session) tea.Cmd) {
	v.onSave = fn
}

// SetOnCancel sets the callback for cancel action.
func (v *EditSessionView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// Update handles input for the edit view.
func (v *EditSessionView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if v.editing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Save the edited value
				v.fields[v.cursor].Value = v.input.Value()
				v.editing = false
				v.changed = true
				return v, nil
			case "esc":
				v.editing = false
				return v, nil
			}
		}
		v.input, cmd = v.input.Update(msg)
		return v, cmd
	}

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
			if v.cursor < len(v.fields)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Edit):
			field := v.fields[v.cursor]
			if field.Editable {
				v.input.SetValue(field.Value)
				v.input.Placeholder = field.Placeholder
				v.input.Focus()
				v.editing = true
			}
		case key.Matches(msg, v.keyMap.Save):
			if v.changed {
				v.applyChanges()
				if v.onSave != nil {
					return v, v.onSave(v.session)
				}
			}
		}
	}

	return v, nil
}

// applyChanges applies the edited fields to the session.
func (v *EditSessionView) applyChanges() {
	for _, field := range v.fields {
		switch field.Field {
		case EditFieldName:
			v.session.Name = field.Value
		case EditFieldProfile:
			v.session.ProfileName = field.Value
		case EditFieldRegion:
			v.session.Region = field.Value
		case EditFieldMFASerial:
			if v.session.Config.IAMUser != nil {
				v.session.Config.IAMUser.MFASerial = field.Value
			}
		case EditFieldSSOStartURL:
			if v.session.Config.AWSSSO != nil {
				v.session.Config.AWSSSO.StartURL = field.Value
			}
		case EditFieldSSOAccountID:
			if v.session.Config.AWSSSO != nil {
				v.session.Config.AWSSSO.AccountID = field.Value
			}
		case EditFieldSSORoleName:
			if v.session.Config.AWSSSO != nil {
				v.session.Config.AWSSSO.RoleName = field.Value
			}
		case EditFieldRoleARN:
			if v.session.Config.IAMRole != nil {
				v.session.Config.IAMRole.RoleARN = field.Value
			}
		case EditFieldRoleExternalID:
			if v.session.Config.IAMRole != nil {
				v.session.Config.IAMRole.ExternalID = field.Value
			}
		}
	}
}

// View renders the edit session view.
func (v *EditSessionView) View() string {
	if v.session == nil {
		return v.theme.Subtitle.Render("No session to edit")
	}

	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Edit Session"))
	if v.changed {
		b.WriteString(v.theme.WarningText.Render(" (unsaved)"))
	}
	b.WriteString("\n")
	b.WriteString(v.theme.Subtitle.Render(formatSessionTypeFull(v.session.Type)))
	b.WriteString("\n\n")

	for i, field := range v.fields {
		isSelected := i == v.cursor

		// Label
		label := field.Label
		if isSelected {
			label = "▶ " + label
		} else {
			label = "  " + label
		}

		labelStyle := v.theme.Label
		if isSelected {
			labelStyle = v.theme.SessionItemSelected
		}
		b.WriteString(labelStyle.Render(label))
		b.WriteString("\n")

		// Value or input
		if isSelected && v.editing {
			b.WriteString("    ")
			b.WriteString(v.input.View())
		} else {
			value := field.Value
			if value == "" {
				value = v.theme.InputPlaceholder.Render(field.Placeholder)
			} else {
				value = v.theme.Value.Render(value)
			}
			b.WriteString("    " + value)
		}
		b.WriteString("\n\n")
	}

	// Help bar
	b.WriteString(v.theme.Footer.Render(v.helpBar.View()))

	return b.String()
}

// Init initializes the view.
func (v *EditSessionView) Init() tea.Cmd {
	return nil
}

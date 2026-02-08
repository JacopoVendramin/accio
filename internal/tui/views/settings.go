package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/config"
	"github.com/jvendramin/accio/internal/tui/components"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// SettingItem represents a setting that can be edited.
type SettingItem struct {
	Key         string
	Label       string
	Description string
	Value       string
	Type        string // "text", "bool", "select"
	Options     []string
}

// SettingsKeyMap defines key bindings for the settings view.
type SettingsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Edit   key.Binding
	Toggle key.Binding
	Save   key.Binding
	Back   key.Binding
}

// DefaultSettingsKeyMap returns the default key bindings.
func DefaultSettingsKeyMap() SettingsKeyMap {
	return SettingsKeyMap{
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
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "back"),
		),
	}
}

// SettingsView shows application settings.
type SettingsView struct {
	config   *config.Config
	settings []SettingItem
	cursor   int
	editing  bool
	input    textinput.Model
	helpBar  *components.HelpBar
	theme    *styles.Theme
	keyMap   SettingsKeyMap
	width    int
	height   int
	changed  bool

	// Callbacks
	onSave func(*config.Config) tea.Cmd
	onBack func() tea.Cmd
}

// NewSettingsView creates a new settings view.
func NewSettingsView(theme *styles.Theme) *SettingsView {
	ti := textinput.New()
	ti.CharLimit = 256

	helpBar := components.NewHelpBar(theme)
	helpBar.SetBindings([]components.KeyBinding{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "edit"},
		{Key: "space", Desc: "toggle"},
		{Key: "ctrl+s", Desc: "save"},
		{Key: "esc", Desc: "back"},
	})

	return &SettingsView{
		input:   ti,
		helpBar: helpBar,
		theme:   theme,
		keyMap:  DefaultSettingsKeyMap(),
	}
}

// SetConfig sets the configuration to display.
func (v *SettingsView) SetConfig(cfg *config.Config) {
	v.config = cfg
	v.settings = v.buildSettings(cfg)
	v.cursor = 0
	v.editing = false
	v.changed = false
}

// buildSettings creates setting items from the config.
func (v *SettingsView) buildSettings(cfg *config.Config) []SettingItem {
	return []SettingItem{
		{
			Key:         "default_region",
			Label:       "Default AWS Region",
			Description: "Default region for new sessions",
			Value:       cfg.DefaultRegion,
			Type:        "text",
		},
		{
			Key:         "default_session_duration",
			Label:       "Default Session Duration",
			Description: "Default duration for temporary credentials",
			Value:       cfg.DefaultSessionDuration.String(),
			Type:        "text",
		},
		{
			Key:         "refresh_before_expiry",
			Label:       "Refresh Before Expiry",
			Description: "Refresh credentials this long before expiry",
			Value:       cfg.RefreshBeforeExpiry.String(),
			Type:        "text",
		},
		{
			Key:         "clear_on_exit",
			Label:       "Clear Credentials on Exit",
			Description: "Clear cached credentials when exiting",
			Value:       fmt.Sprintf("%t", cfg.ClearOnExit),
			Type:        "bool",
		},
		{
			Key:         "use_credential_process",
			Label:       "Use credential_process",
			Description: "Use credential_process instead of writing credentials",
			Value:       fmt.Sprintf("%t", cfg.AWS.UseCredentialProcess),
			Type:        "bool",
		},
		{
			Key:         "show_timestamps",
			Label:       "Show Timestamps",
			Description: "Show timestamps in session list",
			Value:       fmt.Sprintf("%t", cfg.UI.ShowTimestamps),
			Type:        "bool",
		},
		{
			Key:         "show_region",
			Label:       "Show Region",
			Description: "Show region in session list",
			Value:       fmt.Sprintf("%t", cfg.UI.ShowRegion),
			Type:        "bool",
		},
		{
			Key:         "compact_mode",
			Label:       "Compact Mode",
			Description: "Use compact UI layout",
			Value:       fmt.Sprintf("%t", cfg.UI.CompactMode),
			Type:        "bool",
		},
		{
			Key:         "theme",
			Label:       "Theme",
			Description: "UI color theme",
			Value:       cfg.UI.Theme,
			Type:        "select",
			Options:     []string{"default", "light", "dark"},
		},
	}
}

// SetSize sets the view size.
func (v *SettingsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSave sets the callback for save action.
func (v *SettingsView) SetOnSave(fn func(*config.Config) tea.Cmd) {
	v.onSave = fn
}

// SetOnBack sets the callback for back action.
func (v *SettingsView) SetOnBack(fn func() tea.Cmd) {
	v.onBack = fn
}

// Update handles input for the settings view.
func (v *SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if v.editing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Save the edited value
				v.settings[v.cursor].Value = v.input.Value()
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
		case key.Matches(msg, v.keyMap.Back):
			if v.onBack != nil {
				return v, v.onBack()
			}
		case key.Matches(msg, v.keyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keyMap.Down):
			if v.cursor < len(v.settings)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Edit):
			setting := &v.settings[v.cursor]
			if setting.Type == "bool" {
				// Toggle boolean
				if setting.Value == "true" {
					setting.Value = "false"
				} else {
					setting.Value = "true"
				}
				v.changed = true
			} else if setting.Type == "select" {
				// Cycle through options
				for i, opt := range setting.Options {
					if opt == setting.Value {
						setting.Value = setting.Options[(i+1)%len(setting.Options)]
						break
					}
				}
				v.changed = true
			} else {
				// Start editing
				v.input.SetValue(setting.Value)
				v.input.Focus()
				v.editing = true
			}
		case key.Matches(msg, v.keyMap.Toggle):
			setting := &v.settings[v.cursor]
			if setting.Type == "bool" {
				if setting.Value == "true" {
					setting.Value = "false"
				} else {
					setting.Value = "true"
				}
				v.changed = true
			}
		case key.Matches(msg, v.keyMap.Save):
			if v.changed && v.onSave != nil {
				v.applySettings()
				return v, v.onSave(v.config)
			}
		}
	}

	return v, nil
}

// applySettings applies the current settings to the config.
func (v *SettingsView) applySettings() {
	for _, s := range v.settings {
		switch s.Key {
		case "default_region":
			v.config.DefaultRegion = s.Value
		case "clear_on_exit":
			v.config.ClearOnExit = s.Value == "true"
		case "use_credential_process":
			v.config.AWS.UseCredentialProcess = s.Value == "true"
		case "show_timestamps":
			v.config.UI.ShowTimestamps = s.Value == "true"
		case "show_region":
			v.config.UI.ShowRegion = s.Value == "true"
		case "compact_mode":
			v.config.UI.CompactMode = s.Value == "true"
		case "theme":
			v.config.UI.Theme = s.Value
		}
	}
}

// View renders the settings view.
func (v *SettingsView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Settings"))
	if v.changed {
		b.WriteString(v.theme.WarningText.Render(" (unsaved)"))
	}
	b.WriteString("\n\n")

	for i, setting := range v.settings {
		isSelected := i == v.cursor

		// Label
		label := setting.Label
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

		// Value
		valueStr := setting.Value
		if setting.Type == "bool" {
			if valueStr == "true" {
				valueStr = v.theme.SuccessText.Render("● Enabled")
			} else {
				valueStr = v.theme.SessionStatus.Render("○ Disabled")
			}
		} else {
			valueStr = v.theme.Value.Render(valueStr)
		}

		if isSelected && v.editing {
			b.WriteString("    ")
			b.WriteString(v.input.View())
		} else {
			b.WriteString("    ")
			b.WriteString(valueStr)
		}
		b.WriteString("\n")

		// Description
		if isSelected {
			b.WriteString("    ")
			b.WriteString(v.theme.Subtitle.Render(setting.Description))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Help bar
	b.WriteString(v.theme.Footer.Render(v.helpBar.View()))

	return b.String()
}

// Init initializes the view.
func (v *SettingsView) Init() tea.Cmd {
	return nil
}

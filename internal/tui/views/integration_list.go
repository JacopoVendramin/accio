package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/tui/components"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// IntegrationListKeyMap defines key bindings for the integration list.
type IntegrationListKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Sync   key.Binding
	Add    key.Binding
	Delete key.Binding
	Back   key.Binding
	Quit   key.Binding
}

// DefaultIntegrationListKeyMap returns the default key bindings.
func DefaultIntegrationListKeyMap() IntegrationListKeyMap {
	return IntegrationListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Sync: key.NewBinding(
			key.WithKeys("enter", "s"),
			key.WithHelp("enter/s", "sync"),
		),
		Add: key.NewBinding(
			key.WithKeys("a", "n"),
			key.WithHelp("a", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

// IntegrationListView shows a list of integrations.
type IntegrationListView struct {
	integrations []*integration.Integration
	cursor       int
	helpBar      *components.HelpBar
	theme        *styles.Theme
	keyMap       IntegrationListKeyMap
	width        int
	height       int

	// Callbacks
	onSync   func(*integration.Integration) tea.Cmd
	onAdd    func() tea.Cmd
	onDelete func(*integration.Integration) tea.Cmd
	onBack   func() tea.Cmd
}

// NewIntegrationListView creates a new integration list view.
func NewIntegrationListView(theme *styles.Theme) *IntegrationListView {
	helpBar := components.NewHelpBar(theme)
	helpBar.SetBindings([]components.KeyBinding{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "sync"},
		{Key: "a", Desc: "add"},
		{Key: "e", Desc: "edit"},
		{Key: "d", Desc: "delete"},
		{Key: "esc", Desc: "back"},
	})

	return &IntegrationListView{
		helpBar: helpBar,
		theme:   theme,
		keyMap:  DefaultIntegrationListKeyMap(),
	}
}

// SetIntegrations sets the integrations to display.
func (v *IntegrationListView) SetIntegrations(integrations []*integration.Integration) {
	v.integrations = integrations
	if v.cursor >= len(integrations) {
		v.cursor = max(0, len(integrations)-1)
	}
}

// SetSize sets the view size.
func (v *IntegrationListView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSync sets the callback for sync action.
func (v *IntegrationListView) SetOnSync(fn func(*integration.Integration) tea.Cmd) {
	v.onSync = fn
}

// SetOnAdd sets the callback for add action.
func (v *IntegrationListView) SetOnAdd(fn func() tea.Cmd) {
	v.onAdd = fn
}

// SetOnDelete sets the callback for delete action.
func (v *IntegrationListView) SetOnDelete(fn func(*integration.Integration) tea.Cmd) {
	v.onDelete = fn
}

// SetOnBack sets the callback for back action.
func (v *IntegrationListView) SetOnBack(fn func() tea.Cmd) {
	v.onBack = fn
}

// Selected returns the currently selected integration.
func (v *IntegrationListView) Selected() *integration.Integration {
	if v.cursor < 0 || v.cursor >= len(v.integrations) {
		return nil
	}
	return v.integrations[v.cursor]
}

// Update handles input for the integration list.
func (v *IntegrationListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if v.cursor < len(v.integrations)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Sync):
			if v.onSync != nil && v.Selected() != nil {
				return v, v.onSync(v.Selected())
			}
		case key.Matches(msg, v.keyMap.Add):
			if v.onAdd != nil {
				return v, v.onAdd()
			}
		case key.Matches(msg, v.keyMap.Delete):
			if v.onDelete != nil && v.Selected() != nil {
				return v, v.onDelete(v.Selected())
			}
		case key.Matches(msg, v.keyMap.Quit):
			return v, tea.Quit
		}
	}

	return v, nil
}

// View renders the integration list.
func (v *IntegrationListView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Integrations"))
	b.WriteString("\n")
	b.WriteString(v.theme.Subtitle.Render("Manage your SSO and identity provider connections"))
	b.WriteString("\n\n")

	if len(v.integrations) == 0 {
		b.WriteString(v.theme.Subtitle.Render("No integrations configured."))
		b.WriteString("\n")
		b.WriteString(v.theme.InfoText.Render("Press 'a' to add an integration."))
	} else {
		for i, integ := range v.integrations {
			isSelected := i == v.cursor
			b.WriteString(v.renderIntegration(integ, isSelected))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(v.theme.Footer.Render(v.helpBar.View()))

	return b.String()
}

func (v *IntegrationListView) renderIntegration(integ *integration.Integration, selected bool) string {
	var b strings.Builder

	// Cursor and icon
	cursor := "  "
	if selected {
		cursor = "▶ "
	}

	// Type icon
	icon := "🔗"
	if integ.Type == integration.IntegrationTypeAWSSSO {
		icon = "☁️"
	}

	// Status indicator
	status := "○"
	statusStyle := v.theme.StatusInactive
	if integ.IsTokenValid() {
		status = "●"
		statusStyle = v.theme.StatusActive
	}

	// Name and type
	style := v.theme.SessionItem
	if selected {
		style = v.theme.SessionItemSelected
	}

	line := fmt.Sprintf("%s%s %s %s", cursor, icon, statusStyle.Render(status), integ.Name)
	b.WriteString(style.Render(line))
	b.WriteString("\n")

	// Details
	if integ.Config.AWSSSO != nil {
		b.WriteString(v.theme.Subtitle.Render(fmt.Sprintf("      %s • %s",
			integ.Config.AWSSSO.StartURL,
			integ.Config.AWSSSO.Region,
		)))
		b.WriteString("\n")

		if !integ.Metadata.LastSyncedAt.IsZero() {
			ago := time.Since(integ.Metadata.LastSyncedAt)
			b.WriteString(v.theme.Subtitle.Render(fmt.Sprintf("      Last synced: %s ago", formatDurationAgo(ago))))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func formatDurationAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// Init initializes the view.
func (v *IntegrationListView) Init() tea.Cmd {
	return nil
}

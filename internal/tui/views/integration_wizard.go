package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// IntegrationWizardStep represents a step in the integration wizard.
type IntegrationWizardStep int

const (
	IntegrationStepType IntegrationWizardStep = iota
	IntegrationStepAlias
	IntegrationStepPortalURL
	IntegrationStepRegion
	IntegrationStepAuthMethod
	IntegrationStepConfirm
)

// AuthMethod represents the authentication method.
type AuthMethod string

const (
	AuthMethodInBrowser AuthMethod = "in_browser"
	AuthMethodInApp     AuthMethod = "in_app"
)

// IntegrationWizardKeyMap defines key bindings for the wizard.
type IntegrationWizardKeyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Up     key.Binding
	Down   key.Binding
	Cancel key.Binding
}

// DefaultIntegrationWizardKeyMap returns the default key bindings.
func DefaultIntegrationWizardKeyMap() IntegrationWizardKeyMap {
	return IntegrationWizardKeyMap{
		Next: key.NewBinding(
			key.WithKeys("enter", "tab"),
			key.WithHelp("enter", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "back"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓", "down"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// IntegrationWizardView is a wizard for creating new integrations.
type IntegrationWizardView struct {
	step   IntegrationWizardStep
	theme  *styles.Theme
	keyMap IntegrationWizardKeyMap
	width  int
	height int

	// Selection cursors
	typeCursor   int
	regionCursor int
	authCursor   int

	// Input fields
	aliasInput     textinput.Model
	portalURLInput textinput.Model
	regionInput    textinput.Model

	// Region search state
	filteredRegions []string
	useCustomRegion bool

	// Selected values
	selectedType   integration.IntegrationType
	selectedRegion string
	selectedAuth   AuthMethod

	// Callbacks
	onCreate func(*integration.Integration) tea.Cmd
	onCancel func() tea.Cmd
}

// NewIntegrationWizardView creates a new integration wizard view.
func NewIntegrationWizardView(theme *styles.Theme) *IntegrationWizardView {
	aliasInput := textinput.New()
	aliasInput.Placeholder = "My Company SSO"
	aliasInput.CharLimit = 100

	portalURLInput := textinput.New()
	portalURLInput.Placeholder = "https://d-xxxxxxxxxx.awsapps.com/start"
	portalURLInput.CharLimit = 256

	regionInput := textinput.New()
	regionInput.Placeholder = "Search or type custom region..."
	regionInput.CharLimit = 50

	return &IntegrationWizardView{
		step:            IntegrationStepType,
		theme:           theme,
		keyMap:          DefaultIntegrationWizardKeyMap(),
		aliasInput:      aliasInput,
		portalURLInput:  portalURLInput,
		regionInput:     regionInput,
		filteredRegions: AWSRegions,
		selectedType:    integration.IntegrationTypeAWSSSO,
		selectedRegion:  "us-east-1",
		selectedAuth:    AuthMethodInBrowser,
	}
}

// SetSize sets the view size.
func (v *IntegrationWizardView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnCreate sets the callback for integration creation.
func (v *IntegrationWizardView) SetOnCreate(fn func(*integration.Integration) tea.Cmd) {
	v.onCreate = fn
}

// SetOnCancel sets the callback for cancellation.
func (v *IntegrationWizardView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// Reset resets the wizard to initial state.
func (v *IntegrationWizardView) Reset() {
	v.step = IntegrationStepType
	v.typeCursor = 0
	v.regionCursor = 0
	v.authCursor = 0
	v.aliasInput.SetValue("")
	v.portalURLInput.SetValue("")
	v.regionInput.SetValue("")
	v.filteredRegions = AWSRegions
	v.useCustomRegion = false
	v.selectedType = integration.IntegrationTypeAWSSSO
	v.selectedRegion = "us-east-1"
	v.selectedAuth = AuthMethodInBrowser
}

// Update handles input for the wizard.
func (v *IntegrationWizardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle region step specially for search
		if v.step == IntegrationStepRegion {
			switch msg.String() {
			case "esc":
				if v.onCancel != nil {
					return v, v.onCancel()
				}
			case "enter":
				return v, v.nextStep()
			case "shift+tab":
				v.prevStep()
				return v, nil
			case "up", "ctrl+p":
				if !v.useCustomRegion && v.regionCursor > 0 {
					v.regionCursor--
				}
				return v, nil
			case "down", "ctrl+n":
				if !v.useCustomRegion && v.regionCursor < len(v.filteredRegions)-1 {
					v.regionCursor++
				}
				return v, nil
			case "tab":
				// Toggle between search results and custom input mode
				v.useCustomRegion = !v.useCustomRegion
				return v, nil
			default:
				// Update region input and filter
				v.regionInput, cmd = v.regionInput.Update(msg)
				v.filterRegions()
				return v, cmd
			}
		}

		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		case key.Matches(msg, v.keyMap.Next):
			return v, v.nextStep()
		case key.Matches(msg, v.keyMap.Prev):
			v.prevStep()
			return v, nil
		case key.Matches(msg, v.keyMap.Up):
			v.handleUp()
			return v, nil
		case key.Matches(msg, v.keyMap.Down):
			v.handleDown()
			return v, nil
		}
	}

	// Update text inputs
	switch v.step {
	case IntegrationStepAlias:
		v.aliasInput, cmd = v.aliasInput.Update(msg)
	case IntegrationStepPortalURL:
		v.portalURLInput, cmd = v.portalURLInput.Update(msg)
	}

	return v, cmd
}

func (v *IntegrationWizardView) filterRegions() {
	query := strings.ToLower(strings.TrimSpace(v.regionInput.Value()))
	if query == "" {
		v.filteredRegions = AWSRegions
		v.regionCursor = 0
		return
	}

	var filtered []string
	for _, region := range AWSRegions {
		if strings.Contains(strings.ToLower(region), query) {
			filtered = append(filtered, region)
		}
	}
	v.filteredRegions = filtered
	v.regionCursor = 0
}

func (v *IntegrationWizardView) handleUp() {
	switch v.step {
	case IntegrationStepType:
		if v.typeCursor > 0 {
			v.typeCursor--
		}
	case IntegrationStepRegion:
		if v.regionCursor > 0 {
			v.regionCursor--
		}
	case IntegrationStepAuthMethod:
		if v.authCursor > 0 {
			v.authCursor--
		}
	}
}

func (v *IntegrationWizardView) handleDown() {
	switch v.step {
	case IntegrationStepType:
		if v.typeCursor < 0 { // Only AWS SSO for now
			v.typeCursor++
		}
	case IntegrationStepRegion:
		if v.regionCursor < len(v.filteredRegions)-1 {
			v.regionCursor++
		}
	case IntegrationStepAuthMethod:
		if v.authCursor < 1 {
			v.authCursor++
		}
	}
}

func (v *IntegrationWizardView) nextStep() tea.Cmd {
	switch v.step {
	case IntegrationStepType:
		v.selectedType = integration.IntegrationTypeAWSSSO
		v.step = IntegrationStepAlias
		v.aliasInput.Focus()

	case IntegrationStepAlias:
		if strings.TrimSpace(v.aliasInput.Value()) == "" {
			return nil // Don't proceed without alias
		}
		v.aliasInput.Blur()
		v.step = IntegrationStepPortalURL
		v.portalURLInput.Focus()

	case IntegrationStepPortalURL:
		if strings.TrimSpace(v.portalURLInput.Value()) == "" {
			return nil // Don't proceed without URL
		}
		v.portalURLInput.Blur()
		v.step = IntegrationStepRegion
		v.regionInput.Focus()
		v.filteredRegions = AWSRegions
		v.regionCursor = 0
		v.useCustomRegion = false

	case IntegrationStepRegion:
		// Use custom region if in custom mode or if there's input that doesn't match any region
		inputValue := strings.TrimSpace(v.regionInput.Value())
		if v.useCustomRegion && inputValue != "" {
			v.selectedRegion = inputValue
		} else if len(v.filteredRegions) > 0 && v.regionCursor < len(v.filteredRegions) {
			v.selectedRegion = v.filteredRegions[v.regionCursor]
		} else if inputValue != "" {
			// No matches but there's input - use as custom
			v.selectedRegion = inputValue
		} else {
			return nil // Don't proceed without a region
		}
		v.regionInput.Blur()
		v.step = IntegrationStepAuthMethod

	case IntegrationStepAuthMethod:
		if v.authCursor == 0 {
			v.selectedAuth = AuthMethodInBrowser
		} else {
			v.selectedAuth = AuthMethodInApp
		}
		v.step = IntegrationStepConfirm

	case IntegrationStepConfirm:
		return v.createIntegration()
	}

	return nil
}

func (v *IntegrationWizardView) prevStep() {
	switch v.step {
	case IntegrationStepAlias:
		v.aliasInput.Blur()
		v.step = IntegrationStepType
	case IntegrationStepPortalURL:
		v.portalURLInput.Blur()
		v.step = IntegrationStepAlias
		v.aliasInput.Focus()
	case IntegrationStepRegion:
		v.regionInput.Blur()
		v.step = IntegrationStepPortalURL
		v.portalURLInput.Focus()
	case IntegrationStepAuthMethod:
		v.step = IntegrationStepRegion
		v.regionInput.Focus()
	case IntegrationStepConfirm:
		v.step = IntegrationStepAuthMethod
	}
}

func (v *IntegrationWizardView) createIntegration() tea.Cmd {
	integ := integration.NewAWSSSOIntegration(
		v.aliasInput.Value(),
		v.portalURLInput.Value(),
		v.selectedRegion,
	)

	if v.onCreate != nil {
		return v.onCreate(integ)
	}
	return nil
}

// View renders the wizard.
func (v *IntegrationWizardView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Add Integration"))
	b.WriteString("\n\n")

	switch v.step {
	case IntegrationStepType:
		b.WriteString(v.theme.Subtitle.Render("Select integration type:"))
		b.WriteString("\n\n")

		types := []struct {
			name string
			desc string
		}{
			{"AWS Single Sign-On", "Connect to AWS Identity Center (SSO)"},
			// Future: {"SAML 2.0", "Connect via SAML identity provider"},
		}

		for i, t := range types {
			cursor := "  "
			style := v.theme.SessionItem
			if i == v.typeCursor {
				cursor = "▶ "
				style = v.theme.SessionItemSelected
			}
			b.WriteString(style.Render(cursor + t.name))
			b.WriteString("\n")
			b.WriteString(v.theme.Subtitle.Render("    " + t.desc))
			b.WriteString("\n\n")
		}

	case IntegrationStepAlias:
		b.WriteString(v.theme.Label.Render("Alias:"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("A friendly name to identify this integration"))
		b.WriteString("\n\n")
		b.WriteString(v.aliasInput.View())

	case IntegrationStepPortalURL:
		b.WriteString(v.theme.Label.Render("Portal URL:"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("Your AWS SSO start URL (e.g., https://d-xxxxxxxxxx.awsapps.com/start)"))
		b.WriteString("\n\n")
		b.WriteString(v.portalURLInput.View())

	case IntegrationStepRegion:
		b.WriteString(v.theme.Label.Render("AWS Region:"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("Search or type a custom region"))
		b.WriteString("\n\n")

		// Search input
		b.WriteString(v.regionInput.View())
		b.WriteString("\n\n")

		if v.useCustomRegion {
			// Show custom region mode
			b.WriteString(v.theme.InfoText.Render("Using custom region: "))
			inputVal := v.regionInput.Value()
			if inputVal == "" {
				b.WriteString(v.theme.Subtitle.Render("(type region name above)"))
			} else {
				b.WriteString(v.theme.Value.Render(inputVal))
			}
			b.WriteString("\n\n")
			b.WriteString(v.theme.Subtitle.Render("Press Tab to switch back to region list"))
		} else if len(v.filteredRegions) == 0 {
			// No matches - suggest custom
			b.WriteString(v.theme.Subtitle.Render("No matching regions found."))
			b.WriteString("\n")
			b.WriteString(v.theme.InfoText.Render("Press Tab to use '"))
			b.WriteString(v.theme.Value.Render(v.regionInput.Value()))
			b.WriteString(v.theme.InfoText.Render("' as custom region"))
		} else {
			// Show filtered regions
			start := v.regionCursor - 3
			if start < 0 {
				start = 0
			}
			end := start + 7
			if end > len(v.filteredRegions) {
				end = len(v.filteredRegions)
				start = end - 7
				if start < 0 {
					start = 0
				}
			}

			for i := start; i < end; i++ {
				region := v.filteredRegions[i]
				cursor := "  "
				style := v.theme.SessionItem
				if i == v.regionCursor {
					cursor = "▶ "
					style = v.theme.SessionItemSelected
				}
				b.WriteString(style.Render(cursor + region))
				b.WriteString("\n")
			}

			if len(v.filteredRegions) < len(AWSRegions) {
				b.WriteString(v.theme.Subtitle.Render("  "))
				b.WriteString(v.theme.Subtitle.Render(formatMatches(len(v.filteredRegions))))
				b.WriteString("\n")
			}

			b.WriteString("\n")
			b.WriteString(v.theme.Subtitle.Render("Press Tab to enter custom region"))
		}

	case IntegrationStepAuthMethod:
		b.WriteString(v.theme.Label.Render("Authentication Method:"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("How do you want to authenticate?"))
		b.WriteString("\n\n")

		methods := []struct {
			name string
			desc string
		}{
			{"In Browser (Recommended)", "Opens your default browser for authentication"},
			{"In App", "Authenticate within the terminal"},
		}

		for i, m := range methods {
			cursor := "  "
			style := v.theme.SessionItem
			if i == v.authCursor {
				cursor = "▶ "
				style = v.theme.SessionItemSelected
			}
			b.WriteString(style.Render(cursor + m.name))
			b.WriteString("\n")
			b.WriteString(v.theme.Subtitle.Render("    " + m.desc))
			b.WriteString("\n\n")
		}

	case IntegrationStepConfirm:
		b.WriteString(v.theme.Subtitle.Render("Review your integration:"))
		b.WriteString("\n\n")

		b.WriteString(v.theme.Label.Render("Type:       "))
		b.WriteString(v.theme.Value.Render("AWS Single Sign-On"))
		b.WriteString("\n")
		b.WriteString(v.theme.Label.Render("Alias:      "))
		b.WriteString(v.theme.Value.Render(v.aliasInput.Value()))
		b.WriteString("\n")
		b.WriteString(v.theme.Label.Render("Portal URL: "))
		b.WriteString(v.theme.Value.Render(v.portalURLInput.Value()))
		b.WriteString("\n")
		b.WriteString(v.theme.Label.Render("Region:     "))
		b.WriteString(v.theme.Value.Render(v.selectedRegion))
		b.WriteString("\n")
		b.WriteString(v.theme.Label.Render("Auth:       "))
		authLabel := "In Browser"
		if v.selectedAuth == AuthMethodInApp {
			authLabel = "In App"
		}
		b.WriteString(v.theme.Value.Render(authLabel))
		b.WriteString("\n\n")

		b.WriteString(v.theme.InfoText.Render("Press Enter to create, Esc to cancel"))
	}

	b.WriteString("\n\n")
	if v.step == IntegrationStepRegion {
		b.WriteString(v.theme.Footer.Render("enter: next • ↑/↓: select • tab: custom • shift+tab: back • esc: cancel"))
	} else {
		b.WriteString(v.theme.Footer.Render("enter: next • shift+tab: back • esc: cancel"))
	}

	return b.String()
}

// Init initializes the view.
func (v *IntegrationWizardView) Init() tea.Cmd {
	return textinput.Blink
}

func formatMatches(n int) string {
	if n == 1 {
		return "(1 match)"
	}
	return "(" + itoa(n) + " matches)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

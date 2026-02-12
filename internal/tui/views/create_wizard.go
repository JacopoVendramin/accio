package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// WizardStep represents a step in the create wizard.
type WizardStep int

const (
	StepSelectType WizardStep = iota
	StepSessionName
	StepProfileName
	StepRegion
	StepIAMAccessKey
	StepIAMSecretKey
	StepIAMMFASerial
	StepSSOStartURL
	StepSSOAccountID
	StepSSORoleName
	StepRoleARN
	StepRoleParent
	StepRoleExternalID
	StepConfirm
)

// CreateWizardKeyMap defines key bindings for the wizard.
type CreateWizardKeyMap struct {
	Next   key.Binding
	Prev   key.Binding
	Select key.Binding
	Cancel key.Binding
}

// DefaultCreateWizardKeyMap returns the default key bindings.
func DefaultCreateWizardKeyMap() CreateWizardKeyMap {
	return CreateWizardKeyMap{
		Next: key.NewBinding(
			key.WithKeys("enter", "tab"),
			key.WithHelp("enter", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "back"),
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

// CreateWizardView is a wizard for creating new sessions.
type CreateWizardView struct {
	step        WizardStep
	sessionType session.SessionType
	theme       *styles.Theme
	keyMap      CreateWizardKeyMap
	width       int
	height      int

	// Input fields
	nameInput       textinput.Model
	profileInput    textinput.Model
	regionInput     textinput.Model
	accessKeyInput  textinput.Model
	secretKeyInput  textinput.Model
	mfaSerialInput  textinput.Model
	ssoURLInput     textinput.Model
	accountIDInput  textinput.Model
	roleNameInput   textinput.Model
	roleARNInput    textinput.Model
	externalIDInput textinput.Model

	// Region search state
	filteredRegions []string
	regionCursor    int
	useCustomRegion bool

	// Parent session selector
	parentSessions []*session.Session
	parentCursor   int

	// Type selector
	typeCursor int

	// Callbacks
	onCreate func(*session.Session, string) tea.Cmd // session and secret key
	onCancel func() tea.Cmd
}

// NewCreateWizardView creates a new create wizard view.
func NewCreateWizardView(theme *styles.Theme) *CreateWizardView {
	// Create text inputs
	makeInput := func(placeholder string, isPassword bool) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 256
		if isPassword {
			ti.EchoMode = textinput.EchoPassword
		}
		return ti
	}

	v := &CreateWizardView{
		step:            StepSelectType,
		theme:           theme,
		keyMap:          DefaultCreateWizardKeyMap(),
		nameInput:       makeInput("My AWS Account", false),
		profileInput:    makeInput("my-aws-profile", false),
		regionInput:     makeInput("Search or type custom region...", false),
		accessKeyInput:  makeInput("AKIA...", false),
		secretKeyInput:  makeInput("secret key", true),
		mfaSerialInput:  makeInput("arn:aws:iam::123456789012:mfa/user (optional)", false),
		ssoURLInput:     makeInput("https://my-sso-portal.awsapps.com/start", false),
		accountIDInput:  makeInput("123456789012", false),
		roleNameInput:   makeInput("MyRole", false),
		roleARNInput:    makeInput("arn:aws:iam::123456789012:role/MyRole", false),
		externalIDInput: makeInput("external-id (optional)", false),
		filteredRegions: AWSRegions,
		regionCursor:    0,
		useCustomRegion: false,
	}

	v.nameInput.Focus()

	return v
}

// SetSize sets the view size.
func (v *CreateWizardView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetParentSessions sets available parent sessions for role chaining.
func (v *CreateWizardView) SetParentSessions(sessions []*session.Session) {
	v.parentSessions = sessions
}

// SetOnCreate sets the callback for session creation.
func (v *CreateWizardView) SetOnCreate(fn func(*session.Session, string) tea.Cmd) {
	v.onCreate = fn
}

// SetOnCancel sets the callback for cancellation.
func (v *CreateWizardView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// filterRegions filters regions based on the current input.
func (v *CreateWizardView) filterRegions() {
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
	if v.regionCursor >= len(v.filteredRegions) {
		v.regionCursor = len(v.filteredRegions) - 1
	}
	if v.regionCursor < 0 {
		v.regionCursor = 0
	}
}

// Reset resets the wizard to the initial state.
func (v *CreateWizardView) Reset() {
	v.step = StepSelectType
	v.typeCursor = 0
	v.parentCursor = 0
	v.regionCursor = 0
	v.useCustomRegion = false
	v.filteredRegions = AWSRegions
	v.nameInput.SetValue("")
	v.profileInput.SetValue("")
	v.regionInput.SetValue("")
	v.accessKeyInput.SetValue("")
	v.secretKeyInput.SetValue("")
	v.mfaSerialInput.SetValue("")
	v.ssoURLInput.SetValue("")
	v.accountIDInput.SetValue("")
	v.roleNameInput.SetValue("")
	v.roleARNInput.SetValue("")
	v.externalIDInput.SetValue("")
}

// Update handles input for the wizard.
func (v *CreateWizardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle region step specially for search and navigation
		if v.step == StepRegion {
			switch msg.String() {
			case "esc":
				if v.onCancel != nil {
					return v, v.onCancel()
				}
				return v, nil
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
				if !v.useCustomRegion && len(v.filteredRegions) == 0 {
					v.useCustomRegion = true // Force custom if no matches
				}
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
			return v, nil

		case key.Matches(msg, v.keyMap.Next):
			return v, v.nextStep()

		case key.Matches(msg, v.keyMap.Prev):
			v.prevStep()
			return v, nil
		}

		// Handle type selection
		if v.step == StepSelectType {
			switch msg.String() {
			case "up", "k":
				if v.typeCursor > 0 {
					v.typeCursor--
				}
			case "down", "j":
				if v.typeCursor < 2 {
					v.typeCursor++
				}
			}
			return v, nil
		}

		// Handle parent selection
		if v.step == StepRoleParent {
			switch msg.String() {
			case "up", "k":
				if v.parentCursor > 0 {
					v.parentCursor--
				}
			case "down", "j":
				if v.parentCursor < len(v.parentSessions)-1 {
					v.parentCursor++
				}
			}
			return v, nil
		}
	}

	// Update current text input
	switch v.step {
	case StepSessionName:
		v.nameInput, cmd = v.nameInput.Update(msg)
	case StepProfileName:
		v.profileInput, cmd = v.profileInput.Update(msg)
	case StepIAMAccessKey:
		v.accessKeyInput, cmd = v.accessKeyInput.Update(msg)
	case StepIAMSecretKey:
		v.secretKeyInput, cmd = v.secretKeyInput.Update(msg)
	case StepIAMMFASerial:
		v.mfaSerialInput, cmd = v.mfaSerialInput.Update(msg)
	case StepSSOStartURL:
		v.ssoURLInput, cmd = v.ssoURLInput.Update(msg)
	case StepSSOAccountID:
		v.accountIDInput, cmd = v.accountIDInput.Update(msg)
	case StepSSORoleName:
		v.roleNameInput, cmd = v.roleNameInput.Update(msg)
	case StepRoleARN:
		v.roleARNInput, cmd = v.roleARNInput.Update(msg)
	case StepRoleExternalID:
		v.externalIDInput, cmd = v.externalIDInput.Update(msg)
	}

	return v, cmd
}

// nextStep advances to the next step or creates the session.
func (v *CreateWizardView) nextStep() tea.Cmd {
	switch v.step {
	case StepSelectType:
		switch v.typeCursor {
		case 0:
			v.sessionType = session.SessionTypeIAMUser
			v.step = StepSessionName
		case 1:
			v.sessionType = session.SessionTypeAWSSSO
			v.step = StepSessionName
		case 2:
			v.sessionType = session.SessionTypeIAMRole
			v.step = StepSessionName
		}
		v.nameInput.Focus()

	case StepSessionName:
		v.nameInput.Blur()
		v.step = StepProfileName
		v.profileInput.Focus()

	case StepProfileName:
		v.profileInput.Blur()
		v.step = StepRegion
		v.regionInput.Focus()

	case StepRegion:
		// Get the selected region value
		var selectedRegion string
		if v.useCustomRegion || len(v.filteredRegions) == 0 {
			selectedRegion = strings.TrimSpace(v.regionInput.Value())
		} else {
			if v.regionCursor < len(v.filteredRegions) {
				selectedRegion = v.filteredRegions[v.regionCursor]
			}
		}

		// Validate region is not empty
		if selectedRegion == "" {
			return nil
		}

		v.regionInput.Blur()
		switch v.sessionType {
		case session.SessionTypeIAMUser:
			v.step = StepIAMMFASerial
			v.mfaSerialInput.Focus()
		case session.SessionTypeAWSSSO:
			v.step = StepSSOStartURL
			v.ssoURLInput.Focus()
		case session.SessionTypeIAMRole:
			v.step = StepRoleARN
			v.roleARNInput.Focus()
		}

	case StepIAMMFASerial:
		v.mfaSerialInput.Blur()
		v.step = StepIAMAccessKey
		v.accessKeyInput.Focus()

	case StepIAMAccessKey:
		v.accessKeyInput.Blur()
		v.step = StepIAMSecretKey
		v.secretKeyInput.Focus()

	case StepIAMSecretKey:
		v.secretKeyInput.Blur()
		v.step = StepConfirm

	case StepSSOStartURL:
		v.ssoURLInput.Blur()
		v.step = StepSSOAccountID
		v.accountIDInput.Focus()

	case StepSSOAccountID:
		v.accountIDInput.Blur()
		v.step = StepSSORoleName
		v.roleNameInput.Focus()

	case StepSSORoleName:
		v.roleNameInput.Blur()
		v.step = StepConfirm

	case StepRoleARN:
		v.roleARNInput.Blur()
		v.step = StepRoleParent

	case StepRoleParent:
		v.step = StepRoleExternalID
		v.externalIDInput.Focus()

	case StepRoleExternalID:
		v.externalIDInput.Blur()
		v.step = StepConfirm

	case StepConfirm:
		return v.createSession()
	}

	return nil
}

// prevStep goes back to the previous step.
func (v *CreateWizardView) prevStep() {
	switch v.step {
	case StepSessionName:
		v.step = StepSelectType
	case StepProfileName:
		v.step = StepSessionName
	case StepRegion:
		v.step = StepProfileName
	case StepIAMMFASerial:
		v.step = StepRegion
	case StepIAMAccessKey:
		v.step = StepIAMMFASerial
	case StepIAMSecretKey:
		v.step = StepIAMAccessKey
	case StepSSOStartURL:
		v.step = StepRegion
	case StepSSOAccountID:
		v.step = StepSSOStartURL
	case StepSSORoleName:
		v.step = StepSSOAccountID
	case StepRoleARN:
		v.step = StepRegion
	case StepRoleParent:
		v.step = StepRoleARN
	case StepRoleExternalID:
		v.step = StepRoleParent
	case StepConfirm:
		switch v.sessionType {
		case session.SessionTypeIAMUser:
			v.step = StepIAMSecretKey
		case session.SessionTypeAWSSSO:
			v.step = StepSSORoleName
		case session.SessionTypeIAMRole:
			v.step = StepRoleExternalID
		}
	}
}

// createSession creates the session from the wizard data.
func (v *CreateWizardView) createSession() tea.Cmd {
	// Get the selected region
	var selectedRegion string
	if v.useCustomRegion || len(v.filteredRegions) == 0 {
		selectedRegion = strings.TrimSpace(v.regionInput.Value())
	} else {
		if v.regionCursor < len(v.filteredRegions) {
			selectedRegion = v.filteredRegions[v.regionCursor]
		} else {
			selectedRegion = "us-east-1" // Default fallback
		}
	}

	sess := session.NewSession(
		v.nameInput.Value(),
		session.ProviderAWS,
		v.sessionType,
		v.profileInput.Value(),
		selectedRegion,
	)

	var secretKey string

	switch v.sessionType {
	case session.SessionTypeIAMUser:
		sess.Config.IAMUser = &session.IAMUserConfig{
			AccessKeyID: v.accessKeyInput.Value(),
			MFASerial:   v.mfaSerialInput.Value(),
		}
		secretKey = v.secretKeyInput.Value()

	case session.SessionTypeAWSSSO:
		sess.Config.AWSSSO = &session.AWSSSOConfig{
			StartURL:  v.ssoURLInput.Value(),
			Region:    selectedRegion,
			AccountID: v.accountIDInput.Value(),
			RoleName:  v.roleNameInput.Value(),
		}

	case session.SessionTypeIAMRole:
		parentID := ""
		if v.parentCursor < len(v.parentSessions) {
			parentID = v.parentSessions[v.parentCursor].ID
		}
		sess.Config.IAMRole = &session.IAMRoleChainedConfig{
			ParentSessionID: parentID,
			RoleARN:         v.roleARNInput.Value(),
			ExternalID:      v.externalIDInput.Value(),
		}
	}

	if v.onCreate != nil {
		return v.onCreate(sess, secretKey)
	}

	return nil
}

// View renders the wizard.
func (v *CreateWizardView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Create New Session"))
	b.WriteString("\n\n")

	switch v.step {
	case StepSelectType:
		b.WriteString(v.theme.Subtitle.Render("Select session type:"))
		b.WriteString("\n\n")
		types := []string{"IAM User", "AWS SSO", "IAM Role (Chained)"}
		for i, t := range types {
			cursor := "  "
			style := v.theme.SessionItem
			if i == v.typeCursor {
				cursor = "▶ "
				style = v.theme.SessionItemSelected
			}
			b.WriteString(style.Render(cursor + t))
			b.WriteString("\n")
		}

	case StepSessionName:
		b.WriteString(v.theme.Label.Render("Session Alias *"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("A friendly name to identify this session"))
		b.WriteString("\n\n")
		b.WriteString(v.nameInput.View())

	case StepProfileName:
		b.WriteString(v.theme.Label.Render("Named Profile *"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("AWS CLI profile name for this session"))
		b.WriteString("\n\n")
		b.WriteString(v.profileInput.View())

	case StepRegion:
		b.WriteString(v.theme.Label.Render("Default Region:"))
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
			if v.regionInput.Value() != "" {
				b.WriteString(v.theme.InfoText.Render("Press Tab to use '"))
				b.WriteString(v.theme.Value.Render(v.regionInput.Value()))
				b.WriteString(v.theme.InfoText.Render("' as custom region"))
			}
		} else {
			// Show filtered regions
			maxVisible := 7
			start := v.regionCursor - 3
			if start < 0 {
				start = 0
			}
			end := start + maxVisible
			if end > len(v.filteredRegions) {
				end = len(v.filteredRegions)
				start = end - maxVisible
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
				b.WriteString("\n")
				b.WriteString(v.theme.Subtitle.Render(fmt.Sprintf("  %d matching regions", len(v.filteredRegions))))
			}

			b.WriteString("\n")
			b.WriteString(v.theme.InfoText.Render("Press Tab to enter custom region"))
		}

	case StepIAMAccessKey:
		b.WriteString(v.theme.Label.Render("Access Key ID *"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("Your AWS IAM user access key ID"))
		b.WriteString("\n\n")
		b.WriteString(v.accessKeyInput.View())

	case StepIAMSecretKey:
		b.WriteString(v.theme.Label.Render("Secret Access Key *"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("Your AWS IAM user secret access key (will be stored securely)"))
		b.WriteString("\n\n")
		b.WriteString(v.secretKeyInput.View())

	case StepIAMMFASerial:
		b.WriteString(v.theme.Label.Render("MFA Device (optional):"))
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("MFA Device ARN or Serial Number"))
		b.WriteString("\n\n")
		b.WriteString(v.mfaSerialInput.View())

	case StepSSOStartURL:
		b.WriteString(v.theme.Label.Render("SSO Start URL:"))
		b.WriteString("\n")
		b.WriteString(v.ssoURLInput.View())

	case StepSSOAccountID:
		b.WriteString(v.theme.Label.Render("Account ID:"))
		b.WriteString("\n")
		b.WriteString(v.accountIDInput.View())

	case StepSSORoleName:
		b.WriteString(v.theme.Label.Render("Role Name:"))
		b.WriteString("\n")
		b.WriteString(v.roleNameInput.View())

	case StepRoleARN:
		b.WriteString(v.theme.Label.Render("Role ARN:"))
		b.WriteString("\n")
		b.WriteString(v.roleARNInput.View())

	case StepRoleParent:
		b.WriteString(v.theme.Subtitle.Render("Select parent session:"))
		b.WriteString("\n\n")
		if len(v.parentSessions) == 0 {
			b.WriteString(v.theme.WarningText.Render("No sessions available. Create an IAM User or SSO session first."))
		} else {
			for i, s := range v.parentSessions {
				cursor := "  "
				style := v.theme.SessionItem
				if i == v.parentCursor {
					cursor = "▶ "
					style = v.theme.SessionItemSelected
				}
				b.WriteString(style.Render(fmt.Sprintf("%s%s (%s)", cursor, s.Name, s.ProfileName)))
				b.WriteString("\n")
			}
		}

	case StepRoleExternalID:
		b.WriteString(v.theme.Label.Render("External ID (optional):"))
		b.WriteString("\n")
		b.WriteString(v.externalIDInput.View())

	case StepConfirm:
		b.WriteString(v.theme.Subtitle.Render("Confirm session details:"))
		b.WriteString("\n\n")
		b.WriteString(v.renderSummary())
		b.WriteString("\n")
		b.WriteString(v.theme.InfoText.Render("Press Enter to create, Esc to cancel"))
	}

	// Help text
	b.WriteString("\n\n")
	b.WriteString(v.theme.Footer.Render("enter: next • shift+tab: back • esc: cancel"))

	return b.String()
}

// renderSummary renders a summary of the session being created.
func (v *CreateWizardView) renderSummary() string {
	var b strings.Builder

	renderLine := func(label, value string) {
		b.WriteString(fmt.Sprintf("%s %s\n",
			v.theme.Label.Render(label+":"),
			v.theme.Value.Render(value),
		))
	}

	// Get the selected region
	var selectedRegion string
	if v.useCustomRegion || len(v.filteredRegions) == 0 {
		selectedRegion = strings.TrimSpace(v.regionInput.Value())
	} else {
		if v.regionCursor < len(v.filteredRegions) {
			selectedRegion = v.filteredRegions[v.regionCursor]
		} else {
			selectedRegion = "us-east-1"
		}
	}

	renderLine("Session Alias", v.nameInput.Value())
	renderLine("Named Profile", v.profileInput.Value())
	renderLine("Region", selectedRegion)
	renderLine("Type", formatSessionTypeFull(v.sessionType))

	switch v.sessionType {
	case session.SessionTypeIAMUser:
		if v.mfaSerialInput.Value() != "" {
			renderLine("MFA Device", v.mfaSerialInput.Value())
		}
		renderLine("Access Key ID", v.accessKeyInput.Value())
		renderLine("Secret Key", "****** (hidden)")
	case session.SessionTypeAWSSSO:
		renderLine("SSO URL", v.ssoURLInput.Value())
		renderLine("Account", v.accountIDInput.Value())
		renderLine("Role", v.roleNameInput.Value())
	case session.SessionTypeIAMRole:
		renderLine("Role ARN", v.roleARNInput.Value())
		if v.parentCursor < len(v.parentSessions) {
			renderLine("Parent", v.parentSessions[v.parentCursor].Name)
		}
	}

	return b.String()
}

// Init initializes the view.
func (v *CreateWizardView) Init() tea.Cmd {
	return textinput.Blink
}

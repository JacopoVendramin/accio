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
	step           WizardStep
	sessionType    session.SessionType
	theme          *styles.Theme
	keyMap         CreateWizardKeyMap
	width          int
	height         int

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

	// Parent session selector
	parentSessions  []*session.Session
	parentCursor    int

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
		regionInput:     makeInput("us-east-1", false),
		accessKeyInput:  makeInput("AKIA...", false),
		secretKeyInput:  makeInput("secret key", true),
		mfaSerialInput:  makeInput("arn:aws:iam::123456789012:mfa/user (optional)", false),
		ssoURLInput:     makeInput("https://my-sso-portal.awsapps.com/start", false),
		accountIDInput:  makeInput("123456789012", false),
		roleNameInput:   makeInput("MyRole", false),
		roleARNInput:    makeInput("arn:aws:iam::123456789012:role/MyRole", false),
		externalIDInput: makeInput("external-id (optional)", false),
	}

	v.nameInput.Focus()
	v.regionInput.SetValue("us-east-1")

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

// Reset resets the wizard to the initial state.
func (v *CreateWizardView) Reset() {
	v.step = StepSelectType
	v.typeCursor = 0
	v.parentCursor = 0
	v.nameInput.SetValue("")
	v.profileInput.SetValue("")
	v.regionInput.SetValue("us-east-1")
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
	case StepRegion:
		v.regionInput, cmd = v.regionInput.Update(msg)
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
		v.regionInput.Blur()
		switch v.sessionType {
		case session.SessionTypeIAMUser:
			v.step = StepIAMAccessKey
			v.accessKeyInput.Focus()
		case session.SessionTypeAWSSSO:
			v.step = StepSSOStartURL
			v.ssoURLInput.Focus()
		case session.SessionTypeIAMRole:
			v.step = StepRoleARN
			v.roleARNInput.Focus()
		}

	case StepIAMAccessKey:
		v.accessKeyInput.Blur()
		v.step = StepIAMSecretKey
		v.secretKeyInput.Focus()

	case StepIAMSecretKey:
		v.secretKeyInput.Blur()
		v.step = StepIAMMFASerial
		v.mfaSerialInput.Focus()

	case StepIAMMFASerial:
		v.mfaSerialInput.Blur()
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
	case StepIAMAccessKey:
		v.step = StepRegion
	case StepIAMSecretKey:
		v.step = StepIAMAccessKey
	case StepIAMMFASerial:
		v.step = StepIAMSecretKey
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
			v.step = StepIAMMFASerial
		case session.SessionTypeAWSSSO:
			v.step = StepSSORoleName
		case session.SessionTypeIAMRole:
			v.step = StepRoleExternalID
		}
	}
}

// createSession creates the session from the wizard data.
func (v *CreateWizardView) createSession() tea.Cmd {
	sess := session.NewSession(
		v.nameInput.Value(),
		session.ProviderAWS,
		v.sessionType,
		v.profileInput.Value(),
		v.regionInput.Value(),
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
			Region:    v.regionInput.Value(),
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
		b.WriteString(v.theme.Label.Render("Session Name:"))
		b.WriteString("\n")
		b.WriteString(v.nameInput.View())

	case StepProfileName:
		b.WriteString(v.theme.Label.Render("AWS Profile Name:"))
		b.WriteString("\n")
		b.WriteString(v.profileInput.View())

	case StepRegion:
		b.WriteString(v.theme.Label.Render("Default Region:"))
		b.WriteString("\n")
		b.WriteString(v.regionInput.View())

	case StepIAMAccessKey:
		b.WriteString(v.theme.Label.Render("Access Key ID:"))
		b.WriteString("\n")
		b.WriteString(v.accessKeyInput.View())

	case StepIAMSecretKey:
		b.WriteString(v.theme.Label.Render("Secret Access Key:"))
		b.WriteString("\n")
		b.WriteString(v.secretKeyInput.View())

	case StepIAMMFASerial:
		b.WriteString(v.theme.Label.Render("MFA Serial (optional):"))
		b.WriteString("\n")
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

	renderLine("Name", v.nameInput.Value())
	renderLine("Profile", v.profileInput.Value())
	renderLine("Region", v.regionInput.Value())
	renderLine("Type", formatSessionTypeFull(v.sessionType))

	switch v.sessionType {
	case session.SessionTypeIAMUser:
		renderLine("Access Key", v.accessKeyInput.Value())
		if v.mfaSerialInput.Value() != "" {
			renderLine("MFA", v.mfaSerialInput.Value())
		}
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

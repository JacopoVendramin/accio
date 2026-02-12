// Package tui provides the terminal user interface for the application.
package tui

import (
	"context"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	integrationApp "github.com/jvendramin/accio/internal/application/integration"
	"github.com/jvendramin/accio/internal/application/session"
	"github.com/jvendramin/accio/internal/config"
	"github.com/jvendramin/accio/internal/domain/integration"
	domainSession "github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/infrastructure/aws/sso"
	"github.com/jvendramin/accio/internal/tui/styles"
	"github.com/jvendramin/accio/internal/tui/views"
	"github.com/jvendramin/accio/pkg/provider"
)

// ViewType represents the current view.
type ViewType int

const (
	ViewSessionList ViewType = iota
	ViewSessionDetail
	ViewSessionCreate
	ViewSessionEdit
	ViewIntegrationList
	ViewIntegrationCreate
	ViewIntegrationEdit
	ViewIntegrationSync
	ViewSettings
	ViewHelp
	ViewMFADialog
	ViewConfirmDialog
)

// App is the root TUI application model.
type App struct {
	// Services
	sessionService     *session.Service
	integrationService *integrationApp.Service
	configManager      *config.Manager

	// State
	currentView           ViewType
	previousView          ViewType
	sessions              []*domainSession.Session
	integrations          []*integration.Integration
	selectedSession       *domainSession.Session
	selectedIntegration   *integration.Integration
	lastSelectedSessionID string // Track last selected session ID for reload
	notification          string
	notifyExpiry          time.Time
	notifyIsError         bool
	width                 int
	height                int

	// Sync state
	syncDeviceAuth *provider.DeviceAuthorizationResponse
	syncAccounts   []integrationApp.AccountWithRoles
	syncToken      *provider.SSOToken

	// Views
	sessionListView       *views.SessionListView
	sessionDetailView     *views.SessionDetailView
	createWizardView      *views.CreateWizardView
	editSessionView       *views.EditSessionView
	integrationListView   *views.IntegrationListView
	integrationWizardView *views.IntegrationWizardView
	editIntegrationView   *views.EditIntegrationView
	syncView              *views.SyncView
	settingsView          *views.SettingsView
	helpView              *views.HelpView
	mfaDialogView         *views.MFADialogView
	confirmDialogView     *views.ConfirmDialogView

	// Styling
	theme *styles.Theme
}

// NewApp creates a new TUI application.
func NewApp(sessionService *session.Service, integrationService *integrationApp.Service, configManager *config.Manager) *App {
	theme := styles.DefaultTheme()

	app := &App{
		sessionService:     sessionService,
		integrationService: integrationService,
		configManager:      configManager,
		currentView:        ViewSessionList,
		theme:              theme,
	}

	// Initialize all views
	app.sessionListView = views.NewSessionListView(theme)
	app.sessionDetailView = views.NewSessionDetailView(theme)
	app.createWizardView = views.NewCreateWizardView(theme)
	app.editSessionView = views.NewEditSessionView(theme)
	app.integrationListView = views.NewIntegrationListView(theme)
	app.integrationWizardView = views.NewIntegrationWizardView(theme)
	app.editIntegrationView = views.NewEditIntegrationView(theme)
	app.syncView = views.NewSyncView(theme)
	app.settingsView = views.NewSettingsView(theme)
	app.helpView = views.NewHelpView(theme)
	app.mfaDialogView = views.NewMFADialogView(theme)
	app.confirmDialogView = views.NewConfirmDialogView(theme)

	// Set up session list callbacks
	app.sessionListView.SetOnStartStop(app.handleStartStop)
	app.sessionListView.SetOnNewIntegration(app.handleAddIntegration)
	app.sessionListView.SetOnDelete(app.handleDelete)
	app.sessionListView.SetOnRefresh(app.handleRefresh)

	// Set up detail view callbacks
	app.sessionDetailView.SetOnBack(app.handleBack)
	app.sessionDetailView.SetOnStart(app.handleStart)
	app.sessionDetailView.SetOnStop(app.handleStop)

	// Set up create wizard callbacks
	app.createWizardView.SetOnCreate(app.handleCreateSession)
	app.createWizardView.SetOnCancel(app.handleCancelCreate)

	// Set up settings view callbacks
	app.settingsView.SetOnSave(app.handleSaveSettings)
	app.settingsView.SetOnBack(app.handleBack)

	// Set up help view callbacks
	app.helpView.SetOnBack(app.handleBack)

	// Set up MFA dialog callbacks
	app.mfaDialogView.SetOnSubmit(app.handleMFASubmit)
	app.mfaDialogView.SetOnCancel(app.handleMFACancel)

	// Set up confirm dialog callbacks
	app.confirmDialogView.SetOnConfirm(app.handleConfirmAction)
	app.confirmDialogView.SetOnCancel(app.handleCancelConfirm)

	// Set up edit session view callbacks
	app.editSessionView.SetOnSave(app.handleSaveSession)
	app.editSessionView.SetOnCancel(app.handleCancelEdit)

	// Set up integration list callbacks
	app.integrationListView.SetOnSync(app.handleSyncIntegration)
	app.integrationListView.SetOnAdd(app.handleAddIntegration)
	app.integrationListView.SetOnDelete(app.handleDeleteIntegration)
	app.integrationListView.SetOnBack(app.handleBack)

	// Set up integration wizard callbacks
	app.integrationWizardView.SetOnCreate(app.handleCreateIntegration)
	app.integrationWizardView.SetOnCancel(app.handleCancelIntegrationCreate)

	// Set up edit integration view callbacks
	app.editIntegrationView.SetOnSave(app.handleSaveIntegration)
	app.editIntegrationView.SetOnCancel(app.handleCancelIntegrationEdit)

	// Set up sync view callbacks
	app.syncView.SetOnCancel(app.handleCancelSync)
	app.syncView.SetOnOpenURL(app.handleOpenURL)

	return app
}

// Messages

type SessionsLoadedMsg struct {
	Sessions []*domainSession.Session
	Err      error
}

type IntegrationsLoadedMsg struct {
	Integrations []*integration.Integration
	Err          error
}

type SessionStartedMsg struct {
	Session *domainSession.Session
	Err     error
}

type SessionStoppedMsg struct {
	Session *domainSession.Session
	Err     error
}

type SessionCreatedMsg struct {
	Session *domainSession.Session
	Err     error
}

type SessionDeletedMsg struct {
	SessionID string
	Err       error
}

type IntegrationCreatedMsg struct {
	Integration *integration.Integration
	Err         error
}

type IntegrationUpdatedMsg struct {
	Integration *integration.Integration
	Err         error
}

type IntegrationDeletedMsg struct {
	IntegrationID string
	Err           error
}

// Sync process messages
type SyncStartedMsg struct {
	DeviceAuth *provider.DeviceAuthorizationResponse
	Err        error
}

type SyncTokenReceivedMsg struct {
	Token *provider.SSOToken
	Err   error
}

type SyncAccountsLoadedMsg struct {
	Accounts []provider.SSOAccount
	Err      error
}

type SyncRolesLoadedMsg struct {
	Accounts []integrationApp.AccountWithRoles
	Err      error
}

type SyncSessionsCreatedMsg struct {
	Count int
	Err   error
}

type SyncPollTickMsg struct{}

type SettingsSavedMsg struct {
	Err error
}

type NotificationMsg struct {
	Message string
	IsError bool
}

type TickMsg time.Time

type SwitchViewMsg struct {
	View ViewType
}

// Init initializes the application.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.loadSessions(),
		a.loadIntegrations(),
		a.tickCmd(),
	)
}

// Update handles messages and updates the model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateViewSizes()

	case SessionsLoadedMsg:
		if msg.Err != nil {
			a.setNotification("Error loading sessions: "+msg.Err.Error(), true)
		} else {
			a.sessions = msg.Sessions
			a.sessionListView.SetSessions(msg.Sessions)
			a.createWizardView.SetParentSessions(msg.Sessions)
			// Restore last selected session if available
			if a.lastSelectedSessionID != "" {
				a.sessionListView.SelectSessionByID(a.lastSelectedSessionID)
			}
		}

	case IntegrationsLoadedMsg:
		if msg.Err != nil {
			a.setNotification("Error loading integrations: "+msg.Err.Error(), true)
		} else {
			a.integrations = msg.Integrations
			a.integrationListView.SetIntegrations(msg.Integrations)
			a.sessionListView.SetIntegrations(msg.Integrations)
		}

	case SessionStartedMsg:
		if msg.Err != nil {
			if isMFAError(msg.Err) {
				a.mfaDialogView.SetSession(msg.Session)
				a.previousView = a.currentView
				a.currentView = ViewMFADialog
			} else {
				a.setNotification("Error starting session: "+msg.Err.Error(), true)
			}
		} else {
			a.setNotification("Session started: "+msg.Session.Name, false)
			// Save the session ID to restore selection after reload
			a.lastSelectedSessionID = msg.Session.ID
			cmds = append(cmds, a.loadSessions())
		}

	case SessionStoppedMsg:
		if msg.Err != nil {
			a.setNotification("Error stopping session: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Session stopped: "+msg.Session.Name, false)
			// Save the session ID to restore selection after reload
			a.lastSelectedSessionID = msg.Session.ID
			cmds = append(cmds, a.loadSessions())
		}

	case SessionCreatedMsg:
		if msg.Err != nil {
			a.setNotification("Error creating session: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Session created: "+msg.Session.Name, false)
			a.currentView = ViewSessionList
			a.createWizardView.Reset()
			cmds = append(cmds, a.loadSessions())
		}

	case SessionDeletedMsg:
		if msg.Err != nil {
			a.setNotification("Error deleting session: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Session deleted", false)
			cmds = append(cmds, a.loadSessions())
		}

	case IntegrationCreatedMsg:
		if msg.Err != nil {
			a.setNotification("Error creating integration: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Integration created: "+msg.Integration.Name, false)
			a.currentView = ViewIntegrationList
			a.integrationWizardView.Reset()
			cmds = append(cmds, a.loadIntegrations())
		}

	case IntegrationUpdatedMsg:
		if msg.Err != nil {
			a.setNotification("Error updating integration: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Integration updated: "+msg.Integration.Name, false)
			a.currentView = ViewIntegrationList
			cmds = append(cmds, a.loadIntegrations())
		}

	case IntegrationDeletedMsg:
		if msg.Err != nil {
			a.setNotification("Error deleting integration: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Integration deleted", false)
			cmds = append(cmds, a.loadIntegrations(), a.loadSessions())
		}

	case SyncStartedMsg:
		if msg.Err != nil {
			a.syncView.SetError(msg.Err.Error())
		} else {
			a.syncDeviceAuth = msg.DeviceAuth
			a.syncView.SetDeviceAuth(msg.DeviceAuth)
			// Start polling for token
			cmds = append(cmds, a.syncPollTick(), a.syncView.SpinnerTick())
		}

	case SyncPollTickMsg:
		if a.currentView == ViewIntegrationSync && a.syncDeviceAuth != nil {
			cmds = append(cmds, a.pollForSSOToken())
		}

	case SyncTokenReceivedMsg:
		if msg.Err != nil {
			// Check if it's "authorization_pending" - continue polling
			if isAuthorizationPending(msg.Err) {
				cmds = append(cmds, a.syncPollTick())
			} else {
				a.syncView.SetError(msg.Err.Error())
			}
		} else {
			a.syncToken = msg.Token
			a.syncView.SetState(views.SyncStateFetchingAccounts)
			// Store the token
			if a.integrationService != nil && a.selectedIntegration != nil {
				_ = a.integrationService.StoreToken(a.selectedIntegration.ID, msg.Token)
			}
			cmds = append(cmds, a.fetchSSOAccounts())
		}

	case SyncAccountsLoadedMsg:
		if msg.Err != nil {
			a.syncView.SetError(msg.Err.Error())
		} else {
			a.syncView.SetState(views.SyncStateFetchingRoles)
			cmds = append(cmds, a.fetchSSORoles(msg.Accounts))
		}

	case SyncRolesLoadedMsg:
		if msg.Err != nil {
			a.syncView.SetError(msg.Err.Error())
		} else {
			a.syncAccounts = msg.Accounts
			var discovered []views.DiscoveredAccount
			for _, acct := range msg.Accounts {
				discovered = append(discovered, views.DiscoveredAccount{
					AccountID:   acct.AccountID,
					AccountName: acct.AccountName,
					Email:       acct.Email,
					Roles:       acct.Roles,
					Selected:    true,
				})
			}
			a.syncView.SetAccounts(discovered)
			a.syncView.SetState(views.SyncStateCreatingSessions)
			cmds = append(cmds, a.createSyncSessions())
		}

	case SyncSessionsCreatedMsg:
		if msg.Err != nil {
			a.syncView.SetError(msg.Err.Error())
		} else {
			a.syncView.SetCreatedCount(msg.Count)
			a.syncView.SetComplete()
			cmds = append(cmds, a.loadSessions(), a.loadIntegrations())
		}

	case SettingsSavedMsg:
		if msg.Err != nil {
			a.setNotification("Error saving settings: "+msg.Err.Error(), true)
		} else {
			a.setNotification("Settings saved", false)
		}

	case NotificationMsg:
		a.setNotification(msg.Message, msg.IsError)

	case SwitchViewMsg:
		a.previousView = a.currentView
		a.currentView = msg.View
		// Start sync process when entering sync view
		if msg.View == ViewIntegrationSync && a.selectedIntegration != nil {
			cmds = append(cmds, a.startSSODeviceAuth(), a.syncView.SpinnerTick())
		}

	case TickMsg:
		if !a.notifyExpiry.IsZero() && time.Now().After(a.notifyExpiry) {
			a.notification = ""
			a.notifyExpiry = time.Time{}
		}
		cmds = append(cmds, a.tickCmd())

	case tea.KeyMsg:
		// Global key handling
		if a.currentView == ViewSessionList {
			// Don't process global shortcuts if search is active
			if !a.sessionListView.IsSearchActive() {
				switch msg.String() {
				case "v":
					if sess := a.sessionListView.Selected(); sess != nil {
						a.selectedSession = sess
						a.sessionDetailView.SetSession(sess)
						a.previousView = a.currentView
						a.currentView = ViewSessionDetail
						return a, nil
					}
				case "i":
					// Show integrations
					a.previousView = a.currentView
					a.currentView = ViewIntegrationList
					return a, a.loadIntegrations()
				case "s":
					a.settingsView.SetConfig(a.configManager.Get())
					a.previousView = a.currentView
					a.currentView = ViewSettings
					return a, nil
				case "?":
					a.previousView = a.currentView
					a.currentView = ViewHelp
					return a, nil
				}
			}
		}

		// Integration list shortcuts
		if a.currentView == ViewIntegrationList {
			switch msg.String() {
			case "e":
				if integ := a.integrationListView.Selected(); integ != nil {
					a.selectedIntegration = integ
					a.editIntegrationView.SetIntegration(integ)
					a.previousView = a.currentView
					a.currentView = ViewIntegrationEdit
					return a, nil
				}
			}
		}

		cmd := a.updateCurrentView(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) updateCurrentView(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch a.currentView {
	case ViewSessionList:
		_, cmd = a.sessionListView.Update(msg)
		// Track the currently selected session ID for stable selection after reload
		if sess := a.sessionListView.Selected(); sess != nil {
			a.lastSelectedSessionID = sess.ID
		}
	case ViewSessionDetail:
		_, cmd = a.sessionDetailView.Update(msg)
	case ViewSessionCreate:
		_, cmd = a.createWizardView.Update(msg)
	case ViewSessionEdit:
		_, cmd = a.editSessionView.Update(msg)
	case ViewIntegrationList:
		_, cmd = a.integrationListView.Update(msg)
	case ViewIntegrationCreate:
		_, cmd = a.integrationWizardView.Update(msg)
	case ViewIntegrationEdit:
		_, cmd = a.editIntegrationView.Update(msg)
	case ViewIntegrationSync:
		_, cmd = a.syncView.Update(msg)
	case ViewSettings:
		_, cmd = a.settingsView.Update(msg)
	case ViewHelp:
		_, cmd = a.helpView.Update(msg)
	case ViewMFADialog:
		_, cmd = a.mfaDialogView.Update(msg)
	case ViewConfirmDialog:
		_, cmd = a.confirmDialogView.Update(msg)
	}

	return cmd
}

func (a *App) updateViewSizes() {
	a.sessionListView.SetSize(a.width, a.height)
	a.sessionDetailView.SetSize(a.width, a.height)
	a.createWizardView.SetSize(a.width, a.height)
	a.editSessionView.SetSize(a.width, a.height)
	a.integrationListView.SetSize(a.width, a.height)
	a.integrationWizardView.SetSize(a.width, a.height)
	a.editIntegrationView.SetSize(a.width, a.height)
	a.syncView.SetSize(a.width, a.height)
	a.settingsView.SetSize(a.width, a.height)
	a.helpView.SetSize(a.width, a.height)
	a.mfaDialogView.SetSize(a.width, a.height)
	a.confirmDialogView.SetSize(a.width, a.height)
}

// View renders the application.
func (a *App) View() string {
	var content string

	switch a.currentView {
	case ViewSessionList:
		content = a.sessionListView.View()
	case ViewSessionDetail:
		content = a.sessionDetailView.View()
	case ViewSessionCreate:
		content = a.createWizardView.View()
	case ViewSessionEdit:
		content = a.editSessionView.View()
	case ViewIntegrationList:
		content = a.integrationListView.View()
	case ViewIntegrationCreate:
		content = a.integrationWizardView.View()
	case ViewIntegrationEdit:
		content = a.editIntegrationView.View()
	case ViewIntegrationSync:
		content = a.syncView.View()
	case ViewSettings:
		content = a.settingsView.View()
	case ViewHelp:
		content = a.helpView.View()
	case ViewMFADialog:
		content = a.mfaDialogView.View()
	case ViewConfirmDialog:
		content = a.confirmDialogView.View()
	default:
		content = a.sessionListView.View()
	}

	if a.notification != "" && time.Now().Before(a.notifyExpiry) {
		notifyStyle := a.theme.InfoText
		if a.notifyIsError {
			notifyStyle = a.theme.ErrorText
		}
		content += "\n\n" + notifyStyle.Render(a.notification)
	}

	return a.theme.App.Render(content)
}

// Commands

func (a *App) loadSessions() tea.Cmd {
	return func() tea.Msg {
		if a.sessionService == nil {
			return SessionsLoadedMsg{Sessions: nil}
		}
		sessions, err := a.sessionService.List(context.Background())
		if err == nil && sessions != nil {
			// Sort sessions by integration ID, then by name for stable ordering
			sort.Slice(sessions, func(i, j int) bool {
				// Get integration IDs
				integIDi := ""
				integIDj := ""
				if sessions[i].Config.AWSSSO != nil {
					integIDi = sessions[i].Config.AWSSSO.IntegrationID
				}
				if sessions[j].Config.AWSSSO != nil {
					integIDj = sessions[j].Config.AWSSSO.IntegrationID
				}

				// First sort by integration ID
				if integIDi != integIDj {
					return integIDi < integIDj
				}
				// Then by session name
				return sessions[i].Name < sessions[j].Name
			})
		}
		return SessionsLoadedMsg{Sessions: sessions, Err: err}
	}
}

func (a *App) loadIntegrations() tea.Cmd {
	return func() tea.Msg {
		if a.integrationService == nil {
			return IntegrationsLoadedMsg{Integrations: nil}
		}
		integrations, err := a.integrationService.List(context.Background())
		return IntegrationsLoadedMsg{Integrations: integrations, Err: err}
	}
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Session callbacks

func (a *App) handleStartStop(sess *domainSession.Session) tea.Cmd {
	return func() tea.Msg {
		if sess.IsActive() {
			return a.stopSession(sess)
		}

		// Stop any currently active session first (single connection only)
		for _, s := range a.sessions {
			if s.IsActive() && s.ID != sess.ID {
				a.stopSession(s)
			}
		}

		if sess.RequiresMFA() {
			a.mfaDialogView.SetSession(sess)
			a.previousView = a.currentView
			return SwitchViewMsg{View: ViewMFADialog}
		}
		return a.startSession(sess, "")
	}
}

func (a *App) handleDelete(sess *domainSession.Session) tea.Cmd {
	return func() tea.Msg {
		a.confirmDialogView.SetContent(
			"Delete Session",
			"Are you sure you want to delete '"+sess.Name+"'?",
		)
		a.confirmDialogView.SetButtons("Delete", "Cancel")
		a.confirmDialogView.SetData(sess)
		a.previousView = a.currentView
		return SwitchViewMsg{View: ViewConfirmDialog}
	}
}

func (a *App) handleRefresh() tea.Cmd {
	return a.loadSessions()
}

func (a *App) handleStart(sess *domainSession.Session) tea.Cmd {
	return func() tea.Msg {
		if sess.RequiresMFA() {
			a.mfaDialogView.SetSession(sess)
			a.previousView = a.currentView
			return SwitchViewMsg{View: ViewMFADialog}
		}
		return a.startSession(sess, "")
	}
}

func (a *App) handleStop(sess *domainSession.Session) tea.Cmd {
	return func() tea.Msg {
		return a.stopSession(sess)
	}
}

func (a *App) handleBack() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: ViewSessionList}
	}
}

func (a *App) handleCreateSession(sess *domainSession.Session, secretKey string) tea.Cmd {
	return func() tea.Msg {
		if a.sessionService == nil {
			return SessionCreatedMsg{Session: sess, Err: nil}
		}
		err := a.sessionService.CreateWithSecret(context.Background(), sess, secretKey)
		return SessionCreatedMsg{Session: sess, Err: err}
	}
}

func (a *App) handleCancelCreate() tea.Cmd {
	return func() tea.Msg {
		a.createWizardView.Reset()
		return SwitchViewMsg{View: ViewSessionList}
	}
}

func (a *App) handleSaveSession(sess *domainSession.Session) tea.Cmd {
	return func() tea.Msg {
		if a.sessionService == nil {
			return NotificationMsg{Message: "Session updated", IsError: false}
		}
		err := a.sessionService.Update(context.Background(), sess)
		if err != nil {
			return NotificationMsg{Message: "Error saving session: " + err.Error(), IsError: true}
		}
		a.currentView = ViewSessionList
		return NotificationMsg{Message: "Session updated: " + sess.Name, IsError: false}
	}
}

func (a *App) handleCancelEdit() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: ViewSessionList}
	}
}

// Settings callbacks

func (a *App) handleSaveSettings(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		a.configManager.Set(cfg)
		err := a.configManager.Save()
		return SettingsSavedMsg{Err: err}
	}
}

// MFA callbacks

func (a *App) handleMFASubmit(sess *domainSession.Session, token string) tea.Cmd {
	return func() tea.Msg {
		result := a.startSession(sess, token)
		a.currentView = a.previousView
		return result
	}
}

func (a *App) handleMFACancel() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: a.previousView}
	}
}

// Confirm dialog callbacks

func (a *App) handleConfirmAction(data interface{}) tea.Cmd {
	return func() tea.Msg {
		switch v := data.(type) {
		case *domainSession.Session:
			if a.sessionService != nil {
				err := a.sessionService.DeleteWithSecret(context.Background(), v.ID)
				a.currentView = a.previousView
				return SessionDeletedMsg{SessionID: v.ID, Err: err}
			}
		case *integration.Integration:
			if a.integrationService != nil {
				err := a.integrationService.Delete(context.Background(), v.ID)
				a.currentView = a.previousView
				return IntegrationDeletedMsg{IntegrationID: v.ID, Err: err}
			}
		}
		return SwitchViewMsg{View: a.previousView}
	}
}

func (a *App) handleCancelConfirm() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: a.previousView}
	}
}

// Integration callbacks

func (a *App) handleSyncIntegration(integ *integration.Integration) tea.Cmd {
	return func() tea.Msg {
		a.selectedIntegration = integ
		a.syncView.SetIntegration(integ)
		a.syncView.SetState(views.SyncStateAuthenticating)
		a.syncDeviceAuth = nil
		a.syncToken = nil
		a.syncAccounts = nil
		a.previousView = a.currentView
		return SwitchViewMsg{View: ViewIntegrationSync}
	}
}

// Sync process commands

func (a *App) startSSODeviceAuth() tea.Cmd {
	return func() tea.Msg {
		if a.selectedIntegration == nil || a.selectedIntegration.Config.AWSSSO == nil {
			return SyncStartedMsg{Err: ErrNoIntegration}
		}

		ssoConfig := a.selectedIntegration.Config.AWSSSO
		client := sso.NewClient(ssoConfig.Region)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		deviceAuth, err := client.StartDeviceAuthorization(ctx, ssoConfig.StartURL)
		return SyncStartedMsg{DeviceAuth: deviceAuth, Err: err}
	}
}

func (a *App) syncPollTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return SyncPollTickMsg{}
	})
}

func (a *App) pollForSSOToken() tea.Cmd {
	return func() tea.Msg {
		if a.syncDeviceAuth == nil || a.selectedIntegration == nil {
			return SyncTokenReceivedMsg{Err: ErrNoDeviceAuth}
		}

		ssoConfig := a.selectedIntegration.Config.AWSSSO
		client := sso.NewClient(ssoConfig.Region)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		token, err := client.PollForToken(ctx,
			a.syncDeviceAuth.ClientID,
			a.syncDeviceAuth.ClientSecret,
			a.syncDeviceAuth.DeviceCode,
		)
		return SyncTokenReceivedMsg{Token: token, Err: err}
	}
}

func (a *App) fetchSSOAccounts() tea.Cmd {
	return func() tea.Msg {
		if a.syncToken == nil || a.selectedIntegration == nil {
			return SyncAccountsLoadedMsg{Err: ErrNoToken}
		}

		ssoConfig := a.selectedIntegration.Config.AWSSSO
		client := sso.NewClient(ssoConfig.Region)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		accounts, err := client.ListAccounts(ctx, a.syncToken.AccessToken)
		return SyncAccountsLoadedMsg{Accounts: accounts, Err: err}
	}
}

func (a *App) fetchSSORoles(accounts []provider.SSOAccount) tea.Cmd {
	return func() tea.Msg {
		if a.syncToken == nil || a.selectedIntegration == nil {
			return SyncRolesLoadedMsg{Err: ErrNoToken}
		}

		ssoConfig := a.selectedIntegration.Config.AWSSSO
		client := sso.NewClient(ssoConfig.Region)

		var result []integrationApp.AccountWithRoles

		for _, acct := range accounts {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			roles, err := client.ListAccountRoles(ctx, a.syncToken.AccessToken, acct.AccountID)
			cancel()

			if err != nil {
				continue // Skip accounts we can't get roles for
			}

			var roleNames []string
			for _, role := range roles {
				roleNames = append(roleNames, role.RoleName)
			}

			result = append(result, integrationApp.AccountWithRoles{
				AccountID:   acct.AccountID,
				AccountName: acct.AccountName,
				Email:       acct.EmailAddress,
				Roles:       roleNames,
			})
		}

		return SyncRolesLoadedMsg{Accounts: result, Err: nil}
	}
}

func (a *App) createSyncSessions() tea.Cmd {
	return func() tea.Msg {
		if a.integrationService == nil || a.selectedIntegration == nil {
			return SyncSessionsCreatedMsg{Count: 0, Err: nil}
		}

		ctx := context.Background()
		created, err := a.integrationService.CreateSessionsFromSync(ctx, a.selectedIntegration, a.syncAccounts)
		return SyncSessionsCreatedMsg{Count: len(created), Err: err}
	}
}

func (a *App) handleOpenURL(url string) tea.Cmd {
	return func() tea.Msg {
		openBrowser(url)
		return nil
	}
}

func (a *App) handleAddIntegration() tea.Cmd {
	return func() tea.Msg {
		a.integrationWizardView.Reset()
		return SwitchViewMsg{View: ViewIntegrationCreate}
	}
}

func (a *App) handleDeleteIntegration(integ *integration.Integration) tea.Cmd {
	return func() tea.Msg {
		a.confirmDialogView.SetContent(
			"Delete Integration",
			"Are you sure you want to delete '"+integ.Name+"'?\nThis will also remove all associated sessions.",
		)
		a.confirmDialogView.SetButtons("Delete", "Cancel")
		a.confirmDialogView.SetData(integ)
		a.previousView = a.currentView
		return SwitchViewMsg{View: ViewConfirmDialog}
	}
}

func (a *App) handleCreateIntegration(integ *integration.Integration) tea.Cmd {
	return func() tea.Msg {
		if a.integrationService == nil {
			return IntegrationCreatedMsg{Integration: integ, Err: nil}
		}
		err := a.integrationService.Create(context.Background(), integ)
		return IntegrationCreatedMsg{Integration: integ, Err: err}
	}
}

func (a *App) handleCancelIntegrationCreate() tea.Cmd {
	return func() tea.Msg {
		a.integrationWizardView.Reset()
		return SwitchViewMsg{View: ViewIntegrationList}
	}
}

func (a *App) handleSaveIntegration(integ *integration.Integration) tea.Cmd {
	return func() tea.Msg {
		if a.integrationService == nil {
			return IntegrationUpdatedMsg{Integration: integ, Err: nil}
		}
		err := a.integrationService.Update(context.Background(), integ)
		return IntegrationUpdatedMsg{Integration: integ, Err: err}
	}
}

func (a *App) handleCancelIntegrationEdit() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: ViewIntegrationList}
	}
}

func (a *App) handleCancelSync() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: ViewIntegrationList}
	}
}

// Session operations

func (a *App) startSession(sess *domainSession.Session, mfaToken string) tea.Msg {
	if a.sessionService == nil {
		return SessionStartedMsg{Session: sess, Err: nil}
	}
	err := a.sessionService.Start(context.Background(), sess.ID, mfaToken)
	return SessionStartedMsg{Session: sess, Err: err}
}

func (a *App) stopSession(sess *domainSession.Session) tea.Msg {
	if a.sessionService == nil {
		return SessionStoppedMsg{Session: sess, Err: nil}
	}
	err := a.sessionService.Stop(context.Background(), sess.ID)
	return SessionStoppedMsg{Session: sess, Err: err}
}

func (a *App) setNotification(msg string, isError bool) {
	a.notification = msg
	a.notifyIsError = isError
	if isError {
		a.notifyExpiry = time.Now().Add(10 * time.Second)
	} else {
		a.notifyExpiry = time.Now().Add(5 * time.Second)
	}
}

func isMFAError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "MFA") || contains(errStr, "mfa") || contains(errStr, "token required")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isAuthorizationPending(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "authorization_pending") ||
		strings.Contains(errStr, "AuthorizationPendingException")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}

// Sync errors
var (
	ErrNoIntegration = errorString("no integration selected")
	ErrNoDeviceAuth  = errorString("no device authorization in progress")
	ErrNoToken       = errorString("no access token available")
)

type errorString string

func (e errorString) Error() string {
	return string(e)
}

// Run starts the TUI application.
func Run(sessionService *session.Service, integrationService *integrationApp.Service, configManager *config.Manager) error {
	app := NewApp(sessionService, integrationService, configManager)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

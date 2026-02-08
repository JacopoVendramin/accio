package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/tui/styles"
	"github.com/jvendramin/accio/pkg/provider"
)

// SyncState represents the current state of the sync process.
type SyncState int

const (
	SyncStateIdle SyncState = iota
	SyncStateAuthenticating
	SyncStateWaitingForBrowser
	SyncStatePollingToken
	SyncStateFetchingAccounts
	SyncStateFetchingRoles
	SyncStateCreatingSessions
	SyncStateComplete
	SyncStateError
)

// SyncKeyMap defines key bindings for the sync view.
type SyncKeyMap struct {
	Cancel key.Binding
	Done   key.Binding
}

// DefaultSyncKeyMap returns the default key bindings.
func DefaultSyncKeyMap() SyncKeyMap {
	return SyncKeyMap{
		Cancel: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "cancel"),
		),
		Done: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "done"),
		),
	}
}

// DiscoveredAccount represents an account found during sync.
type DiscoveredAccount struct {
	AccountID   string
	AccountName string
	Email       string
	Roles       []string
	Selected    bool
}

// SyncView shows the sync process for an integration.
type SyncView struct {
	integration *integration.Integration
	state       SyncState
	spinner     spinner.Model
	theme       *styles.Theme
	keyMap      SyncKeyMap
	width       int
	height      int

	// Sync data
	deviceAuth   *provider.DeviceAuthorizationResponse
	accounts     []DiscoveredAccount
	totalRoles   int
	createdCount int
	errorMsg     string

	// Callbacks
	onComplete func([]*DiscoveredAccount) tea.Cmd
	onCancel   func() tea.Cmd
	onOpenURL  func(string) tea.Cmd
}

// NewSyncView creates a new sync view.
func NewSyncView(theme *styles.Theme) *SyncView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.InfoText

	return &SyncView{
		state:   SyncStateIdle,
		spinner: s,
		theme:   theme,
		keyMap:  DefaultSyncKeyMap(),
	}
}

// SetIntegration sets the integration to sync.
func (v *SyncView) SetIntegration(integ *integration.Integration) {
	v.integration = integ
	v.state = SyncStateIdle
	v.deviceAuth = nil
	v.accounts = nil
	v.totalRoles = 0
	v.createdCount = 0
	v.errorMsg = ""
}

// SetState sets the current sync state.
func (v *SyncView) SetState(state SyncState) {
	v.state = state
}

// SetDeviceAuth sets the device authorization response.
func (v *SyncView) SetDeviceAuth(auth *provider.DeviceAuthorizationResponse) {
	v.deviceAuth = auth
	v.state = SyncStateWaitingForBrowser
}

// SetAccounts sets the discovered accounts.
func (v *SyncView) SetAccounts(accounts []DiscoveredAccount) {
	v.accounts = accounts
	for _, a := range accounts {
		v.totalRoles += len(a.Roles)
	}
}

// SetCreatedCount sets the number of created sessions.
func (v *SyncView) SetCreatedCount(count int) {
	v.createdCount = count
}

// SetError sets an error message.
func (v *SyncView) SetError(err string) {
	v.state = SyncStateError
	v.errorMsg = err
}

// SetComplete marks the sync as complete.
func (v *SyncView) SetComplete() {
	v.state = SyncStateComplete
}

// SetSize sets the view size.
func (v *SyncView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnComplete sets the callback for completion.
func (v *SyncView) SetOnComplete(fn func([]*DiscoveredAccount) tea.Cmd) {
	v.onComplete = fn
}

// SetOnCancel sets the callback for cancellation.
func (v *SyncView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// SetOnOpenURL sets the callback for opening URLs.
func (v *SyncView) SetOnOpenURL(fn func(string) tea.Cmd) {
	v.onOpenURL = fn
}

// GetDeviceAuth returns the device authorization response.
func (v *SyncView) GetDeviceAuth() *provider.DeviceAuthorizationResponse {
	return v.deviceAuth
}

// GetIntegration returns the integration being synced.
func (v *SyncView) GetIntegration() *integration.Integration {
	return v.integration
}

// Update handles input for the sync view.
func (v *SyncView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.state != SyncStateComplete {
				if v.onCancel != nil {
					return v, v.onCancel()
				}
			}
		case key.Matches(msg, v.keyMap.Done):
			if v.state == SyncStateComplete || v.state == SyncStateError {
				if v.onCancel != nil {
					return v, v.onCancel()
				}
			}
		case msg.String() == "o":
			// Open browser
			if v.state == SyncStateWaitingForBrowser && v.deviceAuth != nil {
				url := v.deviceAuth.VerificationURIComplete
				if url == "" {
					url = v.deviceAuth.VerificationURI
				}
				if v.onOpenURL != nil {
					return v, v.onOpenURL(url)
				}
			}
		}

	case spinner.TickMsg:
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	return v, nil
}

// View renders the sync view.
func (v *SyncView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("Sync Integration"))
	b.WriteString("\n")
	if v.integration != nil {
		b.WriteString(v.theme.Subtitle.Render(v.integration.Name))
	}
	b.WriteString("\n\n")

	switch v.state {
	case SyncStateIdle, SyncStateAuthenticating:
		b.WriteString(v.spinner.View())
		b.WriteString(" Starting authentication...")

	case SyncStateWaitingForBrowser:
		b.WriteString(v.theme.Label.Render("Please authenticate in your browser"))
		b.WriteString("\n\n")

		if v.deviceAuth != nil {
			url := v.deviceAuth.VerificationURIComplete
			if url == "" {
				url = v.deviceAuth.VerificationURI
			}
			b.WriteString(v.theme.Label.Render("URL: "))
			b.WriteString(v.theme.Value.Render(url))
			b.WriteString("\n\n")

			if v.deviceAuth.UserCode != "" {
				b.WriteString(v.theme.Label.Render("Code: "))
				b.WriteString(v.theme.Title.Render(v.deviceAuth.UserCode))
				b.WriteString("\n\n")
			}
		}

		b.WriteString(v.spinner.View())
		b.WriteString(" Waiting for browser authorization...")
		b.WriteString("\n\n")
		b.WriteString(v.theme.InfoText.Render("Press 'o' to open browser"))

	case SyncStatePollingToken:
		b.WriteString(v.spinner.View())
		b.WriteString(" Verifying authentication...")

	case SyncStateFetchingAccounts:
		b.WriteString(v.spinner.View())
		b.WriteString(" Discovering accounts...")

	case SyncStateFetchingRoles:
		b.WriteString(v.spinner.View())
		b.WriteString(fmt.Sprintf(" Fetching roles for %d accounts...", len(v.accounts)))

	case SyncStateCreatingSessions:
		b.WriteString(v.spinner.View())
		b.WriteString(fmt.Sprintf(" Creating sessions... (%d/%d)", v.createdCount, v.totalRoles))

	case SyncStateComplete:
		b.WriteString(v.theme.SuccessText.Render("✓ Sync complete!"))
		b.WriteString("\n\n")

		b.WriteString(v.theme.Label.Render("Accounts discovered: "))
		b.WriteString(v.theme.Value.Render(fmt.Sprintf("%d", len(v.accounts))))
		b.WriteString("\n")

		b.WriteString(v.theme.Label.Render("Sessions created: "))
		b.WriteString(v.theme.Value.Render(fmt.Sprintf("%d", v.createdCount)))
		b.WriteString("\n\n")

		// Show accounts summary
		for _, acct := range v.accounts {
			b.WriteString(fmt.Sprintf("  • %s (%s) - %d roles\n",
				acct.AccountName,
				acct.AccountID,
				len(acct.Roles),
			))
		}

		b.WriteString("\n")
		b.WriteString(v.theme.InfoText.Render("Press Enter to continue"))

	case SyncStateError:
		b.WriteString(v.theme.ErrorText.Render("✗ Sync failed"))
		b.WriteString("\n\n")
		b.WriteString(v.theme.ErrorText.Render(v.errorMsg))
		b.WriteString("\n\n")
		b.WriteString(v.theme.InfoText.Render("Press Enter to go back"))
	}

	b.WriteString("\n\n")
	if v.state == SyncStateComplete || v.state == SyncStateError {
		b.WriteString(v.theme.Footer.Render("enter: continue"))
	} else {
		b.WriteString(v.theme.Footer.Render("esc: cancel"))
	}

	return b.String()
}

// Init initializes the view.
func (v *SyncView) Init() tea.Cmd {
	return v.spinner.Tick
}

// SpinnerTick returns a command to tick the spinner.
func (v *SyncView) SpinnerTick() tea.Cmd {
	return v.spinner.Tick
}

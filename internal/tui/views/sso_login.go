package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/tui/styles"
	"github.com/jvendramin/accio/pkg/provider"
)

// SSOLoginState represents the current state of the SSO login flow.
type SSOLoginState int

const (
	SSOStateIdle SSOLoginState = iota
	SSOStateStarting
	SSOStateWaitingForUser
	SSOStatePolling
	SSOStateSuccess
	SSOStateError
)

// SSOLoginKeyMap defines key bindings for the SSO login view.
type SSOLoginKeyMap struct {
	OpenBrowser key.Binding
	CopyURL     key.Binding
	Cancel      key.Binding
}

// DefaultSSOLoginKeyMap returns the default key bindings.
func DefaultSSOLoginKeyMap() SSOLoginKeyMap {
	return SSOLoginKeyMap{
		OpenBrowser: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in browser"),
		),
		CopyURL: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy URL"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// SSOLoginView handles the SSO device authorization flow.
type SSOLoginView struct {
	state         SSOLoginState
	startURL      string
	region        string
	deviceAuth    *provider.DeviceAuthorizationResponse
	spinner       spinner.Model
	theme         *styles.Theme
	keyMap        SSOLoginKeyMap
	width         int
	height        int
	err           error
	expiresAt     time.Time
	integrationID string

	// Callbacks
	onSuccess func(string, *provider.SSOToken) tea.Cmd // integrationID, token
	onCancel  func() tea.Cmd
	onOpenURL func(string) tea.Cmd
	onCopyURL func(string) tea.Cmd
}

// NewSSOLoginView creates a new SSO login view.
func NewSSOLoginView(theme *styles.Theme) *SSOLoginView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.InfoText

	return &SSOLoginView{
		state:   SSOStateIdle,
		spinner: s,
		theme:   theme,
		keyMap:  DefaultSSOLoginKeyMap(),
	}
}

// SetStartURL sets the SSO start URL.
func (v *SSOLoginView) SetStartURL(startURL, region, integrationID string) {
	v.startURL = startURL
	v.region = region
	v.integrationID = integrationID
	v.state = SSOStateIdle
	v.err = nil
	v.deviceAuth = nil
}

// SetDeviceAuthorization sets the device authorization response.
func (v *SSOLoginView) SetDeviceAuthorization(auth *provider.DeviceAuthorizationResponse) {
	v.deviceAuth = auth
	v.state = SSOStateWaitingForUser
	v.expiresAt = time.Now().Add(time.Duration(auth.ExpiresIn) * time.Second)
}

// SetPolling marks the view as polling for token.
func (v *SSOLoginView) SetPolling() {
	v.state = SSOStatePolling
}

// SetSuccess marks the login as successful.
func (v *SSOLoginView) SetSuccess() {
	v.state = SSOStateSuccess
}

// SetError sets an error state.
func (v *SSOLoginView) SetError(err error) {
	v.state = SSOStateError
	v.err = err
}

// SetSize sets the view size.
func (v *SSOLoginView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnSuccess sets the callback for successful login.
func (v *SSOLoginView) SetOnSuccess(fn func(string, *provider.SSOToken) tea.Cmd) {
	v.onSuccess = fn
}

// SetOnCancel sets the callback for cancellation.
func (v *SSOLoginView) SetOnCancel(fn func() tea.Cmd) {
	v.onCancel = fn
}

// SetOnOpenURL sets the callback for opening URL.
func (v *SSOLoginView) SetOnOpenURL(fn func(string) tea.Cmd) {
	v.onOpenURL = fn
}

// SetOnCopyURL sets the callback for copying URL.
func (v *SSOLoginView) SetOnCopyURL(fn func(string) tea.Cmd) {
	v.onCopyURL = fn
}

// GetDeviceAuth returns the device authorization response.
func (v *SSOLoginView) GetDeviceAuth() *provider.DeviceAuthorizationResponse {
	return v.deviceAuth
}

// GetIntegrationID returns the integration ID.
func (v *SSOLoginView) GetIntegrationID() string {
	return v.integrationID
}

// Update handles input for the SSO login view.
func (v *SSOLoginView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Cancel):
			if v.onCancel != nil {
				return v, v.onCancel()
			}
		case key.Matches(msg, v.keyMap.OpenBrowser):
			if v.deviceAuth != nil && v.onOpenURL != nil {
				url := v.deviceAuth.VerificationURIComplete
				if url == "" {
					url = v.deviceAuth.VerificationURI
				}
				return v, v.onOpenURL(url)
			}
		case key.Matches(msg, v.keyMap.CopyURL):
			if v.deviceAuth != nil && v.onCopyURL != nil {
				url := v.deviceAuth.VerificationURIComplete
				if url == "" {
					url = v.deviceAuth.VerificationURI
				}
				return v, v.onCopyURL(url)
			}
		}

	case spinner.TickMsg:
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd
	}

	return v, nil
}

// View renders the SSO login view.
func (v *SSOLoginView) View() string {
	var b strings.Builder

	b.WriteString(v.theme.Title.Render("AWS SSO Login"))
	b.WriteString("\n\n")

	switch v.state {
	case SSOStateIdle, SSOStateStarting:
		b.WriteString(v.spinner.View())
		b.WriteString(" Starting device authorization...")

	case SSOStateWaitingForUser:
		b.WriteString(v.theme.Subtitle.Render("Please authorize in your browser"))
		b.WriteString("\n\n")

		// Verification URL
		url := v.deviceAuth.VerificationURIComplete
		if url == "" {
			url = v.deviceAuth.VerificationURI
		}
		b.WriteString(v.theme.Label.Render("URL: "))
		b.WriteString(v.theme.Value.Render(url))
		b.WriteString("\n\n")

		// User code
		if v.deviceAuth.UserCode != "" {
			b.WriteString(v.theme.Label.Render("Code: "))
			b.WriteString(v.theme.Title.Render(v.deviceAuth.UserCode))
			b.WriteString("\n\n")
		}

		// Time remaining
		remaining := time.Until(v.expiresAt)
		if remaining > 0 {
			b.WriteString(v.theme.Label.Render("Expires in: "))
			b.WriteString(v.theme.Value.Render(fmt.Sprintf("%d seconds", int(remaining.Seconds()))))
		} else {
			b.WriteString(v.theme.ErrorText.Render("Authorization expired"))
		}
		b.WriteString("\n\n")

		b.WriteString(v.theme.InfoText.Render("Waiting for authorization..."))
		b.WriteString(" ")
		b.WriteString(v.spinner.View())

	case SSOStatePolling:
		b.WriteString(v.spinner.View())
		b.WriteString(" Waiting for authorization to complete...")

	case SSOStateSuccess:
		b.WriteString(v.theme.SuccessText.Render("✓ Successfully logged in!"))

	case SSOStateError:
		b.WriteString(v.theme.ErrorText.Render("✗ Login failed"))
		if v.err != nil {
			b.WriteString("\n\n")
			b.WriteString(v.theme.ErrorText.Render(v.err.Error()))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(v.theme.Footer.Render("o: open browser • c: copy URL • esc: cancel"))

	return v.theme.Dialog.Width(70).Render(b.String())
}

// Init initializes the view.
func (v *SSOLoginView) Init() tea.Cmd {
	return v.spinner.Tick
}

// SpinnerTick returns a command to tick the spinner.
func (v *SSOLoginView) SpinnerTick() tea.Cmd {
	return v.spinner.Tick
}

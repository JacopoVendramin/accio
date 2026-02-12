package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/tui/components"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// SessionDetailKeyMap defines key bindings for the session detail view.
type SessionDetailKeyMap struct {
	Back  key.Binding
	Start key.Binding
	Stop  key.Binding
	Edit  key.Binding
	Copy  key.Binding
	Quit  key.Binding
}

// DefaultSessionDetailKeyMap returns the default key bindings.
func DefaultSessionDetailKeyMap() SessionDetailKeyMap {
	return SessionDetailKeyMap{
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy profile"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// SessionDetailView shows details of a single session.
type SessionDetailView struct {
	session *session.Session
	helpBar *components.HelpBar
	theme   *styles.Theme
	keyMap  SessionDetailKeyMap
	width   int
	height  int

	// Callbacks
	onBack  func() tea.Cmd
	onStart func(*session.Session) tea.Cmd
	onStop  func(*session.Session) tea.Cmd
	onEdit  func(*session.Session) tea.Cmd
	onCopy  func(*session.Session) tea.Cmd
}

// NewSessionDetailView creates a new session detail view.
func NewSessionDetailView(theme *styles.Theme) *SessionDetailView {
	helpBar := components.NewHelpBar(theme)
	helpBar.SetBindings(components.DefaultDetailViewBindings())

	return &SessionDetailView{
		helpBar: helpBar,
		theme:   theme,
		keyMap:  DefaultSessionDetailKeyMap(),
	}
}

// SetSession sets the session to display.
func (v *SessionDetailView) SetSession(sess *session.Session) {
	v.session = sess
}

// SetSize sets the view size.
func (v *SessionDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnBack sets the callback for back action.
func (v *SessionDetailView) SetOnBack(fn func() tea.Cmd) {
	v.onBack = fn
}

// SetOnStart sets the callback for start action.
func (v *SessionDetailView) SetOnStart(fn func(*session.Session) tea.Cmd) {
	v.onStart = fn
}

// SetOnStop sets the callback for stop action.
func (v *SessionDetailView) SetOnStop(fn func(*session.Session) tea.Cmd) {
	v.onStop = fn
}

// SetOnEdit sets the callback for edit action.
func (v *SessionDetailView) SetOnEdit(fn func(*session.Session) tea.Cmd) {
	v.onEdit = fn
}

// SetOnCopy sets the callback for copy action.
func (v *SessionDetailView) SetOnCopy(fn func(*session.Session) tea.Cmd) {
	v.onCopy = fn
}

// Update handles input for the detail view.
func (v *SessionDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Back):
			if v.onBack != nil {
				return v, v.onBack()
			}
		case key.Matches(msg, v.keyMap.Start):
			if v.onStart != nil && v.session != nil {
				return v, v.onStart(v.session)
			}
		case key.Matches(msg, v.keyMap.Stop):
			if v.onStop != nil && v.session != nil {
				return v, v.onStop(v.session)
			}
		case key.Matches(msg, v.keyMap.Edit):
			if v.onEdit != nil && v.session != nil {
				return v, v.onEdit(v.session)
			}
		case key.Matches(msg, v.keyMap.Copy):
			if v.onCopy != nil && v.session != nil {
				return v, v.onCopy(v.session)
			}
		case key.Matches(msg, v.keyMap.Quit):
			return v, tea.Quit
		}
	}
	return v, nil
}

// View renders the session detail view.
func (v *SessionDetailView) View() string {
	if v.session == nil {
		return v.theme.Subtitle.Render("No session selected")
	}

	var b strings.Builder

	// Title
	b.WriteString(v.theme.Title.Render(v.session.Name))
	b.WriteString("\n\n")

	// Status badge
	b.WriteString(components.StatusBadge(string(v.session.Status), v.theme))
	b.WriteString("\n\n")

	// Details
	b.WriteString(v.renderField("Profile", v.session.ProfileName))
	b.WriteString(v.renderField("Type", formatSessionTypeFull(v.session.Type)))
	b.WriteString(v.renderField("Region", v.session.Region))
	b.WriteString(v.renderField("Provider", string(v.session.Provider)))

	// Type-specific details
	switch v.session.Type {
	case session.SessionTypeIAMUser:
		if cfg := v.session.Config.IAMUser; cfg != nil {
			b.WriteString("\n")
			b.WriteString(v.theme.Subtitle.Render("IAM User Configuration"))
			b.WriteString("\n")
			b.WriteString(v.renderField("Access Key ID", maskString(cfg.AccessKeyID)))
			if cfg.MFASerial != "" {
				b.WriteString(v.renderField("MFA Device", cfg.MFASerial))
			}
		}
	case session.SessionTypeAWSSSO:
		if cfg := v.session.Config.AWSSSO; cfg != nil {
			b.WriteString("\n")
			b.WriteString(v.theme.Subtitle.Render("AWS SSO Configuration"))
			b.WriteString("\n")
			b.WriteString(v.renderField("Start URL", cfg.StartURL))
			b.WriteString(v.renderField("Account ID", cfg.AccountID))
			b.WriteString(v.renderField("Role Name", cfg.RoleName))
		}
	case session.SessionTypeIAMRole:
		if cfg := v.session.Config.IAMRole; cfg != nil {
			b.WriteString("\n")
			b.WriteString(v.theme.Subtitle.Render("IAM Role Configuration"))
			b.WriteString("\n")
			b.WriteString(v.renderField("Role ARN", cfg.RoleARN))
			b.WriteString(v.renderField("Parent Session", cfg.ParentSessionID))
		}
	}

	// Expiry info for active sessions
	if v.session.Status == session.StatusActive && !v.session.ExpiresAt().IsZero() {
		b.WriteString("\n")
		remaining := v.session.TimeUntilExpiry()
		percent := remaining.Seconds() / float64(v.session.GetSessionDuration())
		b.WriteString(v.renderField("Expires In", formatDurationFull(remaining)))
		b.WriteString(components.ProgressBar(percent, 30, v.theme))
		b.WriteString("\n")
	}

	// Metadata
	if !v.session.Metadata.CreatedAt.IsZero() {
		b.WriteString("\n")
		b.WriteString(v.theme.Subtitle.Render("Metadata"))
		b.WriteString("\n")
		b.WriteString(v.renderField("Created", v.session.Metadata.CreatedAt.Format("2006-01-02 15:04")))
		if !v.session.Metadata.LastUsedAt.IsZero() {
			b.WriteString(v.renderField("Last Used", v.session.Metadata.LastUsedAt.Format("2006-01-02 15:04")))
		}
	}

	// Help bar
	b.WriteString("\n\n")
	b.WriteString(v.theme.Footer.Render(v.helpBar.View()))

	return b.String()
}

// Init initializes the view.
func (v *SessionDetailView) Init() tea.Cmd {
	return nil
}

// renderField renders a label-value pair.
func (v *SessionDetailView) renderField(label, value string) string {
	return fmt.Sprintf("%s %s\n",
		v.theme.Label.Render(label+":"),
		v.theme.Value.Render(value),
	)
}

// formatSessionTypeFull returns a full string for the session type.
func formatSessionTypeFull(t session.SessionType) string {
	switch t {
	case session.SessionTypeIAMUser:
		return "IAM User"
	case session.SessionTypeAWSSSO:
		return "AWS SSO"
	case session.SessionTypeIAMRole:
		return "IAM Role (Chained)"
	case session.SessionTypeSAML:
		return "SAML Federation"
	default:
		return string(t)
	}
}

// formatDurationFull formats a duration with full units.
func formatDurationFull(d interface{}) string {
	// Handle time.Duration
	switch dur := d.(type) {
	case interface{ Minutes() float64 }:
		mins := int(dur.Minutes())
		if mins < 1 {
			return "< 1 minute"
		}
		hours := mins / 60
		mins = mins % 60
		if hours == 0 {
			return fmt.Sprintf("%d minutes", mins)
		}
		if mins == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%d hours %d minutes", hours, mins)
	default:
		return fmt.Sprintf("%v", d)
	}
}

// maskString masks a string, showing only the first and last few characters.
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

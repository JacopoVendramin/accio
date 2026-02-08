package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// SessionList is a component for displaying a list of sessions.
type SessionList struct {
	sessions []*session.Session
	cursor   int
	theme    *styles.Theme
	width    int
	height   int
	showTimestamps bool
	showRegion     bool
}

// NewSessionList creates a new session list component.
func NewSessionList(theme *styles.Theme) *SessionList {
	return &SessionList{
		theme:          theme,
		showTimestamps: true,
		showRegion:     true,
	}
}

// SetSessions sets the sessions to display.
func (l *SessionList) SetSessions(sessions []*session.Session) {
	l.sessions = sessions
	if l.cursor >= len(sessions) {
		l.cursor = max(0, len(sessions)-1)
	}
}

// SetSize sets the component size.
func (l *SessionList) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetShowTimestamps sets whether to show timestamps.
func (l *SessionList) SetShowTimestamps(show bool) {
	l.showTimestamps = show
}

// SetShowRegion sets whether to show region.
func (l *SessionList) SetShowRegion(show bool) {
	l.showRegion = show
}

// MoveUp moves the cursor up.
func (l *SessionList) MoveUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

// MoveDown moves the cursor down.
func (l *SessionList) MoveDown() {
	if l.cursor < len(l.sessions)-1 {
		l.cursor++
	}
}

// Selected returns the currently selected session.
func (l *SessionList) Selected() *session.Session {
	if l.cursor < 0 || l.cursor >= len(l.sessions) {
		return nil
	}
	return l.sessions[l.cursor]
}

// SelectedIndex returns the index of the currently selected session.
func (l *SessionList) SelectedIndex() int {
	return l.cursor
}

// View renders the session list.
func (l *SessionList) View() string {
	if len(l.sessions) == 0 {
		return l.theme.Subtitle.Render("No sessions configured. Press 'n' to create one.")
	}

	var b strings.Builder
	for i, sess := range l.sessions {
		isSelected := i == l.cursor
		line := l.renderSession(sess, isSelected)
		b.WriteString(line)
		if i < len(l.sessions)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderSession renders a single session item.
func (l *SessionList) renderSession(sess *session.Session, selected bool) string {
	// Status icon
	statusIcon := styles.StatusIcon(string(sess.Status))
	statusStyle := l.theme.StatusStyle(string(sess.Status))

	// Build the line
	var parts []string

	// Status icon
	parts = append(parts, statusStyle.Render(statusIcon))

	// Session name
	nameStyle := l.theme.SessionName
	if selected {
		nameStyle = nameStyle.Bold(true)
	}
	parts = append(parts, nameStyle.Render(sess.Name))

	// Profile name
	profileStyle := l.theme.SessionProfile
	parts = append(parts, profileStyle.Render(fmt.Sprintf("(%s)", sess.ProfileName)))

	// Region (optional)
	if l.showRegion && sess.Region != "" {
		regionStyle := l.theme.SessionRegion
		parts = append(parts, regionStyle.Render(sess.Region))
	}

	// Session type
	typeStyle := l.theme.SessionStatus
	parts = append(parts, typeStyle.Render(formatSessionType(sess.Type)))

	// Expiry time for active sessions
	if sess.Status == session.StatusActive && !sess.ExpiresAt().IsZero() {
		remaining := sess.TimeUntilExpiry()
		expiryStr := formatDuration(remaining)
		var expiryStyle lipgloss.Style
		if remaining < 5*time.Minute {
			expiryStyle = l.theme.WarningText
		} else {
			expiryStyle = l.theme.SessionStatus
		}
		parts = append(parts, expiryStyle.Render(expiryStr))
	}

	line := strings.Join(parts, " ")

	// Apply selection style
	if selected {
		return l.theme.SessionItemSelected.Render("▶ " + line)
	}
	return l.theme.SessionItem.Render("  " + line)
}

// formatSessionType returns a short string for the session type.
func formatSessionType(t session.SessionType) string {
	switch t {
	case session.SessionTypeIAMUser:
		return "[IAM]"
	case session.SessionTypeAWSSSO:
		return "[SSO]"
	case session.SessionTypeIAMRole:
		return "[Role]"
	case session.SessionTypeSAML:
		return "[SAML]"
	default:
		return "[?]"
	}
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

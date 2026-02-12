// Package views provides TUI views for the application.
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/tui/components"
	"github.com/jvendramin/accio/internal/tui/styles"
)

// SessionListKeyMap defines the key bindings for the session list view.
type SessionListKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	New     key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Search  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultSessionListKeyMap returns the default key bindings.
func DefaultSessionListKeyMap() SessionListKeyMap {
	return SessionListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "connect"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new integration"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// SessionListView is the main session list view.
type SessionListView struct {
	helpBar *components.HelpBar
	theme   *styles.Theme
	keyMap  SessionListKeyMap
	width   int
	height  int

	// Data
	allSessions      []*session.Session
	filteredSessions []*session.Session
	integrations     []*integration.Integration
	cursor           int
	scrollOffset     int // Track scroll position for viewport

	// Search
	searchInput  textinput.Model
	searchActive bool

	// Callbacks
	onStartStop      func(*session.Session) tea.Cmd
	onNewIntegration func() tea.Cmd
	onDelete         func(*session.Session) tea.Cmd
	onRefresh        func() tea.Cmd
}

// NewSessionListView creates a new session list view.
func NewSessionListView(theme *styles.Theme) *SessionListView {
	helpBar := components.NewHelpBar(theme)
	helpBar.SetBindings([]components.KeyBinding{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "connect"},
		{Key: "/", Desc: "search"},
		{Key: "i", Desc: "integrations"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	})

	searchInput := textinput.New()
	searchInput.Placeholder = "Search sessions..."
	searchInput.CharLimit = 50

	return &SessionListView{
		helpBar:     helpBar,
		theme:       theme,
		keyMap:      DefaultSessionListKeyMap(),
		searchInput: searchInput,
	}
}

// SetSessions sets the sessions to display.
func (v *SessionListView) SetSessions(sessions []*session.Session) {
	v.allSessions = sessions
	v.filterSessions()
}

// SetIntegrations sets the integrations for grouping headers.
func (v *SessionListView) SetIntegrations(integrations []*integration.Integration) {
	v.integrations = integrations
}

// filterSessions filters sessions based on search query.
func (v *SessionListView) filterSessions() {
	query := strings.ToLower(strings.TrimSpace(v.searchInput.Value()))
	if query == "" {
		v.filteredSessions = v.allSessions
	} else {
		var filtered []*session.Session
		for _, sess := range v.allSessions {
			searchText := sess.Name + " " + sess.ProfileName + " " + sess.Region
			if sess.Config.AWSSSO != nil {
				searchText += " " + sess.Config.AWSSSO.AccountID + " " + sess.Config.AWSSSO.AccountEmail
			}
			if strings.Contains(strings.ToLower(searchText), query) {
				filtered = append(filtered, sess)
			}
		}
		v.filteredSessions = filtered
	}
	// Reset cursor if out of bounds
	if v.cursor >= len(v.filteredSessions) {
		v.cursor = max(0, len(v.filteredSessions)-1)
	}
}

// SetSize sets the view size.
func (v *SessionListView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// SetOnStartStop sets the callback for start/stop action.
func (v *SessionListView) SetOnStartStop(fn func(*session.Session) tea.Cmd) {
	v.onStartStop = fn
}

// SetOnNewIntegration sets the callback for new integration action.
func (v *SessionListView) SetOnNewIntegration(fn func() tea.Cmd) {
	v.onNewIntegration = fn
}

// SetOnDelete sets the callback for delete action.
func (v *SessionListView) SetOnDelete(fn func(*session.Session) tea.Cmd) {
	v.onDelete = fn
}

// SetOnRefresh sets the callback for refresh action.
func (v *SessionListView) SetOnRefresh(fn func() tea.Cmd) {
	v.onRefresh = fn
}

// Selected returns the currently selected session.
func (v *SessionListView) Selected() *session.Session {
	if v.cursor < 0 || v.cursor >= len(v.filteredSessions) {
		return nil
	}
	return v.filteredSessions[v.cursor]
}

// SelectSessionByID selects a session by its ID after a reload.
func (v *SessionListView) SelectSessionByID(id string) {
	if id == "" {
		return
	}
	for i, sess := range v.filteredSessions {
		if sess.ID == id {
			v.cursor = i
			return
		}
	}
}

// IsSearchActive returns whether search mode is active.
func (v *SessionListView) IsSearchActive() bool {
	return v.searchActive
}

// Update handles input for the session list view.
func (v *SessionListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode
		if v.searchActive {
			switch msg.String() {
			case "esc":
				v.searchActive = false
				v.searchInput.Blur()
				v.searchInput.SetValue("")
				v.filterSessions()
				return v, nil
			case "enter":
				v.searchActive = false
				v.searchInput.Blur()
				return v, nil
			default:
				v.searchInput, cmd = v.searchInput.Update(msg)
				v.filterSessions()
				return v, cmd
			}
		}

		// Normal mode
		switch {
		case key.Matches(msg, v.keyMap.Search):
			v.searchActive = true
			v.searchInput.Focus()
			return v, textinput.Blink
		case key.Matches(msg, v.keyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, v.keyMap.Down):
			if v.cursor < len(v.filteredSessions)-1 {
				v.cursor++
			}
		case key.Matches(msg, v.keyMap.Enter):
			if v.onStartStop != nil {
				if sess := v.Selected(); sess != nil {
					return v, v.onStartStop(sess)
				}
			}
		case key.Matches(msg, v.keyMap.Delete):
			if v.onDelete != nil {
				if sess := v.Selected(); sess != nil {
					return v, v.onDelete(sess)
				}
			}
		case key.Matches(msg, v.keyMap.Refresh):
			if v.onRefresh != nil {
				return v, v.onRefresh()
			}
		case key.Matches(msg, v.keyMap.Quit):
			return v, tea.Quit
		}
	case tea.MouseMsg:
		// Handle mouse wheel scrolling
		if !v.searchActive {
			switch msg.Type {
			case tea.MouseWheelUp:
				if v.cursor > 0 {
					v.cursor--
				}
			case tea.MouseWheelDown:
				if v.cursor < len(v.filteredSessions)-1 {
					v.cursor++
				}
			}
		}
	}
	return v, nil
}

// View renders the session list view.
func (v *SessionListView) View() string {
	var b strings.Builder

	// Header with summary statistics
	activeSessions := 0
	for _, sess := range v.allSessions {
		if sess.Status == session.StatusActive {
			activeSessions++
		}
	}

	// Title and stats
	title := v.theme.Title.Render("AWS Sessions")
	stats := v.theme.Subtitle.Render(fmt.Sprintf(
		"%d total • %d active • %d integrations",
		len(v.allSessions),
		activeSessions,
		len(v.integrations),
	))

	b.WriteString(title)
	b.WriteString("  ")
	b.WriteString(stats)
	b.WriteString("\n\n")

	// Calculate available height for content (subtract header, footer, padding)
	availableHeight := v.height - 7 // Reserve more space for header

	// Search bar (only show when active or has value)
	headerLines := 0
	if v.searchActive {
		b.WriteString(v.theme.Label.Render("Search: "))
		b.WriteString(v.searchInput.View())
		b.WriteString("\n\n")
		headerLines = 2
	} else if v.searchInput.Value() != "" {
		b.WriteString(v.theme.Subtitle.Render(fmt.Sprintf("Filter: %s", v.searchInput.Value())))
		b.WriteString("\n\n")
		headerLines = 2
	}

	availableHeight -= headerLines

	// Sessions grouped by integration
	if len(v.filteredSessions) == 0 {
		if len(v.allSessions) == 0 {
			b.WriteString(v.theme.Subtitle.Render("No sessions configured."))
			b.WriteString("\n")
			b.WriteString(v.theme.InfoText.Render("Press 'i' to add an integration."))
		} else {
			b.WriteString(v.theme.Subtitle.Render("No sessions match your search."))
		}
	} else {
		// Adjust scroll offset to keep cursor visible
		if v.cursor < v.scrollOffset {
			v.scrollOffset = v.cursor
		} else if v.cursor >= v.scrollOffset+availableHeight {
			v.scrollOffset = v.cursor - availableHeight + 1
		}

		// Group sessions by integration
		groups := v.groupSessionsByIntegration()
		currentIndex := 0
		renderedLines := 0

		for i, group := range groups {
			// Check if we should render spacing between groups
			if i > 0 && currentIndex > v.scrollOffset && renderedLines < availableHeight {
				b.WriteString("\n")
				renderedLines++
			}

			// Render integration header if visible
			groupStartIndex := currentIndex
			if groupStartIndex <= v.scrollOffset+availableHeight && groupStartIndex+len(group.Sessions) > v.scrollOffset {
				if currentIndex >= v.scrollOffset && renderedLines < availableHeight {
					b.WriteString(v.theme.Label.Render(group.Name))
					b.WriteString("\n")
					renderedLines++
				}
			}

			for _, sess := range group.Sessions {
				// Only render if within visible viewport
				if currentIndex >= v.scrollOffset && renderedLines < availableHeight {
					isSelected := v.cursor == currentIndex
					b.WriteString(v.renderSession(sess, isSelected))
					b.WriteString("\n")
					renderedLines++
				}
				currentIndex++
			}
		}
	}

	// Footer with help
	b.WriteString("\n")
	if v.searchActive {
		b.WriteString(v.theme.Footer.Render("enter: apply • esc: cancel"))
	} else {
		b.WriteString(v.theme.Footer.Render(v.helpBar.View()))
	}

	return b.String()
}

// SessionGroup represents a group of sessions under an integration.
type SessionGroup struct {
	Name     string
	Sessions []*session.Session
}

// groupSessionsByIntegration groups sessions by their integration.
func (v *SessionListView) groupSessionsByIntegration() []SessionGroup {
	// Build integration ID to name map
	integrationNames := make(map[string]string)
	for _, integ := range v.integrations {
		integrationNames[integ.ID] = integ.Name
	}

	groups := make(map[string][]*session.Session)
	order := []string{}

	for _, sess := range v.filteredSessions {
		var groupKey string
		var groupName string

		if sess.Config.AWSSSO != nil && sess.Config.AWSSSO.IntegrationID != "" {
			groupKey = sess.Config.AWSSSO.IntegrationID
			if name, ok := integrationNames[groupKey]; ok {
				groupName = name
			} else {
				groupName = sess.Config.AWSSSO.StartURL
			}
		} else if sess.Type == session.SessionTypeIAMUser {
			groupKey = "iam-users"
			groupName = "IAM Users"
		} else if sess.Type == session.SessionTypeIAMRole {
			groupKey = "iam-roles"
			groupName = "IAM Roles"
		} else {
			groupKey = "other"
			groupName = "Other"
		}

		if _, exists := groups[groupKey]; !exists {
			order = append(order, groupKey)
			// Store the name with the key for later retrieval
			groups[groupKey+"_name"] = nil
		}
		groups[groupKey] = append(groups[groupKey], sess)
		// Hacky way to store name - use first session's group name
		if groups[groupKey+"_name"] == nil {
			integrationNames[groupKey] = groupName
		}
	}

	var result []SessionGroup
	for _, key := range order {
		result = append(result, SessionGroup{
			Name:     integrationNames[key],
			Sessions: groups[key],
		})
	}
	return result
}

// renderSession renders a single session.
func (v *SessionListView) renderSession(sess *session.Session, selected bool) string {
	// Status icon
	var statusIcon string
	var statusStyle = v.theme.StatusInactive
	if sess.Status == session.StatusActive {
		statusIcon = "●"
		statusStyle = v.theme.StatusActive
	} else {
		statusIcon = "○"
	}

	// Cursor
	cursor := " "
	if selected {
		cursor = "▶"
	}

	// Build the line: cursor, status, name, profile, region
	line := fmt.Sprintf("  %s %s %s", cursor, statusStyle.Render(statusIcon), sess.Name)

	// Add profile and region info (without extra margin)
	extras := []string{}
	if sess.ProfileName != "" {
		extras = append(extras, sess.ProfileName)
	}
	if sess.Region != "" {
		extras = append(extras, sess.Region)
	}
	if len(extras) > 0 {
		// Use inline style without margin
		subtitleStyle := v.theme.Subtitle.Copy().MarginBottom(0)
		line += " " + subtitleStyle.Render("("+strings.Join(extras, " • ")+")")
	}

	// Apply bold if selected
	if selected {
		line = v.theme.SessionItemSelected.Render(line)
	}

	return line
}

// Init initializes the view.
func (v *SessionListView) Init() tea.Cmd {
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

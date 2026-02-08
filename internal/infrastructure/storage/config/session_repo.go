// Package config provides file-based storage for sessions and integrations.
package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/session"
)

// SessionRepository implements session.Repository using YAML file storage.
type SessionRepository struct {
	filePath string
	mu       sync.RWMutex
	cache    map[string]*session.Session
}

// sessionData represents the YAML structure for sessions.
type sessionData struct {
	Sessions []*session.Session `yaml:"sessions"`
}

// NewSessionRepository creates a new file-based session repository.
func NewSessionRepository(filePath string) (*SessionRepository, error) {
	repo := &SessionRepository{
		filePath: filePath,
		cache:    make(map[string]*session.Session),
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return nil, err
	}

	// Load existing sessions
	if err := repo.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return repo, nil
}

// Save persists a session.
func (r *SessionRepository) Save(ctx context.Context, sess *session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[sess.ID] = sess
	return r.persist()
}

// Get retrieves a session by ID.
func (r *SessionRepository) Get(ctx context.Context, id string) (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sess, ok := r.cache[id]
	if !ok {
		return nil, domainErrors.NewDomainError("SessionRepository.Get", domainErrors.ErrSessionNotFound, nil).
			WithContext("id", id)
	}

	return sess, nil
}

// GetByProfileName retrieves a session by AWS profile name.
func (r *SessionRepository) GetByProfileName(ctx context.Context, profileName string) (*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, sess := range r.cache {
		if sess.ProfileName == profileName {
			return sess, nil
		}
	}

	return nil, domainErrors.NewDomainError("SessionRepository.GetByProfileName", domainErrors.ErrSessionNotFound, nil).
		WithContext("profile", profileName)
}

// List returns all sessions.
func (r *SessionRepository) List(ctx context.Context) ([]*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*session.Session, 0, len(r.cache))
	for _, sess := range r.cache {
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// ListByProvider returns sessions for a specific provider.
func (r *SessionRepository) ListByProvider(ctx context.Context, provider session.Provider) ([]*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sessions []*session.Session
	for _, sess := range r.cache {
		if sess.Provider == provider {
			sessions = append(sessions, sess)
		}
	}

	return sessions, nil
}

// ListByStatus returns sessions with a specific status.
func (r *SessionRepository) ListByStatus(ctx context.Context, status session.SessionStatus) ([]*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sessions []*session.Session
	for _, sess := range r.cache {
		if sess.Status == status {
			sessions = append(sessions, sess)
		}
	}

	return sessions, nil
}

// Delete removes a session.
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.cache[id]; !ok {
		return domainErrors.NewDomainError("SessionRepository.Delete", domainErrors.ErrSessionNotFound, nil).
			WithContext("id", id)
	}

	delete(r.cache, id)
	return r.persist()
}

// GetChildren returns sessions that depend on the given parent session.
func (r *SessionRepository) GetChildren(ctx context.Context, parentID string) ([]*session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var children []*session.Session
	for _, sess := range r.cache {
		if sess.Type == session.SessionTypeIAMRole &&
			sess.Config.IAMRole != nil &&
			sess.Config.IAMRole.ParentSessionID == parentID {
			children = append(children, sess)
		}
	}

	return children, nil
}

// load reads sessions from the file.
func (r *SessionRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var sd sessionData
	if err := yaml.Unmarshal(data, &sd); err != nil {
		return err
	}

	r.cache = make(map[string]*session.Session)
	for _, sess := range sd.Sessions {
		// Reset runtime state on load
		sess.Status = session.StatusInactive
		r.cache[sess.ID] = sess
	}

	return nil
}

// persist writes sessions to the file.
func (r *SessionRepository) persist() error {
	sessions := make([]*session.Session, 0, len(r.cache))
	for _, sess := range r.cache {
		sessions = append(sessions, sess)
	}

	sd := sessionData{Sessions: sessions}
	data, err := yaml.Marshal(&sd)
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0600)
}

// Reload reloads sessions from the file.
func (r *SessionRepository) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.load()
}

package session

import "context"

// Repository defines the interface for session persistence.
type Repository interface {
	// Save persists a session.
	Save(ctx context.Context, session *Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*Session, error)

	// GetByProfileName retrieves a session by AWS profile name.
	GetByProfileName(ctx context.Context, profileName string) (*Session, error)

	// List returns all sessions.
	List(ctx context.Context) ([]*Session, error)

	// ListByProvider returns sessions for a specific provider.
	ListByProvider(ctx context.Context, provider Provider) ([]*Session, error)

	// ListByStatus returns sessions with a specific status.
	ListByStatus(ctx context.Context, status SessionStatus) ([]*Session, error)

	// Delete removes a session.
	Delete(ctx context.Context, id string) error

	// GetChildren returns sessions that depend on the given parent session.
	GetChildren(ctx context.Context, parentID string) ([]*Session, error)
}

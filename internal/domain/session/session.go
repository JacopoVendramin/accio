package session

import (
	"time"

	"github.com/google/uuid"
	"github.com/jvendramin/accio/internal/domain/credential"
	"github.com/jvendramin/accio/internal/domain/errors"
)

// Session represents a cloud credentials session.
type Session struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Provider     Provider        `json:"provider"`
	Type         SessionType     `json:"type"`
	Status       SessionStatus   `json:"status"`
	ProfileName  string          `json:"profile_name"`
	Region       string          `json:"region"`
	Config       SessionConfig   `json:"config"`
	Metadata     SessionMetadata `json:"metadata"`

	// Runtime state (not persisted)
	credential   *credential.Credential
	lastError    error
	expiresAt    time.Time
}

// NewSession creates a new session with the given parameters.
func NewSession(name string, provider Provider, sessionType SessionType, profileName, region string) *Session {
	now := time.Now()
	return &Session{
		ID:          uuid.New().String(),
		Name:        name,
		Provider:    provider,
		Type:        sessionType,
		Status:      StatusInactive,
		ProfileName: profileName,
		Region:      region,
		Metadata: SessionMetadata{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// Validate checks if the session configuration is valid.
func (s *Session) Validate() error {
	if s.Name == "" {
		return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
			WithContext("field", "name")
	}
	if s.ProfileName == "" {
		return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
			WithContext("field", "profile_name")
	}

	switch s.Type {
	case SessionTypeIAMUser:
		if s.Config.IAMUser == nil || s.Config.IAMUser.AccessKeyID == "" {
			return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
				WithContext("field", "iam_user.access_key_id")
		}
	case SessionTypeAWSSSO:
		if s.Config.AWSSSO == nil {
			return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
				WithContext("field", "aws_sso")
		}
		if s.Config.AWSSSO.StartURL == "" || s.Config.AWSSSO.AccountID == "" || s.Config.AWSSSO.RoleName == "" {
			return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
				WithContext("field", "aws_sso config incomplete")
		}
	case SessionTypeIAMRole:
		if s.Config.IAMRole == nil || s.Config.IAMRole.ParentSessionID == "" || s.Config.IAMRole.RoleARN == "" {
			return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
				WithContext("field", "iam_role")
		}
	case SessionTypeSAML:
		if s.Config.SAML == nil || s.Config.SAML.IntegrationID == "" {
			return errors.NewDomainError("Session.Validate", errors.ErrInvalidConfig, nil).
				WithContext("field", "saml")
		}
	default:
		return errors.NewDomainError("Session.Validate", errors.ErrInvalidSessionType, nil).
			WithContext("type", string(s.Type))
	}

	return nil
}

// Start marks the session as active with the given credential.
func (s *Session) Start(cred *credential.Credential) error {
	if s.Status == StatusActive {
		return errors.NewDomainError("Session.Start", errors.ErrSessionAlreadyActive, nil)
	}

	s.credential = cred
	s.Status = StatusActive
	s.expiresAt = cred.Expiration
	s.lastError = nil
	s.Metadata.UpdatedAt = time.Now()
	s.Metadata.LastUsedAt = time.Now()

	return nil
}

// Stop marks the session as inactive.
func (s *Session) Stop() error {
	s.credential = nil
	s.Status = StatusInactive
	s.expiresAt = time.Time{}
	s.lastError = nil
	s.Metadata.UpdatedAt = time.Now()

	return nil
}

// SetError marks the session as having an error.
func (s *Session) SetError(err error) {
	s.Status = StatusError
	s.lastError = err
	s.Metadata.UpdatedAt = time.Now()
}

// SetPending marks the session as pending user action.
func (s *Session) SetPending() {
	s.Status = StatusPending
	s.Metadata.UpdatedAt = time.Now()
}

// GetCredential returns the current credential.
func (s *Session) GetCredential() *credential.Credential {
	return s.credential
}

// SetCredential sets the credential (used when loading from storage).
func (s *Session) SetCredential(cred *credential.Credential) {
	s.credential = cred
	if cred != nil {
		s.expiresAt = cred.Expiration
	}
}

// GetLastError returns the last error that occurred.
func (s *Session) GetLastError() error {
	return s.lastError
}

// IsActive returns true if the session is active.
func (s *Session) IsActive() bool {
	return s.Status == StatusActive
}

// NeedsRefresh returns true if the credential needs to be refreshed.
func (s *Session) NeedsRefresh(refreshBefore time.Duration) bool {
	if s.credential == nil {
		return true
	}
	return s.credential.IsExpiringSoon(refreshBefore)
}

// ExpiresAt returns when the session credential expires.
func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

// TimeUntilExpiry returns the duration until the session expires.
func (s *Session) TimeUntilExpiry() time.Duration {
	if s.expiresAt.IsZero() {
		return 0
	}
	return time.Until(s.expiresAt)
}

// RequiresMFA returns true if this session requires MFA.
func (s *Session) RequiresMFA() bool {
	if s.Type == SessionTypeIAMUser && s.Config.IAMUser != nil {
		return s.Config.IAMUser.MFASerial != ""
	}
	return false
}

// GetSessionDuration returns the configured session duration or default.
func (s *Session) GetSessionDuration() int {
	const defaultDuration = 3600 // 1 hour

	switch s.Type {
	case SessionTypeIAMUser:
		if s.Config.IAMUser != nil && s.Config.IAMUser.SessionDuration > 0 {
			return s.Config.IAMUser.SessionDuration
		}
	case SessionTypeIAMRole:
		if s.Config.IAMRole != nil && s.Config.IAMRole.SessionDuration > 0 {
			return s.Config.IAMRole.SessionDuration
		}
	}

	return defaultDuration
}

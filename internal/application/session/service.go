// Package session provides session management use cases.
package session

import (
	"context"
	"time"

	"github.com/jvendramin/accio/internal/domain/credential"
	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/infrastructure/storage/awsconfig"
	"github.com/jvendramin/accio/pkg/provider"
)

// Service provides session management operations.
type Service struct {
	sessionRepo          session.Repository
	secureStore          SecureStore
	awsConfig            *awsconfig.Manager
	providers            map[session.Provider]provider.CloudProvider
	refreshBefore        time.Duration
	inactivityTimeout    time.Duration
	activeDefaultSession string // ID of session currently set as default profile
}

// SecureStore defines the interface for credential storage.
type SecureStore interface {
	StoreCredential(sessionID string, cred *credential.Credential) error
	GetCredential(sessionID string) (*credential.Credential, error)
	DeleteCredential(sessionID string) error
	StoreSecret(key string, value []byte) error
	GetSecret(key string) ([]byte, error)
	DeleteSecret(key string) error
}

// NewService creates a new session service.
func NewService(
	sessionRepo session.Repository,
	secureStore SecureStore,
	refreshBefore time.Duration,
	inactivityTimeout time.Duration,
) *Service {
	return &Service{
		sessionRepo:       sessionRepo,
		secureStore:       secureStore,
		awsConfig:         awsconfig.NewManager("", "", "", false),
		providers:         make(map[session.Provider]provider.CloudProvider),
		refreshBefore:     refreshBefore,
		inactivityTimeout: inactivityTimeout,
	}
}

// RegisterProvider registers a cloud provider.
func (s *Service) RegisterProvider(p provider.CloudProvider) {
	s.providers[session.Provider(p.Name())] = p
}

// List returns all sessions.
func (s *Service) List(ctx context.Context) ([]*session.Session, error) {
	sessions, err := s.sessionRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Load credentials for each session
	for _, sess := range sessions {
		if cred, err := s.secureStore.GetCredential(sess.ID); err == nil {
			sess.SetCredential(cred)
			// Update status based on credential state
			if cred.IsExpired() {
				sess.SetError(domainErrors.ErrCredentialExpired)
			} else if cred.IsExpiringSoon(s.refreshBefore) {
				sess.Status = session.StatusExpiring
			}
		}
	}

	return sessions, nil
}

// Get returns a session by ID.
func (s *Service) Get(ctx context.Context, id string) (*session.Session, error) {
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load credential if exists
	if cred, err := s.secureStore.GetCredential(sess.ID); err == nil {
		sess.SetCredential(cred)
	}

	return sess, nil
}

// Create creates a new session.
func (s *Service) Create(ctx context.Context, sess *session.Session) error {
	if err := sess.Validate(); err != nil {
		return err
	}
	return s.sessionRepo.Save(ctx, sess)
}

// Update updates an existing session.
func (s *Service) Update(ctx context.Context, sess *session.Session) error {
	if err := sess.Validate(); err != nil {
		return err
	}
	sess.Metadata.UpdatedAt = time.Now()
	return s.sessionRepo.Save(ctx, sess)
}

// Delete removes a session.
func (s *Service) Delete(ctx context.Context, id string) error {
	// Stop the session first if active
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if sess.IsActive() {
		if err := s.Stop(ctx, id); err != nil {
			return err
		}
	}

	// Delete credential - ignore error as session deletion should succeed
	// even if credential cleanup fails (credential may not exist)
	_ = s.secureStore.DeleteCredential(id)

	return s.sessionRepo.Delete(ctx, id)
}

// invalidateInactiveSession invalidates a session that has been inactive for too long.
func (s *Service) invalidateInactiveSession(ctx context.Context, sess *session.Session) error {
	// Delete credential from secure store
	_ = s.secureStore.DeleteCredential(sess.ID)

	// Clear credentials from AWS config files (best-effort cleanup)
	if sess.ProfileName != "" {
		_ = s.awsConfig.ClearCredentials(sess.ProfileName)
	}

	// Clear default profile if this session was the active default
	if s.activeDefaultSession == sess.ID {
		_ = s.awsConfig.ClearCredentials("default")
		s.activeDefaultSession = ""
	}

	// Mark session as inactive
	if err := sess.Stop(); err != nil {
		return err
	}

	return s.sessionRepo.Save(ctx, sess)
}

// CheckAndInvalidateInactiveSessions checks all sessions and invalidates those that are inactive.
func (s *Service) CheckAndInvalidateInactiveSessions(ctx context.Context) error {
	if s.inactivityTimeout == 0 {
		return nil // Inactivity checking disabled
	}

	sessions, err := s.sessionRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, sess := range sessions {
		if sess.IsActive() && sess.IsInactive(s.inactivityTimeout) {
			_ = s.invalidateInactiveSession(ctx, sess)
		}
	}

	return nil
}

// Start starts a session and obtains credentials.
func (s *Service) Start(ctx context.Context, id string, mfaToken string) error {
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if sess.IsActive() {
		return domainErrors.NewDomainError("Start", domainErrors.ErrSessionAlreadyActive, nil)
	}

	// Get the provider
	p, ok := s.providers[sess.Provider]
	if !ok {
		return domainErrors.NewDomainError("Start", domainErrors.ErrProviderNotFound, nil).
			WithContext("provider", string(sess.Provider))
	}

	// Start the session with the provider
	sess.SetPending()
	cred, err := p.StartSession(ctx, sess, mfaToken)
	if err != nil {
		sess.SetError(err)
		// Best effort save - we're returning the actual error anyway
		_ = s.sessionRepo.Save(ctx, sess)
		return err
	}

	// Store credential securely
	if err := s.secureStore.StoreCredential(sess.ID, cred); err != nil {
		sess.SetError(err)
		_ = s.sessionRepo.Save(ctx, sess)
		return err
	}

	// Write credentials to AWS config files for CLI access.
	// These are best-effort operations - credentials are already stored
	// securely in the keyring, so CLI config failures are non-fatal.
	if sess.ProfileName != "" {
		_ = s.awsConfig.WriteCredentials(
			sess.ProfileName,
			cred.AccessKeyID,
			cred.SecretAccessKey,
			cred.SessionToken,
			sess.Region,
		)
	}

	// Also write to default profile so user doesn't need --profile flag
	_ = s.awsConfig.WriteCredentials(
		"default",
		cred.AccessKeyID,
		cred.SecretAccessKey,
		cred.SessionToken,
		sess.Region,
	)
	// Track which session is currently the default
	s.activeDefaultSession = sess.ID

	// Update session state
	if err := sess.Start(cred); err != nil {
		return err
	}

	return s.sessionRepo.Save(ctx, sess)
}

// Stop stops an active session.
func (s *Service) Stop(ctx context.Context, id string) error {
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if !sess.IsActive() && sess.Status != session.StatusExpiring && sess.Status != session.StatusError {
		return domainErrors.NewDomainError("Stop", domainErrors.ErrSessionNotActive, nil)
	}

	// Provider cleanup is best-effort - local cleanup should always proceed
	if p, ok := s.providers[sess.Provider]; ok {
		_ = p.StopSession(ctx, sess)
	}

	// Delete credential from secure store (best-effort, may already be deleted)
	_ = s.secureStore.DeleteCredential(sess.ID)

	// Clear credentials from AWS config files (best-effort cleanup)
	if sess.ProfileName != "" {
		_ = s.awsConfig.ClearCredentials(sess.ProfileName)
	}

	// Clear default profile if this session was the active default
	if s.activeDefaultSession == sess.ID {
		_ = s.awsConfig.ClearCredentials("default")
		s.activeDefaultSession = ""
	}

	// Update session state
	if err := sess.Stop(); err != nil {
		return err
	}

	return s.sessionRepo.Save(ctx, sess)
}

// Rotate rotates credentials for an active session.
func (s *Service) Rotate(ctx context.Context, id string, mfaToken string) error {
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if !sess.IsActive() && sess.Status != session.StatusExpiring {
		return domainErrors.NewDomainError("Rotate", domainErrors.ErrSessionNotActive, nil)
	}

	// Get the provider
	p, ok := s.providers[sess.Provider]
	if !ok {
		return domainErrors.NewDomainError("Rotate", domainErrors.ErrProviderNotFound, nil)
	}

	// Rotate credentials
	cred, err := p.RotateCredentials(ctx, sess, mfaToken)
	if err != nil {
		sess.SetError(err)
		_ = s.sessionRepo.Save(ctx, sess)
		return err
	}

	// Store new credential
	if err := s.secureStore.StoreCredential(sess.ID, cred); err != nil {
		return err
	}

	// Write credentials to AWS config files (best-effort, non-fatal)
	if sess.ProfileName != "" {
		_ = s.awsConfig.WriteCredentials(
			sess.ProfileName,
			cred.AccessKeyID,
			cred.SecretAccessKey,
			cred.SessionToken,
			sess.Region,
		)
	}

	// Update default profile if this is the active default session
	if s.activeDefaultSession == sess.ID {
		_ = s.awsConfig.WriteCredentials(
			"default",
			cred.AccessKeyID,
			cred.SecretAccessKey,
			cred.SessionToken,
			sess.Region,
		)
	}

	// Update session state
	sess.SetCredential(cred)
	sess.Status = session.StatusActive
	sess.Metadata.UpdatedAt = time.Now()

	return s.sessionRepo.Save(ctx, sess)
}

// GetCredential returns the credential for a session.
func (s *Service) GetCredential(ctx context.Context, id string) (*credential.Credential, error) {
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if session is inactive
	if s.inactivityTimeout > 0 && sess.IsInactive(s.inactivityTimeout) {
		// Invalidate the session
		if err := s.invalidateInactiveSession(ctx, sess); err != nil {
			return nil, domainErrors.NewDomainError("GetCredential", domainErrors.ErrCredentialExpired, err).
				WithContext("reason", "session inactive")
		}
		return nil, domainErrors.NewDomainError("GetCredential", domainErrors.ErrCredentialExpired, nil).
			WithContext("reason", "session inactive")
	}

	cred, err := s.secureStore.GetCredential(sess.ID)
	if err != nil {
		return nil, err
	}

	// Check if credential needs refresh
	if cred.IsExpiringSoon(s.refreshBefore) {
		// Try to rotate
		if err := s.Rotate(ctx, id, ""); err != nil {
			// If rotation fails (e.g., needs MFA), return existing credential
			if cred.IsExpired() {
				return nil, domainErrors.ErrCredentialExpired
			}
		} else {
			// Get refreshed credential
			cred, err = s.secureStore.GetCredential(sess.ID)
			if err != nil {
				return nil, err
			}
		}
	}

	// Update last used timestamp
	sess.UpdateLastUsed()
	_ = s.sessionRepo.Save(ctx, sess)

	return cred, nil
}

// GetByProfileName returns a session by AWS profile name.
func (s *Service) GetByProfileName(ctx context.Context, profileName string) (*session.Session, error) {
	return s.sessionRepo.GetByProfileName(ctx, profileName)
}

// CreateWithSecret creates a new session and stores the secret key.
func (s *Service) CreateWithSecret(ctx context.Context, sess *session.Session, secretKey string) error {
	if err := sess.Validate(); err != nil {
		return err
	}

	// Store the secret key first if provided
	if secretKey != "" && sess.Type == session.SessionTypeIAMUser {
		key := "iam-user-secret:" + sess.ID
		if err := s.secureStore.StoreSecret(key, []byte(secretKey)); err != nil {
			return err
		}
	}

	return s.sessionRepo.Save(ctx, sess)
}

// DeleteWithSecret deletes a session and its associated secrets.
func (s *Service) DeleteWithSecret(ctx context.Context, id string) error {
	sess, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Stop the session first if active - continue with deletion even if stop fails
	if sess.IsActive() {
		_ = s.Stop(ctx, id)
	}

	// Clean up credentials and secrets (best-effort, may not exist)
	_ = s.secureStore.DeleteCredential(id)

	if sess.Type == session.SessionTypeIAMUser {
		key := "iam-user-secret:" + id
		_ = s.secureStore.DeleteSecret(key)
	}

	return s.sessionRepo.Delete(ctx, id)
}

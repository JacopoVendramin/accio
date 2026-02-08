package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/jvendramin/accio/internal/domain/credential"
	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/infrastructure/aws/sts"
	keyringStore "github.com/jvendramin/accio/internal/infrastructure/storage/keyring"
)

// RoleChainedProvider handles IAM Role chained sessions.
type RoleChainedProvider struct {
	stsClient   *sts.Client
	secureStore keyringStore.SecureStore
	sessionRepo session.Repository
}

// NewRoleChainedProvider creates a new role chained provider.
func NewRoleChainedProvider(secureStore keyringStore.SecureStore, sessionRepo session.Repository) *RoleChainedProvider {
	return &RoleChainedProvider{
		stsClient:   sts.NewClient(""),
		secureStore: secureStore,
		sessionRepo: sessionRepo,
	}
}

// Name returns the provider name.
func (p *RoleChainedProvider) Name() string {
	return "aws"
}

// SupportedSessionTypes returns supported session types.
func (p *RoleChainedProvider) SupportedSessionTypes() []session.SessionType {
	return []session.SessionType{session.SessionTypeIAMRole}
}

// StartSession starts a role chained session.
func (p *RoleChainedProvider) StartSession(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	if sess.Type != session.SessionTypeIAMRole {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidSessionType, nil)
	}

	cfg := sess.Config.IAMRole
	if cfg == nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidConfig, nil)
	}

	// Get parent session credentials
	parentCred, err := p.getParentCredentials(ctx, cfg.ParentSessionID)
	if err != nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrParentSessionFailed, err)
	}

	// Get session duration
	duration := cfg.SessionDuration
	if duration == 0 {
		duration = defaultSessionDuration
	}

	// Create STS client with the session's region
	stsClient := sts.NewClient(sess.Region)

	// Generate session name
	sessionName := generateSessionName(sess.Name)

	// Assume the role
	cred, err := stsClient.AssumeRole(
		ctx,
		parentCred,
		cfg.RoleARN,
		sessionName,
		int32(duration),
		cfg.ExternalID,
	)
	if err != nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrStorageFailure, err)
	}

	return cred, nil
}

// StopSession stops a role chained session.
func (p *RoleChainedProvider) StopSession(ctx context.Context, sess *session.Session) error {
	// For role sessions, stopping just means clearing the cached credentials
	return nil
}

// RotateCredentials rotates credentials for a role chained session.
func (p *RoleChainedProvider) RotateCredentials(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	// For role sessions, rotation is the same as starting a new session
	return p.StartSession(ctx, sess, mfaToken)
}

// getParentCredentials retrieves credentials from the parent session.
func (p *RoleChainedProvider) getParentCredentials(ctx context.Context, parentID string) (*credential.Credential, error) {
	// Get parent session
	parent, err := p.sessionRepo.Get(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent session: %w", err)
	}

	// Check if parent is active
	if !parent.IsActive() {
		return nil, fmt.Errorf("parent session is not active")
	}

	// Get credentials from secure storage
	cred, err := p.secureStore.GetCredential(parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent credentials: %w", err)
	}

	// Check if credentials are expired
	if cred.IsExpired() {
		return nil, fmt.Errorf("parent session credentials are expired")
	}

	return cred, nil
}

// generateSessionName generates a role session name from the session name.
func generateSessionName(name string) string {
	// AWS role session names must be 2-64 characters, alphanumeric plus =,.@-
	// Replace invalid characters
	name = strings.ReplaceAll(name, " ", "-")

	// Truncate if too long
	if len(name) > 64 {
		name = name[:64]
	}

	// Ensure minimum length
	if len(name) < 2 {
		name = "accio-session"
	}

	return name
}

// RefreshParentAndChild refreshes both parent and child session credentials.
func (p *RoleChainedProvider) RefreshParentAndChild(ctx context.Context, childSess *session.Session, parentProvider interface {
	RotateCredentials(context.Context, *session.Session, string) (*credential.Credential, error)
}) (*credential.Credential, error) {
	if childSess.Type != session.SessionTypeIAMRole || childSess.Config.IAMRole == nil {
		return nil, domainErrors.NewDomainError("RefreshParentAndChild", domainErrors.ErrInvalidSessionType, nil)
	}

	// Get parent session
	parentSess, err := p.sessionRepo.Get(ctx, childSess.Config.IAMRole.ParentSessionID)
	if err != nil {
		return nil, err
	}

	// Check if parent needs refresh
	parentCred, err := p.secureStore.GetCredential(parentSess.ID)
	if err != nil || parentCred.IsExpiringSoon(5*60) {
		// Refresh parent first
		newParentCred, err := parentProvider.RotateCredentials(ctx, parentSess, "")
		if err != nil {
			return nil, fmt.Errorf("failed to refresh parent session: %w", err)
		}

		// Store new parent credentials
		if err := p.secureStore.StoreCredential(parentSess.ID, newParentCred); err != nil {
			return nil, err
		}
	}

	// Now refresh the child session
	return p.StartSession(ctx, childSess, "")
}

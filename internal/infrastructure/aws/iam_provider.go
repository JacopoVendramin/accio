// Package aws provides AWS-specific implementations.
package aws

import (
	"context"
	"fmt"

	"github.com/jvendramin/accio/internal/domain/credential"
	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/infrastructure/aws/sts"
	keyringStore "github.com/jvendramin/accio/internal/infrastructure/storage/keyring"
)

const (
	defaultSessionDuration = 3600 // 1 hour
)

// IAMUserProvider handles IAM User sessions.
type IAMUserProvider struct {
	stsClient   *sts.Client
	secureStore keyringStore.SecureStore
}

// NewIAMUserProvider creates a new IAM User provider.
func NewIAMUserProvider(secureStore keyringStore.SecureStore) *IAMUserProvider {
	return &IAMUserProvider{
		stsClient:   sts.NewClient(""),
		secureStore: secureStore,
	}
}

// Name returns the provider name.
func (p *IAMUserProvider) Name() string {
	return "aws"
}

// SupportedSessionTypes returns supported session types.
func (p *IAMUserProvider) SupportedSessionTypes() []session.SessionType {
	return []session.SessionType{session.SessionTypeIAMUser}
}

// StartSession starts an IAM User session.
func (p *IAMUserProvider) StartSession(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	if sess.Type != session.SessionTypeIAMUser {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidSessionType, nil)
	}

	cfg := sess.Config.IAMUser
	if cfg == nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidConfig, nil)
	}

	// Get the secret access key from secure storage
	secretKey, err := p.secureStore.GetSecret(secretKeyFor(sess.ID))
	if err != nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrCredentialNotFound, err)
	}

	// Check if MFA is required
	if cfg.MFASerial != "" && mfaToken == "" {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrMFARequired, nil)
	}

	// Get session duration
	duration := cfg.SessionDuration
	if duration == 0 {
		duration = defaultSessionDuration
	}

	// Create STS client with the session's region
	stsClient := sts.NewClient(sess.Region)

	// Get session token
	cred, err := stsClient.GetSessionToken(
		ctx,
		cfg.AccessKeyID,
		string(secretKey),
		int32(duration),
		cfg.MFASerial,
		mfaToken,
	)
	if err != nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrStorageFailure, err)
	}

	return cred, nil
}

// StopSession stops an IAM User session.
func (p *IAMUserProvider) StopSession(ctx context.Context, sess *session.Session) error {
	// For IAM User sessions, stopping just means clearing the cached credentials
	// There's no server-side action needed
	return nil
}

// RotateCredentials rotates credentials for an IAM User session.
func (p *IAMUserProvider) RotateCredentials(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	// For IAM User sessions, rotation is the same as starting a new session
	return p.StartSession(ctx, sess, mfaToken)
}

// StoreIAMUserSecret stores the secret access key for an IAM User session.
func (p *IAMUserProvider) StoreIAMUserSecret(sessionID, secretAccessKey string) error {
	return p.secureStore.StoreSecret(secretKeyFor(sessionID), []byte(secretAccessKey))
}

// DeleteIAMUserSecret deletes the secret access key for an IAM User session.
func (p *IAMUserProvider) DeleteIAMUserSecret(sessionID string) error {
	return p.secureStore.DeleteSecret(secretKeyFor(sessionID))
}

// secretKeyFor returns the secret key storage key for a session.
func secretKeyFor(sessionID string) string {
	return fmt.Sprintf("iam-user-secret:%s", sessionID)
}

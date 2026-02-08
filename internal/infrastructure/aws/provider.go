// Package aws provides AWS-specific implementations.
package aws

import (
	"context"

	"github.com/jvendramin/accio/internal/domain/credential"
	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/domain/session"
	keyringStore "github.com/jvendramin/accio/internal/infrastructure/storage/keyring"
)

// Provider is a unified AWS provider that handles all AWS session types.
type Provider struct {
	iamUserProvider *IAMUserProvider
	ssoProvider     *SSOProvider
	roleProvider    *RoleChainedProvider
}

// NewProvider creates a new unified AWS provider.
func NewProvider(
	secureStore keyringStore.SecureStore,
	integrationRepo integration.Repository,
	sessionRepo session.Repository,
) *Provider {
	return &Provider{
		iamUserProvider: NewIAMUserProvider(secureStore),
		ssoProvider:     NewSSOProvider(secureStore, integrationRepo),
		roleProvider:    NewRoleChainedProvider(secureStore, sessionRepo),
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "aws"
}

// SupportedSessionTypes returns all supported AWS session types.
func (p *Provider) SupportedSessionTypes() []session.SessionType {
	return []session.SessionType{
		session.SessionTypeIAMUser,
		session.SessionTypeAWSSSO,
		session.SessionTypeIAMRole,
	}
}

// StartSession starts an AWS session based on its type.
func (p *Provider) StartSession(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	switch sess.Type {
	case session.SessionTypeIAMUser:
		return p.iamUserProvider.StartSession(ctx, sess, mfaToken)
	case session.SessionTypeAWSSSO:
		return p.ssoProvider.StartSession(ctx, sess, mfaToken)
	case session.SessionTypeIAMRole:
		return p.roleProvider.StartSession(ctx, sess, mfaToken)
	default:
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidSessionType, nil).
			WithContext("type", string(sess.Type))
	}
}

// StopSession stops an AWS session.
func (p *Provider) StopSession(ctx context.Context, sess *session.Session) error {
	switch sess.Type {
	case session.SessionTypeIAMUser:
		return p.iamUserProvider.StopSession(ctx, sess)
	case session.SessionTypeAWSSSO:
		return p.ssoProvider.StopSession(ctx, sess)
	case session.SessionTypeIAMRole:
		return p.roleProvider.StopSession(ctx, sess)
	default:
		return nil
	}
}

// RotateCredentials rotates credentials for an AWS session.
func (p *Provider) RotateCredentials(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	switch sess.Type {
	case session.SessionTypeIAMUser:
		return p.iamUserProvider.RotateCredentials(ctx, sess, mfaToken)
	case session.SessionTypeAWSSSO:
		return p.ssoProvider.RotateCredentials(ctx, sess, mfaToken)
	case session.SessionTypeIAMRole:
		return p.roleProvider.RotateCredentials(ctx, sess, mfaToken)
	default:
		return nil, domainErrors.NewDomainError("RotateCredentials", domainErrors.ErrInvalidSessionType, nil)
	}
}

// SSO returns the SSO provider for SSO-specific operations.
func (p *Provider) SSO() *SSOProvider {
	return p.ssoProvider
}

// IAMUser returns the IAM User provider for IAM-specific operations.
func (p *Provider) IAMUser() *IAMUserProvider {
	return p.iamUserProvider
}

// RoleChained returns the Role Chained provider for role-specific operations.
func (p *Provider) RoleChained() *RoleChainedProvider {
	return p.roleProvider
}

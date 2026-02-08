// Package provider defines interfaces for cloud providers.
package provider

import (
	"context"

	"github.com/jvendramin/accio/internal/domain/credential"
	"github.com/jvendramin/accio/internal/domain/session"
)

// CloudProvider defines the interface for cloud credential providers.
type CloudProvider interface {
	// Name returns the provider name (e.g., "aws", "azure", "gcp").
	Name() string

	// SupportedSessionTypes returns the session types this provider supports.
	SupportedSessionTypes() []session.SessionType

	// StartSession starts a session and returns credentials.
	// The mfaToken parameter is used for MFA-enabled sessions.
	StartSession(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error)

	// StopSession stops an active session.
	StopSession(ctx context.Context, sess *session.Session) error

	// RotateCredentials rotates credentials for an active session.
	RotateCredentials(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error)
}

// AuthMethod defines an authentication method for a provider.
type AuthMethod interface {
	// Name returns the auth method name.
	Name() string

	// Description returns a human-readable description.
	Description() string

	// RequiresMFA returns true if this method requires MFA.
	RequiresMFA() bool

	// RequiresUserInteraction returns true if this method requires user interaction (e.g., browser).
	RequiresUserInteraction() bool
}

// SSOProvider extends CloudProvider with SSO-specific operations.
type SSOProvider interface {
	CloudProvider

	// StartDeviceAuthorization begins the SSO device authorization flow.
	// Returns the verification URI and user code.
	StartDeviceAuthorization(ctx context.Context, startURL, region string) (*DeviceAuthorizationResponse, error)

	// PollForToken polls for the access token after user authorization.
	PollForToken(ctx context.Context, clientID, clientSecret, deviceCode, region string) (*SSOToken, error)

	// ListAccounts returns available accounts for the SSO portal.
	ListAccounts(ctx context.Context, accessToken, region string) ([]SSOAccount, error)

	// ListAccountRoles returns available roles for an account.
	ListAccountRoles(ctx context.Context, accessToken, accountID, region string) ([]SSORole, error)

	// GetRoleCredentials gets credentials for a specific role.
	GetRoleCredentials(ctx context.Context, accessToken, accountID, roleName, region string) (*credential.Credential, error)
}

// DeviceAuthorizationResponse contains the device authorization response.
type DeviceAuthorizationResponse struct {
	ClientID             string
	ClientSecret         string
	DeviceCode           string
	UserCode             string
	VerificationURI      string
	VerificationURIComplete string
	ExpiresIn            int
	Interval             int
}

// SSOToken contains an SSO access token.
type SSOToken struct {
	AccessToken  string
	ExpiresAt    int64
	RefreshToken string
}

// SSOAccount represents an AWS account in SSO.
type SSOAccount struct {
	AccountID    string
	AccountName  string
	EmailAddress string
}

// SSORole represents a role in an SSO account.
type SSORole struct {
	RoleName    string
	AccountID   string
}

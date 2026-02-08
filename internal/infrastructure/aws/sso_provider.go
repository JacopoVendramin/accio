package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jvendramin/accio/internal/domain/credential"
	domainErrors "github.com/jvendramin/accio/internal/domain/errors"
	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/internal/infrastructure/aws/sso"
	keyringStore "github.com/jvendramin/accio/internal/infrastructure/storage/keyring"
	"github.com/jvendramin/accio/pkg/provider"
)

// SSOProvider handles AWS SSO sessions.
type SSOProvider struct {
	secureStore     keyringStore.SecureStore
	integrationRepo integration.Repository
}

// NewSSOProvider creates a new SSO provider.
func NewSSOProvider(secureStore keyringStore.SecureStore, integrationRepo integration.Repository) *SSOProvider {
	return &SSOProvider{
		secureStore:     secureStore,
		integrationRepo: integrationRepo,
	}
}

// Name returns the provider name.
func (p *SSOProvider) Name() string {
	return "aws"
}

// SupportedSessionTypes returns supported session types.
func (p *SSOProvider) SupportedSessionTypes() []session.SessionType {
	return []session.SessionType{session.SessionTypeAWSSSO}
}

// StartSession starts an AWS SSO session.
func (p *SSOProvider) StartSession(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	if sess.Type != session.SessionTypeAWSSSO {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidSessionType, nil)
	}

	cfg := sess.Config.AWSSSO
	if cfg == nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrInvalidConfig, nil)
	}

	// Get SSO token from integration or secure storage
	accessToken, err := p.getAccessToken(ctx, cfg)
	if err != nil {
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrSSOLoginRequired, err)
	}

	// Get credentials using the SSO token
	ssoClient := sso.NewClient(cfg.Region)
	cred, err := ssoClient.GetRoleCredentials(ctx, accessToken, cfg.AccountID, cfg.RoleName)
	if err != nil {
		// Token might be expired
		if strings.Contains(err.Error(), "ExpiredToken") || strings.Contains(err.Error(), "UnauthorizedException") {
			return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrSSOLoginRequired, err)
		}
		return nil, domainErrors.NewDomainError("StartSession", domainErrors.ErrStorageFailure, err)
	}

	return cred, nil
}

// StopSession stops an AWS SSO session.
func (p *SSOProvider) StopSession(ctx context.Context, sess *session.Session) error {
	// For SSO sessions, stopping just clears the cached credentials
	// The SSO token is shared across sessions from the same portal
	return nil
}

// RotateCredentials rotates credentials for an AWS SSO session.
func (p *SSOProvider) RotateCredentials(ctx context.Context, sess *session.Session, mfaToken string) (*credential.Credential, error) {
	// For SSO sessions, rotation is the same as starting
	return p.StartSession(ctx, sess, mfaToken)
}

// StartDeviceAuthorization begins the SSO device authorization flow.
func (p *SSOProvider) StartDeviceAuthorization(ctx context.Context, startURL, region string) (*provider.DeviceAuthorizationResponse, error) {
	ssoClient := sso.NewClient(region)
	return ssoClient.StartDeviceAuthorization(ctx, startURL)
}

// PollForToken polls for the access token after user authorization.
func (p *SSOProvider) PollForToken(ctx context.Context, clientID, clientSecret, deviceCode, region string, interval, timeout time.Duration) (*provider.SSOToken, error) {
	ssoClient := sso.NewClient(region)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		token, err := ssoClient.PollForToken(ctx, clientID, clientSecret, deviceCode)
		if err != nil {
			// Check if still pending
			if strings.Contains(err.Error(), "AuthorizationPendingException") ||
				strings.Contains(err.Error(), "authorization_pending") {
				time.Sleep(interval)
				continue
			}
			// Check if slow down requested
			if strings.Contains(err.Error(), "SlowDownException") {
				interval = interval * 2
				time.Sleep(interval)
				continue
			}
			return nil, err
		}
		return token, nil
	}

	return nil, errors.New("device authorization timed out")
}

// ListAccounts returns available accounts for the SSO portal.
func (p *SSOProvider) ListAccounts(ctx context.Context, accessToken, region string) ([]provider.SSOAccount, error) {
	ssoClient := sso.NewClient(region)
	return ssoClient.ListAccounts(ctx, accessToken)
}

// ListAccountRoles returns available roles for an account.
func (p *SSOProvider) ListAccountRoles(ctx context.Context, accessToken, accountID, region string) ([]provider.SSORole, error) {
	ssoClient := sso.NewClient(region)
	return ssoClient.ListAccountRoles(ctx, accessToken, accountID)
}

// GetRoleCredentials gets credentials for a specific role.
func (p *SSOProvider) GetRoleCredentials(ctx context.Context, accessToken, accountID, roleName, region string) (*credential.Credential, error) {
	ssoClient := sso.NewClient(region)
	return ssoClient.GetRoleCredentials(ctx, accessToken, accountID, roleName)
}

// StoreAccessToken stores an SSO access token.
func (p *SSOProvider) StoreAccessToken(integrationID string, token *provider.SSOToken) error {
	key := ssoTokenKey(integrationID)
	data := fmt.Sprintf("%s|%d|%s", token.AccessToken, token.ExpiresAt, token.RefreshToken)
	return p.secureStore.StoreSecret(key, []byte(data))
}

// GetAccessToken retrieves an SSO access token.
func (p *SSOProvider) GetAccessToken(integrationID string) (*provider.SSOToken, error) {
	key := ssoTokenKey(integrationID)
	data, err := p.secureStore.GetSecret(key)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(string(data), "|", 3)
	if len(parts) < 2 {
		return nil, errors.New("invalid token data")
	}

	var expiresAt int64
	_, _ = fmt.Sscanf(parts[1], "%d", &expiresAt)

	token := &provider.SSOToken{
		AccessToken: parts[0],
		ExpiresAt:   expiresAt,
	}
	if len(parts) > 2 {
		token.RefreshToken = parts[2]
	}

	return token, nil
}

// DeleteAccessToken removes an SSO access token.
func (p *SSOProvider) DeleteAccessToken(integrationID string) error {
	key := ssoTokenKey(integrationID)
	return p.secureStore.DeleteSecret(key)
}

// getAccessToken retrieves the access token for an SSO session.
func (p *SSOProvider) getAccessToken(ctx context.Context, cfg *session.AWSSSOConfig) (string, error) {
	// Try to get from integration if specified
	if cfg.IntegrationID != "" {
		token, err := p.GetAccessToken(cfg.IntegrationID)
		if err == nil && token.ExpiresAt > time.Now().Add(5*time.Minute).Unix() {
			return token.AccessToken, nil
		}
	}

	// Try to get by start URL
	token, err := p.GetAccessToken(tokenKeyFromStartURL(cfg.StartURL))
	if err == nil && token.ExpiresAt > time.Now().Add(5*time.Minute).Unix() {
		return token.AccessToken, nil
	}

	return "", errors.New("no valid SSO token found")
}

// ssoTokenKey returns the storage key for an SSO token.
func ssoTokenKey(integrationID string) string {
	return fmt.Sprintf("sso-token:%s", integrationID)
}

// tokenKeyFromStartURL generates a token key from the start URL.
func tokenKeyFromStartURL(startURL string) string {
	// Simple hash of the URL for storage key
	return fmt.Sprintf("sso-url:%s", startURL)
}

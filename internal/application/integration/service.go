// Package integration provides integration management use cases.
package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/jvendramin/accio/internal/domain/integration"
	"github.com/jvendramin/accio/internal/domain/session"
	"github.com/jvendramin/accio/pkg/provider"
)

// SecureStore defines the interface for token storage.
type SecureStore interface {
	StoreSecret(key string, value []byte) error
	GetSecret(key string) ([]byte, error)
	DeleteSecret(key string) error
}

// SSOClient defines the interface for SSO operations.
type SSOClient interface {
	StartDeviceAuthorization(ctx context.Context, startURL string) (*provider.DeviceAuthorizationResponse, error)
	PollForToken(ctx context.Context, clientID, clientSecret, deviceCode string) (*provider.SSOToken, error)
	ListAccounts(ctx context.Context, accessToken string) ([]provider.SSOAccount, error)
	ListAccountRoles(ctx context.Context, accessToken, accountID string) ([]provider.SSORole, error)
}

// Service provides integration management operations.
type Service struct {
	integrationRepo integration.Repository
	sessionRepo     session.Repository
	secureStore     SecureStore
}

// NewService creates a new integration service.
func NewService(
	integrationRepo integration.Repository,
	sessionRepo session.Repository,
	secureStore SecureStore,
) *Service {
	return &Service{
		integrationRepo: integrationRepo,
		sessionRepo:     sessionRepo,
		secureStore:     secureStore,
	}
}

// List returns all integrations.
func (s *Service) List(ctx context.Context) ([]*integration.Integration, error) {
	integrations, err := s.integrationRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Load token expiry for each integration
	for _, integ := range integrations {
		if integ.Type == integration.IntegrationTypeAWSSSO {
			s.loadTokenExpiry(integ)
		}
	}

	return integrations, nil
}

// Get returns an integration by ID.
func (s *Service) Get(ctx context.Context, id string) (*integration.Integration, error) {
	integ, err := s.integrationRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if integ.Type == integration.IntegrationTypeAWSSSO {
		s.loadTokenExpiry(integ)
	}

	return integ, nil
}

// Create creates a new integration.
func (s *Service) Create(ctx context.Context, integ *integration.Integration) error {
	return s.integrationRepo.Save(ctx, integ)
}

// Update updates an existing integration.
func (s *Service) Update(ctx context.Context, integ *integration.Integration) error {
	integ.Metadata.UpdatedAt = time.Now()
	return s.integrationRepo.Save(ctx, integ)
}

// Delete removes an integration and its associated sessions.
func (s *Service) Delete(ctx context.Context, id string) error {
	// Get the integration first
	integ, err := s.integrationRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete associated token
	if integ.Type == integration.IntegrationTypeAWSSSO {
		_ = s.secureStore.DeleteSecret(tokenKey(id))
	}

	// Delete sessions associated with this integration
	sessions, err := s.sessionRepo.List(ctx)
	if err == nil {
		for _, sess := range sessions {
			if sess.Config.AWSSSO != nil && sess.Config.AWSSSO.IntegrationID == id {
				_ = s.sessionRepo.Delete(ctx, sess.ID)
			}
		}
	}

	return s.integrationRepo.Delete(ctx, id)
}

// StoreToken stores an SSO access token for an integration.
func (s *Service) StoreToken(integrationID string, token *provider.SSOToken) error {
	data := fmt.Sprintf("%s|%d", token.AccessToken, token.ExpiresAt)
	return s.secureStore.StoreSecret(tokenKey(integrationID), []byte(data))
}

// GetToken retrieves an SSO access token for an integration.
func (s *Service) GetToken(integrationID string) (*provider.SSOToken, error) {
	data, err := s.secureStore.GetSecret(tokenKey(integrationID))
	if err != nil {
		return nil, err
	}

	var accessToken string
	var expiresAt int64
	_, err = fmt.Sscanf(string(data), "%s|%d", &accessToken, &expiresAt)
	if err != nil {
		// Try parsing just the token
		parts := splitToken(string(data))
		if len(parts) >= 2 {
			accessToken = parts[0]
			_, _ = fmt.Sscanf(parts[1], "%d", &expiresAt)
		}
	}

	return &provider.SSOToken{
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
	}, nil
}

// loadTokenExpiry loads the token expiry from storage.
func (s *Service) loadTokenExpiry(integ *integration.Integration) {
	token, err := s.GetToken(integ.ID)
	if err == nil && integ.Config.AWSSSO != nil {
		integ.Config.AWSSSO.ExpiresAt = time.Unix(token.ExpiresAt, 0)
	}
}

// SyncResult represents the result of syncing an integration.
type SyncResult struct {
	Accounts []AccountWithRoles
	Sessions []*session.Session
}

// AccountWithRoles represents an account with its roles.
type AccountWithRoles struct {
	AccountID   string
	AccountName string
	Email       string
	Roles       []string
}

// CreateSessionsFromSync creates sessions from discovered accounts and roles.
func (s *Service) CreateSessionsFromSync(
	ctx context.Context,
	integ *integration.Integration,
	accounts []AccountWithRoles,
) ([]*session.Session, error) {
	var created []*session.Session

	for _, acct := range accounts {
		for _, roleName := range acct.Roles {
			// Check if session already exists
			existing, _ := s.findExistingSession(ctx, integ.ID, acct.AccountID, roleName)
			if existing != nil {
				continue // Skip if already exists
			}

			// Create new session
			sessionName := fmt.Sprintf("%s - %s", acct.AccountName, roleName)
			profileName := fmt.Sprintf("%s-%s", sanitizeProfileName(acct.AccountName), sanitizeProfileName(roleName))

			sess := session.NewSession(
				sessionName,
				session.ProviderAWS,
				session.SessionTypeAWSSSO,
				profileName,
				integ.Config.AWSSSO.Region,
			)

			sess.Config.AWSSSO = &session.AWSSSOConfig{
				StartURL:      integ.Config.AWSSSO.StartURL,
				Region:        integ.Config.AWSSSO.Region,
				AccountID:     acct.AccountID,
				AccountName:   acct.AccountName,
				AccountEmail:  acct.Email,
				RoleName:      roleName,
				IntegrationID: integ.ID,
			}

			if err := s.sessionRepo.Save(ctx, sess); err != nil {
				continue // Log and continue
			}

			created = append(created, sess)
		}
	}

	// Update integration's last synced time
	integ.MarkSynced()
	_ = s.integrationRepo.Save(ctx, integ)

	return created, nil
}

func (s *Service) findExistingSession(ctx context.Context, integID, accountID, roleName string) (*session.Session, error) {
	sessions, err := s.sessionRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, sess := range sessions {
		if sess.Config.AWSSSO != nil &&
			sess.Config.AWSSSO.IntegrationID == integID &&
			sess.Config.AWSSSO.AccountID == accountID &&
			sess.Config.AWSSSO.RoleName == roleName {
			return sess, nil
		}
	}

	return nil, nil
}

func tokenKey(integrationID string) string {
	return fmt.Sprintf("sso-token:%s", integrationID)
}

func splitToken(s string) []string {
	result := make([]string, 0, 2)
	idx := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '|' {
			idx = i
			break
		}
	}
	if idx > 0 {
		result = append(result, s[:idx])
		result = append(result, s[idx+1:])
	}
	return result
}

func sanitizeProfileName(name string) string {
	// Replace spaces and special chars with hyphens
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else if c == ' ' {
			result = append(result, '-')
		}
	}
	return string(result)
}

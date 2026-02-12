// Package integration defines SSO and SAML integration entities.
package integration

import (
	"time"

	"github.com/google/uuid"
)

// IntegrationType represents the type of integration.
type IntegrationType string

const (
	IntegrationTypeAWSSSO IntegrationType = "aws_sso"
	IntegrationTypeSAML   IntegrationType = "saml"
)

// Integration represents an SSO or SAML identity provider integration.
type Integration struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Type     IntegrationType     `json:"type"`
	Config   IntegrationConfig   `json:"config"`
	Metadata IntegrationMetadata `json:"metadata"`
}

// IntegrationConfig holds integration-specific configuration.
type IntegrationConfig struct {
	AWSSSO *AWSSSOIntegrationConfig `json:"aws_sso,omitempty"`
	SAML   *SAMLIntegrationConfig   `json:"saml,omitempty"`
}

// AWSSSOIntegrationConfig holds AWS SSO portal configuration.
type AWSSSOIntegrationConfig struct {
	StartURL string `json:"start_url"`
	Region   string `json:"region"`

	// Cached after successful login
	AccessToken string    `json:"-"` // Stored in keyring
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// SAMLIntegrationConfig holds SAML IdP configuration.
type SAMLIntegrationConfig struct {
	IdPType  string `json:"idp_type"` // okta, onelogin, azure_ad, etc.
	IdPURL   string `json:"idp_url"`
	AppID    string `json:"app_id,omitempty"`
	Username string `json:"username,omitempty"`
}

// IntegrationMetadata holds non-essential integration metadata.
type IntegrationMetadata struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastSyncedAt time.Time `json:"last_synced_at,omitempty"`
}

// NewAWSSSOIntegration creates a new AWS SSO integration.
func NewAWSSSOIntegration(name, startURL, region string) *Integration {
	now := time.Now()
	return &Integration{
		ID:   uuid.New().String(),
		Name: name,
		Type: IntegrationTypeAWSSSO,
		Config: IntegrationConfig{
			AWSSSO: &AWSSSOIntegrationConfig{
				StartURL: startURL,
				Region:   region,
			},
		},
		Metadata: IntegrationMetadata{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// NewSAMLIntegration creates a new SAML integration.
func NewSAMLIntegration(name, idpType, idpURL string) *Integration {
	now := time.Now()
	return &Integration{
		ID:   uuid.New().String(),
		Name: name,
		Type: IntegrationTypeSAML,
		Config: IntegrationConfig{
			SAML: &SAMLIntegrationConfig{
				IdPType: idpType,
				IdPURL:  idpURL,
			},
		},
		Metadata: IntegrationMetadata{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// IsTokenValid returns true if the SSO token is still valid.
func (i *Integration) IsTokenValid() bool {
	if i.Type != IntegrationTypeAWSSSO || i.Config.AWSSSO == nil {
		return false
	}
	if i.Config.AWSSSO.ExpiresAt.IsZero() {
		return false
	}
	// Consider token invalid if it expires in less than 5 minutes
	return time.Now().Add(5 * time.Minute).Before(i.Config.AWSSSO.ExpiresAt)
}

// SetAccessToken sets the SSO access token.
func (i *Integration) SetAccessToken(token string, expiresAt time.Time) {
	if i.Config.AWSSSO != nil {
		i.Config.AWSSSO.AccessToken = token
		i.Config.AWSSSO.ExpiresAt = expiresAt
	}
	i.Metadata.UpdatedAt = time.Now()
}

// ClearAccessToken clears the SSO access token.
func (i *Integration) ClearAccessToken() {
	if i.Config.AWSSSO != nil {
		i.Config.AWSSSO.AccessToken = ""
		i.Config.AWSSSO.ExpiresAt = time.Time{}
	}
	i.Metadata.UpdatedAt = time.Now()
}

// MarkSynced updates the last synced timestamp.
func (i *Integration) MarkSynced() {
	i.Metadata.LastSyncedAt = time.Now()
	i.Metadata.UpdatedAt = time.Now()
}

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAWSSSOIntegration(t *testing.T) {
	integration := NewAWSSSOIntegration("My SSO", "https://mycompany.awsapps.com/start", "us-east-1")

	assert.NotEmpty(t, integration.ID)
	assert.Equal(t, "My SSO", integration.Name)
	assert.Equal(t, IntegrationTypeAWSSSO, integration.Type)
	assert.NotNil(t, integration.Config.AWSSSO)
	assert.Equal(t, "https://mycompany.awsapps.com/start", integration.Config.AWSSSO.StartURL)
	assert.Equal(t, "us-east-1", integration.Config.AWSSSO.Region)
	assert.False(t, integration.Metadata.CreatedAt.IsZero())
	assert.False(t, integration.Metadata.UpdatedAt.IsZero())
}

func TestNewSAMLIntegration(t *testing.T) {
	integration := NewSAMLIntegration("Okta SAML", "okta", "https://mycompany.okta.com")

	assert.NotEmpty(t, integration.ID)
	assert.Equal(t, "Okta SAML", integration.Name)
	assert.Equal(t, IntegrationTypeSAML, integration.Type)
	assert.NotNil(t, integration.Config.SAML)
	assert.Equal(t, "okta", integration.Config.SAML.IdPType)
	assert.Equal(t, "https://mycompany.okta.com", integration.Config.SAML.IdPURL)
}

func TestIntegration_IsTokenValid(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Integration
		expected bool
	}{
		{
			name: "valid token",
			setup: func() *Integration {
				i := NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
				i.Config.AWSSSO.ExpiresAt = time.Now().Add(1 * time.Hour)
				return i
			},
			expected: true,
		},
		{
			name: "expired token",
			setup: func() *Integration {
				i := NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
				i.Config.AWSSSO.ExpiresAt = time.Now().Add(-1 * time.Hour)
				return i
			},
			expected: false,
		},
		{
			name: "token expiring soon (within 5 minutes)",
			setup: func() *Integration {
				i := NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
				i.Config.AWSSSO.ExpiresAt = time.Now().Add(3 * time.Minute)
				return i
			},
			expected: false,
		},
		{
			name: "no expiration set",
			setup: func() *Integration {
				return NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
			},
			expected: false,
		},
		{
			name: "SAML integration (not SSO)",
			setup: func() *Integration {
				return NewSAMLIntegration("Test", "okta", "https://test.okta.com")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			integration := tt.setup()
			assert.Equal(t, tt.expected, integration.IsTokenValid())
		})
	}
}

func TestIntegration_SetAccessToken(t *testing.T) {
	integration := NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
	expiresAt := time.Now().Add(1 * time.Hour)

	integration.SetAccessToken("my-token", expiresAt)

	assert.Equal(t, "my-token", integration.Config.AWSSSO.AccessToken)
	assert.Equal(t, expiresAt.Unix(), integration.Config.AWSSSO.ExpiresAt.Unix())
}

func TestIntegration_ClearAccessToken(t *testing.T) {
	integration := NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
	integration.Config.AWSSSO.AccessToken = "my-token"
	integration.Config.AWSSSO.ExpiresAt = time.Now().Add(1 * time.Hour)

	integration.ClearAccessToken()

	assert.Empty(t, integration.Config.AWSSSO.AccessToken)
	assert.True(t, integration.Config.AWSSSO.ExpiresAt.IsZero())
}

func TestIntegration_MarkSynced(t *testing.T) {
	integration := NewAWSSSOIntegration("Test", "https://test.awsapps.com/start", "us-east-1")
	assert.True(t, integration.Metadata.LastSyncedAt.IsZero())

	integration.MarkSynced()

	assert.False(t, integration.Metadata.LastSyncedAt.IsZero())
	assert.True(t, integration.Metadata.LastSyncedAt.After(integration.Metadata.CreatedAt) ||
		integration.Metadata.LastSyncedAt.Equal(integration.Metadata.CreatedAt))
}

func TestIntegrationTypes(t *testing.T) {
	assert.Equal(t, IntegrationType("aws_sso"), IntegrationTypeAWSSSO)
	assert.Equal(t, IntegrationType("saml"), IntegrationTypeSAML)
}

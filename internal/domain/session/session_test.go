package session

import (
	"testing"
	"time"

	"github.com/jvendramin/accio/internal/domain/credential"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	sess := NewSession("Test Session", ProviderAWS, SessionTypeIAMUser, "test-profile", "us-east-1")

	assert.NotEmpty(t, sess.ID)
	assert.Equal(t, "Test Session", sess.Name)
	assert.Equal(t, ProviderAWS, sess.Provider)
	assert.Equal(t, SessionTypeIAMUser, sess.Type)
	assert.Equal(t, "test-profile", sess.ProfileName)
	assert.Equal(t, "us-east-1", sess.Region)
	assert.Equal(t, StatusInactive, sess.Status)
	assert.False(t, sess.Metadata.CreatedAt.IsZero())
}

func TestSession_Validate_IAMUser(t *testing.T) {
	tests := []struct {
		name        string
		session     *Session
		expectError bool
	}{
		{
			name: "valid IAM user session",
			session: &Session{
				Name:        "Test",
				ProfileName: "test-profile",
				Type:        SessionTypeIAMUser,
				Config: SessionConfig{
					IAMUser: &IAMUserConfig{
						AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			session: &Session{
				ProfileName: "test-profile",
				Type:        SessionTypeIAMUser,
				Config: SessionConfig{
					IAMUser: &IAMUserConfig{
						AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
					},
				},
			},
			expectError: true,
		},
		{
			name: "missing profile name",
			session: &Session{
				Name: "Test",
				Type: SessionTypeIAMUser,
				Config: SessionConfig{
					IAMUser: &IAMUserConfig{
						AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
					},
				},
			},
			expectError: true,
		},
		{
			name: "missing access key",
			session: &Session{
				Name:        "Test",
				ProfileName: "test-profile",
				Type:        SessionTypeIAMUser,
				Config: SessionConfig{
					IAMUser: &IAMUserConfig{},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSession_Validate_AWSSSO(t *testing.T) {
	tests := []struct {
		name        string
		session     *Session
		expectError bool
	}{
		{
			name: "valid SSO session",
			session: &Session{
				Name:        "Test",
				ProfileName: "test-profile",
				Type:        SessionTypeAWSSSO,
				Config: SessionConfig{
					AWSSSO: &AWSSSOConfig{
						StartURL:  "https://my-sso.awsapps.com/start",
						Region:    "us-east-1",
						AccountID: "123456789012",
						RoleName:  "Admin",
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing start URL",
			session: &Session{
				Name:        "Test",
				ProfileName: "test-profile",
				Type:        SessionTypeAWSSSO,
				Config: SessionConfig{
					AWSSSO: &AWSSSOConfig{
						AccountID: "123456789012",
						RoleName:  "Admin",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSession_StartStop(t *testing.T) {
	sess := NewSession("Test", ProviderAWS, SessionTypeIAMUser, "test", "us-east-1")
	sess.Config.IAMUser = &IAMUserConfig{AccessKeyID: "AKIA..."}

	cred := &credential.Credential{
		AccessKeyID:     "AKIA...",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      time.Now().Add(1 * time.Hour),
	}

	// Start session
	err := sess.Start(cred)
	require.NoError(t, err)
	assert.Equal(t, StatusActive, sess.Status)
	assert.True(t, sess.IsActive())
	assert.NotNil(t, sess.GetCredential())

	// Can't start an already active session
	err = sess.Start(cred)
	assert.Error(t, err)

	// Stop session
	err = sess.Stop()
	require.NoError(t, err)
	assert.Equal(t, StatusInactive, sess.Status)
	assert.False(t, sess.IsActive())
	assert.Nil(t, sess.GetCredential())
}

func TestSession_NeedsRefresh(t *testing.T) {
	sess := NewSession("Test", ProviderAWS, SessionTypeIAMUser, "test", "us-east-1")

	// No credential = needs refresh
	assert.True(t, sess.NeedsRefresh(5*time.Minute))

	// Set credential expiring soon
	cred := &credential.Credential{
		AccessKeyID:     "AKIA...",
		SecretAccessKey: "secret",
		Expiration:      time.Now().Add(3 * time.Minute),
	}
	sess.SetCredential(cred)

	// Should need refresh within 5 minutes
	assert.True(t, sess.NeedsRefresh(5*time.Minute))

	// Should not need refresh within 1 minute
	assert.False(t, sess.NeedsRefresh(1*time.Minute))
}

func TestSession_RequiresMFA(t *testing.T) {
	sess := NewSession("Test", ProviderAWS, SessionTypeIAMUser, "test", "us-east-1")
	sess.Config.IAMUser = &IAMUserConfig{
		AccessKeyID: "AKIA...",
	}

	// No MFA serial = doesn't require MFA
	assert.False(t, sess.RequiresMFA())

	// With MFA serial = requires MFA
	sess.Config.IAMUser.MFASerial = "arn:aws:iam::123456789012:mfa/user"
	assert.True(t, sess.RequiresMFA())
}

func TestSession_GetSessionDuration(t *testing.T) {
	sess := NewSession("Test", ProviderAWS, SessionTypeIAMUser, "test", "us-east-1")
	sess.Config.IAMUser = &IAMUserConfig{
		AccessKeyID: "AKIA...",
	}

	// Default duration
	assert.Equal(t, 3600, sess.GetSessionDuration())

	// Custom duration
	sess.Config.IAMUser.SessionDuration = 7200
	assert.Equal(t, 7200, sess.GetSessionDuration())
}

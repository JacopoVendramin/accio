package credential

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredential_IsExpired(t *testing.T) {
	tests := []struct {
		name       string
		expiration time.Time
		expected   bool
	}{
		{
			name:       "not expired",
			expiration: time.Now().Add(1 * time.Hour),
			expected:   false,
		},
		{
			name:       "expired",
			expiration: time.Now().Add(-1 * time.Hour),
			expected:   true,
		},
		{
			name:       "no expiration (static credentials)",
			expiration: time.Time{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				AccessKeyID:     "AKIA...",
				SecretAccessKey: "secret",
				Expiration:      tt.expiration,
			}
			assert.Equal(t, tt.expected, cred.IsExpired())
		})
	}
}

func TestCredential_IsExpiringSoon(t *testing.T) {
	tests := []struct {
		name       string
		expiration time.Time
		within     time.Duration
		expected   bool
	}{
		{
			name:       "expiring within window",
			expiration: time.Now().Add(3 * time.Minute),
			within:     5 * time.Minute,
			expected:   true,
		},
		{
			name:       "not expiring within window",
			expiration: time.Now().Add(10 * time.Minute),
			within:     5 * time.Minute,
			expected:   false,
		},
		{
			name:       "no expiration",
			expiration: time.Time{},
			within:     5 * time.Minute,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				AccessKeyID:     "AKIA...",
				SecretAccessKey: "secret",
				Expiration:      tt.expiration,
			}
			assert.Equal(t, tt.expected, cred.IsExpiringSoon(tt.within))
		})
	}
}

func TestCredential_IsTemporary(t *testing.T) {
	tests := []struct {
		name         string
		sessionToken string
		expected     bool
	}{
		{
			name:         "temporary credential",
			sessionToken: "token123",
			expected:     true,
		},
		{
			name:         "static credential",
			sessionToken: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := &Credential{
				AccessKeyID:     "AKIA...",
				SecretAccessKey: "secret",
				SessionToken:    tt.sessionToken,
			}
			assert.Equal(t, tt.expected, cred.IsTemporary())
		})
	}
}

func TestCredential_ToJSON(t *testing.T) {
	cred := &Credential{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "token123",
		Expiration:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Region:          "us-east-1",
	}

	data, err := cred.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), "AKIAIOSFODNN7EXAMPLE")
	assert.Contains(t, string(data), "token123")

	// Parse back
	parsed, err := FromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, cred.AccessKeyID, parsed.AccessKeyID)
	assert.Equal(t, cred.SecretAccessKey, parsed.SecretAccessKey)
	assert.Equal(t, cred.SessionToken, parsed.SessionToken)
	assert.Equal(t, cred.Region, parsed.Region)
}

func TestCredential_ToCredentialProcessOutput(t *testing.T) {
	expiration := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	cred := &Credential{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "token123",
		Expiration:      expiration,
	}

	output := cred.ToCredentialProcessOutput()
	assert.Equal(t, 1, output.Version)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", output.AccessKeyId)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", output.SecretAccessKey)
	assert.Equal(t, "token123", output.SessionToken)
	assert.Equal(t, "2024-01-01T12:00:00Z", output.Expiration)
}

func TestCredential_TimeUntilExpiry(t *testing.T) {
	// No expiration
	cred := &Credential{
		AccessKeyID:     "AKIA...",
		SecretAccessKey: "secret",
	}
	assert.Equal(t, time.Duration(0), cred.TimeUntilExpiry())

	// With expiration
	cred.Expiration = time.Now().Add(30 * time.Minute)
	remaining := cred.TimeUntilExpiry()
	assert.True(t, remaining > 29*time.Minute && remaining <= 30*time.Minute)
}

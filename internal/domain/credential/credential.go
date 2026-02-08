// Package credential defines the credential value object and related types.
package credential

import (
	"encoding/json"
	"time"
)

// Credential represents AWS credentials.
type Credential struct {
	AccessKeyID     string    `json:"access_key_id"`
	SecretAccessKey string    `json:"secret_access_key"`
	SessionToken    string    `json:"session_token,omitempty"`
	Expiration      time.Time `json:"expiration,omitempty"`
	Region          string    `json:"region,omitempty"`
}

// IsExpired returns true if the credential has expired.
func (c *Credential) IsExpired() bool {
	if c.Expiration.IsZero() {
		return false // Static credentials don't expire
	}
	return time.Now().After(c.Expiration)
}

// IsExpiringSoon returns true if the credential will expire within the given duration.
func (c *Credential) IsExpiringSoon(within time.Duration) bool {
	if c.Expiration.IsZero() {
		return false
	}
	return time.Now().Add(within).After(c.Expiration)
}

// TimeUntilExpiry returns the duration until the credential expires.
func (c *Credential) TimeUntilExpiry() time.Duration {
	if c.Expiration.IsZero() {
		return 0
	}
	return time.Until(c.Expiration)
}

// IsTemporary returns true if this is a temporary/session credential.
func (c *Credential) IsTemporary() bool {
	return c.SessionToken != ""
}

// ToJSON serializes the credential to JSON.
func (c *Credential) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON deserializes a credential from JSON.
func FromJSON(data []byte) (*Credential, error) {
	var cred Credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

// CredentialProcessOutput represents the output format for AWS credential_process.
type CredentialProcessOutput struct {
	Version         int    `json:"Version"`
	AccessKeyId     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

// ToCredentialProcessOutput converts a credential to the credential_process format.
func (c *Credential) ToCredentialProcessOutput() *CredentialProcessOutput {
	out := &CredentialProcessOutput{
		Version:         1,
		AccessKeyId:     c.AccessKeyID,
		SecretAccessKey: c.SecretAccessKey,
		SessionToken:    c.SessionToken,
	}
	if !c.Expiration.IsZero() {
		out.Expiration = c.Expiration.Format(time.RFC3339)
	}
	return out
}

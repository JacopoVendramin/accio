// Package session defines the session entity and related types.
package session

import (
	"time"
)

// SessionType represents the type of AWS session.
type SessionType string

const (
	SessionTypeIAMUser SessionType = "iam_user"
	SessionTypeAWSSSO  SessionType = "aws_sso"
	SessionTypeIAMRole SessionType = "iam_role_chained"
	SessionTypeSAML    SessionType = "saml_federation"
)

// SessionStatus represents the current status of a session.
type SessionStatus string

const (
	StatusInactive SessionStatus = "inactive"
	StatusActive   SessionStatus = "active"
	StatusExpiring SessionStatus = "expiring"
	StatusError    SessionStatus = "error"
	StatusPending  SessionStatus = "pending" // Waiting for user action (e.g., MFA, SSO login)
)

// Provider represents a cloud provider.
type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderAzure Provider = "azure" // Future
	ProviderGCP   Provider = "gcp"   // Future
)

// IAMUserConfig holds configuration specific to IAM User sessions.
type IAMUserConfig struct {
	AccessKeyID     string `json:"access_key_id"`
	MFASerial       string `json:"mfa_serial,omitempty"`
	SessionDuration int    `json:"session_duration,omitempty"` // Seconds, default 3600
}

// AWSSSOConfig holds configuration specific to AWS SSO sessions.
type AWSSSOConfig struct {
	StartURL      string `json:"start_url"`
	Region        string `json:"region"`
	AccountID     string `json:"account_id"`
	AccountName   string `json:"account_name,omitempty"`
	AccountEmail  string `json:"account_email,omitempty"`
	RoleName      string `json:"role_name"`
	IntegrationID string `json:"integration_id,omitempty"`
}

// IAMRoleChainedConfig holds configuration for role chaining.
type IAMRoleChainedConfig struct {
	ParentSessionID string `json:"parent_session_id"`
	RoleARN         string `json:"role_arn"`
	ExternalID      string `json:"external_id,omitempty"`
	SessionDuration int    `json:"session_duration,omitempty"`
}

// SAMLConfig holds configuration for SAML federation.
type SAMLConfig struct {
	IntegrationID string `json:"integration_id"`
	RoleARN       string `json:"role_arn"`
	PrincipalARN  string `json:"principal_arn"`
}

// SessionConfig is a union type for session-specific configuration.
type SessionConfig struct {
	IAMUser *IAMUserConfig        `json:"iam_user,omitempty"`
	AWSSSO  *AWSSSOConfig         `json:"aws_sso,omitempty"`
	IAMRole *IAMRoleChainedConfig `json:"iam_role,omitempty"`
	SAML    *SAMLConfig           `json:"saml,omitempty"`
}

// SessionMetadata holds non-essential session metadata.
type SessionMetadata struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastUsedAt  time.Time `json:"last_used_at,omitempty"`
	Color       string    `json:"color,omitempty"`
	Description string    `json:"description,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

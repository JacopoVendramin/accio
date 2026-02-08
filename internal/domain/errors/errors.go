// Package errors defines domain-specific errors for the application.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common domain error conditions.
var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionAlreadyActive = errors.New("session is already active")
	ErrSessionNotActive     = errors.New("session is not active")
	ErrCredentialNotFound   = errors.New("credential not found")
	ErrCredentialExpired    = errors.New("credential has expired")
	ErrInvalidSessionType   = errors.New("invalid session type")
	ErrInvalidConfig        = errors.New("invalid configuration")
	ErrParentSessionFailed  = errors.New("parent session failed")
	ErrMFARequired          = errors.New("MFA token required")
	ErrSSOLoginRequired     = errors.New("SSO login required")
	ErrIntegrationNotFound  = errors.New("integration not found")
	ErrStorageFailure       = errors.New("secure storage operation failed")
	ErrProviderNotFound     = errors.New("provider not found")
)

// DomainError wraps an error with additional context.
type DomainError struct {
	Op      string // Operation that failed
	Kind    error  // Category of error
	Err     error  // Underlying error
	Context map[string]string
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v: %v", e.Op, e.Kind, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Kind)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func (e *DomainError) Is(target error) bool {
	return errors.Is(e.Kind, target)
}

// NewDomainError creates a new domain error.
func NewDomainError(op string, kind error, err error) *DomainError {
	return &DomainError{
		Op:   op,
		Kind: kind,
		Err:  err,
	}
}

// WithContext adds context to a domain error.
func (e *DomainError) WithContext(key, value string) *DomainError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

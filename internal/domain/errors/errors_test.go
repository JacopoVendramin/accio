package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DomainError
		expected string
	}{
		{
			name: "with underlying error",
			err: &DomainError{
				Op:   "GetSession",
				Kind: ErrSessionNotFound,
				Err:  errors.New("database error"),
			},
			expected: "GetSession: session not found: database error",
		},
		{
			name: "without underlying error",
			err: &DomainError{
				Op:   "GetSession",
				Kind: ErrSessionNotFound,
			},
			expected: "GetSession: session not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestDomainError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &DomainError{
		Op:   "Test",
		Kind: ErrSessionNotFound,
		Err:  underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
}

func TestDomainError_Is(t *testing.T) {
	err := &DomainError{
		Op:   "GetSession",
		Kind: ErrSessionNotFound,
	}

	assert.True(t, errors.Is(err, ErrSessionNotFound))
	assert.False(t, errors.Is(err, ErrCredentialNotFound))
}

func TestNewDomainError(t *testing.T) {
	underlying := errors.New("connection failed")
	err := NewDomainError("Connect", ErrStorageFailure, underlying)

	assert.Equal(t, "Connect", err.Op)
	assert.Equal(t, ErrStorageFailure, err.Kind)
	assert.Equal(t, underlying, err.Err)
}

func TestDomainError_WithContext(t *testing.T) {
	err := NewDomainError("GetSession", ErrSessionNotFound, nil)

	// Add context
	err = err.WithContext("sessionID", "abc123")
	assert.Equal(t, "abc123", err.Context["sessionID"])

	// Add more context
	err = err.WithContext("provider", "aws")
	assert.Equal(t, "aws", err.Context["provider"])
	assert.Equal(t, "abc123", err.Context["sessionID"])
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are distinct
	errs := []error{
		ErrSessionNotFound,
		ErrSessionAlreadyActive,
		ErrSessionNotActive,
		ErrCredentialNotFound,
		ErrCredentialExpired,
		ErrInvalidSessionType,
		ErrInvalidConfig,
		ErrParentSessionFailed,
		ErrMFARequired,
		ErrSSOLoginRequired,
		ErrIntegrationNotFound,
		ErrStorageFailure,
		ErrProviderNotFound,
	}

	// Check each error has a non-empty message
	for _, err := range errs {
		assert.NotEmpty(t, err.Error())
	}

	// Check errors are not equal to each other
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			assert.NotEqual(t, errs[i], errs[j])
		}
	}
}

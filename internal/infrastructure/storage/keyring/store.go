// Package keyring provides secure credential storage using the OS keychain.
package keyring

import (
	"encoding/json"
	"fmt"

	"github.com/99designs/keyring"
	"github.com/jvendramin/accio/internal/domain/credential"
	"github.com/jvendramin/accio/internal/domain/errors"
)

const (
	serviceName      = "accio"
	credentialPrefix = "cred:"
	secretPrefix     = "secret:"
)

// SecureStore provides secure storage using the OS keychain.
type SecureStore interface {
	// StoreCredential stores a credential for a session.
	StoreCredential(sessionID string, cred *credential.Credential) error

	// GetCredential retrieves a credential for a session.
	GetCredential(sessionID string) (*credential.Credential, error)

	// DeleteCredential removes a credential for a session.
	DeleteCredential(sessionID string) error

	// StoreSecret stores an arbitrary secret.
	StoreSecret(key string, value []byte) error

	// GetSecret retrieves an arbitrary secret.
	GetSecret(key string) ([]byte, error)

	// DeleteSecret removes a secret.
	DeleteSecret(key string) error

	// ListCredentials returns all stored credential session IDs.
	ListCredentials() ([]string, error)

	// Clear removes all stored credentials and secrets.
	Clear() error
}

// KeyringStore implements SecureStore using 99designs/keyring.
type KeyringStore struct {
	ring keyring.Keyring
}

// Config holds configuration for the keyring store.
type Config struct {
	// ServiceName is the name used in the keychain
	ServiceName string
	// KeychainName is the macOS keychain name (optional)
	KeychainName string
	// FileDir is the directory for file-based backend (Linux fallback)
	FileDir string
	// FilePasswordFunc returns the password for file-based backend
	FilePasswordFunc func(string) (string, error)
}

// NewKeyringStore creates a new keyring-based secure store.
func NewKeyringStore(cfg Config) (*KeyringStore, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = serviceName
	}

	ringCfg := keyring.Config{
		ServiceName: cfg.ServiceName,
		// macOS Keychain
		KeychainName:             cfg.KeychainName,
		KeychainTrustApplication: true,
		// Linux Secret Service / Windows Credential Manager
		// File-based fallback for environments without keychain
		FileDir:          cfg.FileDir,
		FilePasswordFunc: cfg.FilePasswordFunc,
	}

	// Try to open keyring with preferred backends
	ring, err := keyring.Open(ringCfg)
	if err != nil {
		return nil, errors.NewDomainError("NewKeyringStore", errors.ErrStorageFailure, err)
	}

	return &KeyringStore{ring: ring}, nil
}

// StoreCredential stores a credential for a session.
func (s *KeyringStore) StoreCredential(sessionID string, cred *credential.Credential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return errors.NewDomainError("StoreCredential", errors.ErrStorageFailure, err)
	}

	key := credentialPrefix + sessionID
	if err := s.ring.Set(keyring.Item{
		Key:  key,
		Data: data,
	}); err != nil {
		return errors.NewDomainError("StoreCredential", errors.ErrStorageFailure, err)
	}

	return nil
}

// GetCredential retrieves a credential for a session.
func (s *KeyringStore) GetCredential(sessionID string) (*credential.Credential, error) {
	key := credentialPrefix + sessionID
	item, err := s.ring.Get(key)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, errors.NewDomainError("GetCredential", errors.ErrCredentialNotFound, nil).
				WithContext("sessionID", sessionID)
		}
		return nil, errors.NewDomainError("GetCredential", errors.ErrStorageFailure, err)
	}

	var cred credential.Credential
	if err := json.Unmarshal(item.Data, &cred); err != nil {
		return nil, errors.NewDomainError("GetCredential", errors.ErrStorageFailure, err)
	}

	return &cred, nil
}

// DeleteCredential removes a credential for a session.
func (s *KeyringStore) DeleteCredential(sessionID string) error {
	key := credentialPrefix + sessionID
	if err := s.ring.Remove(key); err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil // Already deleted
		}
		return errors.NewDomainError("DeleteCredential", errors.ErrStorageFailure, err)
	}
	return nil
}

// StoreSecret stores an arbitrary secret.
func (s *KeyringStore) StoreSecret(key string, value []byte) error {
	fullKey := secretPrefix + key
	if err := s.ring.Set(keyring.Item{
		Key:  fullKey,
		Data: value,
	}); err != nil {
		return errors.NewDomainError("StoreSecret", errors.ErrStorageFailure, err)
	}
	return nil
}

// GetSecret retrieves an arbitrary secret.
func (s *KeyringStore) GetSecret(key string) ([]byte, error) {
	fullKey := secretPrefix + key
	item, err := s.ring.Get(fullKey)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, errors.NewDomainError("GetSecret", errors.ErrCredentialNotFound, nil).
				WithContext("key", key)
		}
		return nil, errors.NewDomainError("GetSecret", errors.ErrStorageFailure, err)
	}
	return item.Data, nil
}

// DeleteSecret removes a secret.
func (s *KeyringStore) DeleteSecret(key string) error {
	fullKey := secretPrefix + key
	if err := s.ring.Remove(fullKey); err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil
		}
		return errors.NewDomainError("DeleteSecret", errors.ErrStorageFailure, err)
	}
	return nil
}

// ListCredentials returns all stored credential session IDs.
func (s *KeyringStore) ListCredentials() ([]string, error) {
	keys, err := s.ring.Keys()
	if err != nil {
		return nil, errors.NewDomainError("ListCredentials", errors.ErrStorageFailure, err)
	}

	var sessionIDs []string
	prefixLen := len(credentialPrefix)
	for _, key := range keys {
		if len(key) > prefixLen && key[:prefixLen] == credentialPrefix {
			sessionIDs = append(sessionIDs, key[prefixLen:])
		}
	}

	return sessionIDs, nil
}

// Clear removes all stored credentials and secrets.
func (s *KeyringStore) Clear() error {
	keys, err := s.ring.Keys()
	if err != nil {
		return errors.NewDomainError("Clear", errors.ErrStorageFailure, err)
	}

	var lastErr error
	for _, key := range keys {
		if err := s.ring.Remove(key); err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return errors.NewDomainError("Clear", errors.ErrStorageFailure, lastErr)
	}
	return nil
}

// CredentialKey returns the keyring key for a session credential.
func CredentialKey(sessionID string) string {
	return fmt.Sprintf("%s%s", credentialPrefix, sessionID)
}

// SecretKey returns the keyring key for a secret.
func SecretKey(key string) string {
	return fmt.Sprintf("%s%s", secretPrefix, key)
}

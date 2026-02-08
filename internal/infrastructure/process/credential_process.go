// Package process provides the credential_process server.
package process

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jvendramin/accio/internal/application/session"
)

// CredentialProcess handles credential_process requests from AWS CLI.
type CredentialProcess struct {
	sessionService *session.Service
}

// NewCredentialProcess creates a new credential process handler.
func NewCredentialProcess(sessionService *session.Service) *CredentialProcess {
	return &CredentialProcess{
		sessionService: sessionService,
	}
}

// Run executes the credential_process for a given profile.
func (cp *CredentialProcess) Run(ctx context.Context, profileName string) error {
	// Find the session by profile name
	sess, err := cp.sessionService.GetByProfileName(ctx, profileName)
	if err != nil {
		return fmt.Errorf("failed to find session for profile %s: %w", profileName, err)
	}

	// Get credentials (will auto-refresh if needed)
	cred, err := cp.sessionService.GetCredential(ctx, sess.ID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Convert to credential_process output format
	output := cred.ToCredentialProcessOutput()

	// Marshal to JSON and write to stdout
	data, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	_, err = os.Stdout.Write(data)
	return err
}

// RunWithArgs executes the credential_process using command line arguments.
// Expected format: accio credential-process --profile <profile-name>
func RunWithArgs(sessionService *session.Service, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: accio credential-process --profile <profile-name>")
	}

	var profileName string
	for i := 0; i < len(args); i++ {
		if args[i] == "--profile" && i+1 < len(args) {
			profileName = args[i+1]
			break
		}
	}

	if profileName == "" {
		return fmt.Errorf("--profile argument is required")
	}

	cp := NewCredentialProcess(sessionService)
	return cp.Run(context.Background(), profileName)
}

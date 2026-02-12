// Package main provides the entry point for the accio application.
package main

import (
	"fmt"
	"os"

	integrationApp "github.com/jvendramin/accio/internal/application/integration"
	"github.com/jvendramin/accio/internal/application/session"
	"github.com/jvendramin/accio/internal/config"
	"github.com/jvendramin/accio/internal/infrastructure/aws"
	"github.com/jvendramin/accio/internal/infrastructure/process"
	configStorage "github.com/jvendramin/accio/internal/infrastructure/storage/config"
	"github.com/jvendramin/accio/internal/infrastructure/storage/keyring"
	"github.com/jvendramin/accio/internal/tui"
)

// Build-time variables (set via ldflags)
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Handle flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			printUsage()
			return nil
		case "-v", "--version", "version":
			fmt.Printf("accio %s (commit: %s, built: %s)\n", version, commit, buildDate)
			return nil
		case "credential-process":
			return runCredentialProcess(os.Args[2:])
		}
	}

	// Load configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		// Continue with defaults if config can't be loaded
		fmt.Fprintf(os.Stderr, "Warning: Could not load config: %v\n", err)
	}
	cfg := configManager.Get()

	// Initialize secure storage
	store, err := keyring.NewKeyringStore(keyring.Config{
		ServiceName: cfg.Storage.KeyringService,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize secure storage: %w", err)
	}

	// Initialize session repository
	sessionRepo, err := configStorage.NewSessionRepository(cfg.Storage.SessionFile)
	if err != nil {
		return fmt.Errorf("failed to initialize session repository: %w", err)
	}

	// Initialize integration repository
	integrationRepo, err := configStorage.NewIntegrationRepository(cfg.Storage.IntegrationFile)
	if err != nil {
		return fmt.Errorf("failed to initialize integration repository: %w", err)
	}

	// Initialize session service
	sessionService := session.NewService(
		sessionRepo,
		store,
		cfg.RefreshBeforeExpiry,
		cfg.SessionInactivityTimeout,
	)

	// Initialize integration service
	integrationService := integrationApp.NewService(
		integrationRepo,
		sessionRepo,
		store,
	)

	// Register unified AWS provider
	awsProvider := aws.NewProvider(store, integrationRepo, sessionRepo)
	sessionService.RegisterProvider(awsProvider)

	// Run TUI
	return tui.Run(sessionService, integrationService, configManager)
}

func runCredentialProcess(args []string) error {
	// Load configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg := configManager.Get()

	// Initialize secure storage
	store, err := keyring.NewKeyringStore(keyring.Config{
		ServiceName: cfg.Storage.KeyringService,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize secure storage: %w", err)
	}

	// Initialize session repository
	sessionRepo, err := configStorage.NewSessionRepository(cfg.Storage.SessionFile)
	if err != nil {
		return fmt.Errorf("failed to initialize session repository: %w", err)
	}

	// Initialize integration repository
	integrationRepo, err := configStorage.NewIntegrationRepository(cfg.Storage.IntegrationFile)
	if err != nil {
		return fmt.Errorf("failed to initialize integration repository: %w", err)
	}

	// Initialize session service
	sessionService := session.NewService(
		sessionRepo,
		store,
		cfg.RefreshBeforeExpiry,
		cfg.SessionInactivityTimeout,
	)

	// Register unified AWS provider
	awsProvider := aws.NewProvider(store, integrationRepo, sessionRepo)
	sessionService.RegisterProvider(awsProvider)

	// Run credential process
	return process.RunWithArgs(sessionService, args)
}

func printUsage() {
	fmt.Println("accio - Summon your AWS credentials")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  accio                              Start the TUI")
	fmt.Println("  accio credential-process           Get credentials for AWS CLI")
	fmt.Println("    --profile <name>                 AWS profile name")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help                         Show this help")
	fmt.Println("  -v, --version                      Show version")
	fmt.Println()
	fmt.Println("Key Bindings (Session List):")
	fmt.Println("  ↑/↓ or j/k    Navigate sessions")
	fmt.Println("  enter         Start/Stop session")
	fmt.Println("  n             Create new session")
	fmt.Println("  i             Manage integrations (SSO)")
	fmt.Println("  e             Edit session")
	fmt.Println("  d             Delete session")
	fmt.Println("  s             Settings")
	fmt.Println("  ?             Help")
	fmt.Println("  q             Quit")
}

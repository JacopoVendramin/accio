// Package awsconfig provides AWS config file management.
package awsconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jvendramin/accio/internal/domain/session"
)

const (
	managedProfileMarker = "# Managed by accio"
)

// Manager manages AWS config and credentials files.
type Manager struct {
	configPath      string
	credentialsPath string
	binaryPath      string
	useCredProcess  bool
}

// NewManager creates a new AWS config manager.
func NewManager(configPath, credentialsPath, binaryPath string, useCredProcess bool) *Manager {
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".aws", "config")
	}
	if credentialsPath == "" {
		home, _ := os.UserHomeDir()
		credentialsPath = filepath.Join(home, ".aws", "credentials")
	}
	if binaryPath == "" {
		binaryPath, _ = os.Executable()
	}

	return &Manager{
		configPath:      configPath,
		credentialsPath: credentialsPath,
		binaryPath:      binaryPath,
		useCredProcess:  useCredProcess,
	}
}

// WriteProfile writes a profile configuration for a session.
func (m *Manager) WriteProfile(sess *session.Session) error {
	if m.useCredProcess {
		return m.writeProfileWithCredProcess(sess)
	}
	return m.writeProfileWithCredentials(sess)
}

// writeProfileWithCredProcess writes a profile using credential_process.
func (m *Manager) writeProfileWithCredProcess(sess *session.Session) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0700); err != nil {
		return err
	}

	// Read existing config
	profiles, err := m.readProfiles(m.configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Build profile section
	profileName := sess.ProfileName
	if !strings.HasPrefix(profileName, "profile ") {
		profileName = "profile " + profileName
	}

	profileContent := []string{
		managedProfileMarker,
		fmt.Sprintf("credential_process = %s credential-process --profile %s", m.binaryPath, sess.ProfileName),
	}
	if sess.Region != "" {
		profileContent = append(profileContent, fmt.Sprintf("region = %s", sess.Region))
	}

	profiles[profileName] = profileContent

	// Write back
	return m.writeProfiles(m.configPath, profiles)
}

// writeProfileWithCredentials writes credentials directly.
func (m *Manager) writeProfileWithCredentials(sess *session.Session) error {
	// This is a fallback method - generally we prefer credential_process
	// Write to credentials file directly
	if err := os.MkdirAll(filepath.Dir(m.credentialsPath), 0700); err != nil {
		return err
	}

	profiles, err := m.readProfiles(m.credentialsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Profile section will be updated when credentials are set
	profileContent := []string{
		managedProfileMarker,
		"# Credentials will be set when session is started",
	}

	profiles[sess.ProfileName] = profileContent

	// Also write region to config file
	if sess.Region != "" {
		if err := os.MkdirAll(filepath.Dir(m.configPath), 0700); err != nil {
			return err
		}

		configProfiles, err := m.readProfiles(m.configPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		profileName := "profile " + sess.ProfileName
		configProfiles[profileName] = []string{
			managedProfileMarker,
			fmt.Sprintf("region = %s", sess.Region),
		}

		if err := m.writeProfiles(m.configPath, configProfiles); err != nil {
			return err
		}
	}

	return m.writeProfiles(m.credentialsPath, profiles)
}

// RemoveProfile removes a profile configuration.
func (m *Manager) RemoveProfile(profileName string) error {
	// Remove from config file
	profiles, err := m.readProfiles(m.configPath)
	if err == nil {
		// Default profile uses [default], other profiles use [profile name]
		configKey := profileName
		if profileName != "default" {
			configKey = "profile " + profileName
		}
		delete(profiles, configKey)
		if err := m.writeProfiles(m.configPath, profiles); err != nil {
			return err
		}
	}

	// Remove from credentials file
	profiles, err = m.readProfiles(m.credentialsPath)
	if err == nil {
		delete(profiles, profileName)
		if err := m.writeProfiles(m.credentialsPath, profiles); err != nil {
			return err
		}
	}

	return nil
}

// readProfiles reads profiles from an INI-style file.
func (m *Manager) readProfiles(path string) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return make(map[string][]string), err
	}
	defer file.Close()

	profiles := make(map[string][]string)
	var currentProfile string
	var currentLines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			// Save previous profile
			if currentProfile != "" {
				profiles[currentProfile] = currentLines
			}
			// Start new profile
			currentProfile = trimmed[1 : len(trimmed)-1]
			currentLines = []string{}
		} else if currentProfile != "" {
			currentLines = append(currentLines, line)
		}
	}

	// Save last profile
	if currentProfile != "" {
		profiles[currentProfile] = currentLines
	}

	return profiles, scanner.Err()
}

// writeProfiles writes profiles to an INI-style file.
func (m *Manager) writeProfiles(path string, profiles map[string][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write profiles
	first := true
	for name, lines := range profiles {
		if !first {
			file.WriteString("\n")
		}
		first = false

		file.WriteString(fmt.Sprintf("[%s]\n", name))
		for _, line := range lines {
			file.WriteString(line + "\n")
		}
	}

	return nil
}

// IsManagedProfile checks if a profile is managed by accio.
func (m *Manager) IsManagedProfile(profileName string) (bool, error) {
	// Check config file
	profiles, err := m.readProfiles(m.configPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	checkProfile := func(lines []string) bool {
		for _, line := range lines {
			if strings.Contains(line, managedProfileMarker) {
				return true
			}
		}
		return false
	}

	// Default profile uses [default], other profiles use [profile name]
	configKey := profileName
	if profileName != "default" {
		configKey = "profile " + profileName
	}

	if lines, ok := profiles[configKey]; ok {
		if checkProfile(lines) {
			return true, nil
		}
	}

	// Check credentials file
	profiles, err = m.readProfiles(m.credentialsPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	if lines, ok := profiles[profileName]; ok {
		return checkProfile(lines), nil
	}

	return false, nil
}

// GetConfigPath returns the AWS config file path.
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetCredentialsPath returns the AWS credentials file path.
func (m *Manager) GetCredentialsPath() string {
	return m.credentialsPath
}

// WriteCredentials writes actual credentials to the credentials file.
func (m *Manager) WriteCredentials(profileName, accessKeyID, secretAccessKey, sessionToken, region string) error {
	if err := os.MkdirAll(filepath.Dir(m.credentialsPath), 0700); err != nil {
		return err
	}

	profiles, err := m.readProfiles(m.credentialsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Build credentials section
	credLines := []string{
		managedProfileMarker,
		fmt.Sprintf("aws_access_key_id = %s", accessKeyID),
		fmt.Sprintf("aws_secret_access_key = %s", secretAccessKey),
	}
	if sessionToken != "" {
		credLines = append(credLines, fmt.Sprintf("aws_session_token = %s", sessionToken))
	}

	profiles[profileName] = credLines

	if err := m.writeProfiles(m.credentialsPath, profiles); err != nil {
		return err
	}

	// Also write region to config file if provided
	if region != "" {
		if err := os.MkdirAll(filepath.Dir(m.configPath), 0700); err != nil {
			return err
		}

		configProfiles, err := m.readProfiles(m.configPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		// Default profile uses [default], other profiles use [profile name]
		profileKey := profileName
		if profileName != "default" {
			profileKey = "profile " + profileName
		}

		// Check if profile already has config
		if existing, ok := configProfiles[profileKey]; ok {
			// Update region in existing config
			updated := false
			for i, line := range existing {
				if strings.HasPrefix(strings.TrimSpace(line), "region") {
					existing[i] = fmt.Sprintf("region = %s", region)
					updated = true
					break
				}
			}
			if !updated {
				existing = append(existing, fmt.Sprintf("region = %s", region))
			}
			configProfiles[profileKey] = existing
		} else {
			configProfiles[profileKey] = []string{
				managedProfileMarker,
				fmt.Sprintf("region = %s", region),
			}
		}

		if err := m.writeProfiles(m.configPath, configProfiles); err != nil {
			return err
		}
	}

	return nil
}

// ClearCredentials removes credentials for a profile from the credentials file.
func (m *Manager) ClearCredentials(profileName string) error {
	profiles, err := m.readProfiles(m.credentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Check if this is a managed profile before removing
	if lines, ok := profiles[profileName]; ok {
		isManaged := false
		for _, line := range lines {
			if strings.Contains(line, managedProfileMarker) {
				isManaged = true
				break
			}
		}
		if isManaged {
			delete(profiles, profileName)
			return m.writeProfiles(m.credentialsPath, profiles)
		}
	}

	return nil
}

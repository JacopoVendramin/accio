# accio

<p align="center">
  <img src="assets/accio-logo.png" alt="accio" width="400">
</p>

**accio** is a terminal-based AWS credentials manager with a beautiful TUI. It securely manages multiple AWS sessions across different authentication methods.

## Features

- **Multiple Authentication Methods**
  - IAM User (with MFA support)
  - AWS SSO / Identity Center
  - IAM Role Chaining
  - SAML Federation

- **Secure Storage**
  - Credentials stored in OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
  - Never written to disk in plain text

- **AWS CLI Integration**
  - Automatic `~/.aws/credentials` management
  - `credential_process` support for seamless CLI integration
  - Writes to both named profile and `default` profile

- **Terminal UI**
  - Beautiful, responsive TUI built with Bubble Tea
  - Keyboard-driven workflow
  - Session status at a glance

## Installation

### From Source

```bash
go install github.com/jvendramin/accio/cmd/accio@latest
```

### Build from Source

```bash
git clone https://github.com/jvendramin/accio.git
cd accio
go build -o accio ./cmd/accio
```

### macOS

When running a downloaded binary for the first time, macOS may show a warning: *"accio cannot be verified"*. This happens because the binary is not signed with an Apple Developer certificate.

To allow execution, remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine ./accio
```

## Usage

### Start the TUI

```bash
accio
```

### Key Bindings

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate sessions |
| `Enter` | Start/Stop session |
| `n` | Create new session |
| `i` | Manage integrations (SSO) |
| `e` | Edit session |
| `d` | Delete session |
| `s` | Settings |
| `?` | Help |
| `q` | Quit |

### Credential Process

accio can be used as a `credential_process` for the AWS CLI:

```bash
accio credential-process --profile my-profile
```

In your `~/.aws/config`:

```ini
[profile my-profile]
credential_process = /path/to/accio credential-process --profile my-profile
```

## Configuration

Configuration is stored in `~/.accio/config.yaml`:

```yaml
default_region: us-east-1
default_session_duration: 1h
refresh_before_expiry: 5m

ui:
  show_timestamps: true
  show_region: true
  theme: default

storage:
  keyring_service: accio
```

### Environment Variables

- `ACCIO_CONFIG` - Path to config file
- `ACCIO_*` - Override any config option (e.g., `ACCIO_DEFAULT_REGION=eu-west-1`)

## Session Types

### IAM User

Basic AWS credentials with optional MFA:

1. Press `n` to create new session
2. Select "IAM User"
3. Enter Access Key ID and Secret Access Key
4. Optionally configure MFA device ARN

### AWS SSO

Integrate with AWS IAM Identity Center:

1. Press `i` to open integrations
2. Add new SSO integration with your start URL
3. Authenticate in browser
4. Select accounts and roles to import as sessions

### IAM Role

Chain roles from an existing session:

1. Press `n` to create new session
2. Select "IAM Role"
3. Choose parent session
4. Enter Role ARN to assume

## Security

- All secrets stored in OS keychain
- Session tokens are temporary and auto-rotated
- No credentials written to disk in plain text
- MFA tokens never stored

## License

MIT License - see [LICENSE](LICENSE) for details.

// Package sso provides AWS SSO operations.
package sso

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/jvendramin/accio/internal/domain/credential"
	"github.com/jvendramin/accio/pkg/provider"
)

const (
	clientName = "accio"
	clientType = "public"
	grantType  = "urn:ietf:params:oauth:grant-type:device_code"
)

// Client wraps AWS SSO and SSO OIDC clients.
type Client struct {
	region string
}

// NewClient creates a new SSO client.
func NewClient(region string) *Client {
	if region == "" {
		region = "us-east-1"
	}
	return &Client{region: region}
}

// StartDeviceAuthorization begins the device authorization flow.
func (c *Client) StartDeviceAuthorization(ctx context.Context, startURL string) (*provider.DeviceAuthorizationResponse, error) {
	// Create SSO OIDC client
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	oidcClient := ssooidc.NewFromConfig(cfg)

	// Register the client
	registerOutput, err := oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(clientName),
		ClientType: aws.String(clientType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	// Start device authorization
	authOutput, err := oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerOutput.ClientId,
		ClientSecret: registerOutput.ClientSecret,
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %w", err)
	}

	return &provider.DeviceAuthorizationResponse{
		ClientID:                *registerOutput.ClientId,
		ClientSecret:            *registerOutput.ClientSecret,
		DeviceCode:              *authOutput.DeviceCode,
		UserCode:                *authOutput.UserCode,
		VerificationURI:         *authOutput.VerificationUri,
		VerificationURIComplete: *authOutput.VerificationUriComplete,
		ExpiresIn:               int(authOutput.ExpiresIn),
		Interval:                int(authOutput.Interval),
	}, nil
}

// PollForToken polls for the access token after user authorization.
func (c *Client) PollForToken(ctx context.Context, clientID, clientSecret, deviceCode string) (*provider.SSOToken, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	oidcClient := ssooidc.NewFromConfig(cfg)

	// Create token request
	tokenOutput, err := oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     aws.String(clientID),
		ClientSecret: aws.String(clientSecret),
		DeviceCode:   aws.String(deviceCode),
		GrantType:    aws.String(grantType),
	})
	if err != nil {
		return nil, err // Return raw error for authorization_pending handling
	}

	expiresAt := time.Now().Add(time.Duration(tokenOutput.ExpiresIn) * time.Second).Unix()

	return &provider.SSOToken{
		AccessToken:  *tokenOutput.AccessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: aws.ToString(tokenOutput.RefreshToken),
	}, nil
}

// ListAccounts returns available accounts for the SSO portal.
func (c *Client) ListAccounts(ctx context.Context, accessToken string) ([]provider.SSOAccount, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)

	var accounts []provider.SSOAccount
	var nextToken *string

	for {
		output, err := ssoClient.ListAccounts(ctx, &sso.ListAccountsInput{
			AccessToken: aws.String(accessToken),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, acct := range output.AccountList {
			accounts = append(accounts, provider.SSOAccount{
				AccountID:    aws.ToString(acct.AccountId),
				AccountName:  aws.ToString(acct.AccountName),
				EmailAddress: aws.ToString(acct.EmailAddress),
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return accounts, nil
}

// ListAccountRoles returns available roles for an account.
func (c *Client) ListAccountRoles(ctx context.Context, accessToken, accountID string) ([]provider.SSORole, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)

	var roles []provider.SSORole
	var nextToken *string

	for {
		output, err := ssoClient.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
			AccessToken: aws.String(accessToken),
			AccountId:   aws.String(accountID),
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list account roles: %w", err)
		}

		for _, role := range output.RoleList {
			roles = append(roles, provider.SSORole{
				RoleName:  aws.ToString(role.RoleName),
				AccountID: accountID,
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return roles, nil
}

// GetRoleCredentials gets credentials for a specific role.
func (c *Client) GetRoleCredentials(ctx context.Context, accessToken, accountID, roleName string) (*credential.Credential, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)

	output, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: aws.String(accessToken),
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role credentials: %w", err)
	}

	expiration := time.UnixMilli(output.RoleCredentials.Expiration)

	return &credential.Credential{
		AccessKeyID:     aws.ToString(output.RoleCredentials.AccessKeyId),
		SecretAccessKey: aws.ToString(output.RoleCredentials.SecretAccessKey),
		SessionToken:    aws.ToString(output.RoleCredentials.SessionToken),
		Expiration:      expiration,
		Region:          c.region,
	}, nil
}

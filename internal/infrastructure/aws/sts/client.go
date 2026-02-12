// Package sts provides AWS STS operations.
package sts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/jvendramin/accio/internal/domain/credential"
)

// Client wraps the AWS STS client.
type Client struct {
	region string
}

// NewClient creates a new STS client.
func NewClient(region string) *Client {
	if region == "" {
		region = "us-east-1"
	}
	return &Client{region: region}
}

// GetSessionToken gets temporary credentials using IAM user credentials.
func (c *Client) GetSessionToken(
	ctx context.Context,
	accessKeyID, secretAccessKey string,
	durationSeconds int32,
	mfaSerial, mfaToken string,
) (*credential.Credential, error) {
	// Create STS client with static credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	client := sts.NewFromConfig(cfg)

	// Build request
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(durationSeconds),
	}

	// Add MFA if provided
	if mfaSerial != "" && mfaToken != "" {
		input.SerialNumber = aws.String(mfaSerial)
		input.TokenCode = aws.String(mfaToken)
	}

	// Get session token
	result, err := client.GetSessionToken(ctx, input)
	if err != nil {
		// Enhance error with more details from AWS SDK
		return nil, enhanceAWSError(err)
	}

	return &credential.Credential{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          c.region,
	}, nil
}

// AssumeRole assumes an IAM role.
func (c *Client) AssumeRole(
	ctx context.Context,
	cred *credential.Credential,
	roleARN, sessionName string,
	durationSeconds int32,
	externalID string,
) (*credential.Credential, error) {
	// Create STS client with provided credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cred.AccessKeyID,
			cred.SecretAccessKey,
			cred.SessionToken,
		)),
	)
	if err != nil {
		return nil, err
	}

	client := sts.NewFromConfig(cfg)

	// Build request
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
		DurationSeconds: aws.Int32(durationSeconds),
	}

	if externalID != "" {
		input.ExternalId = aws.String(externalID)
	}

	// Assume role
	result, err := client.AssumeRole(ctx, input)
	if err != nil {
		return nil, err
	}

	return &credential.Credential{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          c.region,
	}, nil
}

// AssumeRoleWithSAML assumes a role using SAML assertion.
func (c *Client) AssumeRoleWithSAML(
	ctx context.Context,
	principalARN, roleARN, samlAssertion string,
	durationSeconds int32,
) (*credential.Credential, error) {
	// Create STS client without credentials (SAML assertion provides auth)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, err
	}

	client := sts.NewFromConfig(cfg)

	// Assume role with SAML
	result, err := client.AssumeRoleWithSAML(ctx, &sts.AssumeRoleWithSAMLInput{
		PrincipalArn:    aws.String(principalARN),
		RoleArn:         aws.String(roleARN),
		SAMLAssertion:   aws.String(samlAssertion),
		DurationSeconds: aws.Int32(durationSeconds),
	})
	if err != nil {
		return nil, err
	}

	return &credential.Credential{
		AccessKeyID:     *result.Credentials.AccessKeyId,
		SecretAccessKey: *result.Credentials.SecretAccessKey,
		SessionToken:    *result.Credentials.SessionToken,
		Expiration:      *result.Credentials.Expiration,
		Region:          c.region,
	}, nil
}

// GetCallerIdentity returns information about the current caller.
func (c *Client) GetCallerIdentity(ctx context.Context, cred *credential.Credential) (*CallerIdentity, error) {
	// Create STS client with provided credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cred.AccessKeyID,
			cred.SecretAccessKey,
			cred.SessionToken,
		)),
	)
	if err != nil {
		return nil, err
	}

	client := sts.NewFromConfig(cfg)

	result, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	return &CallerIdentity{
		Account: *result.Account,
		ARN:     *result.Arn,
		UserID:  *result.UserId,
	}, nil
}

// CallerIdentity contains information about the AWS caller.
type CallerIdentity struct {
	Account string
	ARN     string
	UserID  string
}

// ValidateCredentials validates that credentials are still valid.
func (c *Client) ValidateCredentials(ctx context.Context, cred *credential.Credential) error {
	_, err := c.GetCallerIdentity(ctx, cred)
	return err
}

// DefaultSessionDuration returns the default session duration.
func DefaultSessionDuration() time.Duration {
	return time.Hour
}

// MaxSessionDuration returns the maximum session duration for GetSessionToken.
func MaxSessionDuration() time.Duration {
	return 36 * time.Hour
}

// enhanceAWSError extracts detailed error information from AWS SDK errors.
func enhanceAWSError(err error) error {
	if err == nil {
		return nil
	}

	// Try to extract API error details
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%s: %s (Code: %s)", apiErr.ErrorMessage(), err.Error(), apiErr.ErrorCode())
	}

	// Return original error if we can't enhance it
	return err
}

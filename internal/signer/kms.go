// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package signer

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KMSProvider implements SignerProvider for AWS KMS-backed signing.
// Keys are stored in AWS KMS and never exported, providing hardware-level
// security for audit log signatures.
type KMSProvider struct{}

// Ensure KMSProvider implements SignerProvider at compile time.
var _ SignerProvider = (*KMSProvider)(nil)

// Name returns the provider identifier for AWS KMS.
func (p *KMSProvider) Name() string {
	return "aws-kms"
}

// Description returns a human-readable description.
func (p *KMSProvider) Description() string {
	return "AWS Key Management Service (KMS) for audit log signing"
}

// Validate checks that the KMS configuration is valid.
func (p *KMSProvider) Validate(cfg ProviderConfig) error {
	keyID := cfg.Extra["kms_key_id"]
	if keyID == "" {
		return errors.New("AWS KMS provider requires kms_key_id in Extra config")
	}
	return nil
}

// Create instantiates a KMS-backed signer from the configuration.
func (p *KMSProvider) Create(cfg ProviderConfig) (Signer, error) {
	keyID := cfg.Extra["kms_key_id"]
	region := cfg.Extra["kms_region"]
	if region == "" {
		region = "us-east-1"
	}

	// Build AWS config
	awsCfg, err := buildAWSConfig(cfg, region)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create KMS client
	kmsClient := kms.NewFromConfig(awsCfg)

	return &KMSSigner{
		client:  kmsClient,
		keyID:   keyID,
		keyIDHex: hex.EncodeToString([]byte(keyID)),
	}, nil
}

// EnvVars returns the environment variables for KMS configuration.
func (p *KMSProvider) EnvVars() []EnvVarDoc {
	return []EnvVarDoc{
		{Name: "GLASSBOX_AWS_KMS_KEY_ID", Required: true, Description: "AWS KMS key ID (alias or ARN)"},
		{Name: "GLASSBOX_AWS_KMS_REGION", Required: false, Description: "AWS region (default: us-east-1)"},
		{Name: "AWS_ACCESS_KEY_ID", Required: false, Description: "AWS access key for authentication"},
		{Name: "AWS_SECRET_ACCESS_KEY", Required: false, Description: "AWS secret key for authentication"},
		{Name: "AWS_PROFILE", Required: false, Description: "AWS credential profile name"},
	}
}

// buildAWSConfig creates an AWS config from provider config.
func buildAWSConfig(cfg ProviderConfig, region string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	// Check for explicit credentials in Extra config
	if accessKey := cfg.Extra["aws_access_key_id"]; accessKey != "" {
		secretKey := cfg.Extra["aws_secret_access_key"]
		if secretKey == "" {
			return aws.Config{}, errors.New("AWS access key provided but secret key missing")
		}
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	} else if profile := cfg.Extra["aws_profile"]; profile != "" {
		// Use profile from config (takes precedence over env)
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	// Otherwise, use default credential chain (env, config file, EC2 role, etc.)

	return config.LoadDefaultConfig(context.Background(), opts...)
}

// KMSSigner implements Signer using AWS KMS.
type KMSSigner struct {
	client    *kms.Client
	keyID     string
	keyIDHex  string
}

// Sign signs a message using the KMS key.
// The hash is computed over the message and passed to KMS for signing.
// Note: AWS KMS doesn't support Ed25519 directly. We use ECDSA_SHA_512
// (P-521 curve with SHA-512) which is the closest available option.
func (s *KMSSigner) Sign(message []byte) ([]byte, error) {
	// Compute SHA-512 hash of the message (KMS supports SIGN_DIGEST_SHA_256 and SHA_512)
	// For ECDSA with SHA-512, we hash the message ourselves
	digest := crypto.SHA512.New()
	digest.Write(message)
	h := digest.Sum(nil)

	// Call KMS Sign API
	input := &kms.SignInput{
		KeyId:            &s.keyID,
		Message:          h,
		MessageType:      kmsTypes.MessageTypeDigest,
		SigningAlgorithm: kmsTypesSigningAlgorithm, // ECDSA_SHA_512
	}

	result, err := s.client.Sign(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("KMS sign request failed: %w", err)
	}

	return result.Signature, nil
}

// PublicKey returns the public key associated with the KMS key.
// This queries KMS for the public key on each call to ensure freshness.
func (s *KMSSigner) PublicKey() ([]byte, error) {
	input := &kms.GetPublicKeyInput{
		KeyId: &s.keyID,
	}

	result, err := s.client.GetPublicKey(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from KMS: %w", err)
	}

	return result.PublicKey, nil
}

// KeyID returns the KMS key ID used for signing.
func (s *KMSSigner) KeyID() string {
	return s.keyID
}

// Algorithm returns the signing algorithm.
func (s *KMSSigner) Algorithm() string {
	return "ECDSA_SHA_512"
}

// Close implements io.Closer. KMS doesn't require cleanup, but we implement
// this interface for consistency with other providers.
func (s *KMSSigner) Close() error {
	return nil
}

// kmsTypesSigningAlgorithm is the signing algorithm for KMS.
// AWS KMS doesn't support Ed25519 directly; ECDSA_SHA_512 uses P-521 curve with SHA-512
// which is the closest available option for SHA-512 based signatures.
var kmsTypesSigningAlgorithm = kmsTypes.SigningAlgorithmSpecEcdsaSha512

// RegisterKMSProvider registers the AWS KMS provider with the default registry.
// This is called automatically during package initialization if the AWS SDK
// is available.
func init() {
	// Register the KMS provider - this will replace any existing provider with the same name
	DefaultRegistry.Register(&KMSProvider{})
}

// Extra keys for KMS configuration in ProviderConfig:
// - kms_key_id: The KMS key ID (alias or ARN) - REQUIRED
// - kms_region: The AWS region - optional, defaults to us-east-1
// - aws_access_key_id: AWS access key - optional
// - aws_secret_access_key: AWS secret key - optional
// - aws_profile: AWS profile name - optional

// Supported key IDs for KMSProvider:
// - Key alias: "alias/GlassboxAuditKey"
// - Key ARN: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
// - Key ID: "12345678-1234-1234-1234-123456789012"

// Example usage:
//
//	GLASSBOX_SIGNING_PROVIDER=aws-kms \
//	GLASSBOX_AWS_KMS_KEY_ID=alias/GlassboxAuditKey \
//	GLASSBOX_AWS_KMS_REGION=us-east-1 \
//	glassbox audit:sign --payload-file data.json
//
// Or via config file:
//
//	[audit]
//	  signing_provider = "aws-kms"
//
//	[audit.kms]
//	  key_id = "alias/GlassboxAuditKey"
//	  region = "us-east-1"

// Verify KMSSigner implements Signer at compile time.
var _ Signer = (*KMSSigner)(nil)

// Ed25519Verify is a helper that verifies an Ed25519 signature.
// This is used for testing and verification.
func Ed25519Verify(publicKey ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}

// GenerateTestKey generates a test Ed25519 key pair for development.
// This should not be used in production.
func GenerateTestKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return pub, priv, err
}

// ValidateKeyID validates that a KMS key ID is properly formatted.
func ValidateKeyID(keyID string) error {
	if keyID == "" {
		return errors.New("KMS key ID cannot be empty")
	}

	// Check for alias format
	if strings.HasPrefix(keyID, "alias/") {
		return nil
	}

	// Check for ARN format
	if strings.HasPrefix(keyID, "arn:aws:kms:") {
		// Basic ARN validation
		parts := strings.Split(keyID, ":")
		if len(parts) < 6 {
			return errors.New("invalid KMS ARN format")
		}
		return nil
	}

	// Check for UUID format (key ID)
	if len(keyID) == 36 { // Standard UUID length
		return nil
	}

	return errors.New("KMS key ID must be an alias (alias/name), ARN, or key ID")
}
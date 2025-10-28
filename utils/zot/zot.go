// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package zot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("utils/zot")

const (
	// DefaultZotConfigPath is the default path to the zot configuration file.
	DefaultZotConfigPath = "/etc/zot/config.json"

	// DefaultPollInterval is the default interval for polling new content.
	DefaultPollInterval = time.Second * 60

	// DefaultRetryDelay is the default delay between retries.
	DefaultRetryDelay = time.Minute * 5

	// DefaultMaxRetries is the default maximum number of retries.
	DefaultMaxRetries = 3
)

// VerifyConfig contains configuration for zot verification.
type VerifyConfig struct {
	RegistryAddress string
	RepositoryName  string
	Username        string
	Password        string
	AccessToken     string
	Insecure        bool
}

// UploadPublicKeyOptions contains options for uploading public keys to zot.
type UploadPublicKeyOptions struct {
	Config    *VerifyConfig
	PublicKey string
}

// VerificationOptions contains options for zot verification.
type VerificationOptions struct {
	Config    *VerifyConfig
	RecordCID string
}

// VerificationResult contains the result of zot verification.
type VerificationResult struct {
	IsSigned  bool
	IsTrusted bool
	Author    string
	Tool      string
}

// UploadPublicKey uploads a public key to zot for signature verification.
// This enables zot to mark signatures as "trusted" when they can be verified with this key.
func UploadPublicKey(ctx context.Context, opts *UploadPublicKeyOptions) error {
	logger.Debug("Uploading public key to zot for signature verification")

	if opts.PublicKey == "" {
		return errors.New("public key is required")
	}

	// Get registry URL for zot cosign endpoint
	registryURL := buildRegistryURL(opts.Config.RegistryAddress, opts.Config.Insecure)
	uploadEndpoint := registryURL + "/v2/_zot/ext/cosign"

	// Create HTTP request with public key as body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadEndpoint, strings.NewReader(opts.PublicKey))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	// Add authentication
	addAuthentication(req, opts.Config)

	// Create HTTP client and execute request
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload public key: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to upload public key, status: %d, response: %s", resp.StatusCode, string(body))
	}

	logger.Debug("Successfully uploaded public key to zot", "endpoint", uploadEndpoint)

	return nil
}

// Verify queries zot's verification API to check if a signature is valid.
func Verify(ctx context.Context, opts *VerificationOptions) (*VerificationResult, error) {
	// Build zot search endpoint URL
	registryURL := buildRegistryURL(opts.Config.RegistryAddress, opts.Config.Insecure)
	searchEndpoint := registryURL + "/v2/_zot/ext/search"
	logger.Debug("Querying zot for signature verification", "endpoint", searchEndpoint, "recordCID", opts.RecordCID)

	// Create GraphQL query for signature verification
	query := fmt.Sprintf(`{
		Image(image: "%s:%s") {
			Digest
			IsSigned
			Tag
			SignatureInfo {
				Tool
				IsTrusted
				Author
			}
		}
	}`, opts.Config.RepositoryName, opts.RecordCID)

	graphqlQuery := map[string]interface{}{
		"query": query,
	}

	jsonData, err := json.Marshal(graphqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create verification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication
	addAuthentication(req, opts.Config)

	// Create HTTP client and execute request
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query zot verification: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read verification response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Debug("Verification query returned non-success status", "status", resp.StatusCode, "body", string(body))

		return nil, fmt.Errorf("verification query returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse GraphQL response
	var graphqlResp struct {
		Data struct {
			Image struct {
				Digest        string `json:"Digest"`
				IsSigned      bool   `json:"IsSigned"`
				Tag           string `json:"Tag"`
				SignatureInfo []struct {
					Tool      string `json:"Tool"`
					IsTrusted bool   `json:"IsTrusted"`
					Author    string `json:"Author"`
				} `json:"SignatureInfo"`
			} `json:"Image"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode verification response: %w", err)
	}

	// Build result
	result := &VerificationResult{
		IsSigned:  graphqlResp.Data.Image.IsSigned,
		IsTrusted: false,
	}

	// Extract signature info if available
	if len(graphqlResp.Data.Image.SignatureInfo) > 0 {
		sigInfo := graphqlResp.Data.Image.SignatureInfo[0]
		result.IsTrusted = sigInfo.IsTrusted
		result.Author = sigInfo.Author
		result.Tool = sigInfo.Tool
	}

	logger.Debug("Zot verification result", "recordCID", opts.RecordCID, "isSigned", result.IsSigned, "isTrusted", result.IsTrusted)

	return result, nil
}

// buildRegistryURL constructs the registry URL with proper protocol.
func buildRegistryURL(registryURL string, insecure bool) string {
	// If URL already has a protocol, return as-is
	if strings.HasPrefix(registryURL, "http://") || strings.HasPrefix(registryURL, "https://") {
		return registryURL
	}

	// Add appropriate protocol based on insecure flag
	if insecure {
		return "http://" + registryURL
	}

	return "https://" + registryURL
}

// addAuthentication adds authentication headers to HTTP requests.
func addAuthentication(req *http.Request, config *VerifyConfig) {
	if config.Username != "" && config.Password != "" {
		req.SetBasicAuth(config.Username, config.Password)
	} else if config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.AccessToken)
	}
}

// CheckReadiness checks if Zot is ready to serve traffic by querying its /readyz endpoint.
// Returns true if Zot responds with 200 OK, false otherwise.
func CheckReadiness(ctx context.Context, registryAddress string, insecure bool) bool {
	// Build URL to Zot's readiness endpoint
	registryURL := buildRegistryURL(registryAddress, insecure)
	readyzURL := registryURL + "/readyz"

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyzURL, nil)
	if err != nil {
		logger.Debug("Failed to create readiness check request", "error", err, "url", readyzURL)

		return false
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second, //nolint:mnd
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("Zot readiness check failed", "error", err, "url", readyzURL)

		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Debug("Zot readiness check passed", "address", registryAddress)

		return true
	}

	logger.Debug("Zot not ready", "address", registryAddress, "status", resp.StatusCode)

	return false
}

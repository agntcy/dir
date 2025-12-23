// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultTokenCacheDir is the default directory for storing cached tokens.
	DefaultTokenCacheDir = ".config/dirctl"

	// TokenCacheFile is the filename for the cached token.
	TokenCacheFile = "github-token.json"

	// DefaultTokenValidityDuration is how long a token is considered valid if no expiry is set.
	DefaultTokenValidityDuration = 8 * time.Hour

	// TokenExpiryBuffer is how much time before actual expiry we consider a token expired.
	TokenExpiryBuffer = 5 * time.Minute
)

// CachedToken represents a cached GitHub OAuth token.
type CachedToken struct {
	// AccessToken is the GitHub access token.
	AccessToken string `json:"access_token"`

	// TokenType is the token type (usually "bearer").
	TokenType string `json:"token_type,omitempty"`

	// ExpiresAt is when the token expires.
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// User is the authenticated GitHub username.
	User string `json:"user,omitempty"`

	// Orgs are the user's GitHub organizations.
	Orgs []string `json:"orgs,omitempty"`

	// CreatedAt is when the token was cached.
	CreatedAt time.Time `json:"created_at"`
}

// TokenCache manages cached GitHub OAuth tokens.
type TokenCache struct {
	// CacheDir is the directory where tokens are stored.
	CacheDir string
}

// NewTokenCache creates a new token cache with the default directory.
func NewTokenCache() *TokenCache {
	home, _ := os.UserHomeDir()
	return &TokenCache{
		CacheDir: filepath.Join(home, DefaultTokenCacheDir),
	}
}

// NewTokenCacheWithDir creates a new token cache with a custom directory.
func NewTokenCacheWithDir(dir string) *TokenCache {
	return &TokenCache{CacheDir: dir}
}

// GetCachePath returns the full path to the token cache file.
func (c *TokenCache) GetCachePath() string {
	return filepath.Join(c.CacheDir, TokenCacheFile)
}

// Load loads the cached token from disk.
// Returns nil if no valid token is found.
func (c *TokenCache) Load() (*CachedToken, error) {
	path := c.GetCachePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cached token
		}
		return nil, fmt.Errorf("failed to read token cache: %w", err)
	}

	var token CachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token cache: %w", err)
	}

	return &token, nil
}

// Save saves a token to the cache.
func (c *TokenCache) Save(token *CachedToken) error {
	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(c.CacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Set creation time if not set
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	path := c.GetCachePath()
	// Write with secure permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token cache: %w", err)
	}

	return nil
}

// Clear removes the cached token.
func (c *TokenCache) Clear() error {
	path := c.GetCachePath()
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token cache: %w", err)
	}
	return nil
}

// IsValid checks if a cached token is still valid.
// A token is considered valid if it exists and hasn't expired.
func (c *TokenCache) IsValid(token *CachedToken) bool {
	if token == nil || token.AccessToken == "" {
		return false
	}

	// If no expiry set, assume valid for DefaultTokenValidityDuration from creation
	if token.ExpiresAt.IsZero() {
		defaultExpiry := token.CreatedAt.Add(DefaultTokenValidityDuration)
		return time.Now().Before(defaultExpiry)
	}

	// Check if token has expired (with buffer)
	return time.Now().Add(TokenExpiryBuffer).Before(token.ExpiresAt)
}

// GetValidToken returns a valid cached token or nil if none exists.
func (c *TokenCache) GetValidToken() (*CachedToken, error) {
	token, err := c.Load()
	if err != nil {
		return nil, err
	}

	if !c.IsValid(token) {
		return nil, nil
	}

	return token, nil
}


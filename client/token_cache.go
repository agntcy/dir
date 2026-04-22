// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultTokenCacheDir is the default directory for storing cached tokens (relative to home directory).
	// When XDG_CONFIG_HOME is set, tokens are stored at $XDG_CONFIG_HOME/dirctl instead.
	//nolint:gosec // G101: This is a directory path, not a credential
	DefaultTokenCacheDir = ".config/dirctl"

	// TokenCacheFile is the filename for the cached token.
	//nolint:gosec // G101: This is a filename, not a credential
	TokenCacheFile = "auth-token.json"

	// TokenCacheSubdir is where issuer-scoped token cache files are stored.
	TokenCacheSubdir = "tokens"

	// DefaultTokenValidityDuration is how long a token is considered valid if no expiry is set.
	DefaultTokenValidityDuration = 8 * time.Hour

	// TokenExpiryBuffer is how much time before actual expiry we consider a token expired.
	TokenExpiryBuffer = 5 * time.Minute

	// File and directory permissions for secure token storage.
	cacheDirPerms  = 0o700 // Owner read/write/execute only
	cacheFilePerms = 0o600 // Owner read/write only
)

var (
	// ErrTokenCacheNotFound indicates that a specific token cache file does not exist.
	ErrTokenCacheNotFound = errors.New("token cache not found")

	// ErrNoCachedIssuer indicates that no issuer-scoped token cache could be selected.
	ErrNoCachedIssuer = errors.New("no cached OIDC issuer found")
)

// CachedToken represents a cached authentication token from any provider.
type CachedToken struct {
	// AccessToken is the authentication token.
	AccessToken string `json:"access_token"` // #nosec G117: intentional field - for cached token

	// TokenType is the token type (usually "bearer").
	TokenType string `json:"token_type,omitempty"`

	// Provider is the authentication provider (oidc, github, google, azure, etc.)
	Provider string `json:"provider,omitempty"`

	// Issuer is the OIDC issuer URL (for Provider=oidc). Enables multi-issuer support.
	Issuer string `json:"issuer,omitempty"`

	// RefreshToken is the refresh token, if the IdP returned one (for token refresh).
	RefreshToken string `json:"refresh_token,omitempty"` // #nosec G101: intentional field - for cached token

	// ExpiresAt is when the token expires.
	ExpiresAt time.Time `json:"expires_at,omitzero"`

	// User is the authenticated username (e.g. preferred_username or name for OIDC).
	User string `json:"user,omitempty"`

	// UserID is the provider-specific user ID (e.g. sub for OIDC).
	UserID string `json:"user_id,omitempty"`

	// Email is the user's email address.
	Email string `json:"email,omitempty"`

	// CreatedAt is when the token was cached.
	CreatedAt time.Time `json:"created_at"`
}

// TokenCacheEntry is an alias for CachedToken (for compatibility).
type TokenCacheEntry = CachedToken

// TokenCache manages cached authentication tokens from any provider.
type TokenCache struct {
	// CacheDir is the directory where tokens are stored.
	CacheDir string

	// CacheFile is the cache filename inside CacheDir.
	CacheFile string

	// Issuer is the issuer this cache is scoped to, if any.
	Issuer string
}

// NewTokenCache creates a new token cache with the default directory.
// Respects XDG_CONFIG_HOME environment variable for config directory location.
func NewTokenCache() *TokenCache {
	return &TokenCache{
		CacheDir:  defaultTokenCacheRoot(),
		CacheFile: TokenCacheFile,
	}
}

// NewTokenCacheWithDir creates a new token cache with a custom directory.
func NewTokenCacheWithDir(dir string) *TokenCache {
	return &TokenCache{
		CacheDir:  dir,
		CacheFile: TokenCacheFile,
	}
}

// NewTokenCacheWithDirAndIssuer creates a token cache for a specific issuer using a custom root directory.
func NewTokenCacheWithDirAndIssuer(dir, issuer string) (*TokenCache, error) {
	normalizedIssuer, err := normalizeIssuer(issuer)
	if err != nil {
		return nil, err
	}

	return &TokenCache{
		CacheDir:  filepath.Join(dir, TokenCacheSubdir),
		CacheFile: issuerCacheFileName(normalizedIssuer),
		Issuer:    normalizedIssuer,
	}, nil
}

// NewTokenCacheForIssuer creates a token cache scoped to a specific issuer.
func NewTokenCacheForIssuer(issuer string) (*TokenCache, error) {
	return NewTokenCacheWithDirAndIssuer(defaultTokenCacheRoot(), issuer)
}

// GetCachePath returns the full path to the token cache file.
func (c *TokenCache) GetCachePath() string {
	return filepath.Join(c.CacheDir, c.cacheFileName())
}

// Load loads the cached token from disk.
// Returns nil if no valid token is found.
func (c *TokenCache) Load() (*CachedToken, error) {
	path := c.GetCachePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			//nolint:nilnil // Returning (nil, nil) is idiomatic for "not found" - not an error condition
			return nil, nil // No cached token
		}

		return nil, fmt.Errorf("failed to read token cache: %w", err)
	}

	var token CachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token cache: %w", err)
	}

	if c.Issuer != "" {
		normalizedIssuer, err := normalizeIssuer(token.Issuer)
		if err != nil {
			return nil, fmt.Errorf("failed to validate cached token issuer: %w", err)
		}

		if normalizedIssuer != c.Issuer {
			return nil, fmt.Errorf("cached token issuer mismatch: expected %s, got %s", c.Issuer, token.Issuer)
		}
	}

	return &token, nil
}

// Save saves a token to the cache.
func (c *TokenCache) Save(token *CachedToken) error {
	return c.save(token, false)
}

// SaveAtomic saves a token using atomic replacement to prevent partial cache writes.
func (c *TokenCache) SaveAtomic(token *CachedToken) error {
	return c.save(token, true)
}

func (c *TokenCache) save(token *CachedToken, atomic bool) error {
	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(c.CacheDir, cacheDirPerms); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Set creation time if not set
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	data, err := json.MarshalIndent(token, "", "  ") // #nosec G117: intentional field - for cached token
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	path := c.GetCachePath()
	if !atomic {
		// Write with secure permissions (owner read/write only)
		if err := os.WriteFile(path, data, cacheFilePerms); err != nil {
			return fmt.Errorf("failed to write token cache: %w", err)
		}

		return nil
	}

	tmpFile, err := os.CreateTemp(c.CacheDir, c.cacheFileName()+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp token cache file: %w", err)
	}

	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := tmpFile.Chmod(cacheFilePerms); err != nil {
		_ = tmpFile.Close()

		return fmt.Errorf("failed to set temp token cache permissions: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()

		return fmt.Errorf("failed to write temp token cache: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp token cache file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to atomically replace token cache: %w", err)
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

		return time.Now().UTC().Truncate(time.Millisecond).Before(defaultExpiry)
	}

	// Check if token has expired (with buffer)
	return time.Now().UTC().Truncate(time.Millisecond).Add(TokenExpiryBuffer).Before(token.ExpiresAt)
}

// GetValidToken returns a valid cached token or nil if none exists.
func (c *TokenCache) GetValidToken() (*CachedToken, error) {
	token, err := c.Load()
	if err != nil {
		return nil, err
	}

	if !c.IsValid(token) {
		//nolint:nilnil // Returning (nil, nil) is idiomatic for "no valid token" - not an error condition
		return nil, nil
	}

	return token, nil
}

// AmbiguousTokenCacheError indicates multiple issuer-scoped caches exist and the caller must select one.
type AmbiguousTokenCacheError struct {
	Issuers []string
}

func (e *AmbiguousTokenCacheError) Error() string {
	return fmt.Sprintf(
		"multiple cached OIDC issuers found; select one explicitly: %s",
		strings.Join(e.Issuers, ", "),
	)
}

// CachedIssuer describes one issuer-scoped cache file.
type CachedIssuer struct {
	Issuer string
	Path   string
}

// ResolveTokenCacheForIssuer resolves the appropriate token cache.
// If issuer is provided, the corresponding issuer-scoped cache is returned.
// If issuer is empty, the only available issuer cache is returned when exactly one exists.
// Returns ErrNoCachedIssuer when no issuer is provided and no caches exist.
func ResolveTokenCacheForIssuer(issuer string) (*TokenCache, error) {
	if strings.TrimSpace(issuer) != "" {
		return NewTokenCacheForIssuer(issuer)
	}

	cachedIssuers, err := ListCachedIssuers()
	if err != nil {
		return nil, err
	}

	switch len(cachedIssuers) {
	case 0:
		return nil, ErrNoCachedIssuer
	case 1:
		return NewTokenCacheForIssuer(cachedIssuers[0].Issuer)
	default:
		issuers := make([]string, 0, len(cachedIssuers))
		for _, cachedIssuer := range cachedIssuers {
			issuers = append(issuers, cachedIssuer.Issuer)
		}

		return nil, &AmbiguousTokenCacheError{Issuers: issuers}
	}
}

// ListCachedIssuers returns all issuer-scoped token caches on disk.
func ListCachedIssuers() ([]CachedIssuer, error) {
	tokensDir := filepath.Join(defaultTokenCacheRoot(), TokenCacheSubdir)

	entries, err := os.ReadDir(tokensDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list token caches: %w", err)
	}

	cachedIssuers := make([]CachedIssuer, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(tokensDir, entry.Name())

		token, err := loadCachedTokenFromPath(path)
		if err != nil {
			if errors.Is(err, ErrTokenCacheNotFound) {
				continue
			}

			return nil, err
		}

		if token == nil || strings.TrimSpace(token.Issuer) == "" {
			continue
		}

		normalizedIssuer, err := normalizeIssuer(token.Issuer)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize cached token issuer from %s: %w", path, err)
		}

		cachedIssuers = append(cachedIssuers, CachedIssuer{
			Issuer: normalizedIssuer,
			Path:   path,
		})
	}

	sort.Slice(cachedIssuers, func(i, j int) bool {
		return cachedIssuers[i].Issuer < cachedIssuers[j].Issuer
	})

	return cachedIssuers, nil
}

func defaultTokenCacheRoot() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, "dirctl")
}

func (c *TokenCache) cacheFileName() string {
	if c.CacheFile != "" {
		return c.CacheFile
	}

	return TokenCacheFile
}

func issuerCacheFileName(normalizedIssuer string) string {
	sum := sha256.Sum256([]byte(normalizedIssuer))

	return hex.EncodeToString(sum[:]) + ".json"
}

func normalizeIssuer(issuer string) (string, error) {
	trimmedIssuer := strings.TrimSpace(issuer)
	if trimmedIssuer == "" {
		return "", errors.New("issuer is required")
	}

	parsed, err := url.Parse(trimmedIssuer)
	if err != nil {
		return "", fmt.Errorf("invalid issuer URL: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid issuer URL: %s", issuer)
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.RawQuery = ""
	parsed.Fragment = ""

	if parsed.Path == "/" {
		parsed.Path = ""
	} else {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed.String(), nil
}

func loadCachedTokenFromPath(path string) (*CachedToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTokenCacheNotFound
		}

		return nil, fmt.Errorf("failed to read token cache %s: %w", path, err)
	}

	var token CachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token cache %s: %w", path, err)
	}

	return &token, nil
}

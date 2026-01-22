// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenCache(t *testing.T) {
	t.Run("should create cache with default directory", func(t *testing.T) {
		cache := NewTokenCache()

		require.NotNil(t, cache)
		assert.NotEmpty(t, cache.CacheDir)
		assert.Contains(t, cache.CacheDir, DefaultTokenCacheDir)
	})

	t.Run("should include home directory in path", func(t *testing.T) {
		cache := NewTokenCache()

		home, err := os.UserHomeDir()
		require.NoError(t, err)

		expectedPath := filepath.Join(home, DefaultTokenCacheDir)
		assert.Equal(t, expectedPath, cache.CacheDir)
	})
}

func TestNewTokenCacheWithDir(t *testing.T) {
	t.Run("should create cache with custom directory", func(t *testing.T) {
		customDir := "/tmp/test-cache"
		cache := NewTokenCacheWithDir(customDir)

		require.NotNil(t, cache)
		assert.Equal(t, customDir, cache.CacheDir)
	})

	t.Run("should accept empty directory", func(t *testing.T) {
		cache := NewTokenCacheWithDir("")

		require.NotNil(t, cache)
		assert.Empty(t, cache.CacheDir)
	})
}

func TestTokenCache_GetCachePath(t *testing.T) {
	t.Run("should return full path to cache file", func(t *testing.T) {
		customDir := "/tmp/test-cache"
		cache := NewTokenCacheWithDir(customDir)

		path := cache.GetCachePath()

		expectedPath := filepath.Join(customDir, TokenCacheFile)
		assert.Equal(t, expectedPath, path)
	})

	t.Run("should handle trailing slash in directory", func(t *testing.T) {
		cache := NewTokenCacheWithDir("/tmp/test-cache/")

		path := cache.GetCachePath()

		assert.Contains(t, path, TokenCacheFile)
	})
}

func TestTokenCache_Save(t *testing.T) {
	t.Run("should save token to cache", func(t *testing.T) {
		// Create temporary directory for test
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		token := &CachedToken{
			AccessToken: "test_token_123",
			TokenType:   "bearer",
			Provider:    "github",
			User:        "testuser",
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := cache.Save(token)

		require.NoError(t, err)

		// Verify file exists
		path := cache.GetCachePath()
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr)
	})

	t.Run("should create directory if it doesn't exist", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "nested", "dir")
		cache := NewTokenCacheWithDir(tmpDir)

		token := &CachedToken{
			AccessToken: "test_token_123",
		}

		err := cache.Save(token)

		require.NoError(t, err)

		// Verify directory was created
		_, statErr := os.Stat(tmpDir)
		assert.NoError(t, statErr)
	})

	t.Run("should set CreatedAt if not set", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		token := &CachedToken{
			AccessToken: "test_token_123",
			// CreatedAt is zero
		}

		before := time.Now()
		err := cache.Save(token)
		after := time.Now()

		require.NoError(t, err)
		assert.False(t, token.CreatedAt.IsZero())
		assert.True(t, token.CreatedAt.After(before.Add(-time.Second)))
		assert.True(t, token.CreatedAt.Before(after.Add(time.Second)))
	})

	t.Run("should preserve existing CreatedAt", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		createdAt := time.Now().Add(-24 * time.Hour)
		token := &CachedToken{
			AccessToken: "test_token_123",
			CreatedAt:   createdAt,
		}

		err := cache.Save(token)

		require.NoError(t, err)
		assert.Equal(t, createdAt, token.CreatedAt)
	})

	t.Run("should write JSON with proper formatting", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		token := &CachedToken{
			AccessToken: "test_token_123",
			User:        "testuser",
		}

		err := cache.Save(token)
		require.NoError(t, err)

		// Read and verify JSON formatting
		data, readErr := os.ReadFile(cache.GetCachePath())
		require.NoError(t, readErr)

		// Should be indented JSON
		assert.Contains(t, string(data), "\n")
		assert.Contains(t, string(data), "  ")
	})

	t.Run("should set secure file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		token := &CachedToken{
			AccessToken: "test_token_123",
		}

		err := cache.Save(token)
		require.NoError(t, err)

		// Check file permissions
		info, statErr := os.Stat(cache.GetCachePath())
		require.NoError(t, statErr)

		// Should be 0600 (owner read/write only)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})
}

func TestTokenCache_Load(t *testing.T) {
	t.Run("should load token from cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Save token first
		originalToken := &CachedToken{
			AccessToken: "test_token_123",
			TokenType:   "bearer",
			Provider:    "github",
			User:        "testuser",
			Email:       "test@example.com",
			Orgs:        []string{"org1", "org2"},
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := cache.Save(originalToken)
		require.NoError(t, err)

		// Load token
		loadedToken, loadErr := cache.Load()

		require.NoError(t, loadErr)
		require.NotNil(t, loadedToken)
		assert.Equal(t, originalToken.AccessToken, loadedToken.AccessToken)
		assert.Equal(t, originalToken.TokenType, loadedToken.TokenType)
		assert.Equal(t, originalToken.Provider, loadedToken.Provider)
		assert.Equal(t, originalToken.User, loadedToken.User)
		assert.Equal(t, originalToken.Email, loadedToken.Email)
		assert.Equal(t, originalToken.Orgs, loadedToken.Orgs)
	})

	t.Run("should return nil when cache doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		token, err := cache.Load()

		require.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("should error on malformed JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Write invalid JSON
		err := os.MkdirAll(tmpDir, cacheDirPerms)
		require.NoError(t, err)

		invalidJSON := []byte("{invalid json")
		writeErr := os.WriteFile(cache.GetCachePath(), invalidJSON, cacheFilePerms)
		require.NoError(t, writeErr)

		// Try to load
		token, loadErr := cache.Load()

		require.Error(t, loadErr)
		assert.Nil(t, token)
		assert.Contains(t, loadErr.Error(), "failed to parse token cache")
	})

	t.Run("should handle empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Write empty file
		err := os.MkdirAll(tmpDir, cacheDirPerms)
		require.NoError(t, err)

		writeErr := os.WriteFile(cache.GetCachePath(), []byte(""), cacheFilePerms)
		require.NoError(t, writeErr)

		// Try to load
		token, loadErr := cache.Load()

		require.Error(t, loadErr)
		assert.Nil(t, token)
	})
}

func TestTokenCache_Clear(t *testing.T) {
	t.Run("should remove cached token", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Save token first
		token := &CachedToken{AccessToken: "test_token"}
		err := cache.Save(token)
		require.NoError(t, err)

		// Clear cache
		clearErr := cache.Clear()

		require.NoError(t, clearErr)

		// Verify file is gone
		_, statErr := os.Stat(cache.GetCachePath())
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should not error if cache doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Clear non-existent cache
		err := cache.Clear()

		assert.NoError(t, err)
	})

	t.Run("should error if file cannot be removed", func(t *testing.T) {
		// This test is platform-dependent, skip on Windows
		if os.PathSeparator == '\\' {
			t.Skip("Skipping on Windows")
		}

		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Save token
		token := &CachedToken{AccessToken: "test_token"}
		err := cache.Save(token)
		require.NoError(t, err)

		// Make directory read-only to prevent file deletion
		chmodErr := os.Chmod(tmpDir, 0o500)
		require.NoError(t, chmodErr)

		defer func() {
			// Restore permissions for cleanup
			_ = os.Chmod(tmpDir, cacheDirPerms)
		}()

		// Try to clear (should fail)
		clearErr := cache.Clear()

		assert.Error(t, clearErr)
	})
}

func TestTokenCache_IsValid(t *testing.T) {
	t.Run("should return false for nil token", func(t *testing.T) {
		cache := NewTokenCache()

		valid := cache.IsValid(nil)

		assert.False(t, valid)
	})

	t.Run("should return false for empty access token", func(t *testing.T) {
		cache := NewTokenCache()

		token := &CachedToken{
			AccessToken: "",
		}

		valid := cache.IsValid(token)

		assert.False(t, valid)
	})

	t.Run("should return true for valid token with future expiry", func(t *testing.T) {
		cache := NewTokenCache()

		token := &CachedToken{
			AccessToken: "test_token",
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		valid := cache.IsValid(token)

		assert.True(t, valid)
	})

	t.Run("should return false for expired token", func(t *testing.T) {
		cache := NewTokenCache()

		token := &CachedToken{
			AccessToken: "test_token",
			ExpiresAt:   time.Now().Add(-time.Hour),
		}

		valid := cache.IsValid(token)

		assert.False(t, valid)
	})

	t.Run("should return false for token expiring within buffer", func(t *testing.T) {
		cache := NewTokenCache()

		// Token expires in 4 minutes (less than 5-minute buffer)
		token := &CachedToken{
			AccessToken: "test_token",
			ExpiresAt:   time.Now().Add(4 * time.Minute),
		}

		valid := cache.IsValid(token)

		assert.False(t, valid)
	})

	t.Run("should use CreatedAt for tokens without expiry", func(t *testing.T) {
		cache := NewTokenCache()

		// Token created 1 hour ago, should be valid (default 8 hours)
		token := &CachedToken{
			AccessToken: "test_token",
			CreatedAt:   time.Now().Add(-time.Hour),
			// ExpiresAt is zero
		}

		valid := cache.IsValid(token)

		assert.True(t, valid)
	})

	t.Run("should return false for old token without expiry", func(t *testing.T) {
		cache := NewTokenCache()

		// Token created 9 hours ago, should be expired (default 8 hours)
		token := &CachedToken{
			AccessToken: "test_token",
			CreatedAt:   time.Now().Add(-9 * time.Hour),
			// ExpiresAt is zero
		}

		valid := cache.IsValid(token)

		assert.False(t, valid)
	})

	t.Run("should handle token at exact expiry boundary", func(t *testing.T) {
		cache := NewTokenCache()

		// Token expires exactly at buffer time (5 minutes from now)
		token := &CachedToken{
			AccessToken: "test_token",
			ExpiresAt:   time.Now().Add(TokenExpiryBuffer),
		}

		valid := cache.IsValid(token)

		// Should be considered expired (not valid)
		assert.False(t, valid)
	})
}

func TestTokenCache_GetValidToken(t *testing.T) {
	t.Run("should return valid token", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Save valid token
		originalToken := &CachedToken{
			AccessToken: "test_token",
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		err := cache.Save(originalToken)
		require.NoError(t, err)

		// Get valid token
		token, getErr := cache.GetValidToken()

		require.NoError(t, getErr)
		require.NotNil(t, token)
		assert.Equal(t, originalToken.AccessToken, token.AccessToken)
	})

	t.Run("should return nil for expired token", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Save expired token
		expiredToken := &CachedToken{
			AccessToken: "test_token",
			ExpiresAt:   time.Now().Add(-time.Hour),
			CreatedAt:   time.Now().Add(-time.Hour),
		}

		err := cache.Save(expiredToken)
		require.NoError(t, err)

		// Get valid token
		token, getErr := cache.GetValidToken()

		require.NoError(t, getErr)
		assert.Nil(t, token)
	})

	t.Run("should return nil when no cache exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Get valid token without saving first
		token, err := cache.GetValidToken()

		require.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("should return error for malformed cache", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewTokenCacheWithDir(tmpDir)

		// Write invalid JSON
		err := os.MkdirAll(tmpDir, cacheDirPerms)
		require.NoError(t, err)

		writeErr := os.WriteFile(cache.GetCachePath(), []byte("{invalid}"), cacheFilePerms)
		require.NoError(t, writeErr)

		// Try to get valid token
		token, getErr := cache.GetValidToken()

		require.Error(t, getErr)
		assert.Nil(t, token)
	})
}

func TestCachedToken_JSONSerialization(t *testing.T) {
	t.Run("should serialize and deserialize correctly", func(t *testing.T) {
		now := time.Now()
		original := &CachedToken{
			AccessToken: "test_token_123",
			TokenType:   "bearer",
			Provider:    "github",
			ExpiresAt:   now.Add(time.Hour),
			User:        "testuser",
			UserID:      "12345",
			Email:       "test@example.com",
			Orgs:        []string{"org1", "org2"},
			CreatedAt:   now,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded CachedToken

		unmarshalErr := json.Unmarshal(data, &decoded)

		require.NoError(t, unmarshalErr)
		assert.Equal(t, original.AccessToken, decoded.AccessToken)
		assert.Equal(t, original.TokenType, decoded.TokenType)
		assert.Equal(t, original.Provider, decoded.Provider)
		assert.Equal(t, original.User, decoded.User)
		assert.Equal(t, original.UserID, decoded.UserID)
		assert.Equal(t, original.Email, decoded.Email)
		assert.Equal(t, original.Orgs, decoded.Orgs)
		// Time comparison with truncation for JSON precision
		assert.True(t, original.ExpiresAt.Truncate(time.Second).Equal(decoded.ExpiresAt.Truncate(time.Second)))
		assert.True(t, original.CreatedAt.Truncate(time.Second).Equal(decoded.CreatedAt.Truncate(time.Second)))
	})

	t.Run("should omit empty optional fields", func(t *testing.T) {
		token := &CachedToken{
			AccessToken: "test_token",
			CreatedAt:   time.Now(),
		}

		data, err := json.Marshal(token)
		require.NoError(t, err)

		// Should not contain omitted fields
		jsonStr := string(data)
		assert.NotContains(t, jsonStr, "token_type")
		assert.NotContains(t, jsonStr, "provider")
		assert.NotContains(t, jsonStr, "user")
		assert.NotContains(t, jsonStr, "user_id")
		assert.NotContains(t, jsonStr, "email")
		assert.NotContains(t, jsonStr, "orgs")
	})

	t.Run("should handle zero time with omitzero", func(t *testing.T) {
		token := &CachedToken{
			AccessToken: "test_token",
			CreatedAt:   time.Now(),
			// ExpiresAt is zero
		}

		data, err := json.Marshal(token)
		require.NoError(t, err)

		// Should not contain expires_at
		jsonStr := string(data)
		assert.NotContains(t, jsonStr, "expires_at")
	})
}

func TestTokenCacheConstants(t *testing.T) {
	t.Run("should have reasonable default values", func(t *testing.T) {
		assert.Equal(t, ".config/dirctl", DefaultTokenCacheDir)
		assert.Equal(t, "auth-token.json", TokenCacheFile)
		assert.Equal(t, 8*time.Hour, DefaultTokenValidityDuration)
		assert.Equal(t, 5*time.Minute, TokenExpiryBuffer)
	})
}

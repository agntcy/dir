// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"testing"
	"time"

	"github.com/agntcy/dir/pkg/authprovider"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider(nil)

	if got := provider.Name(); got != ProviderNameGithub {
		t.Errorf("Name() = %v, want %v", got, ProviderNameGithub)
	}
}

func TestProvider_ValidateToken(t *testing.T) {
	// Note: These are integration-style tests that would need actual GitHub tokens
	// For unit tests, we'd mock the GitHub API client
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			token:   "not-a-token",
			wantErr: true,
		},
		// Add more test cases as needed
	}

	provider := NewProvider(&Config{
		CacheTTL:   1 * time.Minute,
		APITimeout: 5 * time.Second,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			identity, err := provider.ValidateToken(ctx, tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && identity == nil {
				t.Error("ValidateToken() returned nil identity without error")
			}

			if !tt.wantErr && identity.Provider != "github" {
				t.Errorf("ValidateToken() identity.Provider = %v, want github", identity.Provider)
			}
		})
	}
}

func TestProvider_GetOrgConstructs(t *testing.T) {
	provider := NewProvider(nil)
	ctx := context.Background()

	// Test with invalid token
	_, err := provider.GetOrgConstructs(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}

	// Additional tests would require mocking or actual GitHub tokens
}

func TestProvider_CacheExpiration(t *testing.T) {
	provider := NewProvider(&Config{
		CacheTTL:   100 * time.Millisecond, // Short TTL for testing
		APITimeout: 5 * time.Second,
	})

	// Manually populate cache
	token := "test-token"
	provider.cache[token] = &cacheEntry{
		identity: &authprovider.UserIdentity{
			Provider: "github",
			UserID:   "123",
			Username: "testuser",
		},
		expiresAt: time.Now().Add(50 * time.Millisecond),
	}

	// Should hit cache
	provider.cacheMu.RLock()
	_, ok := provider.cache[token]
	provider.cacheMu.RUnlock()

	if !ok {
		t.Error("Cache entry should exist")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Verify cache logic (entry exists but expired)
	provider.cacheMu.RLock()
	entry, ok := provider.cache[token]
	provider.cacheMu.RUnlock()

	if !ok {
		t.Error("Cache entry should still exist (not yet cleaned up)")
	}

	if entry != nil && time.Now().Before(entry.expiresAt) {
		t.Error("Cache entry should be expired")
	}
}

func TestProvider_IsMemberOfOrgConstruct(t *testing.T) {
	provider := NewProvider(nil)

	// Manually set cache for testing
	token := "test-token"
	provider.cache[token] = &cacheEntry{
		orgConstructs: []authprovider.OrgConstruct{
			{Name: "agntcy", Type: "github-org"},
			{Name: "spiffe", Type: "github-org"},
		},
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	tests := []struct {
		name    string
		orgName string
		want    bool
	}{
		{"member of agntcy", "agntcy", true},
		{"member of spiffe", "spiffe", true},
		{"not member of kubernetes", "kubernetes", false},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.IsMemberOfOrgConstruct(ctx, token, tt.orgName)
			if err != nil {
				t.Errorf("IsMemberOfOrgConstruct() error = %v", err)

				return
			}

			if got != tt.want {
				t.Errorf("IsMemberOfOrgConstruct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_ClearCache(t *testing.T) {
	provider := NewProvider(nil)

	// Add cache entry
	provider.cache["token1"] = &cacheEntry{
		identity:  &authprovider.UserIdentity{Username: "user1"},
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	provider.cache["token2"] = &cacheEntry{
		identity:  &authprovider.UserIdentity{Username: "user2"},
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	if len(provider.cache) != 2 {
		t.Errorf("Expected 2 cache entries, got %d", len(provider.cache))
	}

	// Clear cache
	provider.ClearCache()

	if len(provider.cache) != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", len(provider.cache))
	}
}

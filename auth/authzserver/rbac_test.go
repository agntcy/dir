// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"log/slog"
	"testing"
)

// TestRoleResolver_Authorize tests various authorization scenarios.
func TestRoleResolver_Authorize(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		username    string
		userKey     string
		userOrgs    []string
		apiMethod   string
		expectError bool
		errorMsg    string
	}{
		{
			name: "user with admin role allows all methods",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"github:alice"},
					},
				},
			},
			username:    "alice",
			userKey:     "github:alice",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Push",
			expectError: false,
		},
		{
			name: "user with reader role allows specific methods",
			config: &Config{
				Roles: map[string]Role{
					"reader": {
						AllowedMethods: []string{"/store.StoreService/Pull", "/store.StoreService/Lookup"},
						Users:          []string{"github:bob"},
					},
				},
			},
			username:    "bob",
			userKey:     "github:bob",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Pull",
			expectError: false,
		},
		{
			name: "user with reader role denied for unauthorized method",
			config: &Config{
				Roles: map[string]Role{
					"reader": {
						AllowedMethods: []string{"/store.StoreService/Pull"},
						Users:          []string{"github:bob"},
					},
				},
			},
			username:    "bob",
			userKey:     "github:bob",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Push",
			expectError: true,
		},
		{
			name: "org with admin role allows all methods",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"*"},
						Orgs:           []string{"agntcy"},
					},
				},
			},
			username:    "charlie",
			userKey:     "github:charlie",
			userOrgs:    []string{"agntcy"},
			apiMethod:   "/store.StoreService/Push",
			expectError: false,
		},
		{
			name: "user in deny list is blocked",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"github:alice"},
					},
				},
				UserDenyList: []string{"github:alice"},
			},
			username:    "alice",
			userKey:     "github:alice",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Push",
			expectError: true,
			errorMsg:    "deny list",
		},
		{
			name: "default role allows access",
			config: &Config{
				Roles: map[string]Role{
					"reader": {
						AllowedMethods: []string{"/store.StoreService/Pull"},
					},
				},
				DefaultRole: "reader",
			},
			username:    "eve",
			userKey:     "github:eve",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Pull",
			expectError: false,
		},
		{
			name: "default role denies unauthorized method",
			config: &Config{
				Roles: map[string]Role{
					"reader": {
						AllowedMethods: []string{"/store.StoreService/Pull"},
					},
				},
				DefaultRole: "reader",
			},
			username:    "eve",
			userKey:     "github:eve",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Push",
			expectError: true,
		},
		{
			name: "no role and no default role - deny",
			config: &Config{
				Roles: map[string]Role{},
			},
			username:    "frank",
			userKey:     "github:frank",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/Push",
			expectError: true,
			errorMsg:    "no assigned role",
		},
		{
			name: "user role takes precedence over org role",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"github:grace"},
					},
					"reader": {
						AllowedMethods: []string{"/store.StoreService/Pull"},
						Orgs:           []string{"contributors"},
					},
				},
			},
			username:    "grace",
			userKey:     "github:grace",
			userOrgs:    []string{"contributors"},
			apiMethod:   "/store.StoreService/Push",
			expectError: false, // User role (admin) allows, org role (reader) would deny
		},
		{
			name: "wildcard method matching",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"/store.StoreService/*"},
						Users:          []string{"github:admin"},
					},
				},
			},
			username:    "admin",
			userKey:     "github:admin",
			userOrgs:    []string{},
			apiMethod:   "/store.StoreService/AnyMethod",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create role resolver
			resolver, err := NewRoleResolver(tt.config, slog.Default())
			if err != nil {
				t.Fatalf("failed to create resolver: %v", err)
			}

			// Test authorization
			err = resolver.Authorize(tt.username, tt.userKey, tt.userOrgs, tt.apiMethod)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestRoleResolver_LoadPolicies tests policy loading from config.
func TestRoleResolver_LoadPolicies(t *testing.T) {
	config := &Config{
		Roles: map[string]Role{
			"admin": {
				AllowedMethods: []string{"*"},
				Users:          []string{"github:alice", "bob"}, // Test both formats
				Orgs:           []string{"agntcy"},
			},
			"reader": {
				AllowedMethods: []string{"/store.StoreService/Pull", "/store.StoreService/Lookup"},
				Users:          []string{"github:charlie"},
				Orgs:           []string{"contributors"},
			},
		},
		DefaultRole:  "reader",
		UserDenyList: []string{"github:blocked"},
	}

	resolver, err := NewRoleResolver(config, slog.Default())
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	// Verify that policies were loaded (implicit test via successful creation)
	if resolver.enforcer == nil {
		t.Error("enforcer should not be nil")
	}

	// Test that alice (admin) can access any method
	err = resolver.Authorize("alice", "github:alice", []string{}, "/store.StoreService/Push")
	if err != nil {
		t.Errorf("alice should be authorized for Push: %v", err)
	}

	// Test that bob (admin, without "github:" prefix) can access any method
	err = resolver.Authorize("bob", "github:bob", []string{}, "/store.StoreService/Push")
	if err != nil {
		t.Errorf("bob should be authorized for Push: %v", err)
	}

	// Test that charlie (reader) can only Pull
	err = resolver.Authorize("charlie", "github:charlie", []string{}, "/store.StoreService/Pull")
	if err != nil {
		t.Errorf("charlie should be authorized for Pull: %v", err)
	}

	err = resolver.Authorize("charlie", "github:charlie", []string{}, "/store.StoreService/Push")
	if err == nil {
		t.Error("charlie should NOT be authorized for Push")
	}

	// Test org-based access
	err = resolver.Authorize("dave", "github:dave", []string{"agntcy"}, "/store.StoreService/Push")
	if err != nil {
		t.Errorf("dave (agntcy org) should be authorized for Push: %v", err)
	}
}

// TestRoleResolver_UserDenyList tests that deny list takes precedence.
func TestRoleResolver_UserDenyList(t *testing.T) {
	config := &Config{
		Roles: map[string]Role{
			"admin": {
				AllowedMethods: []string{"*"},
				Users:          []string{"github:blocked"},
			},
		},
		UserDenyList: []string{"github:blocked"},
	}

	resolver, err := NewRoleResolver(config, slog.Default())
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	// Even though user is admin, deny list should block
	err = resolver.Authorize("blocked", "github:blocked", []string{}, "/store.StoreService/Push")
	if err == nil {
		t.Error("blocked user should be denied even with admin role")
	}

	if !contains(err.Error(), "deny list") {
		t.Errorf("error should mention deny list, got: %v", err)
	}
}

// TestConfig_Validate tests configuration validation.
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"github:alice"},
					},
				},
				DefaultRole: "admin",
			},
			expectError: false,
		},
		{
			name: "invalid default role",
			config: &Config{
				Roles: map[string]Role{
					"admin": {
						AllowedMethods: []string{"*"},
					},
				},
				DefaultRole: "nonexistent",
			},
			expectError: true,
			errorMsg:    "not defined",
		},
		{
			name: "nil roles map is valid (auto-initialized)",
			config: &Config{
				Roles: nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

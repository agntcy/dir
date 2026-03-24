// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"log/slog"
	"strings"
	"testing"
)

//nolint:gocognit // test table with multiple branches; complexity acceptable for tests
func TestOIDCRoleResolver_Authorize(t *testing.T) {
	tests := []struct {
		name        string
		config      *OIDCConfig
		principal   string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "user with admin role allows all methods",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"user:https://tenant.zitadel.cloud:77776025198584418"},
					},
				},
			},
			principal:   "user:https://tenant.zitadel.cloud:77776025198584418",
			path:        "/agntcy.dir.store.v1.StoreService/Push",
			expectError: false,
		},
		{
			name: "viewer allows Pull and Lookup only",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"viewer": {
						AllowedMethods: []string{
							"/agntcy.dir.store.v1.StoreService/Pull",
							"/agntcy.dir.store.v1.StoreService/Lookup",
						},
						Users: []string{"user:https://tenant.zitadel.cloud:111"},
					},
				},
			},
			principal:   "user:https://tenant.zitadel.cloud:111",
			path:        "/agntcy.dir.store.v1.StoreService/Pull",
			expectError: false,
		},
		{
			name: "viewer denied for Push",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"viewer": {
						AllowedMethods: []string{"/agntcy.dir.store.v1.StoreService/Pull"},
						Users:          []string{"user:https://tenant.zitadel.cloud:111"},
					},
				},
			},
			principal:   "user:https://tenant.zitadel.cloud:111",
			path:        "/agntcy.dir.store.v1.StoreService/Push",
			expectError: true,
		},
		{
			name: "client principal with ci-writer role",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"ci-writer": {
						AllowedMethods: []string{
							"/agntcy.dir.store.v1.StoreService/Push",
							"/agntcy.dir.store.v1.StoreService/Pull",
						},
						Clients: []string{"client:https://tenant.zitadel.cloud:69234237810729234"},
					},
				},
			},
			principal:   "client:https://tenant.zitadel.cloud:69234237810729234",
			path:        "/agntcy.dir.store.v1.StoreService/Push",
			expectError: false,
		},
		{
			name: "GitHub workflow principal with prod-deployer role",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"prod-deployer": {
						AllowedMethods:  []string{"*"},
						GitHubWorkflows: []string{"ghwf:repo:agntcy/dir:workflow:deploy.yml:ref:refs/heads/main:env:prod"},
					},
				},
			},
			principal:   "ghwf:repo:agntcy/dir:workflow:deploy.yml:ref:refs/heads/main:env:prod",
			path:        "/agntcy.dir.store.v1.StoreService/Push",
			expectError: false,
		},
		{
			name: "principal in deny list is blocked",
			config: &OIDCConfig{
				Claims:       ClaimsConfig{UserID: "sub"},
				UserDenyList: []string{"user:https://tenant.zitadel.cloud:77776025198584418"},
				Roles: map[string]OIDCRole{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"user:https://tenant.zitadel.cloud:77776025198584418"},
					},
				},
			},
			principal:   "user:https://tenant.zitadel.cloud:77776025198584418",
			path:        "/agntcy.dir.store.v1.StoreService/Push",
			expectError: true,
			errorMsg:    "deny list",
		},
		{
			name: "no role assignment - deny",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"admin": {
						AllowedMethods: []string{"*"},
						Users:          []string{"user:https://tenant.zitadel.cloud:other"},
					},
				},
			},
			principal:   "user:https://tenant.zitadel.cloud:unknown",
			path:        "/agntcy.dir.store.v1.StoreService/Push",
			expectError: true,
		},
		{
			name: "wildcard method matching",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"store-admin": {
						AllowedMethods: []string{"/agntcy.dir.store.v1.StoreService/*"},
						Users:          []string{"user:https://tenant.zitadel.cloud:admin"},
					},
				},
			},
			principal:   "user:https://tenant.zitadel.cloud:admin",
			path:        "/agntcy.dir.store.v1.StoreService/AnyMethod",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.PublicPaths = []string{"/healthz"}
			if err := tt.config.Validate(); err != nil {
				t.Fatalf("invalid config: %v", err)
			}

			resolver, err := NewOIDCRoleResolver(tt.config, slog.Default())
			if err != nil {
				t.Fatalf("failed to create resolver: %v", err)
			}

			// Deny list is checked before Authorize
			if resolver.IsDenied(tt.principal, "") {
				if !tt.expectError {
					t.Errorf("principal was denied but expected allow")
				}

				return
			}

			err = resolver.Authorize(tt.principal, tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
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

func TestOIDCRoleResolver_IsDenied(t *testing.T) {
	config := &OIDCConfig{
		Claims:       ClaimsConfig{UserID: "sub"},
		UserDenyList: []string{"user:https://iss:bad", "oidc:blocked@example.com"},
		Roles: map[string]OIDCRole{
			"admin": {
				AllowedMethods: []string{"*"},
				Users:          []string{"user:https://iss:good"},
			},
		},
		PublicPaths: []string{},
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("invalid config: %v", err)
	}

	resolver, err := NewOIDCRoleResolver(config, slog.Default())
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	if !resolver.IsDenied("user:https://iss:bad", "") {
		t.Error("user:https://iss:bad should be denied")
	}

	if !resolver.IsDenied("other", "blocked@example.com") {
		t.Error("oidc:blocked@example.com should be denied via email")
	}

	if resolver.IsDenied("user:https://iss:good", "") {
		t.Error("user:https://iss:good should not be denied")
	}
}

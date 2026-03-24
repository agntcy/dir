// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"strings"
	"testing"
)

func validConfig() *OIDCConfig {
	return &OIDCConfig{
		Claims:      ClaimsConfig{UserID: "sub"},
		PublicPaths: []string{"/healthz"},
		Roles: map[string]OIDCRole{
			"admin": {
				AllowedMethods: []string{"*"},
				Users:          []string{"user:https://iss:123"},
			},
		},
	}
}

func TestOIDCConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *OIDCConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid config",
			config:      validConfig(),
			expectError: false,
		},
		{
			name: "valid config with issuers",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Issuers: []IssuerConfig{
					{Provider: "https://iss.example.com", PrincipalType: PrincipalTypeAuto},
					{Provider: GitHubIssuer, PrincipalType: PrincipalTypeGitHub},
				},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: false,
		},
		{
			name: "missing claims.userID",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: ""},
				PublicPaths: []string{},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: true,
			errorMsg:    "userID",
		},
		{
			name: "no roles",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: "sub"},
				PublicPaths: []string{},
				Roles:       map[string]OIDCRole{},
			},
			expectError: true,
			errorMsg:    "at least one role",
		},
		{
			name: "role with empty allowedMethods",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: "sub"},
				PublicPaths: []string{},
				Roles: map[string]OIDCRole{
					"empty": {AllowedMethods: []string{}},
				},
			},
			expectError: true,
			errorMsg:    "allowedMethods",
		},
		{
			name: "invalid principalType.mode",
			config: &OIDCConfig{
				Claims:        ClaimsConfig{UserID: "sub"},
				PublicPaths:   []string{},
				PrincipalType: PrincipalTypeConfig{Mode: "invalid"},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: true,
			errorMsg:    "principalType",
		},
		{
			name: "invalid issuer principalType",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: "sub"},
				PublicPaths: []string{},
				Issuers: []IssuerConfig{
					{Provider: "https://iss", PrincipalType: "invalid"},
				},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: true,
			errorMsg:    "issuers",
		},
		{
			name: "issuer missing provider",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: "sub"},
				PublicPaths: []string{},
				Issuers: []IssuerConfig{
					{Provider: "", PrincipalType: PrincipalTypeAuto},
				},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: true,
			errorMsg:    "provider",
		},
		{
			name: "invalid principalType.machineSubPattern",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: "sub"},
				PublicPaths: []string{},
				PrincipalType: PrincipalTypeConfig{
					Mode:              PrincipalTypeAuto,
					MachineSubPattern: `[invalid`,
				},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: true,
			errorMsg:    "machineSubPattern",
		},
		{
			name: "invalid issuer machineSubPattern",
			config: &OIDCConfig{
				Claims:      ClaimsConfig{UserID: "sub"},
				PublicPaths: []string{},
				Issuers: []IssuerConfig{
					{Provider: "https://iss", MachineSubPattern: `(unclosed`},
				},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: true,
			errorMsg:    "machineSubPattern",
		},
		{
			name: "valid principalType modes",
			config: &OIDCConfig{
				Claims:        ClaimsConfig{UserID: "sub"},
				PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeAuto},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid issuer principalTypes",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Issuers: []IssuerConfig{
					{Provider: "https://a", PrincipalType: PrincipalTypeUser},
					{Provider: "https://b", PrincipalType: PrincipalTypeClient},
					{Provider: "https://c", PrincipalType: PrincipalTypeGitHub},
				},
				Roles: map[string]OIDCRole{
					"admin": {AllowedMethods: []string{"*"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid githubWorkflow wildcard in refs heads branch suffix",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"ci-oidc-test": {
						AllowedMethods: []string{"/agntcy.dir.search.v1.SearchService/SearchCIDs"},
						GitHubWorkflows: []string{
							"ghwf:repo:agntcy/dir:workflow:oidc-test.yml:ref:refs/heads/feat/*",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid githubWorkflow wildcard with multiple stars",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"ci-oidc-test": {
						AllowedMethods: []string{"/agntcy.dir.search.v1.SearchService/SearchCIDs"},
						GitHubWorkflows: []string{
							"ghwf:repo:agntcy/dir:workflow:oidc-test.yml:ref:refs/heads/*/*",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid githubWorkflows wildcard",
		},
		{
			name: "invalid githubWorkflow wildcard not at end",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"ci-oidc-test": {
						AllowedMethods: []string{"/agntcy.dir.search.v1.SearchService/SearchCIDs"},
						GitHubWorkflows: []string{
							"ghwf:repo:agntcy/dir:workflow:oidc-test.yml:ref:refs/heads/*:env:dev",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid githubWorkflows wildcard",
		},
		{
			name: "invalid githubWorkflow wildcard outside refs heads",
			config: &OIDCConfig{
				Claims: ClaimsConfig{UserID: "sub"},
				Roles: map[string]OIDCRole{
					"ci-oidc-test": {
						AllowedMethods: []string{"/agntcy.dir.search.v1.SearchService/SearchCIDs"},
						GitHubWorkflows: []string{
							"ghwf:repo:agntcy/dir:workflow:oidc-test.yml:ref:refs/tags/*",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid githubWorkflows wildcard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
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

func TestOIDCConfig_Validate_NormalizesPublicPaths(t *testing.T) {
	config := &OIDCConfig{
		Claims:      ClaimsConfig{UserID: "sub"},
		PublicPaths: nil, // explicitly nil
		Roles: map[string]OIDCRole{
			"admin": {AllowedMethods: []string{"*"}},
		},
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if config.PublicPaths == nil {
		t.Error("expected PublicPaths to be normalized to empty slice, got nil")
	}

	if len(config.PublicPaths) != 0 {
		t.Errorf("expected empty PublicPaths, got %v", config.PublicPaths)
	}
}

func TestOIDCConfig_GetIssuerConfig(t *testing.T) {
	config := &OIDCConfig{
		Issuers: []IssuerConfig{
			{Provider: "https://iss-a.example.com", PrincipalType: PrincipalTypeUser},
			{Provider: "https://iss-b.example.com", PrincipalType: PrincipalTypeClient},
			{Provider: GitHubIssuer, PrincipalType: PrincipalTypeGitHub},
		},
	}

	tests := []struct {
		name              string
		issuerURL         string
		wantFound         bool
		wantPrincipalType string
	}{
		{"found user", "https://iss-a.example.com", true, PrincipalTypeUser},
		{"found client", "https://iss-b.example.com", true, PrincipalTypeClient},
		{"found github", GitHubIssuer, true, PrincipalTypeGitHub},
		{"unknown issuer", "https://unknown.example.com", false, ""},
		{"empty issuer", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ic := config.GetIssuerConfig(tt.issuerURL)
			if tt.wantFound { //nolint:nestif // test assertions; refactoring would reduce clarity
				if ic == nil {
					t.Fatal("expected issuer config, got nil")
				}

				if ic.Provider != tt.issuerURL {
					t.Errorf("Provider = %q, want %q", ic.Provider, tt.issuerURL)
				}

				if ic.PrincipalType != tt.wantPrincipalType {
					t.Errorf("PrincipalType = %q, want %q", ic.PrincipalType, tt.wantPrincipalType)
				}
			} else if ic != nil {
				t.Errorf("expected nil for unknown issuer, got %+v", ic)
			}
		})
	}
}

func TestOIDCConfig_GetIssuerConfig_EmptyIssuers(t *testing.T) {
	config := &OIDCConfig{Issuers: nil}
	if ic := config.GetIssuerConfig("https://any"); ic != nil {
		t.Errorf("expected nil for empty issuers, got %+v", ic)
	}

	config.Issuers = []IssuerConfig{}
	if ic := config.GetIssuerConfig("https://any"); ic != nil {
		t.Errorf("expected nil for empty issuers slice, got %+v", ic)
	}
}

func TestOIDCConfig_IsPublicPath(t *testing.T) {
	config := &OIDCConfig{
		PublicPaths: []string{"/healthz", "/grpc.reflection"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/healthz", true},
		{"/grpc.reflection", true},
		{"/agntcy.dir.store.v1.StoreService/Push", false},
		{"/", false},
		{"/healthz/", false}, // exact match, no trailing slash
		{"", false},
	}

	for _, tt := range tests {
		name := tt.path
		if name == "" {
			name = "empty"
		}

		t.Run(name, func(t *testing.T) {
			got := config.IsPublicPath(tt.path)
			if got != tt.want {
				t.Errorf("IsPublicPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestOIDCConfig_IsPublicPath_EmptyList(t *testing.T) {
	config := &OIDCConfig{PublicPaths: []string{}}
	if config.IsPublicPath("/any") {
		t.Error("expected no public paths when list is empty")
	}
}

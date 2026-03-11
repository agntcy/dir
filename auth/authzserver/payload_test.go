// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"testing"
)

func TestExtractPrincipal_User(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: map[string]IssuerConfig{
			"https://tenant.zitadel.cloud": {PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","sub":"77776025198584418"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://tenant.zitadel.cloud:77776025198584418" {
		t.Errorf("principal = %q, want user:https://tenant.zitadel.cloud:77776025198584418", principal)
	}

	if pt != PrincipalTypeUser {
		t.Errorf("principalType = %q, want user", pt)
	}
}

func TestExtractPrincipal_Client(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: map[string]IssuerConfig{
			"https://tenant.zitadel.cloud": {
				PrincipalType:        PrincipalTypeClient,
				MachineIdentityClaim: "client_id",
			},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeClient, MachineIdentityClaim: "client_id"},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","client_id":"69234237810729234"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://tenant.zitadel.cloud:69234237810729234" {
		t.Errorf("principal = %q, want client:https://tenant.zitadel.cloud:69234237810729234", principal)
	}

	if pt != PrincipalTypeClient {
		t.Errorf("principalType = %q, want client", pt)
	}
}

func TestExtractPrincipal_ClientAzpFallback(t *testing.T) {
	config := &OIDCConfig{
		Claims:        ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeClient, MachineIdentityClaim: "client_id"},
	}

	// No client_id, has azp
	payload := `{"iss":"https://tenant.zitadel.cloud","azp":"fallback-client-id"}`

	principal, _, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://tenant.zitadel.cloud:fallback-client-id" {
		t.Errorf("principal = %q, want client:...:fallback-client-id", principal)
	}
}

func TestExtractPrincipal_GitHub(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: map[string]IssuerConfig{
			GitHubIssuer: {PrincipalType: PrincipalTypeGitHub},
		},
	}

	payload := `{
		"iss":"https://token.actions.githubusercontent.com",
		"repository":"agntcy/dir",
		"ref":"refs/heads/main",
		"environment":"prod",
		"workflow_ref":"agntcy/dir/.github/workflows/deploy.yml@refs/heads/main"
	}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	want := "ghwf:repo:agntcy/dir:workflow:deploy.yml:ref:refs/heads/main:env:prod"
	if principal != want {
		t.Errorf("principal = %q, want %q", principal, want)
	}

	if pt != "ghwf" {
		t.Errorf("principalType = %q, want ghwf", pt)
	}
}

func TestExtractPrincipal_GitHub_JobWorkflowRefFallback(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: map[string]IssuerConfig{
			GitHubIssuer: {PrincipalType: PrincipalTypeGitHub},
		},
	}

	payload := `{
		"iss":"https://token.actions.githubusercontent.com",
		"repository":"agntcy/dir",
		"ref":"refs/heads/main",
		"job_workflow_ref":"agntcy/dir/.github/workflows/build.yml@refs/heads/main"
	}`

	principal, _, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	want := "ghwf:repo:agntcy/dir:workflow:build.yml:ref:refs/heads/main"
	if principal != want {
		t.Errorf("principal = %q, want %q", principal, want)
	}
}

func TestExtractPrincipal_Auto_User(t *testing.T) {
	config := &OIDCConfig{
		Claims:        ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeAuto},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","sub":"user-123"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://tenant.zitadel.cloud:user-123" {
		t.Errorf("principal = %q", principal)
	}

	if pt != PrincipalTypeUser {
		t.Errorf("principalType = %q", pt)
	}
}

func TestExtractPrincipal_Auto_Client(t *testing.T) {
	config := &OIDCConfig{
		Claims:        ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeAuto, MachineIdentityClaim: "client_id"},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","sub":"machine-123","client_id":"machine-123"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://tenant.zitadel.cloud:machine-123" {
		t.Errorf("principal = %q", principal)
	}

	if pt != PrincipalTypeClient {
		t.Errorf("principalType = %q", pt)
	}
}

func TestExtractPrincipal_Auto_NoSubWithClientID(t *testing.T) {
	config := &OIDCConfig{
		Claims:        ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeAuto, MachineIdentityClaim: "client_id"},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","client_id":"machine-only"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://tenant.zitadel.cloud:machine-only" {
		t.Errorf("principal = %q", principal)
	}

	if pt != PrincipalTypeClient {
		t.Errorf("principalType = %q", pt)
	}
}

func TestExtractPrincipal_Auto_NoSubNoClientID(t *testing.T) {
	config := &OIDCConfig{
		Claims:        ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeAuto, MachineIdentityClaim: "client_id"},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud"}`

	_, _, err := ExtractPrincipal(payload, config)
	if err == nil {
		t.Error("expected error when both sub and client_id are missing")
	}
}

func TestExtractPrincipal_Auto_MachineSubPattern(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{
			Mode:                 PrincipalTypeAuto,
			MachineIdentityClaim: "client_id",
			MachineSubPattern:    `^machine@`,
		},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","sub":"machine@service","client_id":"svc-123"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://tenant.zitadel.cloud:svc-123" {
		t.Errorf("principal = %q", principal)
	}

	if pt != PrincipalTypeClient {
		t.Errorf("principalType = %q", pt)
	}
}

func TestExtractPrincipal_Auto_MachineSubPatternNoClientID(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{
			Mode:                 PrincipalTypeAuto,
			MachineIdentityClaim: "client_id",
			MachineSubPattern:    `^machine@`,
		},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","sub":"machine@service"}`

	_, _, err := ExtractPrincipal(payload, config)
	if err == nil {
		t.Error("expected error when sub matches machine pattern but client_id is missing")
	}
}

func TestExtractPrincipal_Auto_MachineSubPatternNoMatch(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		PrincipalType: PrincipalTypeConfig{
			Mode:                 PrincipalTypeAuto,
			MachineIdentityClaim: "client_id",
			MachineSubPattern:    `^machine@`,
		},
	}

	payload := `{"iss":"https://tenant.zitadel.cloud","sub":"human-user"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://tenant.zitadel.cloud:human-user" {
		t.Errorf("principal = %q", principal)
	}

	if pt != PrincipalTypeUser {
		t.Errorf("principalType = %q", pt)
	}
}

func TestExtractPrincipal_Errors(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		config  *OIDCConfig
	}{
		{"nil config", `{}`, nil},
		{"empty payload", "", &OIDCConfig{Claims: ClaimsConfig{UserID: "sub"}}},
		{"invalid JSON", `{invalid}`, &OIDCConfig{Claims: ClaimsConfig{UserID: "sub"}}},
		{"missing iss", `{"sub":"123"}`, &OIDCConfig{Claims: ClaimsConfig{UserID: "sub"}}},
		{"user missing sub", `{"iss":"https://iss"}`, &OIDCConfig{
			Claims:        ClaimsConfig{UserID: "sub"},
			PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
		}},
		{"client missing client_id", `{"iss":"https://iss"}`, &OIDCConfig{
			Claims:        ClaimsConfig{UserID: "sub"},
			PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeClient},
		}},
		{"GitHub missing repository", `{"iss":"` + GitHubIssuer + `","workflow_ref":"a/b/.github/workflows/x.yml@main"}`, &OIDCConfig{
			Issuers: map[string]IssuerConfig{GitHubIssuer: {PrincipalType: PrincipalTypeGitHub}},
		}},
		{"GitHub missing workflow_ref", `{"iss":"` + GitHubIssuer + `","repository":"a/b"}`, &OIDCConfig{
			Issuers: map[string]IssuerConfig{GitHubIssuer: {PrincipalType: PrincipalTypeGitHub}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ExtractPrincipal(tt.payload, tt.config)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestGetEmail(t *testing.T) {
	tests := []struct {
		name      string
		payload   string
		emailPath string
		want      string
	}{
		{
			name:      "top-level email",
			payload:   `{"email":"user@example.com"}`,
			emailPath: "email",
			want:      "user@example.com",
		},
		{
			name:      "nested claims.email",
			payload:   `{"claims":{"email":"nested@example.com"}}`,
			emailPath: "claims.email",
			want:      "nested@example.com",
		},
		{
			name:      "deep nested a.b.c",
			payload:   `{"a":{"b":{"c":"deep@example.com"}}}`,
			emailPath: "a.b.c",
			want:      "deep@example.com",
		},
		{
			name:      "empty path",
			payload:   `{"email":"x@y.com"}`,
			emailPath: "",
			want:      "",
		},
		{
			name:      "empty payload",
			payload:   "",
			emailPath: "email",
			want:      "",
		},
		{
			name:      "invalid JSON",
			payload:   `{invalid}`,
			emailPath: "email",
			want:      "",
		},
		{
			name:      "path not string",
			payload:   `{"email":123}`,
			emailPath: "email",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEmail(tt.payload, tt.emailPath)
			if got != tt.want {
				t.Errorf("GetEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

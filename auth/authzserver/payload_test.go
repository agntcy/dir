// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"encoding/base64"
	"testing"
)

func TestExtractPrincipal_EnvoyForwardPayloadHeader_Base64URL(t *testing.T) {
	// Mirrors Envoy jwt_authn forward_payload_header: base64url-encoded payload segment.
	// Uses UserID: "sub" to test backward compatibility with sub-based extraction.
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: []IssuerConfig{
			{Provider: "https://dex.example.com", PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	jsonPayload := `{"iss":"https://dex.example.com","sub":"77776025198584418"}`
	forwarded := base64.RawURLEncoding.EncodeToString([]byte(jsonPayload))

	principal, pt, err := ExtractPrincipal(forwarded, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:77776025198584418" {
		t.Errorf("principal = %q, want user:https://dex.example.com:77776025198584418", principal)
	}

	if pt != PrincipalTypeUser {
		t.Errorf("principalType = %q, want user", pt)
	}
}

func TestExtractPrincipal_FullJWTString(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: []IssuerConfig{
			{Provider: "https://dex.example.com", PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	jsonPayload := `{"iss":"https://dex.example.com","sub":"sub-xyz"}`
	seg := base64.RawURLEncoding.EncodeToString([]byte(jsonPayload))
	full := "xx." + seg + ".yy"

	principal, _, err := ExtractPrincipal(full, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:sub-xyz" {
		t.Errorf("principal = %q", principal)
	}
}

func TestExtractPrincipal_User_SubClaim(t *testing.T) {
	// Tests backward compatibility: UserID: "sub" extracts principal from sub claim.
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: []IssuerConfig{
			{Provider: "https://dex.example.com", PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	payload := `{"iss":"https://dex.example.com","sub":"77776025198584418"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:77776025198584418" {
		t.Errorf("principal = %q, want user:https://dex.example.com:77776025198584418", principal)
	}

	if pt != PrincipalTypeUser {
		t.Errorf("principalType = %q, want user", pt)
	}
}

func TestExtractPrincipal_User_EmailClaim(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "email"},
		Issuers: []IssuerConfig{
			{Provider: "https://dex.example.com", PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	payload := `{"iss":"https://dex.example.com","sub":"CgcyMzQyNzQ5EgZnaXRodWI","email":"user@example.com"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:user@example.com" { //nolint:gosec // not credentials
		t.Errorf("principal = %q, want user:https://dex.example.com:user@example.com", principal)
	}

	if pt != PrincipalTypeUser {
		t.Errorf("principalType = %q, want user", pt)
	}
}

func TestExtractPrincipal_User_EmailClaim_Missing(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "email"},
		Issuers: []IssuerConfig{
			{Provider: "https://dex.example.com", PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	payload := `{"iss":"https://dex.example.com","sub":"CgcyMzQyNzQ5EgZnaXRodWI"}`

	_, _, err := ExtractPrincipal(payload, config)
	if err == nil {
		t.Error("expected error when email claim is missing but userID is configured as email")
	}
}

func TestExtractPrincipal_User_EmptyUserIDDefaultsToSub(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: ""},
		Issuers: []IssuerConfig{
			{Provider: "https://dex.example.com", PrincipalType: PrincipalTypeUser},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeUser},
	}

	payload := `{"iss":"https://dex.example.com","sub":"fallback-sub-123"}`

	principal, _, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:fallback-sub-123" {
		t.Errorf("principal = %q, want user:https://dex.example.com:fallback-sub-123", principal)
	}
}

func TestExtractPrincipal_Client(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: []IssuerConfig{
			{
				Provider:             "https://dex.example.com",
				PrincipalType:        PrincipalTypeClient,
				MachineIdentityClaim: "client_id",
			},
		},
		PrincipalType: PrincipalTypeConfig{Mode: PrincipalTypeClient, MachineIdentityClaim: "client_id"},
	}

	payload := `{"iss":"https://dex.example.com","client_id":"69234237810729234"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://dex.example.com:69234237810729234" {
		t.Errorf("principal = %q, want client:https://dex.example.com:69234237810729234", principal)
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
	payload := `{"iss":"https://dex.example.com","azp":"fallback-client-id"}`

	principal, _, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://dex.example.com:fallback-client-id" {
		t.Errorf("principal = %q, want client:...:fallback-client-id", principal)
	}
}

func TestExtractPrincipal_GitHub(t *testing.T) {
	config := &OIDCConfig{
		Claims: ClaimsConfig{UserID: "sub"},
		Issuers: []IssuerConfig{
			{Provider: GitHubIssuer, PrincipalType: PrincipalTypeGitHub},
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
		Issuers: []IssuerConfig{
			{Provider: GitHubIssuer, PrincipalType: PrincipalTypeGitHub},
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

	payload := `{"iss":"https://dex.example.com","sub":"user-123"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:user-123" {
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

	payload := `{"iss":"https://dex.example.com","sub":"machine-123","client_id":"machine-123"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://dex.example.com:machine-123" {
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

	payload := `{"iss":"https://dex.example.com","client_id":"machine-only"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://dex.example.com:machine-only" {
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

	payload := `{"iss":"https://dex.example.com"}`

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

	payload := `{"iss":"https://dex.example.com","sub":"machine@service","client_id":"svc-123"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "client:https://dex.example.com:svc-123" {
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

	payload := `{"iss":"https://dex.example.com","sub":"machine@service"}`

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

	payload := `{"iss":"https://dex.example.com","sub":"human-user"}`

	principal, pt, err := ExtractPrincipal(payload, config)
	if err != nil {
		t.Fatalf("ExtractPrincipal: %v", err)
	}

	if principal != "user:https://dex.example.com:human-user" {
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
			Issuers: []IssuerConfig{{Provider: GitHubIssuer, PrincipalType: PrincipalTypeGitHub}},
		}},
		{"GitHub missing workflow_ref", `{"iss":"` + GitHubIssuer + `","repository":"a/b"}`, &OIDCConfig{
			Issuers: []IssuerConfig{{Provider: GitHubIssuer, PrincipalType: PrincipalTypeGitHub}},
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

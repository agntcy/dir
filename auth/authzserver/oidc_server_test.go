// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"context"
	"log/slog"
	"testing"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc/codes"
)

const testPayloadAdmin = `{"iss":"https://dex.example.com","sub":"CgcyMzQyNzQ5EgZnaXRodWI","email":"admin@example.com"}`

func makeCheckRequest(path string, headers map[string]string) *authv3.CheckRequest {
	if headers == nil {
		headers = make(map[string]string)
	}

	return &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Path:    path,
					Method:  "POST",
					Headers: headers,
				},
			},
		},
	}
}

func validOIDCConfig() *OIDCConfig {
	return &OIDCConfig{
		Claims:      ClaimsConfig{UserID: "email", EmailPath: "email"},
		PublicPaths: []string{"/healthz"},
		Roles: map[string]OIDCRole{
			"admin": {
				AllowedMethods: []string{"*"},
				Users:          []string{"user:https://dex.example.com:admin@example.com"},
			},
		},
	}
}

func TestNewOIDCAuthorizationServer(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		_, err := NewOIDCAuthorizationServer(nil, slog.Default())
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := &OIDCConfig{Claims: ClaimsConfig{UserID: ""}, Roles: map[string]OIDCRole{"x": {AllowedMethods: []string{"*"}}}}

		_, err := NewOIDCAuthorizationServer(cfg, slog.Default())
		if err == nil {
			t.Error("expected error for invalid config")
		}
	})

	t.Run("success", func(t *testing.T) {
		srv, err := NewOIDCAuthorizationServer(validOIDCConfig(), slog.Default())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if srv == nil {
			t.Error("expected non-nil server")
		}
	})

	t.Run("nil logger uses default", func(t *testing.T) {
		srv, err := NewOIDCAuthorizationServer(validOIDCConfig(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if srv == nil {
			t.Error("expected non-nil server")
		}
	})
}

//nolint:gocognit,cyclop // Test function with multiple subtests; high complexity is acceptable.
func TestOIDCAuthorizationServer_Check(t *testing.T) {
	config := validOIDCConfig()

	srv, err := NewOIDCAuthorizationServer(config, slog.Default())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()

	t.Run("public path allows without auth", func(t *testing.T) {
		req := makeCheckRequest("/healthz", nil)

		resp, err := srv.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}

		if resp.GetStatus().GetCode() != int32(codes.OK) {
			t.Errorf("expected OK, got code %d", resp.GetStatus().GetCode())
		}
	})

	t.Run("missing x-jwt-payload returns 401", func(t *testing.T) {
		req := makeCheckRequest("/api/test", nil)

		resp, err := srv.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}

		if resp.GetStatus().GetCode() != int32(codes.Unauthenticated) {
			t.Errorf("expected Unauthenticated, got code %d", resp.GetStatus().GetCode())
		}
	})

	t.Run("invalid payload returns 401", func(t *testing.T) {
		req := makeCheckRequest("/api/test", map[string]string{HeaderJWTPayload: "invalid-json"})

		resp, err := srv.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}

		if resp.GetStatus().GetCode() != int32(codes.Unauthenticated) {
			t.Errorf("expected Unauthenticated, got code %d", resp.GetStatus().GetCode())
		}
	})

	t.Run("principal in deny list returns 403", func(t *testing.T) {
		cfg := validOIDCConfig()
		cfg.UserDenyList = []string{"user:https://dex.example.com:admin@example.com"}

		srv2, err := NewOIDCAuthorizationServer(cfg, slog.Default())
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}

		req := makeCheckRequest("/api/test", map[string]string{HeaderJWTPayload: testPayloadAdmin})

		resp, err := srv2.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}

		if resp.GetStatus().GetCode() != int32(codes.PermissionDenied) {
			t.Errorf("expected PermissionDenied, got code %d", resp.GetStatus().GetCode())
		}
	})

	t.Run("authorized request returns 200 with headers", func(t *testing.T) {
		req := makeCheckRequest("/api/test", map[string]string{HeaderJWTPayload: testPayloadAdmin})

		resp, err := srv.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}

		if resp.GetStatus().GetCode() != int32(codes.OK) {
			t.Errorf("expected OK, got code %d", resp.GetStatus().GetCode())
		}

		okResp := resp.GetOkResponse()
		if okResp == nil {
			t.Fatal("expected OkResponse")
		}

		headers := okResp.GetHeaders()
		if len(headers) < 2 {
			t.Errorf("expected at least 2 headers, got %d", len(headers))
		}
	})

	t.Run("unauthorized path returns 403", func(t *testing.T) {
		cfg := validOIDCConfig()
		cfg.Roles = map[string]OIDCRole{
			"viewer": {
				AllowedMethods: []string{"/other/path"},
				Users:          []string{"user:https://dex.example.com:admin@example.com"},
			},
		}

		srv2, err := NewOIDCAuthorizationServer(cfg, slog.Default())
		if err != nil {
			t.Fatalf("failed to create server: %v", err)
		}

		req := makeCheckRequest("/api/forbidden", map[string]string{HeaderJWTPayload: testPayloadAdmin})

		resp, err := srv2.Check(ctx, req)
		if err != nil {
			t.Fatalf("Check: %v", err)
		}

		if resp.GetStatus().GetCode() != int32(codes.PermissionDenied) {
			t.Errorf("expected PermissionDenied, got code %d", resp.GetStatus().GetCode())
		}
	})
}

func TestGetHeader(t *testing.T) {
	// getHeader is unexported but exercised via Check with x-jwt-payload.
	// Test case-insensitive lookup via integration: Envoy may send lowercase.
	config := validOIDCConfig()
	srv, _ := NewOIDCAuthorizationServer(config, slog.Default())
	ctx := context.Background()

	req := makeCheckRequest("/api/test", map[string]string{"X-JWT-Payload": testPayloadAdmin})

	resp, err := srv.Check(ctx, req)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if resp.GetStatus().GetCode() != int32(codes.OK) {
		t.Errorf("expected OK with X-JWT-Payload (case-insensitive), got code %d", resp.GetStatus().GetCode())
	}
}

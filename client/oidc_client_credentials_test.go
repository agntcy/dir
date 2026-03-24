// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMaxFormBodyBytes = 1 << 20 // 1 MiB

func TestRunClientCredentialsFlow_WithDiscovery(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token_endpoint": srv.URL + "/oauth/v2/token",
		})
	})

	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, testMaxFormBodyBytes)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		assertFormEquals(t, r.Form, "grant_type", "client_credentials")
		assertFormEquals(t, r.Form, "client_id", "machine-client")
		assertFormEquals(t, r.Form, "client_secret", "machine-secret")
		assertFormEquals(t, r.Form, "scope", "openid profile")

		accessToken := makeJWT("machine-sub", srv.URL)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   3600,
			"scope":        "openid profile",
		})
	})

	res, err := OIDC.RunClientCredentialsFlow(context.Background(), &ClientCredentialsConfig{
		Issuer:       srv.URL,
		ClientID:     "machine-client",
		ClientSecret: "machine-secret",
		Scopes:       []string{"openid", "profile"},
	})
	if err != nil {
		t.Fatalf("RunClientCredentialsFlow failed: %v", err)
	}

	if res.AccessToken == "" {
		t.Fatal("expected access token")
	}

	if res.Subject != "machine-sub" {
		t.Fatalf("expected subject machine-sub, got %q", res.Subject)
	}

	if res.Issuer != srv.URL {
		t.Fatalf("expected issuer %q, got %q", srv.URL, res.Issuer)
	}
}

func TestRunClientCredentialsFlow_WithSecretFile(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, testMaxFormBodyBytes)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		assertFormEquals(t, r.Form, "client_secret", "from-file-secret")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": makeJWT("sub-from-file", "https://issuer.example"),
			"token_type":   "Bearer",
			"expires_in":   60,
		})
	})

	tmpDir := t.TempDir()

	secretPath := filepath.Join(tmpDir, "machine-secret.txt")
	if err := os.WriteFile(secretPath, []byte("from-file-secret\n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	res, err := OIDC.RunClientCredentialsFlow(context.Background(), &ClientCredentialsConfig{
		Issuer:           "https://issuer.example",
		TokenEndpoint:    srv.URL + "/oauth/v2/token",
		ClientID:         "machine-client",
		ClientSecretFile: secretPath,
	})
	if err != nil {
		t.Fatalf("RunClientCredentialsFlow failed: %v", err)
	}

	if res.AccessToken == "" {
		t.Fatal("expected access token")
	}

	if res.Subject != "sub-from-file" {
		t.Fatalf("expected subject sub-from-file, got %q", res.Subject)
	}
}

func makeJWT(sub, iss string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + sub + `","iss":"` + iss + `"}`))

	return strings.Join([]string{header, payload, "sig"}, ".")
}

func assertFormEquals(t *testing.T, values url.Values, key, expected string) {
	t.Helper()

	if got := values.Get(key); got != expected {
		t.Fatalf("expected %s=%q, got %q", key, expected, got)
	}
}

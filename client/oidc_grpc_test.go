// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_serverNameFromAddr(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dev.gateway.ads.outshift.io", serverNameFromAddr("dev.gateway.ads.outshift.io:443"))
	assert.Equal(t, "localhost", serverNameFromAddr("localhost:9999"))
	assert.Equal(t, "badaddr", serverNameFromAddr("badaddr"))
}

func TestWithAuth_OIDC_WithOIDCToken(t *testing.T) {
	t.Parallel()

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			OIDCToken:     "test-access-token",
		},
	}

	ctx := context.Background()
	opt := withAuth(ctx)
	err := opt(opts)
	require.NoError(t, err)
	assert.NotEmpty(t, opts.authOpts)
	assert.Nil(t, opts.authClient)
}

func TestOIDCBearerCredentials_GetRequestMetadata(t *testing.T) {
	t.Parallel()

	c := newOIDCBearerCredentials("mytoken")
	md, err := c.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer mytoken", md["authorization"])
	assert.True(t, c.RequireTransportSecurity())
}

func TestSetupOIDCAuth_WithMachineCredentials_MintsAndCachesToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	mux := http.NewServeMux()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"token_endpoint": srv.URL + "/oauth/v2/token",
		})
	})

	mux.HandleFunc("/oauth/v2/token", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, testMaxFormBodyBytes)
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			http.Error(w, "bad form", http.StatusBadRequest)

			return
		}

		assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		assert.Equal(t, "machine-client", r.Form.Get("client_id"))
		assert.Equal(t, "machine-secret", r.Form.Get("client_secret"))
		assert.Equal(t, "openid profile", r.Form.Get("scope"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": makeTestJWT("machine-sub", srv.URL),
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	opts := &options{
		config: &Config{
			ServerAddress:            "gateway.example.com:443",
			AuthMode:                 "oidc",
			OIDCIssuer:               srv.URL,
			OIDCMachineClientID:      "machine-client",
			OIDCMachineClientSecret:  "machine-secret",
			OIDCMachineScopes:        []string{"openid", "profile"},
			OIDCMachineTokenEndpoint: "",
		},
	}

	err := opts.setupOIDCAuth(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, opts.authOpts)

	cache := NewTokenCache()
	tok, err := cache.GetValidToken()
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Equal(t, "oidc", tok.Provider)
	assert.Equal(t, "machine-client", tok.User)
	assert.Equal(t, "machine-sub", tok.UserID)
}

func TestSetupOIDCAuth_NoTokenAndNoMachineConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
		},
	}

	err := opts.setupOIDCAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dirctl auth machine")
}

func makeTestJWT(sub, iss string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + sub + `","iss":"` + iss + `"}`))

	return strings.Join([]string{header, payload, "sig"}, ".")
}

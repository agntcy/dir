// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_serverNameFromAddr(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dev.ads.outshift.io", serverNameFromAddr("dev.ads.outshift.io:443"))
	assert.Equal(t, "localhost", serverNameFromAddr("localhost:9999"))
	assert.Equal(t, "badaddr", serverNameFromAddr("badaddr"))
}

func TestWithAuth_OIDC_WithAuthToken(t *testing.T) {
	t.Parallel()

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			AuthToken:     "test-access-token",
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

	c := newOIDCBearerCredentials(staticTokenSource("mytoken"))
	md, err := c.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer mytoken", md["authorization"])
	assert.True(t, c.RequireTransportSecurity())
}

func TestOIDCBearerCredentials_ResolvesTokenPerRequest(t *testing.T) {
	t.Parallel()

	token := "first-token"
	c := newOIDCBearerCredentials(func(context.Context) (string, error) { return token, nil })

	md, err := c.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer first-token", md["authorization"])

	// A renewed token must reach the next RPC; the old design froze the first.
	token = "renewed-token"

	md, err = c.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Bearer renewed-token", md["authorization"])
}

func TestOIDCBearerCredentials_PropagatesTokenSourceError(t *testing.T) {
	t.Parallel()

	c := newOIDCBearerCredentials(func(context.Context) (string, error) {
		return "", errors.New("mint failed")
	})

	_, err := c.GetRequestMetadata(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mint failed")
}

func TestSetupOIDCAuth_NoTokenReturnsError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// The error names the interactive login only outside a workflow that can
	// mint ID tokens, and the job running these tests has that environment.
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
		},
	}

	err := opts.setupOIDCAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dirctl auth login")
}

func TestSetupOIDCAuth_ExpiredCachedTokenReturnsAuthError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cache, err := NewTokenCacheForIssuer("https://issuer.example.com")
	require.NoError(t, err)

	err = cache.Save(&CachedToken{
		AccessToken: "expired-token",
		Issuer:      "https://issuer.example.com",
		Provider:    "oidc",
		ExpiresAt:   time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
		},
	}

	err = opts.setupOIDCAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cached OIDC token has expired")
	assert.Contains(t, err.Error(), "dirctl auth login")
}

func TestDefaultOIDCScopes(t *testing.T) {
	scopes := resolveScopes(nil)
	assert.Contains(t, scopes, "offline_access")
	assert.Contains(t, scopes, "groups")
}

func TestSetupOIDCAuth_ExpiredCachedTokenRefreshSuccess(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/token", r.URL.Path)
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "cached-refresh-token", r.FormValue("refresh_token"))
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access-token","refresh_token":"rotated-refresh-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer srv.Close()

	cache, err := NewTokenCacheForIssuer(srv.URL)
	require.NoError(t, err)
	//nolint:gosec // G101: test fixture token values, not real credentials
	err = cache.Save(&CachedToken{
		AccessToken:  "expired-token",
		RefreshToken: "cached-refresh-token",
		Issuer:       srv.URL,
		Provider:     "oidc",
		ExpiresAt:    time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			OIDCClientID:  "test-client-id",
		},
	}

	err = opts.setupOIDCAuth(context.Background())
	require.NoError(t, err)
	assert.Len(t, opts.authOpts, 2)

	updatedToken, loadErr := cache.Load()
	require.NoError(t, loadErr)
	require.NotNil(t, updatedToken)
	assert.Equal(t, "new-access-token", updatedToken.AccessToken)
	assert.Equal(t, "rotated-refresh-token", updatedToken.RefreshToken)
}

func TestSetupOIDCAuth_ExpiredCachedTokenRefreshRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh token expired"}`))
	}))
	defer srv.Close()

	cache, err := NewTokenCacheForIssuer(srv.URL)
	require.NoError(t, err)
	//nolint:gosec // G101: test fixture token values, not real credentials
	err = cache.Save(&CachedToken{
		AccessToken:  "expired-token",
		RefreshToken: "cached-refresh-token",
		Issuer:       srv.URL,
		Provider:     "oidc",
		ExpiresAt:    time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			OIDCClientID:  "test-client-id",
		},
	}

	err = opts.setupOIDCAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refresh failed")
	assert.Contains(t, err.Error(), "invalid_grant")
	assert.Contains(t, err.Error(), "dirctl auth login")
}

func TestSetupOIDCAuth_ExpiredCachedTokenMissingRefreshToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cache, err := NewTokenCacheForIssuer("https://issuer.example.com")
	require.NoError(t, err)

	err = cache.Save(&CachedToken{
		AccessToken: "expired-token",
		Issuer:      "https://issuer.example.com",
		Provider:    "oidc",
		ExpiresAt:   time.Now().Add(-time.Hour),
		CreatedAt:   time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
		},
	}

	err = opts.setupOIDCAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no refresh token is available")
	assert.Contains(t, err.Error(), "dirctl auth login")
}

func TestSetupOIDCAuth_RefreshPreservesCachedRefreshTokenWhenMissingInResponse(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer srv.Close()

	cache, err := NewTokenCacheForIssuer(srv.URL)
	require.NoError(t, err)
	//nolint:gosec // G101: test fixture token values, not real credentials
	err = cache.Save(&CachedToken{
		AccessToken:  "expired-token",
		RefreshToken: "cached-refresh-token",
		Issuer:       srv.URL,
		Provider:     "oidc",
		ExpiresAt:    time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			OIDCClientID:  "test-client-id",
		},
	}

	err = opts.setupOIDCAuth(context.Background())
	require.NoError(t, err)

	updatedToken, loadErr := cache.Load()
	require.NoError(t, loadErr)
	require.NotNil(t, updatedToken)
	assert.Equal(t, "new-access-token", updatedToken.AccessToken)
	assert.Equal(t, "cached-refresh-token", updatedToken.RefreshToken)
}

func TestCachedOIDCTokenSource_RefreshesMidRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var refreshes atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := refreshes.Add(1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":"refreshed-token-%d","token_type":"Bearer","expires_in":3600}`, count)
	}))
	defer srv.Close()

	cache, err := NewTokenCacheForIssuer(srv.URL)
	require.NoError(t, err)
	//nolint:gosec // G101: test fixture token values, not real credentials
	require.NoError(t, cache.Save(&CachedToken{
		AccessToken:  "dial-time-token",
		RefreshToken: "cached-refresh-token",
		Issuer:       srv.URL,
		Provider:     "oidc",
		ExpiresAt:    time.Now().Add(time.Hour),
		CreatedAt:    time.Now(),
	}))

	source := &cachedOIDCTokenSource{cache: cache, clientID: "test-client-id"}

	token, err := source.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "dial-time-token", token)

	// A token with time left is served from memory, without touching disk or IdP.
	token, err = source.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "dial-time-token", token)
	assert.Equal(t, int64(0), refreshes.Load())

	// The cached token has now aged out mid-run, which is exactly the case that
	// used to fail: the frozen dial-time token was sent until the command died.
	//nolint:gosec // G101: test fixture token values, not real credentials
	require.NoError(t, cache.Save(&CachedToken{
		AccessToken:  "dial-time-token",
		RefreshToken: "cached-refresh-token",
		Issuer:       srv.URL,
		Provider:     "oidc",
		ExpiresAt:    time.Now().Add(-time.Minute),
		CreatedAt:    time.Now().Add(-time.Hour),
	}))

	source.expiresAt = time.Now().Add(time.Minute)
	source.lastRead = time.Now().Add(-minCachedTokenRecheck - time.Second)

	token, err = source.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token-1", token)
	assert.Equal(t, int64(1), refreshes.Load())
}

func TestCachedOIDCTokenSource_BoundsRecheckFrequency(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cache, err := NewTokenCacheForIssuer("https://issuer.example.com")
	require.NoError(t, err)
	require.NoError(t, cache.Save(&CachedToken{
		AccessToken: "short-lived-token",
		Issuer:      "https://issuer.example.com",
		Provider:    "oidc",
		ExpiresAt:   time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
	}))

	source := &cachedOIDCTokenSource{cache: cache}

	_, err = source.Token(context.Background())
	require.NoError(t, err)

	// Expiry is inside the buffer, but the cache was just read. Re-reading on
	// every RPC would turn a short-lived token into a refresh per call.
	source.expiresAt = time.Now().Add(time.Second)

	require.NoError(t, cache.Clear())

	token, err := source.Token(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "short-lived-token", token)
}

func TestResolveOIDCTokenSource_AuthTokenWinsOverActionsEnvironment(t *testing.T) {
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "https://actions.example.com/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			AuthToken:     "explicit-token",
			OIDCAudience:  "dir",
		},
	}

	source, err := opts.resolveOIDCTokenSource(context.Background())
	require.NoError(t, err)

	token, err := source(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "explicit-token", token)
}

func TestResolveOIDCTokenSource_PrefersActionsEnvironmentOverCache(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cache, err := NewTokenCacheForIssuer("https://issuer.example.com")
	require.NoError(t, err)
	require.NoError(t, cache.Save(&CachedToken{
		AccessToken: "cached-token",
		Issuer:      "https://issuer.example.com",
		Provider:    "oidc",
		ExpiresAt:   time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
	}))

	// Port 1 refuses immediately, so the error identifies which source ran
	// without depending on name resolution.
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "https://127.0.0.1:1/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
			OIDCAudience:  "dir",
		},
	}

	// Minting happens while dialing, so a broken environment fails before any
	// RPC runs rather than silently falling back to a token that cannot renew.
	_, err = opts.resolveOIDCTokenSource(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to request GitHub OIDC token")
}

func TestSetupOIDCAuth_RequiresIssuerSelectionWhenMultipleCachesExist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cacheA, err := NewTokenCacheForIssuer("https://issuer-a.example.com")
	require.NoError(t, err)
	require.NoError(t, cacheA.Save(&CachedToken{
		AccessToken: "token-a",
		Issuer:      "https://issuer-a.example.com",
		Provider:    "oidc",
		ExpiresAt:   time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
	}))

	cacheB, err := NewTokenCacheForIssuer("https://issuer-b.example.com")
	require.NoError(t, err)
	require.NoError(t, cacheB.Save(&CachedToken{
		AccessToken: "token-b",
		Issuer:      "https://issuer-b.example.com",
		Provider:    "oidc",
		ExpiresAt:   time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
	}))

	opts := &options{
		config: &Config{
			ServerAddress: "gateway.example.com:443",
			AuthMode:      "oidc",
		},
	}

	err = opts.setupOIDCAuth(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "select one explicitly")
	assert.Contains(t, err.Error(), "https://issuer-a.example.com")
	assert.Contains(t, err.Error(), "https://issuer-b.example.com")
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGitHubIDTokenServer serves the Actions OIDC endpoint, handing out a token
// numbered per request so callers can tell a re-mint from a reuse.
func newGitHubIDTokenServer(t *testing.T, requestToken string) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	var mints atomic.Int64

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer "+requestToken, r.Header.Get("Authorization"))

		count := mints.Add(1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"value":"minted-token-%d","count":%d}`, count, count)
	}))
	t.Cleanup(srv.Close)

	return srv, &mints
}

func TestNewGitHubIDTokenSource_RequiresActionsEnvironment(t *testing.T) {
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")
	assert.Nil(t, newGitHubIDTokenSource("dir"))

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "https://actions.example.com/token")
	assert.Nil(t, newGitHubIDTokenSource("dir"), "request token alone is not enough")

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")
	assert.Nil(t, newGitHubIDTokenSource("dir"), "request URL alone is not enough")

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "https://actions.example.com/token")
	assert.NotNil(t, newGitHubIDTokenSource("dir"))
}

func TestGitHubIDTokenSource_MintsWithAudience(t *testing.T) {
	var gotAudience string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAudience = r.URL.Query().Get("audience")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"minted-token"}`))
	}))
	defer srv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL+"/token?api-version=2.0")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	source := newGitHubIDTokenSource("dir")
	require.NotNil(t, source)
	source.httpClient = srv.Client()

	token, err := source.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "minted-token", token)
	assert.Equal(t, "dir", gotAudience)
}

func TestGitHubIDTokenSource_ReMintsOnlyOnceStale(t *testing.T) {
	srv, mints := newGitHubIDTokenServer(t, "request-token")

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL+"/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	source := newGitHubIDTokenSource("dir")
	require.NotNil(t, source)
	source.httpClient = srv.Client()

	first, err := source.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "minted-token-1", first)

	// Still fresh: the same token is served without contacting the endpoint.
	again, err := source.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, first, again)
	assert.Equal(t, int64(1), mints.Load())

	source.mintedAt = time.Now().Add(-githubIDTokenRefreshAfter - time.Second)

	renewed, err := source.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "minted-token-2", renewed)
	assert.Equal(t, int64(2), mints.Load())
}

func TestGitHubIDTokenSource_ServesStaleTokenWhileMintingFails(t *testing.T) {
	var failMinting atomic.Bool

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failMinting.Load() {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"minted-token"}`))
	}))
	defer srv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL+"/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	source := newGitHubIDTokenSource("dir")
	require.NotNil(t, source)
	source.httpClient = srv.Client()

	_, err := source.Token(t.Context())
	require.NoError(t, err)

	failMinting.Store(true)

	source.mintedAt = time.Now().Add(-githubIDTokenRefreshAfter - time.Second)

	// Within the stale window the existing token is still accepted by the
	// gateway, so a failed re-mint must not fail the RPC.
	token, err := source.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "minted-token", token)

	source.mintedAt = time.Now().Add(-githubIDTokenServeStaleFor - time.Second)

	_, err = source.Token(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
}

func TestGitHubIDTokenSource_RejectsNonHTTPSEndpoint(t *testing.T) {
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "http://actions.example.com/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	source := newGitHubIDTokenSource("dir")
	require.NotNil(t, source)

	_, err := source.Token(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must use https")
}

func TestGitHubIDTokenSource_ErrorsNeverLeakCredentials(t *testing.T) {
	const requestToken = "super-secret-request-token" //nolint:gosec // G101: test fixture, not a real credential

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"value":"leaked-token-in-body"}`))
	}))
	defer srv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL+"/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", requestToken)

	source := newGitHubIDTokenSource("dir")
	require.NotNil(t, source)
	source.httpClient = srv.Client()

	_, err := source.Token(t.Context())
	require.Error(t, err)
	assert.NotContains(t, err.Error(), requestToken)
	assert.NotContains(t, err.Error(), "leaked-token-in-body")
}

func TestGitHubIDTokenSource_RejectsEmptyToken(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":"  "}`))
	}))
	defer srv.Close()

	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", srv.URL+"/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	source := newGitHubIDTokenSource("dir")
	require.NotNil(t, source)
	source.httpClient = srv.Client()

	_, err := source.Token(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token")
}

// Minting is opt-in through the audience: GitHub's default audience is the
// repository owner URL, which the gateway rejects, so an audience-less config
// must fall through to the other token sources rather than mint unusable tokens.
func TestNewGitHubIDTokenSource_RequiresAudience(t *testing.T) {
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "https://actions.example.com/token")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "request-token")

	assert.Nil(t, newGitHubIDTokenSource(""))
	assert.Nil(t, newGitHubIDTokenSource("   "), "a blank audience is not a configured audience")
	assert.NotNil(t, newGitHubIDTokenSource("dir"))
}

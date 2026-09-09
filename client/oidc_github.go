// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// GitHub Actions ID tokens are valid for roughly five minutes, so a command that
// runs longer must re-mint rather than reuse the token it started with.
const (
	githubIDTokenRefreshAfter   = 2 * time.Minute
	githubIDTokenServeStaleFor  = 4 * time.Minute
	githubIDTokenRequestTimeout = 10 * time.Second

	// githubIDTokenMaxBody caps the minting response; the endpoint returns a
	// single small JSON object, and the URL comes from the environment.
	githubIDTokenMaxBody = 1 << 20
)

// githubIDTokenSource mints ID tokens from the GitHub Actions OIDC endpoint,
// which is reachable only from inside a workflow that requested `id-token:
// write`. Tokens are minted on demand so a long-running command keeps sending a
// token the gateway still accepts.
type githubIDTokenSource struct {
	requestURL   string
	requestToken string
	audience     string
	httpClient   *http.Client

	mu       sync.Mutex
	token    string
	mintedAt time.Time
}

// newGitHubIDTokenSource returns nil when not running inside a workflow that can
// mint ID tokens, or when no audience was configured, so callers fall through to
// their other token sources. Minting is opt-in through the audience because
// GitHub's default audience is the repository owner URL, which a Directory
// gateway rejects.
func newGitHubIDTokenSource(audience string) *githubIDTokenSource {
	audience = strings.TrimSpace(audience)
	if audience == "" || !inGitHubActionsOIDC() {
		return nil
	}

	return &githubIDTokenSource{
		requestURL:   strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")),
		requestToken: strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")),
		audience:     audience,
		httpClient:   &http.Client{Timeout: githubIDTokenRequestTimeout},
	}
}

// inGitHubActionsOIDC reports whether the ID token endpoint is reachable, which
// is the case only inside a workflow job granted `id-token: write`.
func inGitHubActionsOIDC() bool {
	return strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")) != "" &&
		strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")) != ""
}

func (s *githubIDTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Since(s.mintedAt) < githubIDTokenRefreshAfter {
		return s.token, nil
	}

	token, err := s.mint(ctx)
	if err != nil {
		// The token already held is accepted for a while yet, so prefer it over
		// failing the RPC on what may be a transient minting error.
		if s.token != "" && time.Since(s.mintedAt) < githubIDTokenServeStaleFor {
			authLogger.Debug("Failed to mint GitHub OIDC token, reusing the current one", "error", err)

			return s.token, nil
		}

		return "", err
	}

	s.token, s.mintedAt = token, time.Now()

	return s.token, nil
}

// mint requests a fresh ID token. Errors never include the response body or
// either token, since both are credentials.
func (s *githubIDTokenSource) mint(ctx context.Context) (string, error) {
	requestURL, err := s.tokenRequestURL()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build GitHub OIDC token request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.requestToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request GitHub OIDC token: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close error is not actionable

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to request GitHub OIDC token: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, githubIDTokenMaxBody))
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub OIDC token response: %w", err)
	}

	var payload struct {
		Value string `json:"value"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return "", errors.New("failed to parse GitHub OIDC token response")
	}

	if strings.TrimSpace(payload.Value) == "" {
		return "", errors.New("GitHub OIDC token response contained no token")
	}

	return payload.Value, nil
}

// tokenRequestURL adds the requested audience to the endpoint from the
// environment. The scheme is checked because that endpoint is environment
// supplied, and the request carries a credential that must not leave TLS.
func (s *githubIDTokenSource) tokenRequestURL() (string, error) {
	parsed, err := url.Parse(s.requestURL)
	if err != nil {
		return "", errors.New("invalid ACTIONS_ID_TOKEN_REQUEST_URL")
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("ACTIONS_ID_TOKEN_REQUEST_URL must use https, got %q", parsed.Scheme)
	}

	if s.audience == "" {
		return parsed.String(), nil
	}

	query := parsed.Query()
	query.Set("audience", s.audience)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

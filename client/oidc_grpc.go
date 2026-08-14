// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// setupOIDCAuth configures TLS to the Directory API (e.g. Envoy gateway on :443) and sends the
// OIDC access token as a gRPC Bearer credential. Token is taken from AuthToken config/env, from
// the GitHub Actions OIDC endpoint, or from the dirctl token cache after `dirctl auth login`.
func (o *options) setupOIDCAuth(ctx context.Context) error {
	tokenSource, err := o.resolveOIDCTokenSource(ctx)
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         serverNameFromAddr(o.config.ServerAddress),
		InsecureSkipVerify: o.config.TlsSkipVerify, //nolint:gosec // user-controlled for dev/testing
	}

	o.authOpts = append(o.authOpts,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
		grpc.WithPerRPCCredentials(newOIDCBearerCredentials(tokenSource)),
	)

	return nil
}

// resolveOIDCTokenSource selects where bearer tokens come from for the lifetime
// of the client. Each source is consulted per RPC rather than once at dial, so a
// command that runs longer than a single token still authenticates.
//
// Every source is also exercised once here, so a missing or unusable credential
// fails while dialing instead of on the first RPC.
func (o *options) resolveOIDCTokenSource(ctx context.Context) (tokenSourceFunc, error) {
	// A caller-supplied token cannot be renewed, so it stays exactly as passed.
	if accessToken := strings.TrimSpace(o.config.AuthToken); accessToken != "" {
		return staticTokenSource(accessToken), nil
	}

	if githubSource := newGitHubIDTokenSource(o.config.OIDCAudience); githubSource != nil {
		if _, err := githubSource.Token(ctx); err != nil {
			return nil, err
		}

		return githubSource.Token, nil
	}

	cache, err := ResolveTokenCacheForIssuer(o.config.OIDCIssuer)
	if err != nil {
		if errors.Is(err, ErrNoCachedIssuer) {
			return nil, errors.New("no OIDC access token: run 'dirctl auth login', or set DIRECTORY_CLIENT_AUTH_TOKEN")
		}

		return nil, err
	}

	cacheSource := &cachedOIDCTokenSource{cache: cache, clientID: o.config.OIDCClientID}
	if _, err := cacheSource.Token(ctx); err != nil {
		return nil, err
	}

	return cacheSource.Token, nil
}

// cachedOIDCTokenSource serves tokens from the `dirctl auth login` cache,
// refreshing through the stored refresh token as they age out.
//
// It keeps the resolved token in memory because re-reading the cache file on
// every RPC would be wasteful, and re-resolves once that copy nears expiry. The
// mutex is what makes a refresh safe under concurrent RPCs: an IdP that rotates
// refresh tokens invalidates the old one, so two refreshes in flight at once
// would leave one of them holding a token the IdP has already retired.
type cachedOIDCTokenSource struct {
	cache    *TokenCache
	clientID string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
	lastRead  time.Time
}

// minCachedTokenRecheck bounds how often the cache is re-read, so an unusually
// short-lived access token cannot turn every RPC into a refresh round trip.
const minCachedTokenRecheck = 30 * time.Second

func (s *cachedOIDCTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && !s.needsRecheck() {
		return s.token, nil
	}

	token, expiresAt, err := s.resolve(ctx)
	if err != nil {
		return "", err
	}

	s.token, s.expiresAt, s.lastRead = token, expiresAt, time.Now()

	return s.token, nil
}

func (s *cachedOIDCTokenSource) needsRecheck() bool {
	if time.Since(s.lastRead) < minCachedTokenRecheck {
		return false
	}

	return !time.Now().Add(TokenExpiryBuffer).Before(s.expiresAt)
}

func (s *cachedOIDCTokenSource) resolve(ctx context.Context) (string, time.Time, error) {
	// Fast path: use only a currently valid token.
	// Note: GetValidToken() returns (nil, nil) for both "missing" and "expired" tokens.
	tok, err := s.cache.GetValidToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read OIDC token cache: %w", err)
	}

	if tok != nil && strings.TrimSpace(tok.AccessToken) != "" {
		return tok.AccessToken, cachedTokenExpiry(tok), nil
	}

	// Disambiguate "missing" vs "expired" so we can return a clearer auth error.
	cachedToken, err := s.cache.Load()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read OIDC token cache: %w", err)
	}

	if isExpiredCachedOIDCToken(s.cache, cachedToken) {
		updatedToken, err := RefreshExpiredCachedOIDCToken(ctx, s.cache, cachedToken, s.clientID)
		if err != nil {
			return "", time.Time{}, err
		}

		return updatedToken.AccessToken, cachedTokenExpiry(updatedToken), nil
	}

	return "", time.Time{}, errors.New("no OIDC access token: run 'dirctl auth login', or set DIRECTORY_CLIENT_AUTH_TOKEN")
}

// cachedTokenExpiry mirrors TokenCache.IsValid, which treats a token without an
// explicit expiry as living DefaultTokenValidityDuration from creation.
func cachedTokenExpiry(token *CachedToken) time.Time {
	if token.ExpiresAt.IsZero() {
		return token.CreatedAt.Add(DefaultTokenValidityDuration)
	}

	return token.ExpiresAt
}

func isExpiredCachedOIDCToken(cache *TokenCache, cachedToken *CachedToken) bool {
	if cachedToken == nil {
		return false
	}

	if strings.TrimSpace(cachedToken.AccessToken) == "" {
		return false
	}

	return !cache.IsValid(cachedToken)
}

// RefreshExpiredCachedOIDCToken refreshes an expired cached OIDC token and persists the updated cache atomically.
func RefreshExpiredCachedOIDCToken(ctx context.Context, cache *TokenCache, cachedToken *CachedToken, clientID string) (*CachedToken, error) {
	if strings.TrimSpace(cachedToken.RefreshToken) == "" {
		return nil, errors.New("cached OIDC token has expired and no refresh token is available; run 'dirctl auth login' to refresh authentication")
	}

	refreshResult, err := OIDC.RefreshAccessToken(ctx, &RefreshTokenConfig{
		Issuer:       cachedToken.Issuer,
		ClientID:     clientID,
		RefreshToken: cachedToken.RefreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("cached OIDC token has expired and refresh failed; run 'dirctl auth login' to refresh authentication: %w", err)
	}

	updatedToken := newRefreshedCachedToken(cachedToken, refreshResult)
	if err := cache.SaveAtomic(updatedToken); err != nil {
		return nil, fmt.Errorf("cached OIDC token was refreshed but failed to persist cache; run 'dirctl auth login' to refresh authentication: %w", err)
	}

	return updatedToken, nil
}

func newRefreshedCachedToken(cachedToken *CachedToken, refreshResult *AuthResult) *CachedToken {
	updatedToken := &CachedToken{
		AccessToken:  refreshResult.AccessToken,
		RefreshToken: cachedToken.RefreshToken,
		TokenType:    refreshResult.TokenType,
		Provider:     cachedToken.Provider,
		Issuer:       cachedToken.Issuer,
		ExpiresAt:    refreshResult.ExpiresAt.UTC().Truncate(time.Millisecond),
		User:         cachedToken.User,
		UserID:       cachedToken.UserID,
		Email:        cachedToken.Email,
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}

	if strings.TrimSpace(refreshResult.RefreshToken) != "" {
		updatedToken.RefreshToken = refreshResult.RefreshToken
	}

	if strings.TrimSpace(updatedToken.Provider) == "" {
		updatedToken.Provider = "oidc"
	}

	if strings.TrimSpace(refreshResult.IDToken) == "" {
		return updatedToken
	}

	if refreshResult.Name != "" {
		updatedToken.User = refreshResult.Name
	}

	if refreshResult.Subject != "" {
		updatedToken.UserID = refreshResult.Subject
	}

	if refreshResult.Email != "" {
		updatedToken.Email = refreshResult.Email
	}

	return updatedToken
}

// serverNameFromAddr returns the hostname for TLS SNI from a gRPC dial target (host:port).
func serverNameFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}

	return host
}

// tokenSourceFunc returns the bearer token to use for a single RPC. Resolving
// per RPC is what allows a token to be renewed while a command is running.
type tokenSourceFunc func(ctx context.Context) (string, error)

// staticTokenSource serves a token that cannot be renewed, such as one passed
// with --auth-token.
func staticTokenSource(token string) tokenSourceFunc {
	return func(context.Context) (string, error) { return token, nil }
}

type oidcBearerCredentials struct {
	tokenSource tokenSourceFunc
}

func newOIDCBearerCredentials(tokenSource tokenSourceFunc) credentials.PerRPCCredentials {
	return &oidcBearerCredentials{tokenSource: tokenSource}
}

func (c *oidcBearerCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	token, err := c.tokenSource(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": "Bearer " + token,
	}, nil
}

func (c *oidcBearerCredentials) RequireTransportSecurity() bool {
	return true
}

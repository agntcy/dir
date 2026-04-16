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
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// setupOIDCAuth configures TLS to the Directory API (e.g. Envoy gateway on :443) and sends the
// OIDC access token as a gRPC Bearer credential. Token is taken from AuthToken config/env, or
// from the dirctl token cache after `dirctl auth login`.
func (o *options) setupOIDCAuth(ctx context.Context) error {
	accessToken, err := o.resolveOIDCAccessToken(ctx)
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
		grpc.WithPerRPCCredentials(newOIDCBearerCredentials(accessToken)),
	)

	return nil
}

func (o *options) resolveOIDCAccessToken(ctx context.Context) (string, error) {
	if accessToken := strings.TrimSpace(o.config.AuthToken); accessToken != "" {
		return accessToken, nil
	}

	cache := NewTokenCache()

	// Fast path: use only a currently valid token.
	// Note: GetValidToken() returns (nil, nil) for both "missing" and "expired" tokens.
	tok, err := cache.GetValidToken()
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC token cache: %w", err)
	}

	if tok != nil && strings.TrimSpace(tok.AccessToken) != "" {
		return tok.AccessToken, nil
	}

	// Disambiguate "missing" vs "expired" so we can return a clearer auth error.
	cachedToken, err := cache.Load()
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC token cache: %w", err)
	}

	if isExpiredCachedOIDCToken(cache, cachedToken) {
		return o.refreshExpiredCachedOIDCToken(ctx, cache, cachedToken)
	}

	return "", errors.New("no OIDC access token: run 'dirctl auth login', or set DIRECTORY_CLIENT_AUTH_TOKEN")
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

func (o *options) refreshExpiredCachedOIDCToken(ctx context.Context, cache *TokenCache, cachedToken *CachedToken) (string, error) {
	if strings.TrimSpace(cachedToken.RefreshToken) == "" {
		return "", errors.New("cached OIDC token has expired and no refresh token is available; run 'dirctl auth login' to refresh authentication")
	}

	refreshResult, err := OIDC.RefreshAccessToken(ctx, &RefreshTokenConfig{
		Issuer:       cachedToken.Issuer,
		ClientID:     o.config.OIDCClientID,
		RefreshToken: cachedToken.RefreshToken,
	})
	if err != nil {
		return "", fmt.Errorf("cached OIDC token has expired and refresh failed; run 'dirctl auth login' to refresh authentication: %w", err)
	}

	updatedToken := newRefreshedCachedToken(cachedToken, refreshResult)
	if err := cache.SaveAtomic(updatedToken); err != nil {
		return "", fmt.Errorf("cached OIDC token was refreshed but failed to persist cache; run 'dirctl auth login' to refresh authentication: %w", err)
	}

	return updatedToken.AccessToken, nil
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

type oidcBearerCredentials struct {
	token string
}

func newOIDCBearerCredentials(token string) credentials.PerRPCCredentials {
	return &oidcBearerCredentials{token: token}
}

func (c *oidcBearerCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + c.token,
	}, nil
}

func (c *oidcBearerCredentials) RequireTransportSecurity() bool {
	return true
}

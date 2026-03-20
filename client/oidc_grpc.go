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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// setupOIDCAuth configures TLS to the Directory API (e.g. Envoy gateway on :443) and sends the
// OIDC access token as a gRPC Bearer credential. Token is taken from OIDCToken config/env, or
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
	if accessToken := strings.TrimSpace(o.config.OIDCToken); accessToken != "" {
		return accessToken, nil
	}

	cache := NewTokenCache()

	tok, err := cache.GetValidToken()
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC token cache: %w", err)
	}

	if tok != nil && strings.TrimSpace(tok.AccessToken) != "" {
		return tok.AccessToken, nil
	}

	if !o.hasMachineOIDCConfig() {
		return "", errors.New("no OIDC access token: run 'dirctl auth login' (human) or 'dirctl auth machine' (service), or set DIRECTORY_CLIENT_OIDC_TOKEN")
	}

	issued, err := OIDC.RunClientCredentialsFlow(ctx, &ClientCredentialsConfig{
		Issuer:           o.config.OIDCIssuer,
		TokenEndpoint:    o.config.OIDCMachineTokenEndpoint,
		ClientID:         o.config.OIDCMachineClientID,
		ClientSecret:     o.config.OIDCMachineClientSecret,
		ClientSecretFile: o.config.OIDCMachineClientSecretFile,
		Scopes:           o.config.OIDCMachineScopes,
		Timeout:          DefaultOAuthTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("failed to obtain machine OIDC token: %w", err)
	}

	userID := issued.Subject
	if userID == "" {
		userID = o.config.OIDCMachineClientID
	}

	if err := cache.Save(&CachedToken{
		AccessToken:  issued.AccessToken,
		TokenType:    issued.TokenType,
		Provider:     "oidc",
		Issuer:       o.config.OIDCIssuer,
		RefreshToken: "",
		ExpiresAt:    issued.ExpiresAt,
		User:         o.config.OIDCMachineClientID,
		UserID:       userID,
	}); err != nil {
		authLogger.Warn("Failed to cache machine OIDC token; continuing with in-memory token", "error", err)
	}

	return issued.AccessToken, nil
}

func (o *options) hasMachineOIDCConfig() bool {
	return strings.TrimSpace(o.config.OIDCIssuer) != "" &&
		strings.TrimSpace(o.config.OIDCMachineClientID) != "" &&
		(strings.TrimSpace(o.config.OIDCMachineClientSecret) != "" || strings.TrimSpace(o.config.OIDCMachineClientSecretFile) != "")
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

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
func (o *options) setupOIDCAuth(_ context.Context) error {
	accessToken := strings.TrimSpace(o.config.OIDCToken)
	if accessToken == "" {
		cache := NewTokenCache()

		tok, err := cache.GetValidToken()
		if err != nil {
			return fmt.Errorf("failed to read OIDC token cache: %w", err)
		}

		if tok == nil || strings.TrimSpace(tok.AccessToken) == "" {
			return errors.New("no OIDC access token: run 'dirctl auth login' or set DIRECTORY_CLIENT_OIDC_TOKEN")
		}

		accessToken = tok.AccessToken
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

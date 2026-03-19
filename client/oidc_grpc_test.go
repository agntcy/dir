// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
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

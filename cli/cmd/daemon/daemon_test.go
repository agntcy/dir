// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"testing"
	"time"

	storeconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigUsesMacOSFriendlyLocalRegistryPort(t *testing.T) {
	originalOpts := opts
	opts = &Options{DataDir: t.TempDir()}
	t.Cleanup(func() {
		opts = originalOpts
	})

	cfg, err := loadConfig()

	require.NoError(t, err)
	require.Equal(t, "localhost:5555", cfg.Server.Store.OCI.RegistryAddress)
	require.Equal(t, "localhost:5555", cfg.Reconciler.LocalRegistry.RegistryAddress)
}

// TestLoadConfigRepublishIntervalEnvOverride asserts that the routing republish
// interval can be overridden by environment alone. AutomaticEnv only resolves
// keys viper already knows, so this depends on the key being declared in the
// embedded daemon.config.yaml.
func TestLoadConfigRepublishIntervalEnvOverride(t *testing.T) {
	originalOpts := opts
	opts = &Options{DataDir: t.TempDir()}
	t.Cleanup(func() {
		opts = originalOpts
	})

	cfg, err := loadConfig()
	require.NoError(t, err)
	require.Equal(t, 36*time.Hour, cfg.Server.Routing.RepublishInterval)

	t.Setenv("DIRECTORY_DAEMON_SERVER_ROUTING_REPUBLISH_INTERVAL", "10m")

	cfg, err = loadConfig()
	require.NoError(t, err)
	require.Equal(t, 10*time.Minute, cfg.Server.Routing.RepublishInterval)
}

// TestEmbeddedZot tests the embedded Zot server.
func TestEmbeddedZot(t *testing.T) {
	address := storeconfig.DefaultRegistryAddress
	rootDirectory := "/tmp/agntcy/dir/oci/"

	go func() {
		ctx := runEmbeddedZot(context.Background(), address, rootDirectory)

		defer ctx.Done()
	}()

	var (
		zotIsReady bool
		err        error
	)

	for range 10 {
		zotIsReady, err = isZotReady(address)
		if err == nil && zotIsReady {
			break
		}

		time.Sleep(1 * time.Second)
	}

	require.NoError(t, err)
	require.True(t, zotIsReady)
}

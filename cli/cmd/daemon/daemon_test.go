// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"os"
	"path/filepath"
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

// TestLoadConfigRoutingAddressEnvOverride asserts that the advertised routing
// endpoints can be set by environment alone. They have no default and are absent
// from the embedded daemon.config.yaml, so this depends on them being registered
// in registerServerDefaults.
func TestLoadConfigRoutingAddressEnvOverride(t *testing.T) {
	originalOpts := opts
	opts = &Options{DataDir: t.TempDir()}
	t.Cleanup(func() {
		opts = originalOpts
	})

	cfg, err := loadConfig()
	require.NoError(t, err)
	require.Empty(t, cfg.Server.Routing.DirectoryAPIAddress)
	require.Empty(t, cfg.Server.Routing.DirectoryOCIAddress)

	t.Setenv("DIRECTORY_DAEMON_SERVER_ROUTING_DIRECTORY_API_ADDRESS", "dir.example.com:8888")
	t.Setenv("DIRECTORY_DAEMON_SERVER_ROUTING_DIRECTORY_OCI_ADDRESS", "ghcr.io/org/agents")

	cfg, err = loadConfig()
	require.NoError(t, err)
	require.Equal(t, "dir.example.com:8888", cfg.Server.Routing.DirectoryAPIAddress)
	require.Equal(t, "ghcr.io/org/agents", cfg.Server.Routing.DirectoryOCIAddress)
}

// TestLoadConfigRoutingAddressEnvOverrideWithUserConfig asserts the same holds
// for a user-supplied config file that does not declare the keys, since --config
// is read as-is without merging the embedded defaults.
func TestLoadConfigRoutingAddressEnvOverrideWithUserConfig(t *testing.T) {
	originalOpts := opts
	dataDir := t.TempDir()

	configPath := filepath.Join(dataDir, DefaultConfigFile)
	require.NoError(t, os.WriteFile(configPath, []byte(defaultConfigYAML), 0o600))

	opts = &Options{DataDir: dataDir, ConfigFile: configPath}

	t.Cleanup(func() {
		opts = originalOpts
	})

	t.Setenv("DIRECTORY_DAEMON_SERVER_ROUTING_DIRECTORY_OCI_ADDRESS", "ghcr.io/org/agents")

	cfg, err := loadConfig()
	require.NoError(t, err)
	require.Equal(t, "ghcr.io/org/agents", cfg.Server.Routing.DirectoryOCIAddress)
}

// TestLoadConfigLocalRegistryCredentialEnvOverride asserts that the reconciler
// local registry credentials can be set by environment alone, including for a
// user-supplied config file that does not declare them, so a token never has to
// be written into the config.
func TestLoadConfigLocalRegistryCredentialEnvOverride(t *testing.T) {
	dataDir := t.TempDir()

	configPath := filepath.Join(dataDir, DefaultConfigFile)
	require.NoError(t, os.WriteFile(configPath, []byte(defaultConfigYAML), 0o600))

	for name, configFile := range map[string]string{
		"embedded config":    "",
		"user-supplied file": configPath,
	} {
		t.Run(name, func(t *testing.T) {
			originalOpts := opts
			opts = &Options{DataDir: dataDir, ConfigFile: configFile}

			t.Cleanup(func() {
				opts = originalOpts
			})

			cfg, err := loadConfig()
			require.NoError(t, err)
			require.Empty(t, cfg.Reconciler.LocalRegistry.Username)
			require.Empty(t, cfg.Reconciler.LocalRegistry.Password)

			t.Setenv("DIRECTORY_DAEMON_RECONCILER_LOCAL_REGISTRY_AUTH_CONFIG_USERNAME", "registry-user")
			t.Setenv("DIRECTORY_DAEMON_RECONCILER_LOCAL_REGISTRY_AUTH_CONFIG_PASSWORD", "registry-token")

			cfg, err = loadConfig()
			require.NoError(t, err)
			require.Equal(t, "registry-user", cfg.Reconciler.LocalRegistry.Username)
			require.Equal(t, "registry-token", cfg.Reconciler.LocalRegistry.Password)
		})
	}
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

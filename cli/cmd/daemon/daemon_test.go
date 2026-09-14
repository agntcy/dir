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
	"github.com/stretchr/testify/assert"
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

// TestLoadConfigExtractorEnvOverride asserts the extractor settings can be set
// by environment alone. They have no defaults and are absent from the embedded
// daemon.config.yaml, so this depends on them being registered in
// registerServerDefaults — AutomaticEnv cannot discover keys Viper has never
// seen.
func TestLoadConfigExtractorEnvOverride(t *testing.T) {
	originalOpts := opts
	opts = &Options{DataDir: t.TempDir()}
	t.Cleanup(func() {
		opts = originalOpts
	})

	cfg, err := loadConfig()
	require.NoError(t, err)
	require.Empty(t, cfg.Server.Extractor.RemoteAddr)
	require.Empty(t, cfg.Server.Extractor.AssetDir)
	require.Empty(t, cfg.Server.Extractor.OASFURL)

	t.Setenv("DIRECTORY_DAEMON_SERVER_EXTRACTOR_REMOTE_ADDR", "localhost:31234")
	t.Setenv("DIRECTORY_DAEMON_SERVER_EXTRACTOR_ASSET_DIR", "/tmp/assets")
	t.Setenv("DIRECTORY_DAEMON_SERVER_EXTRACTOR_OASF_URL", "https://schema.example.com")

	cfg, err = loadConfig()
	require.NoError(t, err)
	require.Equal(t, "localhost:31234", cfg.Server.Extractor.RemoteAddr)
	require.Equal(t, "/tmp/assets", cfg.Server.Extractor.AssetDir)
	require.Equal(t, "https://schema.example.com", cfg.Server.Extractor.OASFURL)
}

// TestLoadConfigExtractorEnvOverrideWithUserConfig asserts the same holds for a
// user-supplied config file that does not declare the keys, since --config is
// read as-is without merging the embedded defaults.
func TestLoadConfigExtractorEnvOverrideWithUserConfig(t *testing.T) {
	originalOpts := opts
	dataDir := t.TempDir()

	configPath := filepath.Join(dataDir, DefaultConfigFile)
	require.NoError(t, os.WriteFile(configPath, []byte(defaultConfigYAML), 0o600))

	opts = &Options{DataDir: dataDir, ConfigFile: configPath}

	t.Cleanup(func() {
		opts = originalOpts
	})

	t.Setenv("DIRECTORY_DAEMON_SERVER_EXTRACTOR_REMOTE_ADDR", "oasf-sdk:31234")

	cfg, err := loadConfig()
	require.NoError(t, err)
	require.Equal(t, "oasf-sdk:31234", cfg.Server.Extractor.RemoteAddr)
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

// TestLoadConfigIdentityAnsEnvOverride asserts that the identity task and its
// ans block bind from the environment under both the server and the reconciler
// prefixes, and that a relative ca_file resolves against the data directory.
func TestLoadConfigIdentityAnsEnvOverride(t *testing.T) {
	originalOpts := opts
	dataDir := t.TempDir()
	opts = &Options{DataDir: dataDir}

	t.Cleanup(func() {
		opts = originalOpts
	})

	t.Setenv("DIRECTORY_DAEMON_SERVER_IDENTITY_ANS_ENABLED", "true")
	t.Setenv("DIRECTORY_DAEMON_SERVER_IDENTITY_ANS_TRUSTED_LOG_HOSTS", "log-a.example.com:8443,log-b.example.com")
	t.Setenv("DIRECTORY_DAEMON_SERVER_IDENTITY_ANS_CA_FILE", "certs/ans-log-ca.pem")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_IDENTITY_ENABLED", "true")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_IDENTITY_INTERVAL", "2m")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_IDENTITY_ANS_ENABLED", "true")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_IDENTITY_ANS_TIMEOUT", "3s")
	t.Setenv("DIRECTORY_DAEMON_RECONCILER_IDENTITY_ANS_CA_FILE", "/etc/agntcy/ans-log-ca.pem")

	cfg, err := loadConfig()
	require.NoError(t, err)

	assert.True(t, cfg.Server.Identity.Ans.Enabled)
	assert.Equal(t, []string{"log-a.example.com:8443", "log-b.example.com"}, cfg.Server.Identity.Ans.TrustedLogHosts)
	assert.Equal(t, filepath.Join(dataDir, "certs", "ans-log-ca.pem"), cfg.Server.Identity.Ans.CAFile)
	assert.True(t, cfg.Reconciler.Identity.Enabled)
	assert.Equal(t, 2*time.Minute, cfg.Reconciler.Identity.Interval)
	assert.True(t, cfg.Reconciler.Identity.Ans.Enabled)
	assert.Equal(t, 3*time.Second, cfg.Reconciler.Identity.Ans.Timeout)
	assert.Equal(t, "/etc/agntcy/ans-log-ca.pem", cfg.Reconciler.Identity.Ans.CAFile)
}

func TestIdentityDriftWarning(t *testing.T) {
	tests := []struct {
		name          string
		serverAns     bool
		reconcilerAns bool
		task          bool
		want          string
	}{
		{name: "ans off everywhere"},
		{name: "ans on in both with the task running", serverAns: true, reconcilerAns: true, task: true},
		{name: "ans on in the server only", serverAns: true, task: true, want: "only one of"},
		{name: "ans on in the reconciler only", reconcilerAns: true, task: true, want: "only one of"},
		{name: "ans on in both but the task is off", serverAns: true, reconcilerAns: true, want: "identity task is off"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &DaemonConfig{}
			cfg.Server.Identity.Ans.Enabled = tt.serverAns
			cfg.Reconciler.Identity.Ans.Enabled = tt.reconcilerAns
			cfg.Reconciler.Identity.Enabled = tt.task

			got := identityDriftWarning(cfg)
			if tt.want == "" {
				assert.Empty(t, got)

				return
			}

			assert.Contains(t, got, tt.want)
		})
	}
}

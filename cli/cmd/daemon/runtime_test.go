// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/agntcy/dir-runtime/discovery"
	adapterprocess "github.com/agntcy/dir-runtime/discovery/runtime/process"
	runtimestoresql "github.com/agntcy/dir-runtime/store/sql"
	runtimestoretypes "github.com/agntcy/dir-runtime/store/types"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/agntcy/dir/client"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigRuntimeAdapters(t *testing.T) {
	originalOpts := opts

	t.Cleanup(func() {
		opts = originalOpts
	})

	t.Run("embedded default runs docker and process discovery", func(t *testing.T) {
		opts = &Options{DataDir: t.TempDir()}

		cfg, err := loadConfig()

		require.NoError(t, err)
		require.Equal(t, []string{"docker", "process"}, cfg.Runtime.EffectiveAdapterTypes())
		require.Empty(t, cfg.Runtime.IgnoredAdapterType())
		require.Equal(t, adapterprocess.DefaultPollInterval, cfg.Runtime.Adapter.Process.PollInterval)
		require.Equal(t, adapterprocess.DefaultLabelKey, cfg.Runtime.Adapter.Process.LabelKey)
		require.Equal(t, []string{runtimev1.DefaultOASFSchemaURL}, cfg.Runtime.OASFSchemaURLs)
		require.Equal(t, runtimestoresql.DefaultPath, cfg.Runtime.Store.SQLite.Path)
	})

	t.Run("a file that only sets adapter.type keeps that one adapter", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "daemon.yaml")
		require.NoError(t, os.WriteFile(file, []byte("runtime:\n  enabled: true\n  adapter:\n    type: kubernetes\n"), 0o600))

		opts = &Options{DataDir: t.TempDir(), ConfigFile: file}

		cfg, err := loadConfig()

		require.NoError(t, err)
		require.Equal(t, []string{"kubernetes"}, cfg.Runtime.EffectiveAdapterTypes())
		require.Empty(t, cfg.Runtime.IgnoredAdapterType())
		require.Equal(t, []string{runtimev1.DefaultOASFSchemaURL}, cfg.Runtime.OASFSchemaURLs)
	})

	t.Run("a file that sets no adapter runs docker", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "daemon.yaml")
		require.NoError(t, os.WriteFile(file, []byte("runtime:\n  enabled: true\n"), 0o600))

		opts = &Options{DataDir: t.TempDir(), ConfigFile: file}

		cfg, err := loadConfig()

		require.NoError(t, err)
		require.Equal(t, []string{"docker"}, cfg.Runtime.EffectiveAdapterTypes())
		require.Empty(t, cfg.Runtime.IgnoredAdapterType())
	})

	t.Run("the singular environment variable is reported as ignored next to the list", func(t *testing.T) {
		t.Setenv("DIRECTORY_DAEMON_RUNTIME_ADAPTER_TYPE", "process")

		opts = &Options{DataDir: t.TempDir()}

		cfg, err := loadConfig()

		require.NoError(t, err)
		require.Equal(t, []string{"docker", "process"}, cfg.Runtime.EffectiveAdapterTypes())
		require.Equal(t, "process", cfg.Runtime.IgnoredAdapterType())
	})

	t.Run("environment overrides the adapter list and schema servers", func(t *testing.T) {
		t.Setenv("DIRECTORY_DAEMON_RUNTIME_ADAPTER_TYPES", "process")
		t.Setenv("DIRECTORY_DAEMON_RUNTIME_OASF_SCHEMA_URLS", "https://schema.oasf.outshift.com,http://localhost:8080")

		opts = &Options{DataDir: t.TempDir()}

		cfg, err := loadConfig()

		require.NoError(t, err)
		require.Equal(t, []string{"process"}, cfg.Runtime.EffectiveAdapterTypes())
		require.Equal(t, []string{"https://schema.oasf.outshift.com", "http://localhost:8080"}, cfg.Runtime.OASFSchemaURLs)
	})

	t.Run("environment sets the discovery instance ID", func(t *testing.T) {
		t.Setenv("DIRECTORY_DAEMON_RUNTIME_INSTANCE_ID", "node-west")
		opts = &Options{DataDir: t.TempDir()}

		cfg, err := loadConfig()
		require.NoError(t, err)
		require.Equal(t, "node-west", cfg.Runtime.InstanceID)
	})
}

func TestRuntimeInstanceIDForSharedStore(t *testing.T) {
	dataDir := t.TempDir()
	cfg := RuntimeConfig{}
	cfg.Store.Type = runtimestoretypes.StoreType("etcd")

	first, err := cfg.EffectiveInstanceID(dataDir)
	require.NoError(t, err)
	require.NotEmpty(t, first)

	second, err := cfg.EffectiveInstanceID(dataDir)
	require.NoError(t, err)
	require.Equal(t, first, second, "restarting from the same data directory must retain ownership")

	other, err := cfg.EffectiveInstanceID(t.TempDir())
	require.NoError(t, err)
	require.NotEqual(t, first, other, "different daemons must not share an owner ID")

	cfg.InstanceID = "operator-selected"
	override, err := cfg.EffectiveInstanceID(dataDir)
	require.NoError(t, err)
	require.Equal(t, "operator-selected", override)
}

func TestRuntimeInstanceIDKeepsLocalStoreCompatible(t *testing.T) {
	dataDir := t.TempDir()
	cfg := RuntimeConfig{}
	cfg.Store.Type = runtimestoresql.StoreTypeSqlite
	cfg.Store.SQLite.Path = filepath.Join(dataDir, "workloads.db")

	id, err := cfg.EffectiveInstanceID(dataDir)
	require.NoError(t, err)
	require.Empty(t, id, "existing local workloads have no instance annotation")
}

func TestRuntimeInstanceIDForDefaultSQLiteStore(t *testing.T) {
	dataDir := t.TempDir()
	cfg := RuntimeConfig{}
	cfg.Store.Type = runtimestoresql.StoreTypeSqlite
	cfg.Store.SQLite.Path = runtimestoresql.DefaultPath

	first, err := cfg.EffectiveInstanceID(dataDir)
	require.NoError(t, err)
	require.NotEmpty(t, first, "the default SQLite path is shared across daemon data directories")

	second, err := cfg.EffectiveInstanceID(dataDir)
	require.NoError(t, err)
	require.Equal(t, first, second)

	other, err := cfg.EffectiveInstanceID(t.TempDir())
	require.NoError(t, err)
	require.NotEqual(t, first, other)
}

func TestRuntimeInstanceIDForExplicitSharedSQLiteStore(t *testing.T) {
	sharedPath := filepath.Join(t.TempDir(), "workloads.db")
	cfg := RuntimeConfig{}
	cfg.Store.Type = runtimestoresql.StoreTypeSqlite
	cfg.Store.SQLite.Path = sharedPath

	first, err := cfg.EffectiveInstanceID(t.TempDir())
	require.NoError(t, err)
	require.NotEmpty(t, first)

	second, err := cfg.EffectiveInstanceID(t.TempDir())
	require.NoError(t, err)
	require.NotEqual(t, first, second)
}

func TestCountLegacyWorkloadsOnlyMatchesUnownedRuntimeType(t *testing.T) {
	workloads := []*runtimev1.Workload{
		{Runtime: "docker"},
		{Runtime: "docker", Annotations: map[string]string{"other": "value"}},
		{Runtime: "docker", Annotations: map[string]string{discovery.InstanceAnnotation: "node-west"}},
		{Runtime: "process"},
	}

	require.Equal(t, 2, countLegacyWorkloads(workloads, "docker"))
	require.Equal(t, 1, countLegacyWorkloads(workloads, "process"))
	require.Zero(t, countLegacyWorkloads(workloads, "kubernetes"))
}

func TestEffectiveAdapterTypes(t *testing.T) {
	require.Equal(t, []string{"docker"}, RuntimeConfig{}.EffectiveAdapterTypes())
	require.Empty(t, RuntimeConfig{}.IgnoredAdapterType())

	withList := RuntimeConfig{AdapterTypes: []string{"process"}}
	withList.Adapter.Type = "docker"
	require.Equal(t, []string{"process"}, withList.EffectiveAdapterTypes())
	require.Equal(t, "docker", withList.IgnoredAdapterType())

	// Ignored even when the list contains it: the list was chosen, not adapter.type.
	listed := RuntimeConfig{AdapterTypes: []string{"docker", "process"}}
	listed.Adapter.Type = "process"
	require.Equal(t, "process", listed.IgnoredAdapterType())

	legacy := RuntimeConfig{}
	legacy.Adapter.Type = "kubernetes"
	require.Equal(t, []string{"kubernetes"}, legacy.EffectiveAdapterTypes())
	require.Empty(t, legacy.IgnoredAdapterType())
}

func TestEffectiveResolverDefaultsTheOASFClientToThisDaemon(t *testing.T) {
	cfg := RuntimeConfig{}
	cfg.Resolver.OASF.Enabled = true

	r := cfg.EffectiveResolver("0.0.0.0:8888")
	require.NotNil(t, r.OASF.Client)
	require.Equal(t, "127.0.0.1:8888", r.OASF.Client.ServerAddress)
	require.Equal(t, "insecure", r.OASF.Client.AuthMode)
	require.Nil(t, cfg.Resolver.OASF.Client, "the configured value must not change")

	require.Equal(t, "localhost:8888", cfg.EffectiveResolver("localhost:8888").OASF.Client.ServerAddress)
	require.Equal(t, "127.0.0.1:9999", cfg.EffectiveResolver(":9999").OASF.Client.ServerAddress)

	own := cfg
	own.Resolver.OASF.Client = &client.Config{ServerAddress: "dir.example.com:443", AuthMode: "oidc"}
	require.Equal(t, "dir.example.com:443", own.EffectiveResolver("0.0.0.0:8888").OASF.Client.ServerAddress)

	off := RuntimeConfig{}
	require.Nil(t, off.EffectiveResolver("0.0.0.0:8888").OASF.Client, "a disabled resolver gets no client")
}

func TestWaitForShutdown(t *testing.T) {
	t.Run("runtime discovery failing is an error", func(t *testing.T) {
		discoveryErr := make(chan error, 1)
		discoveryErr <- errors.New("runtime discovery failed for every adapter")

		err := waitForShutdown(context.Background(), make(chan os.Signal), discoveryErr, make(chan struct{}))
		require.ErrorContains(t, err, "every adapter")
	})

	t.Run("a signal is a normal shutdown", func(t *testing.T) {
		sig := make(chan os.Signal, 1)
		sig <- syscall.SIGTERM

		require.NoError(t, waitForShutdown(context.Background(), sig, make(chan error), make(chan struct{})))
	})

	t.Run("a cancelled context is a normal shutdown", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.NoError(t, waitForShutdown(ctx, make(chan os.Signal), make(chan error), make(chan struct{})))
	})
}

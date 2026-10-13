// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"os"
	"path/filepath"
	"testing"

	adapterprocess "github.com/agntcy/dir-runtime/discovery/runtime/process"
	runtimestoresql "github.com/agntcy/dir-runtime/store/sql"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
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
		require.Equal(t, []string{runtimev1.DefaultOASFSchemaURL}, cfg.Runtime.OASFSchemaURLs)
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
}

func TestEffectiveAdapterTypes(t *testing.T) {
	require.Empty(t, RuntimeConfig{}.EffectiveAdapterTypes())

	withList := RuntimeConfig{AdapterTypes: []string{"process"}}
	withList.Adapter.Type = "docker"
	require.Equal(t, []string{"process"}, withList.EffectiveAdapterTypes())

	legacy := RuntimeConfig{}
	legacy.Adapter.Type = "docker"
	require.Equal(t, []string{"docker"}, legacy.EffectiveAdapterTypes())
}

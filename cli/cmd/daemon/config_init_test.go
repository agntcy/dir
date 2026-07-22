// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newConfigInitTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)

	return cmd, out
}

// withConfigInitState resets the package-level opts/configInitOpts globals
// for the duration of a test and restores them afterwards. Both structs are
// mutated by cobra flag binding, so tests must not leak state between runs.
func withConfigInitState(t *testing.T, dataDir string) {
	t.Helper()

	originalOpts := opts
	originalConfigInitOpts := configInitOpts

	opts = &Options{DataDir: dataDir}
	configInitOpts = &struct {
		Output string
		Force  bool
	}{}

	t.Cleanup(func() {
		opts = originalOpts
		configInitOpts = originalConfigInitOpts
	})
}

func TestRunConfigInitWritesDefaultConfig(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	cmd, out := newConfigInitTestCmd()
	err := runConfigInit(cmd, nil)
	require.NoError(t, err)

	target := filepath.Join(dataDir, DefaultConfigFile)
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML, string(got)) //nolint:testifylint // byte-exact dump check; YAMLEq would ignore the comment and formatting fidelity the dump preserves
	require.Contains(t, out.String(), target)
}

func TestRunConfigInitRefusesOverwriteWithoutForce(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	cmd, _ := newConfigInitTestCmd()
	require.NoError(t, runConfigInit(cmd, nil))

	cmd2, _ := newConfigInitTestCmd()
	err := runConfigInit(cmd2, nil)
	require.ErrorContains(t, err, "already exists")

	// The original file must be untouched.
	target := filepath.Join(dataDir, DefaultConfigFile)
	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	require.Equal(t, defaultConfigYAML, string(got)) //nolint:testifylint // byte-exact dump check; YAMLEq would ignore the comment and formatting fidelity the dump preserves
}

func TestRunConfigInitOverwritesWithForce(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	target := filepath.Join(dataDir, DefaultConfigFile)
	require.NoError(t, os.WriteFile(target, []byte("stale: true\n"), 0o600))

	configInitOpts.Force = true

	cmd, _ := newConfigInitTestCmd()
	require.NoError(t, runConfigInit(cmd, nil))

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML, string(got)) //nolint:testifylint // byte-exact dump check; YAMLEq would ignore the comment and formatting fidelity the dump preserves
}

func TestRunConfigInitRespectsOutputFlag(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	target := filepath.Join(t.TempDir(), "nested", "custom.yaml")
	configInitOpts.Output = target

	cmd, out := newConfigInitTestCmd()
	require.NoError(t, runConfigInit(cmd, nil))

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML, string(got)) //nolint:testifylint // byte-exact dump check; YAMLEq would ignore the comment and formatting fidelity the dump preserves
	require.Contains(t, out.String(), target)

	// data-dir's default location must be untouched.
	require.NoFileExists(t, filepath.Join(dataDir, DefaultConfigFile))
}

func TestRunConfigInitRespectsConfigFileOverride(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	target := filepath.Join(t.TempDir(), "explicit.yaml")
	opts.ConfigFile = target

	cmd, _ := newConfigInitTestCmd()
	require.NoError(t, runConfigInit(cmd, nil))

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML, string(got)) //nolint:testifylint // byte-exact dump check; YAMLEq would ignore the comment and formatting fidelity the dump preserves
}

// TestWriteIfAbsentRejectsExistingFile exercises writeIfAbsent directly:
// a second write to the same path must fail with errConfigExists and leave
// the first file's contents untouched, and must not leak the staging temp
// file into the directory.
func TestWriteIfAbsentRejectsExistingFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yaml")

	require.NoError(t, writeIfAbsent(target, []byte("first"), 0o600))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no leftover temp file after a successful write")

	err = writeIfAbsent(target, []byte("second"), 0o600)
	require.ErrorIs(t, err, errConfigExists)

	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	require.Equal(t, "first", string(got))
}

// TestRunConfigInitWritesRestrictivePerms confirms the dumped file lands with
// mode 0600 on both write paths: the default (writeIfAbsent) path and the
// --force (fsutil.WriteAtomic) path apply the chmod through different code, so
// both are checked. The config can hold credentials, so a world-readable dump
// would be a real leak. POSIX-only: Windows does not honour Unix permission
// bits.
func TestRunConfigInitWritesRestrictivePerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions are not enforced on Windows")
	}

	tests := []struct {
		name  string
		force bool
	}{
		{name: "default path", force: false},
		{name: "force path", force: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			withConfigInitState(t, dataDir)

			target := filepath.Join(dataDir, DefaultConfigFile)
			if tt.force {
				// Force overwrites, so seed an existing file first. The
				// atomic write replaces the inode, so the resulting mode
				// comes from the chmod, not from this seed.
				require.NoError(t, os.WriteFile(target, []byte("stale: true\n"), 0o600))

				configInitOpts.Force = true
			}

			cmd, _ := newConfigInitTestCmd()
			require.NoError(t, runConfigInit(cmd, nil))

			info, err := os.Stat(target)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		})
	}
}

// TestRunConfigInitErrorsWhenDirCreationFails confirms a failure to create the
// parent directory is surfaced (not swallowed) and leaves nothing behind. The
// parent path element is a regular file, so MkdirAll cannot create the
// directory under it.
func TestRunConfigInitErrorsWhenDirCreationFails(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

	target := filepath.Join(blocker, "config.yaml")
	configInitOpts.Output = target

	cmd, _ := newConfigInitTestCmd()
	err := runConfigInit(cmd, nil)
	require.ErrorContains(t, err, "create directory")
	require.NoFileExists(t, target)
}

func TestRunConfigInitDumpedFileLoadsUnchanged(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	// Load the config the way `dirctl daemon start` would without --config:
	// straight from the embedded default.
	wantCfg, err := loadConfig()
	require.NoError(t, err)

	cmd, _ := newConfigInitTestCmd()
	require.NoError(t, runConfigInit(cmd, nil))

	// Point --config at the dumped file and confirm loadConfig produces the
	// exact same configuration (mirrors what `dirctl daemon start --config
	// <file>` would do). Full-struct equality, not just a spot-checked
	// field, so any drift between the embedded and dumped config is caught.
	opts.ConfigFile = filepath.Join(dataDir, DefaultConfigFile)

	gotCfg, err := loadConfig()
	require.NoError(t, err)
	require.Equal(t, wantCfg, gotCfg)
}

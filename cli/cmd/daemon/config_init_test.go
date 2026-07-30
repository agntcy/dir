// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// osWindows keeps the GOOS guards below from repeating the literal.
const osWindows = "windows"

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
	configInitOpts = &configInitOptions{}

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

// The data directory this command creates is the one the daemon later fills
// with its peer identity key, database, and object store. `daemon start`
// creates it 0700, and MkdirAll is a no-op on an existing directory, so if
// `config init` gets there first with a looser mode nothing ever repairs it.
// Both write paths are checked because --force goes through
// fsutil.WriteAtomic, which creates parents at its own 0755.
func TestRunConfigInitCreatesDataDirWithRestrictivePerms(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%v", force), func(t *testing.T) {
			parent := t.TempDir()
			dataDir := filepath.Join(parent, "agntcy-dir")
			withConfigInitState(t, dataDir)

			configInitOpts.Force = force

			cmd, _ := newConfigInitTestCmd()
			require.NoError(t, runConfigInit(cmd, nil))

			info, err := os.Stat(dataDir)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o700), info.Mode().Perm(),
				"data directory must not be broader than the 0700 daemon start uses")
		})
	}
}

// Nothing else in this file goes through cobra: the other tests call
// runConfigInit directly with a bare command. That leaves flag registration
// untested, so a renamed flag, a wrong binding, or a shorthand collision would
// not fail anything. This drives the real command with argv.
func TestConfigInitCommand_ParsesFlags(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	target := filepath.Join(t.TempDir(), "custom.config.yaml")

	// Rebind the flags to the test's replacement opts struct, since init()
	// bound them to the original one at package load.
	cmd := &cobra.Command{Use: "init", RunE: runConfigInit}
	cmd.Flags().StringVarP(&configInitOpts.Output, "output", "o", "", "")
	cmd.Flags().BoolVar(&configInitOpts.Force, "force", false, "")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"--output", target})

	require.NoError(t, cmd.Execute())
	require.FileExists(t, target)

	// And --force actually reaches the overwrite path.
	cmd.SetArgs([]string{"--output", target, "--force"})
	require.NoError(t, cmd.Execute())

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, defaultConfigYAML, string(got)) //nolint:testifylint // byte-exact dump check
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
	if runtime.GOOS == osWindows {
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

// --force takes a different write path (fsutil.WriteAtomic) from the default
// one, so its failure mode needs its own cover: pointing --output at an
// existing directory must surface an error rather than a panic or a silent
// no-op, and must not leave the directory disturbed.
func TestRunConfigInitForceErrorsWhenTargetIsDirectory(t *testing.T) {
	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	target := filepath.Join(t.TempDir(), "config-as-a-directory")
	require.NoError(t, os.MkdirAll(target, 0o755))

	configInitOpts.Output = target
	configInitOpts.Force = true

	cmd, _ := newConfigInitTestCmd()
	err := runConfigInit(cmd, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to write default config")

	info, statErr := os.Stat(target)
	require.NoError(t, statErr)
	require.True(t, info.IsDir(), "the target directory should be left as a directory")
}

// A directory the process cannot write to fails at the temp-file stage rather
// than at MkdirAll, which is a different branch from
// TestRunConfigInitErrorsWhenDirCreationFails.
func TestRunConfigInitErrorsWhenDirectoryIsNotWritable(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("POSIX permission bits do not gate directory writes on Windows")
	}

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permission bits")
	}

	dataDir := t.TempDir()
	withConfigInitState(t, dataDir)

	readonly := filepath.Join(t.TempDir(), "readonly")
	require.NoError(t, os.MkdirAll(readonly, 0o500))

	t.Cleanup(func() { _ = os.Chmod(readonly, 0o700) })

	target := filepath.Join(readonly, "config.yaml")
	configInitOpts.Output = target

	cmd, _ := newConfigInitTestCmd()
	err := runConfigInit(cmd, nil)
	require.ErrorContains(t, err, "create")
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
	// same configuration, mirroring `dirctl daemon start --config <file>`.
	//
	// This checks that the dumped file is loadable and round-trips through
	// viper's file path to the same struct. It does NOT catch content drift:
	// writeDefaultConfig writes defaultConfigYAML verbatim, so both loads
	// parse the same bytes by construction, and a corrupted dump still passes
	// here. TestRunConfigInitWritesDefaultConfig is the byte-exact check.
	opts.ConfigFile = filepath.Join(dataDir, DefaultConfigFile)

	gotCfg, err := loadConfig()
	require.NoError(t, err)
	require.Equal(t, wantCfg, gotCfg)
}

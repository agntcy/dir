// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/internal/fsutil"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// configFilePerm is the permission mode used for dumped config files. It
// matches the mode the daemon uses when writing other sensitive files under
// the data directory (e.g. the generated peer identity key).
const configFilePerm = 0o600

// configDirPerm is the permission mode used when creating the parent directory
// for a dumped config file.
//
// 0700, matching the mode `daemon start` uses for the data directory. This
// command is meant to be run BEFORE the daemon has ever started, so it is
// usually what creates ~/.agntcy/dir. MkdirAll is a no-op on an existing
// directory, so a looser mode set here would never be repaired by a later
// `daemon start`, and the daemon goes on to write its peer identity key,
// database, and object store beneath it.
const configDirPerm = 0o700

// errConfigExists is returned by writeIfAbsent when the target path already
// exists.
var errConfigExists = errors.New("config file already exists")

// configCmd is the parent command for daemon config subcommands.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the daemon configuration file",
}

// configInitOptions holds the flags for `dirctl daemon config init`. The
// package-level Options/opts pair is already taken by the daemon command
// itself, so this one carries the subcommand's name.
type configInitOptions struct {
	// Output overrides the destination path. Empty means the daemon's
	// resolved config path.
	Output string
	// Force allows overwriting an existing file.
	Force bool
}

var configInitOpts = &configInitOptions{}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write the default daemon config to disk",
	Long: `Write the daemon's embedded default configuration to disk so it can be
customized and passed back in via --config.

Without --output, the file is written to the daemon's config path
(<data-dir>/` + DefaultConfigFile + ` by default, or the path passed via
--config on the daemon command). It refuses to overwrite an existing file
unless --force is given. The directory is created 0700 and the file 0600,
matching what the daemon expects of its data directory.

Usage examples:

1. Write the default config to the default location:

	dirctl daemon config init

2. Write the default config to a specific file:

	dirctl daemon config init --output /path/to/daemon.config.yaml

3. Overwrite an existing config file:

	dirctl daemon config init --force
`,
	RunE: runConfigInit,
}

func init() {
	configInitCmd.Flags().StringVarP(&configInitOpts.Output, "output", "o", "",
		"Path to write the default config to (default: "+DefaultConfigFile+" resolved under --data-dir, or --config if set)")
	configInitCmd.Flags().BoolVar(&configInitOpts.Force, "force", false, "Overwrite the file if it already exists")

	configCmd.AddCommand(configInitCmd)
}

func runConfigInit(cmd *cobra.Command, _ []string) error {
	target := opts.ConfigFilePath()
	if configInitOpts.Output != "" {
		target = configInitOpts.Output
	}

	if err := writeDefaultConfig(target, configInitOpts.Force); err != nil {
		return err
	}

	presenter.Printf(cmd, "Wrote default daemon config to %s\n", target)

	return nil
}

// writeDefaultConfig writes the embedded default config to target. With force,
// it replaces any existing file; otherwise it refuses to clobber one.
func writeDefaultConfig(target string, force bool) error {
	if force {
		// Create the parent at configDirPerm first. fsutil.WriteAtomic creates
		// missing parents at its own 0755, which would leave the daemon's data
		// directory world-traversable when --force is what creates it. MkdirAll
		// is a no-op if the directory already exists, so this never loosens an
		// existing one either.
		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, configDirPerm); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		if err := fsutil.WriteAtomic(target, []byte(defaultConfigYAML), configFilePerm); err != nil {
			return fmt.Errorf("failed to write default config to %s: %w", target, err)
		}

		return nil
	}

	err := writeIfAbsent(target, []byte(defaultConfigYAML), configFilePerm)
	if errors.Is(err, errConfigExists) {
		// A directory at the target also surfaces as "exists", but --force
		// cannot overwrite one either, so pointing the user at it would send
		// them down a dead end.
		if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
			return fmt.Errorf("%s is a directory, not a config file", target)
		}

		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", target)
	}

	if err != nil {
		return fmt.Errorf("failed to write default config to %s: %w", target, err)
	}

	return nil
}

// writeIfAbsent creates path with data, failing with errConfigExists if a file
// is already there.
//
// O_CREATE|O_EXCL makes the existence check and the create a single syscall,
// so nothing can slip in between them the way it could with a Stat followed by
// a write. It needs no hardlink or rename support, which matters because
// --output can point anywhere: an earlier revision staged a temp file and used
// os.Link to move it into place, which fails outright on exFAT, on SMB shares,
// and in FUSE bind mounts, and failed there only on the default path while
// --force kept working.
//
// This writes in place rather than staging, so a crash mid-write can leave a
// truncated file. That is the accepted trade: the file is new by definition,
// there is no previous version to lose, and a truncated YAML config fails
// loudly when the daemon reads it. On any error after creation the partial
// file is removed.
func writeIfAbsent(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errConfigExists
		}

		return fmt.Errorf("create %s: %w", path, err)
	}

	if _, writeErr := f.Write(data); writeErr != nil {
		_ = f.Close()
		_ = os.Remove(path)

		return fmt.Errorf("write %s: %w", path, writeErr)
	}

	if closeErr := f.Close(); closeErr != nil {
		_ = os.Remove(path)

		return fmt.Errorf("close %s: %w", path, closeErr)
	}

	// O_CREATE applies the umask, which can only clear bits, so the file is
	// never looser than perm. This restores the intended mode when a
	// restrictive umask (e.g. 0200) stripped the owner-write bit.
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}

	return nil
}

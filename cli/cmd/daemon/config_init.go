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

// configDirPerm is the permission mode used when creating the parent
// directory for a dumped config file.
const configDirPerm = 0o755

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
--config on the daemon command). The write is atomic and refuses to
overwrite an existing file unless --force is given.

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
// it overwrites atomically; otherwise it refuses to clobber an existing file.
func writeDefaultConfig(target string, force bool) error {
	if force {
		if err := fsutil.WriteAtomic(target, []byte(defaultConfigYAML), configFilePerm); err != nil {
			return fmt.Errorf("failed to write default config to %s: %w", target, err)
		}

		return nil
	}

	err := writeIfAbsent(target, []byte(defaultConfigYAML), configFilePerm)
	if errors.Is(err, errConfigExists) {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", target)
	}

	if err != nil {
		return fmt.Errorf("failed to write default config to %s: %w", target, err)
	}

	return nil
}

// writeIfAbsent atomically creates path with data, failing with
// errConfigExists if a file is already there. It stages data in a temp file
// in the same directory first (like fsutil.WriteAtomic), then links the temp
// file to path instead of renaming over it: os.Link fails with fs.ErrExist
// if path already exists, so the existence check and the create happen as a
// single filesystem operation. A separate os.Stat followed by a write would
// leave a window where a file created between the two steps gets silently
// clobbered.
func writeIfAbsent(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, configDirPerm); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	tmpName := tmp.Name()

	// Best-effort cleanup of the temp name. Once Link succeeds below, this
	// only removes the extra directory entry - the target inode remains.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("write temp file %s: %w", tmpName, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpName, err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp file %s: %w", tmpName, err)
	}

	if err := os.Link(tmpName, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errConfigExists
		}

		return fmt.Errorf("link %s to %s: %w", tmpName, path, err)
	}

	return nil
}

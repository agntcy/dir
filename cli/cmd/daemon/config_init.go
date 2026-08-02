// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"errors"
	"fmt"
	"os"

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

// userPathDirPerm is used for parents of a path the caller named via --output.
// Those are ordinary locations like /etc/agntcy or a project directory, not
// the daemon's private data directory, and creating them 0700 would lock out
// the account that has to read the file.
const userPathDirPerm = 0o755

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

	// Whether the caller chose the path decides the parent directory mode
	// below, so track it rather than re-deriving it from the path.
	callerChosePath := configInitOpts.Output != ""
	if callerChosePath {
		target = configInitOpts.Output
	}

	if err := writeDefaultConfig(target, configInitOpts.Force, callerChosePath); err != nil {
		return err
	}

	presenter.Printf(cmd, "Wrote default daemon config to %s\n", target)

	return nil
}

// dirPermFor returns the mode to create the target's parent with.
//
// 0700 is right for the daemon's own data directory, which this command
// usually creates and which later holds the peer identity key, the database,
// and the object store. It is wrong for a path the caller named: creating
// /etc/agntcy or /srv/app 0700 on the way to an --output file would lock out
// the service account that has to read the config, and MkdirAll applies the
// mode to every intermediate directory it creates.
func dirPermFor(callerChosePath bool) os.FileMode {
	if callerChosePath {
		return userPathDirPerm
	}

	return configDirPerm
}

// writeDefaultConfig writes the embedded default config to target. With force,
// it replaces any existing file; otherwise it refuses to clobber one.
//
// Everything mechanical — creating parents at the right mode, applying the file
// mode through the descriptor, removing a partial file — lives in fsutil and is
// tested there against both writers. What stays here is the policy (which mode
// the parent gets, which writer --force selects) and the messaging.
func writeDefaultConfig(target string, force, callerChosePath bool) error {
	// Check for a directory before choosing a writer. Both writers fail on a
	// directory target, so telling the user to retry with --force would send
	// them down a dead end, and the atomic path's own error leaks the staging
	// temp file's name.
	if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
		return fmt.Errorf("%s is a directory, not a config file", target)
	}

	wo := fsutil.WriteOptions{FileMode: configFilePerm, DirMode: dirPermFor(callerChosePath)}

	write := fsutil.WriteNew
	if force {
		write = fsutil.WriteAtomic
	}

	if err := write(target, []byte(defaultConfigYAML), wo); err != nil {
		if errors.Is(err, fsutil.ErrExists) {
			return fmt.Errorf("config file already exists at %s (use --force to overwrite)", target)
		}

		return fmt.Errorf("failed to write default config to %s: %w", target, err)
	}

	return nil
}

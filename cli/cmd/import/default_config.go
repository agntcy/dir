// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/internal/fsutil"
)

const (
	// DefaultConfigFile is the default import config filename, stored under the
	// same ~/.agntcy/dir directory as the daemon config.
	DefaultConfigFile = "import.config.yaml"

	configFilePerm = 0o600
	configDirPerm  = 0o700
)

//go:embed import.config.yaml
var defaultConfigYAML string

// defaultConfigPath returns <home>/.agntcy/dir/import.config.yaml.
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".agntcy", "dir", DefaultConfigFile)
	}

	return filepath.Join(home, ".agntcy", "dir", DefaultConfigFile)
}

// ensureDefaultConfig writes the default import config to path when it does not
// exist yet, leaving an existing file untouched.
func ensureDefaultConfig(path string) error {
	err := fsutil.WriteNew(path, []byte(defaultConfigYAML), fsutil.WriteOptions{
		FileMode: configFilePerm,
		DirMode:  configDirPerm,
	})
	if err != nil && !errors.Is(err, fsutil.ErrExists) {
		return fmt.Errorf("failed to write default import config to %s: %w", path, err)
	}

	return nil
}

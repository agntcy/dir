// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// manifestName is the oasf-sdk manifest file written under the asset dir by
// Provision. Its presence is a cheap provisioned/not-provisioned signal; this
// is the one place that couples to the SDK's on-disk layout, and it is used for
// UX only, never for correctness.
const manifestName = "manifest.json"

// IsProvisioned reports whether the asset dir already holds provisioned assets,
// detected by the presence of the SDK manifest.
func IsProvisioned(cfg Config) bool {
	cfg = cfg.Resolve()

	info, err := os.Stat(filepath.Join(cfg.AssetDir, manifestName))

	return err == nil && !info.IsDir()
}

// Teardown removes the provisioned asset dir. It refuses to remove an empty
// path, the filesystem root, or the user's home directory, so a misconfigured
// asset dir can never wipe unrelated files. Removing an absent dir is a no-op.
func Teardown(cfg Config) error {
	// Guard the original value to catch misconfiguration before resolving defaults
	if err := guardAssetDir(strings.TrimSpace(cfg.AssetDir)); err != nil {
		return err
	}

	cfg = cfg.Resolve()

	dir := strings.TrimSpace(cfg.AssetDir)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove asset dir %s: %w", dir, err)
	}

	return nil
}

// guardAssetDir rejects paths that must never be recursively removed.
func guardAssetDir(dir string) error {
	if dir == "" {
		return errors.New("refusing to remove empty asset dir")
	}

	// Require an absolute path: a relative dir like ".." or "../x" would make
	// RemoveAll delete outside the intended asset directory.
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("refusing to remove non-absolute asset dir %q; an absolute path is required", dir)
	}

	clean := filepath.Clean(dir)
	if clean == string(filepath.Separator) {
		return fmt.Errorf("refusing to remove filesystem root %q", dir)
	}

	if home, err := os.UserHomeDir(); err == nil && clean == filepath.Clean(home) {
		return fmt.Errorf("refusing to remove home directory %q as asset dir", dir)
	}

	return nil
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package fsutil provides filesystem helpers for `dirctl`: atomic
// writes that never leave partial files behind.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

const dirPerm = 0o755

// WriteAtomic writes data to path atomically: it creates parent directories,
// writes to a temp file in the same directory, then renames it over the target.
// A rename within a directory is atomic on POSIX filesystems, so readers never
// observe a partially written file and no .bak file is created.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	tmpName := tmp.Name()

	// Best-effort cleanup if we bail out before the rename succeeds.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("write temp file %s: %w", tmpName, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpName, err)
	}

	// The target path is a known agent config location resolved by the integrate
	// descriptors, not untrusted input, so writing to it is intentional.
	if err := os.Chmod(tmpName, perm); err != nil { //nolint:gosec // G703: path is a resolved agent config location
		return fmt.Errorf("chmod temp file %s: %w", tmpName, err)
	}

	if err := os.Rename(tmpName, path); err != nil { //nolint:gosec // G703: path is a resolved agent config location
		return fmt.Errorf("rename %s to %s: %w", tmpName, path, err)
	}

	return nil
}

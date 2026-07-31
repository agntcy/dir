// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package fsutil

import (
	"syscall"
	"testing"
)

// setUmask sets the process umask and returns a function restoring the previous
// one. Build-tagged because syscall.Umask does not exist on Windows.
//
// The umask is process-wide, so a test using this cannot run in parallel with
// one that asserts file modes. None of the tests here call t.Parallel(), which
// is deliberate rather than an omission.
func setUmask(t *testing.T, mask int) func() {
	t.Helper()

	old := syscall.Umask(mask)

	return func() { syscall.Umask(old) }
}

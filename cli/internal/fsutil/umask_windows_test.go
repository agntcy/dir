// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package fsutil

import "testing"

// setUmask is a no-op on Windows, which has no umask. The only caller skips
// before reaching it (POSIX mode bits are not meaningful here); this exists so
// the package still compiles for GOOS=windows.
func setUmask(t *testing.T, _ int) func() {
	t.Helper()

	return func() {}
}

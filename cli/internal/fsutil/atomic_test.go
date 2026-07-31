// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const osWindows = "windows"

// writer names one of the two exported writers so the mechanism tests below can
// run over both. Everything they share — creating parents, applying modes,
// leaving nothing behind on failure — is one implementation, so it is asserted
// once here rather than once per caller package.
type writer struct {
	name  string
	write func(string, []byte, WriteOptions) error
}

func writers() []writer {
	return []writer{
		{name: "WriteAtomic", write: WriteAtomic},
		{name: "WriteNew", write: WriteNew},
	}
}

// requirePOSIXPerms skips on Windows, which does not enforce Unix mode bits.
func requirePOSIXPerms(t *testing.T) {
	t.Helper()

	if runtime.GOOS == osWindows {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
}

func TestWritersCreateParentDirs(t *testing.T) {
	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "deep", "file.json")

			require.NoError(t, w.write(path, []byte("hello"), WriteOptions{}))

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(got))
		})
	}
}

func TestWritersApplyFileMode(t *testing.T) {
	requirePOSIXPerms(t)

	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "file.txt")

			require.NoError(t, w.write(path, []byte("x"), WriteOptions{FileMode: 0o640}))

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())
		})
	}
}

// TestWritersApplyFileModeUnderRestrictiveUmask is the reason the writers chmod
// through the descriptor instead of trusting the open mode. O_CREATE and
// CreateTemp both filter the requested mode through the umask, which can only
// clear bits, so a umask of 0200 would otherwise strip owner-write and leave an
// 0400 config the user cannot edit.
func TestWritersApplyFileModeUnderRestrictiveUmask(t *testing.T) {
	requirePOSIXPerms(t)

	restore := setUmask(t, 0o277)
	defer restore()

	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "file.txt")

			require.NoError(t, w.write(path, []byte("x"), WriteOptions{FileMode: 0o600}))

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
				"umask must not survive into the written file")
		})
	}
}

// TestWritersApplyDirMode covers the case that motivated WriteOptions: the
// daemon's data directory has to be created 0700, and a writer that hard-coded
// 0755 for parents would leave it world-traversable.
func TestWritersApplyDirMode(t *testing.T) {
	requirePOSIXPerms(t)

	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			root := t.TempDir()
			parent := filepath.Join(root, "private")

			require.NoError(t, w.write(filepath.Join(parent, "f.txt"), []byte("x"),
				WriteOptions{DirMode: 0o700}))

			info, err := os.Stat(parent)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
		})
	}
}

// TestWritersZeroOptionsUseDefaults pins the documented zero-value behaviour.
// A zero FileMode must not be taken literally as mode 0000.
func TestWritersZeroOptionsUseDefaults(t *testing.T) {
	requirePOSIXPerms(t)

	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			root := t.TempDir()
			parent := filepath.Join(root, "sub")
			path := filepath.Join(parent, "f.txt")

			require.NoError(t, w.write(path, []byte("x"), WriteOptions{}))

			file, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(defaultFileMode), file.Mode().Perm())

			dir, err := os.Stat(parent)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(defaultDirMode), dir.Mode().Perm())
		})
	}
}

// TestWritersLeaveNothingBehindWhenParentCannotBeCreated uses a regular file as
// a path element, so MkdirAll cannot create the directory beneath it. The error
// must surface rather than be swallowed, and nothing may be left on disk.
func TestWritersLeaveNothingBehindWhenParentCannotBeCreated(t *testing.T) {
	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			root := t.TempDir()
			blocker := filepath.Join(root, "not-a-dir")
			require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

			err := w.write(filepath.Join(blocker, "f.txt"), []byte("data"), WriteOptions{})
			require.Error(t, err)

			entries, readErr := os.ReadDir(root)
			require.NoError(t, readErr)
			assert.Len(t, entries, 1, "only the blocking file should remain")
		})
	}
}

// TestWritersLeaveNothingBehindWhenParentIsUnwritable covers the failure that
// happens after the parent exists: the open itself is refused. Neither writer
// may leave a partial or staging file.
func TestWritersLeaveNothingBehindWhenParentIsUnwritable(t *testing.T) {
	requirePOSIXPerms(t)

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permission bits")
	}

	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			parent := filepath.Join(t.TempDir(), "ro")
			require.NoError(t, os.Mkdir(parent, 0o500))

			err := w.write(filepath.Join(parent, "f.txt"), []byte("data"), WriteOptions{})
			require.Error(t, err)

			// 0500 still permits reading the directory, so a leftover staging
			// file would be visible here.
			entries, readErr := os.ReadDir(parent)
			require.NoError(t, readErr)
			assert.Empty(t, entries, "no partial or staging file may survive a failed write")
		})
	}
}

// TestWritersRejectDirectoryTarget documents that neither writer will write
// over a directory. Callers phrase their own message; both must fail.
func TestWritersRejectDirectoryTarget(t *testing.T) {
	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "adir")
			require.NoError(t, os.Mkdir(target, 0o755))

			require.Error(t, w.write(target, []byte("data"), WriteOptions{}))
		})
	}
}

func TestWritersLeaveNoTempFilesOnSuccess(t *testing.T) {
	for _, w := range writers() {
		t.Run(w.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "file.txt")

			require.NoError(t, w.write(path, []byte("data"), WriteOptions{}))

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			assert.Equal(t, "file.txt", entries[0].Name())
		})
	}
}

// ── behaviour that differs between the two writers ─────────────────────────

func TestWriteAtomicOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")

	require.NoError(t, WriteAtomic(path, []byte("first"), WriteOptions{}))
	require.NoError(t, WriteAtomic(path, []byte("second"), WriteOptions{}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "second", string(got))
}

// TestWriteNewRefusesExistingFile pins the contract callers depend on: the
// refusal is matchable with errors.Is rather than by message text, and it does
// not touch what is already there.
func TestWriteNewRefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))

	err := WriteNew(path, []byte("replacement"), WriteOptions{})
	require.ErrorIs(t, err, ErrExists)

	got, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, "original", string(got))
}

// TestWriteAtomicReplacesContentNotJustAppends guards the rename: a writer that
// opened the target and wrote without truncating would leave a tail of the old
// content behind when the new payload is shorter.
func TestWriteAtomicReplacesContentNotJustAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")

	require.NoError(t, WriteAtomic(path, []byte("a much longer original payload"), WriteOptions{}))
	require.NoError(t, WriteAtomic(path, []byte("short"), WriteOptions{}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "short", string(got))
}

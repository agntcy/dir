// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package fsutil provides filesystem helpers for `dirctl`: writes that apply a
// caller-chosen mode and never leave a partial file behind.
//
// Two writers, differing only in what they do about an existing target:
//
//	WriteAtomic  replaces it, via a temp file and a rename
//	WriteNew     refuses, returning ErrExists
//
// Both take the same WriteOptions and share one implementation, so directory
// creation, mode handling, and partial-file cleanup behave identically and are
// tested once rather than once per caller.
package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Defaults applied when the corresponding WriteOptions field is zero.
const (
	defaultFileMode = 0o600
	defaultDirMode  = 0o755
)

// ErrExists is returned by WriteNew when the target already exists. Callers
// match it with errors.Is to phrase their own message; the writers deliberately
// do not know what the file is for.
var ErrExists = errors.New("file already exists")

// WriteOptions controls the modes the writers apply.
//
// A zero FileMode means 0600 and a zero DirMode means 0755, so a caller that
// only cares about one can set one. The defaults are the conservative choice:
// these helpers write config and credential-adjacent files.
//
// DirMode applies to every directory MkdirAll has to create, which is why it is
// worth passing explicitly when the parent is the daemon's private data
// directory (0700) rather than a location the user named (0755).
type WriteOptions struct {
	FileMode os.FileMode
	DirMode  os.FileMode
}

func (o WriteOptions) fileMode() os.FileMode {
	if o.FileMode == 0 {
		return defaultFileMode
	}

	return o.FileMode
}

func (o WriteOptions) dirMode() os.FileMode {
	if o.DirMode == 0 {
		return defaultDirMode
	}

	return o.DirMode
}

// WriteAtomic replaces path's contents atomically: it writes to a temp file in
// the same directory, then renames it over the target. A rename within a
// directory is atomic on POSIX filesystems, so readers never observe a
// partially written file and no .bak is left behind.
//
// Missing parent directories are created with o.DirMode.
func WriteAtomic(path string, data []byte, o WriteOptions) error {
	return write(path, data, o, true)
}

// WriteNew creates path, returning ErrExists if something is already there.
//
// O_CREATE|O_EXCL makes the existence check and the create one syscall, so
// nothing can slip in between them the way it could with a Stat followed by a
// write. It also needs no hardlink support, which matters because the target
// can be any path the user names: os.Link is unimplemented on some FUSE mounts.
//
// Unlike WriteAtomic this writes in place, so a machine crash mid-write can
// leave a truncated file. A process crash cannot: callers pass a few KB, which
// goes out in a single write. Anything that fails after creation removes the
// partial file. Use WriteAtomic when the target may already exist and readers
// could be looking at it.
func WriteNew(path string, data []byte, o WriteOptions) error {
	return write(path, data, o, false)
}

// write is the shared implementation. `atomic` selects staging-then-rename over
// writing straight at the target; everything else — creating parents, applying
// the mode through the descriptor, and cleaning up a partial file — is common,
// which is the point of having one function.
func write(path string, data []byte, o WriteOptions, atomic bool) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, o.dirMode()); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	f, err := open(path, dir, o.fileMode(), atomic)
	if err != nil {
		return err
	}

	// Where the bytes actually land. For the atomic path this is the staging
	// file, which is also what has to be removed if anything below fails.
	name := f.Name()

	// Chmod through the descriptor, not the path. O_CREATE and CreateTemp both
	// apply the umask, which can only clear bits, so the file is never looser
	// than requested; this restores the intended mode when a restrictive umask
	// (e.g. 0200) stripped the owner-write bit. Going through f rather than
	// os.Chmod(name) means it cannot follow a symlink or land on a file someone
	// swapped in, and it happens before any content is written.
	if err := f.Chmod(o.fileMode()); err != nil {
		return cleanup(f, name, fmt.Errorf("chmod %s: %w", name, err))
	}

	if _, err := f.Write(data); err != nil {
		return cleanup(f, name, fmt.Errorf("write %s: %w", name, err))
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(name)

		return fmt.Errorf("close %s: %w", name, err)
	}

	if atomic {
		// The destination is caller-chosen by design: agentcfg resolves paths
		// from the integrate descriptors, and `daemon config init` passes
		// --output straight through.
		if err := os.Rename(name, path); err != nil { //nolint:gosec // G703: destination is caller-chosen by design
			_ = os.Remove(name)

			return fmt.Errorf("rename %s to %s: %w", name, path, err)
		}
	}

	return nil
}

// open creates the file the bytes go into: a staging file beside the target for
// the atomic path, or the target itself under O_EXCL for the exclusive one.
func open(path, dir string, mode os.FileMode, atomic bool) (*os.File, error) {
	if atomic {
		f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
		if err != nil {
			return nil, fmt.Errorf("create temp file in %s: %w", dir, err)
		}

		return f, nil
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrExists
		}

		return nil, fmt.Errorf("create %s: %w", path, err)
	}

	return f, nil
}

// cleanup closes and removes a partially written file, returning the original
// error. Neither close nor remove can tell the caller anything more useful than
// what already went wrong, so both are best-effort.
func cleanup(f *os.File, name string, err error) error {
	_ = f.Close()
	_ = os.Remove(name)

	return err
}

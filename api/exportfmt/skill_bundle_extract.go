// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const skillManifestFile = "SKILL.md"

// SkillMarkdownFromArchive returns the SKILL.md content from a skill artifact.
// Plain-text archives are returned as-is; gzip tar bundles must contain SKILL.md.
func SkillMarkdownFromArchive(archive []byte) (string, error) {
	if len(archive) == 0 {
		return "", fmt.Errorf("skill bundle archive is empty")
	}

	if !isGzipArchive(archive) {
		return string(archive), nil
	}

	iterator, err := NewTarIterator(archive, WithTypeflag(tar.TypeReg))
	if err != nil {
		return "", fmt.Errorf("invalid gzip archive: %w", err)
	}

	for entry, err := range iterator {
		if err != nil {
			return "", fmt.Errorf("read tar entry: %w", err)
		}

		rel, err := localTarEntryPath(entry.header.Name)
		if err != nil {
			return "", fmt.Errorf("invalid tar entry %q: %w", entry.header.Name, err)
		}

		if rel != skillManifestFile {
			continue
		}

		return string(entry.payload), nil
	}

	return "", fmt.Errorf("archive does not contain %q", skillManifestFile)
}

// SkillBundleMatchesDir reports whether dir already contains the same files as archive.
func SkillBundleMatchesDir(archive []byte, dir string) (bool, error) {
	if len(archive) == 0 {
		return false, fmt.Errorf("skill bundle archive is empty")
	}

	if !isGzipArchive(archive) {
		return matchesSkillManifest(dir, archive)
	}

	return matchesGzip(dir, archive)
}

func matchesSkillManifest(dir string, b []byte) (bool, error) {
	content, err := os.ReadFile(filepath.Join(dir, skillManifestFile))

	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("read %s: %w", skillManifestFile, err)
	}

	return bytes.Equal(content, b), nil
}

func matchesGzip(dir string, b []byte) (bool, error) {
	iterator, err := NewTarIterator(b, WithTypeflag(tar.TypeReg))
	if err != nil {
		return false, fmt.Errorf("invalid gzip archive: %w", err)
	}

	paths := make(map[string]bool)

	for entry, err := range iterator {
		if err != nil {
			return false, fmt.Errorf("read tar entry: %w", err)
		}

		path, err := localTarEntryPath(entry.header.Name)
		if err != nil {
			return false, fmt.Errorf("invalid tar entry %q: %w", entry.header.Name, err)
		}

		match, err := matchesTarEntry(dir, entry)
		if err != nil {
			return false, err
		}

		if !match {
			return false, nil
		}

		paths[path] = true
	}

	if len(paths) == 0 {
		return false, fmt.Errorf("archive contains no regular files")
	}

	match, err := matchesPaths(dir, paths)
	if err != nil {
		return false, err
	}

	if !match {
		return false, nil
	}

	return true, nil
}

func matchesPaths(dir string, paths map[string]bool) (bool, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("stat %s: %w", dir, err)
	}

	result := true

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("rel path for %q: %w", path, err)
		}

		if _, ok := paths[rel]; !ok {
			result = false

			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("walk %s: %w", dir, err)
	}

	return result, nil
}

func matchesTarEntry(dir string, entry *TarEntry) (bool, error) {
	path, err := localTarEntryPath(entry.header.Name)
	if err != nil {
		return false, fmt.Errorf("invalid tar entry %q: %w", entry.header.Name, err)
	}

	content, err := os.ReadFile(filepath.Join(dir, path))
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("read %q: %w", path, err)
	}

	return bytes.Equal(content, entry.payload), nil
}

// ExtractSkillBundleArchive extracts a skill artifact into destDir.
// The artifact is either a gzip-compressed tar bundle (full skill with code samples)
// or a plain-text SKILL.md manifest. Both formats are handled transparently.
func ExtractSkillBundleArchive(archive []byte, destDir string) error {
	if len(archive) == 0 {
		return fmt.Errorf("skill bundle archive is empty")
	}

	parent := filepath.Dir(destDir)
	if err := os.MkdirAll(parent, 0o755); err != nil { //nolint:mnd
		return fmt.Errorf("create parent directory: %w", err)
	}

	tmpDir, err := os.MkdirTemp(parent, filepath.Base(destDir)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}

	installed := false

	defer func() {
		if !installed {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	if err := writeSkillBundleContents(archive, tmpDir); err != nil {
		return err
	}

	if err := replaceSkillBundleDir(tmpDir, destDir); err != nil {
		return err
	}

	installed = true

	return nil
}

func writeSkillBundleContents(archive []byte, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil { //nolint:mnd
		return fmt.Errorf("create destination directory: %w", err)
	}

	if !isGzipArchive(archive) {
		return os.WriteFile(filepath.Join(destDir, skillManifestFile), archive, 0o600) //nolint:mnd,wrapcheck
	}

	root, err := os.OpenRoot(destDir)
	if err != nil {
		return fmt.Errorf("open destination directory: %w", err)
	}
	defer root.Close()

	iterator, err := NewTarIterator(archive)
	if err != nil {
		return fmt.Errorf("invalid gzip archive: %w", err)
	}

	for entry, err := range iterator {
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		if err := extractTarEntry(root, entry); err != nil {
			return err
		}
	}

	return nil
}

func replaceSkillBundleDir(tmpDir, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("remove existing skill directory: %w", err)
	}

	if err := os.Rename(tmpDir, destDir); err != nil {
		return fmt.Errorf("install skill directory: %w", err)
	}

	return nil
}

func extractTarEntry(root *os.Root, entry *TarEntry) error {
	header := entry.header

	rel, err := localTarEntryPath(header.Name)
	if err != nil {
		return fmt.Errorf("invalid tar entry %q: %w", header.Name, err)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		if err := root.MkdirAll(rel, dirPerm(header)); err != nil {
			return fmt.Errorf("create directory %q: %w", header.Name, err)
		}

		return nil
	case tar.TypeReg:
		if parent := filepath.Dir(rel); parent != "." {
			if err := root.MkdirAll(parent, 0o755); err != nil { //nolint:mnd
				return fmt.Errorf("create parent directory for %q: %w", header.Name, err)
			}
		}

		file, err := root.OpenFile(rel, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm(header))
		if err != nil {
			return fmt.Errorf("create file %q: %w", header.Name, err)
		}

		if _, err := file.Write(entry.payload); err != nil {
			_ = file.Close()

			return fmt.Errorf("write file %q: %w", header.Name, err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("close file %q: %w", header.Name, err)
		}

		return nil
	default:
		return nil
	}
}

func localTarEntryPath(name string) (string, error) {
	rel := filepath.FromSlash(name)
	if rel == "" {
		return "", fmt.Errorf("empty tar path")
	}

	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("non-local tar path")
	}

	return rel, nil
}

func isGzipArchive(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}

func dirPerm(hdr *tar.Header) os.FileMode {
	return tarEntryPerm(hdr.Mode, 0o755) //nolint:mnd
}

func filePerm(hdr *tar.Header) os.FileMode {
	return tarEntryPerm(hdr.Mode, 0o600) //nolint:mnd
}

func tarEntryPerm(mode int64, defaultPerm os.FileMode) os.FileMode {
	if mode == 0 {
		return defaultPerm
	}

	perm := mode & 0o777          //nolint:mnd
	if perm < 0 || perm > 0o777 { //nolint:mnd
		return defaultPerm
	}

	return os.FileMode(uint32(perm))
}

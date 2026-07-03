// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const skillManifestFile = "SKILL.md"

// ExtractSkillBundleArchive extracts a skill artifact into destDir.
// The artifact is either a gzip-compressed tar bundle (full skill with code samples)
// or a plain-text SKILL.md manifest. Both formats are handled transparently.
func ExtractSkillBundleArchive(archive []byte, destDir string) error {
	if len(archive) == 0 {
		return fmt.Errorf("skill bundle archive is empty")
	}

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

	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		if err := extractTarEntry(tarReader, header, root); err != nil {
			return err
		}
	}

	return nil
}

func extractTarEntry(tarReader *tar.Reader, header *tar.Header, root *os.Root) error {
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

		reader := io.Reader(tarReader)
		if header.Size >= 0 {
			reader = io.LimitReader(tarReader, header.Size)
		}

		if _, err := io.Copy(file, reader); err != nil {
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

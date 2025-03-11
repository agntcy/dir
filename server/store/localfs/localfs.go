// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

// nolint:mnd
package localfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/spf13/afero"
)

type store struct {
	metaFs afero.Fs
	dataFs afero.Fs
}

func New(baseDir string) (types.StoreService, error) {
	dataDir := filepath.Join(baseDir, "contents")
	if err := DefaultFs.MkdirAll(dataDir, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dataTmpDir := filepath.Join(baseDir, "contents", "tmp")
	if err := DefaultFs.MkdirAll(dataTmpDir, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create data tmp directory: %w", err)
	}

	metaDir := filepath.Join(baseDir, "metadata")
	if err := DefaultFs.MkdirAll(metaDir, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &store{
		metaFs: afero.NewBasePathFs(DefaultFs, metaDir),
		dataFs: afero.NewBasePathFs(DefaultFs, dataDir),
	}, nil
}

func (c *store) Push(ctx context.Context, meta *coretypes.ObjectMeta, contents io.Reader) (*coretypes.Digest, error) {
	// TODO: this can be done natively without temp file and renames
	//  using a background writer
	// TODO: chunking does not work properly, it needs to be moved to a util
	// package as we will need to reuse it across providers
	// create temp file that will hold the contents
	contentsFile, err := afero.TempFile(c.dataFs, "tmp", "*")
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer contentsFile.Close()

	// Process contents by reading in chunks, writing them to a file,
	// and obtain final content hash
	hasher := sha256.New()
	// byte array is allocated on the stack and reused
	var chunk [4096]byte
loop:
	for {
		// Check context
		select {
		case <-ctx.Done():
			return &coretypes.Digest{}, errors.New("context expired")
		default:
			n, err := contents.Read(chunk[:])
			if errors.Is(err, io.EOF) {
				break loop
			}
			if err != nil {
				return &coretypes.Digest{}, fmt.Errorf("failed to read contents: %w", err)
			}

			// Write contents to the file (only the bytes that were read)
			_, err = contentsFile.Write(chunk[:n])
			if err != nil {
				return &coretypes.Digest{}, fmt.Errorf("failed to write contents: %w", err)
			}

			// Update hash (only with the bytes that were read)
			_, err = hasher.Write(chunk[:n])
			if err != nil {
				return &coretypes.Digest{}, fmt.Errorf("failed to calculate hash: %w", err)
			}
		}
	}

	// Get content dig
	dig := coretypes.Digest{
		Type:  coretypes.DigestType_DIGEST_TYPE_SHA256,
		Value: hex.EncodeToString(hasher.Sum(nil)),
	}

	// Rename contents file to the digest hash
	_ = contentsFile.Close()

	if err := c.dataFs.Rename(contentsFile.Name(), dig.Encode()); err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to rename file: %w", err)
	}

	// Convert metadata to raw
	metadataRaw, err := json.Marshal(meta)
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to process metadata: %w", err)
	}

	// Store raw metadata
	err = afero.WriteReader(c.metaFs, dig.Encode(), bytes.NewReader(metadataRaw))
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to store metadata: %w", err)
	}

	return &dig, nil
}

func (c *store) Lookup(_ context.Context, dig *coretypes.Digest) (*coretypes.ObjectMeta, error) {
	// Read metadata file from FS
	metadataRaw, err := afero.ReadFile(c.metaFs, dig.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata: %w", err)
	}

	// Convert to object
	var metadata coretypes.ObjectMeta
	if err = json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to process metadata: %w", err)
	}

	return &metadata, nil
}

func (c *store) Pull(_ context.Context, dig *coretypes.Digest) (io.Reader, error) {
	// Open the file for reading
	file, err := c.dataFs.Open(dig.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to open contents: %w", err)
	}

	// Return the file as an io.Reader
	return file, nil
}

func (c *store) Delete(_ context.Context, dig *coretypes.Digest) error {
	err := c.dataFs.Remove(dig.Encode())
	if err != nil {
		return fmt.Errorf("failed to remove contents: %w", err)
	}

	err = c.metaFs.Remove(dig.Encode())
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

// nolint:mnd
package localfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	fsconfig "github.com/agntcy/dir/server/store/localfs/config"
	"github.com/agntcy/dir/server/types"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/afero"
)

type store struct {
	metaFs afero.Fs
	dataFs afero.Fs
}

func New(cfg fsconfig.Config) (types.StoreAPI, error) {
	parentFs := afero.NewOsFs()

	dataDir := filepath.Join(cfg.Dir, "contents")
	if err := parentFs.MkdirAll(dataDir, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	metaDir := filepath.Join(cfg.Dir, "metadata")
	if err := parentFs.MkdirAll(metaDir, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &store{
		metaFs: afero.NewBasePathFs(parentFs, metaDir),
		dataFs: afero.NewBasePathFs(parentFs, dataDir),
	}, nil
}

func (c *store) Push(ctx context.Context, ref *coretypes.ObjectRef, contents io.Reader) (*coretypes.ObjectRef, error) {
	// TODO: Chunking read/write needs to be moved to a util
	// package as we will need to reuse it across providers
	contentsFile, err := afero.TempFile(c.dataFs, ".", "*")
	if err != nil {
		return &coretypes.ObjectRef{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer contentsFile.Close()

	// Read contents in chunks and write to a local temp file.
	// Byte array is allocated on the stack and reused.
	// Calculate contents hash on the fly.
	var chunk [4096]byte

	var size uint64

	hasher := sha256.New()

loop:
	for {
		// Check context
		select {
		case <-ctx.Done():
			return &coretypes.ObjectRef{}, errors.New("context expired")
		default:
			n, err := contents.Read(chunk[:])
			if errors.Is(err, io.EOF) {
				break loop
			}
			if err != nil {
				return &coretypes.ObjectRef{}, fmt.Errorf("failed to read contents: %w", err)
			}

			// Write contents to the file (only the bytes that were read)
			_, err = contentsFile.Write(chunk[:n])
			if err != nil {
				return &coretypes.ObjectRef{}, fmt.Errorf("failed to write contents: %w", err)
			}
			size += uint64(n) //nolint:gosec // We are not dealing with sensitive data here.

			// Update hash (only with the bytes that were read)
			_, err = hasher.Write(chunk[:n])
			if err != nil {
				return &coretypes.ObjectRef{}, fmt.Errorf("failed to calculate hash: %w", err)
			}
		}
	}

	_ = contentsFile.Close()

	// Calculate content digest
	digest := digest.NewDigestFromBytes(digest.SHA256, hasher.Sum(nil)).String()

	// Store contents
	if err := c.dataFs.Rename(contentsFile.Name(), digest); err != nil {
		return &coretypes.ObjectRef{}, fmt.Errorf("failed to rename file: %w", err)
	}

	// Update metadata
	metadataRef := &coretypes.ObjectRef{
		Digest:      digest,
		Size:        size,
		Type:        ref.GetType(),
		Annotations: ref.GetAnnotations(),
	}

	metadataRaw, err := json.Marshal(metadataRef)
	if err != nil {
		return &coretypes.ObjectRef{}, fmt.Errorf("failed to process metadata: %w", err)
	}

	// Store metadata
	err = afero.WriteReader(c.metaFs, digest, bytes.NewReader(metadataRaw))
	if err != nil {
		return &coretypes.ObjectRef{}, fmt.Errorf("failed to store metadata: %w", err)
	}

	return metadataRef, nil
}

func (c *store) Lookup(_ context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	// Read metadata
	metadataRaw, err := afero.ReadFile(c.metaFs, ref.GetDigest())
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Process metadata
	var metadata coretypes.ObjectRef
	if err = json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to process metadata: %w", err)
	}

	return &metadata, nil
}

func (c *store) Pull(_ context.Context, ref *coretypes.ObjectRef) (io.ReadCloser, error) {
	file, err := c.dataFs.Open(ref.GetDigest())
	if err != nil {
		return nil, fmt.Errorf("failed to open contents: %w", err)
	}

	return file, nil
}

func (c *store) Delete(_ context.Context, ref *coretypes.ObjectRef) error {
	// TODO: allow delete to be called even when the data is not found.
	// if data removal succeeds but metadata removal fails,
	// we should still be able to remove meta
	err := c.dataFs.Remove(ref.GetDigest())
	if err != nil {
		return fmt.Errorf("failed to remove contents: %w", err)
	}

	err = c.metaFs.Remove(ref.GetDigest())
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

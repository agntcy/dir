// Copyright AGNTCY Contributors (https://github.com/agntcy)
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
	"github.com/agntcy/dir/utils/logging"
	"github.com/opencontainers/go-digest"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.Logger("store/localfs")

type store struct {
	metaFs afero.Fs
	dataFs afero.Fs
}

func New(cfg fsconfig.Config) (types.StoreAPI, error) {
	logger.Debug("Creating localfs store with config", "config", cfg)

	parentFs := afero.NewOsFs()

	dataDir := filepath.Join(cfg.Dir, "contents")
	if err := parentFs.MkdirAll(dataDir, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create contents directory: %w", err)
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
	logger.Debug("Pushing object to LocalFS store", "ref", ref)

	// TODO: Chunking read/write needs to be moved to a util
	// package as we will need to reuse it across providers
	contentsFile, err := afero.TempFile(c.dataFs, ".", "*")
	if err != nil {
		return &coretypes.ObjectRef{}, status.Errorf(codes.FailedPrecondition, "failed to create temp file for contents: %v", err)
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
			return &coretypes.ObjectRef{}, status.Error(codes.Canceled, "context expired") //nolint:wrapcheck
		default:
			n, err := contents.Read(chunk[:])
			if errors.Is(err, io.EOF) {
				logger.Info("Finished reading contents", "ref", ref)

				break loop
			}
			if err != nil {
				return &coretypes.ObjectRef{}, status.Errorf(codes.Internal, "failed to read contents: %v", err)
			}

			// Write contents to the file (only the bytes that were read)
			_, err = contentsFile.Write(chunk[:n])
			if err != nil {
				return &coretypes.ObjectRef{}, status.Errorf(codes.Internal, "failed to write contents: %v", err)
			}
			size += uint64(n) //nolint:gosec // We are not dealing with sensitive data here.

			// Update hash (only with the bytes that were read)
			_, err = hasher.Write(chunk[:n])
			if err != nil {
				return &coretypes.ObjectRef{}, status.Errorf(codes.Internal, "failed to calculate hash: %v", err)
			}
		}
	}

	_ = contentsFile.Close()

	// Calculate content digest
	digest := digest.NewDigestFromBytes(digest.SHA256, hasher.Sum(nil)).String()

	// Store contents
	if err := c.dataFs.Rename(contentsFile.Name(), digest); err != nil {
		if errors.Is(err, afero.ErrFileExists) {
			return &coretypes.ObjectRef{}, status.Errorf(codes.FailedPrecondition, "file already exists: %v", digest)
		}

		return &coretypes.ObjectRef{}, status.Errorf(codes.Internal, "failed to rename file: %v", err)
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
		return &coretypes.ObjectRef{}, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	// Store metadata
	err = afero.WriteReader(c.metaFs, digest, bytes.NewReader(metadataRaw))
	if err != nil {
		if errors.Is(err, afero.ErrFileExists) {
			return &coretypes.ObjectRef{}, status.Errorf(codes.FailedPrecondition, "metadata already exists: %s", digest)
		}

		return &coretypes.ObjectRef{}, status.Errorf(codes.Internal, "failed to store metadata: %v", err)
	}

	return metadataRef, nil
}

func (c *store) Lookup(_ context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	logger.Debug("Looking up object in LocalFS store", "ref", ref)

	// Read metadata
	metadataRaw, err := afero.ReadFile(c.metaFs, ref.GetDigest())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "object not found: %s", ref.GetDigest())
		}

		return nil, status.Errorf(codes.Internal, "failed to read metadata: %v", err)
	}

	// Process metadata
	var metadata coretypes.ObjectRef
	if err = json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal metadata: %v", err)
	}

	return &metadata, nil
}

func (c *store) Pull(_ context.Context, ref *coretypes.ObjectRef) (io.ReadCloser, error) {
	logger.Debug("Pulling object from LocalFS store", "ref", ref)

	file, err := c.dataFs.Open(ref.GetDigest())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "object not found: %s", ref.GetDigest())
		}

		return nil, status.Errorf(codes.Internal, "failed to open contents: %v", err)
	}

	return file, nil
}

func (c *store) Delete(_ context.Context, ref *coretypes.ObjectRef) error {
	logger.Debug("Deleting object from LocalFS store", "ref", ref)

	// TODO: allow delete to be called even when the data is not found.
	// if data removal succeeds but metadata removal fails,
	// we should still be able to remove meta
	err := c.dataFs.Remove(ref.GetDigest())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return status.Errorf(codes.NotFound, "object not found: %s", ref.GetDigest())
		}

		return status.Errorf(codes.Internal, "failed to remove contents: %v", err)
	}

	err = c.metaFs.Remove(ref.GetDigest())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return status.Errorf(codes.NotFound, "metadata not found: %s", ref.GetDigest())
		}

		return status.Errorf(codes.Internal, "failed to remove metadata: %v", err)
	}

	return nil
}

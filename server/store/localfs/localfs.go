// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:mnd
package localfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	fsconfig "github.com/agntcy/dir/server/store/localfs/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/afero"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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

func (c *store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	logger.Debug("Pushing record to LocalFS store", "record", record)

	// Marshal the record to bytes using proto.Marshal
	recordBytes, err := proto.Marshal(record)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	// CID must be set by the controller
	recordCID := record.GetCid()
	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required")
	}

	logger.Debug("Using CID from record", "cid", recordCID)

	// Create temp file for contents
	contentsFile, err := afero.TempFile(c.dataFs, ".", "*")
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to create temp file for contents: %v", err)
	}
	defer contentsFile.Close()

	// Write record bytes to file
	_, err = contentsFile.Write(recordBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to write record contents: %v", err)
	}

	_ = contentsFile.Close()

	// Store contents with CID as filename
	if err := c.dataFs.Rename(contentsFile.Name(), recordCID); err != nil {
		if errors.Is(err, afero.ErrFileExists) {
			return nil, status.Errorf(codes.FailedPrecondition, "file already exists: %v", recordCID)
		}
		return nil, status.Errorf(codes.Internal, "failed to rename file: %v", err)
	}

	// Create metadata
	recordMeta := &corev1.RecordMeta{
		Cid:           recordCID,
		Annotations:   make(map[string]string), // TODO: extract from record if needed
		SchemaVersion: "v1",                    // TODO: determine from record type
		CreatedAt:     "",                      // TODO: set current timestamp
	}

	metadataRaw, err := json.Marshal(recordMeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	// Store metadata
	err = afero.WriteReader(c.metaFs, recordCID, bytes.NewReader(metadataRaw))
	if err != nil {
		if errors.Is(err, afero.ErrFileExists) {
			return nil, status.Errorf(codes.FailedPrecondition, "metadata already exists: %s", recordCID)
		}
		return nil, status.Errorf(codes.Internal, "failed to store metadata: %v", err)
	}

	logger.Info("Record stored successfully", "cid", recordCID)

	return &corev1.RecordRef{Cid: recordCID}, nil
}

func (c *store) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	logger.Debug("Looking up record in LocalFS store", "ref", ref)

	// Read metadata
	metadataRaw, err := afero.ReadFile(c.metaFs, ref.GetCid())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
		}

		return nil, status.Errorf(codes.Internal, "failed to read metadata: %v", err)
	}

	// Process metadata
	var metadata corev1.RecordMeta
	if err = json.Unmarshal(metadataRaw, &metadata); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal metadata: %v", err)
	}

	return &metadata, nil
}

func (c *store) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	logger.Debug("Pulling record from LocalFS store", "ref", ref)

	// Read record data from file
	recordData, err := afero.ReadFile(c.dataFs, ref.GetCid())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
		}

		return nil, status.Errorf(codes.Internal, "failed to read record data: %v", err)
	}

	// Unmarshal data back to Record
	var record corev1.Record
	if err := proto.Unmarshal(recordData, &record); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal record: %v", err)
	}

	logger.Debug("Record pulled successfully", "cid", ref.GetCid())

	return &record, nil
}

func (c *store) Delete(_ context.Context, ref *corev1.RecordRef) error {
	logger.Debug("Deleting record from LocalFS store", "ref", ref)

	// TODO: allow delete to be called even when the data is not found.
	// if data removal succeeds but metadata removal fails,
	// we should still be able to remove meta
	err := c.dataFs.Remove(ref.GetCid())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
		}

		return status.Errorf(codes.Internal, "failed to remove contents: %v", err)
	}

	err = c.metaFs.Remove(ref.GetCid())
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			return status.Errorf(codes.NotFound, "metadata not found: %s", ref.GetCid())
		}

		return status.Errorf(codes.Internal, "failed to remove metadata: %v", err)
	}

	return nil
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	maxAgentSize = 1024 * 1024 * 4 // 4MB
)

var storeLogger = logging.Logger("controller/store")

type storeCtrl struct {
	storetypes.UnimplementedStoreServiceServer
	store types.StoreAPI
}

func NewStoreController(store types.StoreAPI) storetypes.StoreServiceServer {
	return &storeCtrl{
		UnimplementedStoreServiceServer: storetypes.UnimplementedStoreServiceServer{},
		store:                           store,
	}
}

func (s storeCtrl) Push(stream storetypes.StoreService_PushServer) error {
	// TODO: validate
	firstMessage, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive first message: %w", err)
	}

	storeLogger.Debug("Called store contoller's Push method",
		"data", firstMessage.GetData(),
		"agent", firstMessage.GetAgent(),
		"object-ref", firstMessage.GetRef(),
	)

	// lookup (skip if exists)
	ref, err := s.store.Lookup(stream.Context(), firstMessage.GetRef())
	if err == nil {
		storeLogger.Info("Object already exists, skipping push to store", "ref", ref)

		return stream.SendAndClose(ref)
	}

	// read packets into a pipe in the background
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()

		if len(firstMessage.GetData()) > 0 {
			if _, err := pw.Write(firstMessage.GetData()); err != nil {
				storeLogger.Error("Failed to write first message to pipe", "error", err)
				pw.CloseWithError(err)

				return
			}
		}

		for {
			obj, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				storeLogger.Error("Failed to receive object", "error", err)
				pw.CloseWithError(err)

				return
			}

			if _, err := pw.Write(obj.GetData()); err != nil {
				storeLogger.Error("Failed to write object to pipe", "error", err)
				pw.CloseWithError(err)

				return
			}
		}
	}()

	// Read input
	agent := &coretypes.Agent{}
	if _, err := agent.LoadFromReader(pr); err != nil {
		return fmt.Errorf("failed to process agent: %w", err)
	}

	// Convert agent to JSON to drop additional fields
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Validate agent
	// Signature validation
	// This does not validate the signature itself, but only checks if it is set.
	// NOTE: we can still push agents with bogus signatures, but we will not be able to verify them.
	if agent.GetSignature() == nil {
		return errors.New("agent signature is required")
	}

	// Size validation
	if len(agentJSON) > maxAgentSize {
		return fmt.Errorf("agent size exceeds maximum size of %d bytes", maxAgentSize)
	}

	// Push to underlying store
	ref, err = s.store.Push(stream.Context(), firstMessage.GetRef(), bytes.NewReader(agentJSON))
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return stream.SendAndClose(ref)
}

func (s storeCtrl) Pull(req *coretypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	storeLogger.Debug("Called store contoller's Pull method", "req", req)

	// lookup (maybe not needed)
	ref, err := s.store.Lookup(stream.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// pull
	reader, err := s.store.Pull(stream.Context(), ref)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	buf := make([]byte, 4096) //nolint:mnd

	for {
		n, readErr := reader.Read(buf)
		if readErr == io.EOF && n == 0 {
			storeLogger.Debug("Finished reading all chunks")

			// exit as we read all the chunks
			return nil
		}

		if readErr != io.EOF && readErr != nil {
			// return if a non-nil error and stream was not fully read
			return fmt.Errorf("failed to read: %w", err)
		}

		// forward data
		err = stream.Send(&coretypes.Object{
			Data: buf[:n],
		})
		if err != nil {
			return fmt.Errorf("failed to send: %w", err)
		}
	}
}

func (s storeCtrl) Lookup(ctx context.Context, req *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	storeLogger.Debug("Called store contoller's Lookup method", "req", req)

	// validate
	if req.GetDigest() == "" {
		return nil, errors.New("digest is required")
	}

	// TODO: add caching to avoid querying the Storage API

	// lookup
	meta, err := s.store.Lookup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	return meta, nil
}

func (s storeCtrl) Delete(ctx context.Context, req *coretypes.ObjectRef) (*emptypb.Empty, error) {
	storeLogger.Debug("Called store contoller's Delete method", "req", req)

	if req.GetDigest() == "" {
		return nil, errors.New("digest is required")
	}

	err := s.store.Delete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}

	return &emptypb.Empty{}, nil
}

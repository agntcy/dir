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
	"github.com/agntcy/dir/server/search/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ocidigest "github.com/opencontainers/go-digest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	maxAgentSize = 1024 * 1024 * 4 // 4MB
)

var storeLogger = logging.Logger("controller/store")

type storeCtrl struct {
	storetypes.UnimplementedStoreServiceServer
	store  types.StoreAPI
	search types.SearchAPI
}

func NewStoreController(store types.StoreAPI, search types.SearchAPI) storetypes.StoreServiceServer {
	return &storeCtrl{
		UnimplementedStoreServiceServer: storetypes.UnimplementedStoreServiceServer{},
		store:                           store,
		search:                          search,
	}
}

func (s storeCtrl) Push(stream storetypes.StoreService_PushServer) error {
	// TODO: validate
	firstMessage, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive first message: %v", err)
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
		return status.Errorf(codes.Internal, "failed to load agent from reader: %v", err)
	}

	// Convert agent to JSON to drop additional fields
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to marshal agent to JSON: %v", err)
	}

	// Validate agent
	// Signature validation
	// This does not validate the signature itself, but only checks if it is set.
	// NOTE: we can still push agents with bogus signatures, but we will not be able to verify them.
	if agent.GetSignature() == nil {
		return status.Error(codes.InvalidArgument, "agent signature is required")
	}

	// Size validation
	if len(agentJSON) > maxAgentSize {
		return status.Errorf(codes.InvalidArgument, "agent size exceeds maximum size of %d bytes", maxAgentSize)
	}

	// Update the ObjectRef size and digest to match the marshalled JSON data
	updatedRef := &coretypes.ObjectRef{
		Digest:      ocidigest.FromBytes(agentJSON).String(),
		Type:        firstMessage.GetRef().GetType(),
		Size:        uint64(len(agentJSON)),
		Annotations: firstMessage.GetRef().GetAnnotations(),
	}

	// Push to underlying store
	ref, err = s.store.Push(stream.Context(), updatedRef, bytes.NewReader(agentJSON))
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to push object to store: %s", st.Message())
	}

	err = s.search.AddRecord(v1alpha1.NewAgentAdapter(agent, ref.GetDigest()))
	if err != nil {
		return fmt.Errorf("failed to add agent to search index: %w", err)
	}

	return stream.SendAndClose(ref)
}

func (s storeCtrl) Pull(req *coretypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	storeLogger.Debug("Called store contoller's Pull method", "req", req)

	// lookup (maybe not needed)
	ref, err := s.store.Lookup(stream.Context(), req)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to lookup object: %v", st.Message())
	}

	// pull
	reader, err := s.store.Pull(stream.Context(), ref)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to pull object: %v", st.Message())
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
			return status.Errorf(codes.Internal, "failed to read: %v", readErr)
		}

		// forward data
		err = stream.Send(&coretypes.Object{
			Data: buf[:n],
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send data: %v", err)
		}
	}
}

func (s storeCtrl) Lookup(ctx context.Context, req *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	storeLogger.Debug("Called store contoller's Lookup method", "req", req)

	// validate
	if req.GetDigest() == "" {
		return nil, status.Error(codes.InvalidArgument, "digest is required")
	}

	// TODO: add caching to avoid querying the Storage API

	// lookup
	meta, err := s.store.Lookup(ctx, req)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	return meta, nil
}

func (s storeCtrl) Delete(ctx context.Context, req *coretypes.ObjectRef) (*emptypb.Empty, error) {
	storeLogger.Debug("Called store contoller's Delete method", "req", req)

	if req.GetDigest() == "" {
		return nil, status.Error(codes.InvalidArgument, "digest is required")
	}

	err := s.store.Delete(ctx, req)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to delete object: %s", st.Message())
	}

	return &emptypb.Empty{}, nil
}

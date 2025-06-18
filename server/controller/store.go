// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/api/objectmanager"
	"github.com/agntcy/dir/server/search/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxAgentSize = 1024 * 1024 * 4 // 4MB
)

var storeLogger = logging.Logger("controller/store")

type StoreController struct {
	store  types.StoreAPI
	search types.SearchAPI
}

func NewStoreController(store types.StoreAPI, search types.SearchAPI) *StoreController {
	return &StoreController{
		store:  store,
		search: search,
	}
}

// Push handles the business logic for pushing an object to the store
func (s *StoreController) Push(ctx context.Context, ref *objectmanager.RecordObject, dataReader io.Reader) (*objectmanager.RecordObject, error) {
	storeLogger.Debug("Called common store controller's Push method", "ref", ref)

	// lookup (skip if exists)
	existingRef, err := s.store.Lookup(ctx, ref)
	if err == nil {
		storeLogger.Info("Object already exists, skipping push to store", "ref", existingRef)
		return existingRef, nil
	}

	// Read input
	agent := &coretypes.Agent{}
	if _, err := agent.LoadFromReader(dataReader); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load agent from reader: %v", err)
	}

	// Convert agent to JSON to drop additional fields
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal agent to JSON: %v", err)
	}

	// Validate agent
	// Signature validation
	// This does not validate the signature itself, but only checks if it is set.
	// NOTE: we can still push agents with bogus signatures, but we will not be able to verify them.
	if agent.GetSignature() == nil {
		return nil, status.Error(codes.InvalidArgument, "agent signature is required")
	}

	// Size validation
	if len(agentJSON) > maxAgentSize {
		return nil, status.Errorf(codes.InvalidArgument, "agent size exceeds maximum size of %d bytes", maxAgentSize)
	}

	// Push to underlying store
	pushedRef, err := s.store.Push(ctx, ref, bytes.NewReader(agentJSON))
	if err != nil {
		st := status.Convert(err)
		return nil, status.Errorf(st.Code(), "failed to push object to store: %s", st.Message())
	}

	err = s.search.AddRecord(v1alpha1.NewAgentAdapter(agent))
	if err != nil {
		return nil, fmt.Errorf("failed to add agent to search index: %w", err)
	}

	return pushedRef, nil
}

// Pull handles the business logic for pulling an object from the store
func (s *StoreController) Pull(ctx context.Context, ref *objectmanager.RecordObject) (io.ReadCloser, error) {
	storeLogger.Debug("Called common store controller's Pull method", "ref", ref)

	// lookup (maybe not needed)
	lookupRef, err := s.store.Lookup(ctx, ref)
	if err != nil {
		st := status.Convert(err)
		return nil, status.Errorf(st.Code(), "failed to lookup object: %v", st.Message())
	}

	// pull
	reader, err := s.store.Pull(ctx, lookupRef)
	if err != nil {
		st := status.Convert(err)
		return nil, status.Errorf(st.Code(), "failed to pull object: %v", st.Message())
	}

	return reader, nil
}

// Lookup handles the business logic for looking up an object in the store
func (s *StoreController) Lookup(ctx context.Context, ref *objectmanager.RecordObject) (*objectmanager.RecordObject, error) {
	storeLogger.Debug("Called common store controller's Lookup method", "ref", ref)

	// TODO: add caching to avoid querying the Storage API

	// lookup
	meta, err := s.store.Lookup(ctx, ref)
	if err != nil {
		st := status.Convert(err)
		return nil, status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	return meta, nil
}

// Delete handles the business logic for deleting an object from the store
func (s *StoreController) Delete(ctx context.Context, ref *objectmanager.RecordObject) error {
	storeLogger.Debug("Called common store controller's Delete method", "ref", ref)

	err := s.store.Delete(ctx, ref)
	if err != nil {
		st := status.Convert(err)
		return status.Errorf(st.Code(), "failed to delete object: %s", st.Message())
	}

	return nil
}

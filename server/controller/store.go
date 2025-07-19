// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"errors"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	storeLogger.Debug("Called store controller's Push method")

	for {
		// Receive complete Record from stream
		record, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("Push stream completed")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive record: %v", err)
		}

		// Validate record has data
		if record.GetData() == nil {
			return status.Error(codes.InvalidArgument, "record has no data")
		}

		storeLogger.Debug("Processing record", "hasData", record.GetData() != nil)

		// Calculate CID for the record using the GetCid method
		recordCID := record.GetCid()
		if recordCID == "" {
			storeLogger.Error("Failed to calculate record CID")
			return status.Error(codes.Internal, "failed to calculate record CID")
		}

		storeLogger.Debug("CID calculated successfully", "cid", recordCID)

		// Check if record already exists in store
		recordRef := &corev1.RecordRef{Cid: recordCID}
		_, err = s.store.Lookup(stream.Context(), recordRef)
		if err == nil {
			// Record already exists, return existing reference
			storeLogger.Info("Record already exists in store", "cid", recordCID)
			if err := stream.Send(recordRef); err != nil {
				return status.Errorf(codes.Internal, "failed to send existing record reference: %v", err)
			}
			continue
		}

		// Record doesn't exist, push to store (with CID already set)
		storeLogger.Debug("Record not found in store, pushing", "cid", recordCID)
		pushedRef, err := s.store.Push(stream.Context(), record)
		if err != nil {
			storeLogger.Error("Failed to push record to store", "error", err, "cid", recordCID)
			return status.Errorf(codes.Internal, "failed to push record to store: %v", err)
		}

		storeLogger.Info("Record pushed to store successfully", "cid", pushedRef.GetCid())

		// Send the RecordRef back via stream
		if err := stream.Send(pushedRef); err != nil {
			return status.Errorf(codes.Internal, "failed to send record reference: %v", err)
		}
	}
}

func (s storeCtrl) Pull(stream storetypes.StoreService_PullServer) error {
	storeLogger.Debug("Called store controller's Pull method")

	for {
		// Receive RecordRef from stream
		recordRef, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("Pull stream completed")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive record reference: %v", err)
		}

		storeLogger.Debug("Pull request received", "cid", recordRef.GetCid())

		// Validate CID
		if recordRef.GetCid() == "" {
			return status.Error(codes.InvalidArgument, "record cid is required")
		}

		// Check if record exists
		_, err = s.store.Lookup(stream.Context(), recordRef)
		if err != nil {
			st := status.Convert(err)
			return status.Errorf(st.Code(), "failed to lookup record: %s", st.Message())
		}

		// Pull record from store (now returns Record directly)
		record, err := s.store.Pull(stream.Context(), recordRef)
		if err != nil {
			st := status.Convert(err)
			return status.Errorf(st.Code(), "failed to pull record: %s", st.Message())
		}

		storeLogger.Debug("Record pulled successfully", "cid", recordRef.GetCid())

		// Send Record back via stream
		if err := stream.Send(record); err != nil {
			return status.Errorf(codes.Internal, "failed to send record: %v", err)
		}
	}
}

func (s storeCtrl) Lookup(stream storetypes.StoreService_LookupServer) error {
	storeLogger.Debug("Called store controller's Lookup method")

	for {
		// Receive RecordRef from stream
		recordRef, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("Lookup stream completed")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive record reference: %v", err)
		}

		storeLogger.Debug("Lookup request received", "cid", recordRef.GetCid())

		// Validate CID
		if recordRef.GetCid() == "" {
			return status.Error(codes.InvalidArgument, "record cid is required")
		}

		// Lookup record metadata
		recordMeta, err := s.store.Lookup(stream.Context(), recordRef)
		if err != nil {
			st := status.Convert(err)
			return status.Errorf(st.Code(), "failed to lookup record: %s", st.Message())
		}

		storeLogger.Debug("Record metadata retrieved successfully", "cid", recordRef.GetCid())

		// Send RecordMeta back via stream
		if err := stream.Send(recordMeta); err != nil {
			return status.Errorf(codes.Internal, "failed to send record metadata: %v", err)
		}
	}
}

func (s storeCtrl) Delete(stream storetypes.StoreService_DeleteServer) error {
	storeLogger.Debug("Called store controller's Delete method")

	for {
		// Receive RecordRef from stream
		recordRef, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("Delete stream completed")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive record reference: %v", err)
		}

		storeLogger.Debug("Delete request received", "cid", recordRef.GetCid())

		// Validate CID
		if recordRef.GetCid() == "" {
			return status.Error(codes.InvalidArgument, "record cid is required")
		}

		// Delete record from store
		err = s.store.Delete(stream.Context(), recordRef)
		if err != nil {
			st := status.Convert(err)
			return status.Errorf(st.Code(), "failed to delete record: %s", st.Message())
		}

		storeLogger.Info("Record deleted successfully", "cid", recordRef.GetCid())
	}
}

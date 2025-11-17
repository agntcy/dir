// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck,dupl
package controller

import (
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var storeLogger = logging.Logger("controller/store")

type storeCtrl struct {
	storev1.UnimplementedStoreServiceServer
	store    types.StoreAPI
	db       types.DatabaseAPI
	eventBus *events.SafeEventBus
}

func NewStoreController(store types.StoreAPI, db types.DatabaseAPI, eventBus *events.SafeEventBus) storev1.StoreServiceServer {
	return &storeCtrl{
		UnimplementedStoreServiceServer: storev1.UnimplementedStoreServiceServer{},
		store:                           store,
		db:                              db,
		eventBus:                        eventBus,
	}
}

func (s storeCtrl) Push(stream storev1.StoreService_PushServer) error {
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

		isValid, validationErrors, err := record.Validate()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to validate record: %v", err)
		}

		if !isValid {
			// Extract record name and version for better error reporting
			recordName, recordVersion := extractRecordInfo(record)

			// Log validation error with record details
			storeLogger.Warn("Record validation failed",
				"name", recordName,
				"version", recordVersion,
				"errors", validationErrors)

			return status.Errorf(codes.InvalidArgument, "record validation failed: %v", validationErrors)
		}

		pushedRef, err := s.pushRecordToStore(stream.Context(), record)
		if err != nil {
			return err
		}

		// Send the RecordRef back via stream
		if err := stream.Send(pushedRef); err != nil {
			return status.Errorf(codes.Internal, "failed to send record reference: %v", err)
		}
	}
}

func (s storeCtrl) Pull(stream storev1.StoreService_PullServer) error {
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

		// Validate record reference
		if err := s.validateRecordRef(recordRef); err != nil {
			return err
		}

		// Pull record from store
		record, err := s.pullRecordFromStore(stream.Context(), recordRef)
		if err != nil {
			return err
		}

		// Send Record back via stream
		if err := stream.Send(record); err != nil {
			return status.Errorf(codes.Internal, "failed to send record: %v", err)
		}
	}
}

func (s storeCtrl) Lookup(stream storev1.StoreService_LookupServer) error {
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

func (s storeCtrl) Delete(stream storev1.StoreService_DeleteServer) error {
	storeLogger.Debug("Called store controller's Delete method")

	for {
		// Receive RecordRef from stream
		recordRef, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("Delete stream completed")

			if err := stream.SendAndClose(&emptypb.Empty{}); err != nil {
				return status.Errorf(codes.Internal, "failed to send response: %v", err)
			}

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

		// Clean up search database (secondary operation - don't fail on errors)
		if err := s.db.RemoveRecord(recordRef.GetCid()); err != nil {
			// Log error but don't fail the delete - storage is source of truth
			storeLogger.Error("Failed to remove record from search index", "error", err, "cid", recordRef.GetCid())
		} else {
			storeLogger.Debug("Record removed from search index", "cid", recordRef.GetCid())
		}

		storeLogger.Info("Record deleted successfully", "cid", recordRef.GetCid())
	}
}

func (s storeCtrl) PushReferrer(stream storev1.StoreService_PushReferrerServer) error {
	storeLogger.Debug("Called store controller's PushReferrer method")

	for {
		// Receive PushReferrerRequest from stream
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("PushReferrer stream completed")

			return nil
		}

		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive push referrer request: %v", err)
		}

		// Validate the record reference
		if err := s.validateRecordRef(request.GetRecordRef()); err != nil {
			return err
		}

		// Handle the referrer directly since we only have one type now
		response := s.pushReferrer(stream.Context(), request)

		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send push referrer response: %v", err)
		}
	}
}

func (s storeCtrl) pushReferrer(ctx context.Context, request *storev1.PushReferrerRequest) *storev1.PushReferrerResponse {
	storeLogger.Debug("Pushing referrer", "cid", request.GetRecordRef().GetCid(), "type", request.GetReferrer().GetType())

	// Try to use referrer storage if the store supports it
	refStore, ok := s.store.(types.ReferrerStoreAPI)
	if !ok {
		errMsg := "referrer storage not supported by current store implementation"

		return &storev1.PushReferrerResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}
	}

	err := refStore.PushReferrer(ctx, request.GetRecordRef().GetCid(), request.GetReferrer())
	if err != nil {
		errMsg := fmt.Sprintf("failed to store referrer for record %s: %v", request.GetRecordRef().GetCid(), err)

		return &storev1.PushReferrerResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}
	}

	storeLogger.Debug("Referrer stored successfully", "cid", request.GetRecordRef().GetCid(), "type", request.GetReferrer().GetType())

	// Emit RECORD_SIGNED event if this is a signature referrer
	if request.GetReferrer().GetType() == corev1.SignatureReferrerType {
		s.eventBus.RecordSigned(request.GetRecordRef().GetCid(), "client")
	}

	return &storev1.PushReferrerResponse{
		Success: true,
	}
}

// PullReferrer handles retrieving referrers (like signatures) for records.
func (s storeCtrl) PullReferrer(stream storev1.StoreService_PullReferrerServer) error {
	storeLogger.Debug("Called store controller's PullReferrer method")

	for {
		// Receive PullReferrerRequest from stream
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("PullReferrer stream completed")

			return nil
		}

		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive pull referrer request: %v", err)
		}

		// Validate the record reference
		if err := s.validateRecordRef(request.GetRecordRef()); err != nil {
			return err
		}

		// Determine referrer type (empty string means all types)
		referrerType := ""
		if request.ReferrerType != nil {
			referrerType = request.GetReferrerType()
		}

		// Try to use referrer storage if the store supports it
		refStore, ok := s.store.(types.ReferrerStoreAPI)
		if !ok {
			storeLogger.Error("Referrer storage not supported by current store implementation")

			return stream.Send(&storev1.PullReferrerResponse{})
		}

		// Use WalkReferrers with a callback that streams each referrer
		walkFn := func(referrer *corev1.RecordReferrer) error {
			response := &storev1.PullReferrerResponse{
				Referrer: referrer,
			}

			if err := stream.Send(response); err != nil {
				return status.Errorf(codes.Internal, "failed to send referrer response: %v", err)
			}

			storeLogger.Debug("Referrer streamed successfully", "cid", request.GetRecordRef().GetCid(), "type", referrerType)

			return nil
		}

		// Walk referrers of the specified type
		err = refStore.WalkReferrers(stream.Context(), request.GetRecordRef().GetCid(), referrerType, walkFn)
		if err != nil {
			storeLogger.Error("Failed to walk referrers by type for record", "error", err, "cid", request.GetRecordRef().GetCid(), "type", referrerType)

			return stream.Send(&storev1.PullReferrerResponse{})
		}
	}
}

// pushRecordToStore pushes a record to the store and adds it to the search index.
func (s storeCtrl) pushRecordToStore(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	// Push the record to store
	pushedRef, err := s.store.Push(ctx, record)
	if err != nil {
		storeLogger.Error("Failed to push record to store", "error", err)

		return nil, status.Errorf(codes.Internal, "failed to push record to store: %v", err)
	}

	storeLogger.Info("Record pushed to store successfully", "cid", pushedRef.GetCid())

	// Add record to search index for discoverability
	// Use the adapter pattern to convert corev1.Record to types.Record
	recordAdapter := adapters.NewRecordAdapter(record)
	if err := s.db.AddRecord(recordAdapter); err != nil {
		// Log error but don't fail the push operation
		storeLogger.Error("Failed to add record to search index", "error", err, "cid", pushedRef.GetCid())
	} else {
		storeLogger.Debug("Record added to search index successfully", "cid", pushedRef.GetCid())
	}

	return pushedRef, nil
}

// validateRecordRef validates a record reference.
func (s storeCtrl) validateRecordRef(recordRef *corev1.RecordRef) error {
	if recordRef.GetCid() == "" {
		return status.Error(codes.InvalidArgument, "record cid is required")
	}

	return nil
}

// pullRecordFromStore pulls a record from the store with validation.
func (s storeCtrl) pullRecordFromStore(ctx context.Context, recordRef *corev1.RecordRef) (*corev1.Record, error) {
	// Pull record from store
	record, err := s.store.Pull(ctx, recordRef)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to pull record: %s", st.Message())
	}

	storeLogger.Debug("Record pulled successfully", "cid", recordRef.GetCid())

	return record, nil
}

// extractRecordInfo extracts name and version from a record for logging.
func extractRecordInfo(record *corev1.Record) (string, string) {
	name := "unknown"
	version := "unknown"

	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		return name, version
	}

	if recordData != nil {
		name = recordData.GetName()
		version = recordData.GetVersion()
	}

	return name, version
}

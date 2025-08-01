// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	maxAgentSize = 1024 * 1024 * 4 // 4MB
)

var storeLogger = logging.Logger("controller/store")

type storeCtrl struct {
	storev1.UnimplementedStoreServiceServer
	store types.StoreAPI
	db    types.DatabaseAPI
}

func NewStoreController(store types.StoreAPI, db types.DatabaseAPI) storev1.StoreServiceServer {
	return &storeCtrl{
		UnimplementedStoreServiceServer: storev1.UnimplementedStoreServiceServer{},
		store:                           store,
		db:                              db,
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

		// Validate record and get CID
		_, err = s.validateAndGetCID(record)
		if err != nil {
			return err
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

// PushWithOptions handles records with optional OCI artifacts like signatures.
func (s storeCtrl) PushWithOptions(stream storev1.StoreService_PushWithOptionsServer) error {
	storeLogger.Debug("Called store controller's PushWithOptions method")

	for {
		// Receive PushWithOptionsRequest from stream
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("PushWithOptions stream completed")

			return nil
		}

		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive push request: %v", err)
		}

		// Validate record and get CID
		recordCID, err := s.validateAndGetCID(request.GetRecord())
		if err != nil {
			return err
		}

		pushedRef, err := s.pushRecordToStore(stream.Context(), request.GetRecord())
		if err != nil {
			return err
		}

		// Handle signature artifact if provided
		if request.GetOptions() != nil && request.GetOptions().GetSignature() != nil {
			if err := s.pushSignatureToStore(stream.Context(), request.GetOptions().GetSignature(), recordCID); err != nil {
				return err
			}
		}

		// Send the response back via stream
		response := &storev1.PushWithOptionsResponse{
			RecordRef: pushedRef,
		}

		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send push options response: %v", err)
		}
	}
}

// PullWithOptions retrieves records along with their associated OCI artifacts.
func (s storeCtrl) PullWithOptions(stream storev1.StoreService_PullWithOptionsServer) error {
	storeLogger.Debug("Called store controller's PullWithOptions method")

	for {
		// Receive PullWithOptionsRequest from stream
		request, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			storeLogger.Debug("PullWithOptions stream completed")

			return nil
		}

		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive pull request: %v", err)
		}

		recordRef := request.GetRecordRef()

		// Validate record reference
		if err := s.validateRecordRef(recordRef); err != nil {
			return err
		}

		// Pull the record
		record, err := s.pullRecordFromStore(stream.Context(), recordRef)
		if err != nil {
			return err
		}

		// Try to get signature artifact if requested and store supports it
		var signature *signv1.Signature
		if request.GetOptions() != nil {
			signature, err = s.pullSignatureFromStore(stream.Context(), recordRef, request.GetOptions().GetIncludeSignature())
			if err != nil {
				return err
			}
		}

		// Send the response
		response := &storev1.PullWithOptionsResponse{
			Record:    record,
			Signature: signature,
		}

		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send pull options response: %v", err)
		}
	}
}

// validateRecord performs common record validation logic.
func (s storeCtrl) validateRecord(record *corev1.Record) error {
	// Validate record has data
	if record.GetData() == nil {
		return status.Error(codes.InvalidArgument, "record has no data")
	}

	// Validate record size (4MB limit for v1alpha2 API)
	recordSize := proto.Size(record)
	if recordSize > maxAgentSize {
		storeLogger.Warn("Record exceeds size limit", "size", recordSize, "limit", maxAgentSize)

		return status.Errorf(codes.InvalidArgument, "record size %d bytes exceeds maximum allowed size of %d bytes (4MB)", recordSize, maxAgentSize)
	}

	return nil
}

// validateAndGetCID validates a record and gets its CID.
func (s storeCtrl) validateAndGetCID(record *corev1.Record) (string, error) {
	if err := s.validateRecord(record); err != nil {
		return "", err
	}

	// Calculate CID for the record
	recordCID := record.GetCid()
	if recordCID == "" {
		storeLogger.Error("Failed to calculate record CID")

		return "", status.Error(codes.Internal, "failed to calculate record CID")
	}

	storeLogger.Debug("CID calculated successfully", "cid", recordCID)

	return recordCID, nil
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

// pushSignatureIfProvided pushes a signature artifact if the store supports it.
func (s storeCtrl) pushSignatureToStore(ctx context.Context, signature *signv1.Signature, recordCID string) error {
	storeLogger.Debug("Processing signature artifact")

	// Check if store supports signature artifacts (OCI store should)
	if ociStore, ok := s.store.(types.SignatureStoreAPI); ok {
		err := ociStore.PushSignature(ctx, recordCID, signature)
		if err != nil {
			storeLogger.Error("Failed to push signature", "error", err, "cid", recordCID)

			return status.Errorf(codes.Internal, "failed to push signature: %v", err)
		}
	} else {
		storeLogger.Warn("Store does not support signature artifacts, ignoring", "storeType", fmt.Sprintf("%T", s.store))
	}

	return nil
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

// pullSignatureIfRequested pulls a signature artifact if requested and supported.
func (s storeCtrl) pullSignatureFromStore(ctx context.Context, recordRef *corev1.RecordRef, includeSignature bool) (*signv1.Signature, error) {
	if !includeSignature {
		storeLogger.Debug("Signature not requested, skipping")

		return nil, nil //nolint:nilnil
	}

	if ociStore, ok := s.store.(types.SignatureStoreAPI); ok {
		signature, err := ociStore.PullSignature(ctx, recordRef.GetCid())
		if err != nil {
			storeLogger.Error("Failed to pull signature from store", "error", err, "cid", recordRef.GetCid())

			return nil, status.Errorf(codes.Internal, "failed to pull signature from store: %v", err)
		}

		return signature, nil
	}

	return nil, status.Error(codes.Unimplemented, "store does not support signature artifacts") //nolint:nilnil
}

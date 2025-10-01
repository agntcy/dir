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
	"google.golang.org/protobuf/types/known/emptypb"
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

		isValid, validationErrors, err := record.Validate()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to validate record: %v", err)
		}

		if !isValid {
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

		var response *storev1.PushReferrerResponse

		switch request.GetOptions().(type) {
		case *storev1.PushReferrerRequest_Signature:
			storeLogger.Debug("Signature referrer request received")

			response = s.pushSignatureReferrer(stream.Context(), request)
		case *storev1.PushReferrerRequest_PublicKey:
			storeLogger.Debug("Public key referrer request received")

			response = s.pushPublicKeyReferrer(stream.Context(), request)
		default:
			storeLogger.Debug("Unknown referrer type, skipping")

			continue
		}

		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send push referrer response: %v", err)
		}
	}
}

func (s storeCtrl) pushSignatureReferrer(ctx context.Context, request *storev1.PushReferrerRequest) *storev1.PushReferrerResponse {
	storeLogger.Debug("Pushing signature referrer", "cid", request.GetRecordRef().GetCid())

	// Try to use signature storage if the store supports it
	sigStore, ok := s.store.(interface {
		PushSignature(context.Context, string, *signv1.Signature) error
	})
	if !ok {
		errMsg := "signature storage not supported by current store implementation"

		return &storev1.PushReferrerResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}
	}

	err := sigStore.PushSignature(ctx, request.GetRecordRef().GetCid(), request.GetSignature())
	if err != nil {
		errMsg := fmt.Sprintf("failed to store signature for record %s: %v", request.GetRecordRef().GetCid(), err)

		return &storev1.PushReferrerResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}
	}

	storeLogger.Debug("Signature stored successfully", "cid", request.GetRecordRef().GetCid())

	return &storev1.PushReferrerResponse{
		Success: true,
	}
}

func (s storeCtrl) pushPublicKeyReferrer(ctx context.Context, request *storev1.PushReferrerRequest) *storev1.PushReferrerResponse {
	storeLogger.Debug("Pushing public key referrer", "cid", request.GetRecordRef().GetCid())

	// Try to use signature storage if the store supports it
	sigStore, ok := s.store.(interface {
		PushPublicKey(context.Context, string, string) error
	})
	if !ok {
		errMsg := "signature storage not supported by current store implementation"

		return &storev1.PushReferrerResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}
	}

	err := sigStore.PushPublicKey(ctx, request.GetRecordRef().GetCid(), request.GetPublicKey())
	if err != nil {
		errMsg := fmt.Sprintf("failed to store public key for record %s: %v", request.GetRecordRef().GetCid(), err)

		return &storev1.PushReferrerResponse{
			Success:      false,
			ErrorMessage: &errMsg,
		}
	}

	storeLogger.Debug("Public key stored successfully", "cid", request.GetRecordRef().GetCid())

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

		switch request.GetOptions().(type) {
		case *storev1.PullReferrerRequest_PullSignature:
			storeLogger.Debug("Pulling signature referrer", "cid", request.GetRecordRef().GetCid())

			if err := s.pullSignatureReferrer(stream.Context(), request, stream); err != nil {
				return err
			}
		case *storev1.PullReferrerRequest_PullPublicKey:
			storeLogger.Debug("Pulling public key referrer", "cid", request.GetRecordRef().GetCid())

			if err := s.pullPublicKeyReferrer(stream.Context(), request, stream); err != nil {
				return err
			}
		default:
			storeLogger.Debug("Unknown referrer type, skipping")

			continue
		}
	}
}

func (s storeCtrl) pullSignatureReferrer(ctx context.Context, request *storev1.PullReferrerRequest, stream storev1.StoreService_PullReferrerServer) error {
	storeLogger.Debug("Pulling signature referrer", "cid", request.GetRecordRef().GetCid())

	// Try to use signature storage if the store supports it
	sigStore, ok := s.store.(interface {
		PullSignatures(context.Context, string) ([]*signv1.Signature, error)
	})
	if !ok {
		storeLogger.Error("Signature storage not supported by current store implementation")

		response := &storev1.PullReferrerResponse{
			Response: &storev1.PullReferrerResponse_Signature{
				Signature: nil,
			},
		}

		return stream.Send(response)
	}

	signatures, err := sigStore.PullSignatures(ctx, request.GetRecordRef().GetCid())
	if err != nil {
		storeLogger.Error("Failed to pull signature for record", "error", err, "cid", request.GetRecordRef().GetCid())

		response := &storev1.PullReferrerResponse{
			Response: &storev1.PullReferrerResponse_Signature{
				Signature: nil,
			},
		}

		return stream.Send(response)
	}

	for _, signature := range signatures {
		response := &storev1.PullReferrerResponse{
			Response: &storev1.PullReferrerResponse_Signature{
				Signature: signature,
			},
		}

		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send signature response: %v", err)
		}

		storeLogger.Debug("Signature streamed successfully", "cid", request.GetRecordRef().GetCid())
	}

	return nil
}

func (s storeCtrl) pullPublicKeyReferrer(ctx context.Context, request *storev1.PullReferrerRequest, stream storev1.StoreService_PullReferrerServer) error {
	storeLogger.Debug("Pulling public key referrer", "cid", request.GetRecordRef().GetCid())

	// Try to use signature storage if the store supports it
	sigStore, ok := s.store.(interface {
		PullPublicKeys(context.Context, string) ([]string, error)
	})

	if !ok {
		storeLogger.Error("Public key storage not supported by current store implementation")

		response := &storev1.PullReferrerResponse{
			Response: &storev1.PullReferrerResponse_PublicKey{
				PublicKey: "",
			},
		}

		return stream.Send(response)
	}

	publicKeys, err := sigStore.PullPublicKeys(ctx, request.GetRecordRef().GetCid())
	if err != nil {
		storeLogger.Error("Failed to pull public key for record", "error", err, "cid", request.GetRecordRef().GetCid())

		response := &storev1.PullReferrerResponse{
			Response: &storev1.PullReferrerResponse_PublicKey{
				PublicKey: "",
			},
		}

		return stream.Send(response)
	}

	// Stream each public key as a separate response
	for _, publicKey := range publicKeys {
		response := &storev1.PullReferrerResponse{
			Response: &storev1.PullReferrerResponse_PublicKey{
				PublicKey: publicKey,
			},
		}

		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send public key response: %v", err)
		}

		storeLogger.Debug("Public key streamed successfully", "cid", request.GetRecordRef().GetCid())
	}

	return nil
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

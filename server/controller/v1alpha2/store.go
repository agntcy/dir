// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package v1alpha2

import (
	"bytes"
	"context"
	"errors"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
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
	firstMessage, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive first message: %v", err)
	}

	// Generate CID for the object if not set
	objRef := firstMessage.GetObjectRef()

	storeLogger.Debug("Called store contoller's Push method",
		"data", firstMessage.GetData(),
		"type", firstMessage.GetObjectType(),
		"object-ref", objRef,
	)

	oRef := adapters.NewObjectRefV1(objRef)
	_, err = s.store.Lookup(stream.Context(), oRef)
	if err == nil {
		storeLogger.Info("Object already exists, skipping push to store", "ref", oRef)

		return stream.SendAndClose(&storetypes.PushResponse{})
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

	coreObject := &coretypes.Object{}
	if _, err := coreObject.LoadFromReader(pr); err != nil {
		return status.Errorf(codes.Internal, "failed to load agent from reader: %v", err)
	}

	if err := generateCID(coreObject); err != nil {
		return status.Errorf(codes.Internal, "failed to generate cid: %v", err)
	}

	// Convert object to JSON to drop additional fields
	// objectJSON, err := json.Marshal(coreObject)
	// if err != nil {
	// 	return status.Errorf(codes.Internal, "failed to marshal object to JSON: %v", err)
	// }

	// Validate agent
	// Signature validation
	// This does not validate the signature itself, but only checks if it is set.
	// NOTE: we can still push agents with bogus signatures, but we will not be able to verify them.
	// if agent.GetSignature() == nil {
	// 	return status.Error(codes.InvalidArgument, "agent signature is required")
	// }

	// Size validation
	// if len(objectJSON) > maxAgentSize {
	// 	return status.Errorf(codes.InvalidArgument, "object size exceeds maximum size of %d bytes", maxAgentSize)
	// }

	object := adapters.NewObjectV1(coreObject)
	// Push to underlying store
	ref, err := s.store.Push(stream.Context(), object, bytes.NewReader(object.Data()))
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to push object to store: %s", st.Message())
	}

	// TODO: should we call AddRecord if the object is not an RecordObject?
	// err = s.search.AddRecord(v1alpha1.NewAgentAdapter(agent, ref.GetDigest()))
	// if err != nil {
	// 	return fmt.Errorf("failed to add agent to search index: %w", err)
	// }

	return stream.SendAndClose(&storetypes.PushResponse{
		ObjectRef: adapters.ObjectRefToV1Proto(ref),
	})
}

func (s storeCtrl) Pull(req *storetypes.PullRequest, stream storetypes.StoreService_PullServer) error {
	storeLogger.Debug("Called store contoller's Pull method", "req", req)
	ref := adapters.NewObjectRefV1(req.GetObjectRef())

	if ref.CID() == "" {
		return status.Error(codes.InvalidArgument, "object cid is required")
	}

	_, err := s.store.Lookup(stream.Context(), ref)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	reader, err := s.store.Pull(stream.Context(), ref)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to pull object: %s", st.Message())
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
		err = stream.Send(&storetypes.PullResponseChunk{
			Data: buf[:n],
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send data: %v", err)
		}
	}
}

func (s storeCtrl) Lookup(ctx context.Context, req *storetypes.LookupRequest) (*storetypes.LookupResponse, error) {
	storeLogger.Debug("Called store contoller's Lookup method", "req", req)
	ref := adapters.NewObjectRefV1(req.GetObjectRef())

	if ref.CID() == "" {
		return nil, status.Error(codes.InvalidArgument, "object cid is required")
	}

	meta, err := s.store.Lookup(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	return &storetypes.LookupResponse{
		Object: adapters.ObjectToV1Proto(meta),
	}, nil
}

func (s storeCtrl) Delete(ctx context.Context, req *storetypes.DeleteRequest) (*storetypes.DeleteResponse, error) {
	storeLogger.Debug("Called store contoller's Delete method", "req", req)
	ref := adapters.NewObjectRefV1(req.GetObjectRef())

	if ref.CID() == "" {
		return nil, status.Error(codes.InvalidArgument, "object cid is required")
	}

	err := s.store.Delete(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to delete object: %s", st.Message())
	}

	return &storetypes.DeleteResponse{}, nil
}

func generateCID(o *coretypes.Object) error {
	if o.GetRef().GetCid() != "" {
		return nil
	}

	// TODO: Type is enum. It is used by string in annotations but it also used as uint64 in CID generation.
	t := coretypes.ObjectType_OBJECT_TYPE_UNSPECIFIED

	pref := cid.Prefix{
		Version:  1, // CIDv1
		Codec:    uint64(t),
		MhType:   mh.SHA2_256, // SHA2-256 hash function
		MhLength: -1,          // default length (32 bytes for SHA2-256)
	}

	c, err := pref.Sum(o.GetData())
	if err != nil {
		return err
	}

	o.Cid = c.String()

	return nil
}

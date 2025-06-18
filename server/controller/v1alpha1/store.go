// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package v1alpha1

import (
	"context"
	"errors"
	"io"

	"github.com/agntcy/dir/api/converter"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var storeLogger = logging.Logger("controller/v1alpha1/store")

type storeCtrl struct {
	storetypes.UnimplementedStoreServiceServer
	commonController *controller.StoreController
}

func NewStoreController(store types.StoreAPI, search types.SearchAPI) storetypes.StoreServiceServer {
	return &storeCtrl{
		UnimplementedStoreServiceServer: storetypes.UnimplementedStoreServiceServer{},
		commonController:                controller.NewStoreController(store, search),
	}
}

func (s storeCtrl) Push(stream storetypes.StoreService_PushServer) error {
	// TODO: validate
	firstMessage, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive first message: %v", err)
	}

	storeLogger.Debug("Called v1alpha1 store controller's Push method",
		"data", firstMessage.GetData(),
		"agent", firstMessage.GetAgent(),
		"object-ref", firstMessage.GetRef(),
	)

	// Convert core.v1alpha1.Object to objectmanager.RecordObject
	recordObject, err := converter.CoreV1Alpha1ToRecordObject(firstMessage)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to convert object: %v", err)
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

	// Use common controller to handle business logic
	resultRef, err := s.commonController.Push(stream.Context(), recordObject, pr)
	if err != nil {
		return err
	}

	// Convert objectmanager.RecordObject back to core.v1alpha1.ObjectRef
	objectRef, err := converter.RecordObjectToCoreV1Alpha1(resultRef)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to convert result: %v", err)
	}

	// Extract the ObjectRef from the Object
	if objectRef.GetRef() == nil {
		return status.Errorf(codes.Internal, "converted object has no ref")
	}

	return stream.SendAndClose(objectRef.GetRef())
}

func (s storeCtrl) Pull(req *coretypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	storeLogger.Debug("Called v1alpha1 store controller's Pull method", "req", req)

	// Convert core.v1alpha1.ObjectRef to objectmanager.RecordObject
	recordObject, err := converter.CoreV1Alpha1ToRecordObject(&coretypes.Object{Ref: req})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to convert object ref: %v", err)
	}

	// Use common controller to handle business logic
	reader, err := s.commonController.Pull(stream.Context(), recordObject)
	if err != nil {
		return err
	}
	defer reader.Close()

	buf := make([]byte, 4096) //nolint:mnd

	for {
		n, readErr := reader.Read(buf)
		if readErr == io.EOF && n == 0 {
			storeLogger.Debug("Finished reading all chunks")
			return nil
		}

		if readErr != io.EOF && readErr != nil {
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
	storeLogger.Debug("Called v1alpha1 store controller's Lookup method", "req", req)

	// validate
	if req.GetDigest() == "" {
		return nil, status.Error(codes.InvalidArgument, "digest is required")
	}

	// Convert core.v1alpha1.ObjectRef to objectmanager.RecordObject
	recordObject, err := converter.CoreV1Alpha1ToRecordObject(&coretypes.Object{Ref: req})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert object ref: %v", err)
	}

	// Use common controller to handle business logic
	resultRef, err := s.commonController.Lookup(ctx, recordObject)
	if err != nil {
		return nil, err
	}

	// Convert objectmanager.RecordObject back to core.v1alpha1.ObjectRef
	objectRef, err := converter.RecordObjectToCoreV1Alpha1(resultRef)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert result: %v", err)
	}

	// Extract the ObjectRef from the Object
	if objectRef.GetRef() == nil {
		return nil, status.Errorf(codes.Internal, "converted object has no ref")
	}

	return objectRef.GetRef(), nil
}

func (s storeCtrl) Delete(ctx context.Context, req *coretypes.ObjectRef) (*emptypb.Empty, error) {
	storeLogger.Debug("Called v1alpha1 store controller's Delete method", "req", req)

	if req.GetDigest() == "" {
		return nil, status.Error(codes.InvalidArgument, "digest is required")
	}

	// Convert core.v1alpha1.ObjectRef to objectmanager.RecordObject
	recordObject, err := converter.CoreV1Alpha1ToRecordObject(&coretypes.Object{Ref: req})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert object ref: %v", err)
	}

	// Use common controller to handle business logic
	err = s.commonController.Delete(ctx, recordObject)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

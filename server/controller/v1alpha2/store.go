// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package v1alpha2

import (
	"context"

	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var storeLogger = logging.Logger("controller/v1alpha2/store")

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
	// TODO: implement v1alpha2 store push logic with converter
	storeLogger.Debug("Called v1alpha2 store controller's Push method")
	return status.Errorf(codes.Unimplemented, "method Push not implemented")
}

func (s storeCtrl) Pull(req *storetypes.ObjectRef, stream storetypes.StoreService_PullServer) error {
	// TODO: implement v1alpha2 store pull logic with converter
	storeLogger.Debug("Called v1alpha2 store controller's Pull method", "req", req)
	return status.Errorf(codes.Unimplemented, "method Pull not implemented")
}

func (s storeCtrl) Lookup(ctx context.Context, req *storetypes.ObjectRef) (*storetypes.Object, error) {
	// TODO: implement v1alpha2 store lookup logic with converter
	storeLogger.Debug("Called v1alpha2 store controller's Lookup method", "req", req)
	return nil, status.Errorf(codes.Unimplemented, "method Lookup not implemented")
}

func (s storeCtrl) Delete(ctx context.Context, req *storetypes.ObjectRef) (*emptypb.Empty, error) {
	// TODO: implement v1alpha2 store delete logic with converter
	storeLogger.Debug("Called v1alpha2 store controller's Delete method", "req", req)
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}

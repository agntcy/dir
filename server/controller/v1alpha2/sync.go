// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"
	"fmt"

	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var syncLogger = logging.Logger("controller/sync")

type syncCtlr struct {
	storetypes.UnimplementedSyncServiceServer
	db types.DatabaseAPI
}

func NewSyncController(db types.DatabaseAPI) storetypes.SyncServiceServer {
	return &syncCtlr{
		UnimplementedSyncServiceServer: storetypes.UnimplementedSyncServiceServer{},
		db:                             db,
	}
}

func (c *syncCtlr) CreateSync(_ context.Context, req *storetypes.CreateSyncRequest) (*storetypes.CreateSyncResponse, error) {
	syncLogger.Debug("Called sync controller's CreateSync method")

	id, err := c.db.CreateSync(req.GetRemoteDirectoryUrl())
	if err != nil {
		return nil, fmt.Errorf("failed to create sync: %w", err)
	}

	syncLogger.Debug("Sync created successfully")

	return &storetypes.CreateSyncResponse{
		SyncId: id,
	}, nil
}

func (c *syncCtlr) ListSyncs(req *storetypes.ListSyncsRequest, srv storetypes.SyncService_ListSyncsServer) error {
	syncLogger.Debug("Called sync controller's ListSyncs method", "req", req)

	syncs, err := c.db.GetSyncs()
	if err != nil {
		return fmt.Errorf("failed to list syncs: %w", err)
	}

	for _, sync := range syncs {
		resp := &storetypes.ListSyncsItem{
			SyncId:             sync.GetID(),
			RemoteDirectoryUrl: sync.GetRemoteDirectoryURL(),
			Status:             sync.GetStatus(),
		}

		syncLogger.Debug("Sending sync object", "sync_id", sync.GetID(), "status", sync.GetStatus())

		if err := srv.Send(resp); err != nil {
			return fmt.Errorf("failed to send sync object: %w", err)
		}
	}

	syncLogger.Debug("Finished sending sync objects")

	return nil
}

func (c *syncCtlr) GetSync(_ context.Context, req *storetypes.GetSyncRequest) (*storetypes.GetSyncResponse, error) {
	syncLogger.Debug("Called sync controller's GetSync method", "req", req)

	syncObj, err := c.db.GetSyncByID(req.GetSyncId())
	if err != nil {
		return nil, fmt.Errorf("failed to get sync by ID: %w", err)
	}

	return &storetypes.GetSyncResponse{
		SyncId:             syncObj.GetID(),
		RemoteDirectoryUrl: syncObj.GetRemoteDirectoryURL(),
		Status:             syncObj.GetStatus(),
	}, nil
}

func (c *syncCtlr) DeleteSync(_ context.Context, req *storetypes.DeleteSyncRequest) (*storetypes.DeleteSyncResponse, error) {
	syncLogger.Debug("Called sync controller's DeleteSync method", "req", req)

	if err := c.db.DeleteSync(req.GetSyncId()); err != nil {
		return nil, fmt.Errorf("failed to delete sync: %w", err)
	}

	syncLogger.Debug("Sync deleted successfully", "sync_id", req.GetSyncId())

	return &storetypes.DeleteSyncResponse{}, nil
}

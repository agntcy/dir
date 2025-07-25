// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var syncLogger = logging.Logger("controller/sync")

// syncCtlr implements the SyncService gRPC interface.
type syncCtlr struct {
	storetypes.UnimplementedSyncServiceServer
	db   types.DatabaseAPI
	opts types.APIOptions
}

// NewSyncController creates a new sync controller.
func NewSyncController(db types.DatabaseAPI, opts types.APIOptions) storetypes.SyncServiceServer {
	return &syncCtlr{
		db:   db,
		opts: opts,
	}
}

func (c *syncCtlr) CreateSync(_ context.Context, req *storetypes.CreateSyncRequest) (*storetypes.CreateSyncResponse, error) {
	syncLogger.Debug("Called sync controller's CreateSync method")

	// Validate the remote directory URL
	if err := validateRemoteDirectoryURL(req.GetRemoteDirectoryUrl()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid remote directory URL: %v", err)
	}

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

	syncs, err := c.db.GetSyncs(int(req.GetOffset()), int(req.GetLimit()))
	if err != nil {
		return fmt.Errorf("failed to list syncs: %w", err)
	}

	for _, sync := range syncs {
		syncLogger.Debug("Sending sync object", "sync_id", sync.GetID(), "status", sync.GetStatus())

		if err := srv.Send(&storetypes.ListSyncsItem{
			SyncId:             sync.GetID(),
			RemoteDirectoryUrl: sync.GetRemoteDirectoryURL(),
			Status:             sync.GetStatus(),
		}); err != nil {
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

	// Get the sync to check its current status
	syncObj, err := c.db.GetSyncByID(req.GetSyncId())
	if err != nil {
		return nil, fmt.Errorf("failed to get sync: %w", err)
	}

	if syncObj.GetStatus() == storetypes.SyncStatus_SYNC_STATUS_DELETED {
		return nil, status.Errorf(codes.NotFound, "sync has already been deleted")
	}

	// Mark sync for deletion - the scheduler will pick this up
	if err := c.db.UpdateSyncStatus(req.GetSyncId(), storetypes.SyncStatus_SYNC_STATUS_DELETE_PENDING); err != nil {
		return nil, fmt.Errorf("failed to mark sync for deletion: %w", err)
	}

	syncLogger.Debug("Sync marked for deletion", "sync_id", req.GetSyncId())

	return &storetypes.DeleteSyncResponse{}, nil
}

// RequestRegistryCredentials handles requests for registry authentication credentials.
func (c *syncCtlr) RequestRegistryCredentials(_ context.Context, req *storetypes.RequestRegistryCredentialsRequest) (*storetypes.RequestRegistryCredentialsResponse, error) {
	syncLogger.Debug("Called sync controller's RequestRegistryCredentials method", "req", req)

	// Validate requesting node ID
	if req.GetRequestingNodeId() == "" {
		return &storetypes.RequestRegistryCredentialsResponse{
			Success:      false,
			ErrorMessage: "requesting node ID is required",
		}, nil
	}

	// Get OCI configuration to determine registry details
	ociConfig := c.opts.Config().OCI

	// Build registry URL based on configuration
	registryURL := ociConfig.RegistryAddress
	if registryURL == "" {
		registryURL = ociconfig.DefaultRegistryAddress
	}

	// TODO Skip credentials generation for now
	return &storetypes.RequestRegistryCredentialsResponse{
		Success:           true,
		RemoteRegistryUrl: registryURL,
		Credentials:       nil,
	}, nil
}

// validateRemoteDirectoryURL validates the format of a remote directory URL.
func validateRemoteDirectoryURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("remote directory URL is required")
	}

	// If the URL doesn't have a scheme, treat it as a raw host:port
	if !strings.Contains(rawURL, "://") {
		// Validate that it looks like host:port
		if !strings.Contains(rawURL, ":") {
			return errors.New("URL must include port (e.g., 'host:port' or 'http://host:port')")
		}

		return nil
	}

	// Parse as full URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported scheme '%s', only 'http' and 'https' are supported", parsedURL.Scheme)
	}

	// Validate hostname
	if parsedURL.Hostname() == "" {
		return errors.New("URL must include a hostname")
	}

	return nil
}

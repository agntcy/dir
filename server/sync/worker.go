// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"fmt"
	"time"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/events"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	syncconfig "github.com/agntcy/dir/server/sync/config"
	"github.com/agntcy/dir/server/sync/monitor"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
	zotutils "github.com/agntcy/dir/utils/zot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	zotsyncconfig "zotregistry.dev/zot/v2/pkg/extensions/config/sync"
)

// Worker processes sync work items.
type Worker struct {
	id             int
	db             types.DatabaseAPI
	store          types.StoreAPI
	workQueue      <-chan synctypes.WorkItem
	timeout        time.Duration
	monitorService *monitor.MonitorService
	eventBus       *events.SafeEventBus
}

// NewWorker creates a new worker instance.
func NewWorker(id int, db types.DatabaseAPI, store types.StoreAPI, workQueue <-chan synctypes.WorkItem, timeout time.Duration, monitorService *monitor.MonitorService, eventBus *events.SafeEventBus) *Worker {
	return &Worker{
		id:             id,
		db:             db,
		store:          store,
		workQueue:      workQueue,
		timeout:        timeout,
		monitorService: monitorService,
		eventBus:       eventBus,
	}
}

// Run starts the worker loop.
func (w *Worker) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting sync worker", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping due to context cancellation", "worker_id", w.id)

			return
		case <-stopCh:
			logger.Info("Worker stopping due to stop signal", "worker_id", w.id)

			return
		case workItem := <-w.workQueue:
			w.processWorkItem(ctx, workItem)
		}
	}
}

// processWorkItem handles a single sync work item.
func (w *Worker) processWorkItem(ctx context.Context, item synctypes.WorkItem) {
	logger.Info("Processing sync work item", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)
	// TODO Check if store is oci and zot. If not, fail

	// Create timeout context for this work item
	workCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	var finalStatus storev1.SyncStatus

	switch item.Type {
	case synctypes.WorkItemTypeSyncCreate:
		// Emit SYNC_CREATED event when sync operation begins
		w.eventBus.SyncCreated(item.SyncID, item.RemoteDirectoryURL)

		finalStatus = storev1.SyncStatus_SYNC_STATUS_IN_PROGRESS

		err := w.addSync(workCtx, item)
		if err != nil {
			logger.Error("Sync failed", "worker_id", w.id, "sync_id", item.SyncID, "error", err)

			// Emit SYNC_FAILED event
			w.eventBus.SyncFailed(item.SyncID, item.RemoteDirectoryURL, err.Error())

			finalStatus = storev1.SyncStatus_SYNC_STATUS_FAILED
		} else {
			// Emit SYNC_COMPLETED event when sync succeeds
			// Note: The sync continues monitoring in background, but the initial sync operation is complete
			recordCount := len(item.CIDs)
			w.eventBus.SyncCompleted(item.SyncID, item.RemoteDirectoryURL, recordCount)
		}

	case synctypes.WorkItemTypeSyncDelete:
		finalStatus = storev1.SyncStatus_SYNC_STATUS_DELETED

		err := w.deleteSync(workCtx, item)
		if err != nil {
			logger.Error("Sync delete failed", "worker_id", w.id, "sync_id", item.SyncID, "error", err)

			finalStatus = storev1.SyncStatus_SYNC_STATUS_FAILED
		}

	default:
		logger.Error("Unknown work item type", "worker_id", w.id, "sync_id", item.SyncID, "type", item.Type)
	}

	// Update status in database
	if err := w.db.UpdateSyncStatus(item.SyncID, finalStatus); err != nil {
		logger.Error("Failed to update sync status", "worker_id", w.id, "sync_id", item.SyncID, "status", finalStatus, "error", err)
	}
}

func (w *Worker) deleteSync(_ context.Context, item synctypes.WorkItem) error {
	logger.Debug("Starting sync delete operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Get remote registry URL from sync object
	remoteRegistryURL, err := w.db.GetSyncRemoteRegistry(item.SyncID)
	if err != nil {
		return fmt.Errorf("failed to get remote registry URL: %w", err)
	}

	// Remove registry from zot configuration
	if err := zotutils.RemoveRegistryFromSyncConfig(zotutils.DefaultZotConfigPath, remoteRegistryURL); err != nil {
		return fmt.Errorf("failed to remove registry from zot sync: %w", err)
	}

	// Start graceful monitoring shutdown - this will continue monitoring
	// until all records that zot may still be syncing are indexed
	if err := w.monitorService.StopSyncMonitoring(item.SyncID); err != nil { //nolint:contextcheck
		// Warn but continue
		logger.Warn("Failed to initiate graceful monitoring shutdown", "worker_id", w.id, "sync_id", item.SyncID, "error", err)
	}

	return nil
}

// addSync implements the core synchronization logic.
//
//nolint:unparam
func (w *Worker) addSync(ctx context.Context, item synctypes.WorkItem) error {
	logger.Debug("Starting sync operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Negotiate credentials with remote node using RequestRegistryCredentials RPC
	remoteRegistryURL, credentials, err := w.negotiateCredentials(ctx, item.RemoteDirectoryURL)
	if err != nil {
		return fmt.Errorf("failed to negotiate credentials: %w", err)
	}

	// Store credentials for later use in sync process
	logger.Debug("Credentials negotiated successfully", "worker_id", w.id, "sync_id", item.SyncID)

	// Update sync object with remote registry URL
	if err := w.db.UpdateSyncRemoteRegistry(item.SyncID, remoteRegistryURL); err != nil {
		return fmt.Errorf("failed to update sync remote registry: %w", err)
	}

	// Update zot configuration with sync extension to trigger sync
	if err := zotutils.AddRegistryToSyncConfig(zotutils.DefaultZotConfigPath, remoteRegistryURL, ociconfig.DefaultRepositoryName, zotsyncconfig.Credentials{
		Username: credentials.Username,
		Password: credentials.Password,
	}, item.CIDs); err != nil {
		return fmt.Errorf("failed to add registry to zot sync: %w", err)
	}

	// Start monitoring the local registry for changes after Zot sync is configured
	if err := w.monitorService.StartSyncMonitoring(item.SyncID); err != nil { //nolint:contextcheck
		return fmt.Errorf("failed to start registry monitoring: %w", err)
	}

	logger.Debug("Sync operation completed", "worker_id", w.id, "sync_id", item.SyncID)

	return nil
}

// negotiateCredentials negotiates registry credentials with the remote Directory node.
func (w *Worker) negotiateCredentials(ctx context.Context, remoteDirectoryURL string) (string, syncconfig.AuthConfig, error) {
	logger.Debug("Starting credential negotiation", "worker_id", w.id, "remote_url", remoteDirectoryURL)

	// Create gRPC connection to the remote Directory node
	conn, err := grpc.NewClient(
		remoteDirectoryURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return "", syncconfig.AuthConfig{}, fmt.Errorf("failed to create gRPC connection to remote node %s: %w", remoteDirectoryURL, err)
	}
	defer conn.Close()

	// Create SyncService client
	syncClient := storev1.NewSyncServiceClient(conn)

	// TODO: Get actual peer ID from the routing system or configuration
	requestingNodeID := "directory://local-node"

	// Make the credential negotiation request
	resp, err := syncClient.RequestRegistryCredentials(ctx, &storev1.RequestRegistryCredentialsRequest{
		RequestingNodeId: requestingNodeID,
	})
	if err != nil {
		return "", syncconfig.AuthConfig{}, fmt.Errorf("failed to request registry credentials from %s: %w", remoteDirectoryURL, err)
	}

	// Check if the negotiation was successful
	if !resp.GetSuccess() {
		return "", syncconfig.AuthConfig{}, fmt.Errorf("credential negotiation failed: %s", resp.GetErrorMessage())
	}

	return resp.GetRemoteRegistryUrl(), syncconfig.AuthConfig{
		Username: resp.GetBasicAuth().GetUsername(),
		Password: resp.GetBasicAuth().GetPassword(),
	}, nil
}

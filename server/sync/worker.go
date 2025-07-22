// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	zotconfig "github.com/agntcy/dir/server/sync/config/zot"
	synctypes "github.com/agntcy/dir/server/sync/types"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Worker processes sync work items.
type Worker struct {
	id        int
	db        types.DatabaseAPI
	store     types.StoreAPI
	workQueue <-chan synctypes.WorkItem
	timeout   time.Duration
}

// NewWorker creates a new worker instance.
func NewWorker(id int, db types.DatabaseAPI, store types.StoreAPI, workQueue <-chan synctypes.WorkItem, timeout time.Duration) *Worker {
	return &Worker{
		id:        id,
		db:        db,
		store:     store,
		workQueue: workQueue,
		timeout:   timeout,
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

	// Create timeout context for this work item
	workCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// TODO Check if store is oci and zot. If not, fail
	err := w.performSync(workCtx, item)

	// Update sync status based on result
	var finalStatus storetypes.SyncStatus

	if err != nil {
		logger.Error("Sync failed", "worker_id", w.id, "sync_id", item.SyncID, "error", err)

		finalStatus = storetypes.SyncStatus_SYNC_STATUS_FAILED
	} else {
		logger.Info("Sync completed successfully", "worker_id", w.id, "sync_id", item.SyncID)

		finalStatus = storetypes.SyncStatus_SYNC_STATUS_COMPLETED
	}

	// Update status in database

	if err := w.db.UpdateSyncStatus(item.SyncID, finalStatus); err != nil {
		logger.Error("Failed to update sync status", "worker_id", w.id, "sync_id", item.SyncID, "status", finalStatus, "error", err)
	}
}

// performSync implements the core synchronization logic.
//
//nolint:unparam
func (w *Worker) performSync(ctx context.Context, item synctypes.WorkItem) error {
	logger.Debug("Starting sync operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Negotiate credentials with remote node using RequestRegistryCredentials RPC
	remoteRegistryURL, err := w.negotiateCredentials(ctx, item.RemoteDirectoryURL)
	if err != nil {
		return fmt.Errorf("failed to negotiate credentials: %w", err)
	}

	// Store credentials for later use in sync process
	logger.Debug("Credentials negotiated successfully", "worker_id", w.id, "sync_id", item.SyncID)

	// Update zot configuration with sync extension to trigger sync
	if err := w.updateZotConfig(ctx, remoteRegistryURL); err != nil {
		return fmt.Errorf("failed to update zot config: %w", err)
	}

	logger.Debug("Sync operation completed", "worker_id", w.id, "sync_id", item.SyncID)

	return nil
}

// negotiateCredentials negotiates registry credentials with the remote Directory node.
func (w *Worker) negotiateCredentials(ctx context.Context, remoteDirectoryURL string) (string, error) {
	logger.Debug("Starting credential negotiation", "worker_id", w.id, "remote_url", remoteDirectoryURL)

	// Create gRPC connection to the remote Directory node
	conn, err := grpc.NewClient(
		remoteDirectoryURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create gRPC connection to remote node %s: %w", remoteDirectoryURL, err)
	}
	defer conn.Close()

	// Create SyncService client
	syncClient := storetypes.NewSyncServiceClient(conn)

	// TODO: Get actual peer ID from the routing system or configuration
	requestingNodeID := "directory://local-node"

	// Make the credential negotiation request
	resp, err := syncClient.RequestRegistryCredentials(ctx, &storetypes.RequestRegistryCredentialsRequest{
		RequestingNodeId: requestingNodeID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to request registry credentials from %s: %w", remoteDirectoryURL, err)
	}

	// Check if the negotiation was successful
	if !resp.GetSuccess() {
		return "", fmt.Errorf("credential negotiation failed: %s", resp.GetErrorMessage())
	}

	logger.Info("Successfully negotiated registry credentials", "worker_id", w.id, "remote_url", remoteDirectoryURL, "registry_url", resp.GetRemoteRegistryUrl())

	// TODO: Get credentials from the response
	return resp.GetRemoteRegistryUrl(), nil
}

func (w *Worker) updateZotConfig(_ context.Context, remoteDirectoryURL string) error {
	logger.Debug("Updating zot config", "worker_id", w.id, "remote_url", remoteDirectoryURL)

	// Validate input
	if remoteDirectoryURL == "" {
		return errors.New("remote directory URL cannot be empty")
	}

	// Load zot config file from /etc/zot in zot's pod
	config, err := os.ReadFile(zotconfig.DefaultZotConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read zot config file %s: %w", zotconfig.DefaultZotConfigPath, err)
	}

	logger.Debug("Zot config file", "worker_id", w.id, "remote_url", remoteDirectoryURL, "file", string(config))

	var zotConfig zotconfig.Config
	if err := json.Unmarshal(config, &zotConfig); err != nil {
		return fmt.Errorf("failed to unmarshal zot config: %w", err)
	}

	// Initialize extensions if nil
	if zotConfig.Extensions == nil {
		zotConfig.Extensions = &zotconfig.SyncExtensions{}
	}

	// Initialize sync config if nil
	syncConfig := zotConfig.Extensions.Sync
	if syncConfig == nil {
		syncConfig = &zotconfig.SyncConfig{}
		zotConfig.Extensions.Sync = syncConfig
	}

	syncConfig.Enable = Ptr(true)

	// Create registry configuration with credentials if provided
	// Add http:// scheme if not present for zot sync
	registryURL, err := w.normalizeRegistryURL(remoteDirectoryURL)
	if err != nil {
		return fmt.Errorf("failed to normalize registry URL: %w", err)
	}

	registry := zotconfig.SyncRegistryConfig{
		URLs:         []string{registryURL},
		OnDemand:     false, // Disable OnDemand for proactive sync
		PollInterval: zotconfig.DefaultPollInterval,
		MaxRetries:   zotconfig.DefaultMaxRetries,
		RetryDelay:   zotconfig.DefaultRetryDelay,
		TLSVerify:    Ptr(false),
		Content: []zotconfig.SyncContent{
			{
				Prefix: ociconfig.DefaultRepositoryName,
			},
		},
	}
	syncConfig.Registries = append(syncConfig.Registries, registry)

	logger.Debug("Zot config updated", "worker_id", w.id, "remote_url", remoteDirectoryURL, "config", zotConfig)

	// Write the updated config back to the file
	updatedConfig, err := json.MarshalIndent(zotConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated zot config: %w", err)
	}

	if err := os.WriteFile(zotconfig.DefaultZotConfigPath, updatedConfig, 0o644); err != nil { //nolint:gosec,mnd
		return fmt.Errorf("failed to write updated zot config: %w", err)
	}

	logger.Info("Successfully updated zot config", "worker_id", w.id, "remote_url", remoteDirectoryURL)

	return nil
}

// normalizeRegistryURL ensures the registry URL has the proper scheme for zot sync.
func (w *Worker) normalizeRegistryURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", errors.New("registry URL cannot be empty")
	}

	// Add http:// scheme if not present for zot sync
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "http://" + rawURL, nil
	}

	// Validate the URL format
	if _, err := url.Parse(rawURL); err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	return rawURL, nil
}

func Ptr[T any](v T) *T {
	return &v
}

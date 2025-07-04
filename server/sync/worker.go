// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	searchtypes "github.com/agntcy/dir/api/search/v1alpha2"
	storetypes "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/server/database/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Worker processes sync work items.
type Worker struct {
	id        int
	db        types.DatabaseAPI
	store     types.StoreAPI
	workQueue <-chan WorkItem
	timeout   time.Duration
}

// NewWorker creates a new worker instance.
func NewWorker(id int, db types.DatabaseAPI, store types.StoreAPI, workQueue <-chan WorkItem, timeout time.Duration) *Worker {
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
func (w *Worker) processWorkItem(ctx context.Context, item WorkItem) {
	logger.Info("Processing sync work item", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Create timeout context for this work item
	workCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Process the sync operation
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
func (w *Worker) performSync(ctx context.Context, item WorkItem) error {
	logger.Debug("Starting sync operation", "worker_id", w.id, "sync_id", item.SyncID, "remote_url", item.RemoteDirectoryURL)

	// Create connection to remote server
	remoteClient, err := client.New(client.WithConfig(&client.Config{
		ServerAddress: item.RemoteDirectoryURL,
	}))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get all records from the remote directory
	records, err := w.searchAllRemoteRecords(ctx, remoteClient)
	if err != nil {
		return fmt.Errorf("failed to search records: %w", err)
	}

	// Process each record
	for _, cid := range records {
		// Check if the record exists in the local directory
		_, err := w.store.Lookup(ctx, &coretypes.ObjectRef{
			Digest: cid,
		})
		if err != nil {
			st := status.Convert(err)
			if st.Code() == codes.NotFound {
				logger.Debug("Record not found locally, syncing", "worker_id", w.id, "sync_id", item.SyncID, "record_cid", cid)

				if err := w.syncRecord(ctx, remoteClient, cid); err != nil {
					return fmt.Errorf("failed to sync record %s: %w", cid, err)
				}

				continue
			}

			return fmt.Errorf("failed to lookup record: %w", err)
		}

		logger.Debug("Record found locally, skipping", "worker_id", w.id, "sync_id", item.SyncID, "record_cid", cid)
	}

	logger.Debug("Sync operation completed", "worker_id", w.id, "sync_id", item.SyncID)

	return nil
}

// searchAllRemoteRecords retrieves all records from the remote directory with pagination support.
func (w *Worker) searchAllRemoteRecords(ctx context.Context, remoteClient *client.Client) ([]string, error) {
	var records []string

	offset := uint32(0)
	limit := uint32(200) //nolint:mnd

	for {
		// Search for records in the remote directory
		ch, err := remoteClient.Search(ctx, &searchtypes.SearchRequest{
			Queries: []*searchtypes.RecordQuery{},
			Limit:   &limit,
			Offset:  &offset,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search: %w", err)
		}

		// Collect record from this batch
		var batchCids []string

		for recordCid := range ch {
			if recordCid == "" {
				continue
			}

			batchCids = append(batchCids, recordCid)
		}

		// If batch is less than limit, end loop
		if len(batchCids) < int(limit) {
			records = append(records, batchCids...)

			break
		}

		// Add this batch to our collection and continue to next page
		records = append(records, batchCids...)
		offset += limit
	}

	return records, nil
}

// syncRecord handles the synchronization of a single record from remote to local store.
func (w *Worker) syncRecord(ctx context.Context, remoteClient *client.Client, recordCid string) error {
	logger.Debug("Syncing record", "worker_id", w.id, "record_cid", recordCid)

	// Lookup the record in the remote directory
	ref, err := remoteClient.Lookup(ctx, &coretypes.ObjectRef{
		Digest: recordCid,
	})
	if err != nil {
		return fmt.Errorf("failed to lookup record: %w", err)
	}

	// Pull the record from the remote directory
	reader, err := remoteClient.Pull(ctx, ref)
	if err != nil {
		return fmt.Errorf("failed to pull record: %w", err)
	}

	// Read all data into memory so we can use it twice
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read record data: %w", err)
	}

	// Push the record to the local directory
	dataReader := bytes.NewReader(data)

	_, err = w.store.Push(ctx, ref, dataReader)
	if err != nil {
		return fmt.Errorf("failed to push record: %w", err)
	}

	// Load agent from data for search index
	agent := &coretypes.Agent{}
	dataReader2 := bytes.NewReader(data)

	if _, err := agent.LoadFromReader(dataReader2); err != nil {
		return fmt.Errorf("failed to load agent from reader: %w", err)
	}

	// Add record to search index
	err = w.db.AddRecord(v1alpha1.NewAgentAdapter(agent, ref.GetDigest()))
	if err != nil {
		return fmt.Errorf("failed to add agent to search index: %w", err)
	}

	logger.Debug("Successfully synced record", "worker_id", w.id, "record_cid", recordCid)

	return nil
}

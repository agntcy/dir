// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	publypes "github.com/agntcy/dir/server/publication/types"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
)

// Worker processes publication requests from the work queue.
type Worker struct {
	id        int
	db        types.DatabaseAPI
	store     types.StoreAPI
	routing   types.RoutingAPI
	workQueue <-chan publypes.WorkItem
	timeout   time.Duration
}

// NewWorker creates a new worker instance.
func NewWorker(id int, db types.DatabaseAPI, store types.StoreAPI, routing types.RoutingAPI, workQueue <-chan publypes.WorkItem, timeout time.Duration) *Worker {
	return &Worker{
		id:        id,
		db:        db,
		store:     store,
		routing:   routing,
		workQueue: workQueue,
		timeout:   timeout,
	}
}

// Run starts the worker loop.
func (w *Worker) Run(ctx context.Context, stopCh <-chan struct{}) {
	logger.Info("Starting publication worker", "worker_id", w.id)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping due to context cancellation", "worker_id", w.id)

			return
		case <-stopCh:
			logger.Info("Worker stopping due to stop signal", "worker_id", w.id)

			return
		case workItem := <-w.workQueue:
			w.processPublication(ctx, workItem)
		}
	}
}

// processPublication processes a single publication request.
func (w *Worker) processPublication(ctx context.Context, workItem publypes.WorkItem) {
	logger.Info("Processing publication", "worker_id", w.id, "publication_id", workItem.PublicationID)

	// Create a timeout context for this operation
	timeoutCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// Get the publication from database
	publicationObj, err := w.db.GetPublicationByID(workItem.PublicationID)
	if err != nil {
		logger.Error("Failed to get publication", "publication_id", workItem.PublicationID, "error", err)
		w.markPublicationFailed(workItem.PublicationID)

		return
	}

	request := publicationObj.GetRequest()
	if request == nil {
		logger.Error("Publication has no request", "publication_id", workItem.PublicationID)
		w.markPublicationFailed(workItem.PublicationID)

		return
	}

	// Get CIDs to publish based on the request type
	cids, err := w.getCIDsFromRequest(timeoutCtx, request)
	if err != nil {
		logger.Error("Failed to get CIDs from request", "publication_id", workItem.PublicationID, "error", err)
		w.markPublicationFailed(workItem.PublicationID)

		return
	}

	if len(cids) == 0 {
		logger.Info("No CIDs found to publish", "publication_id", workItem.PublicationID)
		w.markPublicationCompleted(workItem.PublicationID)

		return
	}

	// Announce each CID to the DHT
	successCount := 0

	for _, cid := range cids {
		if err := w.announceToDHT(timeoutCtx, cid); err != nil {
			logger.Error("Failed to announce CID to DHT", "publication_id", workItem.PublicationID, "cid", cid, "error", err)
		} else {
			successCount++

			logger.Debug("Successfully announced CID to DHT", "publication_id", workItem.PublicationID, "cid", cid)
		}
	}

	logger.Info("Publication processing completed", "worker_id", w.id, "publication_id", workItem.PublicationID,
		"total_cids", len(cids), "successful_announcements", successCount)

	// Mark as completed if we announced all CIDs successfully
	if successCount == len(cids) {
		w.markPublicationCompleted(workItem.PublicationID)
	} else {
		w.markPublicationFailed(workItem.PublicationID)
	}
}

// getCIDsFromRequest extracts CIDs from the publication request based on its type.
func (w *Worker) getCIDsFromRequest(_ context.Context, request *routingv1.PublishRequest) ([]string, error) {
	switch req := request.GetRequest().(type) {
	case *routingv1.PublishRequest_RecordRefs:
		// Direct CID references
		var cids []string
		for _, ref := range req.RecordRefs.GetRefs() {
			cids = append(cids, ref.GetCid())
		}

		return cids, nil

	case *routingv1.PublishRequest_Queries:
		// Convert search query to database filter options
		filterOpts, err := databaseutils.QueryToFilters(req.Queries.GetQueries())
		if err != nil {
			return nil, fmt.Errorf("failed to convert query to filter options: %w", err)
		}

		// Get CIDs using the filter options
		return w.db.GetRecordCIDs(filterOpts...) //nolint:wrapcheck

	default:
		return nil, errors.New("unknown request type")
	}
}

// announceToDHT announces a single CID to the DHT.
func (w *Worker) announceToDHT(ctx context.Context, cid string) error {
	// Create a RecordRef for the CID
	recordRef := &corev1.RecordRef{
		Cid: cid,
	}

	// Pull the record from the store
	record, err := w.store.Pull(ctx, recordRef)
	if err != nil {
		return fmt.Errorf("failed to pull record from store: %w", err)
	}

	// Wrap record with adapter for interface-based publishing
	adapter := adapters.NewRecordAdapter(record)

	// Publish the record to the network
	err = w.routing.Publish(ctx, adapter)
	if err != nil {
		return fmt.Errorf("failed to publish record to network: %w", err)
	}

	return nil
}

// markPublicationCompleted marks a publication as completed.
func (w *Worker) markPublicationCompleted(publicationID string) {
	if err := w.db.UpdatePublicationStatus(publicationID, routingv1.PublicationStatus_PUBLICATION_STATUS_COMPLETED); err != nil {
		logger.Error("Failed to mark publication as completed", "publication_id", publicationID, "error", err)
	}
}

// markPublicationFailed marks a publication as failed.
func (w *Worker) markPublicationFailed(publicationID string) {
	if err := w.db.UpdatePublicationStatus(publicationID, routingv1.PublicationStatus_PUBLICATION_STATUS_FAILED); err != nil {
		logger.Error("Failed to mark publication as failed", "publication_id", publicationID, "error", err)
	}
}

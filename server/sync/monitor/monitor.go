// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/store/oci"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/sync/monitor/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/zot"
	"github.com/agntcy/oasf-sdk/validation/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2/registry/remote"
)

var logger = logging.Logger("sync/monitor")

// MonitorService manages registry monitoring based on active sync operations.
//
//nolint:revive
type MonitorService struct {
	// Configuration
	db            types.DatabaseAPI
	store         types.StoreAPI
	ociConfig     ociconfig.Config
	checkInterval time.Duration

	// Monitoring state
	mu            sync.RWMutex
	isRunning     bool
	lastSnapshot  *RegistrySnapshot
	ticker        *time.Ticker
	cancelMonitor context.CancelFunc

	// Sync management
	activeSyncs map[string]struct{} // Track active sync operations

	// ORAS repository client
	repo *remote.Repository

	// Validation
	validator *service.ValidationService
}

// NewMonitorService creates a new monitor service.
func NewMonitorService(db types.DatabaseAPI, store types.StoreAPI, ociConfig ociconfig.Config, monitorConfig config.Config) (*MonitorService, error) {
	// Create ORAS repository client
	repo, err := oci.NewORASRepository(ociConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ORAS repository client: %w", err)
	}

	validator, err := service.NewValidationService()
	if err != nil {
		return nil, fmt.Errorf("failed to create validation service: %w", err)
	}

	return &MonitorService{
		db:            db,
		store:         store,
		ociConfig:     ociConfig,
		checkInterval: monitorConfig.CheckInterval,
		activeSyncs:   make(map[string]struct{}),
		repo:          repo,
		validator:     validator,
	}, nil
}

// Stop gracefully shuts down the monitor service.
func (s *MonitorService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Info("Stopping monitor service")

	// Stop monitoring if active
	if s.isRunning {
		logger.Info("Stopping registry monitoring")

		// Cancel monitoring
		if s.cancelMonitor != nil {
			s.cancelMonitor()
		}

		s.ticker.Stop()

		// Update state
		s.isRunning = false
	}

	// Clear active syncs
	s.activeSyncs = make(map[string]struct{})

	logger.Info("Monitor service stopped")

	return nil
}

// StartSyncMonitoring begins monitoring when a sync operation starts.
func (s *MonitorService) StartSyncMonitoring(syncID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add sync to active list
	s.activeSyncs[syncID] = struct{}{}

	// Start monitoring if this is the first active sync
	if len(s.activeSyncs) == 1 && !s.isRunning {
		s.startMonitoring(context.Background())

		logger.Info("Started registry monitoring", "active_syncs", len(s.activeSyncs))
	}

	logger.Debug("Sync added to monitoring", "sync_id", syncID, "active_syncs", len(s.activeSyncs))

	return nil
}

// StopSyncMonitoring stops monitoring when a sync operation ends.
// It performs a final indexing scan before stopping to ensure no records are missed.
func (s *MonitorService) StopSyncMonitoring(syncID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove sync from active list
	delete(s.activeSyncs, syncID)

	// Stop monitoring if no more active syncs
	if len(s.activeSyncs) == 0 && s.isRunning {
		// Cancel monitoring
		if s.cancelMonitor != nil {
			s.cancelMonitor()
		}

		// Update state
		s.isRunning = false

		// Run graceful shutdown in background to not block deletion
		go s.gracefulShutdown()

		s.ticker.Stop()

		logger.Info("Stopped registry monitoring")
	}

	logger.Debug("Sync removed from monitoring", "sync_id", syncID, "active_syncs", len(s.activeSyncs))

	return nil
}

// gracefulShutdown performs final monitoring checks in the background
// to ensure all synced records are indexed before stopping monitoring.
func (s *MonitorService) gracefulShutdown() {
	logger.Debug("Starting graceful shutdown with final monitoring checks")

	// Perform final indexing scan
	ctx, cancel := context.WithTimeout(context.Background(), config.DefaultCheckInterval*2) //nolint:mnd
	defer cancel()

	// Create a separate ticker for graceful shutdown since main ticker is stopped
	shutdownTicker := time.NewTicker(config.DefaultCheckInterval) //nolint:mnd
	defer shutdownTicker.Stop()

	// Perform monitoring checks until timeout is reached
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Graceful shutdown timeout reached")

			return
		case <-shutdownTicker.C:
			s.performMonitoringCheck(ctx)
		}
	}
}

// startMonitoring begins registry monitoring.
func (s *MonitorService) startMonitoring(ctx context.Context) {
	if s.isRunning {
		logger.Debug("Registry monitoring already running")

		return
	}

	// Initialize monitoring state
	s.isRunning = true
	s.ticker = time.NewTicker(s.checkInterval)

	// Create cancelable context for monitoring
	monitorCtx, cancel := context.WithCancel(ctx)
	s.cancelMonitor = cancel

	// Start monitoring goroutine
	go s.runRegistryMonitoring(monitorCtx)
}

// runRegistryMonitoring runs the registry monitoring loop.
func (s *MonitorService) runRegistryMonitoring(ctx context.Context) {
	logger.Info("Registry monitoring started")

	// Create initial snapshot
	snapshot, err := s.createRegistrySnapshot(ctx)
	if err != nil {
		logger.Error("Failed to create initial registry snapshot", "error", err)

		return
	}

	s.lastSnapshot = snapshot

	for {
		select {
		case <-ctx.Done():
			logger.Info("Registry monitoring stopping")

			return
		case <-s.ticker.C:
			s.performMonitoringCheck(ctx)
		}
	}
}

// performMonitoringCheck performs a single monitoring check.
func (s *MonitorService) performMonitoringCheck(ctx context.Context) {
	logger.Debug("Performing registry monitoring check")

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current registry snapshot
	snapshot, err := s.createRegistrySnapshot(ctx)
	if err != nil {
		logger.Error("Failed to create registry snapshot", "error", err)

		return
	}

	// Compare with last snapshot to detect changes
	changes := s.detectChanges(s.lastSnapshot, snapshot)
	if changes.HasChanges {
		logger.Info("Registry changes detected", "new_tags", len(changes.NewTags))
		s.processChanges(ctx, changes)
	} else {
		logger.Debug("No registry changes detected")
	}

	// Update last snapshot
	s.lastSnapshot = snapshot
}

// createRegistrySnapshot creates a snapshot of the current registry state.
func (s *MonitorService) createRegistrySnapshot(ctx context.Context) (*RegistrySnapshot, error) {
	// List all tags in the repository
	// No sorting needed, OCI spec requires tags in lexical order
	var tags []string

	err := s.repo.Tags(ctx, "", func(tagDescriptors []string) error {
		// Filter tags to only include valid CIDs
		for _, tag := range tagDescriptors {
			if corev1.IsValidCID(tag) {
				tags = append(tags, tag)
			} else {
				logger.Debug("Skipping non-CID tag", "tag", tag)
			}
		}

		return nil
	})
	if err != nil {
		// Check if this is a "repository not found" error (404)
		if isRepositoryNotFoundError(err) {
			logger.Debug("Repository not found yet, returning empty snapshot", "error", err)

			// Return empty snapshot - this is normal when sync hasn't pulled content yet
			return EmptySnapshot, nil
		}

		return nil, fmt.Errorf("failed to list repository tags: %w", err)
	}

	// Create content hash for quick comparison
	contentHash := s.createContentHash(tags)

	return &RegistrySnapshot{
		Timestamp:    time.Now(),
		Tags:         tags,
		ContentHash:  contentHash,
		LastModified: time.Now(),
	}, nil
}

// createContentHash creates a hash of the tags for quick comparison.
func (s *MonitorService) createContentHash(tags []string) string {
	// Create a deterministic hash by writing each tag individually
	hasher := sha256.New()

	for _, tag := range tags {
		hasher.Write([]byte(tag))
		hasher.Write([]byte("|"))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// detectChanges compares two registry snapshots and detects changes.
func (s *MonitorService) detectChanges(oldSnapshot, newSnapshot *RegistrySnapshot) *RegistryChanges {
	// If content hashes match, no changes
	if oldSnapshot.ContentHash == newSnapshot.ContentHash {
		return &RegistryChanges{
			HasChanges: false,
			DetectedAt: time.Now(),
		}
	}

	// Create sets for efficient comparison
	oldTags := make(map[string]struct{})
	for _, tag := range oldSnapshot.Tags {
		oldTags[tag] = struct{}{}
	}

	// Find new tags
	var addedTags []string

	for _, tag := range newSnapshot.Tags {
		if _, ok := oldTags[tag]; !ok {
			addedTags = append(addedTags, tag)
		}
	}

	changes := &RegistryChanges{
		NewTags:    addedTags,
		HasChanges: len(addedTags) > 0,
		DetectedAt: time.Now(),
	}

	return changes
}

// processChanges processes detected registry changes by indexing new records.
func (s *MonitorService) processChanges(ctx context.Context, changes *RegistryChanges) {
	for _, tag := range changes.NewTags {
		// Index record
		if err := s.indexRecord(ctx, tag); err != nil {
			// Warn but continue processing other records even if one fails
			logger.Error("Failed to index record", "tag", tag, "error", err)
		} else {
			logger.Debug("Successfully indexed record", "tag", tag)
		}

		// Upload public key to OCI store
		if err := s.uploadPublicKey(ctx, tag); err != nil {
			logger.Error("Failed to upload public key", "tag", tag, "error", err)
		}
	}
}

// indexRecord indexes a single record from the registry into the database.
func (s *MonitorService) indexRecord(ctx context.Context, tag string) error {
	logger.Debug("Indexing record", "tag", tag)

	// Pull record from local store
	recordRef := &corev1.RecordRef{Cid: tag}

	record, err := s.store.Pull(ctx, recordRef)
	if err != nil {
		return fmt.Errorf("failed to pull record from local store: %w", err)
	}

	if recordV3 := record.GetV3(); recordV3 != nil {
		req := &validationv1.ValidateRecordRequest{
			Record: recordV3,
		}

		isValid, errors, err := s.validator.ValidateRecord(req)
		if err != nil {
			logger.Error("Failed to validate record", "error", err)

			return status.Errorf(codes.Internal, "failed to validate record: %v", err)
		}

		if !isValid {
			logger.Warn("Record validation failed", "errors", errors)

			return status.Errorf(codes.InvalidArgument, "record validation failed: %v", errors)
		}
	}

	// Add to database
	recordAdapter := adapters.NewRecordAdapter(record)
	if err := s.db.AddRecord(recordAdapter); err != nil {
		// Check if this is a duplicate record error - if so, it's not really an error
		if s.isDuplicateRecordError(err) {
			logger.Debug("Record already indexed, skipping", "cid", tag)

			return nil
		}

		return fmt.Errorf("failed to add record to database: %w", err)
	}

	logger.Info("Successfully indexed local record", "cid", tag)

	return nil
}

// uploadPublicKey uploads a public key to the OCI store.
func (s *MonitorService) uploadPublicKey(ctx context.Context, tag string) error {
	logger.Debug("Uploading public key", "tag", tag)

	// Try to use signature storage if the store supports it
	ociStore, ok := s.store.(interface {
		PullPublicKey(context.Context, string) (string, error)
	})
	if !ok {
		logger.Error("Store does not support public key upload, skipping", "tag", tag)

		return nil
	}

	// Pull public key from OCI store
	publicKey, err := ociStore.PullPublicKey(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to pull public key: %w", err)
	}

	// Upload the public key to zot for signature verification
	// This enables zot to mark this signature as "trusted" in verification queries
	uploadOpts := &zot.UploadPublicKeyOptions{
		Config: &zot.VerifyConfig{
			RegistryAddress: s.ociConfig.RegistryAddress,
			RepositoryName:  s.ociConfig.RepositoryName,
			Username:        s.ociConfig.Username,
			Password:        s.ociConfig.Password,
			AccessToken:     s.ociConfig.AccessToken,
			Insecure:        s.ociConfig.Insecure,
		},
		PublicKey: publicKey,
	}

	err = zot.UploadPublicKey(ctx, uploadOpts)
	if err != nil {
		return fmt.Errorf("failed to upload public key to zot for verification: %w", err)
	}

	logger.Debug("Successfully uploaded public key to zot for verification", "tag", tag)

	return nil
}

// isDuplicateRecordError checks if the error indicates a duplicate record.
func (s *MonitorService) isDuplicateRecordError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	return strings.Contains(errStr, "duplicate") ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "unique constraint") ||
		strings.Contains(errStr, "primary key")
}

// isRepositoryNotFoundError checks if the error is a "repository not found" (404) error.
func isRepositoryNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's an HTTP error with status code 404
	errStr := err.Error()

	return strings.Contains(errStr, "404") &&
		(strings.Contains(errStr, "name unknown") ||
			strings.Contains(errStr, "repository name not known") ||
			strings.Contains(errStr, "not found"))
}

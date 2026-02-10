// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package indexer implements the reconciliation task for indexing records.
// It monitors the local OCI registry for new records and indexes them into
// the database to enable search functionality.
package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"oras.land/oras-go/v2/registry/remote"
)

var logger = logging.Logger("reconciler/indexer")

// Task implements the indexer reconciliation task.
// It scans the local OCI registry for unindexed records and adds them to the database.
type Task struct {
	config    Config
	db        types.SearchDatabaseAPI
	store     types.StoreAPI
	ociConfig ociconfig.Config
	repo      *remote.Repository

	mu           sync.Mutex
	lastSnapshot *registrySnapshot
}

// registrySnapshot represents a point-in-time view of registry tags.
type registrySnapshot struct {
	Timestamp   time.Time
	Tags        []string
	ContentHash string
}

// emptySnapshot is an empty snapshot for initialization.
var emptySnapshot = &registrySnapshot{
	Tags:        []string{},
	ContentHash: "",
}

// NewTask creates a new indexer reconciliation task.
func NewTask(config Config, localRegistry ociconfig.Config, store types.StoreAPI, repo *remote.Repository, db types.SearchDatabaseAPI) (*Task, error) {
	return &Task{
		config:       config,
		db:           db,
		store:        store,
		ociConfig:    localRegistry,
		repo:         repo,
		lastSnapshot: emptySnapshot,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "indexer"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run executes the reconciliation logic.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running indexer reconciliation")

	t.mu.Lock()
	defer t.mu.Unlock()

	// Get current registry snapshot
	snapshot, err := t.createRegistrySnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to create registry snapshot: %w", err)
	}

	// Compare with last snapshot to detect new tags
	newTags := t.detectNewTags(t.lastSnapshot, snapshot)
	if len(newTags) == 0 {
		logger.Debug("No new records to index")

		t.lastSnapshot = snapshot

		return nil
	}

	logger.Info("Found new records to index", "count", len(newTags))

	// Process each new tag
	var indexedCount, failedCount int

	for _, tag := range newTags {
		if err := t.indexRecord(ctx, tag); err != nil {
			logger.Error("Failed to index record", "tag", tag, "error", err)

			failedCount++

			continue
		}

		indexedCount++
	}

	logger.Info("Indexing complete", "indexed", indexedCount, "failed", failedCount)

	// Update last snapshot
	t.lastSnapshot = snapshot

	return nil
}

// createRegistrySnapshot creates a snapshot of the current registry state.
func (t *Task) createRegistrySnapshot(ctx context.Context) (*registrySnapshot, error) {
	var tags []string

	err := t.repo.Tags(ctx, "", func(tagDescriptors []string) error {
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

			return emptySnapshot, nil
		}

		return nil, fmt.Errorf("failed to list repository tags: %w", err)
	}

	// Create content hash for quick comparison
	contentHash := createContentHash(tags)

	return &registrySnapshot{
		Timestamp:   time.Now(),
		Tags:        tags,
		ContentHash: contentHash,
	}, nil
}

// createContentHash creates a hash of the tags for quick comparison.
func createContentHash(tags []string) string {
	hasher := sha256.New()

	for _, tag := range tags {
		hasher.Write([]byte(tag))
		hasher.Write([]byte("|"))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// detectNewTags compares two snapshots and returns new tags.
func (t *Task) detectNewTags(oldSnapshot, newSnapshot *registrySnapshot) []string {
	// If content hashes match, no changes
	if oldSnapshot.ContentHash == newSnapshot.ContentHash {
		return nil
	}

	// Create set of old tags for efficient lookup
	oldTags := make(map[string]struct{})
	for _, tag := range oldSnapshot.Tags {
		oldTags[tag] = struct{}{}
	}

	// Find new tags
	var newTags []string

	for _, tag := range newSnapshot.Tags {
		if _, ok := oldTags[tag]; !ok {
			newTags = append(newTags, tag)
		}
	}

	return newTags
}

// indexRecord indexes a single record from the registry into the database.
func (t *Task) indexRecord(ctx context.Context, tag string) error {
	logger.Debug("Indexing record", "tag", tag)

	// Pull record from local store
	recordRef := &corev1.RecordRef{Cid: tag}

	record, err := t.store.Pull(ctx, recordRef)
	if err != nil {
		return fmt.Errorf("failed to pull record from local store: %w", err)
	}

	// Validate record
	isValid, validationErrors, err := record.Validate(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate record: %w", err)
	}

	if !isValid {
		return fmt.Errorf("record validation failed: %v", validationErrors)
	}

	// Add to database
	recordAdapter := adapters.NewRecordAdapter(record)
	if err := t.db.AddRecord(recordAdapter); err != nil {
		// Check if this is a duplicate record error - if so, it's not really an error
		if isDuplicateRecordError(err) {
			logger.Debug("Record already indexed, skipping", "cid", tag)

			return nil
		}

		return fmt.Errorf("failed to add record to database: %w", err)
	}

	logger.Info("Successfully indexed record", "cid", tag)

	return nil
}

// isDuplicateRecordError checks if the error indicates a duplicate record.
func isDuplicateRecordError(err error) bool {
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

	errStr := err.Error()

	return strings.Contains(errStr, "404") &&
		(strings.Contains(errStr, "name unknown") ||
			strings.Contains(errStr, "repository name not known") ||
			strings.Contains(errStr, "not found"))
}

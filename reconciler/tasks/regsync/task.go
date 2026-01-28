// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package regsync implements the reconciliation task for regsync configuration.
// It monitors the database for pending sync operations and runs the regsync
// binary to synchronize images from non-Zot registries.
package regsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	storev1 "github.com/agntcy/dir/api/store/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	serversync "github.com/agntcy/dir/server/sync"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

const (
	// configDirPermissions is the permission mode for creating the config directory.
	configDirPermissions = 0o755
)

var logger = logging.Logger("reconciler/regsync")

// Task implements the regsync reconciliation task.
// It checks the database for pending syncs and runs the regsync binary.
type Task struct {
	config        Config
	localRegistry ociconfig.Config
	db            types.SyncDatabaseAPI
	regsyncConfig *RegsyncConfig

	mu            sync.Mutex
	currentSyncID string // ID of sync currently being processed
}

// NewTask creates a new regsync reconciliation task.
func NewTask(config Config, localRegistry ociconfig.Config, db types.SyncDatabaseAPI) (*Task, error) {
	// Ensure config directory exists
	configDir := filepath.Dir(config.GetConfigPath())
	if err := os.MkdirAll(configDir, configDirPermissions); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Initialize regsync config with sensible defaults
	regsyncConfig := NewRegsyncConfig()

	// Add local registry credential
	localHost := trimScheme(localRegistry.RegistryAddress)
	regsyncConfig.AddCredential(
		localHost,
		localRegistry.Username,
		localRegistry.Password,
		localRegistry.Insecure,
	)

	return &Task{
		config:        config,
		localRegistry: localRegistry,
		db:            db,
		regsyncConfig: regsyncConfig,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "regsync"
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
	logger.Debug("Running regsync reconciliation")

	// Check if we're currently processing a sync
	if t.isProcessing() {
		logger.Debug("Sync currently in progress, skipping new syncs", "sync_id", t.currentSyncID)

		return nil
	}

	// Find pending syncs that need regsync
	pendingSyncs, err := t.db.GetRegsyncSyncsByStatus(storev1.SyncStatus_SYNC_STATUS_PENDING)
	if err != nil {
		return fmt.Errorf("failed to get pending syncs: %w", err)
	}

	// Process the first pending sync that needs regsync
	for _, syncObj := range pendingSyncs {
		if err := t.processSyncEntry(ctx, syncObj); err != nil {
			logger.Error("Failed to process sync", "sync_id", syncObj.GetID(), "error", err)

			continue
		}

		// Only process one sync at a time
		break
	}

	return nil
}

// processSyncEntry processes a single sync entry by running the regsync binary.
func (t *Task) processSyncEntry(ctx context.Context, syncObj types.SyncObject) error {
	syncID := syncObj.GetID()
	remoteDirectoryURL := syncObj.GetRemoteDirectoryURL()

	logger.Info("Processing sync entry", "sync_id", syncID, "remote_directory", remoteDirectoryURL)

	// Mark as current sync
	t.mu.Lock()
	t.currentSyncID = syncID
	t.mu.Unlock()

	// Update status to IN_PROGRESS
	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_IN_PROGRESS); err != nil {
		t.clearCurrentSync()

		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Negotiate credentials with the remote Directory node
	credentials, err := serversync.NegotiateCredentials(ctx, remoteDirectoryURL, t.config.Authn)
	if err != nil {
		t.markSyncFailed(syncID, "credential negotiation failed")

		return fmt.Errorf("failed to negotiate credentials: %w", err)
	}

	logger.Debug("Credentials negotiated successfully", "sync_id", syncID, "registry", credentials.RegistryAddress, "repository", credentials.RepositoryName)

	// Add credentials for the remote registry
	t.regsyncConfig.AddCredential(
		credentials.RegistryAddress,
		credentials.Credentials.Username,
		credentials.Credentials.Password,
		credentials.Credentials.Insecure,
	)

	t.regsyncConfig.ClearSyncs()
	t.regsyncConfig.AddSync(
		credentials.FullRepositoryURL(),
		t.buildTargetURL(),
		syncObj.GetCIDs(),
	)

	// Write config file
	if err := t.regsyncConfig.WriteToFile(t.config.GetConfigPath(), syncID); err != nil {
		t.markSyncFailed(syncID, "failed to write config")

		return fmt.Errorf("failed to write regsync config: %w", err)
	}

	logger.Info("Running regsync command", "sync_id", syncID, "source", credentials.FullRepositoryURL(), "target", t.buildTargetURL())

	// Run the regsync binary and wait for completion
	if err := t.runRegsync(ctx, syncID); err != nil {
		t.markSyncFailed(syncID, err.Error())

		return fmt.Errorf("regsync command failed: %w", err)
	}

	// Mark sync as completed
	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_COMPLETED); err != nil {
		logger.Error("Failed to update sync status to COMPLETED", "sync_id", syncID, "error", err)
	}

	logger.Info("Sync completed successfully", "sync_id", syncID)
	t.clearCurrentSync()

	return nil
}

// runRegsync executes the regsync binary with the current configuration.
func (t *Task) runRegsync(ctx context.Context, syncID string) error {
	// Create a context with timeout
	timeout := t.config.GetTimeout()

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build the command: regsync once -c <config_path>
	binaryPath := t.config.GetBinaryPath()
	configPath := t.config.GetConfigPath()

	//nolint:gosec // Binary path is from trusted configuration
	cmd := exec.CommandContext(execCtx, binaryPath, "once", "-c", configPath)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.Debug("Executing regsync command", "sync_id", syncID, "command", cmd.String(), "timeout", timeout)

	// Run the command
	err := cmd.Run()

	// Log output regardless of success/failure
	if stdout.Len() > 0 {
		logger.Debug("regsync stdout", "sync_id", syncID, "output", stdout.String())
	}

	if stderr.Len() > 0 {
		logger.Debug("regsync stderr", "sync_id", syncID, "output", stderr.String())
	}

	if err != nil {
		// Check if it was a timeout
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("regsync command timed out after %v", timeout)
		}

		return fmt.Errorf("regsync command failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// markSyncFailed marks a sync as failed and clears the current sync.
func (t *Task) markSyncFailed(syncID, reason string) {
	logger.Error("Sync failed", "sync_id", syncID, "reason", reason)

	if err := t.db.UpdateSyncStatus(syncID, storev1.SyncStatus_SYNC_STATUS_FAILED); err != nil {
		logger.Error("Failed to update sync status to FAILED", "sync_id", syncID, "error", err)
	}

	t.clearCurrentSync()
}

// clearCurrentSync clears the current sync ID.
func (t *Task) clearCurrentSync() {
	t.mu.Lock()
	t.currentSyncID = ""
	t.mu.Unlock()
}

// isProcessing checks if there's currently a sync being processed.
func (t *Task) isProcessing() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.currentSyncID == "" {
		return false
	}

	// Double-check by querying the database
	syncObj, err := t.db.GetSyncByID(t.currentSyncID)
	if err != nil {
		logger.Debug("Failed to get current sync status, assuming still in progress",
			"sync_id", t.currentSyncID, "error", err)

		return true
	}

	status := syncObj.GetStatus()
	if status == storev1.SyncStatus_SYNC_STATUS_COMPLETED ||
		status == storev1.SyncStatus_SYNC_STATUS_FAILED ||
		status == storev1.SyncStatus_SYNC_STATUS_DELETED {
		// Current sync is done, clear it
		logger.Debug("Previous sync finished, clearing", "sync_id", t.currentSyncID, "status", status)
		t.currentSyncID = ""

		return false
	}

	return true
}

// buildTargetURL builds the target registry URL for syncs.
func (t *Task) buildTargetURL() string {
	localHost := trimScheme(t.localRegistry.RegistryAddress)

	if t.localRegistry.RepositoryName != "" {
		return fmt.Sprintf("%s/%s", localHost, t.localRegistry.RepositoryName)
	}

	return localHost
}

// Initialize writes the initial regsync configuration file.
func (t *Task) Initialize() error {
	logger.Info("Initializing regsync configuration", "config_path", t.config.GetConfigPath())

	if err := t.regsyncConfig.WriteToFile(t.config.GetConfigPath(), ""); err != nil {
		return fmt.Errorf("failed to write initial regsync config: %w", err)
	}

	return nil
}

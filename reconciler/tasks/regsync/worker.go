// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
)

// Worker processes a single sync request atomically.
// Each worker has its own isolated configuration to ensure
// sync operations don't interfere with each other.
// Worker does not have direct database access - it reports results
// back to the task which handles database updates.
type Worker struct {
	config        Config
	localRegistry ociconfig.Config
	syncID        string
	syncObj       types.SyncObject
}

// NewWorker creates a new worker to process a single sync request.
func NewWorker(config Config, localRegistry ociconfig.Config, syncObj types.SyncObject) *Worker {
	return &Worker{
		config:        config,
		localRegistry: localRegistry,
		syncID:        syncObj.GetID(),
		syncObj:       syncObj,
	}
}

// Run executes the sync operation for this worker.
// Returns a WorkerResult indicating success or failure.
// The caller (task) is responsible for updating the database based on the result.
func (w *Worker) Run(ctx context.Context) error {
	remoteDirectoryURL := w.syncObj.GetRemoteDirectoryURL()

	logger.Info("Executing sync", "sync_id", w.syncID, "remote_directory", remoteDirectoryURL)

	// Negotiate credentials with the remote Directory node
	credentials, err := NegotiateCredentials(ctx, remoteDirectoryURL, w.config.Authn)
	if err != nil {
		return fmt.Errorf("failed to negotiate credentials: %w", err)
	}

	logger.Debug("Credentials negotiated successfully",
		"sync_id", w.syncID,
		"registry", credentials.RegistryAddress,
		"repository", credentials.RepositoryName,
	)

	// Create isolated regsync config for this worker
	regsyncConfig := NewRegsyncConfig()

	// Add local registry credential
	regsyncConfig.AddCredential(
		w.localRegistry.RegistryAddress,
		w.localRegistry.Username,
		w.localRegistry.Password,
		w.localRegistry.Insecure,
	)

	// Add credentials for the remote registry
	regsyncConfig.AddCredential(
		credentials.RegistryAddress,
		credentials.Credentials.Username,
		credentials.Credentials.Password,
		credentials.Credentials.Insecure,
	)

	// Configure the sync entry
	regsyncConfig.AddSync(
		credentials.FullRepositoryURL(),
		w.localRegistry.GetRepositoryURL(),
		w.syncObj.GetCIDs(),
	)

	// Create config file
	configPath, err := regsyncConfig.WriteToFile(w.syncID)
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	defer os.Remove(configPath) // Ensure config file is cleaned up after execution

	// Run the regsync command with the generated config
	logger.Info("Running regsync command",
		"sync_id", w.syncID,
		"source", credentials.FullRepositoryURL(),
		"target", w.localRegistry.GetRepositoryURL(),
		"config_path", configPath,
	)

	// Run the regsync binary
	if err := w.runRegsync(ctx, configPath); err != nil {
		return fmt.Errorf("regsync command failed: %w", err)
	}

	return nil
}

// runRegsync executes the regsync binary with the worker's configuration.
func (w *Worker) runRegsync(ctx context.Context, configPath string) error {
	// Create a context with timeout
	timeout := w.config.GetTimeout()

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build the command: regsync once -c <config_path>
	binaryPath := w.config.GetBinaryPath()

	//nolint:gosec // Binary path is from trusted configuration
	cmd := exec.CommandContext(execCtx, binaryPath, "once", "-c", configPath)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	logger.Debug("Executing regsync command",
		"sync_id", w.syncID,
		"command", cmd.String(),
		"timeout", timeout,
	)

	// Run the command
	err := cmd.Run()

	// Log output regardless of success/failure
	if stdout.Len() > 0 {
		logger.Debug("regsync stdout", "sync_id", w.syncID, "output", stdout.String())
	}

	if stderr.Len() > 0 {
		logger.Debug("regsync stderr", "sync_id", w.syncID, "output", stderr.String())
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

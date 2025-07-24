// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"errors"
	"fmt"

	storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "sync",
	Short: "Manage synchronization operations with remote Directory nodes",
	Long: `Sync command allows you to manage synchronization operations between Directory nodes.
It provides subcommands to create, list, monitor, and delete sync operations.`,
}

// Create sync subcommand.
var createCmd = &cobra.Command{
	Use:   "create <remote-directory-url>",
	Short: "Create a new synchronization operation",
	Long: `Create initiates a new synchronization operation from a remote Directory node.
The operation is asynchronous and returns a sync ID for tracking progress.

Examples:
  dir sync create https://directory.example.com
  dir sync create http://localhost:8080`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCreateSync(cmd, args[0])
	},
}

// List syncs subcommand.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all synchronization operations",
	Long: `List displays all sync operations known to the system, including active, 
completed, and failed synchronizations.

Pagination can be controlled using --limit and --offset flags:
  dir sync list --limit 10 --offset 20`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runListSyncs(cmd)
	},
}

// Get sync status subcommand.
var statusCmd = &cobra.Command{
	Use:   "status <sync-id>",
	Short: "Get detailed status of a synchronization operation",
	Long: `Status retrieves comprehensive information about a specific sync operation,
including progress, timing, and error details if applicable.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGetSyncStatus(cmd, args[0])
	},
}

// Delete sync subcommand.
var deleteCmd = &cobra.Command{
	Use:   "delete <sync-id>",
	Short: "Delete a synchronization operation",
	Long: `Delete removes a sync operation from the system. For active syncs,
this will attempt to cancel the operation gracefully.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeleteSync(cmd, args[0])
	},
}

func init() {
	// Add subcommands
	Command.AddCommand(createCmd)
	Command.AddCommand(listCmd)
	Command.AddCommand(statusCmd)
	Command.AddCommand(deleteCmd)
}

func runCreateSync(cmd *cobra.Command, remoteURL string) error {
	// Validate remote URL
	if remoteURL == "" {
		return errors.New("remote URL is required")
	}

	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	syncID, err := client.CreateSync(cmd.Context(), remoteURL)
	if err != nil {
		return fmt.Errorf("failed to create sync: %w", err)
	}

	presenter.Printf(cmd, "Sync created with ID: %s", syncID)

	return nil
}

func runListSyncs(cmd *cobra.Command) error {
	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	itemCh, err := client.ListSyncs(cmd.Context(), &storev1alpha2.ListSyncsRequest{
		Limit:  &opts.Limit,
		Offset: &opts.Offset,
	})
	if err != nil {
		return fmt.Errorf("failed to list syncs: %w", err)
	}

	for {
		select {
		case sync, ok := <-itemCh:
			if !ok {
				// Channel closed, all items received
				return nil
			}

			presenter.Printf(cmd,
				"ID %s Status %s RemoteDirectoryUrl %s\n",
				sync.GetSyncId(),
				sync.GetStatus(),
				sync.GetRemoteDirectoryUrl(),
			)
		case <-cmd.Context().Done():
			return fmt.Errorf("context cancelled while listing syncs: %w", cmd.Context().Err())
		}
	}
}

func runGetSyncStatus(cmd *cobra.Command, syncID string) error {
	// Validate sync ID
	if syncID == "" {
		return errors.New("sync ID is required")
	}

	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	sync, err := client.GetSync(cmd.Context(), syncID)
	if err != nil {
		return fmt.Errorf("failed to get sync status: %w", err)
	}

	presenter.Printf(cmd,
		"ID %s Status %s RemoteDirectoryUrl %s\n",
		sync.GetSyncId(),
		storev1alpha2.SyncStatus_name[int32(sync.GetStatus())],
		sync.GetRemoteDirectoryUrl(),
	)

	return nil
}

func runDeleteSync(cmd *cobra.Command, syncID string) error {
	// Validate sync ID
	if syncID == "" {
		return errors.New("sync ID is required")
	}

	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	err := client.DeleteSync(cmd.Context(), syncID)
	if err != nil {
		return fmt.Errorf("failed to delete sync: %w", err)
	}

	return nil
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
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

When --stdin flag is used, the command parses JSON routing search output from stdin
and creates sync operations for each provider found in the search results.

Usage examples:

1. Create sync with remote peer:
  dir sync create https://directory.example.com

2. Create sync with specific CIDs:
  dir sync create http://localhost:8080 --cids cid1,cid2,cid3

3. Create sync from routing search output:
  dirctl routing search --skill "AI" --output json | dirctl sync create --stdin`,
	Args: func(cmd *cobra.Command, args []string) error {
		if opts.Stdin {
			return cobra.MaximumNArgs(0)(cmd, args)
		}

		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if opts.Stdin {
			return runCreateSyncFromStdin(cmd)
		}

		return runCreateSync(cmd, args[0], opts.CIDs)
	},
}

// List syncs subcommand.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all synchronization operations",
	Long: `List displays all sync operations known to the system, including active, 
completed, and failed synchronizations.

Usage examples:

1. List all syncs:
  dirctl sync list

2. Pagination:
  dirctl sync list --limit 10 --offset 20

3. Output formats:
  # Get syncs as JSON
  dirctl sync list --output json
  
  # Get syncs as JSONL for streaming
  dirctl sync list --output jsonl
  
  # Get raw sync data
  dirctl sync list --output raw`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runListSyncs(cmd)
	},
}

// Get sync status subcommand.
var statusCmd = &cobra.Command{
	Use:   "status <sync-id>",
	Short: "Get detailed status of a synchronization operation",
	Long: `Status retrieves comprehensive information about a specific sync operation,
including progress, timing, and error details if applicable.

Usage examples:

1. Get sync status:
  dirctl sync status <sync-id>

2. Output formats:
  # Get sync status as JSON
  dirctl sync status <sync-id> --output json
  
  # Get raw status data for scripting
  dirctl sync status <sync-id> --output raw`,
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
this will attempt to cancel the operation gracefully.

Usage examples:

1. Delete a sync:
  dirctl sync delete <sync-id>

2. Output formats:
  # Delete sync with JSON confirmation
  dirctl sync delete <sync-id> --output json
  
  # Delete sync with raw output for scripting
  dirctl sync delete <sync-id> --output raw`,
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

func runCreateSync(cmd *cobra.Command, remoteURL string, cids []string) error {
	// Validate remote URL
	if remoteURL == "" {
		return errors.New("remote URL is required")
	}

	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	syncID, err := client.CreateSync(cmd.Context(), remoteURL, cids)
	if err != nil {
		return fmt.Errorf("failed to create sync: %w", err)
	}

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "sync", "Sync created with ID", syncID)
}

func runListSyncs(cmd *cobra.Command) error {
	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	itemCh, err := client.ListSyncs(cmd.Context(), &storev1.ListSyncsRequest{
		Limit:  &opts.Limit,
		Offset: &opts.Offset,
	})
	if err != nil {
		return fmt.Errorf("failed to list syncs: %w", err)
	}

	// Collect results
	var results []interface{}

	for {
		select {
		case sync, ok := <-itemCh:
			if !ok {
				// Channel closed, all items received
				goto done
			}

			results = append(results, sync)
		case <-cmd.Context().Done():
			return fmt.Errorf("context cancelled while listing syncs: %w", cmd.Context().Err())
		}
	}

done:

	return presenter.PrintMessage(cmd, "syncs", "Sync results", results)
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

	return presenter.PrintMessage(cmd, "sync", "Sync status", sync.GetStatus())
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

	return presenter.PrintMessage(cmd, "sync", "Sync deleted with ID", syncID)
}

func runCreateSyncFromStdin(cmd *cobra.Command) error {
	// Parse the search output from stdin
	results, err := parseSearchOutput(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("failed to parse search output: %w", err)
	}

	if len(results) == 0 {
		presenter.PrintSmartf(cmd, "No search results found in stdin\n")

		return nil
	}

	// Group results by API address (one sync per peer)
	peerResults := groupResultsByAPIAddress(results)

	// Create sync operations for each peer
	return createSyncOperations(cmd, peerResults)
}

func parseSearchOutput(input io.Reader) ([]*routingv1.SearchResponse, error) {
	// Read JSON input from routing search --output json
	inputBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	var searchResponses []*routingv1.SearchResponse

	err = json.Unmarshal(inputBytes, &searchResponses)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return searchResponses, nil
}

// PeerSyncInfo holds sync information for a peer (grouped by API address).
type PeerSyncInfo struct {
	APIAddress string
	CIDs       []string
}

func groupResultsByAPIAddress(results []*routingv1.SearchResponse) map[string]PeerSyncInfo {
	peerResults := make(map[string]PeerSyncInfo)

	for _, result := range results {
		// Get the first API address if available
		var apiAddress string
		if result.GetPeer() != nil && len(result.GetPeer().GetAddrs()) > 0 {
			apiAddress = result.GetPeer().GetAddrs()[0]
		}

		// Skip results without API address
		if apiAddress == "" {
			continue
		}

		cid := result.GetRecordRef().GetCid()

		if existing, exists := peerResults[apiAddress]; exists {
			existing.CIDs = append(existing.CIDs, cid)
			peerResults[apiAddress] = existing
		} else {
			peerResults[apiAddress] = PeerSyncInfo{
				APIAddress: apiAddress,
				CIDs:       []string{cid},
			}
		}
	}

	return peerResults
}

func createSyncOperations(cmd *cobra.Command, peerResults map[string]PeerSyncInfo) error {
	client, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	totalSyncs := 0
	totalCIDs := 0

	syncIDs := make([]interface{}, 0, len(peerResults))

	for apiAddress, syncInfo := range peerResults {
		if syncInfo.APIAddress == "" {
			presenter.PrintSmartf(cmd, "WARNING: No API address found for peer\n")
			presenter.PrintSmartf(cmd, "Skipping sync for this peer\n")

			continue
		}

		// Create sync operation
		syncID, err := client.CreateSync(cmd.Context(), syncInfo.APIAddress, syncInfo.CIDs)
		if err != nil {
			presenter.PrintSmartf(cmd, "ERROR: Failed to create sync for peer %s: %v\n", apiAddress, err)

			continue
		}

		syncIDs = append(syncIDs, syncID)

		totalSyncs++
		totalCIDs += len(syncInfo.CIDs)
	}

	return presenter.PrintMessage(cmd, "sync IDs", "Sync IDs created", syncIDs)
}

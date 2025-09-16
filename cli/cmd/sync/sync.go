// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"

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

When --stdin flag is used, the command parses routing search output from stdin
and creates sync operations for each provider found in the search results.

Usage examples:

1. Create sync with remote peer:
  dir sync create https://directory.example.com

2. Create sync with specific CIDs:
  dir sync create http://localhost:8080 --cids cid1,cid2,cid3

3. Create sync from routing search output:
  dirctl routing search --skill "AI" | dirctl sync create --stdin`,
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

	presenter.Printf(cmd, "Sync created with ID: %s", syncID)

	return nil
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
		storev1.SyncStatus_name[int32(sync.GetStatus())],
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

// SearchResult represents a parsed search result.
type SearchResult struct {
	CID        string
	APIAddress string // API address for the provider
}

func runCreateSyncFromStdin(cmd *cobra.Command) error {
	// Parse the search output from stdin
	results, err := parseSearchOutput(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("failed to parse search output: %w", err)
	}

	if len(results) == 0 {
		presenter.Printf(cmd, "No search results found in stdin\n")

		return nil
	}

	// Group results by API address (one sync per peer)
	peerResults := groupResultsByAPIAddress(results)

	// Create sync operations for each peer
	return createSyncOperations(cmd, peerResults)
}

func parseSearchOutput(input io.Reader) ([]SearchResult, error) {
	var results []SearchResult

	scanner := bufio.NewScanner(input)

	// Regular expressions to match the search output format
	recordRegex := regexp.MustCompile(`^Record: (.+)$`)
	apiAddressRegex := regexp.MustCompile(`^    api address (.+)$`)

	var currentCID string

	for scanner.Scan() {
		line := scanner.Text() // Don't trim the line, preserve exact spacing

		// Match record line: "Record: <CID>"
		if matches := recordRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentCID = matches[1]

			continue
		}

		// Match API address line: "    api address <address>"
		if matches := apiAddressRegex.FindStringSubmatch(line); len(matches) > 1 && currentCID != "" {
			apiAddress := matches[1]

			results = append(results, SearchResult{
				CID:        currentCID,
				APIAddress: apiAddress,
			})

			// Reset for next record
			currentCID = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return results, nil
}

// PeerSyncInfo holds sync information for a peer (grouped by API address).
type PeerSyncInfo struct {
	APIAddress string
	CIDs       []string
}

func groupResultsByAPIAddress(results []SearchResult) map[string]PeerSyncInfo {
	peerResults := make(map[string]PeerSyncInfo)

	for _, result := range results {
		if existing, exists := peerResults[result.APIAddress]; exists {
			existing.CIDs = append(existing.CIDs, result.CID)
			peerResults[result.APIAddress] = existing
		} else {
			peerResults[result.APIAddress] = PeerSyncInfo{
				APIAddress: result.APIAddress,
				CIDs:       []string{result.CID},
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

	for apiAddress, syncInfo := range peerResults {
		if syncInfo.APIAddress == "" {
			presenter.Printf(cmd, "WARNING: No API address found for peer\n")
			presenter.Printf(cmd, "Skipping sync for this peer\n")

			continue
		}

		// Create sync operation
		syncID, err := client.CreateSync(cmd.Context(), syncInfo.APIAddress, syncInfo.CIDs)
		if err != nil {
			presenter.Printf(cmd, "ERROR: Failed to create sync for peer %s: %v\n", apiAddress, err)

			continue
		}

		presenter.Printf(cmd, "Sync created with ID: %s\n", syncID)
		presenter.Printf(cmd, "\n")

		totalSyncs++
		totalCIDs += len(syncInfo.CIDs)
	}

	presenter.Printf(cmd, "Summary: Created %d sync operation(s) for %d CID(s)\n", totalSyncs, totalCIDs)

	return nil
}

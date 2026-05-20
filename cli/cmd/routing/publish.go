// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package routing

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish <cid> [cid...]",
	Short: "Publish record to the network for discovery",
	Long: `Publish a record to the network to allow content discovery by other peers.

This command announces a record that is already stored locally to the distributed
network, making it discoverable by other peers through the DHT.

Records must already exist in local storage (use 'dirctl push' first if needed).

Key Features:
- Network announcement: Makes record discoverable by other peers
- Local storage: Stores record in local routing index
- DHT announcement: Announces record and labels to distributed network
- Background retry: Failed announcements are retried automatically
- Batch publication: Submit multiple CIDs in one request

Usage examples:

1. Publish a record to the network:
   dirctl routing publish <cid>

2. Publish multiple records in a single request:
   dirctl routing publish <cid1> <cid2> <cid3>

3. Publish records from stdin (JSON array or line-delimited CIDs):
   dirctl search --format cid --limit 100 --output json | dirctl routing publish --stdin

4. Output formats:
   # Publish with JSON confirmation
   dirctl routing publish <cid> --output json
   
   # Publish with raw output for scripting
   dirctl routing publish <cid> --output raw

Note: The record must already be pushed to storage before publishing.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if publishOpts.FromStdin {
			return cobra.MaximumNArgs(0)(cmd, args)
		}

		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: runPublishCommand,
}

var publishOpts struct {
	FromStdin bool
}

func init() {
	publishCmd.Flags().BoolVar(&publishOpts.FromStdin, "stdin", false,
		"Read CIDs from standard input. Supports JSON array output from 'dirctl search --output json' and line-delimited CIDs.")
}

func runPublishCommand(cmd *cobra.Command, args []string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	cids := append([]string{}, args...)

	if publishOpts.FromStdin {
		stdinCIDs, err := readCIDsFromStdin(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to read CIDs from stdin: %w", err)
		}

		cids = append(cids, stdinCIDs...)
	}

	cids = deduplicateCIDs(cids)
	if len(cids) == 0 {
		return errors.New("at least one CID is required (pass arguments or use --stdin)")
	}

	recordRefs := make([]*corev1.RecordRef, 0, len(cids))
	for _, cid := range cids {
		recordRefs = append(recordRefs, &corev1.RecordRef{Cid: cid})
	}

	// Lookup metadata to verify records exist
	_, err := c.LookupBatch(cmd.Context(), recordRefs)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// Start publishing using record references
	if err := c.Publish(cmd.Context(), &routingv1.PublishRequest{
		Request: &routingv1.PublishRequest_RecordRefs{
			RecordRefs: &routingv1.RecordRefs{
				Refs: recordRefs,
			},
		},
	}); err != nil {
		if strings.Contains(err.Error(), "failed to announce object") {
			return errors.New("failed to announce object, it will be retried in the background on the API server")
		}

		return fmt.Errorf("failed to publish: %w", err)
	}

	// Output in the appropriate format
	result := map[string]any{
		"count":   len(recordRefs),
		"cids":    cids,
		"status":  "Successfully submitted publication request",
		"message": "Records will be discoverable by other peers once the publication service processes the request",
	}

	if len(cids) == 1 {
		result["cid"] = cids[0]
	}

	return presenter.PrintMessage(cmd, "Publish", "Successfully submitted publication request", result)
}

func readCIDsFromStdin(reader io.Reader) ([]string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		return nil, nil
	}

	if strings.HasPrefix(input, "[") {
		var cids []string
		if err := json.Unmarshal([]byte(input), &cids); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array of CIDs: %w", err)
		}

		return cids, nil
	}

	cids := make([]string, 0)

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		cid := strings.TrimSpace(scanner.Text())
		if cid == "" {
			continue
		}

		cids = append(cids, cid)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan stdin: %w", err)
	}

	return cids, nil
}

func deduplicateCIDs(cids []string) []string {
	seen := make(map[string]struct{}, len(cids))
	out := make([]string, 0, len(cids))

	for _, cid := range cids {
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}

		if _, exists := seen[cid]; exists {
			continue
		}

		seen[cid] = struct{}{}
		out = append(out, cid)
	}

	return out
}

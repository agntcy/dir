// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package routing

import (
	"errors"
	"fmt"
	"os"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	cidsUtils "github.com/agntcy/dir/cli/util/cids"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/prompt"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const publishAllPrompt = "This operation will publish every currently stored record. Would you like to proceed?"

var publishCmd = &cobra.Command{
	Use:   "publish [cid...]",
	Short: "Publish record to the network for discovery",
	Long: `Publish records to the network to allow content discovery by other peers.

This command announces records that are already stored locally to the distributed
network, making them discoverable by other peers through the DHT.

Records must already exist in local storage (use 'dirctl push' first if needed).

Key Features:
- Network announcement: Makes records discoverable by other peers
- Local storage: Stores records in the local routing index
- DHT announcement: Announces records and labels to the distributed network
- Background retry: Failed announcements are retried automatically
- Batch publication: Submit multiple CIDs in one request
- Bulk publication: Publish every currently stored record

Usage examples:

1. Publish a record to the network:
   dirctl routing publish <cid>

2. Publish multiple records in a single request:
   dirctl routing publish <cid1> <cid2> <cid3>

3. Publish records from stdin (JSON array or line-delimited CIDs):
   dirctl search --format cid --limit 100 --output json | dirctl routing publish --stdin

4. Publish every currently stored record:
   dirctl routing publish --all

5. Publish all records without an interactive confirmation:
   dirctl routing publish --all --yes

6. Output formats:
   # Publish with JSON confirmation
   dirctl routing publish <cid> --output json

   # Publish with raw output for scripting
   dirctl routing publish <cid> --output raw

Note: Records must already be pushed to storage before publishing.
`,
	Args: validatePublishArgs,
	RunE: runPublishCommand,
}

var publishOpts struct {
	FromStdin bool
	All       bool
	Yes       bool
}

func init() {
	publishCmd.Flags().BoolVar(
		&publishOpts.FromStdin, "stdin", false, cidsUtils.StdinFlagUsage)
	publishCmd.Flags().BoolVar(
		&publishOpts.All, "all", false,
		"Publish every currently stored record")
	publishCmd.Flags().BoolVarP(
		&publishOpts.Yes, "yes", "y", false,
		"Skip the confirmation prompt when using --all")
}

func validatePublishArgs(cmd *cobra.Command, args []string) error {
	if publishOpts.Yes && !publishOpts.All {
		return errors.New("--yes can only be used with --all")
	}

	if publishOpts.All {
		if publishOpts.FromStdin {
			return errors.New("--all cannot be used with --stdin")
		}

		if len(args) > 0 {
			return errors.New("--all cannot be used with CID arguments")
		}

		return nil
	}

	return cidsUtils.Args(&publishOpts.FromStdin)(cmd, args)
}

func runPublishCommand(cmd *cobra.Command, args []string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	if publishOpts.All {
		return runPublishAllCommand(cmd, c)
	}

	cids, err := cidsUtils.Collect(args, cmd.InOrStdin(), publishOpts.FromStdin)
	if err != nil {
		return err
	}

	recordRefs := make([]*corev1.RecordRef, 0, len(cids))
	for _, cid := range cids {
		recordRefs = append(recordRefs, &corev1.RecordRef{Cid: cid})
	}

	// Lookup metadata to verify records exist
	if _, err := c.LookupBatch(cmd.Context(), recordRefs); err != nil {
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
		return normalizePublishError(err)
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

	return presenter.PrintMessage(
		cmd,
		"Publish",
		"Successfully submitted publication request",
		result,
	)
}

func runPublishAllCommand(cmd *cobra.Command, c *client.Client) error {
	confirmed, err := confirmPublishAllIfNeeded(cmd)
	if err != nil {
		return err
	}

	if !confirmed {
		presenter.PrintSmartf(cmd, "Aborted. No records were submitted for publication.\n")

		return nil
	}

	if err := c.Publish(cmd.Context(), &routingv1.PublishRequest{
		Request: &routingv1.PublishRequest_AllRecords{
			AllRecords: true,
		},
	}); err != nil {
		return normalizePublishError(err)
	}

	result := map[string]any{
		"all_records": true,
		"status":      "Successfully submitted publication request",
		"message":     "Every currently stored record will be submitted for network publication",
	}

	return presenter.PrintMessage(
		cmd,
		"Publish",
		"Successfully submitted publication request",
		result,
	)
}

func confirmPublishAllIfNeeded(cmd *cobra.Command) (bool, error) {
	if publishOpts.Yes {
		return true, nil
	}

	in, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) { //nolint:gosec // G115: a file descriptor fits in an int.
		return false, errors.New(
			"refusing to prompt for --all on non-terminal stdin; pass --yes to continue",
		)
	}

	return prompt.Confirm(cmd, publishAllPrompt)
}

func normalizePublishError(err error) error {
	if strings.Contains(err.Error(), "failed to announce object") {
		return errors.New(
			"failed to announce object, it will be retried in the background on the API server")
	}

	return fmt.Errorf("failed to publish: %w", err)
}

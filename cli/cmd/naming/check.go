// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck,dupl
package naming

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check <cid>",
	Short: "Check if a record has verified name ownership",
	Long: `Check if a record has verified name ownership.

This command checks whether a record has a stored name verification.
Unlike 'dirctl naming verify', this does not perform verification - it only
queries for an existing verification result.

Use this command to:
- Check if a record has been name-verified
- Retrieve verification details (domain, method, key ID, timestamp)

Usage examples:

1. Check name verification status:
   dirctl naming check <cid>

2. Check with JSON output:
   dirctl naming check <cid> --output json
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCheckCommand(cmd, args[0])
	},
}

func runCheckCommand(cmd *cobra.Command, cid string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Call GetVerificationInfo
	resp, err := c.GetVerificationInfo(cmd.Context(), cid)
	if err != nil {
		return fmt.Errorf("failed to get verification info: %w", err)
	}

	if !resp.GetVerified() {
		errMsg := resp.GetErrorMessage()
		if errMsg == "" {
			errMsg = "no verification found"
		}

		// Output the result
		result := map[string]any{
			"cid":      cid,
			"verified": false,
			"message":  errMsg,
		}

		return presenter.PrintMessage(cmd, "Name Verification Check", "No name verification found", result)
	}

	// Output the verification details
	v := resp.GetVerification()
	dv := v.GetDomain()
	result := map[string]any{
		"cid":         cid,
		"verified":    true,
		"domain":      dv.GetDomain(),
		"method":      dv.GetMethod(),
		"key_id":      dv.GetKeyId(),
		"verified_at": dv.GetVerifiedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}

	return presenter.PrintMessage(cmd, "Name Verification Check", "Record has verified name ownership", result)
}

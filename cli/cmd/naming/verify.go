// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package naming

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <cid>",
	Short: "Verify domain ownership for a signed record",
	Long: `Verify domain ownership for a signed record.

This command performs domain ownership verification for a record that has
already been signed. It proves that the signing key is authorized by the
domain claimed in the record's name field.

The verification result is stored as a referrer to the record, so subsequent
calls to 'dirctl naming check' can retrieve the stored verification without
re-verifying.

Prerequisites:
- Record must be pushed to the store
- Record must be signed (public key referrer must exist)
- Record's name must be in format "<domain>/<path>" (e.g., "cisco.com/agent")

Verification methods (tried in order):
1. DNS TXT record at _oasf.<domain>
2. Well-known file at https://<domain>/.well-known/oasf.json

Usage examples:

1. Verify domain ownership:
   dirctl naming verify <cid>

2. Verify with JSON output:
   dirctl naming verify <cid> --output json
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVerifyCommand(cmd, args[0])
	},
}

func runVerifyCommand(cmd *cobra.Command, cid string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Call VerifyDomain
	resp, err := c.VerifyDomain(cmd.Context(), cid)
	if err != nil {
		return fmt.Errorf("failed to verify domain: %w", err)
	}

	if !resp.GetVerified() {
		errMsg := resp.GetErrorMessage()
		if errMsg == "" {
			errMsg = "verification failed"
		}

		// Output the failure
		result := map[string]interface{}{
			"cid":      cid,
			"verified": false,
			"error":    errMsg,
		}

		return presenter.PrintMessage(cmd, "Domain Verification", "Domain verification failed", result)
	}

	// Output the success
	v := resp.GetVerification()
	result := map[string]interface{}{
		"cid":            cid,
		"verified":       true,
		"domain":         v.GetDomain(),
		"method":         v.GetMethod(),
		"matched_key_id": v.GetMatchedKeyId(),
		"verified_at":    v.GetVerifiedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}

	return presenter.PrintMessage(cmd, "Domain Verification", "Domain ownership verified successfully", result)
}

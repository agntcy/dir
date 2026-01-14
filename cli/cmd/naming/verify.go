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

var verifyCmd = &cobra.Command{
	Use:   "verify <cid>",
	Short: "Verify name ownership for a signed record",
	Long: `Verify name ownership for a signed record.

This command performs name ownership verification for a record that has
already been signed. It proves that the signing key is authorized by the
domain claimed in the record's name field.

The record's name must include a protocol prefix to specify the verification method:
- dns://domain/path - verify using DNS TXT records
- wellknown://domain/path - verify using well-known file

Records without a protocol prefix will not be verified.

The verification result is stored as a referrer to the record, so subsequent
calls to 'dirctl naming check' can retrieve the stored verification without
re-verifying.

Prerequisites:
- Record must be pushed to the store
- Record must be signed (public key referrer must exist)
- Record's name must include a protocol prefix (dns:// or wellknown://)
- Record's name must contain a valid domain (with at least one dot)

Verification methods:
1. DNS TXT record at _oasf.<domain> with format "v=oasf1; k=<type>; p=<key>"
2. Well-known file at https://<domain>/.well-known/oasf.json

Usage examples:

1. Verify name ownership:
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

	// Call Verify
	resp, err := c.VerifyName(cmd.Context(), cid)
	if err != nil {
		return fmt.Errorf("failed to verify name: %w", err)
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

		return presenter.PrintMessage(cmd, "Name Verification", "Name verification failed", result)
	}

	// Output the success
	v := resp.GetVerification()
	dv := v.GetDomain()
	result := map[string]interface{}{
		"cid":            cid,
		"verified":       true,
		"domain":         dv.GetDomain(),
		"method":         dv.GetMethod(),
		"matched_key_id": dv.GetMatchedKeyId(),
		"verified_at":    dv.GetVerifiedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
	}

	return presenter.PrintMessage(cmd, "Name Verification", "Name ownership verified successfully", result)
}

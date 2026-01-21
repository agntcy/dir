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
	Short: "Check if a record has verified name ownership",
	Long: `Check if a record has verified name ownership.

This command checks whether a record has a stored name verification.
It queries the server for an existing verification result that was created
during the signing process.

Name verification proves that the signing key is authorized by the domain
claimed in the record's name field. Verification is performed automatically
when a record is signed using 'dirctl sign'.

The record's name must include a protocol prefix to specify the verification method:
- https://domain/path - verify using JWKS well-known file (RFC 7517)
- http://domain/path - verify using JWKS via HTTP (testing only)

Verification method:
JWKS well-known file at <scheme>://<domain>/.well-known/jwks.json

The server automatically re-verifies records based on TTL to ensure
domain ownership remains valid.

Usage examples:

1. Check name verification status:
   dirctl naming verify <cid>

2. Check with JSON output:
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

	// Call GetVerificationInfo to check existing verification
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

		return presenter.PrintMessage(cmd, "Name Verification", "No name verification found", result)
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

	return presenter.PrintMessage(cmd, "Name Verification", "Record has verified name ownership", result)
}

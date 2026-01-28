// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package naming

import (
	"errors"
	"fmt"

	namingv1 "github.com/agntcy/dir/api/naming/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <cid-or-name[:version]>",
	Short: "Check if a record has verified name ownership",
	Long: `Check if a record has verified name ownership.

This command checks whether a record has a stored name verification.
It queries the server for an existing verification result that was created
during the signing process.

You can specify the record by:
- CID directly (e.g., "bafyreib...")
- Name (e.g., "cisco.com/agent") - uses the latest version
- Name with version (e.g., "cisco.com/agent:v1.0.0")

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

1. Check name verification status by CID:
   dirctl naming verify <cid>

2. Check name verification status by name:
   dirctl naming verify cisco.com/agent

3. Check name verification status by name and version:
   dirctl naming verify cisco.com/agent:v1.0.0

4. Check with JSON output:
   dirctl naming verify <cid-or-name> --output json
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVerifyCommand(cmd, args[0])
	},
}

func runVerifyCommand(cmd *cobra.Command, input string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Parse the input to determine if it's a CID or name reference
	ref := reference.Parse(input)

	var cid string

	if ref.IsCID() {
		// Direct CID lookup
		cid = input
	} else {
		// Name-based lookup - resolve to CID first
		resolvedCID, err := reference.ResolveToCID(cmd.Context(), c, input)
		if err != nil {
			return err
		}

		cid = resolvedCID
	}

	// Get verification info by CID
	resp, err := c.GetVerificationInfo(cmd.Context(), cid)
	if err != nil {
		return fmt.Errorf("failed to get verification info: %w", err)
	}

	return outputVerificationResult(cmd, cid, resp)
}

func outputVerificationResult(cmd *cobra.Command, cid string, resp *namingv1.GetVerificationInfoResponse) error {
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

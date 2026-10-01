// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package identity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <cid-or-name[:version]>",
	Short: "Show the verification result of a record's identity and ownership claims",
	Long: `Show the verification result of a record's identity and ownership claims.

For each of the two claims, the result is "verified" or "failed" (with the
reason), or "no result" if the claim has not been verified yet. A claim pushed
with "dirctl identity claim" has no result until the server has checked it.

You can specify the record by:
- CID directly (e.g., "bafyreib...")
- Name (e.g., "cisco.com/agent") - uses the highest semantic version
- Name with version (e.g., "cisco.com/agent:v1.0.0")

Usage examples:

1. Show the claims of a record:
   dirctl identity status <cid>

2. Show the claims of the latest version of a name:
   dirctl identity status cisco.com/agent

3. Machine-readable output:
   dirctl identity status <cid> --output json
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(cmd, args[0])
	},
}

// claimResult is one claim's verification result as printed.
type claimResult struct {
	Role       string `json:"role"`
	Subject    string `json:"subject"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	VerifiedAt string `json:"verified_at,omitempty"`
}

// statusResult is a record's claim results as printed. A claim with no result
// is null.
type statusResult struct {
	CID      string       `json:"cid"`
	Identity *claimResult `json:"identity"`
	Owner    *claimResult `json:"owner"`
}

func runStatus(cmd *cobra.Command, input string) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	cid, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return err
	}

	resp, err := c.GetIdentityStatus(cmd.Context(), cid)
	if err != nil {
		return err
	}

	result := statusResult{
		CID:      cid,
		Identity: toClaimResult(resp.GetIdentity()),
		Owner:    toClaimResult(resp.GetOwner()),
	}

	if presenter.GetOutputOptions(cmd).Format == presenter.FormatHuman {
		presenter.Printf(cmd, "%s", formatStatus(result))

		return nil
	}

	return presenter.PrintMessage(cmd, "Identity status", "Identity status", result)
}

func toClaimResult(v *identityv1.ClaimVerification) *claimResult {
	if v == nil {
		return nil
	}

	result := &claimResult{
		Role:    strings.ToLower(strings.TrimPrefix(v.GetRole().String(), "CLAIM_ROLE_")),
		Subject: v.GetSubject(),
		Status:  strings.ToLower(strings.TrimPrefix(v.GetStatus().String(), "CLAIM_VERIFICATION_STATUS_")),
		Error:   v.GetError(),
	}

	if v.GetVerifiedAt() != nil {
		result.VerifiedAt = v.GetVerifiedAt().AsTime().Format(time.RFC3339)
	}

	return result
}

func formatStatus(s statusResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Record:   %s\n", s.CID)
	fmt.Fprintf(&b, "Identity: %s\n", formatClaim(s.Identity))
	fmt.Fprintf(&b, "Owner:    %s\n", formatClaim(s.Owner))

	return b.String()
}

func formatClaim(r *claimResult) string {
	if r == nil {
		return "no result"
	}

	line := fmt.Sprintf("%s %s (checked %s)", r.Status, r.Subject, r.VerifiedAt)
	if r.Error != "" {
		line += ": " + r.Error
	}

	return line
}

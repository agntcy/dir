// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package identity

import (
	"errors"
	"fmt"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

// Command is the parent command for record identity/ownership claim operations.
var Command = &cobra.Command{
	Use:   "identity",
	Short: "Manage verifiable record identity/ownership claims",
	Long: `Manage verifiable record identity and ownership claims.

This command group provides:

- claim: attach a signed identity or ownership claim to a record
- status: check the cached verification status of a record's claims
- resolve: resolve a record name (with optional version) to its CIDs

A record's own identity and owning entity are declared as annotations on the
record itself ("agntcy.dir/identity", "agntcy.dir/owner") and are part of the
record's content-addressed CID. A claim is the cryptographic proof that a
specific identity (DNS, HTTPS, DID, or SPIFFE) legitimately signed for that
subject, bound to this exact record via its CID.

Examples:

1. Claim a record's own identity, signing with a plain key:
   dirctl identity claim --record <cid> --role identity \
     --subject did:web:acme.com:agents:finance --key ./key.pem

2. Claim ownership, signing with a SPIFFE X.509-SVID:
   dirctl identity claim --record <cid> --role owner \
     --subject spiffe://acme.com/agents/finance --key ./svid.key --cert ./svid.crt

3. Check claim verification status:
   dirctl identity status <cid-or-name>

4. Resolve a name to its CIDs:
   dirctl identity resolve cisco.com/agent
`,
}

var (
	claimRecord  string
	claimRole    string
	claimSubject string
	claimKeyPath string
	claimCert    string
)

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Attach a signed identity or ownership claim to a record",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runClaim(cmd)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status <cid-or-name[:version]>",
	Short: "Check the cached verification status of a record's identity/ownership claims",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus(cmd, args[0])
	},
}

var resolveCmd = &cobra.Command{
	Use:   "resolve <name>[:version]",
	Short: "Resolve a record name (with optional version) to its CIDs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runResolve(cmd, args[0])
	},
}

func init() {
	claimCmd.Flags().StringVar(&claimRecord, "record", "", "CID or name of the record to claim (required)")
	claimCmd.Flags().StringVar(&claimRole, "role", "", "Claim role: \"identity\" or \"owner\" (required)")
	claimCmd.Flags().StringVar(&claimSubject, "subject", "", "The identity/owner URI being claimed (required)")
	claimCmd.Flags().StringVar(&claimKeyPath, "key", "", "Path to a PEM-encoded private key to sign with")
	claimCmd.Flags().StringVar(&claimCert, "cert", "", "Path to a PEM-encoded X.509-SVID certificate (SPIFFE subjects only)")

	_ = claimCmd.MarkFlagRequired("record")
	_ = claimCmd.MarkFlagRequired("role")
	_ = claimCmd.MarkFlagRequired("subject")
	claimCmd.MarkFlagsRequiredTogether("key", "cert")

	presenter.AddOutputFlags(statusCmd)

	Command.AddCommand(claimCmd, statusCmd, resolveCmd)
}

func runClaim(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	cid, err := reference.ResolveToCID(cmd.Context(), c, claimRecord)
	if err != nil {
		return err
	}

	signer, err := loadSigner(claimKeyPath, claimCert)
	if err != nil {
		return err
	}

	switch claimRole {
	case "identity":
		if err := c.ClaimIdentity(cmd.Context(), cid, claimSubject, signer); err != nil {
			return fmt.Errorf("failed to claim identity: %w", err)
		}
	case "owner":
		if err := c.ClaimOwnership(cmd.Context(), cid, claimSubject, signer); err != nil {
			return fmt.Errorf("failed to claim ownership: %w", err)
		}
	default:
		return fmt.Errorf("invalid --role %q: must be \"identity\" or \"owner\"", claimRole)
	}

	cmd.Printf("Claim pushed successfully for record %s\n", cid)

	return nil
}

// loadSigner builds an identityv1.Signer from the --key/--cert flags. A
// SPIFFE signer is used when --cert is set, otherwise a plain key signer.
func loadSigner(keyPath, certPath string) (identityv1.Signer, error) {
	if keyPath == "" {
		return nil, errors.New("--key is required to sign a claim")
	}

	if certPath != "" {
		signer, err := identityv1.NewSpiffeSignerFromFile(keyPath, certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load SPIFFE signer: %w", err)
		}

		return signer, nil
	}

	signer, err := identityv1.NewKeySignerFromFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load key signer: %w", err)
	}

	return signer, nil
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
		return fmt.Errorf("failed to get identity status: %w", err)
	}

	result := map[string]any{"cid": cid}
	result["identity"] = claimStatusToMap(resp.GetIdentity())
	result["owner"] = claimStatusToMap(resp.GetOwner())

	return presenter.PrintMessage(cmd, "Identity Status", "Record identity/ownership claim status", result)
}

func claimStatusToMap(s *identityv1.ClaimStatus) map[string]any {
	if s == nil {
		return map[string]any{"present": false}
	}

	return map[string]any{
		"present":       true,
		"verified":      s.GetVerified(),
		"subject":       s.GetSubject(),
		"error_message": s.GetErrorMessage(),
	}
}

func runResolve(cmd *cobra.Command, input string) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	ref := reference.Parse(input)

	name := ref.Name
	if name == "" {
		name = input
	}

	resp, err := c.Resolve(cmd.Context(), name, ref.Version)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", input, err)
	}

	for _, r := range resp.GetRecords() {
		cmd.Printf("%s\t%s\t%s\n", r.GetCid(), r.GetName(), r.GetVersion())
	}

	return nil
}

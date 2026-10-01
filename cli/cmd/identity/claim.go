// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package identity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/spf13/cobra"
)

const (
	roleIdentity = "identity"
	roleOwner    = "owner"

	spiffeScheme = "spiffe://"
)

var claimOpts struct {
	Record        string
	Role          string
	Subject       string
	Key           string
	PasswordStdin bool
	Cert          string
}

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Sign an identity or ownership claim and attach it to a record",
	Long: `Sign an identity or ownership claim and attach it to a record.

The claim asserts that --subject is the record's own identity (--role identity)
or its owner (--role owner). It is signed with the private key in --key and
stored on the record. It is stored unverified: the server verifies it later, and
"dirctl identity status" shows the result.

The record can be given as:
- CID directly (e.g., "bafyreib...")
- Name (e.g., "cisco.com/agent") - uses the highest semantic version
- Name with version (e.g., "cisco.com/agent:v1.0.0")

The key is a PEM file: an EC, RSA or Ed25519 private key, unencrypted or an
encrypted PKCS#8 key ("ENCRYPTED PRIVATE KEY"). The public half must be the key
published for the subject. For an encrypted key, the password is read from the
COSIGN_PASSWORD environment variable, from standard input with --password-stdin,
or prompted for on a terminal.

--cert is only for spiffe:// subjects, where the signer's X.509-SVID is the proof
rather than a published key. It is a PEM or DER certificate for the key, whose URI
SAN must be the subject. It is not used, and not accepted, for any other subject.

Usage examples:

1. Claim the identity of a record, with an unencrypted key:
   dirctl identity claim --record <cid> --role identity \
       --subject did:web:acme.com:agents:finance --key identity.key

2. Claim ownership by a domain, with an encrypted key:
   COSIGN_PASSWORD=secret dirctl identity claim --record cisco.com/agent:v1.0.0 \
       --role owner --subject dns:acme.com --key owner.key

3. Claim a SPIFFE identity:
   dirctl identity claim --record <cid> --role identity \
       --subject spiffe://acme.com/agents/finance --key svid.key --cert svid.pem
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runClaim(cmd)
	},
}

func init() {
	flags := claimCmd.Flags()
	flags.StringVar(&claimOpts.Record, "record", "", "Record to claim: CID, name or name:version (required)")
	flags.StringVar(&claimOpts.Role, "role", "", "What the claim asserts: identity or owner (required)")
	flags.StringVar(&claimOpts.Subject, "subject", "", "Identity URI being claimed, e.g. did:web:acme.com (required)")
	flags.StringVar(&claimOpts.Key, "key", "", "Path to the PEM private key to sign with (required)")
	flags.BoolVar(&claimOpts.PasswordStdin, "password-stdin", false,
		"Read the private key password from standard input; do not combine with other stdin input flags")
	flags.StringVar(&claimOpts.Cert, "cert", "",
		"Path to the PEM or DER X.509 certificate of the key (only for spiffe:// subjects)")

	_ = claimCmd.MarkFlagRequired("record")
	_ = claimCmd.MarkFlagRequired("role")
	_ = claimCmd.MarkFlagRequired("subject")
	_ = claimCmd.MarkFlagRequired("key")
}

func validateClaimFlags() error {
	if claimOpts.Role != roleIdentity && claimOpts.Role != roleOwner {
		return fmt.Errorf("invalid --role %q: must be %q or %q", claimOpts.Role, roleIdentity, roleOwner)
	}

	switch isSPIFFE := strings.HasPrefix(claimOpts.Subject, spiffeScheme); {
	case isSPIFFE && claimOpts.Cert == "":
		return errors.New("--cert is required for a spiffe:// subject: its X.509-SVID is the proof of the claim")
	case !isSPIFFE && claimOpts.Cert != "":
		return errors.New("--cert only applies to spiffe:// subjects")
	}

	return nil
}

func runClaim(cmd *cobra.Command) error {
	if err := validateClaimFlags(); err != nil {
		return err
	}

	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	signer, err := loadSigner(claimOpts.Key, claimOpts.PasswordStdin)
	if err != nil {
		return err
	}

	var signOpts []clientidentity.SignOption

	if claimOpts.Cert != "" {
		cert, err := readFile(claimOpts.Cert, "certificate")
		if err != nil {
			return err
		}

		signOpts = append(signOpts, clientidentity.WithCertificate(cert))
	}

	cid, err := reference.ResolveToCID(cmd.Context(), c, claimOpts.Record)
	if err != nil {
		return err
	}

	if claimOpts.Role == roleIdentity {
		err = c.ClaimIdentity(cmd.Context(), cid, claimOpts.Subject, signer, signOpts...)
	} else {
		err = c.ClaimOwnership(cmd.Context(), cid, claimOpts.Subject, signer, signOpts...)
	}

	if err != nil {
		return err
	}

	if presenter.GetOutputOptions(cmd).Format == presenter.FormatHuman {
		presenter.Printf(cmd, "Pushed %s claim %s for %s\nIt is verified later: see \"dirctl identity status %s\"\n",
			claimOpts.Role, claimOpts.Subject, cid, cid)

		return nil
	}

	result := map[string]any{
		"cid":     cid,
		"role":    claimOpts.Role,
		"subject": claimOpts.Subject,
		"status":  "pushed",
	}

	return presenter.PrintMessage(cmd, "Identity claim", "Claim pushed", result)
}

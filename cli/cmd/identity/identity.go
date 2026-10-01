// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// Command is the parent command for identity and ownership claim operations.
var Command = &cobra.Command{
	Use:   "identity",
	Short: "Identity and ownership claim operations",
	Long: `Identity and ownership claim operations.

A claim is a signed assertion about a record: that it has a given identity
(the record's own name in the world, e.g. did:web:acme.com:agents:finance), or
that it is owned by a given entity (e.g. dns:acme.com). The claim is signed
locally with a private key and attached to the record.

Pushing a claim stores it unverified. The server checks it later by looking up
the subject's public key, and "dirctl identity status" shows the outcome.

This command group provides:

- claim: Sign a claim and attach it to a record
- status: Show the verification result of a record's claims
- resolve: Resolve a record name to the CIDs of its versions

Supported subjects:
- dns:<domain> or <domain>      key published in a TXT record
- https://<domain>[/path]       key published in <domain>/.well-known/jwks.json
- did:web:<domain>[:path]       key in the DID document
- did:key:<key>                 key embedded in the DID itself
- spiffe://<domain>/<path>      key in an X.509-SVID passed with --cert

Examples:

1. Claim an identity for a record:
   dirctl identity claim --record <cid> --role identity \
       --subject did:web:acme.com:agents:finance --key identity.key

2. Claim ownership with a SPIFFE SVID:
   dirctl identity claim --record <cid> --role owner \
       --subject spiffe://acme.com/team --key svid.key --cert svid.pem

3. Check the result:
   dirctl identity status <cid>

4. List the versions of a name:
   dirctl identity resolve cisco.com/agent
`,
}

func init() {
	Command.AddCommand(claimCmd, statusCmd, resolveCmd)

	presenter.AddOutputFlags(claimCmd)
	presenter.AddOutputFlags(statusCmd)
	presenter.AddOutputFlags(resolveCmd)
}

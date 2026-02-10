// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package verify

import (
	"errors"
	"fmt"
	"os"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

//nolint:mnd
var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify record signature against identity-based OIDC or key-based signing",
	Long: `This command verifies the record signature against
identity-based OIDC or key-based signing process.

Verification is performed client-side by fetching signatures from the
directory and verifying them locally using Sigstore libraries.

Usage examples:

1. Verify a record signature (any valid signature):

	dirctl verify <record-cid>

2. Verify against a specific public key:

	dirctl verify <record-cid> --key <key-file>

3. Verify against OIDC identity (exact match):

	dirctl verify <record-cid> \
		--oidc-issuer https://github.com/login/oauth \
		--oidc-subject user@example.com

4. Verify against just OIDC issuer (any identity from that issuer):

	dirctl verify <record-cid> --oidc-issuer https://github.com/login/oauth

5. Verify using regular expression patterns (cosign compatible):

	# Match any identity from GitHub Actions (regexp)
	dirctl verify <record-cid> \
		--oidc-issuer "https://token.actions.githubusercontent.com" \
		--oidc-subject ".*@users.noreply.github.com"

	# Match any issuer containing "github" (regexp)
	dirctl verify <record-cid> \
		--oidc-issuer ".*github.*" \
		--oidc-subject "user@example.com"

	# Both issuer and identity as regexp
	dirctl verify <record-cid> \
		--oidc-issuer ".*sigstore.*" \
		--oidc-subject ".*@example.com"

6. Advanced verification options (OIDC only):

	# Skip transparency log verification (for private signatures)
	dirctl verify <record-cid> --ignore-tlog

	# Use custom TUF mirror for trusted root material
	dirctl verify <record-cid> --tuf-mirror-url https://custom-tuf.example.com

	# Offline/air-gapped verification with local trusted root
	dirctl verify <record-cid> --trusted-root-path /path/to/trusted_root.json

	# Skip multiple verification checks
	dirctl verify <record-cid> --ignore-tlog --ignore-tsa --ignore-sct

7. Output formats:

	# Get verification result as JSON
	dirctl verify <record-cid> --output json

	# Get raw verification status for scripting
	dirctl verify <record-cid> --output raw
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var recordRef string
		if len(args) > 1 {
			return errors.New("one argument is allowed")
		} else if len(args) == 1 {
			recordRef = args[0]
		} else {
			return errors.New("record CID is required")
		}

		return runCommand(cmd, recordRef)
	},
}

// nolint:mnd,gocognit,nestif,cyclop
func runCommand(cmd *cobra.Command, recordRef string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Construct the verification request
	var provider *signv1.VerifyRequestProvider

	switch {
	case opts.Key != "":
		// Read public key from flag and add to request
		pubKey, err := os.ReadFile(opts.Key)
		if err != nil {
			return fmt.Errorf("failed to read public key file: %w", err)
		}

		provider = &signv1.VerifyRequestProvider{
			Request: &signv1.VerifyRequestProvider_Key{
				Key: &signv1.VerifyWithKey{
					PublicKey: pubKey,
				},
			},
		}

	case opts.OIDCIssuer != "" || opts.OIDCSubject != "":
		provider = &signv1.VerifyRequestProvider{
			Request: &signv1.VerifyRequestProvider_Oidc{
				Oidc: &signv1.VerifyWithOIDC{
					Issuer:  opts.OIDCIssuer,
					Subject: opts.OIDCSubject,
					Options: &signv1.VerifyOptionsOIDC{
						TufMirrorUrl:    opts.TufMirrorUrl,
						TrustedRootPath: opts.TrustedRootPath,
						IgnoreTlog:      opts.IgnoreTlog,
						IgnoreTsa:       opts.IgnoreTsa,
						IgnoreSct:       opts.IgnoreSct,
					},
				},
			},
		}

	default:
		// Use VerifyWithAny which will verify against any valid signature
		// with optional OIDC verification options
		provider = &signv1.VerifyRequestProvider{
			Request: &signv1.VerifyRequestProvider_Any{
				Any: &signv1.VerifyWithAny{
					OidcOptions: &signv1.VerifyOptionsOIDC{
						TufMirrorUrl:    opts.TufMirrorUrl,
						TrustedRootPath: opts.TrustedRootPath,
						IgnoreTlog:      opts.IgnoreTlog,
						IgnoreTsa:       opts.IgnoreTsa,
						IgnoreSct:       opts.IgnoreSct,
					},
				},
			},
		}
	}

	// Perform client-side verification
	response, err := c.Verify(cmd.Context(), &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{Cid: recordRef},
		Provider:  provider,
	})
	if err != nil {
		return fmt.Errorf("failed to verify record: %w", err)
	}

	// If output file is specified, write JSON directly to file and return
	// This takes priority over other output options to avoid stdout pollution
	if opts.OutputFile != "" {
		return presenter.WriteMessageToFile(opts.OutputFile, response)
	}

	// Output in the appropriate format
	opts := presenter.GetOutputOptions(cmd)
	if opts.Format == presenter.FormatHuman {
		if response.GetSuccess() {
			presenter.Println(cmd, "Record signature is: trusted")

			// Show signers info
			signers := response.GetSigners()
			if len(signers) > 0 {
				presenter.Println(cmd, fmt.Sprintf("Found %d valid signer(s):", len(signers)))

				for i, signer := range signers {
					if keyInfo := signer.GetKey(); keyInfo != nil {
						presenter.Println(cmd, fmt.Sprintf("  [%d] Type: key", i+1))
						// Truncate public key for display
						pubKey := keyInfo.GetPublicKey()
						if len(pubKey) > 60 {
							pubKey = pubKey[:60] + "..."
						}

						presenter.Println(cmd, fmt.Sprintf("       Public Key: %s", pubKey))
					} else if oidcInfo := signer.GetOidc(); oidcInfo != nil {
						presenter.Println(cmd, fmt.Sprintf("  [%d] Type: oidc", i+1))
						presenter.Println(cmd, fmt.Sprintf("       Issuer: %s", oidcInfo.GetIssuer()))
						presenter.Println(cmd, fmt.Sprintf("       Identity: %s", oidcInfo.GetSubject()))
					}
				}
			}
		} else if response.GetErrorMessage() != "" {
			presenter.Println(cmd, "Record signature is: not trusted")
			presenter.Println(cmd, fmt.Sprintf("Reason: %s", response.GetErrorMessage()))
		}

		return nil
	}

	// For structured output formats, print the full response as JSON
	return presenter.PrintMessage(cmd, "", "", response)
}

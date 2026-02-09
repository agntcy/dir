// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package verify

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var (
	keyFlag          string
	oidcIssuerFlag   string
	oidcIdentityFlag string
)

func init() {
	// Add output format flags
	presenter.AddOutputFlags(Command)

	// Add verification option flags
	Command.Flags().StringVar(&keyFlag, "key", "", "PEM-encoded public key to verify against")
	Command.Flags().StringVar(&oidcIssuerFlag, "oidc-issuer", "", "OIDC issuer URL to verify against (e.g., https://github.com/login/oauth)")
	Command.Flags().StringVar(&oidcIdentityFlag, "oidc-identity", "", "OIDC identity/subject to verify against (e.g., user@example.com)")
}

//nolint:mnd
var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify record signature against identity-based OIDC or key-based signing",
	Long: `This command verifies the record signature against
identity-based OIDC or key-based signing process.

Usage examples:

1. Verify a record signature (any valid signature):

	dirctl verify <record-cid>

2. Verify against a specific public key:

	dirctl verify <record-cid> --key "$(cat pubkey.pem)"

3. Verify against OIDC identity:

	dirctl verify <record-cid> --oidc-issuer https://github.com/login/oauth --oidc-identity user@example.com

4. Verify against just OIDC issuer (any identity from that issuer):

	dirctl verify <record-cid> --oidc-issuer https://github.com/login/oauth

5. Output formats:

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
		}

		// Validate flag combinations
		if keyFlag != "" && (oidcIssuerFlag != "" || oidcIdentityFlag != "") {
			return errors.New("cannot use --key with --oidc-issuer or --oidc-identity")
		}

		if oidcIdentityFlag != "" && oidcIssuerFlag == "" {
			return errors.New("--oidc-identity requires --oidc-issuer to be set")
		}

		return runCommand(cmd, recordRef)
	},
}

// nolint:mnd
func runCommand(cmd *cobra.Command, recordRef string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Build verification request with options
	req := &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordRef,
		},
	}

	// Add verification options based on flags
	if keyFlag != "" {
		req.Options = &signv1.VerifyOptions{
			VerificationType: &signv1.VerifyOptions_Key{
				Key: &signv1.VerifyWithPublicKey{
					PublicKey: keyFlag,
				},
			},
		}
	} else if oidcIssuerFlag != "" {
		req.Options = &signv1.VerifyOptions{
			VerificationType: &signv1.VerifyOptions_Oidc{
				Oidc: &signv1.VerifyWithOIDCIdentity{
					Issuer:   oidcIssuerFlag,
					Identity: oidcIdentityFlag,
				},
			},
		}
	}

	response, err := c.Verify(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to verify record: %w", err)
	}

	// Output in the appropriate format
	status := "trusted"
	if !response.GetSuccess() {
		status = "not trusted"
	}

	opts := presenter.GetOutputOptions(cmd)
	if opts.Format == presenter.FormatHuman {
		presenter.Println(cmd, fmt.Sprintf("Record signature is: %s", status))

		if response.GetSuccess() {
			// Show signers info (new format)
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
						presenter.Println(cmd, fmt.Sprintf("       Identity: %s", oidcInfo.GetIdentity()))
					}
				}
			}

			// Also show legacy metadata if present
			metadata := response.GetSignerMetadata()
			if len(metadata) > 0 && len(signers) == 0 {
				presenter.Println(cmd, "Signer Metadata:")

				for k, v := range metadata {
					presenter.Println(cmd, fmt.Sprintf("  %s: %s", k, v))
				}
			}
		}

		return nil
	}

	result := map[string]any{
		"status":  status,
		"success": response.GetSuccess(),
	}

	if response.GetSuccess() {
		// Include signers in JSON output
		signers := response.GetSigners()
		if len(signers) > 0 {
			signersData := make([]map[string]any, 0, len(signers))
			for _, signer := range signers {
				signerData := make(map[string]any)
				if keyInfo := signer.GetKey(); keyInfo != nil {
					signerData["type"] = "key"
					signerData["public_key"] = keyInfo.GetPublicKey()
				} else if oidcInfo := signer.GetOidc(); oidcInfo != nil {
					signerData["type"] = "oidc"
					signerData["issuer"] = oidcInfo.GetIssuer()
					signerData["identity"] = oidcInfo.GetIdentity()
				}
				signersData = append(signersData, signerData)
			}
			result["signers"] = signersData
		}

		if metadata := response.GetSignerMetadata(); len(metadata) > 0 {
			result["signer_metadata"] = metadata
		}
	}

	return presenter.PrintMessage(cmd, "signature", "Record signature is", result)
}

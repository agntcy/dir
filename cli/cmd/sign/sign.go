// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	utils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "sign",
	Short: "Sign record using identity-based OIDC or key-based signing",
	Long: `This command signs the record using identity-based signing.
It uses a short-lived signing certificate issued by Sigstore Fulcio
along with a local ephemeral signing key and OIDC identity.

Verification data is attached to the signed record,
and the transparency log is pushed to Sigstore Rekor.

This command opens a browser window to authenticate the user
with the default OIDC provider.

Usage examples:

1. Sign a record from file:

	dirctl sign record.json

2. Sign a record from standard input:

	cat record.json | dirctl sign --stdin

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var fpath string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			fpath = args[0]
		}

		// get source
		source, err := utils.GetReader(fpath, opts.FromStdin)
		if err != nil {
			return fmt.Errorf("failed to get reader: %w", err)
		}

		return runCommand(cmd, source)
	},
}

func runCommand(cmd *cobra.Command, source io.ReadCloser) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Load OASF data (supports v1, v2, v3) into a Record
	record, err := corev1.LoadOASFFromReader(source)
	if err != nil {
		return fmt.Errorf("failed to load OASF: %w", err)
	}

	var signature *signv1.Signature

	//nolint:nestif,gocritic
	if opts.Key != "" {
		// Load the key from file
		rawKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		// Read password from environment variable
		pw, err := cosign.ReadPrivateKeyPassword()()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		req := &signv1.SignRequest{
			Record: record,
			Provider: &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Key{
					Key: &signv1.SignWithKey{
						PrivateKey: rawKey,
						Password:   pw,
					},
				},
			},
		}

		// Sign the record using the provided key
		response, err := c.SignWithKey(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to sign record with key: %w", err)
		}

		signature = response.GetSignature()
	} else if opts.OIDCToken != "" {
		req := &signv1.SignRequest{
			Record: record,
			Provider: &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Oidc{
					Oidc: &signv1.SignWithOIDC{
						IdToken: opts.OIDCToken,
						Options: &signv1.SignWithOIDC_SignOpts{
							FulcioUrl:       &opts.FulcioURL,
							RekorUrl:        &opts.RekorURL,
							TimestampUrl:    &opts.TimestampURL,
							OidcProviderUrl: &opts.OIDCProviderURL,
						},
					},
				},
			},
		}

		// Sign the record using the OIDC provider
		response, err := c.SignWithOIDC(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to sign record: %w", err)
		}

		signature = response.GetSignature()
	} else {
		// Retrieve the token from the OIDC provider
		token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}

		req := &signv1.SignRequest{
			Record: record,
			Provider: &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Oidc{
					Oidc: &signv1.SignWithOIDC{
						IdToken: token.RawString,
						Options: &signv1.SignWithOIDC_SignOpts{
							FulcioUrl:       &opts.FulcioURL,
							RekorUrl:        &opts.RekorURL,
							TimestampUrl:    &opts.TimestampURL,
							OidcProviderUrl: &opts.OIDCProviderURL,
						},
					},
				},
			},
		}

		// Sign the record using the OIDC provider
		response, err := c.SignWithOIDC(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to sign record: %w", err)
		}

		signature = response.GetSignature()
	}

	// Print signature
	signatureJSON, err := json.MarshalIndent(signature, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	presenter.Print(cmd, string(signatureJSON))

	return nil
}

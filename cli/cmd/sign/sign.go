// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package sign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
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

1. Sign a record using OIDC:

	dirctl sign <record-cid>

2. Sign a record using key:

	dirctl sign <record-cid> --key <key-file>

3. Output formats:

	# Get signing result as JSON
	dirctl sign <record-cid> --output json
	
	# Sign with key and JSON output
	dirctl sign <record-cid> --key <key-file> --output json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var recordCID string
		if len(args) > 1 {
			return errors.New("only one record CID is allowed")
		} else if len(args) == 1 {
			recordCID = args[0]
		} else {
			return errors.New("record CID is required")
		}

		return runCommand(cmd, recordCID)
	},
}

func runCommand(cmd *cobra.Command, recordCID string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	err := Sign(cmd.Context(), c, recordCID)
	if err != nil {
		return fmt.Errorf("failed to sign record: %w", err)
	}

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "signature", "Record is", "signed")
}

func Sign(ctx context.Context, c *client.Client, recordCID string) error {
	switch {
	case opts.Key != "":
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
			RecordRef: &corev1.RecordRef{Cid: recordCID},
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
		_, err = c.SignWithKey(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to sign record with key: %w", err)
		}
	case opts.OIDCToken != "":
		req := &signv1.SignRequest{
			RecordRef: &corev1.RecordRef{Cid: recordCID},
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
		_, err := c.SignWithOIDC(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to sign record: %w", err)
		}
	default:
		// Retrieve the token from the OIDC provider
		token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}

		req := &signv1.SignRequest{
			RecordRef: &corev1.RecordRef{Cid: recordCID},
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
		_, err = c.SignWithOIDC(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to sign record: %w", err)
		}
	}

	return nil
}

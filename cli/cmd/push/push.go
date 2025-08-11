// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupword
package push

import (
	"errors"
	"fmt"
	"io"
	"os"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/cli/presenter"
	agentUtils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "push",
	Short: "Push agent data model to Directory server",
	Long: `This command pushes the agent data model to local storage layer via Directory API. The data is stored into
content-addressable object store.

Usage examples:

1. From agent data model file:

	dirctl push model.json

2. Data from standard input. Useful for piping:

	cat model.json | dirctl push --stdin

3. Push with signature:

	dirctl push model.json --sign

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var path string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			path = args[0]
		}

		// get source
		source, err := agentUtils.GetReader(path, opts.FromStdin)
		if err != nil {
			return err //nolint:wrapcheck // Error is already wrapped
		}

		return runCommand(cmd, source)
	},
}

func runCommand(cmd *cobra.Command, source io.ReadCloser) error {
	defer source.Close()

	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Override client registry config with command-specific flags if provided
	if opts.RegistryAddress != "" {
		c.SetRegistryAddress(opts.RegistryAddress)
	}

	if opts.RepositoryName != "" {
		c.SetRepositoryName(opts.RepositoryName)
	}

	// Load OASF data (supports v1, v2, v3) into a Record
	record, err := corev1.LoadOASFFromReader(source)
	if err != nil {
		return fmt.Errorf("failed to load OASF: %w", err)
	}

	var recordRef *corev1.RecordRef

	//nolint:nestif
	if opts.Sign {
		var err error

		var resp *storev1.PushWithOptionsResponse

		if opts.Key != "" {
			// Read the private key file content
			var privateKeyBytes []byte

			privateKeyBytes, err = os.ReadFile(opts.Key)
			if err != nil {
				return fmt.Errorf("failed to read private key file: %w", err)
			}

			resp, err = c.PushWithOptions(cmd.Context(), record, opts.Sign, &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Key{
					Key: &signv1.SignWithKey{
						PrivateKey: privateKeyBytes,
					},
				},
			})
		} else {
			// Handle OIDC token retrieval if not provided
			oidcToken := opts.OIDCToken
			if oidcToken == "" {
				// Retrieve the token from the OIDC provider
				token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
				if err != nil {
					return fmt.Errorf("failed to get OIDC token: %w", err)
				}

				oidcToken = token.RawString
			}

			resp, err = c.PushWithOptions(cmd.Context(), record, opts.Sign, &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Oidc{
					Oidc: &signv1.SignWithOIDC{
						IdToken: oidcToken,
						Options: &signv1.SignWithOIDC_SignOpts{
							FulcioUrl:       &opts.FulcioURL,
							RekorUrl:        &opts.RekorURL,
							TimestampUrl:    &opts.TimestampURL,
							OidcProviderUrl: &opts.OIDCProviderURL,
						},
					},
				},
			})
		}

		if err != nil {
			return fmt.Errorf("failed to push data: %w", err)
		}

		recordRef = resp.GetRecordRef()
	} else {
		// Use the client's Push method to send the record
		recordRef, err = c.Push(cmd.Context(), record)
		if err != nil {
			return fmt.Errorf("failed to push data: %w", err)
		}
	}

	presenter.Print(cmd, recordRef.GetCid())

	return nil
}

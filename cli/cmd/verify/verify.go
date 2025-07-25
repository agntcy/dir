// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	signv1alpha1 "github.com/agntcy/dir/api/sign/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	agentUtils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify agent model signature against identity-based OIDC signing",
	Long: `This command verifies the agent data model signature against
identity-based OIDC signing process.
It relies on Sigstore Rekor for signature verification.

Usage examples:

1. Verify an agent model from file:

	dirctl verify agent.json

2. Verify an agent model from standard input:

	dirctl pull <digest> | dirctl verify --stdin

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
			return err //nolint:wrapcheck
		}

		return runCommand(cmd, source)
	},
}

// nolint:mnd
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

	// Extract v1 Agent for verification (verification service currently only supports v1)
	var agent *objectsv1.Agent
	switch data := record.GetData().(type) {
	case *corev1.Record_V1:
		agent = data.V1
	case *corev1.Record_V2:
		return errors.New("verification of OASF v0.4.0 records is not yet supported")
	case *corev1.Record_V3:
		return errors.New("verification of OASF v0.5.0 records is not yet supported")
	default:
		return errors.New("unsupported record type for verification")
	}

	//nolint:nestif
	if opts.Key != "" {
		// Load the public key from file
		rawPubKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		req := &signv1alpha1.VerifyRequest{
			Agent: agent,
			Provider: &signv1alpha1.VerifyRequestProvider{
				Provider: &signv1alpha1.VerifyRequestProvider_Key{
					Key: &signv1alpha1.VerifyWithKey{
						PublicKey: rawPubKey,
					},
				},
			},
		}

		// Verify the agent using the provided key
		_, err = c.VerifyWithKey(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to verify agent: %w", err)
		}
	} else {
		req := &signv1alpha1.VerifyRequest{
			Agent: agent,
			Provider: &signv1alpha1.VerifyRequestProvider{
				Provider: &signv1alpha1.VerifyRequestProvider_Oidc{
					Oidc: &signv1alpha1.VerifyWithOIDC{
						ExpectedIssuer: opts.OIDCIssuer,
						ExpectedSigner: opts.OIDCIdentity,
					},
				},
			},
		}

		// Verify the agent using the OIDC provider
		_, err := c.VerifyWithOIDC(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to verify agent: %w", err)
		}
	}

	// Print success message
	presenter.Print(cmd, "Agent signature verified successfully!")

	return nil
}

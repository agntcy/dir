// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	coretypesv2 "github.com/agntcy/dir/api/core/v1alpha2"
	signv1alpha2 "github.com/agntcy/dir/api/sign/v1alpha2"
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

	agent := &coretypes.Agent{}
	record := &coretypesv2.Record{}

	if opts.Experimental {
		// Load into an Record struct
		if _, err := record.LoadFromReader(source); err != nil {
			return fmt.Errorf("failed to load record: %w", err)
		}
	} else {
		// Load into an Agent struct
		if _, err := agent.LoadFromReader(source); err != nil {
			return fmt.Errorf("failed to load agent: %w", err)
		}
	}

	//nolint:nestif
	if opts.Key != "" {
		// Load the public key from file
		rawPubKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		if opts.Experimental {
			// Verify the record using the provided key
			req := &signv1alpha2.VerifyWithKeyRequest{
				Record:    record,
				PublicKey: rawPubKey,
			}

			_, err = c.VerifyWithKeyv2(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to verify record: %w", err)
			}
		} else {
			// Verify the agent using the provided key
			err = c.VerifyWithKey(cmd.Context(), rawPubKey, agent)
			if err != nil {
				return fmt.Errorf("failed to verify agent: %w", err)
			}
		}
	} else {
		if opts.Experimental {
			// Verify the record using the OIDC provider
			req := &signv1alpha2.VerifyOIDCRequest{
				Record:         record,
				ExpectedIssuer: opts.OIDCIssuer,
				ExpectedSigner: opts.OIDCIdentity,
			}

			_, err := c.VerifyOIDCv2(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to verify record: %w", err)
			}

			// Print success message
			presenter.Print(cmd, "Record signature verified successfully!")
		} else {
			// Verify the agent using the OIDC provider
			err := c.VerifyOIDC(cmd.Context(), opts.OIDCIssuer, opts.OIDCIdentity, agent)
			if err != nil {
				return fmt.Errorf("failed to verify agent: %w", err)
			}

			// Print success message
			presenter.Print(cmd, "Agent signature verified successfully!")
		}
	}

	return nil
}

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

func init() {
	// Add output format flags
	presenter.AddOutputFlags(Command)
}

//nolint:mnd
var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify record signature against identity-based OIDC or key-based signing",
	Long: `This command verifies the record signature against
identity-based OIDC or key-based signing process.

Usage examples:

1. Verify a record from file:

	dirctl verify <record-cid>
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var recordRef string
		if len(args) > 1 {
			return errors.New("one argument is allowed")
		} else if len(args) == 1 {
			recordRef = args[0]
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

	response, err := c.Verify(cmd.Context(), &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordRef,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to verify record with Zot: %w", err)
	}

	// Get output options
	outputOpts := presenter.GetOutputOptions(cmd)

	// Output in the appropriate format
	if !response.GetSuccess() {
		return presenter.OutputSingleValue(cmd, outputOpts, "status", "Record signature is", "not trusted")
	}

	return presenter.OutputSingleValue(cmd, outputOpts, "status", "Record signature is", "trusted")
}

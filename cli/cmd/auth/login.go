// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with OIDC",
	Long: `Authenticate with OIDC (OpenID Connect).

OIDC login will be implemented in a subsequent step.
`,
	RunE: runLogin,
}

func runLogin(_ *cobra.Command, _ []string) error {
	return errors.New("OIDC login not yet implemented")
}

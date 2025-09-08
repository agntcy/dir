// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"

	baseauth "github.com/agntcy/dir/hub/auth"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/spf13/cobra"
)

func CheckForCreds(cmd *cobra.Command, currentSession *sessionstore.HubSession, serverAddress string, jsonOutput bool) error {
	if !baseauth.HasLoginCreds(currentSession) && baseauth.HasAPIKey(currentSession) {
		if !jsonOutput {
			fmt.Fprintf(cmd.OutOrStdout(), "User is authenticated with API key, using it to get credentials...\n")
		}

		if err := baseauth.RefreshAPIKeyAccessToken(cmd.Context(), currentSession, serverAddress); err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	}

	if !baseauth.HasLoginCreds(currentSession) && !baseauth.HasAPIKey(currentSession) {
		return errors.New("you need to be logged to execute this action\nuse `dirctl hub login` command to login")
	}

	return nil
}

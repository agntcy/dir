// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
	"os"

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

// GetOrCreateSession gets session from context or creates in-memory session with API key with the following priority:
// 1. API key from environment variables
// 2. API key from command flags
// 3. Existing session from context (session file)
func GetOrCreateSession(cmd *cobra.Command, serverAddress, clientID, secret string) (*sessionstore.HubSession, error) {
	// Check for API key credentials in environment.
	effectiveClientID := os.Getenv("DIRCTL_CLIENT_ID")
	effectiveSecret := os.Getenv("DIRCTL_CLIENT_SECRET")

	if effectiveClientID == "" {
		effectiveClientID = clientID
	}
	if effectiveSecret == "" {
		effectiveSecret = secret
	}

	// If API key credentials are available, use in-memory session.
	if effectiveClientID != "" && effectiveSecret != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Using API key authentication...\n")
		return baseauth.CreateInMemorySessionFromAPIKey(cmd.Context(), serverAddress, effectiveClientID, effectiveSecret)
	}

	// Use existing session from context.
	ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
	currentSession, ok := ctxSession.(*sessionstore.HubSession)
	if !ok || currentSession == nil {
		return nil, errors.New("could not get current hub session")
	}

	if err := CheckForCreds(cmd, currentSession, serverAddress); err != nil {
		return nil, err
	}

	return currentSession, nil
}

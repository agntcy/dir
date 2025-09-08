// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
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

// resolveAPIKeyCredentials returns the effective clientID and secret, with JSON decoding applied if needed.
func resolveAPIKeyCredentials(clientID, secret string) (string, string, string) {
	effectiveClientID := clientID
	effectiveSecret := secret

	// Decode JSON-encoded strings from CLI flags if both are provided.
	if effectiveClientID != "" && effectiveSecret != "" {
		var decodedSecret string
		if err := json.Unmarshal([]byte(`"`+effectiveSecret+`"`), &decodedSecret); err == nil {
			effectiveSecret = decodedSecret
		}

		return effectiveClientID, effectiveSecret, "command opts"
	}

	// Only use env vars if both are provided
	envClientID := os.Getenv("DIRCTL_CLIENT_ID")
	envSecret := os.Getenv("DIRCTL_CLIENT_SECRET")

	if envClientID != "" && envSecret != "" {
		var decodedSecret string
		if err := json.Unmarshal([]byte(`"`+envSecret+`"`), &decodedSecret); err == nil {
			envSecret = decodedSecret
		}

		return envClientID, envSecret, "environment variables"
	}

	return effectiveClientID, effectiveSecret, ""
}

// GetOrCreateSession gets session from context or creates in-memory session with API key with the following priority:
// 1. API key from command opts
// 2. API key from environment variables
// 3. Existing session from context (session file created via 'dirctl hub login').
func GetOrCreateSession(cmd *cobra.Command, serverAddress, clientID, secret string, jsonOutput bool) (*sessionstore.HubSession, error) {
	effectiveClientID, effectiveSecret, source := resolveAPIKeyCredentials(clientID, secret)
	if source == "" {
		source = "session file"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Using API key authentication from %s\n", source)

	// If API key credentials are available, use in-memory session.
	if effectiveClientID != "" && effectiveSecret != "" {
		session, err := baseauth.CreateInMemorySessionFromAPIKey(cmd.Context(), serverAddress, effectiveClientID, effectiveSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to create in-memory session: %w", err)
		}

		return session, nil
	}

	// Use existing session from context.
	ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
	currentSession, ok := ctxSession.(*sessionstore.HubSession)

	if !ok || currentSession == nil {
		return nil, errors.New("could not get current hub session")
	}

	if err := CheckForCreds(cmd, currentSession, serverAddress, jsonOutput); err != nil {
		// this error need to be return without modification in order to be displayed
		return nil, err //nolint:wrapcheck
	}

	return currentSession, nil
}

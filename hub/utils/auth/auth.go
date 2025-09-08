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

// Obtain the effective clientID and secret with JSON decoding
func resolveAPIKeyCredentials(clientID, secret string) (string, string, string) {
	// Start with CLI flags as highest priority.
	effectiveClientID := clientID
	effectiveSecret := secret

	// Decode JSON-encoded strings from CLI flags if both are provided.
	if effectiveClientID != "" && effectiveSecret != "" {
		var decodedClientID, decodedSecret string
		if err := json.Unmarshal([]byte(effectiveClientID), &decodedClientID); err == nil {
			effectiveClientID = decodedClientID
		}
		if err := json.Unmarshal([]byte(effectiveSecret), &decodedSecret); err == nil {
			effectiveSecret = decodedSecret
		}
		return effectiveClientID, effectiveSecret, "command flags"
	}

	// Use environment variables if CLI flags are not both provided.
	envClientID := os.Getenv("DIRCTL_CLIENT_ID")
	envSecret := os.Getenv("DIRCTL_CLIENT_SECRET")

	// Only use env vars if both are provided
	if envClientID != "" && envSecret != "" {
		var decodedClientID, decodedSecret string
		if err := json.Unmarshal([]byte(envClientID), &decodedClientID); err == nil {
			envClientID = decodedClientID
		}
		if err := json.Unmarshal([]byte(envSecret), &decodedSecret); err == nil {
			envSecret = decodedSecret
		}
		return envClientID, envSecret, "environment variables"
	}

	return effectiveClientID, effectiveSecret, ""
}

// GetOrCreateSession gets session from context or creates in-memory session with API key with the following priority:
// 1. API key from command flags
// 2. API key from environment variables
// 3. Existing session from context (session file created via 'dirctl hub login')
func GetOrCreateSession(cmd *cobra.Command, serverAddress, clientID, secret string) (*sessionstore.HubSession, error) {
	effectiveClientID, effectiveSecret, source := resolveAPIKeyCredentials(clientID, secret)

	// If API key credentials are available, use in-memory session.
	if effectiveClientID != "" && effectiveSecret != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Using API key authentication from %s\n", source)
		return baseauth.CreateInMemorySessionFromAPIKey(cmd.Context(), serverAddress, effectiveClientID, effectiveSecret)
	}
	fmt.Print(cmd.OutOrStdout(), "Using API key authentication from sessionf file")
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

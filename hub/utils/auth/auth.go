// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,unparam
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	baseauth "github.com/agntcy/dir/hub/auth"
	"github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/spf13/cobra"
)

func CheckForCreds(cmd *cobra.Command, currentSession *sessionstore.HubSession, serverAddress string, jsonOutput bool) error {
	if !baseauth.HasLoginCreds(currentSession) {
		return errors.New("you need to be logged to execute this action\nuse `dirctl hub login` command to login")
	}

	return nil
}

// resolveAPIKeyCredentials returns the effective clientID and secret (base64-encoded).
// It checks sources in the following order of priority:
//  1. API key file (if provided)
//  2. Environment variables
func resolveAPIKeyCredentials(apikeyFile string) (string, string, error) {
	// Check API key file if provided
	if apikeyFile != "" {
		fileContent, err := os.ReadFile(apikeyFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read API key file: %w", err)
		}

		// Parse JSON file content to extract clientID and secret
		var apiKey service.APIKeyWithSecret
		if err := json.Unmarshal(fileContent, &apiKey); err != nil {
			return "", "", fmt.Errorf("failed to parse API key file as JSON: %w", err)
		}

		if apiKey.ClientID == "" || apiKey.Secret == "" {
			return "", "", errors.New("API key file must contain both client_id and secret fields")
		}

		return apiKey.ClientID, apiKey.Secret, nil
	}

	// Environment variables
	envClientID := os.Getenv("DIRCTL_CLIENT_ID")
	envSecret := os.Getenv("DIRCTL_CLIENT_SECRET")

	if envClientID != "" && envSecret != "" {
		return envClientID, envSecret, nil
	}

	// Return empty values if no authentication source was found
	// This will trigger the session-based authentication flow
	return "", "", nil
}

// GetOrCreateSession gets session from context or creates in-memory session with API key with the following priority:
// 1. API key from file (if provided via apikeyFile)
// 2. API key from environment variables
// 3. Existing session from context (session file created via 'dirctl hub login').
// Secret must be provided as base64-encoded.
func GetOrCreateSession(cmd *cobra.Command, serverAddress, apikeyFile string, jsonOutput bool) (*sessionstore.HubSession, error) {
	effectiveClientID, effectiveSecret, err := resolveAPIKeyCredentials(apikeyFile)
	if err != nil {
		return nil, err
	}

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

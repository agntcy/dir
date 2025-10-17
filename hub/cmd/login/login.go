// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package login provides the CLI command for logging in to the Agent Hub.
package login

import (
	"errors"
	"fmt"

	saasv1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	auth "github.com/agntcy/dir/hub/auth"
	authUtils "github.com/agntcy/dir/hub/auth/utils"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	"github.com/agntcy/dir/hub/utils/http"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewCommand creates the "login" command for the Agent Hub CLI.
// It handles user authentication, session creation, and token storage.
// Returns the configured *cobra.Command.
func NewCommand(hubOptions *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "login",
		Short:            "Login to the Agent Hub",
		TraverseChildren: true,
	}

	opts := options.NewLoginOptions(hubOptions)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || currentSession == nil {
			return errors.New("failed to get current session from context")
		}
		// Load session store for saving
		sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())
		// Construct Okta client
		oktaClient := okta.NewClient(currentSession.IdpIssuerAddress, http.CreateSecureHTTPClient())
		// Call auth.Login with loaded objects
		updatedSession, err := auth.Login(cmd.Context(), oktaClient, currentSession)
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}

		hc, err := hubClient.New(currentSession.HubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		ctx := authUtils.AddAuthToContext(cmd.Context(), updatedSession)

		var loginSuccess bool

		defer func() {
			if !loginSuccess {
				if logoutErr := auth.Logout(hubOptions, currentSession, sessionStore, oktaClient); logoutErr != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Warning: Failed to logout after error: %v\n", logoutErr)
				}
			}
		}()

		// Register user with the Agent Hub (if not already registered)
		_, err = hc.GetUser(ctx, &saasv1alpha1.GetUserRequest{})
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
				fmt.Fprintf(cmd.OutOrStdout(), "User not registered yet. Self-registering user with username \"%s\"\n", updatedSession.User)

				_, err = hc.CreateUser(ctx, &saasv1alpha1.CreateUserRequest{})
				if err != nil {
					return fmt.Errorf("failed to register user: %w", err)
				}
			}
		}

		if err := sessionStore.SaveHubSession(opts.ServerAddress, updatedSession); err != nil {
			return fmt.Errorf("failed to save tokens: %w", err)
		}

		loginSuccess = true

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully logged in to Agent Hub\nAddress: %s\nUser: %s\nClientID: %s\n",
			opts.ServerAddress, updatedSession.User, updatedSession.ClientID)

		return nil
	}

	return cmd
}

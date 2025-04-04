// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package login

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/agntcy/dir/cli/config"
	configUtils "github.com/agntcy/dir/cli/hub/config"
	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/hub/secretstore"
	"github.com/agntcy/dir/cli/hub/webserver"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

const timeout = 60 * time.Second

func NewCommand(hubOptions *options.HubOptions) *cobra.Command {
	opts := options.NewLoginOptions(hubOptions)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the Agent Hub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get secret store from context
			secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return errors.New("failed to get secret store from context")
			}

			// Get auth config
			authConfig, err := configUtils.FetchAuthConfig(opts.ServerAddress)
			if err != nil {
				return fmt.Errorf("failed to fetch auth config: %w", err)
			}

			// Init IDP client
			idpClient := idp.NewClient(authConfig.IdpIssuerAddress)

			return runCmd(cmd, opts, idpClient, authConfig, secretStore)
		},
		TraverseChildren: true,
	}

	return cmd
}

func runCmd(cmd *cobra.Command, opts *options.LoginOptions, idpClient idp.Client, authConfig *configUtils.AuthConfig, secretStore secretstore.SecretStore) error {
	// Set up the webserver
	//// Init the error channel
	errCh := make(chan error, 1)

	//// Init session store
	sessionStore := &webserver.SessionStore{}

	handler := webserver.NewHandler(&webserver.Config{
		ClientID:           authConfig.ClientID,
		IdpFrontendURL:     authConfig.IdpFrontendAddress,
		IdpBackendURL:      authConfig.IdpBackendAddress,
		LocalWebserverPort: config.LocalWebserverPort,
		IdpClient:          idpClient,
		SessionStore:       sessionStore,
		ErrChan:            errCh,
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	server := webserver.StartLocalServer(handler, config.LocalWebserverPort, errCh)
	defer server.Shutdown(ctx) //nolint:errcheck

	// Open the browser
	if err := openBrowser(authConfig); err != nil {
		return err
	}

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	var err error
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	// Get tokens
	err = secretStore.SaveHubSecret(opts.ServerAddress, &secretstore.HubSecret{
		AuthConfig: &secretstore.AuthConfig{
			ClientID:           authConfig.ClientID,
			ProductID:          authConfig.IdpProductID,
			IdpFrontendAddress: authConfig.IdpFrontendAddress,
			IdpBackendAddress:  authConfig.IdpBackendAddress,
			IdpIssuerAddress:   authConfig.IdpIssuerAddress,
			HubBackendAddress:  authConfig.HubBackendAddress,
		},
		TokenSecret: &secretstore.TokenSecret{
			AccessToken:  sessionStore.Tokens.AccessToken,
			IDToken:      sessionStore.Tokens.IDToken,
			RefreshToken: sessionStore.Tokens.RefreshToken,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully logged in to Agent Hub\nAddress: %s\n", opts.ServerAddress)

	return nil
}

func openBrowser(authConfig *configUtils.AuthConfig) error {
	params := url.Values{}
	params.Add("redirectUri", fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort))
	loginPageWithRedirect := fmt.Sprintf("%s/%s/login?%s", authConfig.IdpFrontendAddress, authConfig.IdpProductID, params.Encode())

	return browser.OpenURL(loginPageWithRedirect) //nolint:wrapcheck
}

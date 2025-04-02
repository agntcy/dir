package login

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/cli/hub/auth"
	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/hub/webserver"
	"github.com/agntcy/dir/cli/options"
	"github.com/agntcy/dir/cli/secretstore"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

func NewCommand(hubOptions *options.HubOptions) *cobra.Command {
	opts := options.NewLoginOptions(hubOptions)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the Agent Hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get secret store from context
			secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get secret store from context")
			}

			// Get auth config
			authConfig, err := auth.FetchAuthConfig(opts.ServerAddress)
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

func runCmd(cmd *cobra.Command, opts *options.LoginOptions, idpClient idp.Client, authConfig *auth.AuthConfig, secretStore secretstore.SecretStore) error {

	// Set up the webserver
	//// Init the error channel
	errCh := make(chan error, 1)

	//// Init session store
	sessionStore := &webserver.SessionStore{}

	handler := webserver.NewHandler(&webserver.Config{
		ClientId:           authConfig.ClientId,
		IdpFrontendUrl:     authConfig.IdpFrontendAddress,
		IdpBackendUrl:      authConfig.IdpBackendAddress,
		LocalWebserverPort: config.LocalWebserverPort,
		IdpClient:          idpClient,
		SessionStore:       sessionStore,
		ErrChan:            errCh,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	server := webserver.StartLocalServer(handler, config.LocalWebserverPort, errCh)
	defer server.Shutdown(ctx)

	// Open the browser
	if err := openBrowser(); err != nil {
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
			ClientId:           authConfig.ClientId,
			ProductId:          authConfig.IdpProductId,
			IdpFrontendAddress: authConfig.IdpFrontendAddress,
			IdpBackendAddress:  authConfig.IdpBackendAddress,
			IdpIssuerAddress:   authConfig.IdpIssuerAddress,
			HubBackendAddress:  authConfig.HubBackendAddress,
		},
		TokenSecret: &secretstore.TokenSecret{
			AccessToken:  sessionStore.Tokens.AccessToken,
			IdToken:      sessionStore.Tokens.IdToken,
			RefreshToken: sessionStore.Tokens.RefreshToken,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully logged in to Agent Hub\nAddress: %s\n", opts.ServerAddress)
	return nil
}

func openBrowser() error {
	params := url.Values{}
	params.Add("redirectUri", fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort))
	loginPageWithRedirect := fmt.Sprintf("%s?%s", config.DefaultLoginPageAddress, params.Encode())
	return browser.OpenURL(loginPageWithRedirect)
}

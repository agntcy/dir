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
	"github.com/agntcy/dir/cli/hub/webserver"
	"github.com/agntcy/dir/cli/options"
	"github.com/agntcy/dir/cli/secretstore"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

func NewLoginCommand(hubOptions *options.HubOptions) *cobra.Command {
	opts := options.NewLoginOptions(hubOptions)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the Phoenix SaaS hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Complete()
			if err := opts.CheckError(); err != nil {
				return err
			}

			return run(cmd, opts.ServerAddress, opts)
		},
		TraverseChildren: true,
	}

	return cmd
}

func run(cmd *cobra.Command, frontendUrl string, opts *options.LoginOptions) error {

	// Set up the webserver
	//// Init the error channel
	errCh := make(chan error, 1)

	//// Get secret store from context
	secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("failed to get secret store from context")
	}

	//// Get auth config
	authConfig, err := auth.FetchAuthConfig(frontendUrl)
	if err != nil {
		return err
	}

	//// Init session store
	sessionStore := &webserver.SessionStore{}

	handler := webserver.NewHandler(&webserver.Config{
		ClientId:           authConfig.ClientId,
		FrontendUrl:        frontendUrl,
		IdpUrl:             authConfig.IdpAddress,
		LocalWebserverPort: config.LocalWebserverPort,
		SessionStore:       sessionStore,
		ErrChan:            errCh,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	server := webserver.StartLocalServer(handler, config.LocalWebserverPort, errCh)
	defer server.Shutdown(ctx)

	// Open the browser
	if err = openBrowser(); err != nil {
		return err
	}

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}

	// Get tokens
	err = secretStore.SaveHubSecret(frontendUrl, &secretstore.HubSecret{
		ClientId:   authConfig.ClientId,
		IdpAddress: authConfig.IdpAddress,
		TokenSecret: &secretstore.TokenSecret{
			AccessToken:  sessionStore.Tokens.AccessToken,
			IdToken:      sessionStore.Tokens.IdToken,
			RefreshToken: sessionStore.Tokens.RefreshToken,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully logged in to Phoenix SaaS hub\nAddress: %s\n", frontendUrl)
	return nil
}

func openBrowser() error {
	params := url.Values{}
	params.Add("redirectUri", fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort))
	loginPageWithRedirect := fmt.Sprintf("%s?%s", config.DefaultLoginPageAddress, params.Encode())
	return browser.OpenURL(loginPageWithRedirect)
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tenantswitch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"time"

	"github.com/agntcy/dir/hub/browser"
	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	ctxUtils "github.com/agntcy/dir/hub/utils/context"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/agntcy/dir/hub/utils/token"
	"github.com/agntcy/dir/hub/webserver"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const timeout = 60 * time.Second

func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch [flags]",
		Short: "Switch between tenants of logged-in user",
		Long: `
This command help switching between logged-in user's tenants. You need to log in first with
'dirctl hub login' command. If --tenant flag is specified user will be logged in to the specified
tenant. In any other case, tenant could be selected from an interactive list.
`,
	}

	opts := options.NewTenantSwitchOptions(hubOpts, cmd)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Token is checked and refreshed and authorized in the persistent prerun of tenants command
		tenants, ok := ctxUtils.GetUserTenantsFromContext(cmd)
		if !ok {
			return errors.New("could not get user tenants")
		}

		sessionStore, ok := ctxUtils.GetSessionStoreFromContext(cmd)
		if !ok {
			return errors.New("could not get session store")
		}

		currentSession, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
		if !ok {
			return errors.New("could not get current hub session")
		}

		oktaClient := okta.NewClient(currentSession.AuthConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())

		tenant, err := switchTenant(opts, tenants, currentSession, sessionStore, oktaClient)

		return handleOutput(cmd.OutOrStdout(), cmd.OutOrStderr(), tenant, err)
	}

	return cmd
}

func switchTenant( //nolint:cyclop
	opts *options.TenantSwitchOptions,
	tenants []*idp.TenantResponse,
	currentSession *sessionstore.HubSession,
	sessionStore sessionstore.SessionStore,
	oktaClient okta.Client,
) (string, error) {
	// If no tenant specified, show selector
	var selectedTenant string
	if opts.Tenant != "" {
		selectedTenant = opts.Tenant
	}

	tenantsMap := tenantsToMap(tenants)
	if selectedTenant == "" {
		s := promptui.Select{
			Label: "Tenants",
			Items: slices.Collect(maps.Keys(tenantsMap)),
		}

		var err error

		_, selectedTenant, err = s.Run()
		if err != nil {
			return "", fmt.Errorf("interactive selection error: %w", err)
		}
	}

	if selectedTenant == currentSession.CurrentTenant {
		return selectedTenant, nil
	}

	if _, ok := currentSession.Tokens[selectedTenant]; ok {
		if !token.IsTokenExpired(currentSession.Tokens[selectedTenant].AccessToken) {
			currentSession.CurrentTenant = selectedTenant
			if err := sessionStore.SaveHubSession(opts.ServerAddress, currentSession); err != nil {
				return "", fmt.Errorf("could not save session: %w", err)
			}

			return selectedTenant, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	webserverSession := &webserver.SessionStore{}
	errChan := make(chan error, 1)
	h := webserver.NewHandler(&webserver.Config{
		ClientID:           currentSession.ClientID,
		IdpFrontendURL:     currentSession.IdpFrontendAddress,
		IdpBackendURL:      currentSession.IdpBackendAddress,
		LocalWebserverPort: config.LocalWebserverPort,
		SessionStore:       webserverSession,
		OktaClient:         oktaClient,
		ErrChan:            errChan,
	})

	server, err := webserver.StartLocalServer(h, config.LocalWebserverPort, errChan)
	if err != nil {
		var errChanErr error
		if len(errChan) > 0 {
			errChanErr = <-errChan
		}

		if server != nil {
			server.Shutdown(ctx) //nolint:errcheck
		}

		return "", fmt.Errorf("could not start local webserver: %w. error from webserver: %w", err, errChanErr)
	}

	defer server.Shutdown(ctx) //nolint:errcheck

	selectedTenantID := tenantsMap[selectedTenant]
	if err = browser.OpenBrowserForSwitch(currentSession.AuthConfig, selectedTenantID); err != nil {
		return "", fmt.Errorf("could not open browser: %w", err)
	}

	select {
	case err = <-errChan:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return "", fmt.Errorf("failed to get tokens: %w", err)
	}

	newTenant, err := token.GetTenantNameFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant name from token: %w", err)
	}

	if newTenant != selectedTenant {
		return "", fmt.Errorf("tenant name from token (%s) does not match selected tenant (%s). it could happen because you logged in another account then the one that has the requested tenant", newTenant, selectedTenant)
	}

	currentSession.CurrentTenant = selectedTenant
	currentSession.Tokens[selectedTenant] = &sessionstore.Tokens{
		IDToken:      webserverSession.Tokens.IDToken,
		RefreshToken: webserverSession.Tokens.RefreshToken,
		AccessToken:  webserverSession.Tokens.AccessToken,
	}

	if err = sessionStore.SaveHubSession(opts.ServerAddress, currentSession); err != nil {
		return "", fmt.Errorf("could not save session to session store: %w", err)
	}

	return selectedTenant, err //nolint:wrapcheck
}

func tenantsToMap(tenants []*idp.TenantResponse) map[string]string {
	m := make(map[string]string, len(tenants))
	for _, tenant := range tenants {
		m[tenant.Name] = tenant.ID
	}

	return m
}

func handleOutput(stdin io.Writer, stdout io.Writer, selectedTenant string, err error) error {
	if err == nil {
		fmt.Fprintf(stdin, "Successfully switched to %s\n", selectedTenant)

		return nil
	}

	fmt.Fprintf(stdout, "An error occoured during tenant switch. Try to call `dirctl hub login` to solve the issue.\nError details: \n")

	return err
}

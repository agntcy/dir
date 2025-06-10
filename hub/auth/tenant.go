package auth

import (
	"context"
	"fmt"
	"io"
	"time"

	"maps"
	"slices"

	"github.com/agntcy/dir/hub/auth/internal/browser"
	"github.com/agntcy/dir/hub/auth/internal/webserver"
	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/agntcy/dir/hub/utils/token"
	"github.com/manifoldco/promptui"
)

const switchTimeout = 60 * time.Second

func SwitchTenant(
	out io.Writer,
	opts *options.TenantSwitchOptions,
	tenants []*idp.TenantResponse,
	currentSession *sessionstore.HubSession,
	oktaClient okta.Client,
) (*sessionstore.HubSession, error) {
	var selectedTenant string
	if opts.Org != "" {
		selectedTenant = opts.Org
	}

	tenantsMap := tenantsToMap(tenants)
	if selectedTenant == "" {
		s := promptui.Select{
			Label: "Organizations",
			Items: slices.Collect(maps.Keys(tenantsMap)),
		}

		var err error

		_, selectedTenant, err = s.Run()
		if err != nil {
			return nil, fmt.Errorf("interactive selection error: %w", err)
		}
	}

	if selectedTenant == currentSession.CurrentTenant {
		fmt.Fprintf(out, "Already on tenant: %s\n", selectedTenant)

		return currentSession, nil
	}

	if _, ok := currentSession.Tokens[selectedTenant]; ok {
		if !token.IsTokenExpired(currentSession.Tokens[selectedTenant].AccessToken) {
			currentSession.CurrentTenant = selectedTenant
			fmt.Fprintf(out, "Switched to tenant: %s\n", selectedTenant)
			return currentSession, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), switchTimeout)
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
		return nil, fmt.Errorf("could not start local webserver: %w. error from webserver: %w", err, errChanErr)
	}
	defer server.Shutdown(ctx) //nolint:errcheck

	selectedTenantID := tenantsMap[selectedTenant]
	if err = browser.OpenBrowserForSwitch(currentSession.AuthConfig, selectedTenantID); err != nil {
		return nil, fmt.Errorf("could not open browser: %w", err)
	}

	select {
	case err = <-errChan:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	newTenant, err := token.GetTenantNameFromToken(webserverSession.Tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get org name from token: %w", err)
	}

	if newTenant != selectedTenant {
		return nil, fmt.Errorf("org name from token (%s) does not match selected org (%s). it could happen because you logged in another account then the one that has the requested org", newTenant, selectedTenant)
	}

	currentSession.CurrentTenant = selectedTenant
	currentSession.Tokens[selectedTenant] = &sessionstore.Tokens{
		IDToken:      webserverSession.Tokens.IDToken,
		RefreshToken: webserverSession.Tokens.RefreshToken,
		AccessToken:  webserverSession.Tokens.AccessToken,
	}

	fmt.Fprintf(out, "Successfully switched to %s\n", selectedTenant)
	return currentSession, nil //nolint:wrapcheck
}

func tenantsToMap(tenants []*idp.TenantResponse) map[string]string {
	m := make(map[string]string, len(tenants))
	for _, tenant := range tenants {
		m[tenant.Name] = tenant.ID
	}
	return m
}

func FetchUserTenants(currentSession *sessionstore.HubSession) ([]*idp.TenantResponse, error) {
	idpClient := idp.NewClient(currentSession.AuthConfig.IdpBackendAddress, httpUtils.CreateSecureHTTPClient())
	accessToken := currentSession.Tokens[currentSession.CurrentTenant].AccessToken
	productID := currentSession.AuthConfig.IdpProductID
	idpResp, err := idpClient.GetTenantsInProduct(productID, idp.WithBearerToken(accessToken))
	if err != nil {
		return nil, err
	}
	if idpResp.TenantList == nil {
		return nil, fmt.Errorf("no tenants found")
	}
	return idpResp.TenantList.Tenants, nil
}

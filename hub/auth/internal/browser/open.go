// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package browser

import (
	"fmt"
	"net/url"

	"github.com/agntcy/dir/hub/auth/internal/webserver"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	oktaUtils "github.com/okta/samples-golang/okta-hosted-login/utils"
	"github.com/pkg/browser"
)

const (
	loginPath string = "login"
)

// OpenBrowserForLogin opens the default web browser to the login page for the given AuthConfig.
func OpenBrowserForLogin(currentSession *sessionstore.HubSession, webserverSession *webserver.SessionStore, oktaClient okta.Client) error {
	if currentSession.AuthConfig == nil {
		return fmt.Errorf("authConfig is nil")
	}

	params := url.Values{}
	loginPageWithRedirect := ""
	if isIAMAuthConfig(currentSession) {
		params.Add("redirectUri", fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort))
		loginPageWithRedirect = fmt.Sprintf("%s/%s/%s?%s", currentSession.AuthConfig.IdpFrontendAddress, currentSession.AuthConfig.IdpProductID, loginPath, params.Encode())
	} else {
		nonce, _ := oktaUtils.GenerateNonce()

		loginPageWithRedirect = oktaClient.AuthorizeURL(&okta.AuthorizeRequest{
			ClientID:      currentSession.AuthConfig.ClientID,
			RedirectURI:   fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort),
			RequestID:     "%7B%22url%22%3A%22/explore%22%7D",
			S256Challenge: webserverSession.Challenge,
			Nonce:         nonce,
		})
	}

	return browser.OpenURL(loginPageWithRedirect) //nolint:wrapcheck
}

func isIAMAuthConfig(currentSession *sessionstore.HubSession) bool {
	if currentSession == nil || currentSession.AuthConfig == nil {
		return false
	}
	if currentSession.AuthConfig.IdpFrontendAddress != "" && currentSession.AuthConfig.IdpBackendAddress != "" {
		return true
	}

	return false
}

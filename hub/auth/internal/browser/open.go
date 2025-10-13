// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package browser

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/agntcy/dir/hub/auth/internal/webserver"
	authUtils "github.com/agntcy/dir/hub/auth/utils"
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
		return errors.New("authConfig is nil")
	}

	params := url.Values{}

	var loginPageWithRedirect string

	if authUtils.IsIAMAuthConfig(currentSession) {
		params.Add("redirectUri", fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort))
		loginPageWithRedirect = fmt.Sprintf("%s/%s/%s?%s", currentSession.IdpFrontendAddress, currentSession.IdpProductID, loginPath, params.Encode())
	} else {
		nonce, _ := oktaUtils.GenerateNonce()

		loginPageWithRedirect = oktaClient.AuthorizeURL(&okta.AuthorizeRequest{
			ClientID:      currentSession.ClientID,
			RedirectURI:   fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort),
			RequestID:     "%7B%22url%22%3A%22/explore%22%7D",
			S256Challenge: webserverSession.Challenge,
			Nonce:         nonce,
		})
	}

	return browser.OpenURL(loginPageWithRedirect) //nolint:wrapcheck
}

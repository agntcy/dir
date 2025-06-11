// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package browser

import (
	"fmt"
	"net/url"

	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/pkg/browser"
)

const (
	loginPath  string = "login"
	switchPath string = "tenants"
)

// OpenBrowserForSwitch opens the default web browser to the tenant switching page for the given AuthConfig.
// If a tenant name is provided, it will be preselected in the UI.
func OpenBrowserForSwitch(authConfig *sessionstore.AuthConfig, tenant ...string) error {
	tenantParam := ""
	if len(tenant) > 0 {
		tenantParam = tenant[0]
	}

	return openBrowser(authConfig, switchPath, tenantParam)
}

// OpenBrowserForLogin opens the default web browser to the login page for the given AuthConfig.
// If a tenant name is provided, it will be preselected in the UI.
func OpenBrowserForLogin(authConfig *sessionstore.AuthConfig, tenant ...string) error {
	tenantParam := ""
	if len(tenant) > 0 {
		tenantParam = tenant[0]
	}

	return openBrowser(authConfig, loginPath, tenantParam)
}

// openBrowser is an internal helper that constructs the appropriate URL and opens it in the default browser.
func openBrowser(authConfig *sessionstore.AuthConfig, path string, tenantParam string) error {
	params := url.Values{}
	params.Add("redirectUri", fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort))

	if tenantParam != "" {
		params.Add("tenant", tenantParam)
	}

	loginPageWithRedirect := fmt.Sprintf("%s/%s/%s?%s", authConfig.IdpFrontendAddress, authConfig.IdpProductID, path, params.Encode())

	return browser.OpenURL(loginPageWithRedirect) //nolint:wrapcheck
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tenants

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	hubOptions "github.com/agntcy/dir/cli/cmd/hub/options"
	"github.com/agntcy/dir/cli/cmd/hub/tenants/options"
	"github.com/agntcy/dir/cli/cmd/hub/tenantswitch"
	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/hub/token"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

const (
	selectionMark = "*"
	gapSize       = 4
)

func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenants",
		Short: "List tenants for logged in user",
	}

	opts := options.NewListTenantsOptions(hubOpts)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// Get user's tenant list
		currentSession, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd.Context())
		if !ok {
			return errors.New("no current session found. please login first")
		}

		if currentSession.Tokens == nil ||
			currentSession.CurrentTenant == "" ||
			currentSession.Tokens[currentSession.CurrentTenant] == nil ||
			currentSession.Tokens[currentSession.CurrentTenant].AccessToken == "" {
			return errors.New("access token is not found. please login first")
		}

		// Get session store from context
		sessionStore, ok := ctxUtils.GetSessionStoreFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get session store from context")
		}

		// Get the okta client from context
		oktaClient, ok := ctxUtils.GetOktaClientFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get IDP client from context")
		}

		// Refresh the access token if expired
		if err := token.RefreshTokenIfExpired(cmd, opts.ServerAddress, currentSession, sessionStore, oktaClient); err != nil {
			return fmt.Errorf("failed to refresh expired access token: %w. login to solve the issue", err)
		}

		if currentSession.AuthConfig.IdpBackendAddress == "" || currentSession.AuthConfig.IdpProductID == "" {
			return errors.New("not all auth config is fetched successfully")
		}

		idpClient := idp.NewClient(currentSession.AuthConfig.IdpBackendAddress)

		accessToken := currentSession.Tokens[currentSession.CurrentTenant].AccessToken

		tenants, err := idpClient.GetTenantsInProduct(currentSession.AuthConfig.IdpProductID, idp.WithBearerToken(accessToken))
		if err != nil {
			return fmt.Errorf("failed to get list of tenants: %w", err)
		}

		if tenants.Response.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get list of tenants: unexpected status code: %d: %s", tenants.Response.StatusCode, string(tenants.Body))
		}

		// Set the tenant list in context
		ctx := cmd.Context()
		ctx = ctxUtils.SetUserTenantsForContext(ctx, tenants.TenantList.Tenants)
		cmd.SetContext(ctx)

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Get the tenant list from context
		tenants, ok := ctxUtils.GetUserTenantsFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get tenant list from context")
		}

		currentSession, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd.Context())
		if !ok {
			return errors.New("no current session found. please login first")
		}

		return runCmd(cmd, tenants, currentSession.CurrentTenant)
	}

	cmd.AddCommand(
		tenantswitch.NewCommand(hubOpts),
	)

	return cmd
}

func runCmd(cmd *cobra.Command, tenants []*idp.TenantResponse, currentTenant string) error {
	// Print the list of tenants
	renderList(cmd.OutOrStdout(), tenants, currentTenant)

	return nil
}

type renderFn func(int, int) string

func renderList(stream io.Writer, tenants []*idp.TenantResponse, currentTenant string) {
	renderFns := make([]renderFn, len(tenants))

	longestNameLen := 0

	longestIDLen := 0

	for i, tenant := range tenants {
		if len(tenant.Name) > longestNameLen {
			longestNameLen = len(tenant.Name)
		}

		if len(tenant.ID) > longestIDLen {
			longestIDLen = len(tenant.ID)
		}

		renderFns[i] = func(lName, lId int) string {
			var selection string
			if tenant.Name == currentTenant {
				selection = selectionMark
			}

			selectionCol := text.AlignLeft.Apply(selection, len(selectionMark)+1) //nolint:mnd
			nameCol := text.AlignLeft.Apply(tenant.Name, lName+gapSize)
			idCol := text.AlignLeft.Apply(tenant.ID, lId)

			return fmt.Sprintf("%s%s%s", selectionCol, nameCol, idCol)
		}
	}

	for _, tenant := range renderFns {
		fmt.Fprintln(stream, tenant(longestNameLen, longestIDLen)) //nolint:errcheck
	}
}

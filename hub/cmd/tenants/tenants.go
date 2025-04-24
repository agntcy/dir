// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tenants

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/agntcy/dir/hub/client/idp"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/cmd/tenantswitch"
	ctxUtils "github.com/agntcy/dir/hub/utils/context"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/agntcy/dir/hub/utils/token"
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

	opts := hubOptions.NewListTenantsOptions(hubOpts)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := token.ValidateAccessTokenFromContext(cmd); err != nil {
			return errors.New("failed to validate access token. please login first or login again")
		}

		if err := token.RefreshContextTokenIfExpired(cmd, opts.HubOptions); err != nil {
			return fmt.Errorf("failed to refresh expired access token: %w", err)
		}

		currentSession, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
		if !ok {
			return errors.New("no current session found. please login first")
		}

		idpClient := idp.NewClient(currentSession.AuthConfig.IdpBackendAddress, httpUtils.CreateSecureHTTPClient())

		accessToken := currentSession.Tokens[currentSession.CurrentTenant].AccessToken

		tenantsResp, err := idpClient.GetTenantsInProduct(currentSession.AuthConfig.IdpProductID, idp.WithBearerToken(accessToken))
		if err != nil {
			return fmt.Errorf("failed to get list of tenants: %w", err)
		}

		if tenantsResp.Response.StatusCode != http.StatusOK {
			return errors.New("failed to get list of tenants")
		}

		if ok = ctxUtils.SetTenantListForContext(cmd, tenantsResp.TenantList.Tenants); !ok {
			return errors.New("failed to set tenant list in context")
		}

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Get the tenant list from context
		tenants, ok := ctxUtils.GetUserTenantsFromContext(cmd)
		if !ok {
			return errors.New("failed to get tenant list from context")
		}

		currentSession, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
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

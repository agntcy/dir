// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package orgs provides the CLI command for listing organizations (tenants) for the logged-in user.
// The orgs command only lists organizations/tenants. To switch between organizations, use the "orgs switch" subcommand.
package orgs

import (
	"errors"
	"fmt"
	"io"

	auth "github.com/agntcy/dir/hub/auth"
	"github.com/agntcy/dir/hub/client/idp"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/cmd/orgswitch"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

const (
	selectionMark = "*"
	gapSize       = 4
)

// NewCommand creates the "orgs" command for the Agent Hub CLI.
// It lists organizations (tenants) for the logged-in user. To switch organizations, use the "orgs switch" subcommand.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "orgs",
		Aliases: []string{"tenants"},
		Short:   "List organizations for logged in user",
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || currentSession == nil {
			return errors.New("no current session found. please login first")
		}

		orgs, err := auth.FetchUserTenants(cmd.Context(), currentSession)
		if err != nil {
			return fmt.Errorf("failed to get orgs list: %w", err)
		}

		renderList(cmd.OutOrStdout(), orgs, currentSession.CurrentTenant)

		return nil
	}

	cmd.AddCommand(
		orgswitch.NewCommand(hubOpts),
	)

	return cmd
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

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package orgs provides the CLI commands for managing organizations.
// The orgs command has subcommands for listing and creating organizations.
package orgs

import (
	"errors"
	"fmt"
	"io"
	"regexp"

	saasv1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	auth "github.com/agntcy/dir/hub/auth"
	authUtils "github.com/agntcy/dir/hub/auth/utils"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

const (
	nameHeader = "Organization Name"
	idHeader   = "Organization ID"
	roleHeader = "Role"
	gapSize    = 4
)

// isOrganizationNameValid validates organization name against the API format and rejects UUID format.
func isOrganizationNameValid(name string) bool {
	if name == "" {
		return true
	}

	uuidRegex := regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	apiRegex := regexp.MustCompile(`^[a-z0-9_-]+(/[a-z0-9_-]+)*$`)

	return !uuidRegex.MatchString(name) && apiRegex.MatchString(name)
}

// NewCommand creates the "orgs" command for the Agent Hub CLI.
// It provides subcommands for managing organizations.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "orgs",
		Aliases: []string{"organizations"},
		Short:   "Manage organizations",
		Long:    "Manage organizations including listing and creating new organizations",
	}

	cmd.AddCommand(newListCommand(hubOpts))
	cmd.AddCommand(newCreateCommand(hubOpts))

	return cmd
}

// newListCommand creates the "orgs list" subcommand.
func newListCommand(_ *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List organizations for logged in user",
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || !auth.HasLoginCreds(currentSession) {
			return errors.New("no current session found. please login first")
		}

		hc, err := hubClient.New(currentSession.HubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		ctx := authUtils.AddAuthToContext(cmd.Context(), currentSession)

		orgs, err := hc.ListOrganizations(ctx, &saasv1alpha1.ListOrganizationsRequest{})
		if err != nil {
			return fmt.Errorf("failed to get orgs list: %w", err)
		}

		renderList(cmd.OutOrStdout(), orgs.GetOrganizations())

		return nil
	}

	return cmd
}

// newCreateCommand creates the "orgs create" subcommand.
func newCreateCommand(_ *hubOptions.HubOptions) *cobra.Command {
	var (
		orgName        string
		orgDescription string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new organization",
		Long: `Create a new organization with the specified name and optional description.

Organization name must include only lowercase letters, digits, underscores or hyphens.

Valid examples:
  - my-org
  - my_org  
  - test123
  - org_name-123

Examples:
  dirctl hub orgs create --name my-organization --description "My test organization"
  dirctl hub orgs create --name my_org`,
	}

	cmd.Flags().StringVarP(&orgName, "name", "n", "", "Organization name (required)")
	cmd.Flags().StringVarP(&orgDescription, "description", "d", "", "Organization description (optional)")

	if err := cmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "failed to mark name flag as required: %s", err.Error())

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || !auth.HasLoginCreds(currentSession) {
			return errors.New("no current session found. please login first")
		}

		hc, err := hubClient.New(currentSession.HubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		ctx := authUtils.AddAuthToContext(cmd.Context(), currentSession)

		if !isOrganizationNameValid(orgName) {
			return errors.New("invalid organization name format. 'name' must include only lowercase letters, digits, underscores or hyphens. Examples: my-org, my_org")
		}

		req := &saasv1alpha1.CreateOrganizationRequest{
			Organization: &saasv1alpha1.Organization{
				Name:        orgName,
				Description: orgDescription,
			},
		}

		org, err := hc.CreateOrganization(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to create organization: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Organization created successfully:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "ID:          %s\n", org.GetId())
		fmt.Fprintf(cmd.OutOrStdout(), "Name:        %s\n", org.GetName())
		fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", org.GetDescription())

		return nil
	}

	return cmd
}

type renderFn func(int, int, int) string

func renderList(stream io.Writer, organizationsWithRoles []*saasv1alpha1.OrganizationWithRole) {
	renderFns := make([]renderFn, len(organizationsWithRoles))

	longestNameLen := len(nameHeader)
	longestRoleLen := len(roleHeader)
	longestIDLen := len(idHeader)

	for i, org := range organizationsWithRoles {
		if len(org.GetOrganization().GetName()) > longestNameLen {
			longestNameLen = len(org.GetOrganization().GetName())
		}

		if len(org.GetOrganization().GetId()) > longestIDLen {
			longestIDLen = len(org.GetOrganization().GetId())
		}

		if len(org.GetRole().String()) > longestRoleLen {
			longestRoleLen = len(org.GetRole().String())
		}

		renderFns[i] = func(lName, lId, lRole int) string {
			nameCol := text.AlignLeft.Apply(org.GetOrganization().GetName(), lName+gapSize)
			idCol := text.AlignLeft.Apply(org.GetOrganization().GetId(), lId+gapSize)
			roleCol := text.AlignLeft.Apply(org.GetRole().String(), lRole)

			return fmt.Sprintf("%s%s%s", nameCol, idCol, roleCol)
		}
	}

	nameHeader := text.AlignLeft.Apply(nameHeader, longestNameLen+gapSize)
	idHeader := text.AlignLeft.Apply(idHeader, longestIDLen+gapSize)
	roleHeader := text.AlignLeft.Apply(roleHeader, longestRoleLen)
	fmt.Fprintf(stream, "%s%s%s\n", nameHeader, idHeader, roleHeader) //nolint:errcheck

	for _, organization := range renderFns {
		fmt.Fprintln(stream, organization(longestNameLen, longestIDLen, longestRoleLen)) //nolint:errcheck
	}
}

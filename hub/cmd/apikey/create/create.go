// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	v1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	"github.com/agntcy/dir/hub/auth"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/cmd/apikey/options"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	service "github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	"github.com/spf13/cobra"
)

// NewCommand creates the "create" command for the Agent apikey Hub CLI.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create --role <role> {--org-id <org_id> | --org-name <org_name>}",
		Short: "create a new API key for hub actions in an organization, with specified role",
		Long: `create a new API key for hub actions in an organization, with specified role.

Parameters:
  --role      <role>      Role name. One of ['ROLE_ORG_ADMIN', 'ROLE_ADMIN', 'ROLE_EDITOR', 'ROLE_VIEWER']
  --org-id    <org_id>    Organization ID
  --org-name  <org_name>  Organization Name

Example:
  # Create a new API key with role OrgAdmin for organization "MyOrg"
  dirctl hub apikey create --role ROLE_ORG_ADMIN --org-name MyOrg

  # Create a new API key with role OrgAdmin for organization with ID 935a67e3-0276-4f61-b1ff-000fb163eedd
  dirctl hub apikey create --role ROLE_ORG_ADMIN --org-id "935a67e3-0276-4f61-b1ff-000fb163eedd"`,
	}

	opts := options.NewApiKeyCreateOptions(hubOpts, cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCommand(cmd, args, opts)
	}

	return cmd
}

func runCommand(cmd *cobra.Command, args []string, opts *options.ApiKeyCreateOptions) error {
	if opts.OrganizationId == "" && opts.OrganizationName == "" {
		return errors.New("organization ID or name is required")
	} else if opts.OrganizationId != "" && opts.OrganizationName != "" {
		return errors.New("only one of organization ID or name should be provided")
	}

	var organization any
	if opts.OrganizationId != "" {
		organization = &v1alpha1.CreateApiKeyRequest_OrganizationId{
			OrganizationId: opts.OrganizationId,
		}
	} else if opts.OrganizationName != "" {
		organization = &v1alpha1.CreateApiKeyRequest_OrganizationName{
			OrganizationName: opts.OrganizationName,
		}
	}

	// Retrieve session from context
	ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
	currentSession, ok := ctxSession.(*sessionstore.HubSession)
	if !ok || currentSession == nil {
		return errors.New("could not get current hub session")
	}
	if !auth.HasLoginCreds(currentSession) && auth.HasApiKey(currentSession) {
		fmt.Println("User is authenticated with API key, using it to get credentials...")
		if err := auth.RefreshApiKeyAccessToken(cmd.Context(), currentSession, opts.ServerAddress); err != nil {
			return fmt.Errorf("failed to refresh API key access token: %w", err)
		}
	}

	if !auth.HasLoginCreds(currentSession) && !auth.HasApiKey(currentSession) {
		return errors.New("you need to be logged in to push to the hub\nuse `dirctl hub login` command to login")
	}

	hc, err := hubClient.New(currentSession.HubBackendAddress)
	if err != nil {
		return fmt.Errorf("failed to create hub client: %w", err)
	}

	resp, err := service.CreateAPIKey(cmd.Context(), hc, opts.Role, organization, currentSession)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	fmt.Printf("API Key created successfully:\n")
	prettyModel, err := json.MarshalIndent(resp.Token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}
	fmt.Fprintf(os.Stdout, "%s\n", string(prettyModel))

	currentSession.ApiKeyAccess = &sessionstore.ApiKey{
		ClientID: resp.Token.ClientId,
		Secret:   resp.Token.Secret,
	}

	// Save session with new api key
	sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())
	if err := sessionStore.SaveHubSession(opts.ServerAddress, currentSession); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	fmt.Printf("The API Key has been added to your session file.\n")
	return nil
}

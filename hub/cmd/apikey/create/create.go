// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package create

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	v1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/cmd/apikey/options"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	service "github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
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
  --json                  Output in JSON format (default: env file format)

Output:
  By default, the command outputs the API key in environment variable format:
  DIRCTL_CLIENT_ID=<client_id>
  DIRCTL_CLIENT_SECRET=<secret>

  With --json flag, it outputs in JSON format:
  {
    "client_id": "<client_id>",
    "secret": "<secret>"
  }

Example:
  # Create a new API key with role OrgAdmin for organization "MyOrg"
  dirctl hub apikey create --role ROLE_ORG_ADMIN --org-name MyOrg

  # Create a new API key with role OrgAdmin for organization with ID 935a67e3-0276-4f61-b1ff-000fb163eedd
  dirctl hub apikey create --role ROLE_ORG_ADMIN --org-id "935a67e3-0276-4f61-b1ff-000fb163eedd"

  # Create a new API key and output in JSON format
  dirctl hub apikey create --role ROLE_ORG_ADMIN --org-name MyOrg --json`,
	}

	opts := options.NewAPIKeyCreateOptions(hubOpts, cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCommand(cmd, args, opts)
	}

	return cmd
}

func runCommand(cmd *cobra.Command, _ []string, opts *options.APIKeyCreateOptions) error { //nolint:cyclop
	if opts.OrganizationID == "" && opts.OrganizationName == "" {
		return errors.New("organization ID or name is required")
	} else if opts.OrganizationID != "" && opts.OrganizationName != "" {
		return errors.New("only one of organization ID or name should be provided")
	}

	var organization any
	if opts.OrganizationID != "" {
		organization = &v1alpha1.CreateApiKeyRequest_OrganizationId{
			OrganizationId: opts.OrganizationID,
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

	// Check for credentials
	if err := authUtils.CheckForCreds(cmd, currentSession, opts.ServerAddress, opts.JSONOutput); err != nil {
		// this error need to be return without modification in order to be displayed
		return err //nolint:wrapcheck
	}

	hc, err := hubClient.New(currentSession.HubBackendAddress)
	if err != nil {
		return fmt.Errorf("failed to create hub client: %w", err)
	}

	apikeyWithSecret, err := service.CreateAPIKey(cmd.Context(), hc, opts.Role, organization, currentSession)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	// Base64 encode the secret after creation.
	encodedSecret := base64.StdEncoding.EncodeToString([]byte(apikeyWithSecret.Secret))

	// Create API key structure with encoded secret for output
	apikey := &service.APIKeyWithSecret{
		ClientID: apikeyWithSecret.ClientID,
		Secret:   encodedSecret,
	}

	if opts.JSONOutput {
		// Output JSON format for API key details
		prettyModel, err := json.MarshalIndent(apikey, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal API key: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(prettyModel))
	} else {
		// Output in environment file format
		fmt.Fprintf(cmd.OutOrStdout(), "DIRCTL_CLIENT_ID=%s\n", apikey.ClientID)
		fmt.Fprintf(cmd.OutOrStdout(), "DIRCTL_CLIENT_SECRET=%s\n\n", apikey.Secret)
	}

	return nil
}

package logout

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

var (
	ErrSecretNotFoundForAddress = fmt.Errorf("No secret found for the given address. Please login first.")
)

func NewLogoutCommand(opts *options.HubOptions) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from the Phoenix SaaS hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get current hub secret from context
			secret, ok := ctxUtils.GetCurrentHubSecretFromContext(cmd.Context())
			if !ok {
				return ErrSecretNotFoundForAddress
			}

			// Get secret store from context
			secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get secret store from context")
			}

			idpClient := idp.NewIdpClient(secret.IdpAddress)
			resp, err := idpClient.Logout(&idp.LogoutRequest{IdToken: secret.IdToken})
			if err != nil {
				return fmt.Errorf("failed to logout: %w", err)
			}
			if resp.Response.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to logout: unexpected status code: %d: %s", resp.Response.StatusCode, resp.Body)
			}

			// Remove secret from secret store
			if err = secretStore.RemoveHubSecret(opts.ServerAddress); err != nil {
				return fmt.Errorf("failed to remove secret from secret store: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged out")
			return nil
		},
	}

	return cmd
}

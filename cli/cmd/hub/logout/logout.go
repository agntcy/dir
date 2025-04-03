package logout

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/hub/idp"
	secretstore2 "github.com/agntcy/dir/cli/hub/secretstore"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

var (
	ErrSecretNotFoundForAddress = fmt.Errorf("No active session found for the address. Please login first.")
)

func NewCommand(opts *options.HubOptions) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Agent Hub",
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

			idpClient, ok := ctxUtils.GetIdpClientFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get idp client from context")
			}

			return runCmd(cmd.OutOrStdout(), opts, secret, secretStore, idpClient)
		},
		TraverseChildren: true,
	}

	return cmd
}

func runCmd(outStream io.Writer, opts *options.HubOptions, secret *secretstore2.HubSecret, secretStore secretstore2.SecretStore, idpClient idp.Client) error {
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

	fmt.Fprintln(outStream, "Successfully logged out.")
	return nil
}

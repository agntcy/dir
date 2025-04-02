package pull

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"

	"github.com/agntcy/dir/cli/config"
	hubClient "github.com/agntcy/dir/cli/hub/client"
	"github.com/agntcy/dir/cli/secretstore"
	contextUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/token"
	"github.com/agntcy/hub/api/v1alpha1"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull {<digest> | <repository>:<version> }",
		Short: "Pull an agent from Agent Hub",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Backend address should be fetched from the context
			secret, ok := contextUtils.GetCurrentHubSecretFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("could not get current hub secret from context")
			}

			secretStore, ok := contextUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get secret store from context")
			}

			idpClient, ok := contextUtils.GetIdpClientFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get IDP client from context")
			}

			serverAddr, ok := contextUtils.GetCurrentServerAddressFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get current server address")
			}

			err := token.RefreshTokenIfExpired(
				cmd,
				serverAddr,
				secret,
				secretStore,
				idpClient,
			)
			if err != nil {
				return fmt.Errorf("failed to refresh expired access token: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("agent id is the only required argument")
			}

			hc, err := hubClient.New(config.DefaultHubBackendAddress)
			if err != nil {
				return fmt.Errorf("failed to create hub client: %w", err)
			}

			agentId, err := parseAgentId(args[0])
			if err != nil {
				return fmt.Errorf("invalid agent id: %w", err)
			}

			secret, ok := contextUtils.GetCurrentHubSecretFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("could not get current hub secret from context")
			}

			return runCmd(cmd.Context(), hc, agentId, secret)
		},
	}

	return cmd
}

func runCmd(ctx context.Context, hc hubClient.Client, agentId *v1alpha1.AgentIdentifier, secret *secretstore.HubSecret) error {
	if secret.TokenSecret != nil && secret.AccessToken != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", secret.TokenSecret.AccessToken)))
	}

	model, err := hc.PullAgent(ctx, &v1alpha1.PullAgentRequest{
		Id: agentId,
	})
	if err != nil {
		return fmt.Errorf("failed to pull agent: %w", err)
	}

	var modelObj map[string]interface{}
	if err = json.Unmarshal(model, &modelObj); err != nil {
		return fmt.Errorf("failed to unmarshal agent: %w", err)
	}
	prettyModel, err := json.MarshalIndent(modelObj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	fmt.Fprintf(os.Stdout, "%s\n", string(prettyModel))

	return nil
}

func parseAgentId(agentId string) (*v1alpha1.AgentIdentifier, error) {
	parts := strings.Split(agentId, ":")
	if len(parts) > 2 {
		return nil, fmt.Errorf("agent id should be in the format <repository>:<version> or <digest>")
	}

	if len(parts) == 1 {
		return &v1alpha1.AgentIdentifier{
			Id: &v1alpha1.AgentIdentifier_Digest{
				Digest: parts[0],
			},
		}, nil
	}

	return &v1alpha1.AgentIdentifier{
		Id: &v1alpha1.AgentIdentifier_RepoVersionId{
			RepoVersionId: &v1alpha1.RepoVersionId{
				RepositoryName: parts[0],
				Version:        parts[1],
			},
		},
	}, nil
}

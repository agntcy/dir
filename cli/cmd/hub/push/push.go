package push

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"

	"github.com/agntcy/dir/cli/config"
	hubClient "github.com/agntcy/dir/cli/hub/client"
	"github.com/agntcy/dir/cli/options"
	"github.com/agntcy/dir/cli/util/agent"
	"github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/token"
	"github.com/agntcy/hub/api/v1alpha1"
)

func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push {<repository>:<version> | <repository_id>:<version>} {<model.json> | --stdin} ",
		Short: "Push model to Agent Hub",
	}

	opts := options.NewHubPushOptions(hubOpts, options.NewPushOptions(hubOpts.BaseOption, cmd))

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Check if the user is logged in
		secret, ok := context.GetCurrentHubSecretFromContext(cmd.Context())
		if !ok || secret.TokenSecret == nil {
			return fmt.Errorf("You need to be logged in to push to the hub.\nUse `dirctl hub login` command to login.")
		}

		// Check if the access token is expired
		idpClient, ok := context.GetIdpClientFromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("failed to get IDP client from context")
		}

		secretStore, ok := context.GetSecretStoreFromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("failed to get secret store from context")
		}

		serverAddr, ok := context.GetCurrentServerAddressFromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("failed to get current server address")
		}

		if err := token.RefreshTokenIfExpired(
			cmd,
			serverAddr,
			secret,
			secretStore,
			idpClient,
		); err != nil {
			return fmt.Errorf("failed to refresh expired access token: %w", err)
		}

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		secret, ok := context.GetCurrentHubSecretFromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("You need to be logged in to push to the hub.\nUse `dirctl hub login` command to login.")
		}

		hc, err := hubClient.New(config.DefaultHubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		if len(args) > 2 {
			return errors.New("The following arguments could be given: <repository>:<version> [model.json]")
		}

		fpath := ""
		if len(args) == 2 {
			fpath = args[1]
		}

		reader, err := agent.GetReader(fpath, opts.FromStdIn)
		if err != nil {
			return err
		}

		agentBytes, err := agent.GetAgentBytes(reader)
		if err != nil {
			return err
		}

		repoId, tag, err := parseRepoTagId(args[0])
		if err != nil {
			return err
		}

		ctx := metadata.NewOutgoingContext(cmd.Context(), metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", secret.AccessToken)))
		resp, err := hc.PushAgent(ctx, agentBytes, repoId, tag)
		if err != nil {
			return fmt.Errorf("failed to push agent: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), resp.Id.Digest)

		return nil
	}

	return cmd
}

func parseRepoTagId(id string) (any, string, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return nil, "", errors.New("invalid agent id format")
	}
	tag := parts[1]
	repoId := parts[0]

	if _, err := uuid.Parse(repoId); err == nil {
		return &v1alpha1.PushAgentRequest_RepositoryId{RepositoryId: repoId}, tag, nil
	} else {
		return &v1alpha1.PushAgentRequest_RepositoryName{RepositoryName: repoId}, tag, nil
	}
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/agntcy/dir/api/hub/v1alpha1"
	hubClient "github.com/agntcy/dir/cli/hub/client"
	"github.com/agntcy/dir/cli/hub/secretstore"
	contextUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/token"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull {<digest> | <repository>:<version> }",
		Short: "Pull an agent from Agent Hub",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			// TODO: Backend address should be fetched from the context
			secret, ok := contextUtils.GetCurrentHubSecretFromContext(cmd.Context())
			if !ok {
				return errors.New("could not get current hub secret from context")
			}

			secretStore, ok := contextUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return errors.New("failed to get secret store from context")
			}

			idpClient, ok := contextUtils.GetIdpClientFromContext(cmd.Context())
			if !ok {
				return errors.New("failed to get IDP client from context")
			}

			serverAddr, ok := contextUtils.GetCurrentServerAddressFromContext(cmd.Context())
			if !ok {
				return errors.New("failed to get current server address")
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
				return errors.New("agent id is the only required argument")
			}

			secret, ok := contextUtils.GetCurrentHubSecretFromContext(cmd.Context())
			if !ok {
				return errors.New("could not get current hub secret from context")
			}

			hc, err := hubClient.New(secret.HubBackendAddress)
			if err != nil {
				return fmt.Errorf("failed to create hub client: %w", err)
			}

			agentID := parseAgentID(args[0])
			if err != nil {
				return fmt.Errorf("invalid agent id: %w", err)
			}

			return runCmd(cmd.Context(), hc, agentID, secret)
		},
	}

	return cmd
}

func runCmd(ctx context.Context, hc hubClient.Client, agentID *v1alpha1.AgentIdentifier, secret *secretstore.HubSecret) error {
	if secret.TokenSecret != nil && secret.AccessToken != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+secret.TokenSecret.AccessToken))
	}

	model, err := hc.PullAgent(ctx, &v1alpha1.PullAgentRequest{
		Id: agentID,
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

func parseAgentID(agentID string) *v1alpha1.AgentIdentifier {
	// TODO: support parsing <repository>:<tag> format
	// Digest is also in the format of <algorithm>:<hash>
	return &v1alpha1.AgentIdentifier{
		Id: &v1alpha1.AgentIdentifier_Digest{
			Digest: agentID,
		},
	}
}

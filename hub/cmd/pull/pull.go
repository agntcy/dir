// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/agntcy/dir/hub/api/v1alpha1"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	ctxUtils "github.com/agntcy/dir/hub/utils/context"
	"github.com/agntcy/dir/hub/utils/token"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	opts := hubOptions.NewHubPullOptions(hubOpts)

	cmd := &cobra.Command{
		Use:   "pull <digest>",
		Short: "Pull an agent from Agent Hub",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := token.RefreshContextTokenIfExpired(cmd, opts.HubOptions); err != nil {
				return fmt.Errorf("failed to refresh expired access token: %w", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("agent id is the only required argument")
			}

			session, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
			if !ok {
				return errors.New("could not get current hub session from context")
			}

			hc, err := hubClient.New(session.HubBackendAddress)
			if err != nil {
				return fmt.Errorf("failed to create hub client: %w", err)
			}

			agentID := parseAgentID(args[0])
			if err != nil {
				return fmt.Errorf("invalid agent id: %w", err)
			}

			return runCmd(cmd.Context(), hc, agentID, session)
		},
	}

	return cmd
}

func runCmd(ctx context.Context, hc hubClient.Client, agentID *v1alpha1.AgentIdentifier, session *sessionstore.HubSession) error {
	if session != nil && session.Tokens != nil && session.CurrentTenant != "" {
		if t, ok := session.Tokens[session.CurrentTenant]; ok && t != nil && t.AccessToken != "" {
			ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+session.Tokens[session.CurrentTenant].AccessToken))
		}
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

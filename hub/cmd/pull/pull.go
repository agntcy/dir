// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

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
		Use:   "pull <agent_ref>",
		Short: "Pull an agent from Agent Hub",
		Long: `Pull an agent from the Agent Hub.

Parameters:
  <agent_ref>    Agent reference in one of the following formats:
                - sha256:<hash>    : Pull by digest
                - <repo>:<version> : Pull by repository and version

Examples:
  # Pull agent by digest
  dirctl hub pull sha256:1234567890abcdef...

  # Pull agent by repository and version
  dirctl hub pull owner/repo-name:v1.0.0`,
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

			agentID, err := parseAgentID(args[0])
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

func parseAgentID(agentID string) (*v1alpha1.AgentIdentifier, error) {
	// If the agentID starts with "sha256", treat it as a digest
	if strings.HasPrefix(agentID, "sha256:") {
		return &v1alpha1.AgentIdentifier{
			Id: &v1alpha1.AgentIdentifier_Digest{
				Digest: agentID,
			},
		}, nil
	}

	// Try to split by ":" for repository:version format
	parts := strings.Split(agentID, ":")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return &v1alpha1.AgentIdentifier{
			Id: &v1alpha1.AgentIdentifier_RepoVersionId{
				RepoVersionId: &v1alpha1.RepoVersionId{
					RepositoryName: parts[0],
					Version:        parts[1],
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("invalid agent ID format: %s. Expected format is either 'sha256:<hash>' or '<repository>:<version>'", agentID)
}

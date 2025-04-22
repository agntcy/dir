// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"context"
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/util/agent"
	"github.com/agntcy/dir/hub/api/v1alpha1"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	ctxUtils "github.com/agntcy/dir/hub/utils/context"
	"github.com/agntcy/dir/hub/utils/token"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push {<repository> | <repository_id>} {<model.json> | --stdin} ",
		Short: "Push model to Agent Hub",
	}

	opts := hubOptions.NewHubPushOptions(hubOpts, cmd)

	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := token.ValidateAccessTokenFromContext(cmd); err != nil {
			return fmt.Errorf("failed to validate access token: %w", err)
		}

		if err := token.RefreshContextTokenIfExpired(cmd, opts.HubOptions); err != nil {
			return fmt.Errorf("failed to refresh expired access token: %w", err)
		}

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		session, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
		if !ok {
			return errors.New("you need to be logged in to push to the hub\nuse `dirctl hub login` command to login")
		}

		hc, err := hubClient.New(session.HubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		if len(args) > 2 { //nolint:mnd
			return errors.New("the following arguments could be given: <repository>:<version> [model.json]")
		}

		fpath := ""
		if len(args) == 2 { //nolint:mnd
			fpath = args[1]
		}

		reader, err := agent.GetReader(fpath, opts.FromStdIn)
		if err != nil {
			return fmt.Errorf("failed to get reader: %w", err)
		}

		agentBytes, err := agent.GetAgentBytes(reader)
		if err != nil {
			return fmt.Errorf("failed to get agent bytes: %w", err)
		}

		// TODO: Push based on repoName and version misleading
		repoID := parseRepoTagID(args[0])

		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+session.Tokens[session.CurrentTenant].AccessToken))

		resp, err := hc.PushAgent(ctx, agentBytes, repoID)
		if err != nil {
			return fmt.Errorf("failed to push agent: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), resp.GetId().GetDigest())

		return nil
	}

	return cmd
}

func parseRepoTagID(id string) any {
	if _, err := uuid.Parse(id); err == nil {
		return &v1alpha1.PushAgentRequest_RepositoryId{RepositoryId: id}
	}

	return &v1alpha1.PushAgentRequest_RepositoryName{RepositoryName: id}
}

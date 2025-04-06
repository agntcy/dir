// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/api/hub/v1alpha1"
	hubClient "github.com/agntcy/dir/cli/hub/client"
	"github.com/agntcy/dir/cli/hub/token"
	"github.com/agntcy/dir/cli/options"
	"github.com/agntcy/dir/cli/util/agent"
	contextUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push {<repository>:<version> | <repository_id>:<version>} {<model.json> | --stdin} ",
		Short: "Push model to Agent Hub",
	}

	opts := options.NewHubPushOptions(hubOpts, options.NewPushOptions(hubOpts.BaseOption, cmd))

	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		// Check if the user is logged in
		secret, ok := contextUtils.GetCurrentHubSessionFromContext(cmd.Context())
		if !ok || secret.Tokens == nil {
			return errors.New("you need to be logged in to push to the hub\nuse `dirctl hub login` command to login")
		}

		// Check if the access token is expired
		idpClient, ok := contextUtils.GetOktaClientFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get IDP client from context")
		}

		secretStore, ok := contextUtils.GetSessionStoreFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get secret store from context")
		}

		if err := token.RefreshTokenIfExpired(
			cmd,
			opts.ServerAddress,
			secret,
			secretStore,
			idpClient,
		); err != nil {
			return fmt.Errorf("failed to refresh expired access token: %w", err)
		}

		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		session, ok := contextUtils.GetCurrentHubSessionFromContext(cmd.Context())
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
		repoID, _, err := parseRepoTagID(args[0])
		if err != nil {
			return fmt.Errorf("failed to parse repo id: %w", err)
		}

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

func parseRepoTagID(id string) (any, string, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 { //nolint:mnd
		return nil, "", errors.New("invalid agent id format")
	}

	tag := parts[1]
	repoID := parts[0]

	if _, err := uuid.Parse(repoID); err == nil {
		return &v1alpha1.PushAgentRequest_RepositoryId{RepositoryId: repoID}, tag, nil
	}

	return &v1alpha1.PushAgentRequest_RepositoryName{RepositoryName: repoID}, tag, nil
}

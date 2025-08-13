// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package service provides reusable business logic for agent operations in the Agent Hub CLI and related applications.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agntcy/dir/hub/api/v1alpha1"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// addAuthToContext adds the authorization header to the context if an access token is available.
func addAuthToContext(ctx context.Context, session *sessionstore.HubSession) context.Context {
	if session != nil && session.Tokens != nil && session.CurrentTenant != "" {
		if t, ok := session.Tokens[session.CurrentTenant]; ok && t != nil && t.AccessToken != "" {
			return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+t.AccessToken))
		}
	}

	return ctx
}

// PullAgent pulls an agent from the hub and returns the pretty-printed JSON.
// It uses the provided session for authentication.
func PullAgent(
	ctx context.Context,
	hc hubClient.Client,
	agentID *v1alpha1.RecordIdentifier,
	session *sessionstore.HubSession,
) ([]byte, error) {
	ctx = addAuthToContext(ctx, session)

	model, err := hc.PullAgent(ctx, &v1alpha1.PullRecordRequest{
		Id: agentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull agent: %w", err)
	}

	var modelObj map[string]interface{}
	if err = json.Unmarshal(model, &modelObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	prettyModel, err := json.MarshalIndent(modelObj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent: %w", err)
	}

	return prettyModel, nil
}

// ParseAgentID parses a string into an AgentIdentifier.
// Accepts either a digest (sha256:<hash>) or repository:version format.
func ParseAgentID(agentID string) (*v1alpha1.RecordIdentifier, error) {
	// If the agentID starts with "sha256", treat it as a digest
	if strings.HasPrefix(agentID, "sha256:") {
		return &v1alpha1.RecordIdentifier{
			Id: &v1alpha1.RecordIdentifier_Digest{
				Digest: agentID,
			},
		}, nil
	}

	// Try to split by ":" for repository:version format
	parts := strings.Split(agentID, ":")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return &v1alpha1.RecordIdentifier{
			Id: &v1alpha1.RecordIdentifier_RepoVersionId{
				RepoVersionId: &v1alpha1.RepoVersionId{
					RepositoryName: parts[0],
					Version:        parts[1],
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("invalid agent ID format: %s. Expected format is either 'sha256:<hash>' or '<repository>:<version>'", agentID)
}

// ParseRepoTagID parses a repository tag or ID string into the appropriate PushAgentRequest field.
// Returns a RepositoryId if the string is a UUID, otherwise a RepositoryName.
func ParseRepoTagID(id string) any {
	if _, err := uuid.Parse(id); err == nil {
		return &v1alpha1.PushRecordRequest_RepositoryId{RepositoryId: id}
	}

	return &v1alpha1.PushRecordRequest_RepositoryName{RepositoryName: id}
}

// PushAgent pushes an agent to the hub and returns the response.
// It uses the provided session for authentication.
func PushAgent(
	ctx context.Context,
	hc hubClient.Client,
	agentBytes []byte,
	repoID any,
	session *sessionstore.HubSession,
) (*v1alpha1.PushRecordResponse, error) {
	ctx = addAuthToContext(ctx, session)

	resp, err := hc.PushAgent(ctx, agentBytes, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to push agent: %w", err)
	}

	return resp, nil
}

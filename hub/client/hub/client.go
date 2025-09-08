// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package hub provides a client for interacting with the Agent Hub backend API, including agent management and related operations.
package hub

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	corev1alpha1 "github.com/agntcy/dir/api/core/v1alpha1"
	v1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	"github.com/opencontainers/go-digest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const chunkSize = 4096 // 4KB

// Client defines the interface for interacting with the Agent Hub backend for agent operations.
type Client interface {
	// PushAgent uploads an agent to the hub and returns the response or an error.
	PushAgent(ctx context.Context, agent []byte, repository any) (*v1alpha1.PushRecordResponse, error)
	// PullAgent downloads an agent from the hub and returns the agent data or an error.
	PullAgent(ctx context.Context, request *v1alpha1.PullRecordRequest) ([]byte, error)
	// CreateAPIKey creates an API key for the specified role and returns the (clientId, secret) or an error.
	CreateAPIKey(ctx context.Context, roleName string, organization any) (*v1alpha1.CreateApiKeyResponse, error)
	// DeleteAPIKey deletes an API key from the hub and returns the response or an error.
	DeleteAPIKey(ctx context.Context, clientID string) (*v1alpha1.DeleteApiKeyResponse, error)
	// ListAPIKeys lists all API keys for a specific organization and returns the response or an error.
	ListAPIKeys(ctx context.Context, organization any) (*v1alpha1.ListApiKeyResponse, error)
}

// client implements the Client interface for the Agent Hub backend.
type client struct {
	v1alpha1.AgentDirServiceClient
	v1alpha1.ApiKeyServiceClient
	v1alpha1.OrganizationServiceClient
	v1alpha1.UserServiceClient
}

// New creates a new Agent Hub client for the given server address.
// Returns the client or an error if the connection could not be established.
func New(serverAddr string) (*client, error) { //nolint:revive
	// Create connection
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	return &client{
		AgentDirServiceClient:     v1alpha1.NewAgentDirServiceClient(conn),
		ApiKeyServiceClient:       v1alpha1.NewApiKeyServiceClient(conn),
		OrganizationServiceClient: v1alpha1.NewOrganizationServiceClient(conn),
		UserServiceClient:         v1alpha1.NewUserServiceClient(conn),
	}, nil
}

// PushAgent uploads an agent to the hub in chunks and returns the response or an error.
func (c *client) PushAgent(ctx context.Context, agent []byte, repository any) (*v1alpha1.PushRecordResponse, error) { //nolint:cyclop
	var parsedAgent *corev1alpha1.Agent
	if err := json.Unmarshal(agent, &parsedAgent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	d := digest.FromBytes(agent).String()
	t := corev1alpha1.ObjectType_OBJECT_TYPE_AGENT.String()

	ref := &corev1alpha1.ObjectRef{
		Digest:      d,
		Type:        t,
		Size:        uint64(len(agent)),
		Annotations: parsedAgent.GetAnnotations(),
	}

	stream, err := c.AgentDirServiceClient.PushRecord(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create push stream: %w", err)
	}

	buf := make([]byte, chunkSize)
	agentReader := bytes.NewReader(agent)

	for {
		var n int

		n, err = agentReader.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}

		if n == 0 {
			break
		}

		msg := &v1alpha1.PushRecordRequest{
			Model: &corev1alpha1.Object{
				Data: buf[:n],
				Ref:  ref,
			},
		}

		switch parsedRepo := repository.(type) {
		case *v1alpha1.PushRecordRequest_RepositoryName:
			msg.Repository = parsedRepo

			if parsedRepo.RepositoryName != parsedAgent.GetName() {
				return nil, fmt.Errorf("repository name mismatch: expected %s, got %s", parsedAgent.GetName(), parsedRepo.RepositoryName)
			}

			msg.OrganizationName, err = getOrganizationNameFromRepository(parsedRepo.RepositoryName)
			if err != nil {
				return nil, fmt.Errorf("failed to parse organization name from repository: %w", err)
			}
		case *v1alpha1.PushRecordRequest_RepositoryId:
			msg.Repository = parsedRepo
			// In this case, we read the organization name from the agent
			msg.OrganizationName, err = getOrganizationNameFromRepository(parsedAgent.GetName())
			if err != nil {
				return nil, fmt.Errorf("failed to parse organization name from agent: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown repository type: %T", repository)
		}

		if err = stream.Send(msg); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("could not send object: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("could not receive response: %w", err)
	}

	return resp, nil
}

// PullAgent downloads an agent from the hub in chunks and returns the agent data or an error.
func (c *client) PullAgent(ctx context.Context, request *v1alpha1.PullRecordRequest) ([]byte, error) {
	stream, err := c.AgentDirServiceClient.PullRecord(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	var buffer bytes.Buffer

	for {
		var chunk *v1alpha1.PullRecordResponse

		chunk, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to receive chunk: %w", err)
		}

		buffer.Write(chunk.GetModel().GetData())
	}

	return buffer.Bytes(), nil
}

func (c *client) CreateAPIKey(ctx context.Context, roleName string, organization any) (*v1alpha1.CreateApiKeyResponse, error) {
	roleValue, ok := v1alpha1.Role_value[roleName]
	if !ok {
		return nil, fmt.Errorf("unknown role: %d", roleValue)
	}

	req := &v1alpha1.CreateApiKeyRequest{
		Role: v1alpha1.Role(roleValue),
	}

	switch parsedOrg := organization.(type) {
	case *v1alpha1.CreateApiKeyRequest_OrganizationName:
		req.Organization = parsedOrg
	case *v1alpha1.CreateApiKeyRequest_OrganizationId:
		req.Organization = parsedOrg
	default:
		return nil, fmt.Errorf("unknown organization type: %T", organization)
	}

	stream, err := c.ApiKeyServiceClient.CreateAPIKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	var chunk *v1alpha1.CreateApiKeyResponse

	chunk, err = stream.Recv()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to receive chunk: %w", err)
	}

	return chunk, nil
}

func (c *client) DeleteAPIKey(ctx context.Context, clientID string) (*v1alpha1.DeleteApiKeyResponse, error) {
	req := &v1alpha1.DeleteApiKeyRequest{
		ClientId: clientID,
	}

	resp, err := c.ApiKeyServiceClient.DeleteAPIKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete API key: %w", err)
	}

	if resp == nil {
		return nil, errors.New("received nil response from delete api key")
	}

	return resp, nil
}

func (c *client) ListAPIKeys(ctx context.Context, organization any) (*v1alpha1.ListApiKeyResponse, error) {
	req := &v1alpha1.ListApiKeyRequest{}

	switch parsedOrg := organization.(type) {
	case *v1alpha1.ListApiKeyRequest_OrganizationName:
		req.Organization = parsedOrg
	case *v1alpha1.ListApiKeyRequest_OrganizationId:
		req.Organization = parsedOrg
	default:
		return nil, fmt.Errorf("unknown organization type: %T", organization)
	}

	resp, err := c.ApiKeyServiceClient.ListApiKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	if resp == nil {
		return nil, errors.New("received nil response from list api keys")
	}

	return resp, nil
}

func getOrganizationNameFromRepository(repository string) (string, error) {
	const orgPartsNumber = 2

	parts := strings.Split(repository, "/")
	if len(parts) != orgPartsNumber {
		return "", fmt.Errorf("invalid repository format: %s. Expected format is '<org>/<repo>'", repository)
	}

	return parts[0], nil
}

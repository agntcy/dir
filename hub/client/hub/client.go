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

	corev1alpha1 "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/hub/api/v1alpha1"
	"github.com/opencontainers/go-digest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const chunkSize = 4096 // 4KB

// Client defines the interface for interacting with the Agent Hub backend for agent operations.
type Client interface {
	// PushAgent uploads an agent to the hub and returns the response or an error.
	PushAgent(ctx context.Context, agent []byte, repositoryID any) (*v1alpha1.PushRecordResponse, error)
	// PullAgent downloads an agent from the hub and returns the agent data or an error.
	PullAgent(ctx context.Context, request *v1alpha1.PullRecordRequest) ([]byte, error)
}

// client implements the Client interface for the Agent Hub backend.
type client struct {
	v1alpha1.AgentDirServiceClient
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

	return &client{AgentDirServiceClient: v1alpha1.NewAgentDirServiceClient(conn)}, nil
}

// PushAgent uploads an agent to the hub in chunks and returns the response or an error.
func (c *client) PushAgent(ctx context.Context, agent []byte, repositoryID any) (*v1alpha1.PushRecordResponse, error) {
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

		switch parsedRepoID := repositoryID.(type) {
		case *v1alpha1.PushRecordRequest_RepositoryName:
			msg.Repository = parsedRepoID
		case *v1alpha1.PushRecordRequest_RepositoryId:
			msg.Repository = parsedRepoID
		default:
			return nil, fmt.Errorf("unknown repository type: %T", repositoryID)
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

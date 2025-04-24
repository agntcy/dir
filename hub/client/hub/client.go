// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

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

type Client interface {
	PushAgent(ctx context.Context, agent []byte, repositoryID any) (*v1alpha1.PushAgentResponse, error)
	PullAgent(ctx context.Context, request *v1alpha1.PullAgentRequest) ([]byte, error)
}

type client struct {
	v1alpha1.AgentDirServiceClient
}

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

func (c *client) PushAgent(ctx context.Context, agent []byte, repositoryID any) (*v1alpha1.PushAgentResponse, error) {
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

	stream, err := c.AgentDirServiceClient.PushAgent(ctx)
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

		msg := &v1alpha1.PushAgentRequest{
			Model: &corev1alpha1.Object{
				Data: buf[:n],
				Ref:  ref,
			},
		}

		switch parsedRepoID := repositoryID.(type) {
		case *v1alpha1.PushAgentRequest_RepositoryName:
			msg.Repository = parsedRepoID
		case *v1alpha1.PushAgentRequest_RepositoryId:
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

func (c *client) PullAgent(ctx context.Context, request *v1alpha1.PullAgentRequest) ([]byte, error) {
	stream, err := c.AgentDirServiceClient.PullAgent(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	var buffer bytes.Buffer

	for {
		var chunk *v1alpha1.PullAgentResponse

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

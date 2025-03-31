package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/agntcy/hub/api/v1alpha1"
)

func NewClient(serverAddr string) (v1alpha1.AgentServiceClient, error) {
	// Create connection
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	return v1alpha1.NewAgentServiceClient(conn), nil
}

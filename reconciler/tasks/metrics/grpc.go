// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCProviderCounter implements ProviderCounterAPI by calling the
// RoutingService.GetProviderCount RPC on the apiserver. Used by the standalone
// reconciler process, which has no direct access to the routing layer.
type GRPCProviderCounter struct {
	client routingv1.RoutingServiceClient
	conn   *grpc.ClientConn
}

// NewGRPCProviderCounter dials the apiserver at addr with the provided dial
// options and returns a GRPCProviderCounter. If no options are provided,
// insecure credentials are used as a fallback. For authenticated deployments
// pass the appropriate TLS/JWT options via opts — the same options returned by
// authn.Service.GetClientOptions() or built from a client.Config.
// The caller is responsible for calling Close when done.
func NewGRPCProviderCounter(addr string, opts ...grpc.DialOption) (*GRPCProviderCounter, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to apiserver at %s: %w", addr, err)
	}

	return &GRPCProviderCounter{
		client: routingv1.NewRoutingServiceClient(conn),
		conn:   conn,
	}, nil
}

// NewGRPCProviderCounterFromClient builds a GRPCProviderCounter from an
// already-dialed RoutingServiceClient. The caller owns the connection lifecycle;
// Close on the returned counter is a no-op.
func NewGRPCProviderCounterFromClient(c routingv1.RoutingServiceClient) *GRPCProviderCounter {
	return &GRPCProviderCounter{client: c}
}

// GetProviderCount calls the apiserver's RoutingService.GetProviderCount RPC.
func (g *GRPCProviderCounter) GetProviderCount(ctx context.Context, cid string) (int, error) {
	resp, err := g.client.GetProviderCount(ctx, &routingv1.GetProviderCountRequest{Cid: cid})
	if err != nil {
		return 0, fmt.Errorf("GetProviderCount RPC failed for %s: %w", cid, err)
	}

	return int(resp.GetCount()), nil
}

// Search streams RoutingService.Search results. Used by the sync task
// in standalone mode, which reaches the same apiserver routing client.
func (g *GRPCProviderCounter) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	stream, err := g.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search stream: %w", err)
	}

	resCh := make(chan *routingv1.SearchResponse, 100) //nolint:mnd

	go func() {
		defer close(resCh)

		for {
			obj, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				logger.Error("error receiving search result", "error", err)

				return
			}

			select {
			case resCh <- obj:
			case <-ctx.Done():
				logger.Error("context cancelled while receiving search response", "error", ctx.Err())

				return
			}
		}
	}()

	return resCh, nil
}

// Close releases the underlying gRPC connection. It is a no-op when the
// counter was created with NewGRPCProviderCounterFromClient.
func (g *GRPCProviderCounter) Close() error {
	if g.conn == nil {
		return nil
	}

	if err := g.conn.Close(); err != nil {
		return fmt.Errorf("failed to close gRPC connection: %w", err)
	}

	return nil
}

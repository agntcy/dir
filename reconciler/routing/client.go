// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package routing provides the reconciler's client for the apiserver's
// RoutingService.
//
// The routing layer uses an embedded Badger key-value store that does NOT
// support concurrent multi-process access, so the standalone reconciler cannot
// share the datastore directory with the server via a volume mount. It reaches
// routing over gRPC instead. In daemon mode the two run in one process and
// server/types.RoutingAPI is used directly; both satisfy the narrow interfaces
// the tasks declare.
package routing

import (
	"context"
	"errors"
	"fmt"
	"io"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var logger = logging.Logger("reconciler/routing")

// searchBufferSize is how many search results are held while the consumer
// catches up, so a slow consumer does not stall the gRPC stream.
const searchBufferSize = 100

// Client calls the apiserver's RoutingService over gRPC. It is used by the
// standalone reconciler process, which has no direct access to the routing
// layer.
type Client struct {
	client routingv1.RoutingServiceClient
	conn   *grpc.ClientConn
}

// New dials the apiserver at addr with the provided dial options and returns a
// Client. If no options are provided, insecure credentials are used as a
// fallback. For authenticated deployments pass the appropriate TLS/JWT options
// via opts — the same options returned by authn.Service.GetClientOptions() or
// built from a client.Config.
// The caller is responsible for calling Close when done.
func New(addr string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to apiserver at %s: %w", addr, err)
	}

	return &Client{
		client: routingv1.NewRoutingServiceClient(conn),
		conn:   conn,
	}, nil
}

// NewFromClient builds a Client from an already-dialed RoutingServiceClient.
// The caller owns the connection lifecycle; Close on the returned Client is a
// no-op.
func NewFromClient(c routingv1.RoutingServiceClient) *Client {
	return &Client{client: c}
}

// GetProviderCount returns the number of distinct peers announcing cid.
func (c *Client) GetProviderCount(ctx context.Context, cid string) (int, error) {
	resp, err := c.client.GetProviderCount(ctx, &routingv1.GetProviderCountRequest{Cid: cid})
	if err != nil {
		return 0, fmt.Errorf("GetProviderCount RPC failed for %s: %w", cid, err)
	}

	return int(resp.GetCount()), nil
}

// Search streams records other peers announce that match req.
func (c *Client) Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	stream, err := c.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search stream: %w", err)
	}

	resCh := make(chan *routingv1.SearchResponse, searchBufferSize)

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

// Close releases the underlying gRPC connection. It is a no-op when the client
// was created with NewFromClient.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close gRPC connection: %w", err)
	}

	return nil
}

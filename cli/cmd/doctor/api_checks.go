// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/client"
)

func directoryAPI(ctx context.Context, addr string, timeout time.Duration) checkResult {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	var dialer net.Dialer

	conn, err := dialer.DialContext(checkCtx, "tcp", addr)

	elapsed := time.Since(start)
	if err != nil {
		return checkResult{
			Name:    "directory_api_tcp",
			Status:  statusFail,
			Message: fmt.Sprintf("Directory API is not reachable at %s", addr),
			Elapsed: elapsed.String(),
			Details: map[string]string{
				"address": addr,
				"error":   err.Error(),
			},
		}
	}

	defer conn.Close()

	return checkResult{
		Name:    "directory_api_tcp",
		Status:  statusPass,
		Message: fmt.Sprintf("Directory API is reachable at %s", addr),
		Elapsed: elapsed.String(),
		Details: map[string]string{
			"address": addr,
		},
	}
}

func directoryClient(ctx context.Context, cfg *client.Config) (*client.Client, checkResult) {
	start := time.Now()
	dirClient, err := client.New(ctx, client.WithConfig(cfg))

	elapsed := time.Since(start)
	if err != nil {
		return nil, checkResult{
			Name:    "grpc_client_setup",
			Status:  statusFail,
			Message: "Failed to create Directory gRPC client",
			Elapsed: elapsed.String(),
			Details: map[string]string{
				"address": cfg.ServerAddress,
				"error":   normalizeClientError(err).Error(),
			},
		}
	}

	return dirClient, checkResult{
		Name:    "grpc_client_setup",
		Status:  statusPass,
		Message: "Created Directory gRPC client",
		Elapsed: elapsed.String(),
		Details: map[string]string{
			"address": cfg.ServerAddress,
		},
	}
}

func normalizeClientError(err error) error {
	if _, ok := errors.AsType[*client.AmbiguousTokenCacheError](err); ok {
		return fmt.Errorf("%w; use `--oidc-issuer` or DIRECTORY_CLIENT_OIDC_ISSUER to select the cached issuer", err)
	}

	if strings.Contains(err.Error(), "no OIDC access token") {
		return fmt.Errorf("%w; run `dirctl auth login` for the selected issuer, use `--oidc-issuer` if multiple issuers are cached, or set `--auth-token` / `DIRECTORY_CLIENT_AUTH_TOKEN`", err)
	}

	return err
}

func routingList(ctx context.Context, dirClient *client.Client, addr string, timeout time.Duration) checkResult {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	limit := uint32(1)

	stream, err := dirClient.RoutingServiceClient.List(checkCtx, &routingv1.ListRequest{
		Limit: &limit,
	})
	if err != nil {
		elapsed := time.Since(start)

		return checkResult{
			Name:    "routing_list",
			Status:  statusFail,
			Message: "Directory routing List RPC failed",
			Elapsed: elapsed.String(),
			Details: map[string]string{
				"address": addr,
				"error":   err.Error(),
			},
		}
	}

	resp, err := stream.Recv()

	elapsed := time.Since(start)
	if errors.Is(err, io.EOF) {
		return checkResult{
			Name:    "routing_list",
			Status:  statusPass,
			Message: "Directory routing List RPC is available; no local records returned",
			Elapsed: elapsed.String(),
			Details: map[string]string{
				"address": addr,
			},
		}
	}

	if err != nil {
		return checkResult{
			Name:    "routing_list",
			Status:  statusFail,
			Message: "Directory routing List stream failed",
			Elapsed: elapsed.String(),
			Details: map[string]string{
				"address": addr,
				"error":   err.Error(),
			},
		}
	}

	return checkResult{
		Name:    "routing_list",
		Status:  statusPass,
		Message: "Directory routing List RPC is available; local records returned",
		Elapsed: elapsed.String(),
		Details: map[string]string{
			"address": addr,
			"record":  fmt.Sprintf("%v", resp.GetRecordRef()),
		},
	}
}

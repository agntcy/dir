// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	synctypes "github.com/agntcy/dir/api/store/v1alpha2"
)

func (c *Client) CreateSync(ctx context.Context, remoteURL string) (string, error) {
	meta, err := c.SyncServiceClient.CreateSync(ctx, &synctypes.CreateSyncRequest{
		RemoteDirectoryUrl: remoteURL,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create sync: %w", err)
	}

	return meta.GetSyncId(), nil
}

func (c *Client) ListSyncs(ctx context.Context, req *synctypes.ListSyncsRequest) (<-chan *synctypes.ListSyncsItem, error) {
	stream, err := c.SyncServiceClient.ListSyncs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create list syncs stream: %w", err)
	}

	resultCh := make(chan *synctypes.ListSyncsItem)

	go func() {
		defer close(resultCh)

		for {
			item, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				logger.Error("failed to receive list syncs response", "error", err)

				break
			}

			select {
			case resultCh <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	return resultCh, nil
}

func (c *Client) GetSync(ctx context.Context, syncID string) (*synctypes.GetSyncResponse, error) {
	meta, err := c.SyncServiceClient.GetSync(ctx, &synctypes.GetSyncRequest{
		SyncId: syncID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get sync: %w", err)
	}

	return meta, nil
}

func (c *Client) DeleteSync(ctx context.Context, syncID string) error {
	_, err := c.SyncServiceClient.DeleteSync(ctx, &synctypes.DeleteSyncRequest{
		SyncId: syncID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete sync: %w", err)
	}

	return nil
}

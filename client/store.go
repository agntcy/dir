// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

func (c *Client) Push(ctx context.Context, meta *coretypes.ObjectMeta, reader io.Reader) (*coretypes.Digest, error) {
	stream, err := c.StoreServiceClient.Push(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create push stream: %w", err)
	}

	buf := make([]byte, chunkSize)

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read data: %w", err)
		}
		if n == 0 {
			break
		}

		obj := &coretypes.Object{
			Metadata: meta,
			Size:     uint64(n),
			Data:     buf[:n],
		}

		if err := stream.Send(obj); err != nil {
			return nil, fmt.Errorf("could not send object: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("could not receive response: %w", err)
	}

	return resp.Digest, nil
}

func (c *Client) Pull(ctx context.Context, dig *coretypes.Digest) (io.Reader, error) {
	req := &coretypes.ObjectRef{
		Digest: dig,
	}

	stream, err := c.StoreServiceClient.Pull(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	var buffer bytes.Buffer

	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to receive object: %w", err)
		}

		if _, err := buffer.Write(obj.Data); err != nil {
			return nil, fmt.Errorf("failed to write data to buffer: %w", err)
		}
	}

	return &buffer, nil
}

func (c *Client) Lookup(ctx context.Context, dig *coretypes.Digest) (*coretypes.ObjectMeta, error) {
	req := &coretypes.ObjectRef{
		Digest: dig,
	}

	meta, err := c.StoreServiceClient.Lookup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup object: %w", err)
	}

	return meta, nil
}

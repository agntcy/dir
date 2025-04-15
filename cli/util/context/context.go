// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"

	"github.com/agntcy/dir/client"
)

type ClientContextKeyType string

const ClientContextKey ClientContextKeyType = "ContextDirClient"

func SetClientForContext(ctx context.Context, c *client.Client) context.Context {
	return context.WithValue(ctx, ClientContextKey, c)
}

func GetClientFromContext(ctx context.Context) (*client.Client, bool) {
	cli, ok := ctx.Value(ClientContextKey).(*client.Client)

	return cli, ok
}

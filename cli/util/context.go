// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"

	"github.com/agntcy/dir/registry/client"
)

type ClientContextKeyType string

const ClientContextKey ClientContextKeyType = "ContextRegistryClient"

func SetRegistryClientForContext(ctx context.Context, c *client.Client) context.Context {
	return context.WithValue(ctx, ClientContextKey, c)
}

func GetRegistryClientFromContext(ctx context.Context) (*client.Client, bool) {
	cli, ok := ctx.Value(ClientContextKey).(*client.Client)

	return cli, ok
}

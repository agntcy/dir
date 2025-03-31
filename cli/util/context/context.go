// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/agntcy/dir/cli/secretstore"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/hub/api/v1alpha1"
)

type ContextKeyType string

const DirClientContextKey ContextKeyType = "ContextDirClient"
const HubClientContextKey ContextKeyType = "ContextHubClient"
const SecretStoreContextKey ContextKeyType = "ContextSecretStore"

func SetDirClientForContext(ctx context.Context, c *client.Client) context.Context {
	return setCliContext(ctx, DirClientContextKey, c)
}

func GetDirClientFromContext(ctx context.Context) (*client.Client, bool) {
	return getCliContext[*client.Client](ctx, DirClientContextKey)
}

func SetHubClientForContext(ctx context.Context, c v1alpha1.AgentServiceClient) context.Context {
	return setCliContext(ctx, HubClientContextKey, c)
}

func GetHubClientFromContext(ctx context.Context) (v1alpha1.AgentServiceClient, bool) {
	return getCliContext[v1alpha1.AgentServiceClient](ctx, HubClientContextKey)
}

func SetSecretStoreForContext(ctx context.Context, s secretstore.SecretStore) context.Context {
	return setCliContext(ctx, SecretStoreContextKey, s)
}

func GetSecretStoreFromContext(ctx context.Context) (secretstore.SecretStore, bool) {
	return getCliContext[secretstore.SecretStore](ctx, SecretStoreContextKey)
}

func setCliContext[T any](ctx context.Context, key ContextKeyType, c T) context.Context {
	return context.WithValue(ctx, key, c)
}

func getCliContext[T any](ctx context.Context, key ContextKeyType) (T, bool) {
	cli, ok := ctx.Value(key).(T)

	return cli, ok
}

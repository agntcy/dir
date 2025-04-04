// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/agntcy/dir/api/hub/v1alpha1"
	"github.com/agntcy/dir/cli/hub/idp"
	secretstore2 "github.com/agntcy/dir/cli/hub/secretstore"
	"github.com/agntcy/dir/client"
)

type KeyType string

const (
	DirClientContextKey            KeyType = "ContextDirClient"
	HubClientContextKey            KeyType = "ContextHubClient"
	SecretStoreContextKey          KeyType = "ContextSecretStore"
	CurrentHubSecretContextKey     KeyType = "ContextCurrentHubSecret"
	IdpClientContextKey            KeyType = "ContextIdpClient"
	CurrentServerAddressContextKey KeyType = "ContextCurrentServerAddress"
)

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

func SetSecretStoreForContext(ctx context.Context, s secretstore2.SecretStore) context.Context {
	return setCliContext(ctx, SecretStoreContextKey, s)
}

func GetSecretStoreFromContext(ctx context.Context) (secretstore2.SecretStore, bool) {
	return getCliContext[secretstore2.SecretStore](ctx, SecretStoreContextKey)
}

func SetCurrentHubSecretForContext(ctx context.Context, secret *secretstore2.HubSecret) context.Context {
	return setCliContext(ctx, CurrentHubSecretContextKey, secret)
}

func GetCurrentHubSecretFromContext(ctx context.Context) (*secretstore2.HubSecret, bool) {
	return getCliContext[*secretstore2.HubSecret](ctx, CurrentHubSecretContextKey)
}

func SetIdpClientForContext(ctx context.Context, c idp.Client) context.Context {
	return setCliContext(ctx, IdpClientContextKey, c)
}

func GetIdpClientFromContext(ctx context.Context) (idp.Client, bool) {
	return getCliContext[idp.Client](ctx, IdpClientContextKey)
}

func SetCurrentServerAddressForContext(ctx context.Context, address string) context.Context {
	return setCliContext(ctx, CurrentServerAddressContextKey, address)
}

func GetCurrentServerAddressFromContext(ctx context.Context) (string, bool) {
	return getCliContext[string](ctx, CurrentServerAddressContextKey)
}

func setCliContext[T any](ctx context.Context, key KeyType, c T) context.Context {
	return context.WithValue(ctx, key, c)
}

func getCliContext[T any](ctx context.Context, key KeyType) (T, bool) {
	cli, ok := ctx.Value(key).(T)

	return cli, ok
}

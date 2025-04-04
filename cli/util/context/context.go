// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/hub/sessionstore"
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

func SetSessionStoreForContext(ctx context.Context, s sessionstore.SessionStore) context.Context {
	return setCliContext(ctx, SecretStoreContextKey, s)
}

func GetSessionStoreFromContext(ctx context.Context) (sessionstore.SessionStore, bool) {
	return getCliContext[sessionstore.SessionStore](ctx, SecretStoreContextKey)
}

func SetCurrentHubSessionForContext(ctx context.Context, secret *sessionstore.HubSession) context.Context {
	return setCliContext(ctx, CurrentHubSecretContextKey, secret)
}

func GetCurrentHubSecretFromContext(ctx context.Context) (*sessionstore.HubSession, bool) {
	return getCliContext[*sessionstore.HubSession](ctx, CurrentHubSecretContextKey)
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

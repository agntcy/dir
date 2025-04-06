// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/hub/okta"
	"github.com/agntcy/dir/cli/hub/sessionstore"
	"github.com/agntcy/dir/client"
)

type KeyType string

const (
	DirClientContextKey        KeyType = "ContextDirClient"
	SecretStoreContextKey      KeyType = "ContextSecretStore"
	CurrentHubSecretContextKey KeyType = "ContextCurrentHubSecret"
	OktaClientContextKey       KeyType = "ContextIdpClient"
	UserTenantsContextKey      KeyType = "ContextUserTenants"
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

func GetCurrentHubSessionFromContext(ctx context.Context) (*sessionstore.HubSession, bool) {
	return getCliContext[*sessionstore.HubSession](ctx, CurrentHubSecretContextKey)
}

func SetOktaClientForContext(ctx context.Context, c okta.Client) context.Context {
	return setCliContext(ctx, OktaClientContextKey, c)
}

func GetOktaClientFromContext(ctx context.Context) (okta.Client, bool) {
	return getCliContext[okta.Client](ctx, OktaClientContextKey)
}

func SetUserTenantsForContext(ctx context.Context, tenants []*idp.TenantResponse) context.Context {
	return setCliContext(ctx, UserTenantsContextKey, tenants)
}

func GetUserTenantsFromContext(ctx context.Context) ([]*idp.TenantResponse, bool) {
	return getCliContext[[]*idp.TenantResponse](ctx, UserTenantsContextKey)
}

func setCliContext[T any](ctx context.Context, key KeyType, c T) context.Context {
	return context.WithValue(ctx, key, c)
}

func getCliContext[T any](ctx context.Context, key KeyType) (T, bool) {
	cli, ok := ctx.Value(key).(T)

	return cli, ok
}

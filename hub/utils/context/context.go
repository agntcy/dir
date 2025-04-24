// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/spf13/cobra"
)

type contextKey string

const (
	sessionStoreContextKey      contextKey = "sessionStore"
	currentHubSessionContextKey contextKey = "currentHubSession"
	oktaClientContextKey        contextKey = "oktaClient"
	tenantListContextKey        contextKey = "tenantList"
)

func setCmdContext[T any](cmd *cobra.Command, key contextKey, value T) bool {
	if cmd == nil {
		return false
	}

	if key == "" {
		return false
	}

	cmd.SetContext(context.WithValue(cmd.Context(), key, value))

	return true
}

func getCmdContext[T any](cmd *cobra.Command, key contextKey) (T, bool) {
	var def T
	if cmd == nil {
		return def, false
	}

	if key == "" {
		return def, false
	}

	value := cmd.Context().Value(key)
	if value == nil {
		return def, false
	}

	v, ok := value.(T)

	return v, ok
}

func SetSessionStoreForContext(cmd *cobra.Command, store sessionstore.SessionStore) bool {
	return setCmdContext(cmd, sessionStoreContextKey, store)
}

func GetSessionStoreFromContext(cmd *cobra.Command) (sessionstore.SessionStore, bool) {
	return getCmdContext[sessionstore.SessionStore](cmd, sessionStoreContextKey)
}

func SetCurrentHubSessionForContext(cmd *cobra.Command, session *sessionstore.HubSession) bool {
	return setCmdContext(cmd, currentHubSessionContextKey, session)
}

func GetCurrentHubSessionFromContext(cmd *cobra.Command) (*sessionstore.HubSession, bool) {
	return getCmdContext[*sessionstore.HubSession](cmd, currentHubSessionContextKey)
}

func SetOktaClientForContext(cmd *cobra.Command, client okta.Client) bool {
	return setCmdContext(cmd, oktaClientContextKey, client)
}

func GetOktaClientFromContext(cmd *cobra.Command) (okta.Client, bool) {
	return getCmdContext[okta.Client](cmd, oktaClientContextKey)
}

func SetTenantListForContext(cmd *cobra.Command, tenants []*idp.TenantResponse) bool {
	return setCmdContext(cmd, tenantListContextKey, tenants)
}

func GetUserTenantsFromContext(cmd *cobra.Command) ([]*idp.TenantResponse, bool) {
	return getCmdContext[[]*idp.TenantResponse](cmd, tenantListContextKey)
}

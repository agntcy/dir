// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"

	v1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	authUtils "github.com/agntcy/dir/hub/auth/utils"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/sessionstore"
)

// A structure that combines an API key with its associated role name.
type APIKeyWithRoleName struct {
	ClientID string `json:"client_id"` // The client ID of the API key.
	RoleName string `json:"role_name"` // The name of the role associated with the API key.
}

type APIKeyWithSecretWithRoleName struct {
	ClientID string `json:"client_id"` // The client ID of the API key.
	Secret   string `json:"secret"`    // The secret of the API key.
	RoleName string `json:"role_name"` // The name of the role associated
}

// CreateAPIKey creates a new API key in the hub and returns the response.
// It uses the provided session for authentication.
func CreateAPIKey(
	ctx context.Context,
	hc hubClient.Client,
	role string,
	organization any,
	session *sessionstore.HubSession,
) (*APIKeyWithSecretWithRoleName, error) {
	ctx = authUtils.AddAuthToContext(ctx, session)

	resp, err := hc.CreateAPIKey(ctx, role, organization)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	if resp == nil || resp.GetToken() == nil || resp.GetToken().GetApikey() == nil {
		return nil, fmt.Errorf("invalid response from server: %v", resp)
	}

	roleName, ok := v1alpha1.Role_name[int32(resp.GetToken().GetApikey().GetRole())]
	if !ok {
		return nil, fmt.Errorf("invalid role: %v", resp.GetToken().GetApikey().GetRole())
	}

	return &APIKeyWithSecretWithRoleName{
		ClientID: resp.GetToken().GetApikey().GetClientId(),
		Secret:   resp.GetToken().GetSecret(),
		RoleName: roleName,
	}, nil
}

// DeleteAPIKey deletes an API key from the hub and returns the response.
// It uses the provided session for authentication.
func DeleteAPIKey(
	ctx context.Context,
	hc hubClient.Client,
	apikeyID string,
	session *sessionstore.HubSession,
) (*v1alpha1.DeleteApiKeyResponse, error) {
	ctx = authUtils.AddAuthToContext(ctx, session)

	resp, err := hc.DeleteAPIKey(ctx, apikeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete API key: %w", err)
	}

	return resp, nil
}

// ListAPIKeys lists API keys from the hub and returns the response.
// It uses the provided session for authentication.
func ListAPIKeys(
	ctx context.Context,
	hc hubClient.Client,
	organization any,
	session *sessionstore.HubSession,
) ([]*APIKeyWithRoleName, error) {
	ctx = authUtils.AddAuthToContext(ctx, session)

	resp, err := hc.ListAPIKeys(ctx, organization)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	apiKeysWithRoleNames := make([]*APIKeyWithRoleName, len(resp.GetApikeys()))

	for i, apiKey := range resp.GetApikeys() {
		roleName, ok := v1alpha1.Role_name[int32(apiKey.GetRole())]
		if !ok {
			return nil, fmt.Errorf("invalid role: %v", apiKey.GetRole())
		}

		apiKeysWithRoleNames[i] = &APIKeyWithRoleName{
			ClientID: apiKey.GetClientId(),
			RoleName: roleName,
		}
	}

	return apiKeysWithRoleNames, nil
}

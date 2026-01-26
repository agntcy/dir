// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"fmt"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client"
	authnconfig "github.com/agntcy/dir/server/authn/config"
	syncconfig "github.com/agntcy/dir/server/sync/config"
)

// CredentialsResult holds the result of credential negotiation with a remote Directory node.
type CredentialsResult struct {
	RegistryAddress string
	RepositoryName  string
	Credentials     syncconfig.AuthConfig
}

// FullRepositoryURL returns the full repository URL (address + path).
func (r *CredentialsResult) FullRepositoryURL() string {
	if r.RepositoryName != "" {
		return r.RegistryAddress + "/" + r.RepositoryName
	}

	return r.RegistryAddress
}

// NegotiateCredentials negotiates registry credentials with a remote Directory node.
func NegotiateCredentials(ctx context.Context, remoteDirectoryURL string, authnConfig authnconfig.Config) (*CredentialsResult, error) {
	// Build client config based on authn settings
	clientConfig := &client.Config{
		ServerAddress: remoteDirectoryURL,
		AuthMode:      "insecure",
	}

	if authnConfig.Enabled {
		clientConfig.AuthMode = string(authnConfig.Mode)
		clientConfig.SpiffeSocketPath = authnConfig.SocketPath

		if len(authnConfig.Audiences) > 0 {
			clientConfig.JWTAudience = authnConfig.Audiences[0]
		}
	}

	dirClient, err := client.New(ctx, client.WithConfig(clientConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to remote node %s: %w", remoteDirectoryURL, err)
	}
	defer dirClient.Close()

	resp, err := dirClient.RequestRegistryCredentials(ctx, &storev1.RequestRegistryCredentialsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to request registry credentials from %s: %w", remoteDirectoryURL, err)
	}

	if !resp.GetSuccess() {
		return nil, fmt.Errorf("credential negotiation failed: %s", resp.GetErrorMessage())
	}

	return &CredentialsResult{
		RegistryAddress: resp.GetRegistryAddress(),
		RepositoryName:  resp.GetRepositoryName(),
		Credentials: syncconfig.AuthConfig{
			Username: resp.GetBasicAuth().GetUsername(),
			Password: resp.GetBasicAuth().GetPassword(),
			Insecure: resp.GetInsecure(),
		},
	}, nil
}

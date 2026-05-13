// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	authnconfig "github.com/agntcy/dir/server/authn/config"
)

// CredentialsResult holds the result of credential negotiation with a remote Directory node.
type CredentialsResult struct {
	RegistryAddress string
	RepositoryName  string
	Credentials     AuthConfig
}

// AuthConfig represents the configuration for authentication.
//
//nolint:gosec // G117: intentional config field for registry auth
type AuthConfig struct {
	Username string `json:"username,omitempty" mapstructure:"username"`
	Password string `json:"password,omitempty" mapstructure:"password"`
	Insecure bool   `json:"insecure,omitempty" mapstructure:"insecure"` // Use plain HTTP instead of HTTPS
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
	clientConfig, err := buildClientConfigForRemote(remoteDirectoryURL, authnConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build client config for %s: %w", remoteDirectoryURL, err)
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

	// Extract credentials from response, handling nil BasicAuth
	var username, password string
	if basicAuth := resp.GetBasicAuth(); basicAuth != nil {
		username = basicAuth.GetUsername()
		password = basicAuth.GetPassword()
	}

	return &CredentialsResult{
		RegistryAddress: resp.GetRegistryAddress(),
		RepositoryName:  resp.GetRepositoryName(),
		Credentials: AuthConfig{
			Username: username,
			Password: password,
			Insecure: resp.GetInsecure(),
		},
	}, nil
}

// buildClientConfigForRemote prepares the client.Config used to talk to a
// remote Directory node.
func buildClientConfigForRemote(remoteDirectoryURL string, authnConfig authnconfig.Config) (*client.Config, error) {
	contextName, err := resolveContextByServerAddress(remoteDirectoryURL)
	if err != nil {
		return nil, err
	}

	if contextName != "" {
		cfg, _, err := clientconfig.Resolve(clientconfig.ResolveOptions{
			Context:        contextName,
			SkipValidation: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve client context %q: %w", contextName, err)
		}

		logger.Info("Using dirctl client context for remote directory", "context", contextName, "remote_directory", remoteDirectoryURL, "auth_mode", cfg.AuthMode)

		return cfg, nil
	}

	cfg := &client.Config{
		ServerAddress: remoteDirectoryURL,
		AuthMode:      "insecure",
	}

	if authnConfig.Enabled {
		cfg.AuthMode = string(authnConfig.Mode)
		cfg.SpiffeSocketPath = authnConfig.SocketPath

		if len(authnConfig.Audiences) > 0 {
			cfg.JWTAudience = authnConfig.Audiences[0]
		}
	}

	return cfg, nil
}

// resolveContextByServerAddress returns the name of the dirctl client context
// whose server_address matches the provided value.
func resolveContextByServerAddress(serverAddress string) (string, error) {
	file, err := clientconfig.LoadFile("")
	if err != nil {
		// A missing config file is a valid setup (e.g. headless daemons),
		// so treat it the same as "no matching context".
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}

		return "", fmt.Errorf("failed to load dirctl client config: %w", err)
	}

	needle := canonicalServerAddress(serverAddress)

	var matches []string

	for name, ctx := range file.Contexts {
		if canonicalServerAddress(ctx.ServerAddress) == needle {
			matches = append(matches, name)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)

		return "", fmt.Errorf("multiple client contexts match server_address %q: %v", serverAddress, matches)
	}
}

// canonicalServerAddress normalizes a Directory server address for comparison.
func canonicalServerAddress(s string) string {
	s = strings.TrimSpace(s)

	scheme := ""

	switch {
	case strings.HasPrefix(s, "https://"):
		scheme = "https"
		s = strings.TrimPrefix(s, "https://")
	case strings.HasPrefix(s, "http://"):
		scheme = "http"
		s = strings.TrimPrefix(s, "http://")
	}

	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}

	if scheme != "" && !strings.Contains(s, ":") {
		switch scheme {
		case "https":
			s += ":443"
		case "http":
			s += ":80"
		}
	}

	return s
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"
	"net/http"

	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func stringPtr(s string) *string {
	return &s
}

// NewORASRepository creates a new ORAS repository client configured with authentication.
func NewORASRepository(cfg ociconfig.Config) (*remote.Repository, error) {
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", cfg.RegistryAddress, cfg.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// Configure repository
	repo.PlainHTTP = cfg.Insecure
	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-client"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			cfg.RegistryAddress,
			auth.Credential{
				Username:     cfg.Username,
				Password:     cfg.Password,
				RefreshToken: cfg.RefreshToken,
				AccessToken:  cfg.AccessToken,
			},
		),
	}

	return repo, nil
}

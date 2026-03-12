// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"fmt"
	"net/http"

	"github.com/agntcy/dir/server/store/oci/internal/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func (s *store) Server() (http.Handler, error) {
	// Create client
	reg, err := remote.NewRegistry(s.config.RegistryAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	// Configure registry client
	reg.PlainHTTP = s.config.Insecure
	reg.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-client"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			s.config.RegistryAddress,
			auth.Credential{
				Username:     s.config.Username,
				Password:     s.config.Password,
				RefreshToken: s.config.RefreshToken,
				AccessToken:  s.config.AccessToken,
			},
		),
	}

	// Create OCI serving registry server
	server, err := registry.NewServer(reg)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry server: %w", err)
	}

	return server, nil
}

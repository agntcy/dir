// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/agntcy/dir/server/registry/config"
	orasreg "github.com/agntcy/dir/server/registry/internal/registry"
	"github.com/agntcy/dir/server/store"
	storeconfig "github.com/agntcy/dir/server/store/config"
	"github.com/agntcy/dir/server/types"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type registry struct {
	server *http.Server
}

func New(cfg config.Config, storeCfg storeconfig.Config) (types.RegistryAPI, error) {
	// Validate configuration
	if storeCfg.Provider != string(store.OCI) {
		return nil, fmt.Errorf("unsupported registry provider: %s", storeCfg.Provider)
	}

	if storeCfg.OCI.RegistryAddress == "" {
		return nil, fmt.Errorf("registry address is required")
	}

	// Create client
	reg, err := remote.NewRegistry(storeCfg.OCI.RegistryAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	// Configure registry client
	reg.PlainHTTP = storeCfg.OCI.Insecure
	reg.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-proxy"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			storeCfg.OCI.RegistryAddress,
			auth.Credential{
				Username:     storeCfg.OCI.Username,
				Password:     storeCfg.OCI.Password,
				RefreshToken: storeCfg.OCI.RefreshToken,
				AccessToken:  storeCfg.OCI.AccessToken,
			},
		),
	}

	// Create OCI serving registry server
	apiHandler, err := orasreg.NewServer(reg)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry server: %w", err)
	}

	// Register HTTP handler for OCI Distribution API
	handler := http.NewServeMux()
	handler.Handle("/v2/", apiHandler)

	return &registry{
		server: &http.Server{
			Addr:              cfg.ListenAddress,
			Handler:           handler,
			ReadHeaderTimeout: 3 * time.Second, //nolint:mnd
		},
	}, nil
}

// Start implements [types.RegistryAPI].
func (r *registry) Start() {
	go func() {
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Errorf("failed to start HTTP server: %w", err))
		}
	}()
}

// Stop implements [types.RegistryAPI].
func (r *registry) Stop() error {
	return r.server.Close() //nolint:wrapcheck
}

package provider

import (
	"fmt"

	"github.com/agntcy/dir/registry/server/config"
	"github.com/agntcy/dir/registry/server/provider/localfs"
	"github.com/agntcy/dir/registry/server/provider/oci"
	"github.com/agntcy/dir/registry/types"
)

type Provider string

const (
	LocalFS = Provider("localfs")
	OCI     = Provider("oci")
)

func New(config *config.Config) (types.Registry, error) {
	switch provider := Provider(config.Provider); provider {
	case OCI:
		registry, err := oci.NewRegistryProvider(config.OCI)
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI registry provider: %w", err)
		}
		return registry, nil
	case LocalFS:
		registry, err := localfs.NewRegistryProvider(config.LocalFS.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to create localfs registry provider: %w", err)
		}
		return registry, nil
	default:
		return nil, fmt.Errorf("unsupported provider=%s", provider)
	}
}

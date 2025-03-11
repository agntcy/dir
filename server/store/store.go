package store

import (
	"fmt"

	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
)

type Provider string

const (
	OCI = Provider("oci")
)

func New(config *config.Config) (types.StoreAPI, error) {
	switch provider := Provider(config.Provider); provider {
	case OCI:
		store, err := oci.New(config.OCI)
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI store: %w", err)
		}
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported provider=%s", provider)
	}
}

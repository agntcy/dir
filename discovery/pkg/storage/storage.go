package storage

import (
	"fmt"

	"github.com/agntcy/dir/discovery/pkg/storage/config"
	"github.com/agntcy/dir/discovery/pkg/storage/etcd"
	"github.com/agntcy/dir/discovery/pkg/types"
)

func NewReader(cfg config.Config) (types.StoreReader, error) {
	switch cfg.StorageType {
	case "etcd":
		return etcd.NewReader(cfg.Etcd)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.StorageType)
	}
}

func NewWriter(cfg config.Config) (types.StoreWriter, error) {
	switch cfg.StorageType {
	case "etcd":
		return etcd.NewWriter(cfg.Etcd)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.StorageType)
	}
}

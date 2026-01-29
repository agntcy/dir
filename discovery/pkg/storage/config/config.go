package config

import (
	"github.com/agntcy/dir/discovery/pkg/storage/crd"
	"github.com/agntcy/dir/discovery/pkg/storage/etcd"
)

type Config struct {
	// StorageType is the type of storage to use (e.g., "crd", "etcd").
	StorageType string `json:"type" mapstructure:"type"`

	// CRD configuration settings.
	CRD crd.Config `json:"crd" mapstructure:"crd"`

	// Etcd configuration settings.
	Etcd etcd.Config `json:"etcd" mapstructure:"etcd"`
}

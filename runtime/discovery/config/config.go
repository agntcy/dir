// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/agntcy/dir/runtime/discovery/resolver/a2a"
	resolver "github.com/agntcy/dir/runtime/discovery/resolver/config"
	"github.com/agntcy/dir/runtime/discovery/resolver/oasf"
	runtime "github.com/agntcy/dir/runtime/discovery/runtime/config"
	docker "github.com/agntcy/dir/runtime/discovery/runtime/docker"
	k8s "github.com/agntcy/dir/runtime/discovery/runtime/k8s"
	store "github.com/agntcy/dir/runtime/store/config"
	"github.com/agntcy/dir/runtime/store/crd"
	"github.com/agntcy/dir/runtime/store/etcd"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// Config params.
	DefaultEnvPrefix  = "DISCOVERY"
	DefaultConfigName = "discovery.config"
	DefaultConfigType = "yml"
	DefaultConfigPath = "/etc/agntcy/discovery"

	// Default workers for processing tasks.
	DefaultWorkers = 4

	// Default runtime type.
	DefaultRuntimeType = docker.RuntimeType

	// Default store type.
	DefaultStoreType = etcd.StoreType
)

// Config holds configuration for the discovery component.
// This includes runtime watching, storage writing, and resolver settings.
type Config struct {
	// Worker count for processing tasks.
	Workers int `json:"workers" mapstructure:"workers"`

	// Store config for writing discovered workloads.
	Store store.Config `json:"store" mapstructure:"store"`

	// Runtime config for watching containers/pods.
	Runtime runtime.Config `json:"runtime" mapstructure:"runtime"`

	// Resolver configuration for inspecting workloads.
	Resolver resolver.Config `json:"resolver" mapstructure:"resolver"`
}

// LoadConfig loads configuration for the discovery component.
// Environment variables are prefixed with DISCOVERY_ and use underscore as separator.
// For example: DISCOVERY_RUNTIME_TYPE, DISCOVERY_STORE_TYPE.
func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetConfigName(DefaultConfigName)
	v.SetConfigType(DefaultConfigType)
	v.AddConfigPath(DefaultConfigPath)
	v.AddConfigPath(".")
	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		fileNotFoundError := viper.ConfigFileNotFoundError{}
		if errors.As(err, &fileNotFoundError) {
			log.Println("Config file not found, using defaults and environment variables.")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// General configuration
	//
	v.SetDefault("workers", DefaultWorkers)

	//
	// Store configuration
	//
	v.SetDefault("store.type", DefaultStoreType)

	//
	// ETCD store
	//
	v.SetDefault("store.etcd.host", etcd.DefaultHost)
	v.SetDefault("store.etcd.port", etcd.DefaultPort)
	v.SetDefault("store.etcd.username", "")
	v.SetDefault("store.etcd.password", "")
	v.SetDefault("store.etcd.dial_timeout", etcd.DefaultDialTimeout)
	v.SetDefault("store.etcd.workloads_prefix", etcd.DefaultWorkloadsPrefix)

	//
	// CRD store
	//
	v.SetDefault("store.crd.namespace", crd.DefaultNamespace)
	v.SetDefault("store.crd.kubeconfig", "")
	v.SetDefault("store.crd.resync_period", crd.DefaultResyncPeriod)

	//
	// Runtime configuration
	//
	v.SetDefault("runtime.type", DefaultRuntimeType)

	//
	// Docker configuration
	//
	v.SetDefault("runtime.docker.host", docker.DefaultHost)
	v.SetDefault("runtime.docker.label_key", docker.DefaultLabelKey)
	v.SetDefault("runtime.docker.label_value", docker.DefaultLabelValue)

	//
	// Kubernetes configuration
	//
	v.SetDefault("runtime.kubernetes.kubeconfig", "")
	v.SetDefault("runtime.kubernetes.namespace", k8s.DefaultNamespace)
	v.SetDefault("runtime.kubernetes.label_key", k8s.DefaultLabelKey)
	v.SetDefault("runtime.kubernetes.label_value", k8s.DefaultLabelValue)

	//
	// Resolver configuration
	//

	//
	// A2A resolver
	//
	v.SetDefault("resolver.a2a.enabled", false)
	v.SetDefault("resolver.a2a.timeout", a2a.DefaultTimeout)
	v.SetDefault("resolver.a2a.paths", a2a.DefaultDiscoveryPaths)
	v.SetDefault("resolver.a2a.label_key", a2a.DefaultLabelKey)
	v.SetDefault("resolver.a2a.label_value", a2a.DefaultLabelValue)

	//
	// OASF resolver
	//
	v.SetDefault("resolver.oasf.enabled", false)
	v.SetDefault("resolver.oasf.timeout", oasf.DefaultTimeout)
	v.SetDefault("resolver.oasf.label_key", oasf.DefaultLabelKey)

	// Load configuration into struct
	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	config := &Config{}
	if err := v.Unmarshal(config, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}

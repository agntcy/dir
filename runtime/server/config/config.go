// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	store "github.com/agntcy/dir/runtime/store/config"
	"github.com/agntcy/dir/runtime/store/crd"
	"github.com/agntcy/dir/runtime/store/etcd"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// Config params.
	DefaultEnvPrefix  = "SERVER"
	DefaultConfigName = "server.config"
	DefaultConfigType = "yml"
	DefaultConfigPath = "/etc/agntcy/server"

	// Server configuration defaults.
	DefaultServerHost = "0.0.0.0"
	DefaultServerPort = 8080

	// Default store type.
	DefaultStoreType = etcd.StoreType
)

// Config holds the server configuration.
type Config struct {
	// Host is the server bind address.
	Host string `json:"host" mapstructure:"host"`

	// Port is the server listen port.
	Port int `json:"port" mapstructure:"port"`

	// Storage holds the storage backend configuration.
	Store store.Config `json:"store" mapstructure:"store"`
}

// Addr returns the server address string.
func (c *Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// LoadConfig loads configuration for the server component.
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
	v.SetDefault("host", DefaultServerHost)
	v.SetDefault("port", DefaultServerPort)

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

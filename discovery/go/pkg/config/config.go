// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration loading from environment variables and config files.
package config

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	processor "github.com/agntcy/dir/discovery/pkg/processor/config"
	runtime "github.com/agntcy/dir/discovery/pkg/runtime/config"
	"github.com/agntcy/dir/discovery/pkg/storage"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// Config params.
	DefaultEnvPrefix  = "DISCOVERY"
	DefaultConfigName = "discovery.config"
	DefaultConfigType = "yml"
	DefaultConfigPath = "/etc/agntcy/discovery"

	// Etcd configuration defaults.
	DefaultEtcdHost            = "localhost"
	DefaultEtcdPort            = 2379
	DefaultEtcdDialTimeout     = 5 * time.Second
	DefaultEtcdWorkloadsPrefix = "/discovery/workloads/"
	DefaultEtcdMetadataPrefix  = "/discovery/metadata/"

	// Runtime configuration defaults.
	DefaultRuntimeType = "docker"

	// Docker configuration defaults.
	DefaultDockerHost       = "unix:///var/run/docker.sock"
	DefaultDockerLabelKey   = "discover"
	DefaultDockerLabelValue = "true"

	// Kubernetes configuration defaults.
	DefaultKubernetesInCluster     = false
	DefaultKubernetesLabelKey      = "discover"
	DefaultKubernetesLabelValue    = "true"
	DefaultKubernetesWatchServices = true

	// Server configuration defaults.
	DefaultServerHost = "0.0.0.0"
	DefaultServerPort = 8080

	// Processor configuration defaults.
	DefaultHealthEnabled    = true
	DefaultHealthTimeout    = 5 * time.Second
	DefaultOpenAPIEnabled   = true
	DefaultOpenAPITimeout   = 10 * time.Second
	DefaultProcessorWorkers = 4
)

// Config holds all discovery service configuration.
type Config struct {
	// Server HTTP API configuration.
	Server ServerConfig `json:"server" mapstructure:"server"`

	// Storage config
	Storage storage.Config `json:"storage" mapstructure:"storage"`

	// Runtime config
	Runtime runtime.Config `json:"runtime" mapstructure:"runtime"`

	// Processor configuration for inspector.
	Processor processor.Config `json:"processor" mapstructure:"processor"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	// Host is the server bind address.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// Port is the server listen port.
	Port int `json:"port,omitempty" mapstructure:"port"`
}

// Addr returns the server address as host:port.
func (c *ServerConfig) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// LoadConfig loads configuration from file and environment variables.
// Environment variables are prefixed with DISCOVERY_ and use underscore as separator.
// For example: DISCOVERY_ETCD_HOST, DISCOVERY_DOCKER_FILTER_LABEL
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
	// Storage configuration
	//
	v.SetDefault("storage.host", DefaultEtcdHost)
	v.SetDefault("storage.port", DefaultEtcdPort)
	v.SetDefault("storage.username", "")
	v.SetDefault("storage.password", "")
	v.SetDefault("storage.dial_timeout", DefaultEtcdDialTimeout)
	v.SetDefault("storage.workloads_prefix", DefaultEtcdWorkloadsPrefix)
	v.SetDefault("storage.metadata_prefix", DefaultEtcdMetadataPrefix)

	//
	// Runtime configuration
	//
	v.SetDefault("runtime.type", DefaultRuntimeType)

	//
	// Docker configuration
	//
	v.SetDefault("runtime.docker.host", DefaultDockerHost)
	v.SetDefault("runtime.docker.label_key", DefaultDockerLabelKey)
	v.SetDefault("runtime.docker.label_value", DefaultDockerLabelValue)

	//
	// Kubernetes configuration
	//
	v.SetDefault("runtime.kubernetes.kubeconfig", "")
	v.SetDefault("runtime.kubernetes.namespace", "")
	v.SetDefault("runtime.kubernetes.in_cluster", DefaultKubernetesInCluster)
	v.SetDefault("runtime.kubernetes.label_key", DefaultKubernetesLabelKey)
	v.SetDefault("runtime.kubernetes.label_value", DefaultKubernetesLabelValue)
	v.SetDefault("runtime.kubernetes.watch_services", DefaultKubernetesWatchServices)

	//
	// Server configuration
	//
	v.SetDefault("server.host", DefaultServerHost)
	v.SetDefault("server.port", DefaultServerPort)

	//
	// Processor configuration
	//
	v.SetDefault("processor.workers", DefaultProcessorWorkers)
	v.SetDefault("processor.health.enabled", DefaultHealthEnabled)
	v.SetDefault("processor.health.timeout", DefaultHealthTimeout)
	v.SetDefault("processor.health.paths", []string{"/health", "/healthz", "/"})
	v.SetDefault("processor.openapi.enabled", DefaultOpenAPIEnabled)
	v.SetDefault("processor.openapi.timeout", DefaultOpenAPITimeout)
	v.SetDefault("processor.openapi.paths", []string{"/openapi.json", "/swagger.json", "/api-docs"})

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

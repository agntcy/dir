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
	DefaultRuntime = "docker"

	// Docker configuration defaults.
	DefaultDockerHost        = "unix:///var/run/docker.sock"
	DefaultDockerFilterLabel = "discover=true"

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
	// Runtime selects the container runtime adapter (docker, kubernetes).
	Runtime string `json:"runtime,omitempty" mapstructure:"runtime"`

	// Etcd configuration for storage.
	Etcd EtcdConfig `json:"etcd" mapstructure:"etcd"`

	// Docker runtime configuration.
	Docker DockerConfig `json:"docker" mapstructure:"docker"`

	// Kubernetes runtime configuration.
	Kubernetes KubernetesConfig `json:"kubernetes" mapstructure:"kubernetes"`

	// Server HTTP API configuration.
	Server ServerConfig `json:"server" mapstructure:"server"`

	// Processor configuration for inspector.
	Processor ProcessorConfig `json:"processor" mapstructure:"processor"`
}

// EtcdConfig holds etcd connection configuration.
type EtcdConfig struct {
	// Host is the etcd server hostname.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// Port is the etcd server port.
	Port int `json:"port,omitempty" mapstructure:"port"`

	// Username for etcd authentication.
	Username string `json:"username,omitempty" mapstructure:"username"`

	// Password for etcd authentication.
	Password string `json:"password,omitempty" mapstructure:"password"`

	// DialTimeout is the timeout for connecting to etcd.
	DialTimeout time.Duration `json:"dial_timeout,omitempty" mapstructure:"dial_timeout"`

	// WorkloadsPrefix is the etcd key prefix for workloads.
	WorkloadsPrefix string `json:"workloads_prefix,omitempty" mapstructure:"workloads_prefix"`

	// MetadataPrefix is the etcd key prefix for metadata.
	MetadataPrefix string `json:"metadata_prefix,omitempty" mapstructure:"metadata_prefix"`
}

// Endpoints returns etcd endpoints as a slice.
func (c *EtcdConfig) Endpoints() []string {
	return []string{c.Host + ":" + strconv.Itoa(c.Port)}
}

// DockerConfig holds Docker runtime configuration.
type DockerConfig struct {
	// Host is the Docker daemon socket path.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// FilterLabel is the label filter for discoverable containers (e.g., "discover=true").
	FilterLabel string `json:"filter_label,omitempty" mapstructure:"filter_label"`
}

// LabelKey returns the label key from FilterLabel.
func (c *DockerConfig) LabelKey() string {
	parts := strings.SplitN(c.FilterLabel, "=", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return "discover"
}

// LabelValue returns the label value from FilterLabel.
func (c *DockerConfig) LabelValue() string {
	parts := strings.SplitN(c.FilterLabel, "=", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return "true"
}

// KubernetesConfig holds Kubernetes runtime configuration.
type KubernetesConfig struct {
	// Kubeconfig is the path to kubeconfig file (empty for in-cluster).
	Kubeconfig string `json:"kubeconfig,omitempty" mapstructure:"kubeconfig"`

	// Namespace to watch (empty for all namespaces).
	Namespace string `json:"namespace,omitempty" mapstructure:"namespace"`

	// InCluster uses in-cluster Kubernetes configuration.
	InCluster bool `json:"in_cluster,omitempty" mapstructure:"in_cluster"`

	// LabelKey is the label key for discoverable pods.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the label value for discoverable pods.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`

	// WatchServices enables watching Kubernetes services.
	WatchServices bool `json:"watch_services,omitempty" mapstructure:"watch_services"`
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

// ProcessorConfig holds inspector processor configuration.
type ProcessorConfig struct {
	// Workers is the number of worker goroutines.
	Workers int `json:"workers,omitempty" mapstructure:"workers"`

	// Health processor configuration.
	Health HealthProcessorConfig `json:"health" mapstructure:"health"`

	// OpenAPI processor configuration.
	OpenAPI OpenAPIProcessorConfig `json:"openapi" mapstructure:"openapi"`
}

// HealthProcessorConfig holds health check processor configuration.
type HealthProcessorConfig struct {
	// Enabled enables the health check processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the health check timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// Paths is the list of health check paths to probe.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
}

// OpenAPIProcessorConfig holds OpenAPI discovery processor configuration.
type OpenAPIProcessorConfig struct {
	// Enabled enables the OpenAPI discovery processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the OpenAPI fetch timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// Paths is the list of OpenAPI spec paths to check.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
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
	// Runtime configuration
	//
	v.SetDefault("runtime", DefaultRuntime)

	//
	// Etcd configuration
	//
	v.SetDefault("etcd.host", DefaultEtcdHost)
	v.SetDefault("etcd.port", DefaultEtcdPort)
	v.SetDefault("etcd.username", "")
	v.SetDefault("etcd.password", "")
	v.SetDefault("etcd.dial_timeout", DefaultEtcdDialTimeout)
	v.SetDefault("etcd.workloads_prefix", DefaultEtcdWorkloadsPrefix)
	v.SetDefault("etcd.metadata_prefix", DefaultEtcdMetadataPrefix)

	//
	// Docker configuration
	//
	v.SetDefault("docker.host", DefaultDockerHost)
	v.SetDefault("docker.filter_label", DefaultDockerFilterLabel)

	//
	// Kubernetes configuration
	//
	v.SetDefault("kubernetes.kubeconfig", "")
	v.SetDefault("kubernetes.namespace", "")
	v.SetDefault("kubernetes.in_cluster", DefaultKubernetesInCluster)
	v.SetDefault("kubernetes.label_key", DefaultKubernetesLabelKey)
	v.SetDefault("kubernetes.label_value", DefaultKubernetesLabelValue)
	v.SetDefault("kubernetes.watch_services", DefaultKubernetesWatchServices)

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

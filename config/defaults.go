// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	// DefaultEnvPrefix is the environment variable prefix for all dir processes.
	// e.g. DIRECTORY_REGISTRY_REGISTRY_ADDRESS overrides registry.registry_address.
	DefaultEnvPrefix = "DIRECTORY"

	// DefaultConfigName is the config file name (without extension).
	DefaultConfigName = "dir.config"

	// DefaultConfigType is the file extension viper looks for when searching by name.
	DefaultConfigType = "yml"

	// DefaultConfigPath is the directory viper searches for dir.config.yml.
	DefaultConfigPath = "/etc/agntcy/dir"

	// DefaultReconcilerEnvPrefix is the environment variable prefix for the
	// standalone reconciler process.
	DefaultReconcilerEnvPrefix = "RECONCILER"

	// DefaultReconcilerConfigName is the standalone reconciler config file name.
	DefaultReconcilerConfigName = "reconciler.config"

	// DefaultReconcilerConfigPath is the directory viper searches for reconciler.config.yml.
	DefaultReconcilerConfigPath = "/etc/agntcy/reconciler"
)

// gRPC connection management defaults.
const (
	DefaultListenAddress = "0.0.0.0:8888"

	DefaultMaxConcurrentStreams = 1000
	DefaultMaxRecvMsgSize       = 4 * 1024 * 1024
	DefaultMaxSendMsgSize       = 4 * 1024 * 1024
	DefaultConnectionTimeout    = 120 * time.Second

	DefaultMaxConnectionIdle     = 15 * time.Minute
	DefaultMaxConnectionAge      = 30 * time.Minute
	DefaultMaxConnectionAgeGrace = 5 * time.Minute
	DefaultKeepaliveTime         = 5 * time.Minute
	DefaultKeepaliveTimeout      = 1 * time.Minute
	DefaultMinTime               = 1 * time.Minute
	DefaultPermitWithoutStream   = true

	DefaultMetricsEnabled = true
	DefaultMetricsAddress = ":9090"
)

// Publication scheduler defaults.
const (
	DefaultPublicationSchedulerInterval = 1 * time.Hour
	DefaultPublicationWorkerCount       = 1
	DefaultPublicationWorkerTimeout     = 30 * time.Minute
)

// Event bus defaults.
const (
	DefaultEventsSubscriberBufferSize = 100
	DefaultEventsLogSlowConsumers     = true
	DefaultEventsLogPublishedEvents   = false
)

// HTTP gateway defaults.
const (
	DefaultHTTPGatewayEnabled      = false
	DefaultHTTPGatewayAddress      = ":8889"
	DefaultHTTPGatewayPublicURL    = "http://localhost:8889"
	DefaultHTTPGatewayCatalogTitle = "AI Catalog"
)

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

// Defaults for LoadConfig. Both DefaultEnvPrefix and DefaultConfigName
// are intentionally unified across apiserver, reconciler, and daemon:
// one file, one prefix.
const (
	// DefaultEnvPrefix is the environment variable prefix used by
	// every dir process. e.g. DIRECTORY_REGISTRY_REGISTRY_ADDRESS.
	DefaultEnvPrefix = "DIRECTORY"

	// DefaultConfigName is the on-disk config file name (no extension).
	DefaultConfigName = "dir.config"

	// DefaultConfigType is the on-disk config file type when discovered
	// by name (yaml, json, toml supported by viper).
	DefaultConfigType = "yml"

	// DefaultConfigPath is the default lookup directory for the config
	// file. Helm charts and docker images install dir.config.yml here.
	DefaultConfigPath = "/etc/agntcy/dir"
)

// API server connection management defaults; see Connection.
const (
	// DefaultListenAddress is the default gRPC listen address.
	DefaultListenAddress = "0.0.0.0:8888"

	// DefaultMaxConcurrentStreams limits concurrent RPC streams per connection (1000).
	DefaultMaxConcurrentStreams = 1000

	// DefaultMaxRecvMsgSize limits maximum received message size (4 MB).
	DefaultMaxRecvMsgSize = 4 * 1024 * 1024

	// DefaultMaxSendMsgSize limits maximum sent message size (4 MB).
	DefaultMaxSendMsgSize = 4 * 1024 * 1024

	// DefaultConnectionTimeout limits time for connection establishment (2 minutes).
	DefaultConnectionTimeout = 120 * time.Second

	// DefaultMaxConnectionIdle closes idle connections after this duration (15 minutes).
	DefaultMaxConnectionIdle = 15 * time.Minute

	// DefaultMaxConnectionAge forces connection rotation after this duration (30 minutes).
	DefaultMaxConnectionAge = 30 * time.Minute

	// DefaultMaxConnectionAgeGrace is grace period after MaxConnectionAge (5 minutes).
	DefaultMaxConnectionAgeGrace = 5 * time.Minute

	// DefaultKeepaliveTime is interval for sending keepalive pings (5 minutes).
	DefaultKeepaliveTime = 5 * time.Minute

	// DefaultKeepaliveTimeout is wait time for keepalive ping response (1 minute).
	DefaultKeepaliveTimeout = 1 * time.Minute

	// DefaultMinTime is minimum time between client keepalive pings (1 minute).
	DefaultMinTime = 1 * time.Minute

	// DefaultPermitWithoutStream allows keepalive pings without active streams.
	DefaultPermitWithoutStream = true

	// DefaultMetricsEnabled enables Prometheus metrics collection.
	DefaultMetricsEnabled = true

	// DefaultMetricsAddress is the default listen address for the metrics HTTP server.
	DefaultMetricsAddress = ":9090"
)

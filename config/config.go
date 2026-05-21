// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"

	"github.com/agntcy/dir/config/auth"
	"github.com/agntcy/dir/config/naming"
	"github.com/agntcy/dir/config/reconciler"
)

// Config is the single, canonical configuration for every dir process:
// the standalone apiserver, the standalone reconciler, and the
// in-process daemon. It deliberately hoists shared infrastructure
// (Registry, Database, OASF validation, Logging) to the top level so
// that both services point at the exact same data plane in a
// deployment, and so that Helm charts, docker envs, and operators
// configure these values only once.
type Config struct {
	// Registry is the OCI registry that backs the directory data plane.
	// Both the apiserver (writer/reader) and the reconciler (reader,
	// regsync writer) talk to the same registry, so it lives at the top.
	Registry Registry `json:"registry" mapstructure:"registry"`

	// Database is the state store (catalogue, sync state, etc.) shared
	// between the apiserver and the reconciler.
	Database Database `json:"database" mapstructure:"database"`

	// OASFAPIValidation holds the OASF schema URL used by both the
	// apiserver (request-time validation) and the reconciler
	// (record-time validation).
	OASFAPIValidation OASFAPIValidation `json:"oasf_api_validation" mapstructure:"oasf_api_validation"`

	// Logging is the process-wide logging configuration.
	Logging Logging `json:"logging" mapstructure:"logging"`

	// APIServer holds settings specific to the apiserver (gRPC frontend).
	APIServer APIServer `json:"apiserver" mapstructure:"apiserver"`

	// Reconciler holds settings specific to the reconciler service.
	Reconciler Reconciler `json:"reconciler" mapstructure:"reconciler"`
}

// APIServer holds settings that only apply to the gRPC apiserver
// process. Shared infrastructure (Registry, Database, OASF schema URL,
// Logging) is intentionally NOT here; it lives on the top-level Config.
type APIServer struct {
	// ListenAddress is the gRPC listen address (host:port).
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// Connection is the gRPC connection management configuration.
	Connection Connection `json:"connection" mapstructure:"connection"`

	// Metrics is the Prometheus metrics endpoint configuration.
	Metrics Metrics `json:"metrics" mapstructure:"metrics"`

	// RateLimit configures the gRPC rate-limiting middleware.
	RateLimit RateLimit `json:"ratelimit" mapstructure:"ratelimit"`

	// Authn configures request authentication (JWT or X.509).
	Authn auth.Authn `json:"authn" mapstructure:"authn"`

	// Authz configures authorization policies.
	Authz auth.Authz `json:"authz" mapstructure:"authz"`

	// Routing configures the libp2p routing layer.
	Routing Routing `json:"routing" mapstructure:"routing"`

	// Sync configures the apiserver's outbound sync auth (used when
	// this node pulls from another directory).
	Sync Sync `json:"sync" mapstructure:"sync"`

	// Publication configures the publication scheduler.
	Publication Publication `json:"publication" mapstructure:"publication"`

	// Events configures the in-process event bus.
	Events Events `json:"events" mapstructure:"events"`

	// Naming configures the naming verification cache used by the
	// naming API. The reconciler "name" task performs re-verification.
	Naming naming.Naming `json:"naming,omitzero" mapstructure:"naming"`
}

// Reconciler holds settings that only apply to the reconciler.
// Shared infrastructure (Registry, Database, OASF schema URL,
// Logging) is intentionally NOT here; it lives on the top-level Config.
type Reconciler struct {
	// Regsync configures the regsync (cross-registry sync) task.
	Regsync reconciler.Regsync `json:"regsync" mapstructure:"regsync"`

	// Indexer configures the indexing task.
	Indexer reconciler.Indexer `json:"indexer" mapstructure:"indexer"`

	// Name configures the name (DNS / well-known) verification task.
	Name reconciler.Name `json:"name" mapstructure:"name"`

	// Signature configures the signature verification task.
	Signature reconciler.Signature `json:"signature" mapstructure:"signature"`
}

// Sync holds the apiserver's outbound sync configuration.
// RegistryAuth is the OCI auth used when this node syncs FROM another
// directory's registry.
type Sync struct {
	RegistryAuth RegistryAuth `json:"auth_config" mapstructure:"auth_config"`
}

// OASFAPIValidation defines OASF API validation configuration. SchemaURL
// is required; the default lives in the Helm chart values.yaml.
type OASFAPIValidation struct {
	// SchemaURL is the OASF schema URL for API-based validation.
	SchemaURL string `json:"schema_url,omitempty" mapstructure:"schema_url"`
}

// Logging defines process-wide logging configuration.
type Logging struct {
	// Verbose enables verbose logging mode (includes gRPC
	// request/response payloads on the apiserver).
	Verbose bool `json:"verbose,omitempty" mapstructure:"verbose"`
}

// Connection defines gRPC connection management for the apiserver.
// These settings control connection lifecycle, resource limits, and
// keepalive behavior to prevent resource exhaustion and detect dead
// connections.
type Connection struct {
	// MaxConcurrentStreams limits concurrent RPCs per connection.
	MaxConcurrentStreams uint32 `json:"max_concurrent_streams,omitempty" mapstructure:"max_concurrent_streams"`

	// MaxRecvMsgSize limits the maximum message size the server can
	// receive (in bytes). Default: 4 MB.
	MaxRecvMsgSize int `json:"max_recv_msg_size,omitempty" mapstructure:"max_recv_msg_size"`

	// MaxSendMsgSize limits the maximum message size the server can
	// send (in bytes). Default: 4 MB.
	MaxSendMsgSize int `json:"max_send_msg_size,omitempty" mapstructure:"max_send_msg_size"`

	// ConnectionTimeout limits the time for connection establishment.
	ConnectionTimeout time.Duration `json:"connection_timeout,omitempty" mapstructure:"connection_timeout"`

	// Keepalive holds keepalive parameters for connection health.
	Keepalive Keepalive `json:"keepalive" mapstructure:"keepalive"`
}

// Keepalive defines gRPC keepalive parameters.
type Keepalive struct {
	MaxConnectionIdle     time.Duration `json:"max_connection_idle,omitempty"      mapstructure:"max_connection_idle"`
	MaxConnectionAge      time.Duration `json:"max_connection_age,omitempty"       mapstructure:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `json:"max_connection_age_grace,omitempty" mapstructure:"max_connection_age_grace"`
	Time                  time.Duration `json:"time,omitempty"                     mapstructure:"time"`
	Timeout               time.Duration `json:"timeout,omitempty"                  mapstructure:"timeout"`
	MinTime               time.Duration `json:"min_time,omitempty"                 mapstructure:"min_time"`
	PermitWithoutStream   bool          `json:"permit_without_stream,omitempty"    mapstructure:"permit_without_stream"`
}

// Metrics holds Prometheus metrics configuration.
type Metrics struct {
	// Enabled toggles Prometheus metrics collection.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Address is the HTTP listen address for the metrics endpoint
	// (separate from the gRPC port). Default: ":9090".
	Address string `json:"address,omitempty" mapstructure:"address"`
}

// DefaultConnection returns connection configuration with production
// defaults: 1000 concurrent streams, 4 MB messages, 2-minute connection
// timeout, 15-min idle / 30-min max-age, 5-min keepalive ping.
func DefaultConnection() Connection {
	return Connection{
		MaxConcurrentStreams: DefaultMaxConcurrentStreams,
		MaxRecvMsgSize:       DefaultMaxRecvMsgSize,
		MaxSendMsgSize:       DefaultMaxSendMsgSize,
		ConnectionTimeout:    DefaultConnectionTimeout,
		Keepalive: Keepalive{
			MaxConnectionIdle:     DefaultMaxConnectionIdle,
			MaxConnectionAge:      DefaultMaxConnectionAge,
			MaxConnectionAgeGrace: DefaultMaxConnectionAgeGrace,
			Time:                  DefaultKeepaliveTime,
			Timeout:               DefaultKeepaliveTimeout,
			MinTime:               DefaultMinTime,
			PermitWithoutStream:   DefaultPermitWithoutStream,
		},
	}
}

// WithDefaults returns the connection configuration with defaults
// applied. If MaxConcurrentStreams is 0 (typical zero-value),
// DefaultConnection() is returned in full.
func (c Connection) WithDefaults() Connection {
	if c.MaxConcurrentStreams == 0 {
		return DefaultConnection()
	}

	return c
}

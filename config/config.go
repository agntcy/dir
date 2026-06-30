// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"strings"
	"time"

	"github.com/agntcy/dir/config/auth"
	"github.com/agntcy/dir/config/naming"
	"github.com/agntcy/dir/config/reconciler"
)

// Config is the single canonical configuration for every dir process:
// the standalone apiserver, the standalone reconciler, and the daemon.
//
// Shared infrastructure (Store, Database) lives at the top level so that
// both services always point at the same data plane without duplication.
// Service-specific settings are nested under APIServer and Reconciler.
type Config struct {
	// Store is the OCI registry shared by the apiserver and the reconciler.
	Store Registry `json:"store" mapstructure:"store"`

	// Database is the state store shared by the apiserver and the reconciler.
	Database Database `json:"database" mapstructure:"database"`

	// OASFAPIValidation holds the schema URL for OASF record validation.
	OASFAPIValidation OASFAPIValidation `json:"oasf_api_validation" mapstructure:"oasf_api_validation"`

	// Logging configures process-wide logging.
	Logging Logging `json:"logging" mapstructure:"logging"`

	// APIServer holds settings only relevant to the apiserver process.
	APIServer APIServer `json:"server" mapstructure:"server"`

	// Reconciler holds settings only relevant to the reconciler process.
	Reconciler Reconciler `json:"reconciler" mapstructure:"reconciler"`
}

// APIServer holds settings specific to the gRPC apiserver.
// Shared infrastructure (Registry, Database) is NOT here — it lives on Config.
type APIServer struct {
	ListenAddress string        `json:"listen_address,omitempty" mapstructure:"listen_address"`
	Connection    Connection    `json:"connection"               mapstructure:"connection"`
	Metrics       Metrics       `json:"metrics"                  mapstructure:"metrics"`
	RateLimit     RateLimit     `json:"ratelimit"                mapstructure:"ratelimit"`
	Authn         auth.Authn    `json:"authn"                    mapstructure:"authn"`
	Authz         auth.Authz    `json:"authz"                    mapstructure:"authz"`
	Routing       Routing       `json:"routing"                  mapstructure:"routing"`
	Sync          Sync          `json:"sync"                     mapstructure:"sync"`
	Publication   Publication   `json:"publication"              mapstructure:"publication"`
	Events        Events        `json:"events"                   mapstructure:"events"`
	Naming        naming.Naming `json:"naming"                   mapstructure:"naming"`
	HTTPGateway   HTTPGateway   `json:"http_gateway"             mapstructure:"http_gateway"`
}

// HTTPGateway configures the in-process grpc-gateway sidecar.
type HTTPGateway struct {
	Enabled       bool   `json:"enabled,omitempty"        mapstructure:"enabled"`
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`
	PublicURL     string `json:"public_url,omitempty"     mapstructure:"public_url"`
	CatalogTitle  string `json:"catalog_title,omitempty"  mapstructure:"catalog_title"`
}

// WithDefaults returns a copy with empty fields populated from package defaults.
func (c HTTPGateway) WithDefaults() HTTPGateway {
	if c.ListenAddress == "" {
		c.ListenAddress = DefaultHTTPGatewayAddress
	}

	if c.PublicURL == "" {
		c.PublicURL = DefaultHTTPGatewayPublicURL
	}

	if strings.TrimSpace(c.CatalogTitle) == "" {
		c.CatalogTitle = DefaultHTTPGatewayCatalogTitle
	}

	return c
}

// Reconciler holds settings specific to the reconciler process.
// Shared infrastructure (Registry, Database) is NOT here — it lives on Config.
type Reconciler struct {
	Regsync   reconciler.Regsync   `json:"regsync"   mapstructure:"regsync"`
	Indexer   reconciler.Indexer   `json:"indexer"   mapstructure:"indexer"`
	Name      reconciler.Name      `json:"name"      mapstructure:"name"`
	Signature reconciler.Signature `json:"signature" mapstructure:"signature"`
	Metrics   reconciler.Metrics   `json:"metrics"   mapstructure:"metrics"`
}

// ReconcilerConfig is the full configuration for the standalone reconciler process.
// When running inside the daemon the reconciler shares the server's database and
// store directly; Database and LocalRegistry are only relevant in standalone mode.
type ReconcilerConfig struct {
	// Database holds the state store connection configuration.
	Database Database `json:"database" mapstructure:"database"`

	// LocalRegistry holds the OCI store configuration for the standalone reconciler.
	LocalRegistry Registry `json:"local_registry" mapstructure:"local_registry"`

	// ServerAddress is the gRPC address of the apiserver used by the metrics task
	// in standalone mode (the routing layer is embedded in the server process).
	// Leave empty in daemon mode — the in-process routing API is used directly.
	ServerAddress string `json:"server_address" mapstructure:"server_address"`

	// ServerAuthn holds authentication configuration for the gRPC connection to
	// the apiserver (standalone mode only).
	ServerAuthn auth.Authn `json:"server_authn" mapstructure:"server_authn"`

	// SchemaURL is the OASF schema URL for record validation.
	SchemaURL string `json:"schema_url" mapstructure:"schema_url"`

	Regsync   reconciler.Regsync   `json:"regsync"   mapstructure:"regsync"`
	Indexer   reconciler.Indexer   `json:"indexer"   mapstructure:"indexer"`
	Name      reconciler.Name      `json:"name"      mapstructure:"name"`
	Signature reconciler.Signature `json:"signature" mapstructure:"signature"`
	Metrics   reconciler.Metrics   `json:"metrics"   mapstructure:"metrics"`
}

// Sync configures the apiserver's outbound sync authentication.
type Sync struct {
	RegistryAuth RegistryAuth `json:"auth_config" mapstructure:"auth_config"`
}

// OASFAPIValidation holds the schema URL for API-based OASF record validation.
type OASFAPIValidation struct {
	SchemaURL string `json:"schema_url,omitempty" mapstructure:"schema_url"`
}

// Logging configures process-wide logging.
type Logging struct {
	Verbose bool `json:"verbose,omitempty" mapstructure:"verbose"`
}

// Connection configures gRPC connection management for the apiserver.
type Connection struct {
	MaxConcurrentStreams uint32        `json:"max_concurrent_streams,omitempty" mapstructure:"max_concurrent_streams"`
	MaxRecvMsgSize       int           `json:"max_recv_msg_size,omitempty"      mapstructure:"max_recv_msg_size"`
	MaxSendMsgSize       int           `json:"max_send_msg_size,omitempty"      mapstructure:"max_send_msg_size"`
	ConnectionTimeout    time.Duration `json:"connection_timeout,omitempty"     mapstructure:"connection_timeout"`
	Keepalive            Keepalive     `json:"keepalive"                        mapstructure:"keepalive"`
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

// Metrics configures the Prometheus metrics HTTP endpoint.
type Metrics struct {
	Enabled bool   `json:"enabled,omitempty" mapstructure:"enabled"`
	Address string `json:"address,omitempty" mapstructure:"address"`
}

// DefaultConnection returns a Connection populated with production-safe defaults.
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

// WithDefaults returns a Connection with defaults applied when MaxConcurrentStreams
// is zero (i.e. the field was not set in the config file or environment).
func (c Connection) WithDefaults() Connection {
	if c.MaxConcurrentStreams == 0 {
		return DefaultConnection()
	}

	return c
}

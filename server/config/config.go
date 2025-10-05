// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"strings"

	authn "github.com/agntcy/dir/server/authn/config"
	authz "github.com/agntcy/dir/server/authz/config"
	database "github.com/agntcy/dir/server/database/config"
	sqliteconfig "github.com/agntcy/dir/server/database/sqlite/config"
	publication "github.com/agntcy/dir/server/publication/config"
	routing "github.com/agntcy/dir/server/routing/config"
	store "github.com/agntcy/dir/server/store/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
	sync "github.com/agntcy/dir/server/sync/config"
	syncmonitor "github.com/agntcy/dir/server/sync/monitor/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// Config params.

	DefaultEnvPrefix  = "DIRECTORY_SERVER"
	DefaultConfigName = "server.config"
	DefaultConfigType = "yml"
	DefaultConfigPath = "/etc/agntcy/dir"

	// API configuration.

	DefaultListenAddress      = "0.0.0.0:8888"
	DefaultHealthCheckAddress = "0.0.0.0:8889"
)

var logger = logging.Logger("config")

type Config struct {
	// API configuration
	ListenAddress      string `json:"listen_address,omitempty"      mapstructure:"listen_address"`
	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`

	// Authn configuration (JWT authentication)
	Authn authn.Config `json:"authn,omitempty" mapstructure:"authn"`

	// Authz configuration
	Authz authz.Config `json:"authz,omitempty" mapstructure:"authz"`

	// Store configuration
	Store store.Config `json:"store,omitempty" mapstructure:"store"`

	// Routing configuration
	Routing routing.Config `json:"routing,omitempty" mapstructure:"routing"`

	// Database configuration
	Database database.Config `json:"database,omitempty" mapstructure:"database"`

	// Sync configuration
	Sync sync.Config `json:"sync,omitempty" mapstructure:"sync"`

	// Publication configuration
	Publication publication.Config `json:"publication,omitempty" mapstructure:"publication"`
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetConfigName(DefaultConfigName)
	v.SetConfigType(DefaultConfigType)
	v.AddConfigPath(DefaultConfigPath)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		fileNotFoundError := viper.ConfigFileNotFoundError{}
		if errors.As(err, &fileNotFoundError) {
			logger.Info("Config file not found, use defaults.")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// API configuration
	//
	_ = v.BindEnv("listen_address")
	v.SetDefault("listen_address", DefaultListenAddress)

	_ = v.BindEnv("healthcheck_address")
	v.SetDefault("healthcheck_address", DefaultHealthCheckAddress)

	//
	// Authn configuration (authentication: JWT or mTLS)
	//
	_ = v.BindEnv("authn.enabled")
	v.SetDefault("authn.enabled", "false")

	_ = v.BindEnv("authn.mode")
	v.SetDefault("authn.mode", "mtls")

	_ = v.BindEnv("authn.socket_path")
	v.SetDefault("authn.socket_path", "")

	_ = v.BindEnv("authn.audiences")
	v.SetDefault("authn.audiences", "")

	//
	// Authz configuration (authorization policies)
	//
	_ = v.BindEnv("authz.enabled")
	v.SetDefault("authz.enabled", "false")

	_ = v.BindEnv("authz.trust_domain")
	v.SetDefault("authz.trust_domain", "")

	//
	// Store configuration
	//
	_ = v.BindEnv("store.provider")
	v.SetDefault("store.provider", store.DefaultProvider)

	_ = v.BindEnv("store.oci.local_dir")
	v.SetDefault("store.oci.local_dir", "")

	_ = v.BindEnv("store.oci.cache_dir")
	v.SetDefault("store.oci.cache_dir", "")

	_ = v.BindEnv("store.oci.registry_address")
	v.SetDefault("store.oci.registry_address", oci.DefaultRegistryAddress)

	_ = v.BindEnv("store.oci.repository_name")
	v.SetDefault("store.oci.repository_name", oci.DefaultRepositoryName)

	_ = v.BindEnv("store.oci.auth_config.insecure")
	v.SetDefault("store.oci.auth_config.insecure", oci.DefaultAuthConfigInsecure)

	_ = v.BindEnv("store.oci.auth_config.username")
	_ = v.BindEnv("store.oci.auth_config.password")
	_ = v.BindEnv("store.oci.auth_config.access_token")
	_ = v.BindEnv("store.oci.auth_config.refresh_token")

	//
	// Routing configuration
	//
	_ = v.BindEnv("routing.listen_address")
	v.SetDefault("routing.listen_address", routing.DefaultListenAddress)

	_ = v.BindEnv("routing.directory_api_address")
	v.SetDefault("routing.directory_api_address", "")

	_ = v.BindEnv("routing.bootstrap_peers")
	v.SetDefault("routing.bootstrap_peers", strings.Join(routing.DefaultBootstrapPeers, ","))

	_ = v.BindEnv("routing.key_path")
	v.SetDefault("routing.key_path", "")

	_ = v.BindEnv("routing.datastore_dir")
	v.SetDefault("routing.datastore_dir", "")

	//
	// Routing GossipSub configuration
	// Note: Only enable/disable is configurable. Protocol parameters (topic, message size)
	// are hardcoded in server/routing/pubsub/constants.go for network compatibility.
	//
	_ = v.BindEnv("routing.gossipsub.enabled")
	v.SetDefault("routing.gossipsub.enabled", routing.DefaultGossipSubEnabled)

	//
	// Database configuration
	//
	_ = v.BindEnv("database.db_type")
	v.SetDefault("database.db_type", database.DefaultDBType)

	_ = v.BindEnv("database.sqlite.db_path")
	v.SetDefault("database.sqlite.db_path", sqliteconfig.DefaultSQLiteDBPath)

	//
	// Sync configuration
	//

	_ = v.BindEnv("sync.scheduler_interval")
	v.SetDefault("sync.scheduler_interval", sync.DefaultSyncSchedulerInterval)

	_ = v.BindEnv("sync.worker_count")
	v.SetDefault("sync.worker_count", sync.DefaultSyncWorkerCount)

	_ = v.BindEnv("sync.worker_timeout")
	v.SetDefault("sync.worker_timeout", sync.DefaultSyncWorkerTimeout)

	_ = v.BindEnv("sync.registry_monitor.check_interval")
	v.SetDefault("sync.registry_monitor.check_interval", syncmonitor.DefaultCheckInterval)

	_ = v.BindEnv("sync.auth_config.username")
	_ = v.BindEnv("sync.auth_config.password")

	//
	// Publication configuration
	//

	_ = v.BindEnv("publication.scheduler_interval")
	v.SetDefault("publication.scheduler_interval", publication.DefaultPublicationSchedulerInterval)

	_ = v.BindEnv("publication.worker_count")
	v.SetDefault("publication.worker_count", publication.DefaultPublicationWorkerCount)

	_ = v.BindEnv("publication.worker_timeout")
	v.SetDefault("publication.worker_timeout", publication.DefaultPublicationWorkerTimeout)

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

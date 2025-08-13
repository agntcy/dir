// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package zot

const (
	// DefaultZotConfigPath is the default path to the zot configuration file.
	DefaultZotConfigPath = "/etc/zot/config.json"

	// DefaultPollInterval is the default interval for polling new content.
	DefaultPollInterval = "60s"

	// DefaultRetryDelay is the default delay between retries.
	DefaultRetryDelay = "5m"

	// DefaultMaxRetries is the default maximum number of retries.
	DefaultMaxRetries = 3
)

// Zot sync config structures based on official zot documentation
// Required to avoid dependency conflicts with zot registry
// Reference: https://zotregistry.dev/v2.0.1/admin-guide/admin-configuration/
type SyncContent struct {
	Prefix      string `json:"prefix"`
	Destination string `json:"destination,omitempty"`
	Tags        *struct {
		Regex  string `json:"regex,omitempty"`
		Semver bool   `json:"semver,omitempty"`
	} `json:"tags,omitempty"`
}

type SyncRegistryConfig struct {
	URLs         []string      `json:"urls"`
	OnDemand     bool          `json:"onDemand"`
	Content      []SyncContent `json:"content,omitempty"`
	TLSVerify    *bool         `json:"tlsVerify,omitempty"`
	MaxRetries   int           `json:"maxRetries,omitempty"`
	RetryDelay   string        `json:"retryDelay,omitempty"`
	PollInterval string        `json:"pollInterval,omitempty"`
	OnlySigned   bool          `json:"onlySigned,omitempty"`
}

type CredentialsFile map[string]Credentials

type Credentials struct {
	Username string
	Password string
}

type SyncConfig struct {
	Enable          *bool                `json:"enable,omitempty"`
	CredentialsFile string               `json:"credentialsFile,omitempty"`
	Registries      []SyncRegistryConfig `json:"registries,omitempty"`
}

type TrustConfig struct {
	Enable   bool `json:"enable,omitempty"`
	Cosign   bool `json:"cosign,omitempty"`
	Notation bool `json:"notation,omitempty"`
}

type SearchConfig struct {
	Enable bool `json:"enable,omitempty"`
}

type Extensions struct {
	Search *SearchConfig `json:"search,omitempty"`
	Sync   *SyncConfig   `json:"sync,omitempty"`
	Trust  *TrustConfig  `json:"trust,omitempty"`
}

type Config struct {
	DistSpecVersion string        `json:"distSpecVersion"      mapstructure:"distSpecVersion"`
	Storage         StorageConfig `json:"storage"              mapstructure:"storage"`
	HTTP            HTTPConfig    `json:"http"                 mapstructure:"http"`
	Log             *LogConfig    `json:"log,omitempty"        mapstructure:"log"`
	Extensions      *Extensions   `json:"extensions,omitempty"`
}

type StorageConfig struct {
	RootDirectory string `json:"rootDirectory" mapstructure:"rootDirectory"`
}

type HTTPConfig struct {
	Address string `json:"address" mapstructure:"address"`
	Port    string `json:"port"    mapstructure:"port"`
}

type LogConfig struct {
	Level string `json:"level" mapstructure:"level"`
}

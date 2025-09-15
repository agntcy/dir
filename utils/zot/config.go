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
// Reference: https://zotregistry.dev/v2.1.8/admin-guide/admin-configuration/
type SyncContent struct {
	Prefix      string           `json:"Prefix"`
	Destination string           `json:"Destination,omitempty"`
	Tags        *SyncContentTags `json:"Tags,omitempty"`
	StripPrefix bool             `json:"StripPrefix,omitempty"`
}

type SyncContentTags struct {
	Regex        string  `json:"Regex,omitempty"`
	ExcludeRegex *string `json:"ExcludeRegex,omitempty"`
	Semver       *bool   `json:"Semver,omitempty"`
}

type SyncRegistryConfig struct {
	URLs             []string      `json:"URLs"`
	OnDemand         bool          `json:"OnDemand"`
	Content          []SyncContent `json:"Content,omitempty"`
	TLSVerify        *bool         `json:"TLSVerify,omitempty"`
	MaxRetries       int           `json:"MaxRetries,omitempty"`
	RetryDelay       string        `json:"RetryDelay,omitempty"`
	PollInterval     string        `json:"PollInterval,omitempty"`
	OnlySigned       *bool         `json:"OnlySigned,omitempty"`
	CertDir          string        `json:"CertDir,omitempty"`
	CredentialHelper string        `json:"CredentialHelper,omitempty"`
	PreserveDigest   bool          `json:"PreserveDigest,omitempty"`
}

type CredentialsFile map[string]Credentials

type Credentials struct {
	Username string
	Password string
}

type SyncConfig struct {
	Enable          *bool                `json:"Enable,omitempty"`
	CredentialsFile string               `json:"CredentialsFile,omitempty"`
	DownloadDir     string               `json:"DownloadDir,omitempty"`
	Registries      []SyncRegistryConfig `json:"Registries,omitempty"`
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

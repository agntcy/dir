// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package regsync

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// configFilePermissions is the permission mode for the regsync config file.
	configFilePermissions = 0o644
)

// RegsyncConfig represents the regsync configuration file format.
// See: https://github.com/regclient/regclient/blob/main/docs/regsync.md
// This struct is not thread-safe and should only be used by a single worker.
type RegsyncConfig struct {
	// Version is the config file version.
	Version int `yaml:"version"`

	// Creds holds registry credentials.
	Creds []Credential `yaml:"creds,omitempty"`

	// Defaults holds default settings for sync operations.
	Defaults *SyncDefaults `yaml:"defaults,omitempty"`

	// Sync holds the list of sync operations.
	Sync []SyncEntry `yaml:"sync,omitempty"`
}

// Credential represents registry authentication credentials.
type Credential struct {
	// Registry is the registry hostname (e.g., "ghcr.io").
	Registry string `yaml:"registry"`

	// User is the username for authentication.
	User string `yaml:"user,omitempty"`

	// Pass is the password or token for authentication.
	Pass string `yaml:"pass,omitempty"`

	// TLS controls TLS settings.
	TLS string `yaml:"tls,omitempty"`
}

// SyncDefaults holds default settings for sync operations.
type SyncDefaults struct {
	// Backup specifies backup settings.
	Backup string `yaml:"backup,omitempty"`

	// Interval is the sync interval.
	Interval string `yaml:"interval,omitempty"`

	// RateLimit controls rate limiting.
	RateLimit *RateLimit `yaml:"rateLimit,omitempty"`
}

// RateLimit controls request rate limiting.
type RateLimit struct {
	// Min is the minimum time between requests.
	Min string `yaml:"min,omitempty"`

	// Retry is the retry delay.
	Retry string `yaml:"retry,omitempty"`
}

// SyncEntry represents a single sync operation.
type SyncEntry struct {
	// Source is the source registry/repository.
	Source string `yaml:"source"`

	// Target is the target registry/repository.
	Target string `yaml:"target"`

	// Type is the sync type (e.g., "repository", "image").
	Type string `yaml:"type,omitempty"`

	// Tags filters which tags to sync.
	Tags *TagFilter `yaml:"tags,omitempty"`

	// Referrers enables syncing referrers (e.g., signatures).
	Referrers bool `yaml:"referrers,omitempty"`
}

// TagFilter defines which tags to sync.
type TagFilter struct {
	// Allow is a list of tags to include.
	Allow []string `yaml:"allow,omitempty"`

	// Deny is a list of tags to exclude.
	Deny []string `yaml:"deny,omitempty"`
}

// NewRegsyncConfig creates a new regsync configuration for a worker.
func NewRegsyncConfig() *RegsyncConfig {
	return &RegsyncConfig{
		Version: 1,
		Creds:   []Credential{},
		Defaults: &SyncDefaults{
			RateLimit: &RateLimit{
				Min:   "100ms",
				Retry: "1s",
			},
		},
		Sync: []SyncEntry{},
	}
}

// AddCredential adds or updates a credential for a registry.
func (c *RegsyncConfig) AddCredential(registry, username, password string, insecure bool) {
	// Remove scheme from registry URL
	registry = trimScheme(registry)

	// Check if credential already exists
	for i, cred := range c.Creds {
		if cred.Registry == registry {
			c.Creds[i].User = username
			c.Creds[i].Pass = password

			if insecure {
				c.Creds[i].TLS = "disabled"
			} else {
				c.Creds[i].TLS = ""
			}

			return
		}
	}

	// Add new credential
	cred := Credential{
		Registry: registry,
		User:     username,
		Pass:     password,
	}

	if insecure {
		cred.TLS = "disabled"
	}

	c.Creds = append(c.Creds, cred)
}

// AddSync adds a new sync entry or updates an existing one.
func (c *RegsyncConfig) AddSync(source, target string, tags []string) {
	// Remove scheme from URLs
	source = trimScheme(source)
	target = trimScheme(target)

	// Check if sync entry already exists
	for i, sync := range c.Sync {
		if sync.Source == source && sync.Target == target {
			// Update existing entry
			if len(tags) > 0 {
				c.Sync[i].Tags = &TagFilter{Allow: tags}
			}

			return
		}
	}

	// Create new sync entry
	entry := SyncEntry{
		Source:    source,
		Target:    target,
		Type:      "repository",
		Referrers: true, // Always enable referrers for signatures
	}

	if len(tags) > 0 {
		entry.Tags = &TagFilter{Allow: tags}
	}

	c.Sync = append(c.Sync, entry)
}

// WriteToFile writes the configuration to a file.
// The syncID is embedded in a header comment for the sidecar to parse.
func (c *RegsyncConfig) WriteToFile(syncID string) (string, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Prepend header comment with sync ID if provided
	var content []byte

	if syncID != "" {
		header := fmt.Sprintf("# sync_id: %s\n", syncID)
		content = append([]byte(header), data...)
	} else {
		content = data
	}

	// Create temporary file in the system temp directory with pattern regsync-{syncID}-*.yaml
	pattern := fmt.Sprintf("regsync-%s-*.yaml", syncID)

	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	file.Close() // Close the file as we only need the path

	path := file.Name()

	if err := os.WriteFile(path, content, configFilePermissions); err != nil {
		os.Remove(path) // Clean up on error

		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return path, nil
}

// trimScheme removes the scheme (http:// or https://) from a URL.
func trimScheme(url string) string {
	if strings.HasPrefix(url, "https://") {
		return url[8:]
	}

	if strings.HasPrefix(url, "http://") {
		return url[7:]
	}

	return url
}

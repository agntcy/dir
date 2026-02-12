// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package zot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	zotconfig "zotregistry.dev/zot/v2/pkg/api/config"
	zotextensionsconfig "zotregistry.dev/zot/v2/pkg/extensions/config"
	zotsyncconfig "zotregistry.dev/zot/v2/pkg/extensions/config/sync"
)

const (
	// DefaultZotConfigPath is the default path to the zot configuration file.
	DefaultZotConfigPath = "/etc/zot/config.json"

	// DefaultPollInterval is the default interval for polling new content.
	DefaultPollInterval = time.Second * 60

	// DefaultRetryDelay is the default delay between retries.
	DefaultRetryDelay = time.Minute * 5

	// DefaultMaxRetries is the default maximum number of retries.
	DefaultMaxRetries = 3
)

// readConfigFile reads and parses the zot configuration file.
func readConfigFile(filePath string) (*zotconfig.Config, error) {
	config, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zot config file %s: %w", filePath, err)
	}

	logger.Debug("Read zot config file", "file", string(config))

	var zotConfig zotconfig.Config
	if err := json.Unmarshal(config, &zotConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal zot config: %w", err)
	}

	return &zotConfig, nil
}

// writeConfigFile marshals and writes the zot configuration file.
func writeConfigFile(filePath string, zotConfig *zotconfig.Config) error {
	updatedConfig, err := json.MarshalIndent(zotConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated zot config: %w", err)
	}

	if err := os.WriteFile(filePath, updatedConfig, 0o644); err != nil { //nolint:gosec,mnd
		return fmt.Errorf("failed to write updated zot config: %w", err)
	}

	return nil
}

// addRegistryToSyncConfig adds a registry to the zot sync configuration.
func AddRegistryToSyncConfig(filePath string, remoteRegistryURL string, remoteRepositoryName string, credentials zotsyncconfig.Credentials, cids []string) error {
	logger.Debug("Adding registry to zot sync", "remote_url", remoteRegistryURL)

	// Validate input
	if remoteRegistryURL == "" {
		return errors.New("remote registry URL cannot be empty")
	}

	// Read current zot config
	zotConfig, err := readConfigFile(filePath)
	if err != nil {
		return err
	}

	// Initialize extensions if nil
	if zotConfig.Extensions == nil {
		zotConfig.Extensions = &zotextensionsconfig.ExtensionConfig{}
	}

	// Initialize sync config if nil
	syncConfig := zotConfig.Extensions.Sync
	if syncConfig == nil {
		syncConfig = &zotsyncconfig.Config{}
		zotConfig.Extensions.Sync = syncConfig
	}

	syncConfig.Enable = toPtr(true)

	// Create credentials file if credentials are provided
	credentialsFile := DefaultCredentialsPath
	if credentials.Username != "" && credentials.Password != "" {
		if err := updateCredentialsFile(credentialsFile, remoteRegistryURL, zotsyncconfig.Credentials{
			Username: credentials.Username,
			Password: credentials.Password,
		}); err != nil {
			return fmt.Errorf("failed to create credentials file: %w", err)
		}

		// Set credentials file path in sync config
		syncConfig.CredentialsFile = credentialsFile
	} else {
		logger.Info("No credentials provided, using default credentials file", "remote_url", remoteRegistryURL)
	}

	// Check if registry already exists
	for _, existingRegistry := range syncConfig.Registries {
		if slices.Contains(existingRegistry.URLs, remoteRegistryURL) {
			logger.Debug("Registry already exists in zot config", "registry_url", remoteRegistryURL)

			return nil
		}
	}

	var syncContent []zotsyncconfig.Content

	if len(cids) > 0 {
		// Create a regex to match the CIDs
		cidsRegex := strings.Join(cids, "|")
		regex := fmt.Sprintf("^(%s)$", cidsRegex)

		// Add the regex to the sync content
		syncContent = []zotsyncconfig.Content{
			{
				Prefix: remoteRepositoryName,
				Tags: &zotsyncconfig.Tags{
					Regex: &regex,
				},
			},
		}
	} else {
		syncContent = []zotsyncconfig.Content{
			{
				Prefix: remoteRepositoryName,
			},
		}
	}

	registry := zotsyncconfig.RegistryConfig{
		URLs:         []string{remoteRegistryURL},
		OnDemand:     false, // Disable OnDemand for proactive sync
		PollInterval: DefaultPollInterval,
		MaxRetries:   toPtr(DefaultMaxRetries),
		RetryDelay:   toPtr(DefaultRetryDelay),
		TLSVerify:    toPtr(false),
		Content:      syncContent,
	}
	syncConfig.Registries = append(syncConfig.Registries, registry)

	logger.Debug("Registry added to zot sync", "remote_url", remoteRegistryURL)

	// Write the updated config back to the file
	if err := writeConfigFile(filePath, zotConfig); err != nil {
		return err
	}

	logger.Info("Successfully added registry to zot sync", "remote_url", remoteRegistryURL)

	return nil
}

// removeRegistryFromSyncConfig removes a registry from the zot sync configuration.
func RemoveRegistryFromSyncConfig(filePath string, remoteRegistryURL string) error {
	logger.Debug("Removing registry from zot sync", "remote_registry_url", remoteRegistryURL)

	// Validate input
	if remoteRegistryURL == "" {
		return errors.New("remote directory URL cannot be empty")
	}

	// Read current zot config
	zotConfig, err := readConfigFile(filePath)
	if err != nil {
		return err
	}

	// Check if sync config exists
	if zotConfig.Extensions == nil || zotConfig.Extensions.Sync == nil {
		logger.Debug("No sync configuration found")

		return nil
	}

	syncConfig := zotConfig.Extensions.Sync

	// Find and remove the registry
	var filteredRegistries []zotsyncconfig.RegistryConfig

	for _, registry := range syncConfig.Registries {
		found := slices.Contains(registry.URLs, remoteRegistryURL)

		if !found {
			filteredRegistries = append(filteredRegistries, registry)
		}
	}

	if len(filteredRegistries) == len(syncConfig.Registries) {
		logger.Debug("Registry not found in zot config", "registry_url", remoteRegistryURL)

		return nil
	}

	syncConfig.Registries = filteredRegistries

	// Write the updated config back to the file
	if err := writeConfigFile(filePath, zotConfig); err != nil {
		return err
	}

	logger.Info("Successfully removed registry from zot sync")

	return nil
}

func toPtr[T any](v T) *T {
	return &v
}

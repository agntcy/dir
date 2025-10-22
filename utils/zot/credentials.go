// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package zot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	zotsyncconfig "zotregistry.dev/zot/v2/pkg/extensions/config/sync"
)

const (
	// DefaultCredentialsPath is the default path to the zot credentials file.
	DefaultCredentialsPath = "/etc/zot/credentials.json" //nolint:gosec
)

// updateCredentialsFile updates a credentials file for zot sync.
func updateCredentialsFile(filePath string, remoteRegistryURL string, credentials zotsyncconfig.Credentials) error {
	// Load existing credentials or create empty map
	credentialsData := make(zotsyncconfig.CredentialsFile)
	if credentialsFile, err := os.ReadFile(filePath); err == nil {
		if err := json.Unmarshal(credentialsFile, &credentialsData); err != nil {
			return fmt.Errorf("failed to unmarshal credentials file: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to read credentials file: %w", err)
	} else {
		logger.Debug("Credentials file not found, creating new one", "path", filePath)
	}

	// Normalize URL and create credentials key
	normalizedURL, err := normalizeRegistryURL(remoteRegistryURL)
	if err != nil {
		return fmt.Errorf("failed to normalize registry URL: %w", err)
	}

	credKey := strings.TrimPrefix(strings.TrimPrefix(normalizedURL, "https://"), "http://")

	// Update credentials
	credentialsData[credKey] = credentials

	// Write credentials file
	credentialsJSON, err := json.MarshalIndent(credentialsData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(filePath, credentialsJSON, 0o600); err != nil { //nolint:gosec,mnd
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	logger.Debug("Updated credentials file", "path", filePath, "registry", credKey)

	return nil
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package zot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	zotsyncconfig "zotregistry.dev/zot/v2/pkg/extensions/config/sync"
)

func TestUpdateCredentialsFile(t *testing.T) {
	t.Run("create new credentials file", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "zot-creds-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		credPath := filepath.Join(tmpDir, "credentials.json")
		testURL := "registry.example.com"
		testCreds := zotsyncconfig.Credentials{
			Username: "testuser",
			Password: "testpass",
		}

		// Test the function
		err = updateCredentialsFile(credPath, testURL, testCreds)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify file was created with correct content
		data, err := os.ReadFile(credPath)
		if err != nil {
			t.Fatalf("Failed to read credentials file: %v", err)
		}

		var credData zotsyncconfig.CredentialsFile
		if err := json.Unmarshal(data, &credData); err != nil {
			t.Fatalf("Failed to unmarshal credentials: %v", err)
		}

		// The key should be the normalized URL without protocol
		expectedKey := "registry.example.com"
		if _, exists := credData[expectedKey]; !exists {
			t.Errorf("Expected key %q not found in credentials", expectedKey)
		}

		if credData[expectedKey].Username != testCreds.Username {
			t.Errorf("Expected username %q, got %q", testCreds.Username, credData[expectedKey].Username)
		}

		if credData[expectedKey].Password != testCreds.Password {
			t.Errorf("Expected password %q, got %q", testCreds.Password, credData[expectedKey].Password)
		}

		// Check file permissions
		info, err := os.Stat(credPath)
		if err != nil {
			t.Fatalf("Failed to stat credentials file: %v", err)
		}

		expectedPerm := os.FileMode(0o600)
		if info.Mode().Perm() != expectedPerm {
			t.Errorf("Expected file permissions %v, got %v", expectedPerm, info.Mode().Perm())
		}
	})

	t.Run("update existing credentials file", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "zot-creds-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		credPath := filepath.Join(tmpDir, "credentials.json")

		// Create existing credentials file
		existingCreds := zotsyncconfig.CredentialsFile{
			"existing.registry.com": {
				Username: "existinguser",
				Password: "existingpass",
			},
		}

		existingData, err := json.MarshalIndent(existingCreds, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal existing credentials: %v", err)
		}

		if err := os.WriteFile(credPath, existingData, 0o600); err != nil {
			t.Fatalf("Failed to write existing credentials: %v", err)
		}

		// Add new credentials
		testURL := "https://new.registry.com"
		testCreds := zotsyncconfig.Credentials{
			Username: "newuser",
			Password: "newpass",
		}

		err = updateCredentialsFile(credPath, testURL, testCreds)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify both old and new credentials exist
		data, err := os.ReadFile(credPath)
		if err != nil {
			t.Fatalf("Failed to read updated credentials file: %v", err)
		}

		var credData zotsyncconfig.CredentialsFile
		if err := json.Unmarshal(data, &credData); err != nil {
			t.Fatalf("Failed to unmarshal updated credentials: %v", err)
		}

		// Check existing credentials are preserved
		if _, exists := credData["existing.registry.com"]; !exists {
			t.Errorf("Existing credentials were lost")
		}

		// Check new credentials were added (normalized URL without https://)
		expectedNewKey := "new.registry.com"
		if _, exists := credData[expectedNewKey]; !exists {
			t.Errorf("New credentials were not added with key %q", expectedNewKey)
		}

		if credData[expectedNewKey].Username != testCreds.Username {
			t.Errorf("Expected new username %q, got %q", testCreds.Username, credData[expectedNewKey].Username)
		}
	})

	t.Run("update existing registry credentials", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "zot-creds-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		credPath := filepath.Join(tmpDir, "credentials.json")

		// Create existing credentials file
		existingCreds := zotsyncconfig.CredentialsFile{
			"registry.example.com": {
				Username: "olduser",
				Password: "oldpass",
			},
		}

		existingData, err := json.MarshalIndent(existingCreds, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal existing credentials: %v", err)
		}

		if err := os.WriteFile(credPath, existingData, 0o600); err != nil {
			t.Fatalf("Failed to write existing credentials: %v", err)
		}

		// Update existing credentials
		testURL := "registry.example.com"
		testCreds := zotsyncconfig.Credentials{
			Username: "newuser",
			Password: "newpass",
		}

		err = updateCredentialsFile(credPath, testURL, testCreds)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify credentials were updated
		data, err := os.ReadFile(credPath)
		if err != nil {
			t.Fatalf("Failed to read updated credentials file: %v", err)
		}

		var credData zotsyncconfig.CredentialsFile
		if err := json.Unmarshal(data, &credData); err != nil {
			t.Fatalf("Failed to unmarshal updated credentials: %v", err)
		}

		if len(credData) != 1 {
			t.Errorf("Expected 1 credential entry, got %d", len(credData))
		}

		if credData["registry.example.com"].Username != testCreds.Username {
			t.Errorf("Expected updated username %q, got %q", testCreds.Username, credData["registry.example.com"].Username)
		}

		if credData["registry.example.com"].Password != testCreds.Password {
			t.Errorf("Expected updated password %q, got %q", testCreds.Password, credData["registry.example.com"].Password)
		}
	})

	t.Run("handle different URL formats", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "zot-creds-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testCases := []struct {
			name        string
			inputURL    string
			expectedKey string
		}{
			{
				name:        "URL without protocol",
				inputURL:    "registry.example.com",
				expectedKey: "registry.example.com",
			},
			{
				name:        "URL with http protocol",
				inputURL:    "http://registry.example.com",
				expectedKey: "registry.example.com",
			},
			{
				name:        "URL with https protocol",
				inputURL:    "https://registry.example.com",
				expectedKey: "registry.example.com",
			},
			{
				name:        "URL with port",
				inputURL:    "registry.example.com:5000",
				expectedKey: "registry.example.com:5000",
			},
			{
				name:        "URL with https and port",
				inputURL:    "https://registry.example.com:5000",
				expectedKey: "registry.example.com:5000",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				credPath := filepath.Join(tmpDir, tc.name+"_credentials.json")
				testCreds := zotsyncconfig.Credentials{
					Username: "testuser",
					Password: "testpass",
				}

				err := updateCredentialsFile(credPath, tc.inputURL, testCreds)
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.name, err)
				}

				// Verify the key is normalized correctly
				data, err := os.ReadFile(credPath)
				if err != nil {
					t.Fatalf("Failed to read credentials file for %s: %v", tc.name, err)
				}

				var credData zotsyncconfig.CredentialsFile
				if err := json.Unmarshal(data, &credData); err != nil {
					t.Fatalf("Failed to unmarshal credentials for %s: %v", tc.name, err)
				}

				if _, exists := credData[tc.expectedKey]; !exists {
					t.Errorf("Expected key %q not found for %s, got keys: %v", tc.expectedKey, tc.name, getKeys(credData))
				}
			})
		}
	})

	t.Run("invalid existing credentials file", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "zot-creds-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		credPath := filepath.Join(tmpDir, "credentials.json")

		// Write invalid JSON
		if err := os.WriteFile(credPath, []byte("invalid json"), 0o600); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		testURL := "registry.example.com"
		testCreds := zotsyncconfig.Credentials{
			Username: "testuser",
			Password: "testpass",
		}

		err = updateCredentialsFile(credPath, testURL, testCreds)
		if err == nil {
			t.Errorf("Expected error for invalid JSON file")
		}

		if !strings.Contains(err.Error(), "failed to unmarshal credentials file") {
			t.Errorf("Expected unmarshal error, got: %v", err)
		}
	})

	t.Run("invalid registry URL", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "zot-creds-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		credPath := filepath.Join(tmpDir, "credentials.json")
		testCreds := zotsyncconfig.Credentials{
			Username: "testuser",
			Password: "testpass",
		}

		err = updateCredentialsFile(credPath, "", testCreds)
		if err == nil {
			t.Errorf("Expected error for empty registry URL")
		}

		if !strings.Contains(err.Error(), "failed to normalize registry URL") {
			t.Errorf("Expected URL normalization error, got: %v", err)
		}
	})

	t.Run("write to invalid directory", func(t *testing.T) {
		invalidPath := "/invalid/directory/credentials.json"
		testCreds := zotsyncconfig.Credentials{
			Username: "testuser",
			Password: "testpass",
		}

		err := updateCredentialsFile(invalidPath, "registry.example.com", testCreds)
		if err == nil {
			t.Errorf("Expected error for invalid directory path")
		}

		if !strings.Contains(err.Error(), "failed to write credentials file") {
			t.Errorf("Expected write error, got: %v", err)
		}
	})

	t.Run("file read permission error", func(t *testing.T) {
		// This test is challenging to implement portably since it requires
		// creating a file with specific permissions that cause read errors
		// Skip this test for now as it's platform-specific
		t.Skip("Skipping file permission test - platform specific")
	})
}

// Helper function to get keys from CredentialsFile for debugging.
func getKeys(credData zotsyncconfig.CredentialsFile) []string {
	keys := make([]string, 0, len(credData))
	for k := range credData {
		keys = append(keys, k)
	}

	return keys
}

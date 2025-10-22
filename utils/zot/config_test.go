// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package zot

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	zotconfig "zotregistry.dev/zot/v2/pkg/api/config"
	zotsyncconfig "zotregistry.dev/zot/v2/pkg/extensions/config/sync"
)

func TestReadConfigFile(t *testing.T) {
	tests := []struct {
		name        string
		configData  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config file",
			configData: `{
				"http": {
					"address": "0.0.0.0",
					"port": "5000"
				},
				"storage": {
					"rootDirectory": "/var/lib/registry"
				}
			}`,
			wantErr: false,
		},
		{
			name:        "invalid JSON",
			configData:  `{"invalid": json}`,
			wantErr:     true,
			errContains: "failed to unmarshal zot config",
		},
		{
			name:        "empty file",
			configData:  "",
			wantErr:     true,
			errContains: "failed to unmarshal zot config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp(t.TempDir(), "zot-config-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write test data
			if _, err := tmpFile.WriteString(tt.configData); err != nil {
				t.Fatalf("Failed to write test data: %v", err)
			}

			tmpFile.Close()

			// Test the function
			config, err := readConfigFile(tmpFile.Name())

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if config == nil {
					t.Errorf("Expected config to be non-nil")
				}
			}
		})
	}

	t.Run("file not found", func(t *testing.T) {
		_, err := readConfigFile("/non/existent/file.json")
		if err == nil {
			t.Errorf("Expected error for non-existent file")
		}

		if !strings.Contains(err.Error(), "failed to read zot config file") {
			t.Errorf("Expected error to contain 'failed to read zot config file', got %q", err.Error())
		}
	})
}

func TestWriteConfigFile(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		// Create temporary file
		tmpFile, err := os.CreateTemp(t.TempDir(), "zot-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		// Create test config
		config := &zotconfig.Config{
			HTTP: zotconfig.HTTPConfig{
				Address: "0.0.0.0",
				Port:    "5000",
			},
		}

		// Test the function
		err = writeConfigFile(tmpFile.Name(), config)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify file contents
		data, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		var writtenConfig zotconfig.Config
		if err := json.Unmarshal(data, &writtenConfig); err != nil {
			t.Errorf("Failed to unmarshal written config: %v", err)
		}

		if writtenConfig.HTTP.Address != "0.0.0.0" || writtenConfig.HTTP.Port != "5000" {
			t.Errorf("Config not written correctly")
		}
	})

	t.Run("write to invalid path", func(t *testing.T) {
		config := &zotconfig.Config{}

		err := writeConfigFile("/invalid/path/config.json", config)
		if err == nil {
			t.Errorf("Expected error for invalid path")
		}

		if !strings.Contains(err.Error(), "failed to write updated zot config") {
			t.Errorf("Expected error to contain 'failed to write updated zot config', got %q", err.Error())
		}
	})
}

func TestNormalizeRegistryURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "URL without scheme",
			input:    "registry.example.com",
			expected: "http://registry.example.com",
			wantErr:  false,
		},
		{
			name:     "URL with http scheme",
			input:    "http://registry.example.com",
			expected: "http://registry.example.com",
			wantErr:  false,
		},
		{
			name:     "URL with https scheme",
			input:    "https://registry.example.com",
			expected: "https://registry.example.com",
			wantErr:  false,
		},
		{
			name:     "URL with port",
			input:    "registry.example.com:5000",
			expected: "http://registry.example.com:5000",
			wantErr:  false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
		},
		{
			name:     "URL with spaces (still valid after normalization)",
			input:    "registry with spaces.com",
			expected: "http://registry with spaces.com",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeRegistryURL(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestToPtr(t *testing.T) {
	t.Run("string pointer", func(t *testing.T) {
		str := "test"

		ptr := toPtr(str)
		if ptr == nil {
			t.Errorf("Expected non-nil pointer")

			return
		}

		if *ptr != str {
			t.Errorf("Expected %q, got %q", str, *ptr)
		}
	})

	t.Run("int pointer", func(t *testing.T) {
		num := 42

		ptr := toPtr(num)
		if ptr == nil {
			t.Errorf("Expected non-nil pointer")

			return
		}

		if *ptr != num {
			t.Errorf("Expected %d, got %d", num, *ptr)
		}
	})

	t.Run("bool pointer", func(t *testing.T) {
		val := true

		ptr := toPtr(val)
		if ptr == nil {
			t.Errorf("Expected non-nil pointer")

			return
		}

		if *ptr != val {
			t.Errorf("Expected %t, got %t", val, *ptr)
		}
	})
}

func TestAddRegistryToZotSync(t *testing.T) {
	// Helper function to create a basic config file
	createBasicConfig := func() string {
		tmpFile, err := os.CreateTemp(t.TempDir(), "zot-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		basicConfig := `{
			"http": {
				"address": "0.0.0.0",
				"port": "5000"
			},
			"storage": {
				"rootDirectory": "/var/lib/registry"
			}
		}`

		if _, err := tmpFile.WriteString(basicConfig); err != nil {
			t.Fatalf("Failed to write basic config: %v", err)
		}

		return tmpFile.Name()
	}

	// Helper function to create config with existing sync
	createConfigWithSync := func() string {
		tmpFile, err := os.CreateTemp(t.TempDir(), "zot-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		configWithSync := `{
			"http": {
				"address": "0.0.0.0",
				"port": "5000"
			},
			"storage": {
				"rootDirectory": "/var/lib/registry"
			},
			"extensions": {
				"sync": {
					"enable": true,
					"registries": [
						{
							"urls": ["http://existing.registry.com"],
							"onDemand": false,
							"pollInterval": 60000000000,
							"maxRetries": 3,
							"retryDelay": 300000000000,
							"tlsVerify": false,
							"content": [
								{
									"prefix": "existing/repo"
								}
							]
						}
					]
				}
			}
		}`

		if _, err := tmpFile.WriteString(configWithSync); err != nil {
			t.Fatalf("Failed to write config with sync: %v", err)
		}

		return tmpFile.Name()
	}

	t.Run("add registry to empty config", func(t *testing.T) {
		configPath := createBasicConfig()
		defer os.Remove(configPath)

		err := AddRegistryToSyncConfig(
			configPath,
			"registry.example.com",
			"test/repo",
			zotsyncconfig.Credentials{},
			nil,
		)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify the config was updated
		config, err := readConfigFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read updated config: %v", err)
		}

		if config.Extensions == nil || config.Extensions.Sync == nil {
			t.Errorf("Sync extension not initialized")
		}

		if !*config.Extensions.Sync.Enable {
			t.Errorf("Sync not enabled")
		}

		if len(config.Extensions.Sync.Registries) != 1 {
			t.Errorf("Expected 1 registry, got %d", len(config.Extensions.Sync.Registries))
		}

		registry := config.Extensions.Sync.Registries[0]
		if len(registry.URLs) != 1 || registry.URLs[0] != "http://registry.example.com" {
			t.Errorf("Registry URL not set correctly: %v", registry.URLs)
		}

		if len(registry.Content) != 1 || registry.Content[0].Prefix != "test/repo" {
			t.Errorf("Registry content not set correctly: %v", registry.Content)
		}
	})

	t.Run("add registry with credentials", func(t *testing.T) {
		configPath := createBasicConfig()
		defer os.Remove(configPath)

		// This test will fail because the credentials directory doesn't exist
		// but we can verify the error is handled properly
		err := AddRegistryToSyncConfig(
			configPath,
			"registry.example.com",
			"test/repo",
			zotsyncconfig.Credentials{
				Username: "testuser",
				Password: "testpass",
			},
			nil,
		)

		// Expect an error because /etc/zot directory doesn't exist
		if err == nil {
			t.Errorf("Expected error when credentials directory doesn't exist")
		} else if !strings.Contains(err.Error(), "failed to create credentials file") {
			t.Errorf("Expected credentials file error, got: %v", err)
		}
	})

	t.Run("add registry with CIDs", func(t *testing.T) {
		configPath := createBasicConfig()
		defer os.Remove(configPath)

		cids := []string{"cid1", "cid2", "cid3"}

		err := AddRegistryToSyncConfig(
			configPath,
			"registry.example.com",
			"test/repo",
			zotsyncconfig.Credentials{},
			cids,
		)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify the config was updated with regex
		config, err := readConfigFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read updated config: %v", err)
		}

		registry := config.Extensions.Sync.Registries[0]
		if len(registry.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(registry.Content))
		}

		content := registry.Content[0]
		if content.Tags == nil || content.Tags.Regex == nil {
			t.Errorf("Tags regex not set")
		}

		expectedRegex := "^(cid1|cid2|cid3)$"
		if *content.Tags.Regex != expectedRegex {
			t.Errorf("Expected regex %q, got %q", expectedRegex, *content.Tags.Regex)
		}
	})

	t.Run("add duplicate registry", func(t *testing.T) {
		configPath := createConfigWithSync()
		defer os.Remove(configPath)

		err := AddRegistryToSyncConfig(
			configPath,
			"existing.registry.com",
			"new/repo",
			zotsyncconfig.Credentials{},
			nil,
		)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify no duplicate was added
		config, err := readConfigFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read updated config: %v", err)
		}

		if len(config.Extensions.Sync.Registries) != 1 {
			t.Errorf("Expected 1 registry (no duplicate), got %d", len(config.Extensions.Sync.Registries))
		}
	})

	t.Run("empty registry URL", func(t *testing.T) {
		configPath := createBasicConfig()
		defer os.Remove(configPath)

		err := AddRegistryToSyncConfig(
			configPath,
			"",
			"test/repo",
			zotsyncconfig.Credentials{},
			nil,
		)

		if err == nil {
			t.Errorf("Expected error for empty registry URL")
		}

		if !strings.Contains(err.Error(), "remote registry URL cannot be empty") {
			t.Errorf("Expected error about empty URL, got %q", err.Error())
		}
	})

	t.Run("invalid config file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp(t.TempDir(), "invalid-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString("invalid json"); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		tmpFile.Close()

		err = AddRegistryToSyncConfig(
			tmpFile.Name(),
			"registry.example.com",
			"test/repo",
			zotsyncconfig.Credentials{},
			nil,
		)

		if err == nil {
			t.Errorf("Expected error for invalid config file")
		}
	})
}

func TestRemoveRegistryFromZotSync(t *testing.T) {
	// Helper function to create config with sync registries
	createConfigWithRegistries := func() string {
		tmpFile, err := os.CreateTemp(t.TempDir(), "zot-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		configWithRegistries := `{
			"http": {
				"address": "0.0.0.0",
				"port": "5000"
			},
			"storage": {
				"rootDirectory": "/var/lib/registry"
			},
			"extensions": {
				"sync": {
					"enable": true,
					"registries": [
						{
							"urls": ["http://registry1.example.com"],
							"onDemand": false,
							"content": [{"prefix": "repo1"}]
						},
						{
							"urls": ["http://registry2.example.com"],
							"onDemand": false,
							"content": [{"prefix": "repo2"}]
						}
					]
				}
			}
		}`

		if _, err := tmpFile.WriteString(configWithRegistries); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		return tmpFile.Name()
	}

	// Helper function to create basic config without sync
	createBasicConfig := func() string {
		tmpFile, err := os.CreateTemp(t.TempDir(), "zot-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		basicConfig := `{
			"http": {
				"address": "0.0.0.0",
				"port": "5000"
			},
			"storage": {
				"rootDirectory": "/var/lib/registry"
			}
		}`

		if _, err := tmpFile.WriteString(basicConfig); err != nil {
			t.Fatalf("Failed to write basic config: %v", err)
		}

		return tmpFile.Name()
	}

	t.Run("remove existing registry", func(t *testing.T) {
		configPath := createConfigWithRegistries()
		defer os.Remove(configPath)

		err := RemoveRegistryFromSyncConfig(configPath, "registry1.example.com")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify the registry was removed
		config, err := readConfigFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read updated config: %v", err)
		}

		if len(config.Extensions.Sync.Registries) != 1 {
			t.Errorf("Expected 1 registry after removal, got %d", len(config.Extensions.Sync.Registries))
		}

		// Verify the correct registry remains
		remaining := config.Extensions.Sync.Registries[0]
		if len(remaining.URLs) != 1 || remaining.URLs[0] != "http://registry2.example.com" {
			t.Errorf("Wrong registry remained: %v", remaining.URLs)
		}
	})

	t.Run("remove non-existent registry", func(t *testing.T) {
		configPath := createConfigWithRegistries()
		defer os.Remove(configPath)

		err := RemoveRegistryFromSyncConfig(configPath, "nonexistent.registry.com")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify no registries were removed
		config, err := readConfigFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read updated config: %v", err)
		}

		if len(config.Extensions.Sync.Registries) != 2 {
			t.Errorf("Expected 2 registries (no removal), got %d", len(config.Extensions.Sync.Registries))
		}
	})

	t.Run("remove from config without sync", func(t *testing.T) {
		configPath := createBasicConfig()
		defer os.Remove(configPath)

		err := RemoveRegistryFromSyncConfig(configPath, "registry.example.com")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify config remains unchanged
		config, err := readConfigFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config: %v", err)
		}

		if config.Extensions != nil && config.Extensions.Sync != nil {
			t.Errorf("Sync config should remain nil")
		}
	})

	t.Run("empty registry URL", func(t *testing.T) {
		configPath := createConfigWithRegistries()
		defer os.Remove(configPath)

		err := RemoveRegistryFromSyncConfig(configPath, "")
		if err == nil {
			t.Errorf("Expected error for empty registry URL")
		}

		if !strings.Contains(err.Error(), "remote directory URL cannot be empty") {
			t.Errorf("Expected error about empty URL, got %q", err.Error())
		}
	})

	t.Run("invalid config file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp(t.TempDir(), "invalid-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString("invalid json"); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		tmpFile.Close()

		err = RemoveRegistryFromSyncConfig(tmpFile.Name(), "registry.example.com")
		if err == nil {
			t.Errorf("Expected error for invalid config file")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		err := RemoveRegistryFromSyncConfig("/non/existent/file.json", "registry.example.com")
		if err == nil {
			t.Errorf("Expected error for non-existent file")
		}
	})
}

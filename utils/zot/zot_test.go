// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint
package zot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildRegistryURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *VerifyConfig
		expected string
	}{
		{
			name: "address without protocol, secure",
			config: &VerifyConfig{
				RegistryAddress: "registry.example.com",
				Insecure:        false,
			},
			expected: "https://registry.example.com",
		},
		{
			name: "address without protocol, insecure",
			config: &VerifyConfig{
				RegistryAddress: "registry.example.com",
				Insecure:        true,
			},
			expected: "http://registry.example.com",
		},
		{
			name: "address with https protocol",
			config: &VerifyConfig{
				RegistryAddress: "https://registry.example.com",
				Insecure:        true, // Should be ignored
			},
			expected: "https://registry.example.com",
		},
		{
			name: "address with http protocol",
			config: &VerifyConfig{
				RegistryAddress: "http://registry.example.com",
				Insecure:        false, // Should be ignored
			},
			expected: "http://registry.example.com",
		},
		{
			name: "address with port, secure",
			config: &VerifyConfig{
				RegistryAddress: "registry.example.com:5000",
				Insecure:        false,
			},
			expected: "https://registry.example.com:5000",
		},
		{
			name: "address with port, insecure",
			config: &VerifyConfig{
				RegistryAddress: "registry.example.com:5000",
				Insecure:        true,
			},
			expected: "http://registry.example.com:5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRegistryURL(tt.config.RegistryAddress, tt.config.Insecure)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAddAuthentication(t *testing.T) {
	t.Run("basic auth", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		config := &VerifyConfig{
			Username: "testuser",
			Password: "testpass",
		}

		addAuthentication(req, config)

		username, password, ok := req.BasicAuth()
		if !ok {
			t.Errorf("Expected basic auth to be set")
		}

		if username != "testuser" {
			t.Errorf("Expected username %q, got %q", "testuser", username)
		}

		if password != "testpass" {
			t.Errorf("Expected password %q, got %q", "testpass", password)
		}
	})

	t.Run("bearer token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		config := &VerifyConfig{
			AccessToken: "test-token",
		}

		addAuthentication(req, config)

		authHeader := req.Header.Get("Authorization")
		expected := "Bearer test-token"

		if authHeader != expected {
			t.Errorf("Expected authorization header %q, got %q", expected, authHeader)
		}
	})

	t.Run("basic auth takes precedence over bearer token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		config := &VerifyConfig{
			Username:    "testuser",
			Password:    "testpass",
			AccessToken: "test-token",
		}

		addAuthentication(req, config)

		username, password, ok := req.BasicAuth()
		if !ok {
			t.Errorf("Expected basic auth to be set")
		}

		if username != "testuser" || password != "testpass" {
			t.Errorf("Expected basic auth, got username=%q, password=%q", username, password)
		}

		// Bearer token should not be set when basic auth is present
		authHeader := req.Header.Get("Authorization")
		if strings.Contains(authHeader, "Bearer") {
			t.Errorf("Expected no Bearer token when basic auth is set, got %q", authHeader)
		}
	})

	t.Run("no authentication", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		config := &VerifyConfig{}

		addAuthentication(req, config)

		_, _, ok := req.BasicAuth()
		if ok {
			t.Errorf("Expected no basic auth to be set")
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader != "" {
			t.Errorf("Expected no authorization header, got %q", authHeader)
		}
	})
}

func TestUploadPublicKey(t *testing.T) {
	t.Run("successful upload", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			expectedPath := "/v2/_zot/ext/cosign"
			if r.URL.Path != expectedPath {
				t.Errorf("Expected path %q, got %q", expectedPath, r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/octet-stream" {
				t.Errorf("Expected Content-Type 'application/octet-stream', got %q", contentType)
			}

			// Verify authentication
			username, password, ok := r.BasicAuth()
			if !ok || username != "testuser" || password != "testpass" {
				t.Errorf("Expected basic auth with testuser/testpass")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Extract host from server URL
		serverURL := strings.TrimPrefix(server.URL, "http://")

		opts := &UploadPublicKeyOptions{
			Config: &VerifyConfig{
				RegistryAddress: serverURL,
				Username:        "testuser",
				Password:        "testpass",
				Insecure:        true,
			},
			PublicKey: "test-public-key",
		}

		err := UploadPublicKey(t.Context(), opts)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("empty public key", func(t *testing.T) {
		opts := &UploadPublicKeyOptions{
			Config: &VerifyConfig{
				RegistryAddress: "registry.example.com",
			},
			PublicKey: "",
		}

		err := UploadPublicKey(t.Context(), opts)
		if err == nil {
			t.Errorf("Expected error for empty public key")
		}

		if !strings.Contains(err.Error(), "public key is required") {
			t.Errorf("Expected 'public key is required' error, got %q", err.Error())
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		opts := &UploadPublicKeyOptions{
			Config: &VerifyConfig{
				RegistryAddress: serverURL,
				Insecure:        true,
			},
			PublicKey: "test-public-key",
		}

		err := UploadPublicKey(t.Context(), opts)
		if err == nil {
			t.Errorf("Expected error for server error response")
		}

		if !strings.Contains(err.Error(), "failed to upload public key") {
			t.Errorf("Expected upload error, got %q", err.Error())
		}
	})
}

func TestVerify(t *testing.T) {
	t.Run("successful verification with signature", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method and path
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got %s", r.Method)
			}

			expectedPath := "/v2/_zot/ext/search"
			if r.URL.Path != expectedPath {
				t.Errorf("Expected path %q, got %q", expectedPath, r.URL.Path)
			}

			// Verify content type
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
			}

			// Verify GraphQL query in request body
			var queryData map[string]interface{}

			err := json.NewDecoder(r.Body).Decode(&queryData)
			if err != nil {
				t.Errorf("Failed to decode request body: %v", err)
			}

			query, ok := queryData["query"].(string)
			if !ok {
				t.Errorf("Expected query field in request")
			}

			if !strings.Contains(query, "test-repo:test-cid") {
				t.Errorf("Expected query to contain 'test-repo:test-cid', got %q", query)
			}

			// Return successful response
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"Image": map[string]interface{}{
						"Digest":   "sha256:abcdef123456",
						"IsSigned": true,
						"Tag":      "test-cid",
						"SignatureInfo": []map[string]interface{}{
							{
								"Tool":      "cosign",
								"IsTrusted": true,
								"Author":    "test@example.com",
							},
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		opts := &VerificationOptions{
			Config: &VerifyConfig{
				RegistryAddress: serverURL,
				RepositoryName:  "test-repo",
				Username:        "testuser",
				Password:        "testpass",
				Insecure:        true,
			},
			RecordCID: "test-cid",
		}

		result, err := Verify(t.Context(), opts)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Fatalf("Expected result to be non-nil")
		}

		if !result.IsSigned {
			t.Errorf("Expected IsSigned to be true")
		}

		if !result.IsTrusted {
			t.Errorf("Expected IsTrusted to be true")
		}

		if result.Author != "test@example.com" {
			t.Errorf("Expected Author 'test@example.com', got %q", result.Author)
		}

		if result.Tool != "cosign" {
			t.Errorf("Expected Tool 'cosign', got %q", result.Tool)
		}
	})

	t.Run("verification without signature", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"Image": map[string]interface{}{
						"Digest":        "sha256:abcdef123456",
						"IsSigned":      false,
						"Tag":           "test-cid",
						"SignatureInfo": []map[string]interface{}{},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		opts := &VerificationOptions{
			Config: &VerifyConfig{
				RegistryAddress: serverURL,
				RepositoryName:  "test-repo",
				Insecure:        true,
			},
			RecordCID: "test-cid",
		}

		result, err := Verify(t.Context(), opts)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Fatalf("Expected result to be non-nil")
		}

		if result.IsSigned {
			t.Errorf("Expected IsSigned to be false")
		}

		if result.IsTrusted {
			t.Errorf("Expected IsTrusted to be false")
		}

		if result.Author != "" {
			t.Errorf("Expected empty Author, got %q", result.Author)
		}

		if result.Tool != "" {
			t.Errorf("Expected empty Tool, got %q", result.Tool)
		}
	})

	t.Run("server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		opts := &VerificationOptions{
			Config: &VerifyConfig{
				RegistryAddress: serverURL,
				RepositoryName:  "test-repo",
				Insecure:        true,
			},
			RecordCID: "test-cid",
		}

		result, err := Verify(t.Context(), opts)
		if err == nil {
			t.Errorf("Expected error for server error response")
		}

		if result != nil {
			t.Errorf("Expected nil result on error")
		}

		if !strings.Contains(err.Error(), "verification query returned status 500") {
			t.Errorf("Expected status error, got %q", err.Error())
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		serverURL := strings.TrimPrefix(server.URL, "http://")

		opts := &VerificationOptions{
			Config: &VerifyConfig{
				RegistryAddress: serverURL,
				RepositoryName:  "test-repo",
				Insecure:        true,
			},
			RecordCID: "test-cid",
		}

		result, err := Verify(t.Context(), opts)
		if err == nil {
			t.Errorf("Expected error for invalid JSON response")
		}

		if result != nil {
			t.Errorf("Expected nil result on error")
		}

		if !strings.Contains(err.Error(), "failed to decode verification response") {
			t.Errorf("Expected decode error, got %q", err.Error())
		}
	})
}

func TestStructs(t *testing.T) {
	t.Run("VerifyConfig", func(t *testing.T) {
		config := &VerifyConfig{
			RegistryAddress: "registry.example.com",
			RepositoryName:  "test-repo",
			Username:        "testuser",
			Password:        "testpass",
			AccessToken:     "test-token",
			Insecure:        true,
		}

		if config.RegistryAddress != "registry.example.com" {
			t.Errorf("Expected RegistryAddress 'registry.example.com', got %q", config.RegistryAddress)
		}

		if config.RepositoryName != "test-repo" {
			t.Errorf("Expected RepositoryName 'test-repo', got %q", config.RepositoryName)
		}

		if config.Username != "testuser" {
			t.Errorf("Expected Username 'testuser', got %q", config.Username)
		}

		if config.Password != "testpass" {
			t.Errorf("Expected Password 'testpass', got %q", config.Password)
		}

		if config.AccessToken != "test-token" {
			t.Errorf("Expected AccessToken 'test-token', got %q", config.AccessToken)
		}

		if !config.Insecure {
			t.Errorf("Expected Insecure to be true")
		}
	})

	t.Run("UploadPublicKeyOptions", func(t *testing.T) {
		config := &VerifyConfig{RegistryAddress: "registry.example.com"}
		opts := &UploadPublicKeyOptions{
			Config:    config,
			PublicKey: "test-key",
		}

		if opts.Config != config {
			t.Errorf("Expected Config to be set correctly")
		}

		if opts.PublicKey != "test-key" {
			t.Errorf("Expected PublicKey 'test-key', got %q", opts.PublicKey)
		}
	})

	t.Run("VerificationOptions", func(t *testing.T) {
		config := &VerifyConfig{RegistryAddress: "registry.example.com"}
		opts := &VerificationOptions{
			Config:    config,
			RecordCID: "test-cid",
		}

		if opts.Config != config {
			t.Errorf("Expected Config to be set correctly")
		}

		if opts.RecordCID != "test-cid" {
			t.Errorf("Expected RecordCID 'test-cid', got %q", opts.RecordCID)
		}
	})

	t.Run("VerificationResult", func(t *testing.T) {
		result := &VerificationResult{
			IsSigned:  true,
			IsTrusted: true,
			Author:    "test@example.com",
			Tool:      "cosign",
		}

		if !result.IsSigned {
			t.Errorf("Expected IsSigned to be true")
		}

		if !result.IsTrusted {
			t.Errorf("Expected IsTrusted to be true")
		}

		if result.Author != "test@example.com" {
			t.Errorf("Expected Author 'test@example.com', got %q", result.Author)
		}

		if result.Tool != "cosign" {
			t.Errorf("Expected Tool 'cosign', got %q", result.Tool)
		}
	})
}

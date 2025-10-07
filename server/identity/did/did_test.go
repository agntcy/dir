package did

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDIDManager_Register(t *testing.T) {
	tests := []struct {
		name          string
		resource      string
		shouldSucceed bool
		setupMockPDS  bool
	}{
		{
			name:          "Valid resource registration",
			resource:      "test-resource",
			shouldSucceed: true,
			setupMockPDS:  true,
		},
		{
			name:          "Empty resource should fail",
			resource:      "",
			shouldSucceed: false,
			setupMockPDS:  false,
		},
		{
			name:          "Special characters in resource",
			resource:      "test-resource-with-special-chars_123",
			shouldSucceed: true,
			setupMockPDS:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var manager *Manager
			
			if tt.setupMockPDS {
				// Create mock PDS server
				mockPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "createAccount") {
						w.WriteHeader(http.StatusCreated)
						w.Write([]byte(`{"did":"test-did","accessJwt":"test-token"}`))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
				defer mockPDS.Close()
				
				manager = NewManager(mockPDS.URL)
			} else {
				manager = NewManager("http://localhost:2583")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			didDoc, err := manager.Register(ctx, tt.resource)

			if tt.shouldSucceed {
				if err != nil && !tt.setupMockPDS {
					t.Logf("Expected: PDS connection error (PDS not running): %v", err)
					return
				}
				
				if err != nil {
					t.Fatalf("Expected success, got error: %v", err)
				}

				if didDoc == nil {
					t.Fatal("Expected DID document, got nil")
				}

				if didDoc.ID == "" {
					t.Fatal("Expected DID ID, got empty string")
				}

				if !strings.HasPrefix(didDoc.ID, "did:plc:") {
					t.Fatalf("Expected DID to start with 'did:plc:', got: %s", didDoc.ID)
				}

				// Verify DID document structure
				if len(didDoc.Context) == 0 {
					t.Fatal("Expected @context to be set")
				}

				if len(didDoc.VerificationMethod) == 0 {
					t.Fatal("Expected verification methods to be set")
				}

				if len(didDoc.Authentication) == 0 {
					t.Fatal("Expected authentication methods to be set")
				}

				// Verify resource can be retrieved
				retrievedResource, err := manager.GetResource(ctx, didDoc.ID)
				if err != nil {
					t.Fatalf("Failed to retrieve resource: %v", err)
				}

				if retrievedResource != tt.resource {
					t.Fatalf("Expected resource %s, got %s", tt.resource, retrievedResource)
				}

				t.Logf("Successfully registered DID: %s for resource: %s", didDoc.ID, tt.resource)
			} else {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				t.Logf("Expected error occurred: %v", err)
			}
		})
	}
}

func TestDIDManager_Validate(t *testing.T) {
	tests := []struct {
		name        string
		did         string
		expectValid bool
		expectError bool
		setupMock   bool
	}{
		{
			name:        "Empty DID should error",
			did:         "",
			expectValid: false,
			expectError: true,
			setupMock:   false,
		},
		{
			name:        "Invalid DID format",
			did:         "invalid-did-format",
			expectValid: false,
			expectError: false,
			setupMock:   false,
		},
		{
			name:        "Valid DID format but non-existent",
			did:         "did:plc:nonexistent123",
			expectValid: false,
			expectError: false,
			setupMock:   false,
		},
		{
			name:        "Valid DID with mock PDS response",
			did:         "did:plc:validtest123",
			expectValid: true,
			expectError: false,
			setupMock:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var manager *Manager

			if tt.setupMock {
				// Create mock PDS server that responds positively
				mockPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "resolveHandle") {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"did":"` + tt.did + `"}`))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
				defer mockPDS.Close()
				
				manager = NewManager(mockPDS.URL)
			} else {
				manager = NewManager("http://localhost:2583")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			valid, err := manager.Validate(ctx, tt.did)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				t.Logf("Expected error occurred: %v", err)
				return
			}

			if err != nil && !tt.setupMock {
				// Connection errors are expected when PDS is not running
				t.Logf("PDS connection error (expected if PDS not running): %v", err)
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if valid != tt.expectValid {
				t.Fatalf("Expected valid=%v, got valid=%v", tt.expectValid, valid)
			}

			t.Logf("DID validation result for %s: valid=%v", tt.did, valid)
		})
	}
}

func TestDIDManager_GetResource(t *testing.T) {
	manager := NewManager("http://localhost:2583")
	ctx := context.Background()

	// Test non-existent DID
	_, err := manager.GetResource(ctx, "did:plc:nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent DID")
	}

	// Test with manually registered resource
	testDID := "did:plc:test123"
	testResource := "test-resource"
	manager.registry[testDID] = testResource

	resource, err := manager.GetResource(ctx, testDID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resource != testResource {
		t.Fatalf("Expected resource %s, got %s", testResource, resource)
	}

	t.Logf("Successfully retrieved resource: %s for DID: %s", resource, testDID)
}

func TestDIDManager_ResolveDID(t *testing.T) {
	tests := []struct {
		name         string
		did          string
		shouldSucceed bool
		setupMock    bool
	}{
		{
			name:         "Valid DID with mock response",
			did:          "did:plc:test123",
			shouldSucceed: true,
			setupMock:    true,
		},
		{
			name:         "Valid DID with no PDS",
			did:          "did:plc:test456",
			shouldSucceed: false,
			setupMock:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var manager *Manager

			if tt.setupMock {
				// Create mock PDS server
				mockPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "resolveHandle") {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"did":"` + tt.did + `"}`))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
				defer mockPDS.Close()
				
				manager = NewManager(mockPDS.URL)
			} else {
				manager = NewManager("http://localhost:2583")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			didDoc, err := manager.ResolveDID(ctx, tt.did)

			if tt.shouldSucceed {
				if err != nil {
					t.Fatalf("Expected success, got error: %v", err)
				}

				if didDoc == nil {
					t.Fatal("Expected DID document, got nil")
				}

				if didDoc.ID != tt.did {
					t.Fatalf("Expected DID ID %s, got %s", tt.did, didDoc.ID)
				}

				t.Logf("Successfully resolved DID: %s", didDoc.ID)
			} else {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				t.Logf("Expected error occurred: %v", err)
			}
		})
	}
}

func TestDIDManager_GeneratePLCDID(t *testing.T) {
	manager := NewManager("http://localhost:2583")

	// Generate test key
	pubKey := make([]byte, 32) // Ed25519 public key size
	for i := range pubKey {
		pubKey[i] = byte(i % 256)
	}

	did1 := manager.generatePLCDID("resource1", pubKey)
	did2 := manager.generatePLCDID("resource2", pubKey)

	// DIDs should start with did:plc:
	if !strings.HasPrefix(did1, "did:plc:") {
		t.Fatalf("Expected DID to start with 'did:plc:', got: %s", did1)
	}

	if !strings.HasPrefix(did2, "did:plc:") {
		t.Fatalf("Expected DID to start with 'did:plc:', got: %s", did2)
	}

	// Different resources should generate different DIDs
	if did1 == did2 {
		t.Fatal("Expected different DIDs for different resources")
	}

	// DID should be deterministic for same input (within same timestamp/nonce)
	// Note: This implementation includes timestamp and nonce, so it won't be deterministic
	t.Logf("Generated DID 1: %s", did1)
	t.Logf("Generated DID 2: %s", did2)
}

func TestDIDManager_EncodePublicKey(t *testing.T) {
	manager := NewManager("http://localhost:2583")

	// Create test public key
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i % 256)
	}

	encoded := manager.encodePublicKey(pubKey)

	// Should start with 'z' (base58btc multibase prefix)
	if !strings.HasPrefix(encoded, "z") {
		t.Fatalf("Expected encoded key to start with 'z', got: %s", encoded)
	}

	// Should be non-empty
	if len(encoded) <= 1 {
		t.Fatalf("Expected encoded key to have content after prefix, got: %s", encoded)
	}

	t.Logf("Encoded public key: %s", encoded)
}

func TestDIDManager_Integration(t *testing.T) {
	// This test demonstrates the full workflow
	t.Run("Full workflow with mock PDS", func(t *testing.T) {
		// Create mock PDS server
		mockPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "createAccount"):
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"did":"test-did","accessJwt":"test-token"}`))
			case strings.Contains(r.URL.Path, "resolveHandle"):
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"did":"test-did"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer mockPDS.Close()

		manager := NewManager(mockPDS.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Step 1: Register a DID
		resource := "integration-test-resource"
		didDoc, err := manager.Register(ctx, resource)
		if err != nil {
			t.Fatalf("Failed to register DID: %v", err)
		}

		// Step 2: Validate the registered DID
		valid, err := manager.Validate(ctx, didDoc.ID)
		if err != nil {
			t.Fatalf("Failed to validate DID: %v", err)
		}
		if !valid {
			t.Fatal("Expected DID to be valid")
		}

		// Step 3: Retrieve the resource
		retrievedResource, err := manager.GetResource(ctx, didDoc.ID)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}
		if retrievedResource != resource {
			t.Fatalf("Expected resource %s, got %s", resource, retrievedResource)
		}

		// Step 4: Resolve the DID
		resolvedDoc, err := manager.ResolveDID(ctx, didDoc.ID)
		if err != nil {
			t.Fatalf("Failed to resolve DID: %v", err)
		}
		if resolvedDoc == nil {
			t.Fatal("Expected resolved DID document")
		}

		t.Logf("Integration test completed successfully:")
		t.Logf("  - Registered DID: %s", didDoc.ID)
		t.Logf("  - Resource: %s", resource)
		t.Logf("  - Validation: %v", valid)
	})
}

// Benchmark DID generation
func BenchmarkDIDGeneration(b *testing.B) {
	manager := NewManager("http://localhost:2583")
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.generatePLCDID("test-resource", pubKey)
	}
}

// Benchmark public key encoding
func BenchmarkPublicKeyEncoding(b *testing.B) {
	manager := NewManager("http://localhost:2583")
	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.encodePublicKey(pubKey)
	}

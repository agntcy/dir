package did

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ExampleDIDManager_Register demonstrates how to register a DID
func ExampleDIDManager_Register() {
	// Create a mock PDS server for demonstration
	mockPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "createAccount") {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"did":"test-did","accessJwt":"test-token"}`))
		}
	}))
	defer mockPDS.Close()

	// Create DID manager
	manager := NewManager(mockPDS.URL)
	ctx := context.Background()

	// Register a DID for a resource
	didDoc, err := manager.Register(ctx, "my-important-resource")
	if err != nil {
		log.Fatalf("Failed to register DID: %v", err)
	}

	fmt.Printf("Registered DID: %s\n", didDoc.ID)
	fmt.Printf("DID has %d verification methods\n", len(didDoc.VerificationMethod))

	// Output will vary due to random generation, but structure will be:
	// Registered DID: did:plc:...
	// DID has 1 verification methods
}

// ExampleDIDManager_Validate demonstrates how to validate a DID
func ExampleDIDManager_Validate() {
	// Create a mock PDS server
	mockPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "resolveHandle") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"did":"did:plc:test123"}`))
		}
	}))
	defer mockPDS.Close()

	manager := NewManager(mockPDS.URL)
	ctx := context.Background()

	// Test various DID formats
	testCases := []struct {
		did      string
		expected string
	}{
		{"", "invalid (empty)"},
		{"invalid-format", "invalid (wrong format)"},
		{"did:plc:test123", "valid"},
	}

	for _, tc := range testCases {
		valid, err := manager.Validate(ctx, tc.did)
		if err != nil {
			fmt.Printf("DID '%s': error - %v\n", tc.did, err)
		} else {
			fmt.Printf("DID '%s': %s\n", tc.did, map[bool]string{true: "valid", false: "invalid"}[valid])
		}
	}

	// Output:
	// DID '': error - DID cannot be empty
	// DID 'invalid-format': invalid
	// DID 'did:plc:test123': valid
}

// ExampleDIDManager_GetResource demonstrates resource retrieval
func ExampleDIDManager_GetResource() {
	manager := NewManager("http://localhost:2583")
	ctx := context.Background()

	// Simulate a registered DID (normally this would come from Register())
	testDID := "did:plc:abc123"
	testResource := "user-profile-data"
	manager.registry[testDID] = testResource

	// Retrieve the resource
	resource, err := manager.GetResource(ctx, testDID)
	if err != nil {
		log.Fatalf("Failed to get resource: %v", err)
	}

	fmt.Printf("Resource for DID %s: %s\n", testDID, resource)
	// Output: Resource for DID did:plc:abc123: user-profile-data
}

// ExampleDIDManager_workflow demonstrates a complete workflow
func ExampleDIDManager_workflow() {
	// Create a mock PDS server that handles all endpoints
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
	ctx := context.Background()

	// Step 1: Register a DID
	fmt.Println("1. Registering DID...")
	didDoc, err := manager.Register(ctx, "my-resource")
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}
	fmt.Printf("   Registered: %s\n", didDoc.ID)

	// Step 2: Validate the DID
	fmt.Println("2. Validating DID...")
	valid, err := manager.Validate(ctx, didDoc.ID)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	fmt.Printf("   Valid: %v\n", valid)

	// Step 3: Retrieve resource
	fmt.Println("3. Retrieving resource...")
	resource, err := manager.GetResource(ctx, didDoc.ID)
	if err != nil {
		log.Fatalf("Resource retrieval failed: %v", err)
	}
	fmt.Printf("   Resource: %s\n", resource)

	// Step 4: Resolve DID document
	fmt.Println("4. Resolving DID document...")
	resolvedDoc, err := manager.ResolveDID(ctx, didDoc.ID)
	if err != nil {
		log.Fatalf("Resolution failed: %v", err)
	}
	fmt.Printf("   Resolved DID: %s\n", resolvedDoc.ID)

	fmt.Println("Workflow completed successfully!")

	// Output:
	// 1. Registering DID...
	//    Registered: did:plc:...
	// 2. Validating DID...
	//    Valid: true
	// 3. Retrieving resource...
	//    Resource: my-resource
	// 4. Resolving DID document...
	//    Resolved DID: did:plc:...
	// Workflow completed successfully!
}

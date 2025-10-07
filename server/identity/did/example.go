package did

import (
	"context"
	"fmt"
	"log"
)

// Example demonstrates how to use the DID manager
func Example() {
	ctx := context.Background()

	// Create a new DID manager with a local PDS URL
	pdsURL := "http://localhost:2583" // Default AT Protocol PDS port
	manager := NewManager(pdsURL)

	// Register a DID for a resource
	resource := "my-container-image"
	fmt.Printf("Registering DID for resource: %s\n", resource)

	didDoc, err := manager.Register(ctx, resource)
	if err != nil {
		log.Printf("Failed to register DID: %v", err)
		return
	}

	fmt.Printf("Successfully registered DID: %s\n", didDoc.ID)

	// Validate the DID
	valid, err := manager.Validate(ctx, didDoc.ID)
	if err != nil {
		log.Printf("Failed to validate DID: %v", err)
		return
	}

	fmt.Printf("DID validation result: %t\n", valid)

	// Get the resource for the DID
	retrievedResource, err := manager.GetResource(ctx, didDoc.ID)
	if err != nil {
		log.Printf("Failed to get resource: %v", err)
		return
	}

	fmt.Printf("Retrieved resource: %s\n", retrievedResource)

	// Resolve the DID document
	resolvedDoc, err := manager.ResolveDID(ctx, didDoc.ID)
	if err != nil {
		log.Printf("Failed to resolve DID: %v", err)
		return
	}

	fmt.Printf("Resolved DID document ID: %s\n", resolvedDoc.ID)
	fmt.Printf("Service endpoints: %d\n", len(resolvedDoc.Service))
}

// ExampleWithCustomPDS shows how to use with a custom PDS
func ExampleWithCustomPDS() {
	ctx := context.Background()

	// Use a custom PDS URL (could be a production instance)
	pdsURL := "https://your-pds.example.com"
	manager := NewManager(pdsURL)

	// Register multiple DIDs for different resources
	resources := []string{
		"container-image:alpine:3.18",
		"container-image:nginx:1.25",
		"oci-artifact:helm-chart:app:v1.0.0",
	}

	for _, resource := range resources {
		didDoc, err := manager.Register(ctx, resource)
		if err != nil {
			log.Printf("Failed to register DID for %s: %v", resource, err)
			continue
		}

		fmt.Printf("Registered %s -> %s\n", resource, didDoc.ID)

		// Validate immediately after registration
		valid, err := manager.Validate(ctx, didDoc.ID)
		if err != nil {
			log.Printf("Failed to validate DID %s: %v", didDoc.ID, err)
			continue
		}

		if !valid {
			log.Printf("DID %s validation failed", didDoc.ID)
			continue
		}

		fmt.Printf("✓ DID %s is valid\n", didDoc.ID)
	}
}

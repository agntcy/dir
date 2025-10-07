# DID (Decentralized Identifier) Manager

This package provides a simple interface for registering and validating DIDs (Decentralized Identifiers) for resources using the AT Protocol. It integrates with a local Personal Data Server (PDS) instance for DID registration and resolution.

## Features

- **DID Registration**: Create and register new DIDs for string resources
- **DID Validation**: Validate existing DIDs against the PDS
- **Resource Mapping**: Associate DIDs with specific resources
- **PDS Integration**: Full integration with AT Protocol Personal Data Server
- **PLC Method**: Uses DID:PLC method compatible with AT Protocol

## Interface

```go
type DIDManager interface {
    Register(ctx context.Context, resource string) (*DIDDocument, error)
    Validate(ctx context.Context, didStr string) (bool, error)
    GetResource(ctx context.Context, didStr string) (string, error)
    ResolveDID(ctx context.Context, didStr string) (*DIDDocument, error)
}
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/agntcy/dir/server/identity/did"
)

func main() {
    ctx := context.Background()
    
    // Create a new DID manager with your PDS URL
    pdsURL := "http://localhost:2583" // Local PDS instance
    manager := did.NewManager(pdsURL)
    
    // Register a DID for a resource
    resource := "container-image:alpine:3.18"
    didDoc, err := manager.Register(ctx, resource)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Registered DID: %s\n", didDoc.ID)
    
    // Validate the DID
    valid, err := manager.Validate(ctx, didDoc.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("DID is valid: %t\n", valid)
    
    // Get the resource for a DID
    retrievedResource, err := manager.GetResource(ctx, didDoc.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Resource: %s\n", retrievedResource)
}
```

### Using with Custom PDS

```go
// Production PDS instance
manager := did.NewManager("https://bsky.social")

// Custom PDS instance
manager := did.NewManager("https://your-pds.example.com")

// Local development PDS
manager := did.NewManager("http://localhost:2583")
```

## PDS Setup

To use this DID manager, you need a running AT Protocol PDS instance. For local development:

1. **Install AT Protocol PDS**:
   ```bash
   npm install -g @atproto/pds
   ```

2. **Run PDS locally**:
   ```bash
   pds server --port 2583
   ```

3. **Or use Docker**:
   ```bash
   docker run -p 2583:2583 atproto/pds
   ```

## DID Format

This implementation generates DIDs using the PLC (Public Ledger of Credentials) method:

```
did:plc:<identifier>
```

Example: `did:plc:abc123def456`

## DID Document Structure

The generated DID documents follow W3C DID specification with AT Protocol extensions:

```json
{
  "@context": [
    "https://www.w3.org/ns/did/v1",
    "https://w3id.org/security/multikey/v1"
  ],
  "id": "did:plc:abc123def456",
  "verificationMethod": [
    {
      "id": "did:plc:abc123def456#atproto",
      "type": "Multikey",
      "controller": "did:plc:abc123def456",
      "publicKeyMultibase": "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"
    }
  ],
  "authentication": ["did:plc:abc123def456#atproto"],
  "assertionMethod": ["did:plc:abc123def456#atproto"],
  "capabilityInvocation": ["did:plc:abc123def456#atproto"],
  "service": [
    {
      "id": "did:plc:abc123def456#atproto_pds",
      "type": "AtprotoPersonalDataServer",
      "serviceEndpoint": "http://localhost:2583"
    }
  ]
}
```

## Error Handling

The interface provides detailed error information:

- **Registration errors**: Key generation, PDS communication, invalid resources
- **Validation errors**: Format validation, PDS resolution failures
- **Resource errors**: DID not found in local registry

## Security Considerations

- Uses Ed25519 cryptographic keys for DID generation
- Implements proper DID:PLC generation with timestamps and nonces
- Integrates with AT Protocol security model
- Supports multibase key encoding

## Testing

Run the test suite:

```bash
go test ./server/identity/did/...
```

Note: Some tests require a running PDS instance for full integration testing.

## Dependencies

- `golang.org/x/crypto/ed25519` - Cryptographic key generation
- `github.com/google/uuid` - Unique identifier generation
- Standard library packages for HTTP, JSON, and cryptography

## Production Considerations

1. **Persistent Storage**: Replace in-memory registry with persistent storage (database)
2. **Authentication**: Implement proper AT Protocol authentication instead of temp passwords
3. **Error Recovery**: Add retry logic for PDS communication failures
4. **Monitoring**: Add logging and metrics for DID operations
5. **Rate Limiting**: Implement rate limiting for DID registration
6. **Key Management**: Secure key storage and rotation

## Contributing

When extending this package:

1. Maintain the `DIDManager` interface compatibility
2. Add comprehensive tests for new features
3. Follow AT Protocol specifications
4. Update documentation for new functionality
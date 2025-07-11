# Registry Sync Proof of Concept

This proof of concept demonstrates two different approaches to synchronizing container artifacts between OCI-compliant registries using [Project Zot](https://zotregistry.io/) as the registry implementation.

## Overview

The POC compares two synchronization methods:
1. **RegClient Sync**: Using the external `regsync` tool from the [regclient](https://github.com/regclient/regclient) project
2. **Zot Sync**: Using Zot's built-in sync extension

## Architecture

The POC consists of:
- **Two Zot registries**: Source registry (localhost:5000) and Target registry (localhost:5001)
- **ORAS Go client**: Used to push test artifacts to the source registry
- **Sync implementations**: Two different sync methods to replicate artifacts
- **Verification**: Automated checks to ensure artifacts exist in both registries

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│  Source Registry│       │   Sync Method   │       │ Target Registry │
│   (localhost:5000)│ ────▶│ (regclient/zot) │ ────▶│  (localhost:5001)│
└─────────────────┘       └─────────────────┘       └─────────────────┘
         ▲                                                    │
         │                                                    │
         └─────────────── Verification ──────────────────────┘
```

## Sync Methods Comparison

### RegClient Sync Method

**How it works:**
- Uses the standalone `regsync` binary (v0.9.0) from the regclient project
- Executes as an external process with YAML configuration
- Performs a one-time sync of all artifacts from source to target
- Operates independently of the registry implementations

**Configuration:**
```yaml
version: 1
creds:
  - registry: localhost:5000
    tls: disabled
  - registry: localhost:5001
    tls: disabled
defaults:
  parallel: 2
  interval: 60m
sync:
  - source: localhost:5000/demo
    target: localhost:5001/demo
    type: repository
```

**Advantages:**
- Registry-agnostic (works with any OCI-compliant registry)
- Mature and well-tested solution
- Supports complex sync scenarios and filtering
- Can handle authentication and TLS configurations
- Supports scheduled syncing
- Direct access to sync operation logs, output is displayed directly in the terminal

**Disadvantages:**
- Requires external binary dependency
- Additional operational overhead
- Separate configuration management

### Zot Sync Method

**How it works:**
- Uses Zot's built-in sync extension
- Configures the target registry to pull artifacts on-demand
- Works by swapping configuration files to enable sync
- Integrated directly into the registry runtime

**Configuration:**
```json
{
  "extensions": {
    "sync": {
      "enable": true,
      "registries": [
        {
          "urls": ["http://zot-source:5000"],
          "content": [
            {
              "prefix": "**",
              "destination": ""
            }
          ],
          "onDemand": true,
          "tlsVerify": false
        }
      ]
    }
  }
}
```

**Advantages:**
- Native registry integration
- No external dependencies
- Automatic on-demand syncing
- Simplified configuration
- Better performance for large-scale scenarios

**Disadvantages:**
- Zot-specific (not registry-agnostic)
- Limited to Zot's sync capabilities
- Requires registry restart for configuration changes
- Sync logs are mixed with other registry logs, requires filtering container logs to monitor sync operations

## Prerequisites

- Docker and Docker Compose
- Go 1.23+ 
- [Task](https://taskfile.dev/) (optional, for automated setup)
- Internet connection (for downloading regsync binary)

## Setup and Usage

### Method 1: Using Task (Recommended)

```bash
# Install dependencies and run the POC
task sync-poc
```

This will:
1. Create the `bin/` directory
2. Download the regsync binary (v0.9.0)
3. Start both Zot registries with Docker Compose
4. Run the sync demonstration
5. Clean up containers and volumes

### Changing Sync Methods

Edit the `syncType` variable in `main.go`:

```go
// Change this to switch between sync methods
syncType := "zot"        // Use Zot's built-in sync
// syncType := "regclient"  // Use regclient's regsync
```

## What the POC Demonstrates

1. **Artifact Creation**: Creates two test artifacts (`artifact1` and `artifact2`) with sample content
2. **Source Push**: Pushes artifacts to the source registry (localhost:5000)
3. **Sync Execution**: Runs either regclient or zot sync method
4. **Verification**: Confirms artifacts exist in both source and target registries

## Key Differences Summary

| Aspect | RegClient Sync | Zot Sync |
|--------|---------------|----------|
| **Architecture** | External tool | Native extension |
| **Dependencies** | Requires binary | Built-in |
| **Configuration** | YAML file | JSON config |
| **Portability** | Registry-agnostic | Zot-specific |
| **Sync Mode** | On-demand/scheduled | On-demand/scheduled |
| **Logging** | Direct sync operation logs | Mixed with registry logs |

## References

- [Project Zot Documentation](https://zotregistry.io/)
- [Zot Mirroring](https://zotregistry.dev/v2.0.1/articles/mirroring/)
- [Regsync](https://regclient.org/usage/regsync/)

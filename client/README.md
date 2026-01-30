# Directory Golang SDK

## Overview

Dir Golang SDK provides a simple way to interact with the Directory API.
It allows developers to integrate and use Directory functionality from their applications with ease.

## Features

The Directory SDK provides comprehensive access to all Directory APIs with a simple, intuitive interface:

### **Store API**
- **Record Management**: Push records to the store and pull them by reference
- **Metadata Operations**: Look up record metadata without downloading full content
- **Data Lifecycle**: Delete records permanently from the store
- **Referrer Support**: Push and pull artifacts for existing records
- **Sync Management**: Manage storage synchronization policies between Directory servers

### **Search API**
- **Flexible Search**: Search stored records using text, semantic, and structured queries
- **Advanced Filtering**: Filter results by metadata, content type, and other criteria

### **Routing API**
- **Network Publishing**: Publish records to make them discoverable across the network
- **Content Discovery**: List and query published records across the network
- **Network Management**: Unpublish records to remove them from network discovery

### **Signing and Verification**
- **Local Signing**: Sign records locally using private keys or OIDC-based authentication. 
- **Remote Verification**: Verify record signatures using the Directory gRPC API

### **Developer Experience**
- **Async Support**: Non-blocking operations with streaming responses for large datasets
- **Error Handling**: Comprehensive gRPC error handling with detailed error messages
- **Configuration**: Flexible configuration via environment variables or direct instantiation

## Installation

1. Initialize the project:
```bash
go mod init example.com/myapp
```

2. Add the SDK to your project:
```bash
go get github.com/agntcy/dir/client
```

## Configuration

The SDK can be configured via environment variables or direct instantiation.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DIRECTORY_CLIENT_SERVER_ADDRESS` | Directory server address | `0.0.0.0:8888` |
| `DIRECTORY_CLIENT_AUTH_MODE` | Authentication mode: `x509`, `jwt`, or empty for insecure | `""` (insecure) |
| `DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH` | SPIFFE Workload API socket path | `""` |
| `DIRECTORY_CLIENT_JWT_AUDIENCE` | JWT audience for JWT authentication | `""` |

### Authentication

The SDK supports three authentication modes:

#### 1. Insecure (No Authentication)

For local development only. Not recommended for production.

**Environment Variables:**
```bash
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
# AUTH_MODE is empty or not set
```

**Code Example:**
```go
import (
    "context"
    "github.com/agntcy/dir/client"
)

ctx := context.Background()
config := &client.Config{
    ServerAddress: "localhost:8888",
    // AuthMode is empty - insecure connection
}
c, err := client.New(ctx, client.WithConfig(config))
if err != nil {
    // handle error
}
defer c.Close() // Always close to cleanup resources
```

#### 2. X509 (X.509-SVID)

Recommended for production. Requires SPIRE agent.

**Environment Variables:**
```bash
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRECTORY_CLIENT_AUTH_MODE="x509"
export DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH="unix:///run/spire/agent-sockets/api.sock"
```

**Code Example:**
```go
import (
    "context"
    "github.com/agntcy/dir/client"
)

ctx := context.Background()
config := &client.Config{
    ServerAddress:    "localhost:8888",
    AuthMode:         "x509",
    SpiffeSocketPath: "unix:///run/spire/agent-sockets/api.sock",
}
c, err := client.New(ctx, client.WithConfig(config))
if err != nil {
    // handle error
}
defer c.Close() // Always close to cleanup resources
```

#### 3. JWT (JWT-SVID)

Alternative to X.509 for client authentication. Requires SPIRE agent.

> **Note**: In JWT mode, the server presents its X.509-SVID via TLS for server 
> authentication and encryption, while the client authenticates using a JWT-SVID. 
> This provides both transport security and client authentication, following the 
> [official SPIFFE JWT pattern](https://github.com/spiffe/go-spiffe/tree/main/examples/spiffe-jwt).

**Environment Variables:**
```bash
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRECTORY_CLIENT_AUTH_MODE="jwt"
export DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH="unix:///run/spire/agent-sockets/api.sock"
export DIRECTORY_CLIENT_JWT_AUDIENCE="spiffe://example.org/dir-server"
```

**Code Example:**
```go
import (
    "context"
    "github.com/agntcy/dir/client"
)

ctx := context.Background()
config := &client.Config{
    ServerAddress:    "localhost:8888",
    AuthMode:         "jwt",
    SpiffeSocketPath: "unix:///run/spire/agent-sockets/api.sock",
    JWTAudience:      "spiffe://example.org/dir-server",
}
c, err := client.New(ctx, client.WithConfig(config))
if err != nil {
    // handle error
}
defer c.Close() // Always close to cleanup resources
```

## Getting Started

### Prerequisites

- [Golang](https://golang.org/dl/) - Go programming language

### 1. Server Setup

**Option A: Local Development Server**

```bash
# Clone the repository and start the server using Taskfile
task server:start
```

**Option B: Custom Server**

```bash
# Set your Directory server address
export DIRECTORY_CLIENT_SERVER_ADDRESS="your-server:8888"
```

### 2. SDK Installation

```bash
# Add the Directory SDK
go get github.com/agntcy/dir/client
```

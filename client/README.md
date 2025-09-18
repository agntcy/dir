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

The SDK can be configured via environment variables or direct instantiation:

```go
import "github.com/agntcy/dir/client"

// Environment variables
os.Setenv("DIRECTORY_CLIENT_SERVER_ADDRESS", "localhost:8888")
os.Setenv("DIRCTL_PATH", "/path/to/dirctl")
client := client.New()

// Or configure directly
config := &client.Config{
    ServerAddress: "localhost:8888",
}
client := client.New(client.WithConfig(config))

// Use SPIRE for mTLS communication
config := &client.Config{
    SpiffeSocketPath: "/tmp/agent.sock",
}
client := client.New(client.WithConfig(config))
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

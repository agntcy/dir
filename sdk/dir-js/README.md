# Directory JavaScript SDK

## Overview

Dir JavaScript SDK provides a simple way to interact with the Directory API.
It allows developers to integrate and use Directory functionality from their applications with ease.
The SDK supports both JavaScript and TypeScript applications.

**Note for users:** The SDK is intended for use in Node.js applications and will not work in Web applications.

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
Requires [dirctl](https://github.com/agntcy/dir/releases) binary to perform signing.
- **Remote Verification**: Verify record signatures using the Directory gRPC API

### **Developer Experience**
- **Type Safety**: Full type hints for better IDE support and fewer runtime errors
- **Async Support**: Non-blocking operations with streaming responses for large datasets
- **Error Handling**: Comprehensive gRPC error handling with detailed error messages
- **Configuration**: Flexible configuration via environment variables or direct instantiation

## Installation

Install the SDK using one of available JS package managers like [npm](https://www.npmjs.com/)

1. Initialize the project:
```bash
npm init -y
```

2. Add the SDK to your project:
```bash
npm install agntcy-dir
```

## Configuration

The SDK can be configured via environment variables or direct instantiation:

```js
// Environment variables (insecure mode)
process.env.DIRECTORY_CLIENT_SERVER_ADDRESS = "localhost:8888";
process.env.DIRCTL_PATH = "/path/to/dirctl";

// Environment variables (X.509 authentication)
process.env.DIRECTORY_CLIENT_SERVER_ADDRESS = "localhost:8888";
process.env.DIRECTORY_CLIENT_AUTH_MODE = "x509";
process.env.DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH = "/tmp/agent.sock";

// Environment variables (JWT authentication)
process.env.DIRECTORY_CLIENT_SERVER_ADDRESS = "localhost:8888";
process.env.DIRECTORY_CLIENT_AUTH_MODE = "jwt";
process.env.DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH = "/tmp/agent.sock";
process.env.DIRECTORY_CLIENT_JWT_AUDIENCE = "spiffe://example.org/dir-server";

// Or configure directly
import {Config, Client} from 'agntcy-dir';

// Insecure mode (default, for development only)
const config = new Config(
    serverAddress="localhost:8888",
    dirctlPath="/usr/local/bin/dirctl"
);
const client = new Client(config);

// X.509 authentication with SPIRE
const x509Config = new Config(
    "localhost:8888", 
    "/usr/local/bin/dirctl",
    "/tmp/agent.sock",  // SPIFFE socket path
    "x509"  // auth mode
);
const x509Transport = await Client.createGRPCTransport(x509Config);
const x509Client = new Client(x509Config, x509Transport);

// JWT authentication with SPIRE
const jwtConfig = new Config(
    "localhost:8888",
    "/usr/local/bin/dirctl",
    "/tmp/agent.sock",  // SPIFFE socket path
    "jwt",  // auth mode
    "spiffe://example.org/dir-server"  // JWT audience
);
const jwtTransport = await Client.createGRPCTransport(jwtConfig);
const jwtClient = new Client(jwtConfig, jwtTransport);
```

## Getting Started

### Prerequisites

- [NodeJS](https://nodejs.org/en/) - JavaScript runtime
- [npm](https://www.npmjs.com/) - Package manager
- [dirctl](https://github.com/agntcy/dir/releases) - Directory CLI binary
- Directory server instance (see setup below)

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
npm install agntcy-dir
```

### Usage Examples

See the [Example JavaScript Project](../examples/example-js/) for a complete working example that demonstrates all SDK features.

```bash
npm install
npm run example
```

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

### OAuth 2.0 for Directory bearer auth

The SDK supports OIDC/OAuth for Directory bearer authentication on gRPC:

- Interactive login via Authorization Code + PKCE with a loopback callback (`authenticateOAuthPkce()`)
- Pre-issued access token via `DIRECTORY_CLIENT_AUTH_TOKEN`

Interactive PKCE sessions are cached alongside other Directory tooling at `$XDG_CONFIG_HOME/dirctl/auth-token.json` or `~/.config/dirctl/auth-token.json`. Pre-issued tokens from configuration are used directly and are not written to the cache by the constructor.

Use this mode when your deployment expects a **Bearer access token** on gRPC (for example via a gateway that validates OIDC tokens). Register your IdP application with a **redirect URI** that matches `DIRECTORY_CLIENT_OIDC_REDIRECT_URI` exactly (for example `http://localhost:8484/callback`). The SDK starts a short-lived HTTP server on loopback to receive the authorization redirect.

Some IdPs use **public clients** with PKCE; your IdP may still expect a `client_secret` field in configuration. In that case, use a **random placeholder** from environment variables, not a real secret in source code.

**Important:** The default in-repo Envoy authorization stack validates **GitHub** tokens. OIDC access tokens from your IdP only work if your environment’s gateway or auth service is configured to accept them.

```bash
export DIRECTORY_CLIENT_AUTH_MODE="oidc"
export DIRECTORY_CLIENT_SERVER_ADDRESS="directory.example.com:443"
export DIRECTORY_CLIENT_OIDC_ISSUER="https://your-idp-provider.example.com"
export DIRECTORY_CLIENT_OIDC_CLIENT_ID="your-app-client-id"
# Optional placeholder for public clients:
export DIRECTORY_CLIENT_OIDC_CLIENT_SECRET="random-non-secret-string"
export DIRECTORY_CLIENT_OIDC_REDIRECT_URI="http://localhost:8484/callback"
# Optional: comma-separated scopes
# export DIRECTORY_CLIENT_OIDC_SCOPES="openid,profile,email"
```

```js
import { Client } from 'agntcy-dir';

// After exporting the variables above (or building a Config with authMode: 'oidc'):
const client = new Client();
await client.authenticateOAuthPkce();
```

For custom transports, call `Client.createGRPCTransport(oidcConfig, { oidcTokenHolder })` with an `OAuthTokenHolder` (exported from this package). The usual path is `new Client(oidcConfig)`, which wires the holder and transport automatically.

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

# Directory Python SDK

## Overview

Dir Python SDK provides a simple way to interact with the Directory API.
It allows developers to integrate and use Directory functionality from their Python applications with ease.

## Features

The Directory Python SDK provides comprehensive access to all Directory APIs with a simple, intuitive interface:

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

Install the SDK using [uv](https://github.com/astral-sh/uv)

1. Initialize the project:
```bash
uv init
```

2. Add the SDK to your project:
```bash
uv add agntcy-dir --index https://buf.build/gen/python
```

## Configuration

The SDK can be configured via environment variables or direct instantiation:

```python
# Environment variables (insecure mode, default)
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRCTL_PATH="/path/to/dirctl"

# Environment variables (X.509 authentication)
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRECTORY_CLIENT_AUTH_MODE="x509"
export DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH="/tmp/agent.sock"

# Environment variables (JWT authentication)
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRECTORY_CLIENT_AUTH_MODE="jwt"
export DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH="/tmp/agent.sock"
export DIRECTORY_CLIENT_JWT_AUDIENCE="spiffe://example.org/dir-server"

# Or configure directly
from agntcy.dir_sdk.client import Config, Client

# Insecure mode (default, for development only)
config = Config(
    server_address="localhost:8888",
    dirctl_path="/usr/local/bin/dirctl"
)
client = Client(config)

# X.509 authentication with SPIRE
x509_config = Config(
    server_address="localhost:8888",
    dirctl_path="/usr/local/bin/dirctl",
    spiffe_socket_path="/tmp/agent.sock",
    auth_mode="x509"
)
x509_client = Client(x509_config)

# JWT authentication with SPIRE
jwt_config = Config(
    server_address="localhost:8888",
    dirctl_path="/usr/local/bin/dirctl",
    spiffe_socket_path="/tmp/agent.sock",
    auth_mode="jwt",
    jwt_audience="spiffe://example.org/dir-server"
)
jwt_client = Client(jwt_config)
```

### OAuth 2.0 for Directory Bearer Auth

The Python SDK currently supports these OIDC/OAuth flows for Directory bearer auth:

- Interactive login via Authorization Code + PKCE with a loopback callback
- Pre-issued access token via `DIRECTORY_CLIENT_AUTH_TOKEN`

Interactive PKCE sessions are cached in the same location as the Go client:
`$XDG_CONFIG_HOME/dirctl/auth-token.json` or `~/.config/dirctl/auth-token.json`.
Explicit pre-issued tokens are used directly and are not cached.

Use this mode when your deployment expects a **Bearer access token** on gRPC (for example via a gateway that validates OIDC tokens). Register your IdP application with a **redirect URI** that matches `oidc_redirect_uri` exactly (for example `http://localhost:8484/callback`). The SDK starts a short-lived HTTP server on loopback to receive the authorization redirect.

Some IdPs use **public clients** with PKCE; Authlib may still expect a `client_secret` value in configuration. In that case, use a **random placeholder** from environment variables, not a real secret in source code.

**Important:** The default in-repo Envoy authz stack validates **GitHub** tokens. OIDC access tokens from your IdP provider only work if your environment’s gateway or auth service is configured to accept them.

```bash
export DIRECTORY_CLIENT_AUTH_MODE="oidc"
export DIRECTORY_CLIENT_SERVER_ADDRESS="directory.example.com:443"
export DIRECTORY_CLIENT_OIDC_ISSUER="https://your-idp-provider.example.com"
export DIRECTORY_CLIENT_OIDC_CLIENT_ID="your-app-client-id"
# Optional placeholder for public clients:
export DIRECTORY_CLIENT_OIDC_CLIENT_SECRET="random-non-secret-string"
export DIRECTORY_CLIENT_OIDC_REDIRECT_URI="http://localhost:8484/callback"
# Optional: comma-separated scopes
export DIRECTORY_CLIENT_OIDC_SCOPES="openid,profile,email"
# Optional: override gRPC TLS server name / authority
export DIRECTORY_CLIENT_TLS_SERVER_NAME="directory.example.com"
# Optional: non-interactive use (CI) after obtaining a token elsewhere
export DIRECTORY_CLIENT_AUTH_TOKEN="your-access-token"
# Optional: skip TLS certificate verification for IdP HTTPS only (development; avoid in production)
export DIRECTORY_CLIENT_TLS_SKIP_VERIFY="false"
```

```python
from agntcy.dir_sdk.client import Client, Config, OAuthPkceError

config = Config(
    server_address="directory.example.com:443",
    auth_mode="oidc",
    oidc_issuer="https://your-idp-provider.example.com",
    oidc_client_id="your-app-client-id",
    oidc_client_secret="random-placeholder-if-required",
    oidc_redirect_uri="http://localhost:8484/callback",
    oidc_callback_port=8484,
    oidc_auth_timeout=300.0,
)
client = Client(config)
# Client construction does not start browser login automatically.
# Opens the system browser and completes PKCE on loopback:
try:
    client.authenticate_oauth_pkce()
except OAuthPkceError as e:
    print(f"Login failed: {e}")
```

gRPC transport to the Directory still uses **TLS with system trust anchors** (or `tls_ca_file` if set). `TLS_SKIP_VERIFY` applies to **HTTPS calls to the OIDC issuer** (discovery and token endpoint), not to relaxing gRPC TLS to the Directory.

If you need to force the TLS server name / authority used by gRPC, set
`DIRECTORY_CLIENT_TLS_SERVER_NAME`.

For non-interactive callers that already have an access token, skip PKCE entirely:

```python
from agntcy.dir_sdk.client import Client, Config

config = Config(
    server_address="directory.example.com:443",
    auth_mode="oidc",
    auth_token="your-access-token",
)
client = Client(config)
```

If no explicit `auth_token` is provided, the SDK will also try to reuse a valid
cached interactive token from the shared `dirctl` cache path before you need to
run `client.authenticate_oauth_pkce()`.

## Error Handling

The SDK primarily raises `grpc.RpcError` exceptions for gRPC communication issues and `RuntimeError` for configuration problems:

```python
import grpc
from agntcy.dir_sdk.client import Client, OAuthPkceError

try:
    client = Client()
    records = client.list(list_request)
except grpc.RpcError as e:
    # Handle gRPC errors
    if e.code() == grpc.StatusCode.NOT_FOUND:
        print("Resource not found")
    elif e.code() == grpc.StatusCode.UNAVAILABLE:
        print("Server unavailable")
    else:
        print(f"gRPC error: {e.details()}")
except RuntimeError as e:
    # Handle configuration or subprocess errors
    print(f"Runtime error: {e}")
except OAuthPkceError as e:
    # Browser / loopback OAuth PKCE flow failed
    print(f"OAuth error: {e}")
```

Common gRPC status codes:
- `NOT_FOUND`: Resource doesn't exist
- `ALREADY_EXISTS`: Resource already exists
- `UNAVAILABLE`: Server is down or unreachable
- `PERMISSION_DENIED`: Authentication/authorization failure
- `INVALID_ARGUMENT`: Invalid request parameters


## Getting Started

### Prerequisites

- Python 3.10 or higher
- [uv](https://github.com/astral-sh/uv) - Package manager
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
uv add agntcy-dir --index https://buf.build/gen/python
```

### Usage Examples

See the [Example Python Project](../examples/example-py/) for a complete working example that demonstrates all SDK features.

```bash
uv sync
uv run example.py
```

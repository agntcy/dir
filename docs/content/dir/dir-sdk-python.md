# Python SDK

[![PyPI version](https://img.shields.io/pypi/v/agntcy-dir.svg)](https://pypi.org/project/agntcy-dir/)

The Directory Python SDK (`agntcy-dir`) provides a high-level client for interacting with the
Directory gRPC API from Python 3.10+ applications.

Source and issue tracker: [github.com/agntcy/dir-sdk-python](https://github.com/agntcy/dir-sdk-python)

## Features

| API | Methods |
|-----|---------|
| **Store** | `push`, `pull`, `lookup`, `delete`, `push_referrer`, `pull_referrer`, `delete_referrer` |
| **Search** | `search_records`, `search_cids` |
| **Routing** | `publish`, `unpublish`, `list`, `search_routing` |
| **Publication** | `create_publication`, `get_publication`, `list_publication` |
| **Naming** | `resolve`, `get_verification_info` |
| **Sync** | `create_sync`, `get_sync`, `list_syncs`, `delete_sync` |
| **Events** | `listen` (server-streaming) |
| **Signing** | `sign` (local via `dirctl`), `verify` (remote gRPC) |

## Installation

Requires [uv](https://github.com/astral-sh/uv).

1. Initialize the project:

    ```bash
    uv init
    ```

1. Add the SDK — the buf.build index is required for the generated protobuf packages:

    ```bash
    uv add agntcy-dir --index https://buf.build/gen/python
    ```

## Configuration

### Environment variables

```bash
# Insecure (local development)
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRCTL_PATH="/path/to/dirctl"

# X.509 / SPIFFE
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRECTORY_CLIENT_AUTH_MODE="x509"
export DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH="/tmp/agent.sock"

# JWT / SPIFFE
export DIRECTORY_CLIENT_AUTH_MODE="jwt"
export DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH="/tmp/agent.sock"
export DIRECTORY_CLIENT_JWT_AUDIENCE="spiffe://example.org/dir-server"

# TLS (custom CA / mTLS)
export DIRECTORY_CLIENT_AUTH_MODE="tls"
export DIRECTORY_CLIENT_TLS_CA_FILE="/path/to/ca.crt"
export DIRECTORY_CLIENT_TLS_CERT_FILE="/path/to/client.crt"
export DIRECTORY_CLIENT_TLS_KEY_FILE="/path/to/client.key"
```

Then create a client from the environment:

```python
from agntcy.dir_sdk.client import Client

client = Client()  # reads DIRECTORY_CLIENT_* env vars
```

### Direct instantiation

```python
from agntcy.dir_sdk.client import Config, Client

# Insecure (local development only)
config = Config(
    server_address="localhost:8888",
    dirctl_path="/usr/local/bin/dirctl",
)
client = Client(config)

# X.509 with SPIRE
config = Config(
    server_address="localhost:8888",
    spiffe_socket_path="/tmp/agent.sock",
    auth_mode="x509",
)
client = Client(config)

# JWT with SPIRE
config = Config(
    server_address="localhost:8888",
    spiffe_socket_path="/tmp/agent.sock",
    auth_mode="jwt",
    jwt_audience="spiffe://example.org/dir-server",
)
client = Client(config)
```

### Docker-based dirctl

If you don't have `dirctl` installed locally you can run it via Docker by passing a
`DockerConfig` instead of `dirctl_path`:

```python
from agntcy.dir_sdk.client import Config, Client
from agntcy.dir_sdk.client.config import DockerConfig

config = Config(
    server_address="localhost:8888",
    docker_config=DockerConfig(
        dirctl_image="ghcr.io/agntcy/dir-ctl",
        dirctl_image_tag="latest",
    ),
)
client = Client(config)
```

!!! note

    `dirctl_path` and `docker_config` are mutually exclusive. Setting both raises a
    `ValueError`.

## OAuth 2.0 / OIDC bearer auth

Use `auth_mode="oidc"` when your deployment expects a Bearer token on gRPC (e.g. via an
Envoy gateway that validates OIDC tokens).

```bash
export DIRECTORY_CLIENT_AUTH_MODE="oidc"
export DIRECTORY_CLIENT_SERVER_ADDRESS="directory.example.com:443"
export DIRECTORY_CLIENT_OIDC_ISSUER="https://your-idp-provider.example.com"
export DIRECTORY_CLIENT_OIDC_CLIENT_ID="your-app-client-id"
# Optional — use a random placeholder for public clients:
export DIRECTORY_CLIENT_OIDC_CLIENT_SECRET="random-non-secret-string"
export DIRECTORY_CLIENT_OIDC_REDIRECT_URI="http://localhost:8484/callback"
# Optional: comma-separated extra scopes
export DIRECTORY_CLIENT_OIDC_SCOPES="openid,profile,email"
# Optional: non-interactive use — skip the browser flow
export DIRECTORY_CLIENT_AUTH_TOKEN="your-access-token"
```

**Interactive PKCE login** (opens a browser):

```python
from agntcy.dir_sdk.client import Client, Config, OAuthPkceError

config = Config(
    server_address="directory.example.com:443",
    auth_mode="oidc",
    oidc_issuer="https://your-idp-provider.example.com",
    oidc_client_id="your-app-client-id",
    oidc_redirect_uri="http://localhost:8484/callback",
)
client = Client(config)

try:
    if not client.has_cached_oauth_token():
        client.authenticate_oauth_pkce()
except OAuthPkceError as e:
    print(f"Login failed: {e}")
```

**Non-interactive** (CI / pre-issued token):

```python
config = Config(
    server_address="directory.example.com:443",
    auth_mode="oidc",
    auth_token="your-access-token",
)
client = Client(config)
```

Interactive sessions are cached at `$XDG_CONFIG_HOME/dirctl/auth-token.json` (or
`~/.config/dirctl/auth-token.json`) and shared with the `dirctl` CLI.

!!! note

    `TLS_SKIP_VERIFY` applies only to HTTPS calls to the OIDC issuer (discovery and token
    endpoints), not to the gRPC TLS connection to the Directory server.

## Usage

### Store — push and pull records

```python
import json
from agntcy.dir_sdk.client import Client
from agntcy.dir_sdk.models import core_v1

client = Client()

# Push a record
record = core_v1.Record()
record.blob = json.dumps({"name": "my-agent", "version": "1.0.0"}).encode()

refs = client.push([record])
cid = refs[0].cid
print(f"Pushed record CID: {cid}")

# Pull it back
records = client.pull(refs)

# Look up metadata only (no content download)
metas = client.lookup(refs)
```

### Routing — publish and discover

```python
from agntcy.dir_sdk.models import routing_v1

# Publish a CID to the routing layer
client.publish(routing_v1.PublishRequest(cid=cid))

# List published records
responses = client.list(routing_v1.ListRequest())
for resp in responses:
    print(resp.cid)

# Search the routing layer
results = client.search_routing(routing_v1.SearchRequest(query="my skill"))

# Unpublish
client.unpublish(routing_v1.UnpublishRequest(cid=cid))
```

### Search

```python
from agntcy.dir_sdk.models import search_v1

# Full-text / semantic record search
results = client.search_records(
    search_v1.SearchRecordsRequest(query="my-agent")
)

# CID-only search
cids = client.search_cids(
    search_v1.SearchCIDsRequest(query="my-agent")
)
```

### Naming — resolve and verify

```python
# Resolve a name to a record reference
resp = client.resolve("my-agent", version="1.0.0")
print(resp.ref.cid)

# Get signing/verification metadata
info = client.get_verification_info(name="my-agent", version="1.0.0")
```

### Events — real-time streaming

```python
from agntcy.dir_sdk.models import events_v1

stream = client.listen(events_v1.ListenRequest())
for event in stream:
    print(event)
```

### Signing and verification

```python
from agntcy.dir_sdk.models import sign_v1

# Sign a record locally (requires dirctl binary)
client.sign(sign_v1.SignRequest(cid=cid))

# Verify a record signature via gRPC
resp = client.verify(sign_v1.VerifyRequest(cid=cid))
print(resp.valid)
```

## Error handling

The SDK raises `grpc.RpcError` for gRPC errors and `RuntimeError` for configuration problems:

```python
import grpc
from agntcy.dir_sdk.client import Client, OAuthPkceError

try:
    client = Client()
    records = client.pull([ref])
except grpc.RpcError as e:
    if e.code() == grpc.StatusCode.NOT_FOUND:
        print("Record not found")
    elif e.code() == grpc.StatusCode.UNAVAILABLE:
        print("Server unavailable")
    else:
        print(f"gRPC error: {e.details()}")
except RuntimeError as e:
    print(f"Configuration error: {e}")
except OAuthPkceError as e:
    print(f"OAuth error: {e}")
```

Common status codes: `NOT_FOUND`, `ALREADY_EXISTS`, `UNAVAILABLE`, `PERMISSION_DENIED`,
`INVALID_ARGUMENT`.

## Prerequisites

- Python 3.10 or higher
- [uv](https://github.com/astral-sh/uv) package manager
- [dirctl](https://github.com/agntcy/dir/releases) CLI binary (signing operations only)
- A running Directory server — see [Quickstart](dir-quickstart.md) or
  [Kubernetes Deployment](dir-deployment-kubernetes.md)

## Examples

### `examples/example.py`

**Source:** [`examples/example.py`](https://github.com/agntcy/dir-sdk-python/blob/main/examples/example.py)

Demonstrates the core store and routing workflow end-to-end:

1. Pushes two OASF records and prints their CIDs
2. Pulls the records back and prints their content
3. Looks up record metadata (without re-downloading content)
4. Publishes one record to the routing layer
5. Lists routing entries filtered by skill
6. Searches CIDs by version pattern (`v1.*`)
7. Searches CIDs by annotation key:value (`env:prod`)
8. Unpublishes the record
9. Deletes both records from the store

**Run:**

```bash
cd examples
uv sync
uv run example.py
```

### `examples/example_interactive_oidc.py`

**Source:** [`examples/example_interactive_oidc.py`](https://github.com/agntcy/dir-sdk-python/blob/main/examples/example_interactive_oidc.py)

Demonstrates OIDC/PKCE interactive login. Reuses a cached token if one exists, otherwise
opens the system browser for the authorization flow. Once authenticated, runs a `SearchCIDs`
query filtered by version.

**Required environment variable:**

```bash
export DIRECTORY_CLIENT_OIDC_CLIENT_ID="your-client-id"
```

**Optional overrides:** `DIRECTORY_CLIENT_SERVER_ADDRESS`, `DIRECTORY_CLIENT_OIDC_CLIENT_SECRET`,
`DIRECTORY_CLIENT_OIDC_REDIRECT_URI`, `DIRECTORY_CLIENT_OIDC_CALLBACK_PORT`,
`DIRECTORY_CLIENT_AUTH_TOKEN` (skips the browser flow when pre-issued token is available).

**Run:**

```bash
cd examples
uv sync
uv run example_interactive_oidc.py --version "v1*" --limit 3
```

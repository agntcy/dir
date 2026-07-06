# JavaScript / TypeScript SDK

[![npm version](https://img.shields.io/npm/v/agntcy-dir.svg)](https://www.npmjs.com/package/agntcy-dir)

The Directory JavaScript SDK (`agntcy-dir`) provides a high-level client for interacting with the
Directory gRPC API from Node.js applications. The package ships TypeScript types and works with
both JavaScript and TypeScript projects.

!!! note

    The SDK targets Node.js only and does not work in browser or edge-runtime environments.

Source and issue tracker: [github.com/agntcy/dir-sdk-javascript](https://github.com/agntcy/dir-sdk-javascript)

## Features

| API | Methods |
|-----|---------|
| **Store** | `push`, `pull`, `lookup`, `delete`, `push_referrer`, `pull_referrer`, `deleteReferrer` |
| **Search** | `searchRecords`, `searchCIDs` |
| **Routing** | `publish`, `unpublish`, `list`, `searchRouting` |
| **Publication** | `create_publication`, `get_publication`, `list_publication` |
| **Naming** | `resolve`, `getVerificationInfo` |
| **Sync** | `create_sync`, `get_sync`, `list_syncs`, `delete_sync` |
| **Events** | `listen` (async iterable server-stream) |
| **Signing** | `sign` (local via `dirctl`), `verify` (remote gRPC) |

## Installation

Requires [Node.js](https://nodejs.org/en/) and [npm](https://www.npmjs.com/).

1. Initialize the project:

    ```bash
    npm init -y
    ```

1. Add the SDK:

    ```bash
    npm install agntcy-dir
    ```

## Configuration

The `Config` constructor accepts positional parameters. All parameters have defaults, so you only
need to pass what differs from the defaults.

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

```js
import { Client } from 'agntcy-dir';

const client = new Client();  // reads DIRECTORY_CLIENT_* env vars
```

### Direct instantiation

```js
import { Config, Client } from 'agntcy-dir';

// Insecure (local development only)
const config = new Config("localhost:8888", "/usr/local/bin/dirctl");
const client = new Client(config);

// X.509 with SPIRE
const x509Config = new Config(
    "localhost:8888",        // serverAddress
    "/usr/local/bin/dirctl", // dirctlPath
    "/tmp/agent.sock",       // spiffeEndpointSocket
    "x509"                   // authMode
);
const x509Transport = await Client.createGRPCTransport(x509Config);
const x509Client = new Client(x509Config, x509Transport);

// JWT with SPIRE
const jwtConfig = new Config(
    "localhost:8888",
    "/usr/local/bin/dirctl",
    "/tmp/agent.sock",
    "jwt",
    "spiffe://example.org/dir-server"  // jwtAudience
);
const jwtTransport = await Client.createGRPCTransport(jwtConfig);
const jwtClient = new Client(jwtConfig, jwtTransport);
```

!!! note

    `dirctl` is only required for local signing operations. All other API calls work without it.

## OAuth 2.0 / OIDC bearer auth

Use `authMode: 'oidc'` when your deployment expects a Bearer token on gRPC (e.g. via an
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
# export DIRECTORY_CLIENT_OIDC_SCOPES="openid,profile,email"
# Optional: non-interactive use — skip the browser flow
export DIRECTORY_CLIENT_AUTH_TOKEN="your-access-token"
```

**Interactive PKCE login** (opens a browser):

```js
import { Client } from 'agntcy-dir';

// After exporting the variables above (or building a Config with authMode: 'oidc'):
const client = new Client();
if (!client.hasCachedOAuthToken()) {
    await client.authenticateOAuthPkce();
}
```

**Non-interactive** (CI / pre-issued token):

```js
import { Config, Client } from 'agntcy-dir';

const config = new Config(
    "directory.example.com:443",
    undefined,   // dirctlPath — not needed if signing is unused
    undefined,   // spiffeEndpointSocket
    "oidc",      // authMode
    undefined,   // jwtAudience
    undefined,   // tlsCaFile
    undefined,   // tlsCertFile
    undefined,   // tlsKeyFile
    "your-access-token"  // authToken
);
const client = new Client(config);
```

Interactive sessions are cached at `$XDG_CONFIG_HOME/dirctl/auth-token.json` (or
`~/.config/dirctl/auth-token.json`) and shared with the `dirctl` CLI.

## Usage

### Store — push and pull records

```js
import { Client } from 'agntcy-dir';

const client = new Client();

// Push a record
const encoder = new TextEncoder();
const record = { blob: encoder.encode(JSON.stringify({ name: "my-agent", version: "1.0.0" })) };

const refs = await client.push([record]);
const cid = refs[0].cid;
console.log(`Pushed record CID: ${cid}`);

// Pull it back
const records = await client.pull(refs);

// Look up metadata only (no content download)
const metas = await client.lookup(refs);
```

### Routing — publish and discover

```js
// Publish a CID to the routing layer
await client.publish({ cid });

// List published records
const responses = await client.list({});
for (const resp of responses) {
    console.log(resp.cid);
}

// Search the routing layer
const results = await client.searchRouting({ query: "my skill" });

// Unpublish
await client.unpublish({ cid });
```

### Search

```js
// Full-text / semantic record search
const results = await client.searchRecords({ query: "my-agent" });

// CID-only search
const cids = await client.searchCIDs({ query: "my-agent" });
```

### Naming — resolve and verify

```js
// Resolve a name to a record reference
const resp = await client.resolve({ name: "my-agent", version: "1.0.0" });
console.log(resp.ref.cid);

// Get signing/verification metadata
const info = await client.getVerificationInfo({ name: "my-agent", version: "1.0.0" });
```

### Events — real-time streaming

```js
const stream = client.listen({});
for await (const event of stream) {
    console.log(event);
}
```

### Signing and verification

```js
// Sign a record locally (requires dirctl binary)
client.sign({ cid });

// Verify a record signature via gRPC
const resp = await client.verify({ cid });
console.log(resp.valid);
```

## Error handling

gRPC errors from `@connectrpc/connect` are thrown as `ConnectError`. Check `error.code` against
the `Code` enum:

```js
import { Client } from 'agntcy-dir';
import { ConnectError, Code } from '@connectrpc/connect';

try {
    const records = await client.pull([ref]);
} catch (e) {
    if (e instanceof ConnectError) {
        if (e.code === Code.NotFound) {
            console.error("Record not found");
        } else if (e.code === Code.Unavailable) {
            console.error("Server unavailable");
        } else {
            console.error(`gRPC error ${e.code}: ${e.message}`);
        }
    }
}
```

Common codes: `Code.NotFound`, `Code.AlreadyExists`, `Code.Unavailable`,
`Code.PermissionDenied`, `Code.InvalidArgument`.

## Prerequisites

- [Node.js](https://nodejs.org/en/) 18 or higher
- [npm](https://www.npmjs.com/) package manager
- [dirctl](https://github.com/agntcy/dir/releases) CLI binary (signing operations only)
- A running Directory server — see [Quickstart](dir-quickstart.md) or
  [Kubernetes Deployment](dir-deployment-kubernetes.md)

## Examples

### `examples/example.js`

**Source:** [`examples/example.js`](https://github.com/agntcy/dir-sdk-javascript/blob/main/examples/example.js)

Demonstrates the core store and routing workflow end-to-end:

1. Pushes two OASF records and prints their references
2. Pulls the records back and prints their content
3. Looks up record metadata (without re-downloading content)
4. Searches CIDs by skill ID
5. Searches CIDs by annotation key:value (`env:prod`)
6. Publishes the records to the routing layer
7. Lists routing entries filtered by skill path
8. Unpublishes the records
9. Deletes both records from the store

**Run:**

```bash
cd examples
npm install
npm run example
```

### `examples/example_interactive_oidc.js`

**Source:** [`examples/example_interactive_oidc.js`](https://github.com/agntcy/dir-sdk-javascript/blob/main/examples/example_interactive_oidc.js)

Demonstrates OIDC/PKCE interactive login. Reuses a cached token if one exists, otherwise
opens the system browser for the authorization flow. Once authenticated, runs a `searchCIDs`
query filtered by version. Accepts `--version` and `--limit` CLI arguments.

**Required environment variable:**

```bash
export DIRECTORY_CLIENT_OIDC_CLIENT_ID="your-client-id"
```

**Optional overrides:** `DIRECTORY_CLIENT_SERVER_ADDRESS`, `DIRECTORY_CLIENT_OIDC_CLIENT_SECRET`,
`DIRECTORY_CLIENT_OIDC_REDIRECT_URI`, `DIRECTORY_CLIENT_OIDC_CALLBACK_PORT`,
`DIRECTORY_CLIENT_OIDC_AUTH_TIMEOUT`, `DIRECTORY_CLIENT_TLS_SERVER_NAME`,
`DIRECTORY_CLIENT_AUTH_TOKEN` (skips the browser flow when a pre-issued token is available).

**Run:**

```bash
cd examples
npm install
node example_interactive_oidc.js --version "v1*" --limit 3
```

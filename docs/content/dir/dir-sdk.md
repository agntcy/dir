# Directory SDK

Libraries for interacting with Directory over gRPC: push records to the store, publish for
routing, and search or pull by skill. All three SDKs target the same workflow; pick the
language that fits your stack.

## Publish and discover (all SDKs)

After pointing the client at a running Directory (`localhost:8888` for the local daemon):

1. **Push** an OASF record JSON to obtain a CID.
2. **Publish** the CID to the routing layer for network discovery.
3. **Search** by skill and **pull** matching records.

Use [Quickstart](dir-quickstart.md) to run `dirctl daemon start` locally. Remote servers require
[OIDC](dir-component-oidc-authentication.md) or SPIFFE configuration per SDK docs below.

Protocol definitions: [buf.build/agntcy/dir](https://buf.build/agntcy/dir) and
[API Reference](dir-api-reference.md).

## JavaScript SDK

Documentation for the JavaScript SDK can be found on [GitHub](https://github.com/agntcy/dir-sdk-javascript). The SDK supports both JavaScript and TypeScript applications. The package is published on [npm](https://www.npmjs.com/package/agntcy-dir).

!!! note

    The SDK is intended for use in Node.js applications and does not work in web applications.

### Installation

Install the SDK using one of available JS package managers like [npm](https://www.npmjs.com/)

1. Initialize the project:

    ```bash
    npm init -y
    ```

1. Add the SDK to your project:

    ```bash
    npm install agntcy-dir
    ```

### Configuration

The SDK can be configured via environment variables or direct instantiation:

```js
import {Config, Client} from 'agntcy-dir';

// Environment variables
process.env.DIRECTORY_CLIENT_SERVER_ADDRESS = "localhost:8888";
process.env.DIRCTL_PATH = "/path/to/dirctl";
const client = new Client();

// Or configure directly
const config = new Config(
    serverAddress="localhost:8888",
    dirctlPath="/usr/local/bin/dirctl"
);
const client = new Client(config);

// Use SPIRE for mTLS communication
const config = new Config(spiffeEndpointSocket="/tmp/agent.sock");
const transport = await Client.createGRPCTransport(config);
const spiffeClient = new Client(config, transport);
```

!!! note

    JavaScript SDK requires Directory CLI (dirctl) only to perform signing operations. If you don't need signing, you can use the SDK without dirctl.

## Python SDK

Documentation for the Python SDK can be found on [GitHub](https://github.com/agntcy/dir-sdk-python).
The SDK supports Python 3.10+ applications. The package is published on [PyPI](https://pypi.org/project/agntcy-dir/).

### Installation

Install the SDK using [uv](https://github.com/astral-sh/uv)

1. Initialize the project:
    ```bash
    uv init
    ```

1. Add the SDK to your project:
    ```bash
    uv add agntcy-dir --index https://buf.build/gen/python
    ```

### Configuration

The SDK can be configured via environment variables or direct instantiation:

```python
from agntcy.dir_sdk.client import Config, Client

# Environment variables
export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:8888"
export DIRCTL_PATH="/path/to/dirctl"
client = Client()

# Or configure directly
config = Config(
    server_address="localhost:8888",
    dirctl_path="/usr/local/bin/dirctl"
)
client = Client(config)

# Use SPIRE for mTLS communication
config = Config(
    spiffe_socket_path="/tmp/agent.sock"
)
client = Client(config)
```

!!! note

    Python SDK requires Directory CLI (dirctl) only to perform signing operations. If you don't need signing, you can use the SDK without dirctl.

## Go SDK

Documentation for the Go SDK can be found at [GitHub](https://github.com/agntcy/dir/tree/main/client). The package is available on [pkg.go.dev](https://pkg.go.dev/github.com/agntcy/dir/client).

### Installation

Install the SDK using `go get`

```bash
go get github.com/agntcy/dir/client
```

### Configuration

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

!!! note

    Golang SDK does not require Directory CLI (dirctl) and can be used standalone.

# Go SDK

The Directory Go SDK (`github.com/agntcy/dir/client`) provides a high-level client for
interacting with the Directory gRPC API from Go applications. It is embedded in the main `dir`
repository alongside the server.

Source: [github.com/agntcy/dir/tree/main/client](https://github.com/agntcy/dir/tree/main/client)  
Package reference: [pkg.go.dev/github.com/agntcy/dir/client](https://pkg.go.dev/github.com/agntcy/dir/client)

## Features

| API | Methods |
|-----|---------|
| **Store** | `Push`, `PushBatch`, `PushStream`, `Pull`, `PullBatch`, `PullStream`, `Lookup`, `LookupBatch`, `LookupStream`, `Delete`, `DeleteBatch`, `DeleteStream`, `PushReferrer`, `PullReferrer`, `DeleteReferrer` |
| **Search** | `SearchRecords`, `SearchCIDs` (streaming via `StreamResult`) |
| **Routing** | `Publish`, `Unpublish`, `List`, `SearchRouting` |
| **Naming** | `Resolve`, `GetVerificationInfo`, `GetVerificationInfoByName` |
| **Sync** | `CreateSync`, `GetSync`, `ListSyncs`, `DeleteSync` |
| **Events** | `ListenStream` (server-streaming via `StreamResult`) |
| **Signing** | `Sign` (local cosign — no `dirctl` required), `Verify`, `PullSignatures`, `PullPublicKeys` |

!!! note

    The Go SDK performs signing natively via [cosign](https://github.com/sigstore/cosign)
    and does not require the `dirctl` binary.

## Installation

```bash
go get github.com/agntcy/dir/client
```

Requires Go 1.21 or higher.

## Configuration

### Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DIRECTORY_CLIENT_SERVER_ADDRESS` | Directory server address | `0.0.0.0:8888` |
| `DIRECTORY_CLIENT_AUTH_MODE` | Auth mode: `x509`, `jwt`, `tls`, `oidc`, `insecure`, or empty | `""` (auto-detect) |
| `DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH` | SPIFFE Workload API socket path | `""` |
| `DIRECTORY_CLIENT_JWT_AUDIENCE` | JWT audience (JWT mode only) | `""` |
| `DIRECTORY_CLIENT_TLS_CA_FILE` | CA certificate file (TLS mode) | `""` |
| `DIRECTORY_CLIENT_TLS_CERT_FILE` | Client certificate file (TLS mode) | `""` |
| `DIRECTORY_CLIENT_TLS_KEY_FILE` | Client key file (TLS mode) | `""` |
| `DIRECTORY_CLIENT_AUTH_TOKEN` | Pre-issued Bearer token (OIDC mode) | `""` |
| `DIRECTORY_CLIENT_OIDC_ISSUER` | OIDC issuer URL | `""` |
| `DIRECTORY_CLIENT_OIDC_CLIENT_ID` | OIDC client ID | `""` |

Load config from the environment with `WithEnvConfig()`:

```go
c, err := client.New(ctx, client.WithEnvConfig())
```

### Direct instantiation

```go
import (
    "context"
    "github.com/agntcy/dir/client"
)

ctx := context.Background()

config := &client.Config{
    ServerAddress: "localhost:8888",
}
c, err := client.New(ctx, client.WithConfig(config))
if err != nil {
    log.Fatal(err)
}
defer c.Close()
```

Always call `c.Close()` to release the gRPC connection and any SPIFFE credential sources.

### Auto-detect mode

When `AuthMode` is empty (the default), the client auto-detects credentials in this order:

1. A pre-issued `AuthToken` or `OIDCIssuer`/`OIDCClientID` set in `Config`
2. A valid cached OIDC token from the shared `dirctl` token cache
3. Falls back to an insecure connection if none of the above are found

## Authentication modes

### Insecure (local development only)

```go
config := &client.Config{
    ServerAddress: "localhost:8888",
    AuthMode:      "insecure",
}
```

### X.509 / SPIFFE (recommended for production)

```go
config := &client.Config{
    ServerAddress:    "localhost:8888",
    AuthMode:         "x509",
    SpiffeSocketPath: "unix:///run/spire/agent-sockets/api.sock",
}
```

### JWT / SPIFFE

In JWT mode the server authenticates via its X.509-SVID (TLS) while the client sends a
JWT-SVID as a per-RPC credential.

```go
config := &client.Config{
    ServerAddress:    "localhost:8888",
    AuthMode:         "jwt",
    SpiffeSocketPath: "unix:///run/spire/agent-sockets/api.sock",
    JWTAudience:      "spiffe://example.org/dir-server",
}
```

### TLS (custom CA / mTLS)

```go
config := &client.Config{
    ServerAddress: "directory.example.com:443",
    AuthMode:      "tls",
    TlsCAFile:     "/path/to/ca.crt",
    TlsCertFile:   "/path/to/client.crt",
    TlsKeyFile:    "/path/to/client.key",
}
```

### OIDC / Bearer token

**Pre-issued token** (CI / scripts):

```go
config := &client.Config{
    ServerAddress: "gateway.example.com:443",
    AuthMode:      "oidc",
    AuthToken:     "eyJhbG...",
}
```

**Interactive login** — run once via `dirctl`, then reuse the cached token:

```bash
dirctl auth login --oidc-issuer=https://dex.example.com --oidc-client-id=dirctl
```

```go
// Reuse the cached token automatically (AuthToken left empty)
config := &client.Config{
    ServerAddress: "gateway.example.com:443",
    AuthMode:      "oidc",
    OIDCIssuer:    "https://dex.example.com",
    OIDCClientID:  "dirctl",
}
```

## Usage

### Store — push and pull records

The Go SDK exposes single-record, batch, and raw-streaming variants for every store operation.

```go
import (
    corev1 "github.com/agntcy/dir/api/core/v1"
)

// Push a single record
record := &corev1.Record{
    Blob: []byte(`{"name":"my-agent","version":"1.0.0"}`),
}
ref, err := c.Push(ctx, record)
if err != nil {
    log.Fatal(err)
}
fmt.Println("CID:", ref.Cid)

// Pull it back
pulled, err := c.Pull(ctx, ref)

// Metadata only (no content download)
meta, err := c.Lookup(ctx, ref)

// Delete
err = c.Delete(ctx, ref)
```

**Batch operations** for efficiency:

```go
refs, err := c.PushBatch(ctx, []*corev1.Record{record1, record2})
records, err := c.PullBatch(ctx, refs)
metas, err := c.LookupBatch(ctx, refs)
err = c.DeleteBatch(ctx, refs)
```

**Streaming operations** using channels (for large or incremental datasets):

```go
import "github.com/agntcy/dir/client/streaming"

// Build a channel of records to push as they become available
recordsCh := streaming.SliceToChan(ctx, records)
result, err := c.PushStream(ctx, recordsCh)
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case ref := <-result.ResCh():
        fmt.Println("pushed:", ref.Cid)
    case err := <-result.ErrCh():
        log.Println("error:", err)
    case <-result.DoneCh():
        return
    }
}
```

### Routing — publish and discover

```go
import routingv1 "github.com/agntcy/dir/api/routing/v1"

// Publish a CID to the routing layer
err := c.Publish(ctx, &routingv1.PublishRequest{Cid: ref.Cid})

// Stream listed records
resCh, err := c.List(ctx, &routingv1.ListRequest{})
for resp := range resCh {
    fmt.Println(resp.Cid)
}

// Search the routing layer (streaming)
searchCh, err := c.SearchRouting(ctx, &routingv1.SearchRequest{Query: "my skill"})
for resp := range searchCh {
    fmt.Println(resp.Cid)
}

// Unpublish
err = c.Unpublish(ctx, &routingv1.UnpublishRequest{Cid: ref.Cid})
```

### Search

`SearchCIDs` and `SearchRecords` return a `StreamResult` that uses channels for non-blocking
consumption:

```go
import searchv1 "github.com/agntcy/dir/api/search/v1"

result, err := c.SearchRecords(ctx, &searchv1.SearchRecordsRequest{Query: "my-agent"})
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case resp := <-result.ResCh():
        fmt.Println(resp.Record)
    case err := <-result.ErrCh():
        log.Println("error:", err)
    case <-result.DoneCh():
        return
    }
}
```

### Naming — resolve and verify

```go
// Resolve a name (optionally versioned) to record references
resp, err := c.Resolve(ctx, "my-agent", "1.0.0")
for _, ref := range resp.Refs {
    fmt.Println(ref.Cid)
}

// Get verification info by CID
info, err := c.GetVerificationInfo(ctx, ref.Cid)

// Get verification info by name
info, err = c.GetVerificationInfoByName(ctx, "my-agent", "1.0.0")
```

### Events — real-time streaming

```go
import eventsv1 "github.com/agntcy/dir/api/events/v1"

// Listen to all events
result, err := c.ListenStream(ctx, &eventsv1.ListenRequest{})
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case resp := <-result.ResCh():
        event := resp.GetEvent()
        fmt.Printf("Event: %s - %s\n", event.Type, event.ResourceId)
    case err := <-result.ErrCh():
        log.Println("stream error:", err)
        return
    case <-result.DoneCh():
        return
    case <-ctx.Done():
        return
    }
}
```

Filter by event type or label:

```go
result, err := c.ListenStream(ctx, &eventsv1.ListenRequest{
    EventTypes: []eventsv1.EventType{
        eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
        eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED,
    },
    LabelFilters: []string{"/skills/AI"},
})
```

### Signing and verification

The Go SDK signs records locally using [cosign](https://github.com/sigstore/cosign). No
`dirctl` binary is required. Signatures and public keys are stored as referrers on the record.

```go
import (
    corev1 "github.com/agntcy/dir/api/core/v1"
    signv1 "github.com/agntcy/dir/api/sign/v1"
)

// Sign with a local key
resp, err := c.Sign(ctx, &signv1.SignRequest{
    RecordRef: ref,
    Provider: &signv1.SignRequestProvider{
        Request: &signv1.SignRequestProvider_Key{
            Key: &signv1.KeySignRequest{
                KeyRef: "/path/to/cosign.key",
            },
        },
    },
})

// Sign with OIDC (keyless via Sigstore Fulcio)
resp, err = c.Sign(ctx, &signv1.SignRequest{
    RecordRef: ref,
    Provider: &signv1.SignRequestProvider{
        Request: &signv1.SignRequestProvider_Oidc{
            Oidc: &signv1.OIDCSignRequest{},
        },
    },
})

// Verify locally (fetches referrers and verifies with cosign)
verifyResp, err := c.Verify(ctx, &signv1.VerifyRequest{
    RecordRef: ref,
})
fmt.Println("valid:", verifyResp.Success)

// Verify via the server (uses the server's cached result)
verifyResp, err = c.Verify(ctx, &signv1.VerifyRequest{
    RecordRef:  ref,
    FromServer: true,
})

// Fetch raw signatures and public keys
sigs, err := c.PullSignatures(ctx, ref)
keys, err := c.PullPublicKeys(ctx, ref)
```

### Sync

```go
import storev1 "github.com/agntcy/dir/api/store/v1"

// Create a sync policy between two Directory servers
syncID, err := c.CreateSync(ctx, "https://remote.example.com", []string{ref.Cid}, nil)

// List active syncs (streaming)
itemsCh, err := c.ListSyncs(ctx, &storev1.ListSyncsRequest{})
for item := range itemsCh {
    fmt.Println(item.SyncId)
}

// Get a specific sync
syncInfo, err := c.GetSync(ctx, syncID)

// Delete a sync
err = c.DeleteSync(ctx, syncID)
```

## Error handling

The client wraps gRPC status errors. Use `google.golang.org/grpc/status` and
`google.golang.org/grpc/codes` to inspect them:

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

_, err := c.Pull(ctx, ref)
if err != nil {
    st, ok := status.FromError(err)
    if !ok {
        log.Fatal("non-gRPC error:", err)
    }
    switch st.Code() {
    case codes.NotFound:
        fmt.Println("record not found")
    case codes.Unavailable:
        fmt.Println("server unavailable")
    case codes.PermissionDenied:
        fmt.Println("authentication failure")
    default:
        fmt.Printf("gRPC error %s: %s\n", st.Code(), st.Message())
    }
}
```

## Prerequisites

- Go 1.21 or higher
- A running Directory server — see [Quickstart](dir-quickstart.md) or
  [Kubernetes Deployment](dir-deployment-kubernetes.md)
- For SPIFFE auth modes: a running [SPIRE](https://spiffe.io/docs/latest/spire-about/) agent

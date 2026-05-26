# Agent Finder API — local testing guide

This branch (`poc/ai-catalog`) adds the AI Catalog projection on top of OASF
records and exposes the Agent Finder Specification's `GET /v1/agents`
endpoint via the new in-process HTTP gateway. This guide walks through
compiling `dirctl`, starting the daemon, pushing a sample agent, and
querying the endpoint with the supported filters.

The default daemon config embeds a local filesystem OCI store and turns
the HTTP gateway on, so no Docker, registry, or extra services are
required.

---

## 1. Prerequisites

- Go (matching the version in `server/go.mod`)
- [Task](https://taskfile.dev/) — `brew install go-task`
- `jq` and `curl` for poking at the JSON responses

## 2. Compile `dirctl`

From the repo root:

```bash
task cli:compile
```

This produces `.bin/dirctl` for the current OS/arch.

## 3. Start the daemon

```bash
./.bin/dirctl daemon start
```

`dirctl daemon start` embeds the reference `cli/cmd/daemon/daemon.config.yaml`
as its default, so no `--config` flag is needed for testing. State lands
under `~/.agntcy/dir/` (SQLite at `dir.db`, the OCI image layout at
`store/`). Wait for two log lines that confirm the gateway is up:

```
HTTP gateway configured  address=localhost:8889 grpc_endpoint=127.0.0.1:8888
HTTP gateway serving     address=localhost:8889
```

Leave the daemon running in this terminal; open a second shell for the
remaining steps.

## 4. Push a sample agent

The repo ships fixtures under `tests/e2e/shared/testdata/`. `record_100.json`
is a v1.0.0 agent with both A2A and MCP module entries — perfect for
exercising the multi-module catalog projection.

```bash
./.bin/dirctl push tests/e2e/shared/testdata/record_100.json
```

Wait roughly one reconciler tick (≤ 60 s) and confirm it has been
indexed:

```bash
./.bin/dirctl search --name '*'
```

The daemon log should also show:

```
Successfully indexed record  cid=baeareiabbog2umgduqhlcb64fzt6adn34kblzvru3fdzkl75hjhwt6h3da
```

## 5. Query `GET /v1/agents`

The endpoint accepts the Agent Finder filter grammar (Appendix A of the
spec):

```
filter   = clause { WS+ "AND" WS+ clause }
clause   = field "=" value
field    = displayName | type | publisherId | createdAfter | updatedAfter
value    = token { "," token }     # comma-OR within a field
```

Logical `AND` across fields, comma-OR within one field's value list, each
field at most once. The HTTP gateway translates the URL into the gRPC
service, so any `INVALID_ARGUMENT` error surfaces as HTTP 400.

### 5.1 List everything

```bash
curl -s 'http://127.0.0.1:8889/v1/agents' | jq .
```

### 5.2 Filter by module media type

Return only entries whose projection exposes an A2A card or MCP server:

```bash
curl -s --get 'http://127.0.0.1:8889/v1/agents' \
  --data-urlencode 'filter=type=application/mcp-server+json,application/a2a-agent-card+json' \
  | jq .
```

Expected shape: a single result with `media_type: "application/ai-catalog+json"`
(container projection because the record has two known modules), and a
nested `data.entries` array containing one leaf entry per module —
`application/a2a-agent-card+json` and `application/mcp-server+json`,
each with its own URN suffix (`…:a2a`, `…:mcp`).

### 5.3 Filter by display name (substring, case-insensitive)

```bash
curl -s --get 'http://127.0.0.1:8889/v1/agents' \
  --data-urlencode 'filter=displayName=burger' \
  | jq '.results[].display_name'
```

### 5.4 Combine filters with `AND`

```bash
curl -s --get 'http://127.0.0.1:8889/v1/agents' \
  --data-urlencode 'filter=displayName=burger AND type=application/mcp-server+json' \
  | jq .
```

### 5.5 Time-window filter

`createdAfter` and `updatedAfter` both resolve to a strict `>` comparison
on the OASF `created_at` field — OASF records are content-addressed and
never logically update, so the two filters are deliberate aliases:

```bash
curl -s --get 'http://127.0.0.1:8889/v1/agents' \
  --data-urlencode 'filter=createdAfter=2024-01-01T00:00:00Z' \
  | jq '.results | length'
```

## 6. Fetch the well-known AI Catalog

The directory also publishes an AI Catalog document at the RFC 8615
well-known URI. It carries the host descriptor and a set of pre-built
collections — convenience links that point back at `GET /v1/agents`
filtered by media type — so external consumers can discover the dynamic
catalog without having to learn the filter grammar first.

```bash
curl -s 'http://127.0.0.1:8889/.well-known/ai-catalog.json' | jq .
```

Expected shape:

```json
{
  "specVersion": "1.0",
  "host": {
    "displayName": "AGNTCY Directory",
    "identifier": "agntcy.org"
  },
  "collections": [
    {
      "displayName": "A2A Agents",
      "url": "http://localhost:8889/v1/agents?filter=type%3Dapplication%2Fa2a-agent-card%2Bjson",
      "description": "Agents that publish an A2A agent card.",
      "mediaType": "application/ai-catalog+json"
    },
    {
      "displayName": "MCP Servers",
      "url": "http://localhost:8889/v1/agents?filter=type%3Dapplication%2Fmcp-server%2Bjson",
      ...
    },
    {
      "displayName": "AI Skills",
      "url": "http://localhost:8889/v1/agents?filter=type%3Dapplication%2Fai-skill%2Bmd",
      ...
    }
  ]
}
```

Notes:

- `entries` is intentionally absent (or empty) for now: there is no
  publish/unpublish write path yet, so the well-known document is a
  pure discovery surface (host + collections).
- The `url` on each collection is absolute and built from the gateway's
  configured `listen_address`, so the values are clickable in any
  HTTP client and `curl`-able verbatim — try one to round-trip back
  into `GET /v1/agents`:

```bash
curl -s "$(curl -s 'http://127.0.0.1:8889/.well-known/ai-catalog.json' \
  | jq -r '.collections[] | select(.displayName == "MCP Servers") | .url')" \
  | jq '.results[].display_name'
```

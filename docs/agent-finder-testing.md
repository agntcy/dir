# Agent Finder API — local testing guide

This branch (`poc/ai-catalog`) adds the AI Catalog projection on top of OASF
records and exposes four discovery endpoints via the new in-process HTTP
gateway:

| Endpoint                              | Returns                                    |
| ------------------------------------- | ------------------------------------------ |
| `GET /v1/agents`                      | Paginated list of `CatalogEntry` summaries |
| `GET /v1/agents/{cid}`                | A single `CatalogEntry` (REST-symmetric)   |
| `GET /v1/agents/{cid}/export?format=` | Full record in any `dirctl export` format  |
| `GET /.well-known/ai-catalog.json`    | Host descriptor + pre-built collections    |

This guide walks through compiling `dirctl`, starting the daemon, pushing a
sample agent, and exercising each endpoint with `curl`.

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

## 6. Fetch a single agent by CID (`GET /v1/agents/{cid}`)

`GET /v1/agents/{cid}` returns the same `CatalogEntry` shape you'd see
inside `results[]` from §5 — just on its own URL with its own cache
key. This is the "drill in" path for any UI that lists, picks a row,
and wants the row's detail view without re-paging.

The full OASF record (and any other `dirctl export` format) lives on
`GET /v1/agents/{cid}/export`, covered next in §7. The two surfaces
are intentionally split: the catalog view is a compact discovery
projection; the export view is the authoritative source document.

The examples below assume the CID printed in the daemon log after §4;
stash it in a shell variable so it's easy to reuse across §6 and §7:

```bash
CID=baeareiabbog2umgduqhlcb64fzt6adn34kblzvru3fdzkl75hjhwt6h3da
curl -s "http://127.0.0.1:8889/v1/agents/${CID}" | jq .
```

## 7. Export an agent in any `dirctl export` format (`GET /v1/agents/{cid}/export`)

`GET /v1/agents/{cid}/export` returns the full record translated into
the requested format — **byte-identical** to the bytes
`dirctl export --format=<X>` would write to stdout. Same translator,
same indentation, same trailing newline. The server picks an
appropriate `Content-Type` per format and writes the bytes straight
into the response body via `google.api.HttpBody`.

The `?format=` query parameter mirrors `dirctl export --format=`.
Omit it and you get the canonical OASF document.

### 7.0 Format prerequisites

Each non-OASF format projects from a specific OASF module. If a record
doesn't carry that module, the request will succeed up to format
resolution but the projection itself fails — see §7.5 for the exact
response shape. OASF is the canonical representation and works for
every record.

| `?format=`      | Reads OASF module                      | `Content-Type`                |
|-----------------|----------------------------------------|-------------------------------|
| _omitted_       | _(none — canonical record)_            | `application/json`            |
| `oasf`          | _(none — canonical record)_            | `application/json`            |
| `a2a`           | `integration/a2a`                      | `application/json`            |
| `mcp-ghcopilot` | `integration/mcp`                      | `application/json`            |
| `agent-skill`   | `core/language_model/agentskills`      | `text/markdown; charset=utf-8`|
| `skill`         | _alias for `agent-skill` (deprecated)_ | `text/markdown; charset=utf-8`|

The `record_100.json` fixture from §4 carries `integration/a2a` and
`integration/mcp` but **not** `core/language_model/agentskills`, so
§7.1, §7.2 and §7.4 work directly against it; §7.3 (`agent-skill`)
demonstrates the `FailedPrecondition` path against that same record.
Push a SKILL.md-derived record if you want to see §7.3 succeed
(`dirctl push <your-skill.md>` — outside the scope of this guide).

### 7.1 OASF (default)

```bash
CID=baeareiabbog2umgduqhlcb64fzt6adn34kblzvru3fdzkl75hjhwt6h3da

curl -s "http://127.0.0.1:8889/v1/agents/${CID}/export" \
  | jq .
```

### 7.2 A2A AgentCard

```bash
curl -s "http://127.0.0.1:8889/v1/agents/${CID}/export?format=a2a" | jq .
```

The body is an [A2A AgentCard](https://a2a.dev/spec/agent-card)
synthesised from the record's `integration/a2a` module.
`Content-Type: application/json`.

### 7.3 SKILL.md (text/markdown)

The `agent-skill` formatter returns Markdown, not JSON, and reads
from the record's `core/language_model/agentskills` module. Against
the `record_100.json` fixture — which has _no_ such module — the
server returns a 400 (gRPC `FailedPrecondition`) with a message that
names both the CID and the format:

```bash
curl -s -i "http://127.0.0.1:8889/v1/agents/${CID}/export?format=agent-skill"
# HTTP/1.1 400 Bad Request
# Content-Type: application/json
# ...
# {"code":9,"message":"record \"baeareiab...\" cannot be exported in \"agent-skill\" format: failed to translate record to SKILL.md: ...","details":[]}
```

This is the documented shape for "the record can't be projected to
the requested format" (`codes.FailedPrecondition` = HTTP 400) — it is
**not** a 5xx and should not page operators. The error message is
self-describing so an HTTP client can surface it directly.

Against a record that _does_ carry `core/language_model/agentskills`
(e.g. one pushed from a SKILL.md), the same call returns the markdown
with `Content-Type: text/markdown; charset=utf-8`:

```bash
SKILL_CID=<a CID for a SKILL.md-derived record>

# Headers — confirms the markdown media type
curl -s -D - -o /dev/null \
  "http://127.0.0.1:8889/v1/agents/${SKILL_CID}/export?format=agent-skill" \
  | grep -i content-type
# Content-Type: text/markdown; charset=utf-8

# Body — the SKILL.md verbatim
curl -s "http://127.0.0.1:8889/v1/agents/${SKILL_CID}/export?format=agent-skill" \
  | head -30
```

`skill` is a deprecated alias that resolves to `agent-skill` — useful
if you've already wired older CLI scripts.

### 7.4 GitHub Copilot MCP config

```bash
curl -s "http://127.0.0.1:8889/v1/agents/${CID}/export?format=mcp-ghcopilot" \
  | jq .
```

The body is a GitHub Copilot–shaped MCP config file with `servers`
and `inputs`. `Content-Type: application/json`.

## 8. Fetch the well-known AI Catalog

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

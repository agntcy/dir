# Directory CLI Command Reference

Command-line reference for `dirctl`. Install the CLI in the [Quickstart](dir-quickstart.md)
only; workflow examples live in [Features and Usage Scenarios](dir-features-scenarios.md).

## Global options

### Output formats

Most data-producing commands support `--output` / `-o`. Exceptions include `validate`, `context`, `auth`, `daemon`, `mcp serve`, and `version`.

| Format | Description | Use case |
|--------|-------------|----------|
| `human` | Formatted tables and colors (default) | Interactive use |
| `json` | Pretty-printed JSON | Debugging, single-record scripts |
| `jsonl` | One JSON object per line | Streaming, batch processing |
| `raw` | Values only (CIDs, IDs) | Shell pipelines |

Structured formats send data to **stdout** and messages to **stderr**, so piping to `jq`
works cleanly.

```bash
CID=$(dirctl push record.json -o raw)
dirctl routing search --skill "AI" -o json | jq '.[] | .record_ref.cid'
dirctl events listen -o jsonl | jq -c 'select(.type == "EVENT_TYPE_RECORD_PUSHED")'
```

### Server connection

```bash
dirctl --server-addr localhost:8888 routing list
export DIRECTORY_CLIENT_SERVER_ADDRESS=localhost:8888
dirctl --context dev routing list   # see Context Operations below
```

### Authentication

For remote servers, authenticate with OIDC before running commands. The full auth model,
gateway endpoints, and SPIFFE/SPIRE integration are documented in
[OIDC Authentication](dir-component-oidc-authentication.md).

| Command | Description |
|---------|-------------|
| `dirctl auth login` | OIDC login (PKCE, `--no-browser`, or `--device`) |
| `dirctl auth logout` | Clear cached credentials |
| `dirctl auth status` | Show current auth state |

```bash
dirctl auth login --oidc-issuer "https://idp.ads.outshift.io" --oidc-client-id "dirctl"
dirctl --auth-mode=oidc --server-addr ads.outshift.io:443 search --skill "AI"
```

Pre-issued tokens: `--auth-token` or `DIRECTORY_CLIENT_AUTH_TOKEN` for CI and automation.
Other modes: `--auth-mode=x509`, `jwt`, `token`, `tls`, `oidc`, `insecure`, or `none`. When
`--auth-mode` is empty, the client auto-detects OIDC (cached token, issuer, or client ID) and
falls back to insecure for local development. SPIFFE modes (`x509`, `jwt`, `token`) require an
explicit `--auth-mode`.

### Command groups

| Group | Commands |
|-------|----------|
| Setup | `init` |
| Daemon | `daemon start`, `stop`, `status` |
| Auth | `auth login`, `logout`, `status` |
| Context | `context list`, `current`, `set`, `show`, `validate` |
| Storage | `push`, `pull`, `delete`, `info` |
| Import / Export | `import`, `export` |
| Routing | `routing publish`, `unpublish`, `list`, `search`, `info` |
| Search | `search` |
| Security | `sign`, `verify`, `validate`, `naming verify` |
| Sync | `sync create`, `status`, `list`, `delete` |
| Events | `events listen` |
| MCP | `mcp serve` |
| Install | `install run`, `install uninstall` (or top-level `uninstall`), `install list` |
| Diagnostics | `doctor`, `version` |

### Getting help

```bash
dirctl --help
dirctl routing --help
dirctl routing search --help
```

## Setup

### `dirctl init [flags]`

Guided first-run setup for a new environment — run it once after installing
`dirctl`. It provisions the **OASF taxonomy extractor**: a small
sentence-transformer model (`all-MiniLM-L6-v2`, ~89 MB) plus the OASF taxonomy,
downloaded to a local asset directory. Those assets power local, **LLM-free**
record enrichment and natural language search that runs in-process — no
Python, no external inference service, no LLM API. The chosen OASF endpoint and
asset directory are saved to the `dirctl` config so other commands load the
provisioned assets automatically.

Provisioning is **opt-out** in an interactive terminal — the prompt defaults to
yes, so pressing Enter provisions. To avoid an unattended ~89 MB download, a
**non-interactive** run (no TTY, e.g. CI or a pipe) skips unless you pass
`--yes`. Re-running is idempotent: nothing is re-downloaded when the assets are
present and current, and the taxonomy is re-embedded only when it changed.

| Flag | Description | Default |
|------|-------------|---------|
| `--oasf-url` | OASF schema endpoint to pull the taxonomy from | `https://schema.oasf.outshift.com` |
| `--asset-dir` | Local directory for the provisioned assets | `~/.agntcy/oasf-sdk/extractor` |
| `--yes` / `-y` | Provision without prompting (required for non-interactive runs) | `false` |
| `--remove` | Remove the provisioned assets and clear the saved config | `false` |

```bash
# Interactive setup (prompts; Enter accepts)
dirctl init

# Non-interactive (CI): provision with defaults
dirctl init --yes

# Provision from a local OASF instance
dirctl init --oasf-url http://localhost:8080 --yes

# Tear down what init provisioned
dirctl init --remove --yes
```

## Diagnostics

### `dirctl doctor [flags]`

Runs connectivity and configuration checks against the configured Directory server and routing layer.

| Flag | Description | Default |
|------|-------------|---------|
| `--timeout` | Per-check timeout | `2s` |
| `--bootstrap-peer` | Bootstrap peer multiaddr to validate (repeatable) | - |

### `dirctl version`

Prints the `dirctl` build version.

## MCP Server

### `dirctl mcp serve`

Starts the built-in MCP server used by external AI tooling. Delegates to the
[`dir-mcp`](https://github.com/agntcy/dir-mcp) module.

## Agent Install

`dirctl install <cid-or-name>` pulls a record from the active Directory and
installs its artifacts — an MCP server entry and/or an Agent Skill, derived from
the record's OASF modules — directly into the configuration of detected AI coding
agents.

Which artifacts are installed is determined by the record's modules, not a flag:

- `core/language_model/agentskills` → an Agent Skill (`SKILL.md`), rendered into
  whatever persistent-instruction mechanism each agent supports (a native skill
  folder, a per-tool rules file, or a managed block in a shared instruction file).
- `integration/mcp` → an MCP server entry (`command`/`args`/`env`) placed under
  each agent's MCP servers key.
- `integration/a2a` → not installable into agent configs; the command errors and
  points you to `dirctl export`.

A multi-module record installs all applicable artifacts. The per-record identity
is the sanitized record name, so re-running `install` with a newer version of the
same-named record replaces the old artifacts cleanly, and different records
coexist.

Supported agents: Claude Code, Claude Desktop, Cursor, VS Code (Copilot),
Windsurf, Cline, Roo Code, Gemini CLI, OpenCode, Zed, Continue, and Codex CLI.

Writes are atomic and surgical: only our own MCP entry, skill file/folder, or
delimited managed block is added/updated/removed — all of your existing
configuration is preserved.

### `dirctl install list`

Lists every supported agent, whether it is detected on this machine, and the
config files that install would touch. Makes no changes and does not contact the
Directory.

### `dirctl install <cid-or-name>` / `dirctl install run <cid-or-name>`

Pulls the record, derives its artifacts, prints the planned changes (per agent and
artifact, marked add / updated / unchanged), asks for confirmation, then installs.
`dirctl install <cid-or-name>` is shorthand for `dirctl install run <cid-or-name>`.
By default it acts on all detected agents. Detection is always required — an agent
is never installed into unless it is detected on this machine; an explicitly
requested agent that is not detected is reported as skipped.

The artifacts to install are determined entirely by the record's modules (above),
not by flags. A record carrying neither an MCP nor an Agent Skill module has
nothing installable: the command errors (making no changes) and, for an
A2A-only record, points you to `dirctl export`.

| Flag | Description | Default |
|------|-------------|---------|
| `--agents` | Agents to target: `all` (every detected agent) or a comma-separated list of agent IDs (e.g. `--agents claude-code,cursor`; repeatable) | `all` |
| `--dry-run` | Preview the plan without writing | `false` |
| `--yes` / `-y` | Skip the confirmation prompt | `false` |

Valid agent IDs: `claude-code`, `claude-desktop`, `cursor`, `vscode`, `windsurf`,
`cline`, `roo`, `gemini`, `opencode`, `zed`, `continue`, `codex` (see
`dirctl install list`). After completion, a summary lists every location added,
updated, removed, or skipped with its absolute path.

```bash
# Preview what installing a record would change
dirctl install cisco.com/agent:v1.0.0 --dry-run

# Install a record's artifacts into all detected agents
dirctl install cisco.com/agent:v1.0.0 --yes

# Install into specific agents only
dirctl install cisco.com/agent --agents claude-code,cursor
```

### `dirctl install uninstall <cid-or-name> [flags]`

Removes what `install` added for that record — its MCP entry and/or skill —
leaving all other content intact. Shares the same flags as install (`--agents`,
`--dry-run`, `--yes`). Idempotent: an agent with nothing of ours installed is
reported as unchanged, never an error.

`dirctl uninstall <cid-or-name>` is a top-level shorthand for
`dirctl install uninstall <cid-or-name>` (same flags and behavior).

```bash
dirctl install uninstall cisco.com/agent --yes
dirctl uninstall cisco.com/agent --agents cursor
```

## Daemon Operations

The daemon commands run a self-contained local directory server that bundles the gRPC apiserver and reconciler into a single process with embedded SQLite and a filesystem OCI store. All state is stored under `~/.agntcy/dir/` by default.

### `dirctl daemon start`

Starts the local directory daemon in the foreground. The process blocks until `SIGINT` or `SIGTERM` is received.

The daemon starts a gRPC apiserver on `localhost:8888` and runs all reconciler tasks (regsync, indexer, name resolution, signature verification) in-process. It uses SQLite for persistence and a local persistent OCI store, so no external dependencies (PostgreSQL, container registry, etc.) are required.

A PID file is written to the data directory to prevent multiple instances from running simultaneously.

Without `--config`, built-in defaults are used. When `--config` is provided, the file is read as the complete configuration.

| Flag | Description | Default |
|------|-------------|---------|
| `--data-dir` | Data directory for daemon state | `~/.agntcy/dir/` |
| `--config` | Path to daemon config file | Built-in embedded defaults |

??? example

    ```bash
    # Start the daemon (foreground, blocks until signal)
    dirctl daemon start

    # Start with a custom data directory
    dirctl daemon start --data-dir /path/to/data

    # Start with a custom config file
    dirctl daemon start --config /path/to/config.yaml

    # Run in the background using shell job control
    dirctl daemon start &
    ```

**Data directory layout:**

```text
~/.agntcy/dir/
├── dir.db          # SQLite database
├── store/          # Persistent OCI store
├── routing/        # DHT routing datastore
└── daemon.pid      # PID file for lifecycle management
```

**Configuration:**

The daemon ships with sensible built-in defaults. To customize, pass a YAML config file via `--config`. Relative paths in the config (e.g. `store`, `dir.db`) are resolved against `--data-dir`. Credentials can be set via environment variables prefixed with `DIRECTORY_DAEMON_` (e.g. `DIRECTORY_DAEMON_SERVER_DATABASE_POSTGRES_PASSWORD`).

??? example "Reference config file"

    ```yaml
    server:
      listen_address: "localhost:8888"
      oasf_api_validation:
        schema_url: "https://schema.oasf.outshift.com"
      store:
        provider: "oci"
        oci:
          local_dir: "store"
          registry_address: "localhost:5555"
          repository_name: "dir"
          auth_config:
            insecure: true
        verification:
          enabled: true
      routing:
        listen_address: "/ip4/0.0.0.0/tcp/8999"
        datastore_dir: "routing"
        gossipsub:
          enabled: true
      database:
        type: "sqlite"
        sqlite:
          path: "dir.db"
        postgres:
          host: "localhost"
          port: 5432
          database: "dir"
          ssl_mode: "disable"
      publication:
        scheduler_interval: 5m
        worker_count: 1
        worker_timeout: 30m
      naming:
        ttl: 168h


    reconciler:
      regsync:
        enabled: true
        interval: 1m
      indexer:
        enabled: true
        interval: 1m
      signature:
        enabled: true
        interval: 1m
        ttl: 168h
        record_timeout: 30s
      name:
        enabled: true
        interval: 1m
        ttl: 168h
        record_timeout: 30s
      local_registry:
        registry_address: "localhost:5555"
        repository_name: "dir"
        auth_config:
          insecure: true
      database:
        type: "sqlite"
        sqlite:
          path: "dir.db"
    ```

### `dirctl daemon stop`

Stops a running daemon by sending `SIGTERM` to the process recorded in the PID file. The command waits for the process to exit gracefully and cleans up the PID file.

| Flag | Description | Default |
|------|-------------|---------|
| `--data-dir` | Data directory for daemon state | `~/.agntcy/dir/` |

??? example

    ```bash
    # Stop the running daemon
    dirctl daemon stop

    # Stop a daemon using a custom data directory
    dirctl daemon stop --data-dir /path/to/data
    ```

### `dirctl daemon status`

Checks whether the daemon is currently running by inspecting the PID file.

| Flag | Description | Default |
|------|-------------|---------|
| `--data-dir` | Data directory for daemon state | `~/.agntcy/dir/` |

??? example

    ```bash
    # Check daemon status
    dirctl daemon status
    ```

    Example output when running:

    ```
    Daemon is running (PID 12345)
    ```

    Example output when stopped:

    ```
    Daemon is not running
    ```

## Context Operations

Starting with Directory v1.4.0, the context commands manage reusable `dirctl` client contexts. Contexts describe Directory endpoints and their client-side authentication, TLS, and SPIFFE settings.

The default config file is `~/.config/dirctl/config.yaml`, or `$XDG_CONFIG_HOME/dirctl/config.yaml` when `XDG_CONFIG_HOME` is set.

```yaml
current_context: prod
contexts:
  prod:
    server_address: gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://idp.example.com
    oidc_client_id: dirctl
  staging:
    server_address: staging.gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://staging.idp.example.com
    oidc_client_id: dirctl
```

Regular `dirctl` commands select a context in this order:

1. `--context <name>`
2. `DIRECTORY_CLIENT_CONTEXT`
3. `current_context` from the config file

After context selection, environment variables and explicit root flags such as `--server-addr`, `--auth-mode`, `--oidc-issuer`, and `--auth-token` override the selected context for that invocation.

!!! note "Sensitive values"

    `dirctl context show` redacts `auth_token` and `spiffe_token` values. Prefer environment variables or a secret manager for bearer tokens instead of storing long-lived tokens in `config.yaml`.

### `dirctl context list`

Lists configured contexts in sorted order and marks the persisted `current_context` with `*`. This command reads local config only; it does not contact a Directory server.

??? example

    ```bash
    dirctl context list
    ```

    Example output:

    ```text
    * prod
      staging
    ```

### `dirctl context current`

Prints the persisted `current_context` from the config file. It intentionally ignores one-off overrides such as `--context` and `DIRECTORY_CLIENT_CONTEXT`, so it matches the marker shown by `dirctl context list`.

| Flag | Description | Default |
|------|-------------|---------|
| `--quiet` | Print only the context name; print nothing when no context is set | `false` |
| `--json` | Print current context details as JSON | `false` |

??? example

    ```bash
    # Print the current context with a trailing newline
    dirctl context current

    # Prompt-friendly output
    dirctl context current --quiet

    # Machine-readable output
    dirctl context current --json
    ```

    Example JSON output:

    ```json
    {
      "name": "prod",
      "source": "current_context",
      "path": "/home/user/.config/dirctl/config.yaml"
    }
    ```

### `dirctl context set <name>`

Sets the persisted active context by updating `current_context` in the client config file. The target context must already exist.

??? example

    ```bash
    # Persist prod as the active context
    dirctl context set prod

    # Run one command against staging without changing current_context
    dirctl --context staging info <record-cid>
    ```

### `dirctl context show [name]`

Shows the effective client configuration for a context with sensitive values redacted. If `[name]` is omitted, the command uses the same context selection rules as other `dirctl` commands. Environment variables and explicitly changed root flags are included in the effective output.

??? example

    ```bash
    # Show the active effective context
    dirctl context show

    # Show a specific context
    dirctl context show prod

    # Preview a context with a one-off server override
    dirctl --server-addr localhost:8888 context show prod
    ```

    Example output:

    ```yaml
    name: prod
    source: current_context
    path: /home/user/.config/dirctl/config.yaml
    config:
      auth_mode: oidc
      oidc_client_id: dirctl
      oidc_issuer: https://idp.example.com
      server_address: gateway.example.com:443
      tls_skip_verify: false
    ```

### `dirctl context validate [name]`

Validates stored context definitions without applying environment variable overrides. It catches missing required fields, unsupported auth modes, and auth-mode-specific configuration mistakes before a context is used.

If `[name]` is provided, only that context is validated. If omitted, all configured contexts are validated in sorted order.

??? example

    ```bash
    # Validate all contexts
    dirctl context validate

    # Validate one context
    dirctl context validate prod
    ```

    Example output:

    ```text
    prod: ok
    staging: ok
    ```

## Storage Operations

### `dirctl push <file>`

Stores records in the content-addressable store. Has the following features:

- Supports OASF v1, v2, v3 record formats
- Content-addressable storage with CID generation
- Optional cryptographic signing
- Data integrity validation

??? example

    ```bash
    # Push from file
    dirctl push agent-model.json

    # Push from stdin
    cat agent-model.json | dirctl push --stdin

    # Push with signature
    dirctl push agent-model.json --sign --key private.key
    ```

### `dirctl pull <reference>`

Retrieves records by their Content Identifier (CID) or name reference.

**Supported Reference Formats:**

| Format | Description |
|--------|-------------|
| `<cid>` | Direct lookup by CID |
| `<name>` | Retrieves the latest version |
| `<name>:<version>` | Retrieves the specified version |
| `<name>@<cid>` | Hash-verified lookup (fails if resolved CID doesn't match) |
| `<name>:<version>@<cid>` | Hash-verified lookup for a specific version |

??? example

    ```bash
    # Pull by CID
    dirctl pull baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Pull by name (latest version)
    dirctl pull cisco.com/agent

    # Pull by name with specific version
    dirctl pull cisco.com/agent:v1.0.0

    # Pull with hash verification
    dirctl pull cisco.com/agent@bafyreib...
    dirctl pull cisco.com/agent:v1.0.0@bafyreib...

    # Pull signature and public-key referrer artifacts from the server
    dirctl pull <cid> --signature --public-key

    # Pull security scan reports for the record
    dirctl pull <cid> --scan-report
    ```

**Output and batch retrieval:**

`pull` defaults to `--output json` (records are primarily machine-consumed); `-o human|jsonl|raw` are also available. Use `--output-file` to write a single record's JSON to a file, or `--output-dir` with search filters to pull many records at once (one `<name>.json` per record).

**Referrer flags** — these extend the output with OCI referrer artifacts attached to the record:

| Flag | Description |
|------|-------------|
| `--public-key` | Include the public key referrer attached to the record |
| `--signature` | Include the signature referrer attached to the record |
| `--scan-report` | Include security scan reports for the record |

When any referrer flag is set, the response is a structured JSON object with a `record` key and one or more of `publicKeys`, `signatures`, `scanReports`.

**Output and batch retrieval:**

| Flag | Description | Default |
|------|-------------|---------|
| `--output-file` | Write the record JSON to a file instead of stdout (single record) | - |
| `--output-dir` | Directory for batch pull from search results (one JSON file per record) | - |
| `--limit` | Maximum number of records to pull in batch mode | `100` |
| `--all-versions` | Keep all versions in batch pull (default: latest per name wins) | `false` |

When `--output-dir` is used, at least one search filter is required. All standard search filters are available (`--name`, `--version`, `--module`, `--skill`, `--author`, etc.). A positional reference and `--output-dir` are mutually exclusive.

??? example

    ```bash
    # Single record JSON to stdout (default) or a file
    dirctl pull cisco.com/agent
    dirctl pull cisco.com/agent --output-file=./agent.json

    # Batch: pull every matching record into a directory
    dirctl pull --output-dir=./records/ --name "web*"
    dirctl pull --output-dir=./records/ --module "integration/mcp" --all-versions
    ```

**Hash Verification:**

The `@<cid>` suffix enables hash verification. This command fails if the resolved CID doesn't match the expected digest:

```bash
# Succeeds if cisco.com/agent:v1.0.0 resolves to bafyreib...
dirctl pull cisco.com/agent:v1.0.0@bafyreib...

# Fails with error if CIDs don't match
dirctl pull cisco.com/agent@wrong-cid
# Error: hash verification failed: resolved CID "bafyreib..." does not match expected digest "wrong-cid"
```

**Version Resolution:**

When no version is specified, commands return the most recently created record (by record's `created_at` field). This allows non-semver tags like `latest`, `dev`, or `stable`.

### `dirctl delete <cid>`

Removes records from storage.

??? example

    ```bash
    # Delete a record
    dirctl delete baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

### `dirctl info <reference>`

Displays metadata about stored records using CID or name reference.

**Supported Reference Formats:**

| Format | Description |
|--------|-------------|
| `<cid>` | Direct lookup by content address |
| `<name>` | Displays the most recently created version |
| `<name>:<version>` | Displays the specified version |
| `<name>@<cid>` | Hash-verified lookup |
| `<name>:<version>@<cid>` | Hash-verified lookup for a specific version |

??? example

    ```bash
    # Info by CID (existing)
    dirctl info baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Info by name (latest version)
    dirctl info cisco.com/agent --output json

    # Info by name with specific version
    dirctl info cisco.com/agent:v1.0.0 --output json
    ```

## Import Operations

Import records from external registries or local files into DIR. Supports automated batch imports with optional LLM enrichment and security scanning.

### `dirctl import [flags]`

Fetch and import records from registries or local sources.

**Import kinds (`--type`):**

| Type | Source | Required flags |
|------|--------|----------------|
| `mcp-registry` | HTTP MCP registry (e.g. v0.1 list API) | `--url` |
| `mcp` | Local JSON (one server or array) | `--file-path` |
| `a2a` | Local A2A AgentCard JSON | `--file-path` |
| `agent-skill` | Local Agent Skills directory with `SKILL.md` | `--file-path` |

**Configuration Options:**

| Flag | Environment Variable | Description | Required | Default |
|------|---------------------|-------------|----------|---------|
| `--config` | - | Path to a YAML import config file (enricher, scanner, authors, and more); values are overridden by command-line flags | No | - |
| `--type` | - | Import kind (`mcp-registry`, `mcp`, `a2a`, `agent-skill`) | No | - |
| `--url` | - | Registry base URL (required when `--type=mcp-registry`) | No | - |
| `--file-path` | - | Path to local JSON file or skill directory | No | - |
| `--output-cids` | - | Write imported CIDs to a file (one per line) | No | - |
| `--filter` | - | Registry-specific filters (key=value, repeatable) | No | - |
| `--limit` | - | Maximum records to import (0 = no limit) | No | 0 |
| `--dry-run` | - | Preview without importing; transformed records are written to `--output-dir` (one JSON file per record) so they can be reviewed and re-imported later via `dirctl push` or `dirctl import` | No | false |
| `--output-dir` | - | Directory to write per-record JSON files when `--dry-run` is set. Each record is written as `<cid>.record.json` | No | `./import-dry-run-<timestamp>` in the current working directory |
| `--debug` | - | Enable debug output (shows MCP source and OASF record for failures) | No | false |
| `--force` | - | Force reimport of existing records (skip deduplication) | No | false |
| `--sign` | - | Sign records after pushing (uses OIDC by default) | No | false |
| `--key` | - | Path to private key file for signing (requires `--sign`) | No | - |
| `--oidc-token` | - | OIDC token for non-interactive signing (requires `--sign`) | No | - |
| `--fulcio-url` | - | Sigstore Fulcio URL (requires `--sign`) | No | `https://fulcio.sigstore.dev` |
| `--rekor-url` | - | Sigstore Rekor URL (requires `--sign`) | No | `https://rekor.sigstore.dev` |
| `--server-addr` | DIRECTORY_CLIENT_SERVER_ADDRESS | DIR server address | No | localhost:8888 |

!!! note

    By default, the importer performs deduplication: it builds a cache of existing records (by name and version) and skips importing records that already exist. This prevents duplicate imports when running the import command multiple times. Use `--force` to bypass deduplication and reimport existing records. Use `--debug` to see detailed output including which records were skipped and why imports failed.

??? example

    ```bash
    # Import from MCP registry
    dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1

    # Import with debug output (shows detailed diagnostics for failures)
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --debug

    # Force reimport of existing records (skips deduplication)
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --force

    # Import with time-based filter
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --filter=updated_since=2025-08-07T13:15:04.280Z

    # Combine multiple filters
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --filter=search=github \
      --filter=version=latest \
      --filter=updated_since=2025-08-07T13:15:04.280Z

    # Limit number of records
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --limit=50

    # Preview without importing (dry run)
    # Records are written as <cid>.record.json into a timestamped directory
    # (default: ./import-dry-run-<timestamp>) so they can be reviewed and
    # re-imported later via `dirctl push` or `dirctl import`.
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --dry-run

    # Dry run with a custom output directory
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --dry-run \
      --output-dir=./out

    # Import and sign records with OIDC (opens browser for authentication)
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --sign

    # Import and sign records with a private key
    dirctl import --type=mcp-registry \
      --url=https://registry.modelcontextprotocol.io/v0.1 \
      --sign \
      --key=/path/to/cosign.key

    ```

**MCP Registry Filters:**

For the Model Context Protocol registry, available filters include:

- `search` - Filter by server name (substring match)
- `version` - Filter by version ('latest' for latest version, or an exact version like '1.2.3')
- `updated_since` - Filter by updated time (RFC3339 datetime format, e.g., '2025-08-07T13:15:04.280Z')

See the [MCP Registry API docs](https://registry.modelcontextprotocol.io/docs#/operations/list-servers#Query-Parameters) for the complete list of supported filters.

### Enrichment

The import command enriches records by mapping them to OASF skills and domains. Three enrichment methods are available, configured under the `enricher` key in the `--config` YAML file. When no config file is provided, LLM enrichment is used by default (azure:gpt-4o, 2 RPM).

#### Extractor enrichment (LLM-free, recommended)

Uses the local OASF sentence-transformer model provisioned by `dirctl init` to classify records in-process — no API key, no external service, no LLM runtime. This is the fastest option and works offline.

**Requires:** `dirctl init` to have been run at least once to provision the model assets.

```yaml
enricher:
  extractor: {}           # uses assets provisioned by dirctl init
```

To override the default asset location:

```yaml
enricher:
  extractor:
    oasf_url: https://schema.oasf.outshift.com   # optional
    asset_dir: /path/to/custom/assets             # optional
```

#### Static enrichment

Assigns the same fixed skills and domains to every imported record. No LLM or model assets required.

```yaml
enricher:
  static:
    skills:
      - name: natural_language_processing/text_completion
        id: 10201
    domains:
      - name: technology
        id: 1
```

#### LLM enrichment

Runs an LLM with tool-calling support against the OASF schema tools exposed by `dirctl mcp serve`. Produces the most semantically accurate skill and domain assignments but requires LLM credentials or a local runtime.

**Requirements:**

- `dirctl` binary (includes the built-in MCP server with `agntcy_oasf_get_schema_skills` and `agntcy_oasf_get_schema_domains` tools)
- An LLM with tool-calling support (GPT-4o, Claude, or compatible Ollama models)

**How it works:**

1. The enricher starts an MCP server using `dirctl mcp serve`
2. The LLM uses the `agntcy_oasf_get_schema_skills` tool to browse available OASF skills
3. The LLM uses the `agntcy_oasf_get_schema_domains` tool to browse available OASF domains
4. Based on the record description and capabilities, the LLM selects appropriate skills and domains

```yaml
enricher:
  llm:
    requests_per_minute: 5
    tool_host:
      model: azure:gpt-4o
      max_steps: 10
      mcp_servers:
        dir-mcp-server:
          command: dirctl
          args: [mcp, serve]
          env:
            OASF_API_VALIDATION_SCHEMA_URL: https://schema.oasf.outshift.com
            DIRECTORY_CLIENT_AUTH_MODE: insecure
    skills_prompt_template: ./prompts/skills.md    # optional custom prompt
    domains_prompt_template: ./prompts/domains.md  # optional custom prompt
```

See `cli/cmd/import/import.config.yaml` in the repository for a fully annotated reference configuration.

**Recommended LLM providers:**

- `azure:gpt-4o` — Azure OpenAI GPT-4o (recommended for speed and accuracy)
- `ollama:qwen3:8b` — Local Qwen3 via Ollama

**Environment variables for LLM providers:**

- Azure OpenAI: `AZURE_OPENAI_API_KEY`, `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT`

### Dry Run Output Directory

When `--dry-run` is set, the importer does **not** push records to the Directory node. Instead, every transformed record is written to disk as a separate JSON file, one per record. This produces a reviewable artifact set that is drop-in compatible with the existing import / push commands, so users who do not want the importer to push directly to a Directory node can review or modify the records first and upload them later.

**Output layout:**

- Target directory: value of `--output-dir`, or — when not set — `./import-dry-run-<timestamp>/` in the current working directory.
- The directory is created if it does not exist.
- Each record is written as `<cid>.record.json`, where `<cid>` is the record's content identifier. The naming scheme is deterministic and filesystem-safe so files can be diffed, signed, or selectively re-imported.

**Typical workflow:**

```bash
# 1. Dry run — write transformed records to ./out
dirctl import --type=mcp-registry \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --dry-run \
  --output-dir=./out

# 2. Review / diff / sign the per-record files on disk
ls ./out
# bafkrei...record.json
# bafkrei...record.json
# ...

# 3. Re-import the artifacts later via dirctl push (or another import flow)
for f in ./out/*.record.json; do
  dirctl push "$f"
done
```

### Signing Records During Import

Records can be signed during import using the `--sign` flag. Signing options work the same as the standalone `dirctl sign` command (see [Security & Verification](#security-verification)).

```bash
# Sign with OIDC (opens browser)
dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1 --sign

# Sign with a private key
dirctl import --type=mcp-registry --url=https://registry.modelcontextprotocol.io/v0.1 --sign --key=/path/to/cosign.key
```

## Export Operations

Export records from the Directory into formats consumable by external tools and agentic CLIs. Supports single-record export by CID/name and batch export from search results.

### `dirctl export <cid-or-name[:version][@digest]> [flags]`

Pull a record and transform it to the requested format.

**Supported Formats:**

| Format | Output | Description |
|--------|--------|-------------|
| `a2a` | `.json` | A2A AgentCard JSON for Agent-to-Agent protocol interop |
| `agent-skill` | `.md` | SKILL.md artifact for agentic CLI consumption (Cursor, Claude Code, etc.) |
| `mcp-ghcopilot` | `.json` | GitHub Copilot MCP configuration JSON |
| `mcp-claudecode` | `.json` | Claude Code MCP configuration JSON (`.mcp.json` `mcpServers` shape) |
| `mcp-cursor` | `.json` | Cursor IDE MCP configuration JSON (`.cursor/mcp.json` `mcpServers` shape) |

> **Note:** For raw OASF record JSON, use [`dirctl pull`](#dirctl-pull-reference) — it supports `--output-file`, `--output-dir`, and search filters for batch retrieval. `dirctl export` no longer accepts `--format=oasf`.

**Single-record Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--format` | Export format (see table above); **required** | - |
| `--output-file` | File path to write the exported data (default: stdout) | - |

**Batch Export Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--output-dir` | Directory for batch export from search results | - |
| `--all-versions` | Keep all versions (default: only latest semver per name) | `false` |
| `--limit` | Maximum number of records to export | `100` |

When `--output-dir` is used, at least one search filter is required. All standard search filters are available (`--name`, `--version`, `--module`, `--skill`, `--author`, etc.).

!!! note "Batch behaviour varies by format"

    For different export formats, the batch behaviour varies:

    - **a2a / oasf**: One file per record (`<name>.json`).
    - **agent-skill**: One subdirectory per skill (`<name>/SKILL.md`).
    - **mcp-ghcopilot**: All matched MCP servers are merged into a single `mcp.json` with combined `servers` and `inputs` maps.
    - **mcp-claudecode**: All matched MCP servers are merged into a single `mcp.json` with a combined `mcpServers` map.
    - **mcp-cursor**: All matched MCP servers are merged into a single `mcp.json` with a combined `mcpServers` map.

??? example

    ```bash
    # Export a single record as A2A AgentCard to stdout
    dirctl export baeareihdr6t7s6... --format=a2a

    # Export to a file (auto-appends extension if omitted)
    dirctl export my-agent:1.0 --format=a2a --output-file=./agent-card.json
    dirctl export my-agent:1.0 --format=agent-skill --output-file=./SKILL.md

    # Batch export A2A records to a directory
    dirctl export --output-dir=./exports/ --format=a2a --module "integration/a2a"

    # Batch export skills (creates subdirectories with SKILL.md)
    dirctl export --output-dir=./exports/ --format=agent-skill \
      --module "core/language_model/agentskills"

    # Batch export MCP servers (merged into a single config)
    dirctl export --output-dir=./exports/ --format=mcp-ghcopilot \
      --module "integration/mcp"

    # Export a single record as a Claude Code .mcp.json
    dirctl export my-mcp-server --format=mcp-claudecode --output-file=.mcp.json

    # Export a single record as a Cursor IDE mcp.json
    dirctl export my-mcp-server --format=mcp-cursor --output-file=.cursor/mcp.json
    
    # Export all versions instead of only the latest
    dirctl export --output-dir=./exports/ --format=a2a \
      --name "my-agent" --all-versions

    # Combine multiple search filters
    dirctl export --output-dir=./exports/ --format=a2a \
      --author "acme" --skill "natural_language_processing"
    ```

## Routing Operations

The routing commands manage record announcement and discovery across the peer-to-peer network.

### `dirctl routing publish <cid>`

Announces records to the network for discovery by other peers. The command does the following:

- Announces record to DHT network.
- Makes record discoverable by other peers.
- Stores routing metadata locally.
- Enables network-wide discovery.

??? example

    ```bash
    # Publish a record to the network
    dirctl routing publish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

### `dirctl routing unpublish <cid>`

Removes records from network discovery while keeping them in local storage. The command does the following:

- Removes DHT announcements.
- Stops network discovery.
- Keeps record in local storage.
- Cleans up routing metadata.

??? example

    ```bash
    # Remove from network discovery
    dirctl routing unpublish baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi
    ```

### `dirctl routing list [flags]`

Queries local published records with optional filtering.

The following flags are available:

- `--skill <skill>` - Filter by skill (repeatable)
- `--locator <type>` - Filter by locator type (repeatable)
- `--domain <domain>` - Filter by domain (repeatable)
- `--module <module>` - Filter by module name (repeatable)
- `--cid <cid>` - List specific record by CID
- `--limit <number>` - Limit number of results

??? example

    ```bash
    # List all local published records
    dirctl routing list

    # List by skill
    dirctl routing list --skill "AI"
    dirctl routing list --skill "Natural Language Processing"

    # List by locator type
    dirctl routing list --locator "docker-image"

    # List by module
    dirctl routing list --module "runtime/framework"

    # Multiple criteria (AND logic)
    dirctl routing list --skill "AI" --locator "docker-image"
    dirctl routing list --domain "healthcare" --module "runtime/language"

    # Specific record by CID
    dirctl routing list --cid baeareihdr6t7s6sr2q4zo456sza66eewqc7huzatyfgvoupaqyjw23ilvi

    # Limit results
    dirctl routing list --skill "AI" --limit 5
    ```

### `dirctl routing search [query] [flags]`

Discovers records from other peers across the DHT network. Two modes are available:

| Mode | Invocation | Requires |
|------|-----------|----------|
| **Natural-language** | `dirctl routing search "free-text query"` | `dirctl init` (OASF extractor) |
| **Structured** | `dirctl routing search --skill "..." --domain "..."` | None |

#### Natural-language routing search

Pass a free-text phrase as a positional argument. The OASF extractor decomposes the phrase into skill and domain signals. Keyword signals are dropped because the DHT only indexes skills, domains, locators, and modules — there is no name or description index at the routing layer.

Each signal is queried against the DHT independently; the discovered window of results is then reordered by match score, best-first. This is not a global top-N: only the records returned within `--limit` are reordered.

```bash
dirctl routing search "Github MCP server that manages issues"
dirctl routing search "real-time fraud detection for banking"
```

Use `--schema-version` to restrict NL extraction to a specific OASF schema version:

```bash
dirctl routing search "code review agent" --schema-version 1.0.0
```

Use `--verbose` to print the extracted signals to stderr:

```bash
dirctl routing search "fraud detection for banking" --verbose
# stderr output:
# [nl-routing-search] signals extracted (2 usable of 3 total):
#   RECORD_QUERY_TYPE_SKILL   fraud_detection
#   RECORD_QUERY_TYPE_DOMAIN  finance
# (keyword signal "banking" dropped — no DHT equivalent)
```

#### Structured routing search

Omit the positional argument and use filter flags. All flags are repeatable and combined with OR logic: a record matches if it satisfies at least `--min-score` of the specified queries.

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--skill` | Search by skill name (repeatable) | — |
| `--domain` | Search by domain name (repeatable) | — |
| `--locator` | Search by locator type (repeatable) | — |
| `--module` | Search by module path (repeatable) | — |
| `--limit` | Maximum results to return | `10` |
| `--min-score` | Minimum number of queries that must match | `1` |
| `--schema-version` | Restrict NL extraction to specific OASF schema version(s) (repeatable; NL mode only) | all versions |
| `--verbose` | Print extracted NL signals and per-signal details to stderr (NL mode only) | `false` |

**Output includes** record CID, provider peer information, match score, and the specific queries that matched.

??? example

    ```bash
    # Natural-language search across the network
    dirctl routing search "Github MCP server that manages issues"

    # Structured: search for AI records
    dirctl routing search --skill "AI"

    # Structured: search with multiple criteria (record must match at least 2)
    dirctl routing search --skill "AI" --skill "ML" --min-score 2

    # Structured: search by locator type
    dirctl routing search --locator "docker-image"

    # Structured: search by module
    dirctl routing search --module "runtime/framework"

    # Structured: combined filters
    dirctl routing search --skill "web-development" --limit 10 --min-score 1
    dirctl routing search --domain "finance" --module "validation" --min-score 2

    # Pipe results into sync
    dirctl routing search --skill "AI" --output json | dirctl sync create --stdin
    ```

### `dirctl routing info`

Shows routing statistics and summary information.

The output includes the following:

- Total published records count
- Skills distribution with counts
- Locators distribution with counts
- Helpful usage tips

??? example

    ```bash
    # Show local routing statistics
    dirctl routing info
    ```

## Search & Discovery

### `dirctl search [query] [flags]`

Search for records in the local index. Two modes are available depending on whether a positional argument is provided:

| Mode | Invocation | Requires |
|------|-----------|----------|
| **Natural-language** | `dirctl search "free-text query"` | `dirctl init` (OASF extractor) |
| **Structured** | `dirctl search --name "..." --skill "..."` | None |

#### Natural-language search

Pass a free-text phrase as a positional argument. The OASF extractor (provisioned by [`dirctl init`](#dirctl-init-flags)) decomposes the phrase into typed signals — skill names, domain names, and keywords. Each signal is queried against the index independently and concurrently; results are ranked by how many signals matched, with the best-matching records returned first.

```bash
dirctl search "Github MCP server that manages issues"
dirctl search "real-time fraud detection for banking" --format record
dirctl search "code review assistant" --limit 5 --format record
```

If the extractor has not been provisioned, the command fails with a prompt to run `dirctl init`.

Use `--verbose` to print signal decomposition and per-signal hit counts to stderr — useful for understanding why a query returns particular results:

```bash
dirctl search "fraud detection for banking" --verbose
# stderr output:
# [nl-search] signals extracted (3):
#   skill     fraud_detection                                       score=0.91
#   domain    finance                                               score=0.87
#   keyword   banking                                               score=0.74
# [nl-search] per-signal hits:
#   skill     fraud_detection                                       → 12 CIDs
#   domain    finance                                               → 31 CIDs
#   keyword   banking                                               → 8 CIDs
# [nl-search] ranked results (36 unique, 3 signals):
#   sha256:abc...  hits=3/3  signals=[skill:fraud_detection, domain:finance, keyword:banking]
#   ...
```

> **Note:** `--sort` is ignored in NL mode — results are always ranked by signal-hit-count.

#### Structured search

Omit the positional argument and use filter flags to query specific fields. All filter flags are repeatable and combinable.

**Filter flags:**

| Flag | Description |
|------|-------------|
| `--name` | Record name (supports wildcards, e.g. `web-*`) |
| `--version` | Record version (e.g. `v1.0.0`, `v1.*`) |
| `--skill-id` | Skill ID (e.g. `10201`) |
| `--skill` | Skill name (e.g. `natural_language_processing`) |
| `--locator` | Locator type (e.g. `docker-image`) |
| `--module` | Module path (e.g. `core/llm/model`) |
| `--domain-id` | Domain ID |
| `--domain` | Domain name |
| `--author` | Author name |
| `--schema-version` | OASF schema version |
| `--module-id` | Module ID |
| `--annotation` | Annotation key=value |
| `--verified` | Only verified records |
| `--trusted` | Only trusted records (signature verification passed) |
| `--safe` | Only records where all security scanners reported `is_safe=true` |
| `--scan-severity` | Only records whose highest scan severity meets or exceeds a threshold (`NONE`, `INFO`, `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`) |

**Output and pagination flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--format` | Output format: `cid` or `record` | `cid` |
| `--sort` | Result ordering for structured search: `relevance`, `popularity`, or `recency` | `recency` |
| `--limit` | Maximum results | `100` |
| `--offset` | Pagination offset | `0` |
| `--verbose` | Print NL signal decomposition and per-signal hit counts to stderr (NL mode only) | `false` |

??? example

    ```bash
    # Search by record name
    dirctl search --name my-agent

    # Search by version
    dirctl search --version v1.0.0

    # Search by skill ID
    dirctl search --skill-id 10201

    # Complex search with multiple criteria
    dirctl search --limit 10 --offset 0 \
      --name my-agent \
      --skill "Text Completion" \
      --locator docker-image \
      --format record

    # Sort by most recently published
    dirctl search --skill "code-review" --sort recency

    # Sort by most popular (pull frequency)
    dirctl search --domain finance --sort popularity

    # Security scan filters
    dirctl search --safe
    dirctl search --scan-severity HIGH
    dirctl search --safe --scan-severity MEDIUM
    ```

## Security & Verification

### Security Scanning

The Directory reconciler automatically runs security scanners against records and stores the results as OCI referrers. Scan results are indexed in the local database so they can be queried as search filters. Three scanners are supported: [mcp-scanner](https://cisco-ai-defense.github.io/docs/mcp-scanner) for MCP server source code, [skill-scanner](https://cisco-ai-defense.github.io/docs/skill-scanner) for agent skill bundles, and [a2a-scanner](https://github.com/cisco-ai-defense/a2a-scanner) for A2A AgentCards.

**Pull scan reports for a record:**

```bash
dirctl pull <cid> --scan-report
```

The response includes a `scanReports` array. Each entry covers one scanner type and contains:

| Field | Description |
|-------|-------------|
| `scanner_type` | Scanner that produced this report (`SCANNER_TYPE_MCP`, `SCANNER_TYPE_SKILL`, `SCANNER_TYPE_A2A`) |
| `scanner_version` | Version of the scanner binary |
| `scanned_at` | RFC 3339 timestamp of when the scan ran |
| `is_safe` | `true` if the scanner found no security issues |
| `max_severity` | Highest severity across all findings (`SEVERITY_NONE`, `SEVERITY_INFO`, `SEVERITY_MEDIUM`, `SEVERITY_HIGH`, `SEVERITY_CRITICAL`) |
| `analyzers` | Analyzer modules that were invoked |
| `findings` | List of individual findings (rule ID, severity, message, location, remediation) |

**Filter records by scan status:**

```bash
# Records where all scanners reported is_safe=true
dirctl search --safe

# Records whose highest severity meets or exceeds a threshold
dirctl search --scan-severity HIGH
dirctl search --scan-severity MEDIUM

# Combine with other filters
dirctl search --name "cisco.com/*" --safe
dirctl search --safe --scan-severity MEDIUM
```

A record appears in `--safe` results only when at least one scanner has run and no scanner has reported `is_safe=false`. Records where all scanners were skipped (no source repo, no skill bundle, no A2A AgentCard) are not included.

### Name Verification

Record name verification proves that the signing key is authorized by the domain claimed in the record's name field.

**Requirements:**

- Record name must include a protocol prefix: `https://domain/path` or `http://domain/path`
- A JWKS file must be hosted at `<scheme>://<domain>/.well-known/jwks.json`
- The record must be signed with the private key corresponding to a public key present in that JWKS file

**Workflow:**

1. Push a record with a verifiable name.

    ```bash
    dirctl push record.json --output raw
    # Returns: bafyreib...
    ```

2. Sign the record (triggers automatic verification).

    ```bash
    dirctl sign <cid> --key private.key
    ```

3. Check verification status using [`dirctl naming verify`](#dirctl-naming-verify-reference).

### `dirctl sign <cid> [flags]`

Signs records for integrity and authenticity. When signing a record with a verifiable name (e.g., `https://domain/path`), the system automatically attempts to verify domain authorization via JWKS. See [Name Verification](#name-verification) for details.

??? example

    ```bash
    # Sign with private key
    dirctl sign <cid> --key private.key

    # Sign with OIDC (keyless signing; default when --key is omitted)
    dirctl sign <cid> --fulcio-url https://fulcio.example.com

    # Sign with a pre-issued OIDC token (non-interactive)
    dirctl sign <cid> --oidc-token "$OIDC_TOKEN"
    ```

### `dirctl naming verify <reference>`

Verifies that a record's signing key is authorized by the domain claimed in its name field. Checks if the signing key matches a public key in the domain's JWKS file hosted at `/.well-known/jwks.json`.

**Supported Reference Formats:**

| Format | Description |
|--------|-------------|
| `<cid>` | Verify by content address |
| `<name>` | Verify the most recently created version |
| `<name>:<version>` | Verify a specific version |

??? example

    ```bash
    # Verify by CID
    dirctl naming verify bafyreib... --output json

    # Verify by name (latest version)
    dirctl naming verify cisco.com/agent --output json

    # Verify by name with specific version
    dirctl naming verify cisco.com/agent:v1.0.0 --output json
    ```

    Example verification response:

    ```json
    {
    "cid": "bafyreib...",
    "verified": true,
    "domain": "cisco.com",
    "method": "jwks",
    "key_id": "key-1",
    "verified_at": "2026-01-21T10:30:00Z"
    }
    ```

### `dirctl verify <record-cid> [flags]`

Verifies a record signature. Signatures are fetched from the directory.

| Flag | Description |
|------|-------------|
| `--key` | Public key file path, URL, or KMS URI |
| `--oidc-issuer` | OIDC issuer to match (supports regexp) |
| `--oidc-subject` | OIDC subject to match (supports regexp) |
| `--from-server` | Use the server's cached verification result |
| `--ignore-tlog` | Skip transparency log verification |

??? example

    ```bash
    # Verify any valid signature on the record
    dirctl verify <record-cid>

    # Verify against a specific public key
    dirctl verify <record-cid> --key /path/to/cosign.pub

    # Verify against an OIDC identity
    dirctl verify <record-cid> \
      --oidc-issuer https://github.com/login/oauth \
      --oidc-subject user@example.com
    ```

### `dirctl validate [<file>] [flags]`

Validates OASF record JSON from a file or stdin against the OASF schema. The JSON can be provided as a file path or piped from stdin (e.g., from `dirctl pull`). A schema URL must be provided via `--url` for API-based validation.

| Flag | Description |
|------|-------------|
| `--url <url>` | OASF schema URL for API-based validation (required) |

??? example

    ```bash
    # Validate a file with API-based validation
    dirctl validate record.json --url https://schema.oasf.outshift.com

    # Validate JSON piped from stdin
    cat record.json | dirctl validate --url https://schema.oasf.outshift.com

    # Validate a record pulled from directory
    dirctl pull <cid> --output json | dirctl validate --url https://schema.oasf.outshift.com

    # Validate all records (using shell scripting)
    for cid in $(dirctl search --output jsonl | jq -r '.record_cid'); do
      dirctl pull "$cid" | dirctl validate --url https://schema.oasf.outshift.com
    done
    ```

## Synchronization

### `dirctl sync create <url>`

Creates peer-to-peer synchronization.

??? example

    ```bash
    # Create sync with remote peer
    dirctl sync create https://peer.example.com
    ```

### `dirctl sync list`

Lists active synchronizations.

??? example

    ```bash
    # Show all active syncs
    dirctl sync list
    ```

### `dirctl sync status <sync-id>`

Checks synchronization status.

??? example

    ```bash
    # Check specific sync status
    dirctl sync status abc123-def456-ghi789
    ```

### `dirctl sync delete <sync-id>`

Removes synchronization.

??? example

    ```bash
    # Delete a sync
    dirctl sync delete abc123-def456-ghi789
    ```

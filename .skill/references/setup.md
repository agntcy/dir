# Setup: dirctl, local daemon, contexts, and diagnostics

Goal: get a working `dirctl` with either a **local workspace** (self-contained
daemon on `localhost:8888`) or a **remote directory** (context + OIDC auth).

## 1. Install dirctl

Check first: `dirctl version` (POSIX: `command -v dirctl`; PowerShell:
`Get-Command dirctl`).

Supported channels — offer the user the choice where more than one applies
(question tool if available):

- **Homebrew** (macOS / Linux only — not available on Windows):

  ```bash
  brew install agntcy/dir/dirctl
  ```

- **GitHub Releases** (all platforms; the only channel on Windows): download
  the binary for the user's OS/arch from
  <https://github.com/agntcy/dir/releases>.

  - Linux / macOS (amd64 + arm64): extract, `chmod +x dirctl`, place it on
    `PATH`.
  - Windows (**amd64 only** — there is no windows-arm64 build): save the
    binary as `dirctl.exe` in a directory on `PATH` (e.g.
    `%LOCALAPPDATA%\Programs\dirctl\`), adding that directory to the user
    `PATH` via Settings → Environment Variables if needed. No `chmod`.

Do **not** suggest building from source or repo-local task commands.

## 2. First-run provisioning: `dirctl init`

Run once after install. Provisions the OASF taxonomy extractor (a ~89 MB
sentence-transformer model + taxonomy) into `~/.agntcy/oasf-sdk/extractor` for
local, LLM-free record enrichment and free-text search. **Warn the user about
the download size before running.**

```bash
dirctl init          # interactive; Enter accepts
dirctl init --yes    # non-interactive (CI, piped shells) — required without a TTY
dirctl init --oasf-url http://localhost:8080 --yes   # from a local OASF instance
dirctl init --remove --yes                           # tear down provisioned assets
```

Idempotent: re-running re-downloads nothing when assets are present and
current. If `init` is not available in the installed version, skip it — it is
not required for core operations.

## 3. Security scanners (optional, required for security scans)

**Security scan results are only attached to records if the scanners are
installed.** Without them, records remain *unscanned* indefinitely and the
daemon log shows `executable file not found in $PATH`. The reconciler retries
on every tick, so scanners can be added later — but scans will not run until
they are present.

```bash
command -v mcp-scanner && command -v skill-scanner   # POSIX — skip if already present
Get-Command mcp-scanner; Get-Command skill-scanner   # PowerShell
```

Install if missing:

```bash
uv tool install cisco-ai-mcp-scanner    # installs mcp-scanner
uv tool install cisco-ai-skill-scanner  # installs skill-scanner
```

If `uv` is unavailable: `pip install cisco-ai-mcp-scanner cisco-ai-skill-scanner`.

**`mcp-scanner` requires LLM credentials** to run its behavioral alignment
check — this is optional but skipping it means MCP records will remain
*unscanned* even with `mcp-scanner` on `PATH`. `skill-scanner` has no such
requirement. See the
[mcp-scanner documentation](https://github.com/cisco-ai-defense/mcp-scanner)
for supported providers and required environment variables, then set them in
the daemon's environment before starting it.

See the verification reference for scan report details.

## 4. Local workspace: the daemon

The daemon is a self-contained local directory server (gRPC apiserver +
reconciler, embedded SQLite, filesystem OCI store). No PostgreSQL, no
registry, no external dependencies.

```bash
dirctl daemon status                 # check (inspects PID file)
dirctl daemon start                  # foreground; blocks until interrupted (Ctrl+C) or `daemon stop`
dirctl daemon stop                   # graceful shutdown via PID file, waits, cleans up
```

- Serves on `localhost:8888` — the default for all other commands, so a local
  workspace needs no `--server-addr` or context at all.
- State lives under `--data-dir` (default `~/.agntcy/dir/`;
  `%USERPROFILE%\.agntcy\dir\` on Windows): `dir.db` (SQLite), `store/`
  (OCI), `routing/` (DHT), `daemon.pid`.
- **`daemon start` runs in the foreground until stopped** — it never
  daemonizes itself. In agent environments, dedicate a terminal to it: launch
  it as a persistent background/async process in its own terminal and leave
  that terminal alone (run no other commands in it; its output is the daemon
  log). Run all subsequent commands — starting with `dirctl daemon status` to
  verify startup — from a different terminal.
- Manage the lifecycle only through `daemon start`/`stop`/`status` — never
  kill the process by signal directly: semantics differ across platforms and
  `stop` also cleans up the PID file.
- `--data-dir` and `--config` are persistent flags on the **`daemon` parent**,
  not on `start`. The PID file is `<data-dir>/daemon.pid`, so **`status` and
  `stop` need the same `--data-dir` as `start`** — without it they inspect the
  default directory and report "Daemon is not running" for a daemon that is
  running fine.

### Daemon configuration

Start from the generated reference config — never hand-write one:

```bash
dirctl daemon config init                      # writes <data-dir>/daemon.config.yaml
dirctl daemon config init --output ./daemon.config.yaml
dirctl daemon config init --force              # overwrite an existing file
dirctl daemon start --config ./daemon.config.yaml
```

`config init` emits a fully commented file covering `server` (listen address,
OASF validation URL, OCI store, routing/DHT, database, publication, HTTP
gateway, naming), `reconciler` (regsync, metrics, indexer, signature, name,
scan — including scanner CLI paths and endpoint-scan safety toggles), and
`runtime`.

Three behaviours to get right, none of which produce an error when you get
them wrong:

- **The config is not auto-discovered.** `<data-dir>/daemon.config.yaml` is
  only where `config init` writes by default; `daemon start` ignores it unless
  you pass `--config` explicitly. A daemon started without `--config` runs the
  embedded defaults even when that file exists — so never report a config as
  applied without checking the startup log.
- **`--config` replaces the defaults, it does not merge with them.** The file
  is read as the complete configuration, so a partial hand-written config
  silently loses everything it omits. Edit a `config init` copy instead.
- **Relative paths inside the config resolve against `--data-dir`** (`store`,
  `dir.db`, `routing`, `node.key`). Use absolute paths to pin a location.

Any setting can also be overridden by environment variable: `DIRECTORY_DAEMON_`
followed by the uppercased, underscore-delimited path, e.g.
`server.routing.republish_interval` →
`DIRECTORY_DAEMON_SERVER_ROUTING_REPUBLISH_INTERVAL`. List values such as
`..._BOOTSTRAP_PEERS` are comma-separated. This is the quickest way to tweak
one knob without touching a file. See the sync reference for when a config
file is genuinely required (e.g. P2P autosync).

### Throwaway test setups

To try a config, reproduce an issue, or demo a flow without touching the
user's real directory, give the daemon its own data directory and keep that
flag on **every** daemon command:

```bash
dirctl daemon config init --output /tmp/dirtest/daemon.config.yaml
# edit /tmp/dirtest/daemon.config.yaml
dirctl daemon start  --data-dir /tmp/dirtest --config /tmp/dirtest/daemon.config.yaml   # own terminal
dirctl daemon status --data-dir /tmp/dirtest                                            # another terminal
dirctl daemon stop   --data-dir /tmp/dirtest
```

- **Confirm the config actually took effect** by reading the startup log
  (`Server starting … address=…`), not by the fact that `start` did not error.
- **Change the ports when the user may already run a daemon**: the default
  `8888` (gRPC), `8889` (HTTP gateway), and `5555` (OCI registry) collide with
  a running instance and startup fails with `address already in use`. Check
  with `dirctl daemon status` first and never stop a daemon you did not start.
- Point clients at the test instance per-command with `--server-addr
  localhost:<port>`; do not repoint the user's current context.
- Tear down with `daemon stop --data-dir <dir>`, then delete the directory —
  it holds the whole state (`dir.db`, `store/`, `routing/`, `daemon.pid`).

## 5. Remote directories: contexts

Contexts live in `~/.config/dirctl/config.yaml` (or
`$XDG_CONFIG_HOME/dirctl/config.yaml`; on Windows under the user profile —
when unsure, `dirctl context show` prints the effective configuration rather
than guessing the path):

```yaml
current_context: prod
contexts:
  prod:
    server_address: gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://idp.example.com
    oidc_client_id: dirctl
```

```bash
dirctl context list           # sorted; * marks current_context
dirctl context current        # persisted current context
dirctl context current --quiet  # name only, nothing when unset (shell prompts, scripts)
dirctl context current --json   # persisted context details as JSON
dirctl context set <name>     # switch persisted context
dirctl context show [name]    # effective config, secrets redacted
dirctl context validate       # catch config mistakes before use
```

**There is no `context add` / `create` / `delete` subcommand** — the group is
`list`, `current`, `set`, `show`, `validate` only. To add a context, edit
`config.yaml` (create it if absent), then confirm with `context validate` and
`context set <name>`. Never claim to have "created a context" via a command.

**The parser rejects unknown keys.** A typo (`server-addr` for
`server_address`, `authMode` for `auth_mode`) is a hard decode error, not a
silently ignored field — so use exact names from this list:

| Key                                | Purpose                                                  |
| ---------------------------------- | -------------------------------------------------------- |
| `server_address`                   | Directory endpoint `host:port` — required for every usable context |
| `auth_mode`                        | `x509`, `jwt`, `jwt-tls`, `token`, `tls`, `oidc`, `insecure`, `none`; omit to auto-detect |
| `oidc_issuer`, `oidc_client_id`    | Used by `auth login` for OIDC contexts                    |
| `oidc_scopes`                      | List; changing it requires `auth login --force`           |
| `auth_token`                       | Pre-issued bearer token — prefer the env var instead      |
| `jwt_audience`                     | For `jwt` / `jwt-tls` modes                               |
| `tls_skip_verify`, `tls_cert_file`, `tls_key_file`, `tls_ca_file` | Custom PKI / mTLS         |
| `spiffe_socket_path`, `spiffe_token` | SPIFFE workload identity                                |
| `doctor.bootstrap_peers`           | List of peer multiaddrs `doctor` checks for this context  |

Pinning bootstrap peers to the context beats passing `--bootstrap-peer` every
time:

```yaml
contexts:
  prod:
    server_address: gateway.example.com:443
    auth_mode: oidc
    oidc_issuer: https://idp.example.com
    oidc_client_id: dirctl
    doctor:
      bootstrap_peers:
        - /dns4/routing.example.com/tcp/5555/p2p/12D3KooWExample
```

The same file also holds a top-level `extractor:` block (`oasf_url`,
`asset_dir`, `remote_addr`) written by `dirctl init` — that is what makes
natural-language search and local enrichment work. Leave it alone unless the
user wants to relocate or repoint the assets.

Selection order per invocation: `--context` flag → `DIRECTORY_CLIENT_CONTEXT`
→ `current_context`. Root flags (`--server-addr`, `--auth-mode`,
`--oidc-issuer`, `--auth-token`) override the selected context for that call.
Prefer `--context <name>` for one-off commands against another node over
switching the persisted context and forgetting to switch back.

Do not store long-lived tokens in `config.yaml` — prefer
`DIRECTORY_CLIENT_AUTH_TOKEN` or a secret manager.

## 6. Authentication (remote only)

Local daemon needs none (auto-detect falls back to insecure for local
development).

```bash
dirctl auth login     # uses the selected context's oidc_issuer / oidc_client_id
dirctl auth status
dirctl auth logout
```

Pass `--oidc-issuer` / `--oidc-client-id` only when the context does not
already carry them — with a configured context, bare `auth login` is correct.

- `auth login` uses OIDC PKCE (browser); `--no-browser` and `--device` exist
  for headless environments. `--callback-port` (default 8484) when that port is
  taken; `--timeout` (default 5m) for slow flows.
- **`auth login` is a no-op when a valid token is cached** — it prints
  `✓ Already authenticated as: <user>` and exits without re-authenticating.
  After changing `oidc_scopes` (or any context auth setting), you **must** use
  `dirctl auth login --force`, otherwise the old token stays in use and the
  change appears to have worked when it has not.
- CI / automation: pre-issued token via `--auth-token` or
  `DIRECTORY_CLIENT_AUTH_TOKEN`.
- Other modes need an explicit `--auth-mode`: `x509`, `jwt`, `token` (SPIFFE),
  `tls`, `oidc`, `insecure`, `none`.
- On auth errors: show the raw error and ask the user which mechanism they
  use. Never guess flags.

Reading `auth status` — report the token line, not the exit code of `login`:

| Token line     | Meaning                                                    |
| -------------- | ---------------------------------------------------------- |
| `Valid ✓`      | Healthy.                                                    |
| `Refreshed ✓`  | Healthy — cached token had expired and was refreshed.       |
| `Expired ✗`    | No usable refresh token; run `auth login` again.            |

### Authenticated ≠ allowed

`auth status` proves who you are, never what you may do. Behind a gateway,
methods are authorized per principal or group, so a token that reports
`Valid ✓` can still be refused:

```text
rpc error: code = PermissionDenied
```

Reads (`pull`, `lookup`, `search`, `list`, `verify`) are commonly granted
broadly while writes (`push`, `sign`, `routing publish`, `delete`, `sync`) are
restricted to administrators. When a write is denied, say which method was
denied and that it needs elevated rights on that deployment — do **not** retry,
re-login, or suggest `--force`, none of which change entitlements. Only
`Unauthenticated` and `Expired ✗` are login problems.

## 7. Verify the environment: `dirctl doctor`

Always finish setup with:

```bash
dirctl doctor                          # connectivity + configuration checks
dirctl doctor --timeout 5s             # slower networks
dirctl doctor --bootstrap-peer <maddr> # validate network bootstrap peers
```

Present failed checks to the user as a table (check | status | hint).

## Recommended local-workspace sequence

POSIX (bash/zsh):

```bash
command -v dirctl || <install via brew/releases>
dirctl init --yes            # optional, ~89 MB — ask first
command -v mcp-scanner   || uv tool install cisco-ai-mcp-scanner
command -v skill-scanner || uv tool install cisco-ai-skill-scanner
dirctl daemon start          # foreground until stopped — dedicate a terminal
dirctl daemon status         # from another terminal
dirctl doctor
```

Windows (PowerShell):

```powershell
Get-Command dirctl           # else install from GitHub Releases (.exe)
dirctl init --yes            # optional, ~89 MB — ask first
if (-not (Get-Command mcp-scanner -ErrorAction SilentlyContinue))   { uv tool install cisco-ai-mcp-scanner }
if (-not (Get-Command skill-scanner -ErrorAction SilentlyContinue)) { uv tool install cisco-ai-skill-scanner }
dirctl daemon start          # foreground until stopped — dedicate a terminal
dirctl daemon status         # from another terminal
dirctl doctor
```

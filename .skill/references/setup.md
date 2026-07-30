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

## 3. Local workspace: the daemon

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
- Custom config: `--config <yaml>` (relative paths resolve against
  `--data-dir`); env overrides use the `DIRECTORY_DAEMON_` prefix. Without
  `--config`, embedded defaults are used — see the sync reference for when a
  config file is required (e.g. P2P autosync).

## 4. Remote directories: contexts

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
dirctl context list          # sorted; * marks current_context
dirctl context current       # persisted current context
dirctl context set <name>    # switch persisted context
dirctl context show [name]   # effective config, secrets redacted
dirctl context validate      # catch config mistakes before use
```

Selection order per invocation: `--context` flag → `DIRECTORY_CLIENT_CONTEXT`
→ `current_context`. Root flags (`--server-addr`, `--auth-mode`,
`--oidc-issuer`, `--auth-token`) override the selected context for that call.

Do not store long-lived tokens in `config.yaml` — prefer
`DIRECTORY_CLIENT_AUTH_TOKEN` or a secret manager.

## 5. Authentication (remote only)

Local daemon needs none (auto-detect falls back to insecure for local
development).

```bash
dirctl auth login --oidc-issuer "https://idp.example.com" --oidc-client-id "dirctl"
dirctl auth status
dirctl auth logout
```

- `auth login` uses OIDC PKCE (browser); `--no-browser` and `--device` exist
  for headless environments.
- CI / automation: pre-issued token via `--auth-token` or
  `DIRECTORY_CLIENT_AUTH_TOKEN`.
- Other modes need an explicit `--auth-mode`: `x509`, `jwt`, `token` (SPIFFE),
  `tls`, `oidc`, `insecure`, `none`.
- On auth errors: show the raw error and ask the user which mechanism they
  use. Never guess flags.

## 6. Verify the environment: `dirctl doctor`

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
dirctl daemon start          # foreground until stopped — dedicate a terminal
dirctl daemon status         # from another terminal
dirctl doctor
```

Windows (PowerShell):

```powershell
Get-Command dirctl           # else install from GitHub Releases (.exe)
dirctl init --yes            # optional, ~89 MB — ask first
dirctl daemon start          # foreground until stopped — dedicate a terminal
dirctl daemon status         # from another terminal
dirctl doctor
```

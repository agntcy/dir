# dirctl Privacy Policy

`dirctl` is the command-line client for AGNTCY Directory. It also contains the
Directory daemon (`dirctl daemon`), which runs a Directory node on your
machine. AGNTCY runs no service behind `dirctl`: it has no telemetry and sends
nothing to AGNTCY. It talks only to the Directory server, identity provider and
other endpoints you configure or name, and what they do with the data is up to
whoever operates them.

## What it reads locally

- Command-line flags, and environment variables prefixed `DIRECTORY_CLIENT_`
  (and `DIRECTORY_DAEMON_` for the daemon).
- The client config file, `~/.config/dirctl/config.yaml` (under
  `$XDG_CONFIG_HOME` when that is set; on Windows `~` is `%USERPROFILE%`). It
  holds your named contexts: server address, authentication mode, OIDC issuer
  and client ID, TLS file paths, and, if you put them there, an `auth_token` or
  `spiffe_token`.
- Files you name: records to push, TLS CA, client certificate and key files,
  private keys passed to `dirctl sign` and `dirctl identity claim`, a trusted
  root for `dirctl verify`, import sources and import configs.
- The SPIFFE Workload API socket, when you choose SPIFFE authentication, to get
  your workload's X.509 or JWT identity.
- The configuration files of the coding agents `dirctl install` and
  `dirctl init` set up (see below).

## What it writes locally

- `dirctl auth login` caches your OIDC tokens under `~/.config/dirctl/`
  (`auth-token.json`, and per-issuer files in `tokens/`): the access token, the
  refresh token when one was issued, the issuer, the expiry, and the user name,
  user ID and email address from the provider. `dirctl auth logout` removes the
  cached token.
- `dirctl context set` and `dirctl init` update `config.yaml`. A token you
  save in a context is stored there in plain text; prefer an environment
  variable or a secret manager for long-lived tokens.
- `dirctl install`, `upgrade` and `uninstall` change the configuration of the
  coding agents on your machine, for example `~/.claude.json` and
  `~/.claude/skills/`, `~/.cursor/`, VS Code's `mcp.json`, and the Gemini,
  Copilot, OpenCode, Zed and Codex settings, in your home directory or the
  current project. They record what they installed in a manifest under
  `~/.config/dirctl/`. An installed record can add an MCP server that your coding
  agent then runs, so read what you are asked to confirm.
- `dirctl init` provisions the OASF taxonomy extractor under
  `~/.agntcy/oasf-sdk/extractor` (about 89 MB).
- `dirctl verify` caches Sigstore's trusted root under `~/.sigstore/tuf`.
- `dirctl network init` writes a peer identity private key, to
  `~/.agntcy/dir/generated.key` or the path you give.
- `dirctl daemon start` keeps its data under `~/.agntcy/dir/` (or `--data-dir`):
  the database, the records you push and their signatures and claims, the
  routing datastore, the peer identity key and a PID file. Records are stored
  as they were pushed.

On Linux and macOS, `dirctl` writes its config, token cache and keys readable
only by you (mode 0600, directories 0700). On Windows they inherit the
permissions of your user profile folder.

## What it sends, and to whom

- **Directory commands** (`push`, `pull`, `search`, `routing`, `sync`,
  `events`, `info`, `delete`, `export`, `identity`, `doctor`, `verify
  --from-server` and the rest) connect to the Directory server in your context. Without one they
  use `0.0.0.0:8888`, on your own machine. They send the records, queries and
  credentials you give them: an OIDC access token, a bearer token, a SPIFFE
  identity or a TLS client certificate. Records you push can be read by anyone
  who can read that server. Choose an authentication mode and TLS settings
  suitable for a remote server: with no TLS (`--auth-mode insecure`),
  credentials and records travel unencrypted.
- **`dirctl auth login`** talks only to the OIDC issuer you configure. It opens
  your browser at the issuer's sign-in page and receives the result on a
  loopback address (`http://localhost:8484/callback` by default), or, with
  `--device`, prints a code to enter on any device. It exchanges the result at the
  issuer's token endpoint, sending the client ID and a one-time PKCE verifier.
- **`dirctl sign`** signs on your machine.
  - With `--key`, the private key never leaves it. The signature and public key
    are then pushed to your Directory server with the record.
  - Without `--key` (keyless signing), you sign in with an OIDC provider
    (`https://oauth2.sigstore.dev/auth` by default). The ID token and a
    one-time public key go to the Sigstore certificate authority (Fulcio),
    which issues a certificate naming your identity, usually your email
    address. Unless you pass `--skip-tlog`, the signature and that certificate
    are also uploaded to the Rekor transparency log, which is public and
    permanent, and a timestamp request is sent to the timestamp authority. The
    endpoints are `fulcio.sigstore.dev`, `rekor.sigstore.dev` and
    `timestamp.sigstore.dev` by default; `--fulcio-url`, `--rekor-url` and
    `--timestamp-url` change them. The certificate is pushed to your Directory
    server with the record.
- **`dirctl verify`** checks signatures on your machine. For keyless signatures
  it fetches Sigstore's trusted root from `https://tuf-repo-cdn.sigstore.dev`
  by default; `--tuf-mirror-url` changes that, and `--trusted-root-path` uses a
  local file instead.
- **`dirctl init`** fetches the OASF taxonomy from the OASF schema server
  (`https://schema.oasf.outshift.com` by default, `--oasf-url` to change it),
  and downloads the embedding model it uses from Hugging Face.
  `dirctl validate --url` sends the record you give it to the schema server you
  name.
- **`dirctl import`** reads the source you point it at: a local file or
  directory, or the registry URL you pass with `--url`. With the `llm` enricher
  it sends record text such as names and descriptions to the model provider you
  configure, using the credentials in your environment. The `static` and local
  `extractor` enrichers do not.
- **`dirctl install`** and `dirctl upgrade` ask your Directory server for the
  records you name. They contact no one else.
- **`dirctl daemon start`** runs a node on your machine. It listens on
  `localhost:8888` for the API, on `0.0.0.0:8999` for peers, and on `:8889` for
  the HTTP gateway. With no bootstrap peers configured, which is the default,
  it connects to no one. Once you configure peers, it connects to them and
  announces the records you publish to the routing network, so other peers can
  find and sync them. Its background tasks look up the identity subjects named
  in the records it holds (DNS, HTTPS and DID documents), fetch Sigstore trust
  material to check signatures, and, if the scanners are installed and the scan
  task is enabled, run them. Those scanners can connect to the public endpoints
  a record lists and send record content to the model provider you configure.
  Runtime discovery, off by default, reads your local Docker or Kubernetes
  API.

## Credentials in output

Credentials are held in memory only while a command runs, apart from the files
described above. `dirctl auth login` prints where it cached the token, not the
token. `dirctl context show` prints `auth_token` and `spiffe_token` as
`<redacted>`.

## Telemetry

`dirctl` collects no usage analytics, crash reports or other telemetry, and
makes no network calls beyond the ones described above.

## Changes to this policy

This file is versioned with `dirctl` in the agntcy/dir repository; its git
history records every change. The WinGet manifest links to the copy at the
release tag.

## Contact

Open an issue at <https://github.com/agntcy/dir/issues>. Report security
vulnerabilities as described in
[SECURITY.md](https://github.com/agntcy/dir/blob/main/SECURITY.md).

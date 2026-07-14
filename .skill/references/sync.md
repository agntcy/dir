# Synchronization: mirror records between directories

Goal: replicate records from a remote directory into the connected one, and
manage the lifecycle of those syncs.

## Choosing a mode

| The user provides | Mode | Mechanism |
| --- | --- | --- |
| A directory **URL / API endpoint** (e.g. a hosted/SaaS instance) | **API sync** — `dirctl sync create` | Server-to-server replication over the Directory API / OCI store |
| Only a **libp2p peer ID** (`12D3KooW…`) | **P2P autosync** — daemon config patch | Records pulled over libp2p RPCs on DHT announcements (no OCI transport) |
| Wants to discover records on peers without copying them | `dirctl routing search` | DHT query |
| Wants own records discoverable | `dirctl routing publish` | DHT announce |

## API sync (SaaS / known endpoint)

Use when the remote directory is a publicly accessible instance with a known
API endpoint — e.g. a hosted/SaaS directory or any reachable
`host:port` gRPC address. The **connected server** performs the replication;
the CLI only manages sync lifecycle.

```bash
dirctl sync create <remote-directory-url>   # start syncing from a remote; returns a sync ID
dirctl sync list -o json                    # all syncs with IDs and states
dirctl sync status <sync-id>                # progress / health of one sync
dirctl sync delete <sync-id>                # stop and remove a sync — confirm with user first
```

Workflow:

1. `sync create` and capture the returned sync ID.
2. Poll `sync status <id>` until it reports an active/completed state — sync
   is asynchronous; initial replication of a large directory takes time. Warn
   the user and offer to check back rather than busy-polling.
3. Confirm arrival with a quick `dirctl search` for a known record from the
   remote.

Render `sync list` as a table:

| Sync ID | Remote | Status |
| --- | --- | --- |

## P2P autosync (peer ID only)

Use when the user can only name a **peer** (libp2p peer ID), not an API
endpoint. There is no CLI command for this: it is enabled by patching the
**daemon configuration**. Once active, records the allow-listed peer
announces on the DHT are pulled over **libp2p RPCs** — not the OCI transport
— and ingested locally with full parity to a normal push (record + referrers:
signatures, scan reports; content store + search index).

Target config shape (deny-by-default; off unless explicitly enabled):

```yaml
server:
  routing:
    # DHT connectivity is required to hear announcements — bootstrap against
    # the peer itself or a shared bootstrap node (multiaddr incl. peer ID):
    bootstrap_peers:
      - "/dns4/remote-dir.example.com/tcp/8999/p2p/<peer-id>"
    autosync:
      enabled: true
      peerlist:                       # allow-list of trusted source peers
        - peer: "12D3KooW...peerID1"
```

Apply it:

1. **Locate or create the daemon config file.** The daemon reads a config
   file only when started with `--config <file>`; otherwise it runs on
   embedded defaults (autosync off). A `--config` file is read as the
   **complete** configuration — start from the full reference daemon config
   (see `dirctl daemon start --help`), not a fragment containing only the
   autosync block.
2. **Patch, don't replace**: merge the `server.routing.autosync` (and
   `bootstrap_peers`) settings into the existing file, preserving everything
   else. `peerlist` is YAML-only — it is a list of objects and cannot be set
   via a `DIRECTORY_DAEMON_*` env var; `bootstrap_peers` may alternatively
   come from `DIRECTORY_DAEMON_SERVER_ROUTING_BOOTSTRAP_PEERS`
   (comma-separated).
3. **Restart the daemon** — config is read at startup only:
   `dirctl daemon stop`, then `dirctl daemon start --config <file>`
   (background it; see setup reference).
4. **Confirm**: after the peer next announces (or republishes), synced
   records show up in local `dirctl search`; `dirctl daemon` logs report
   "Autosync ingested record" per CID.

P2P autosync notes:

- Allow-list matching uses libp2p's **authenticated** peer ID — never a
  self-reported field. An invalid peer ID in `peerlist` fails at daemon
  startup rather than being silently skipped — surface that error verbatim.
- Pulled records are CID-checked and OASF-validated before ingestion;
  mismatches are rejected, not stored.
- A bare peer ID is not dialable by itself: the DHT needs a route. If the
  user has no multiaddr for the peer and no shared bootstrap node, ask for
  one — don't guess addresses.
- Autosync is continuous and event-driven (announcement-triggered), unlike
  API sync's explicit lifecycle; to stop it, set `enabled: false` (or remove
  the peer from `peerlist`) and restart the daemon.

## Notes (both modes)

- Synced records land in local storage and become searchable locally; their
  CIDs are unchanged (content-addressed).
- Deleting a sync (or disabling autosync) stops future replication; it does
  not delete already-synced records.
- Auth: the connected server performs the sync — remote credentials are a
  server-side concern. If `sync create` fails with auth errors, surface the
  raw error; do not invent flags.


# Synchronization: mirror records between directories

Goal: replicate records from a remote directory into the connected one, and
manage the lifecycle of those syncs.

## Unified sync process

1. **Source announces** — push and publish so the record is discoverable:

   ```bash
   CID=$(dirctl push record.json -o raw)   # store → CID
   dirctl routing publish "$CID"           # announce on the DHT
   ```

2. **Consumer discovers and syncs** — feed `routing search` straight into
   `sync create`; the CLI picks the right transport per peer automatically:

   ```bash
   dirctl routing search "<query>" -o json --limit 25 \
     | dirctl sync create --stdin
   ```

   Already know the remote's endpoint (a hosted/SaaS instance, a known
   `host:port`)? Skip discovery and sync straight from it:

   ```bash
   dirctl sync create <remote-directory-url>
   ```

3. **Track and confirm**:

   ```bash
   dirctl sync list -o json               # all syncs with IDs and states
   dirctl sync status <sync-id>           # progress / health of one sync
   dirctl search "<known record>"         # confirm it arrived locally
   dirctl sync delete <sync-id>           # stop/remove — confirm with user first
   ```

Sync is asynchronous — poll `status`, don't busy-wait; warn the user before a
large initial replication. Render `sync list` as a table:

| Sync ID | Remote | Status |
| --- | --- | --- |

## Notes: Directory API vs OCI-only transport

- `routing search` results carry each peer's address(es) (`Peer.Addrs`,
  prefixed `/dir/<address>` and/or `/oci/<registry>[/<repo>]`). Piping into
  `sync create --stdin` auto-prefers `/dir/` (credential-negotiated API sync)
  when present, falling back to a direct **anonymous** OCI pull when only
  `/oci/` is advertised — e.g. a peer whose Directory gRPC API is
  private/unreachable but whose OCI store points at a public registry (like
  GHCR). No `--registry` flag is needed for records discovered this way.
- Manual OCI override — use when a registry URL is known **out-of-band**
  (not from `routing search`):

  ```bash
  dirctl sync create --registry https://ghcr.io --repository <repo> --cids <cid1>,<cid2>
  ```

  (`--stdin --registry` together forces every piped CID to that one
  registry, ignoring discovered addresses.)
- Synced records land in local storage and become searchable locally; CIDs
  are unchanged (content-addressed).
- Deleting a sync stops future replication; it does not delete
  already-synced records.
- Auth is a server-side concern for API sync — on `sync create` auth errors,
  surface the raw error; do not invent flags. OCI-only pulls are anonymous by
  nature.


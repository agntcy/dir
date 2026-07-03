# Reconciler

The Reconciler is a standalone service that handles periodic reconciliation operations for the Directory. It runs as a separate process and can be scaled independently from the main API server.

## Architecture

The reconciler uses a task-based architecture where different reconciliation tasks can be registered and run at their configured intervals. This allows for:

- **Independent scaling**: The reconciler can be scaled separately from the API server
- **Separation of concerns**: Long-running background operations don't impact API performance
- **Extensibility**: New reconciliation tasks can be added without modifying the core service
- **Reliability**: Tasks are idempotent and handle partial failures gracefully

## Tasks

### Regsync Task

The regsync task handles synchronization from non-Zot registries. It:

1. Polls the database for pending sync operations that require regsync (non-Zot registries)
2. Negotiates credentials with the remote Directory node
3. Generates a regsync configuration file for the sync operation
4. Executes the `regsync once` command and waits for completion
5. Updates the sync status to COMPLETED or FAILED based on the result

### Indexer Task

The indexer task monitors the local OCI registry and indexes records into the search database. It:

1. Creates a snapshot of current registry tags (filtering to valid record CIDs)
2. Compares with the previous snapshot to detect new tags
3. For each new tag, pulls the record from the local store and validates it
4. Adds the record to the search database to enable search and filtering

### Name Task

The name task re-verifies DNS/name ownership of named records and caches results. It:

1. Queries the database for signed records with verifiable names that need verification (missing or expired)
2. For each record, retrieves the record name and public keys attached to the record
3. Verifies name ownership (e.g. via well-known JWKS at the record’s domain)
4. Stores the verification result (verified or failed) in the database for efficient API filtering

### Signature Task

The signature task verifies record signatures and caches results. It:

1. Queries the database for signed records with no or expired verification (per TTL)
2. For each record, collects signatures and public keys from the store
3. Verifies each signature using shared verification logic (key-based or OIDC)
4. Upserts verification results to the database

### Metrics Task

The metrics task refreshes computed usage metrics for locally known records. It:

1. Queries the search database for all locally known record CIDs
2. For each CID, queries the routing layer for the number of distinct announcing peers
3. Persists the provider count into the `record_usage_metrics` table for use in popularity ranking

The routing layer uses an embedded Badger store that does not support concurrent multi-process access. In standalone reconciler mode the task reaches the routing layer over gRPC rather than sharing the datastore directory.

### Scan Task

The scan task runs security scanners against record artifacts and persists the results. It:

1. Queries the database for records with no recent scan result (per configured TTL)
2. For each record, pulls the full record from the store
3. Runs each configured scanner (mcp-scanner, skill-scanner) independently
4. Pushes each scanner's `ScanReport` as an OCI referrer (`agntcy.dir.security.v1.ScanReport`) attached to the record CID
5. Upserts a summary row into the `scan_reports` table for efficient TTL-based filtering

Scanner failures for one runner do not block the others. Referrer storage failures are logged as warnings and do not abort the scan.

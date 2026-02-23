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

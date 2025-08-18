# Enable Post Sync Search Feature

We need to implement post-synchronisation processing to make newly synced records available to search service.

After zot syncs records, we need to create indexing and notification mechanisms to update local record stores after successful sync operations.

Please find the implementation options below.

## Option 1: Zot Log Parser

**Implementation**: Parse zot container logs in real-time using log streaming
**Trigger**: `"successfully synced image"` log events
**Components**: Log parser service, event handler, existing search indexer

Process:
1. Stream zot logs
2. Parse JSON events for `"successfully synced image"` messages  
3. Extract CID from `"image":"dir:<CID>"` field
4. Index new records using `ociStore.Pull()` + `db.AddRecord()`

Example Sync Run 1 (record synced):
```json
{"repository":"dir","registry":"http://agntcy-dir-zot.peer1.svc.cluster.local:5000","time":"2025-08-18T10:42:36.912971001Z","message":"syncing repo"}
{"repository":"dir","time":"2025-08-18T10:42:36.938385209Z","message":"syncing tags [baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u]"}
{"remote image":"agntcy-dir-zot.peer1.svc.cluster.local:5000/dir:baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","local image":"dir:baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","time":"2025-08-18T10:42:36.951385334Z","message":"syncing image"}
{"syncTempDir":"/var/lib/registry/dir/.sync/a50c8ffb-e96f-4f2f-b427-03e1e61f2189/dir","reference":"baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","time":"2025-08-18T10:42:36.975122209Z","message":"pushing synced local image to local registry"}
{"image":"dir:baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","time":"2025-08-18T10:42:36.998087501Z","message":"successfully synced image"}
{"image":"agntcy-dir-zot.peer1.svc.cluster.local:5000/dir:baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","time":"2025-08-18T10:42:36.998550167Z","message":"finished syncing image"}
{"repository":"dir","subject":"sha256:14358159f9649629a6b038ef7c75f0af3a5663bdd8185085d290b2ea33e6092e","time":"2025-08-18T10:42:37.001581376Z","message":"skipping oci references for image, already synced"}
{"repository":"dir","time":"2025-08-18T10:42:37.001615709Z","message":"finished syncing repository"}
{,"time":"2025-08-18T10:43:36.966297292Z","message":"finished syncing all repositories"}
```

Example Sync Run 2 (skip record sync):
```json
{"repository":"dir","registry":"http://agntcy-dir-zot.peer1.svc.cluster.local:5000","time":"2025-08-18T10:43:36.966605084Z","message":"syncing repo"}
{"repository":"dir","time":"2025-08-18T10:43:36.982008792Z","message":"syncing tags [baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u]"}
{"image":"agntcy-dir-zot.peer1.svc.cluster.local:5000/dir:baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","time":"2025-08-18T10:43:36.994119876Z","message":"skipping image because it's already synced"}
{"image":"agntcy-dir-zot.peer1.svc.cluster.local:5000/dir:baeareigsjuoxgfaq2jjxmtmxd5st2tkfzsxz3unwnaz5cs33rl4nqibj3u","time":"2025-08-18T10:43:36.994257042Z","message":"finished syncing image"}
{"repository":"dir","subject":"sha256:14358159f9649629a6b038ef7c75f0af3a5663bdd8185085d290b2ea33e6092e","time":"2025-08-18T10:43:36.998599292Z","message":"skipping oci references for image, already synced"}
{"repository":"dir","time":"2025-08-18T10:43:36.998667167Z","message":"finished syncing repository"}
{,"time":"2025-08-18T10:44:37.011177459Z","message":"finished syncing all repositories"}
```

## Option 2: Registry Tags Monitor

**Implementation**: Periodic polling service comparing registry state
**Trigger**: Scheduled checks (every 60s via sync worker poll interval)
**Components**: Registry API client, bloom filter cache, existing indexer

Process:
1. Poll local zot registry to list tags
2. Compute bloom filter of current tags vs cached previous state  
3. Detect new CIDs from tag differences
4. Index new records using `ociStore.Pull()` + `db.AddRecord()`
5. Update bloom filter cache for next iteration

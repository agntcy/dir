```sh
# start docker compoe
docker compose -f install/docker/docker-compose.yml up -d --build

# list running processes
grpcurl -plaintext -d '{}' localhost:8888 agntcy.dir.runtime.v1.DiscoveryService/ListProcesses
```

## Discussion: Runtime Process Discovery

### Problem Statement

The current implementation uses a container label (`agntcy.dir.record/cid`) to identify DIR-managed containers. This approach works well when containers are created through the `RuntimeService.CreateProcess` API, as we can explicitly set the label during container creation.

However, this creates a significant limitation: **we cannot discover containers that were started independently** (e.g., via `docker run` or `kubectl`) without DIR involvement.

### Design Goal

Enable discovery of OASF-described processes running in arbitrary runtime environments **without** requiring users to replace their existing tooling (Docker CLI, kubectl, etc.) with a DIR-specific process manager.

### Challenge

How can we inspect a running process so that it can self-identify as OASF-compliant and report: *"I am described by this OASF record"*?

### Proposed Solutions

#### Option 1: Well-Known Endpoint Convention

All OASF-compliant agents expose a standardized discovery endpoint:

```
GET http://<container-ip>:<port>/.well-known/oasf
```

**Response:**
```json
{
  "cid": "baeareiam...",
  "version": "0.8.0"
}
```

**Implementation:**
- SDKs embed a lightweight HTTP server that serves this endpoint on a well-known port (e.g., `1234`)
- The runtime periodically probes running containers to discover OASF-compliant processes
- Works for any container that implements the convention

**Pros:**
- Simple HTTP-based protocol
- Language-agnostic, easy to implement in any SDK
- No modification to existing orchestration tools

**Cons:**
- Requires port scanning or a predefined port convention
- Only works for network-accessible processes

#### Option 2: A2A Skill Requirement

For agents implementing the A2A (Agent-to-Agent) protocol, require an `oasf-compliant` skill that responds to discovery queries.

**Pros:**
- Leverages existing A2A infrastructure
- Natural fit for agent-based workloads

**Cons:**
- Limited to A2A-compatible agents
- Does not address non-agent workloads

### Open Questions

1. Should the well-known port be standardized (e.g., `1234`) or configurable via environment variable?
2. How do we handle containers that are not network-accessible?
3. Should we support multiple discovery mechanisms simultaneously?


### Options

DNS SD - TXT record that points to -X-


### Simplifications to Runtime Scopes

In K8s, the discovery is limited to NETWORK, so it must be a networked process.
In local runtimes, we need to be able to access (pid, fdid) for stdio.

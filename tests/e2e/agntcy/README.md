# Live AGNTCY identity E2E

This opt-in harness tests Directory's native storage, reconciliation, status and
search against the external AGNTCY verifier and real Identity Node. PostgreSQL
and Keycloak are disposable fixtures. Nothing is mocked. Node's HTTP listener is
fronted by a temporary TLS proxy; verification logic still runs in Node.

## Run

Use the Go version required by `tests/go.mod` and `cli/go.mod` (currently
1.27.2), Docker, and checkouts of the companion
[verifier](https://github.com/agntcy/identity/pull/182) and
[Node resolver fix](https://github.com/agntcy/identity/pull/181).
The suite uses the existing Go/Ginkgo dependencies; no Python environment is needed.

Build **Linux binaries matching Docker's architecture**. Run the following from
the Directory checkout, adjusting the companion checkout paths. Set `GOARCH` to
`amd64` for x86 Docker hosts or `arm64` for ARM Docker hosts (including Apple Silicon).

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go -C cli build -o /tmp/dirctl .
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go -C ../identity-verifier/integrations/directory-verifier \
  build -o /tmp/directory-verifier ./cmd/verifier
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go -C ../identity-node \
  build -o /tmp/identity-node ./cmd/node

# Run the Go test process on the Docker host, using the Linux binaries above.
export AGNTCY_E2E=1
export AGNTCY_E2E_DIRCTL=/tmp/dirctl
export AGNTCY_E2E_VERIFIER=/tmp/directory-verifier
export AGNTCY_E2E_NODE=/tmp/identity-node
export AGNTCY_E2E_ARTIFACTS="$(mktemp -d /tmp/agntcy-e2e-results.XXXXXX)"
go -C tests test ./e2e/agntcy -count=1 -timeout=10m -ginkgo.v \
  -ginkgo.json-report="$AGNTCY_E2E_ARTIFACTS/report.json"
```

Without `AGNTCY_E2E=1`, the suite compiles and skips. When enabled, missing binary
paths or an artifact directory fail the run instead of silently skipping it.
The default port range is 19300–19305; set `AGNTCY_E2E_PORT_BASE` to change it.
Docker must support `host.docker.internal` (the fixture provides a host-gateway
mapping for Linux). The ordered suite runs serially and manages its own disposable
services; it does not require an existing Directory deployment.

The fixture creates a unique Docker network, containers, temporary signing keys
and state. Ginkgo cleanup saves container logs and removes the containers/network
on success or failure, including failed setup. Test certificates and credentials
are disposable. Container logs and the Ginkgo JSON report remain in
`AGNTCY_E2E_ARTIFACTS`; report entries include the agent subject and tested CIDs.

## Assertions

- OIDC proof enrolls an agent and Node resolves its public identity metadata.
- A record without a claim does not appear in verified identity search.
- A native signed claim and valid badge for the exact record become verified.
- The JSON claim is a native OCI referrer whose manifest subject is the record.
- Another CID sharing the same Agent ID cannot reuse that badge.
- A valid badge cannot authorize an unrelated claim signer.
- Expired badges fail verification and verified search.
- Verifier outages cannot extend a signed observation beyond its deadline.
- Reconciliation and verified search recover when the verifier restarts.

The client unit suite separately asserts exactly one combined request per claim,
response-signature pinning, request/subject/CID binding, policy/check enforcement,
expiry and maximum age. The reconciler unit suite checks local claim signature
verification even when the configured verifier reports success, and regression
coverage for other resolver schemes. The verifier suite covers assertion-key
authorization and explicit control-only operation without badge API calls.

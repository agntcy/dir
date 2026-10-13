# Live AGNTCY identity E2E

This opt-in harness tests Directory's native storage, reconciliation, status and
search against the external AGNTCY verifier and real Identity Node. PostgreSQL
and Keycloak are disposable fixtures. Nothing is mocked. Node's HTTP listener is
fronted by a temporary TLS proxy; verification logic still runs in Node.

## Run

Use Go 1.27.1 and Docker. Build **Linux binaries matching Docker's architecture**;
for example, build inside `golang:1.27.1-bookworm` with the checkout mounted.

```sh
# Directory checkout
go -C cli build -o /tmp/dirctl .

# Companion AGNTCY Identity checkout
go -C integrations/directory-verifier build -o /tmp/directory-verifier ./cmd/verifier

# Node checkout. Until identity#181 merges, use codex/resolver-controller.
go build -o /tmp/identity-node ./cmd/node

# Directory checkout, on the Docker host
python3 -m venv /tmp/agntcy-e2e-venv
/tmp/agntcy-e2e-venv/bin/pip install -r tests/e2e/agntcy/requirements.txt
/tmp/agntcy-e2e-venv/bin/python tests/e2e/agntcy/run.py \
  --dirctl /tmp/dirctl --verifier /tmp/directory-verifier \
  --node /tmp/identity-node --artifacts /tmp/agntcy-e2e-results
```

The default port range is 19300–19305; change `--port-base` if necessary. Docker
must support `host.docker.internal` (the harness provides a host-gateway mapping
for Linux). Do not run against a shared deployment. The harness creates a unique
Docker network, containers, fixture signing keys and state, and removes its own
containers/network in `finally`. Test certificates and credentials are disposable.
Logs and the successful run's `results.json` remain under `--artifacts`.

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

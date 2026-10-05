---
icon: material/account-check
---

# Identity and Ownership Claims

A record can carry two signed claims: one for the identity of the record itself and one
for the owner behind it. A claim ties the record to a subject (a domain, a DID or a
SPIFFE ID) and is proven with a signature that anyone can check against key material the
subject publishes. The reconciler verifies every claim on a schedule and stores the outcome, so
a rotated key or a revoked trust bundle shows up on the next run.

This page describes the claim schema, how each kind of subject is verified, and how to read
the results. For the commands, see [CLI Reference — Identity](dir-cli-reference.md#dirctl-identity-claim-flags);
for a walkthrough, see [Usage Guide — Identity and Ownership Claims](dir-features-scenarios.md#identity-and-ownership-claims).

## How it works

```mermaid
sequenceDiagram
    participant P as Publisher
    participant S as Directory server
    participant R as Reconciler (identity task)
    participant K as Subject (DNS, HTTPS, DID, SPIFFE bundle)
    participant D as Clients (CLI, SDK, search)

    P->>S: Push record declaring agntcy.dir/owner
    P->>S: Push signed claim as a record referrer
    loop every reconciler.identity.interval
        R->>S: Read the record's claims
        R->>K: Look up the subject's current public keys
        R->>R: Check the claim and its signature
        R->>S: Store verified or failed
    end
    D->>S: GetIdentityStatus, or search --owner-verified
```

1. The record declares the subject it claims in its `agntcy.dir/identity` and
   `agntcy.dir/owner` annotations.
2. The publisher signs a claim for that subject with a private key whose public half the
   subject publishes, and attaches it to the record as an OCI referrer.
3. The reconciler's `identity` task verifies the claim: it resolves the subject's current
   keys and checks the signature against them. The result is stored in the server database.
4. The server reports the result through the `IdentityService` API, `dirctl identity status`
   and the search filters.

A claim is never verified at the time it is pushed. Until the task has run, the claim has
`no result`.

## Declaring the subject

A claim is only evaluated against the subject its record declares. The declaration is part of
the record, so it is covered by the record's CID and cannot be changed afterwards.

| Annotation | Role | Meaning |
|------------|------|---------|
| `agntcy.dir/identity` | `identity` | The record's own claimed identity |
| `agntcy.dir/owner` | `owner` | The entity that owns the record |

```json
{
  "name": "example.com/agents/research-assistant",
  "version": "v1.0.0",
  "annotations": {
    "agntcy.dir/identity": "did:web:example.com:agents:research-assistant",
    "agntcy.dir/owner": "dns:example.com"
  }
}
```

A record may declare one, both or neither. A claim for a role the record does not declare is
refused when it is created with `dirctl identity claim` or the SDK, and is ignored by the
reconciler if it is pushed by hand.

## Names

A record's `name` field is how it is referenced as `name`, `name:version` or
`name:version@cid`. A name is a label chosen by the publisher and is not verified. The
claims on this page are what prove who stands behind a record.

## Claim schema

A claim is the `agntcy.dir.identity.v1.Claim` message. It is stored as a
[record referrer](dir-component-store.md) of the record it is about, with the message encoded
as JSON in the referrer's data.

| Field | Type | Description |
|-------|------|-------------|
| `role` | `ClaimRole` | `CLAIM_ROLE_IDENTITY` or `CLAIM_ROLE_OWNER` |
| `recordCid` | string | CID of the record the claim is attached to |
| `subject` | string | The identity URI being claimed; must equal the record's annotation for the role |
| `signedAt` | string | RFC 3339 signing time |
| `expiresAt` | string, optional | RFC 3339 expiry; an expired claim fails |
| `signature` | string | Detached JWS (RFC 7515) in compact serialization |
| `certificate` | string, optional | Base64 DER X.509 certificate; only for `spiffe://` subjects |

The referrer type follows the role:

| Role | Referrer type |
|------|---------------|
| `identity` | `agntcy.dir.identity.v1.IdentityClaim` |
| `owner` | `agntcy.dir.identity.v1.OwnershipClaim` |

```json
{
  "role": "CLAIM_ROLE_OWNER",
  "recordCid": "baeareie...",
  "subject": "dns:example.com",
  "signedAt": "2026-10-02T10:15:00Z",
  "expiresAt": "2027-10-02T10:15:00Z",
  "signature": "eyJhbGciOiJFUzI1NiJ9..<signature>"
}
```

### What is signed

The signature covers a canonical JSON payload of five fields, in this order:

```json
{"record_cid":"...","role":"CLAIM_ROLE_OWNER","subject":"dns:example.com","signed_at":"...","expires_at":"..."}
```

`expires_at` is the empty string when the claim does not expire. Because the CID and the role
are signed, a claim cannot be replayed against another record or presented as the other role.
The `signature` and `certificate` fields are not part of the payload.

!!! warning

    The certificate is not covered by the signature. For that reason it is only accepted
    on a `spiffe://` claim, where it is validated against a trust bundle on its own, and a claim for any other subject that carries one fails.

### Signing keys

The signature algorithm is chosen from the public key, never from the JWS header, so a claim
cannot pick its own verification algorithm.

| Key | JWS algorithm |
|-----|---------------|
| ECDSA P-256 | `ES256` |
| ECDSA P-384 | `ES384` |
| Ed25519 | `EdDSA` |
| RSA, 2048 bits or more | `RS256` |

## Subjects and key publication

The scheme of the subject decides how its keys are found. A claim verifies when any one of
the keys the subject currently publishes validates its signature.

| Subject | Where the key comes from |
|---------|--------------------------|
| `dns:example.com`, or a bare `example.com` | TXT record at `_agntcy-key.example.com` |
| `https://example.com` | `https://example.com/.well-known/jwks.json` |
| `did:web:example.com` | `https://example.com/.well-known/did.json` |
| `did:web:example.com:agents:finance` | `https://example.com/agents/finance/did.json` |
| `did:key:z...` | The key encoded in the DID itself; no network access |
| `spiffe://example.com/agents/finance` | The SVID in the claim, validated against a configured trust bundle |

Any other scheme, such as `http://` or `mailto:`, is not supported and fails the claim.

### DNS

Publish one TXT record per key at `_agntcy-key.<domain>`:

```text
_agntcy-key.example.com.  300  IN  TXT  "v=akv1;key=<base64 DER SubjectPublicKeyInfo>"
```

The value is the standard base64 of the key's DER `SubjectPublicKeyInfo`, which is what
`openssl pkey -pubout -outform DER` produces. Records that are not a well-formed `akv1` record
are skipped.

### HTTPS and JWKS

For an `https://` subject the verifier fetches `/.well-known/jwks.json` from the subject's
host, over HTTPS only, and uses every key in the [JWK set](https://www.rfc-editor.org/rfc/rfc7517).
A subject with a user-info part, such as `https://example.com@evil.example`, is refused.

### DID

For `did:web` the document is fetched from the location the [did:web method](https://w3c-ccg.github.io/did-method-web/)
defines. A document is only accepted when its `id` equals the subject. Only keys listed under
`assertionMethod` count, whether embedded or referenced from `verificationMethod`; a key that
the document lists only as key material is not authorized to make assertions. Keys are read from
`publicKeyJwk`.

A `did:key` subject carries its key, so it needs no network access. P-256, P-384, Ed25519 and
RSA keys are supported.

### SPIFFE

A `spiffe://` claim carries the signer's X.509-SVID in `certificate`, and no key is published
anywhere. The verifier accepts the claim only when the certificate fulfils the following conditions:

- Chains to the trust bundle configured for the subject's trust domain.
- Has a URI SAN equal to the subject.
- Is within its validity period.

The claim carries only the leaf certificate, so its issuer must be a root in the bundle. A
trust domain with no configured bundle fails. See
[Configuration](#configuration) for how to provide the bundles.

### Fetch limits

Documents are fetched with an SSRF-safe client, because the URL comes from untrusted record
data. It speaks HTTPS only, refuses to connect to private, loopback, link-local and other
non-public addresses (checked after DNS resolution), follows at most three redirects, and
accepts responses of at most 1 MiB within a 10 second timeout.

## Verification

The `identity` task of the [reconciler](dir-architecture.md) checks claims on a fixed interval.
For each record that carries claims, and for each role, it does the following:

- Selects the claims.

    A claim whose `subject` is not the one the record declares for the role is skipped. Anyone can attach a claim to any record, so such a claim says nothing about the record, and it leaves no result behind.

- Runs the checks that need no key.

    The claim is for this record (`recordCid`), it is not expired, `signedAt` parses, and the certificate rule holds (a certificate if and only if the subject is `spiffe://`). A claim that fails these costs no network lookup.

- Resolves the subject's current keys for its scheme.
- Verifies the signature against those keys.

If several claims of one role pass the selection, a verified claim decides the result,
whoever else attached claims. Otherwise the failure of the claim with the newest `signedAt` is
reported. A `signedAt` in the future, or one that does not parse, counts as the oldest, so a
forged date cannot outrank an honest claim.

Within one run a lookup is made once per subject, however many records share it.

### Results

| State | Meaning |
|-------|---------|
| `no result` | No claim of this role has been verified yet, or the claim no longer applies |
| `verified` | A claim for the declared subject verified against the keys the subject publishes |
| `failed` | Every claim for the declared subject failed; the reason is kept (up to 1,024 characters) |

A failed result always names the subject the record declares, never one a claim chose.
If a claim is withdrawn, or no claim applies any more, its stored result is removed. If a
record's claims cannot be read at all, the stored results are left as they were.

### Unreachable subjects

A failed lookup is not the same as a wrong claim. If the subject cannot be reached, because
of a timeout, a refused connection, a DNS failure other than "no such name", or a 5xx, 408 or
429 response, a result that is already stored is left as it is, and nothing is written for
that run. The result's age is counted from when it was last written, so the grace lasts seven
days from the last run that reached the subject. The period is fixed and cannot be configured.
After that, the next run that still cannot reach the subject stores a `failed` result with the
lookup error.

Two cases get no grace:

- A claim that has no stored result yet fails at once if its subject is unreachable on the first run, and verifies on the first run that reaches it.
- An answer that says the subject publishes no usable key, a `404`, a "no such host" or a
  blocked address is not transient, so it fails the claim immediately.

### Stored result

Each result is one row per record and role in the server database.

| Column | Description |
|--------|-------------|
| `record_cid` | Record the claim is about (primary key with `role`) |
| `role` | `identity` or `owner` |
| `subject` | The subject the record declares |
| `status` | `verified` or `failed` |
| `error` | Reason for a failure |
| `verified_at` | When the claim was last checked |

## Reading the results

### `IdentityService`

`GetIdentityStatus` returns the last result of a record's claims. A claim that has not been
verified is left unset, which is not an error. An unknown record is `NotFound`.

| Message | Fields |
|---------|--------|
| `GetIdentityStatusRequest` | `cid`, or `name` with an optional `version` |
| `GetIdentityStatusResponse` | `identity`, `owner` (each an optional `ClaimVerification`) |
| `ClaimVerification` | `role`, `subject`, `status`, `error` (only when failed), `verified_at` |

`ClaimVerificationStatus` is `CLAIM_VERIFICATION_STATUS_VERIFIED` or
`CLAIM_VERIFICATION_STATUS_FAILED`.

`Resolve` turns a record name, with an optional version, into the CIDs of the matching
records, newest first. It is read-only like the rest of the service: results are only written
by the reconciler.

### CLI and SDK

```bash
# Create the claim, then read the result
dirctl identity claim --record <cid> --role owner --key owner.key
dirctl identity status <cid> --output json

# Resolve a name
dirctl identity resolve example.com/agents/research-assistant
```

In Go, `ClaimIdentity` and `ClaimOwnership` create claims, and `GetIdentityStatus` and
`Resolve` read results; see the [Go SDK](dir-sdk-go.md#identity-resolve-and-verify-claims).

### Search

Records can be searched by the subject of their claims and by whether a claim verified:

```bash
dirctl search --owner 'dns:example.com'        # by subject (wildcards allowed)
dirctl search --identity 'did:web:example.com:*'
dirctl search --owner-verified                 # owner claim verified
dirctl search --identity-verified              # identity claim verified
dirctl search --verified                       # same as --owner-verified
dirctl search --trusted --verified             # signed and owner-verified
```

`--identity-verified` and `--owner-verified` only filter when set. `--verified` is tri-state:
`--verified=false` matches records without a verified owner. A subject filter matches the
subject the record declares. See
[CLI Reference — Search](dir-cli-reference.md#dirctl-search-query-flags) for all flags.

## Configuration

Claim verification is a reconciler task and is disabled by default. The `dirctl daemon`
default configuration enables it.

```yaml
reconciler:
  identity:
    enabled: true
    interval: 1h          # how often every claim is verified again
    record_timeout: 1m    # time allowed for all the claims of one record
    spiffe_trust_bundles:
      - trust_domain: example.org
        bundle_file: /etc/agntcy/spiffe/example.org.pem
```

| Key | Environment variable | Default | Description |
|-----|----------------------|---------|-------------|
| `enabled` | `RECONCILER_IDENTITY_ENABLED` | `false` | Run the task |
| `interval` | `RECONCILER_IDENTITY_INTERVAL` | `1h` | Time between runs |
| `record_timeout` | `RECONCILER_IDENTITY_RECORD_TIMEOUT` | `1m` | Time allowed for one record |
| `spiffe_trust_bundles` | none, YAML only | empty | Trust bundle per SPIFFE trust domain |

Each `spiffe_trust_bundles` entry has a `trust_domain` and a `bundle_file` holding the trust
domain's PEM root certificates. The files are read on every run, so a rotated or revoked bundle
applies on the next one. A bundle that cannot be read only fails the claims of its own trust
domain; the other domains and the run are not affected. With no bundle, `spiffe://` claims fail.

## Security properties

- Anyone who can push a referrer can attach a claim to any record, so a result only comes from
  a claim that names the declared subject and verifies against that subject's own keys.
- The declared subject is part of the content-addressed record, so a claim cannot change who
  the record says it is about.
- The record CID and the role are signed, so a claim cannot be replayed against another record
  or as the other role.
- The signature algorithm is derived from the key, not from the claim, and RSA keys below 2048
  bits are rejected.
- A verified claim decides the result, whatever else is attached to the record.
- Claims are verified again on every run, so key rotation, DNS changes, a revoked trust bundle
  and expiry take effect on the next one.
- Keys are fetched with an SSRF-safe client, because the URLs come from record data.

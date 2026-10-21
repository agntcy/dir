# Identity: claim ownership and prove who an agent is

Goal: take a publisher from "I own example.com" to a record that shows as
**Verified** in search. A claim is a signed statement that a record's owner (or
its own identity) is a given **subject**. The reconciler checks it against the
keys that subject publishes *right now*, on every run.

Record names are labels the publisher picks and are not verified. Only a claim
proves who stands behind a record.

## The flow

1. Pick a subject (who you are) — table below.
2. Generate a key pair, publish the public half where the subject's scheme
   looks for it.
3. Declare the subject in the record's annotations, then push.
4. Sign the claim with `dirctl identity claim`.
5. Read the result with `dirctl identity status`.

Ask the user which subject they control before generating anything. Do not
guess a domain.

## 1. Pick a subject

| Subject | Key is found at | Good for |
| --- | --- | --- |
| `dns:example.com` (or bare `example.com`) | TXT record at `_agntcy-key.example.com` | **Default.** Anyone who controls a domain's DNS |
| `https://example.com` | `https://example.com/.well-known/jwks.json` | Teams that can serve a file on the domain |
| `did:web:example.com` | `https://example.com/.well-known/did.json` | Teams already using DIDs |
| `did:web:example.com:agents:finance` | `https://example.com/agents/finance/did.json` | One DID per agent |
| `did:key:z…` | The DID itself; nothing to publish | Tests, or a user who already has a did:key |
| `spiffe://example.org/…` | The X.509-SVID passed with `--cert` | Workloads in a SPIFFE trust domain the node trusts |
| `ans://v1.0.0.agent.example.com` | The ANS identity certificate passed with `--cert` | Agents registered with the Agent Name Service |

Any other scheme (`http://`, `mailto:`, …) fails. `spiffe://` claims only
verify if the node operator configured a trust bundle for that trust domain,
and `ans://` claims only if ANS is enabled with the log in
`trusted_log_hosts` — check with the operator before choosing either (see
policy reference for operator settings).

Roles: `--role owner` is checked against `agntcy.dir/owner` ("who owns this
record"); `--role identity` against `agntcy.dir/identity` ("what this agent
is"). Most publishers want **owner**. A record may declare both.

## 2. Key pair and key publication

Supported keys: ECDSA P-256 / P-384, Ed25519, RSA ≥ 2048. Prefer **P-256**:
its DNS value fits in one TXT string (RSA needs the value split into several
strings at the DNS provider).

```bash
openssl ecparam -name prime256v1 -genkey -noout -out owner.key   # private key (PEM) — keep it secret
openssl pkey -in owner.key -pubout -out owner.pub                # public half
```

An existing cosign or other PEM key works too. Encrypted PKCS#8 keys are
accepted; the password comes from `COSIGN_PASSWORD`, `--password-stdin`, or a
terminal prompt.

### DNS (`dns:` subjects)

```bash
KEY=$(openssl pkey -in owner.key -pubout -outform DER | base64 | tr -d '\n')
echo "_agntcy-key.example.com.  300  IN  TXT  \"v=akv1;key=$KEY\""
```

PowerShell (OpenSSL on `PATH`):

```powershell
openssl pkey -in owner.key -pubout -outform DER -out owner.der
$KEY = [Convert]::ToBase64String([IO.File]::ReadAllBytes("owner.der"))
"_agntcy-key.example.com.  300  IN  TXT  `"v=akv1;key=$KEY`""
```

Give the user the record to add at their DNS provider. One TXT record per key;
publishing two keys (old and new) during a rotation is fine — any one of them
verifies. Check it is live before claiming: `dig +short TXT
_agntcy-key.example.com` (PowerShell: `Resolve-DnsName -Type TXT
_agntcy-key.example.com`).

### HTTPS / JWKS (`https://` subjects)

Serve a JWK set over HTTPS at `/.well-known/jwks.json` on the subject's host.
For a P-256 key, the coordinates are the last 64 bytes of the DER public key:

```bash
b64url() { base64 | tr -d '\n=' | tr '+/' '-_'; }
openssl pkey -in owner.key -pubout -outform DER > owner.der
X=$(tail -c 64 owner.der | head -c 32 | b64url)
Y=$(tail -c 32 owner.der | b64url)
printf '{"keys":[{"kty":"EC","crv":"P-256","x":"%s","y":"%s"}]}\n' "$X" "$Y"
```

For Ed25519 the key is the last 32 bytes:
`{"kty":"OKP","crv":"Ed25519","x":"<b64url>"}`.

### did:web

Serve `did.json` at the location the did:web method defines (table above). Its
`id` must equal the subject, and the key must be listed under
`assertionMethod` — a key only in `verificationMethod` does not count:

```json
{
  "id": "did:web:example.com",
  "verificationMethod": [{
    "id": "did:web:example.com#owner",
    "type": "JsonWebKey2020",
    "controller": "did:web:example.com",
    "publicKeyJwk": {"kty": "EC", "crv": "P-256", "x": "<X>", "y": "<Y>"}
  }],
  "assertionMethod": ["did:web:example.com#owner"]
}
```

### SPIFFE / ANS

No key is published. Pass the certificate with `--cert` at claim time: the
signer's X.509-SVID for `spiffe://`, the identity certificate the agent's
transparency log attests for `ans://`. Its URI SAN must equal the subject.
`ans://` subjects are written `ans://v<major>.<minor>.<patch>.<host>` with a
lowercase host, and the host needs a `_ans-badge` TXT record naming the log.
Push a new claim after each certificate renewal.

## 3. Declare the subject, then push

The annotation is part of the record, so it is covered by the CID — it must be
set **before** the push, and changing it later means a new record and a new
CID.

```json
"annotations": {
  "agntcy.dir/owner": "dns:example.com",
  "agntcy.dir/identity": "did:web:example.com:agents:research-assistant"
}
```

```bash
dirctl validate record.json --url https://schema.oasf.outshift.com
CID=$(dirctl push record.json -o raw)
```

For an already-pushed record without the annotation: add it, push again (new
CID), and claim the new CID.

## 4. Claim

```bash
dirctl identity claim --record "$CID" --role owner --key owner.key
dirctl identity claim --record "$CID" --role identity --key svid.key --cert svid.pem   # spiffe:// or ans://
COSIGN_PASSWORD=… dirctl identity claim --record "example.com/agent:v1.0.0" --role owner --key owner.key
```

`--record` takes a CID, `name` (highest version) or `name:version`. `--cert`
is refused for any subject other than `spiffe://` and `ans://`. A role the
record does not declare is refused.

## 5. Check the result

```bash
dirctl identity status "$CID" -o json
```

Each role reports `verified`, `failed` (with `error`), or `no result` (`null`
in JSON). A fresh claim reads `no result` until the reconciler's next identity
run (the local daemon runs it every minute) — wait and re-check; do not
re-claim.

Once verified, the record matches these search filters:

```bash
dirctl search --owner-verified               # owner claim verified (same as --verified)
dirctl search --identity-verified            # identity claim verified
dirctl search --owner 'dns:example.com'      # by declared subject; wildcards allowed
dirctl search --identity 'did:web:example.com:*'
dirctl search --verified=false               # records without a verified owner
```

## Troubleshooting a `failed` claim

Show the raw `error` first, then map it:

| `error` contains | Likely cause | Fix |
| --- | --- | --- |
| `no usable public keys found`, `no document at`, `no such host`, `lookup TXT records for` | Key not published, typo in the TXT name, no `v=akv1;` prefix, JWKS/did.json missing, or a did.json key only under `verificationMethod` | Publish at the exact location in the subject table; check with `dig` / `curl` |
| `signature does not verify` | The published key is not the public half of `--key` (often an old key) | Re-derive the value from the key you signed with; publish it next to the old one |
| `DID document id … does not match subject` | `did.json` `id` ≠ the declared subject | Fix the document's `id` |
| `unsupported subject scheme`, `unsupported did method` | Scheme not served (`http://`, `did:` other than web/key, or `ans://` with ANS off on the node) | Pick a supported subject, or ask the operator |
| `trust domain`, `SVID is for`, `certificate does not cover claimed subject` | No trust bundle for the SPIFFE domain, wrong SVID, or URI SAN ≠ subject | Ask the operator for the bundle; claim with the SVID that names the subject |
| `ans name:`, `ans certificate:`, `ans badge:`, `ans log:` | That ANS step failed (subject spelling, cert validity, `_ans-badge` TXT, or log status / trust) | Fix the named step; an untrusted log host is an operator setting |
| `claim has expired` | Claim was created with an expiry (SDK) | Claim again |

Behaviour worth telling the user:

- **Rotation and revocation are live.** Removing the old key from DNS/JWKS
  flips its claims to `failed` on the next run. Rotate by publishing the new
  key next to the old one, re-claiming with the new key, then removing the old.
- **Unreachable is not wrong.** A timeout or 5xx keeps the last stored result
  for up to seven days (24 h for `ans://`); a 404 or "no such host" fails at
  once.
- **Claims on other people's records do nothing.** Anyone can attach a claim,
  but only a claim for the subject the record itself declares is evaluated.

# Publishing: push, sign, announce, prove ownership

Goal: get a validated record stored, signed, discoverable, and (optionally)
name-verified.

## The pipeline

```bash
dirctl validate record.json --url https://schema.oasf.outshift.com   # 1. validate
CID=$(dirctl push record.json -o raw)                                # 2. store → CID
dirctl sign "$CID"                                                   # 3. sign (optional, recommended)
dirctl routing publish "$CID"                                        # 4. announce to the network (optional)
```

PowerShell (Windows): capture with `$CID = dirctl push record.json -o raw`,
then pass `$CID` unquoted to the subsequent commands.

Report the CID back to the user prominently — it is the durable handle for
every later operation.

## 1–2. Push

- `push` stores the record in the content-addressable store and prints the
  CID (`-o raw` for scripting).
- Idempotent: pushing identical bytes returns the same CID, no duplicate.
  Re-push only when the record actually changed.
- Validation failures come back as `InvalidArgument` with per-attribute
  messages — render them as a table and route the user to the authoring
  reference.

## 3. Sign

```bash
dirctl sign <cid>                      # keyless OIDC (Sigstore; opens browser)
dirctl sign <cid> --key cosign.key     # private key
```

- Signatures are stored as OCI **referrers** attached to the CID — the record
  itself is unchanged.
- Signing enables the `--trusted` search filter and `dirctl verify` for
  consumers.
- Non-interactive signing: `--oidc-token` (CI).

## 4. Announce to the network (routing)

Records are local until published to the DHT:

```bash
dirctl routing publish <cid>      # announce; discoverable by peers
dirctl routing unpublish <cid>    # withdraw from discovery; stays in local storage
dirctl routing list               # what this node has published (filter: --skill, --domain, --module, --locator, --cid)
dirctl routing info               # publication stats: counts, skills/locators distribution
```

Clarify intent with the user: **push** = store on the connected server;
**routing publish** = make discoverable across the peer-to-peer network. A
record can be pushed but unpublished (private to that directory).

## Record identity & ownership claims (optional)

Proves the record's own identity and/or who owns/controls it. Independent of
the `name` field; the subject can use `did:web:`, `did:key:`, `https://`
(JWKS), `dns:` (TXT record), or `spiffe://` (X.509-SVID).

Workflow:

```bash
CID=$(dirctl push record.json -o raw)
dirctl identity claim --record "$CID" --role identity --subject did:web:my-agent.example.com --key private.key
dirctl identity claim --record "$CID" --role owner --subject did:web:acme.com --key owner.key
dirctl identity status "$CID"             # check verification status
```

A verified ownership claim lights up the `--owner-verified` search filter. If
verification fails, check (in order): subject scheme matches the signer's key
material, the resolution endpoint (DID document / JWKS / TXT record) is
reachable, the signing key matches what's published there.

## Maintenance

```bash
dirctl info <cid-or-name>     # metadata for a stored record
dirctl delete <cid>           # remove from storage — confirm with the user first
```

`delete` does not retract network announcements on other peers that already
synced the record; unpublish before deleting when the intent is "remove from
the network".

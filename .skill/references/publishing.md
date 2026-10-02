# Publishing: push, sign, announce, prove ownership

Goal: get a validated record stored, signed, discoverable, and (optionally)
ownership-verified.

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
- Import flows can sign in bulk: `dirctl import ... --sign [--key ...]`.
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

## Ownership claim (optional)

Proves who owns the record: a signed claim that the owner the record declares
is its owner, checked against the key material that owner publishes. The record
declares it in its `agntcy.dir/owner` annotation (`agntcy.dir/identity` for the
record's own identity), which must be set before the push.

```bash
# record.json: "annotations": {"agntcy.dir/owner": "dns:example.com"}
CID=$(dirctl push record.json -o raw)
dirctl identity claim --record "$CID" --role owner --key owner.key   # the owner publishes the public key (DNS TXT, JWKS, DID, SPIFFE)
dirctl identity status "$CID"                                         # check the verification result
```

The reconciler verifies claims in the background, so a fresh claim reads
`no result` until its next run. Verified owners light up the `--verified` and
`--owner-verified` search filters. If verification fails, `identity status`
shows the reason; check that the subject publishes the public half of the key.
Use `--role identity` for a claim about the record's own identity.

## Maintenance

```bash
dirctl info <cid-or-name>     # metadata for a stored record
dirctl delete <cid>           # remove from storage — confirm with the user first
```

`delete` does not retract network announcements on other peers that already
synced the record; unpublish before deleting when the intent is "remove from
the network".

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

## Name ownership verification (optional, for `https://` names)

Proves the signing key is authorized by the domain in the record name.

Requirements:

1. Record `name` uses a protocol prefix: `https://example.com/my-agent`.
2. The domain hosts a JWKS at `https://example.com/.well-known/jwks.json`.
3. The record is signed with a private key whose public key is in that JWKS.

Workflow:

```bash
CID=$(dirctl push record.json -o raw)
dirctl sign "$CID" --key private.key      # triggers automatic domain verification
dirctl naming verify "$CID"               # check verification status
```

Verified names light up the `--verified` search filter. If verification
fails, check (in order): name has the scheme prefix, JWKS is reachable,
signing key matches a JWKS entry.

## Maintenance

```bash
dirctl info <cid-or-name>     # metadata for a stored record
dirctl delete <cid>           # remove from storage — confirm with the user first
```

`delete` does not retract network announcements on other peers that already
synced the record; unpublish before deleting when the intent is "remove from
the network".

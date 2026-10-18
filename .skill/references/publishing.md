# Publishing: push, sign, announce, prove ownership

Goal: get a validated record stored, signed, discoverable, and (optionally)
ownership-verified.

## The pipeline

```bash
dirctl validate record.json --url https://schema.oasf.outshift.com   # 1. validate
CID=$(dirctl push record.json -o raw)                                # 2. store → CID
dirctl sign "$CID"                                                   # 3. sign (optional, recommended)
dirctl routing publish "$CID"                                        # 4. announce to the network (optional)
dirctl identity claim --record "$CID" --role owner --key owner.key   # 5. prove ownership (optional; see identity.md)
```

If the user wants ownership verified, set `agntcy.dir/owner` in the record
**before** step 2 — adding it later changes the CID.

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
dirctl sign <cid> --key cosign.key     # private key (EC, RSA or Ed25519 PEM)
dirctl push record.json --sign [--key cosign.key] -o raw   # push and sign in one step
```

- Signatures are stored as OCI **referrers** attached to the CID — the record
  itself is unchanged.
- Private Sigstore deployments: put the Fulcio/Rekor/OIDC URLs in the
  context's `sigstore:` block (setup reference) instead of repeating flags.
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

## 5. Claim ownership (optional, recommended)

Signing proves the bytes weren't changed; a **claim** proves who owns the
record (e.g. `dns:example.com`), and shows it as **Verified** in search. It
needs the `agntcy.dir/owner` annotation set before the push and a public key
the owner publishes:

```bash
dirctl identity claim --record "$CID" --role owner --key owner.key
dirctl identity status "$CID"
```

Follow [identity.md](identity.md) for the full flow: choosing a subject,
generating and publishing the key, and fixing failed claims.

## Maintenance

```bash
dirctl info <cid-or-name>     # metadata for a stored record
dirctl delete <cid>           # remove from storage — confirm with the user first
```

`delete` does not retract network announcements on other peers that already
synced the record; unpublish before deleting when the intent is "remove from
the network".

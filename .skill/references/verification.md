# Verification: signatures, name ownership, and security scans

Goal: answer "can I trust this record?" with concrete, verifiable signals —
never a synthesized score.

## The three trust signals

| Signal | Meaning | Command / filter |
| --- | --- | --- |
| **Trusted** | Signature verification passed | `dirctl verify <cid>`; search `--trusted` |
| **Verified** | Signing key authorized by the domain in the record name (JWKS) | `dirctl naming verify <ref>`; search `--verified` |
| **Safe** | All security scanners reported `is_safe=true` | `dirctl pull <cid> --scan-report`; search `--safe`, `--scan-severity` |

Report them separately; they answer different questions (integrity, identity,
content safety).

## Verify a signature

```bash
dirctl verify <cid>                                   # fetch signature from the directory and verify
dirctl verify <cid> --key public.pem                  # against a specific key (path, URL, or KMS URI)
dirctl verify <cid> --oidc-issuer "https://accounts.google.com" --oidc-subject "ci@example.com"   # keyless identity match (regexp supported)
dirctl verify <cid> --from-server                     # use the server's cached verification result
```

`--ignore-tlog` skips transparency-log verification — only when the user
explicitly asks.

## Verify name ownership

```bash
dirctl naming verify <cid>
dirctl naming verify "https://example.com/agent:v1.0.0"
```

Checks the signing key against `https://<domain>/.well-known/jwks.json`. Only
meaningful for records whose name has an `http(s)://` prefix; plain names are
reported as not applicable, not as failures.

## Security scan reports

The directory reconciler runs scanners automatically and attaches results as
referrers: **mcp-scanner** (MCP server source) and **skill-scanner** (agent
skill bundles).

```bash
dirctl pull <cid> --scan-report -o json
```

The `scanReports` array contains per-scanner entries: `scanner_type`,
`scanner_version`, `scanned_at`, `is_safe`, `max_severity`
(`SEVERITY_NONE|INFO|MEDIUM|HIGH|CRITICAL`), and `findings` (rule ID,
severity, message, location, remediation).

Render findings as a table:

| Scanner | Severity | Rule | Message | Remediation |
| --- | --- | --- | --- | --- |

Summarize with `is_safe` + `max_severity` per scanner. A record with **no**
reports is *unscanned* — say so explicitly rather than implying safety.

## Filtering by trust at search time

```bash
dirctl search --safe                          # all scanners is_safe=true (unscanned records excluded)
dirctl search --scan-severity HIGH            # highest finding ≥ HIGH
dirctl search --trusted --verified            # signed + name-verified
dirctl search "code review agent" --safe      # combine with NL query
```

## Pre-install trust check (recommended flow)

Before installing a record the user picked:

```bash
dirctl verify <cid> --from-server
dirctl naming verify <cid>
dirctl pull <cid> --scan-report -o json
```

Present the three signals as a one-row table. If any signal is negative or
missing, tell the user and let them decide — do not block, do not
editorialize beyond the data.

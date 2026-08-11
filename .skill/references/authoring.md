# Authoring: create, validate, and import OASF records

Goal: produce valid OASF record JSON — written by hand or imported from
existing artifacts (MCP servers, A2A cards, Agent Skills) — ready to push.

## Author a record by hand

Minimal top-level fields: `name`, `schema_version`, `version`, `description`,
`authors`, `created_at` (RFC 3339), `skills`. Recommended: `domains`,
`locators`, `modules`. Custom metadata goes in the `annotations` map.

```json
{
  "name": "example.com/research-agent",
  "schema_version": "1.0.0",
  "version": "v1.0.0",
  "description": "Answers research questions with cited sources.",
  "authors": ["Example Corp"],
  "created_at": "2026-07-14T00:00:00Z",
  "skills": [{ "id": 10201, "name": "natural_language_processing/text_completion" }],
  "domains": [{ "name": "technology" }],
  "locators": [{ "type": "docker-image", "url": "ghcr.io/example/research-agent:v1.0.0" }],
  "modules": []
}
```

Rules:

- Skills/domains are closed taxonomies — resolve valid `id`/`name` pairs from
  the OASF schema (`https://schema.oasf.outshift.com/<version>/skills`) or the
  `agntcy_oasf_get_schema_skills` / `..._domains` MCP tools. Do not invent
  taxonomy entries; ID and name must refer to the same class.
- Names starting with `https://` / `http://` opt into domain-based name
  ownership verification (see publishing reference). Plain names skip it.
- Modules determine installability later: `integration/mcp` → MCP server,
  `core/language_model/agentskills` → Agent Skill, `integration/a2a` → A2A
  card.
- Records are capped at 4 MB (metadata 100 KB) — reference large blobs via
  locators.

## Validate before pushing

The server rejects invalid records with `InvalidArgument`, so validate first:

```bash
dirctl validate record.json --url https://schema.oasf.outshift.com
dirctl pull <cid> | dirctl validate --url https://schema.oasf.outshift.com  # stdin works
```

`--url` is required (API-based validation). Use the same OASF endpoint the
target server validates against — schema-version drift between client and
server is a common source of rejections.

Render validation ERRORs/WARNINGs as a table (severity | path | message) and
fix ERRORs before pushing.

## Import existing artifacts

`dirctl import` converts external artifacts into OASF records and pushes them.

| `--type` | Source | Required flag |
| --- | --- | --- |
| `mcp-registry` | HTTP MCP registry (v0.1 list API) | `--url` |
| `mcp` | Local JSON (one server or array) | `--file-path` |
| `a2a` | Local A2A AgentCard JSON | `--file-path` |
| `agent-skill` | Local directory containing `SKILL.md` | `--file-path` |

Useful flags: `--config <yaml>` (see config reference below), `--limit N`,
`--filter key=value` (registry: `search`, `version`, `updated_since`),
`--output-cids <file>`, `--force` (skip name+version dedup), `--debug`,
`--sign` (+ `--key` / `--oidc-token`).

### Recommended workflow: dry-run → review → push

`--dry-run` writes transformed records to disk instead of pushing — one
`<cid>.record.json` per record — so the user can review before anything
reaches the directory:

```bash
dirctl import --type=mcp-registry \
  --url=https://registry.modelcontextprotocol.io/v0.1 \
  --filter search=github --limit 10 \
  --dry-run --output-dir=./out

# review ./out/*.record.json with the user, then:
for f in ./out/*.record.json; do dirctl push "$f"; done
```

PowerShell (Windows) equivalent of the push loop:

```powershell
Get-ChildItem ./out/*.record.json | ForEach-Object { dirctl push $_.FullName }
```

Prefer dry-run whenever the user hasn't seen the records yet; import directly
only when they explicitly ask for it.

### Enrichment

Import assigns OASF skills/domains to every record it builds. Exactly one of
three methods is used, selected by the `enricher:` block of `--config <yaml>`.
**When no valid block is present, import falls back to the LLM enricher**,
which needs credentials — so a malformed `enricher:` block fails as a missing
API key, not as a config error.

| Block                | Needs                                       | Use when                                     |
| -------------------- | ------------------------------------------- | -------------------------------------------- |
| `enricher.extractor` | `dirctl init` (local assets); no credentials | **Default choice** — per-record, LLM-free     |
| `enricher.static`    | nothing                                      | Stamp one fixed skill/domain set on every record |
| `enricher.llm`       | LLM credentials (e.g. `AZURE_OPENAI_API_KEY`) | An LLM is available and per-record quality matters |

```yaml
# Local OASF taxonomy model — no LLM, no credentials. Requires `dirctl init`.
enricher:
  extractor:
    oasf_url: https://schema.oasf.outshift.com
    asset_dir: /absolute/path/to/.agntcy/oasf-sdk/extractor
```

```yaml
# Fixed taxonomy stamped on every imported record — no LLM, no init.
enricher:
  static:
    skills:
      - name: natural_language_processing/text_completion
        id: 10201
    domains:
      - name: technology
        id: 1
```

Two traps, both of which surface as the LLM credential error rather than as a
complaint about the config:

- **Unknown keys are silently ignored** (unlike the client config, the import
  config parser is not strict). A misplaced key — `skills:` directly under
  `enricher:`, or a `skip_enricher:` flag, which does not exist — leaves no
  recognised enricher and triggers the LLM fallback.
- **A bare `enricher.extractor:` with no fields parses as null**, which is also
  no enricher. Write `extractor: {}` to inherit the endpoint and asset
  directory saved by `dirctl init`, or set both fields explicitly.

`asset_dir` must be an **absolute path** — `~` is not expanded.

When an import fails with `AZURE_OPENAI_API_KEY is required`, do not go looking
for credentials first: check the `enricher:` block spelling, then offer
`extractor` (if `dirctl init` has been run) or `static` as the no-credentials
path.

### Config file reference

Everything below can live in `--config <yaml>` instead of on the command line.
**Flags override the file**, so one file can drive several runs:

```yaml
type: mcp-registry            # mcp | mcp-registry | a2a | agent-skill
url: https://registry.modelcontextprotocol.io/v0.1   # required for mcp-registry
# file_path: ./servers.json   # required for mcp | a2a | agent-skill

filters:
  search: github              # registry filters: search, version, updated_since
limit: 100                    # 0 = no limit

# authors:                    # overrides authors derived from the source
#   - "Example Corp"

enricher:                     # see above — extractor | static | llm
  extractor: {}

schema_version: "1.1.0"       # OASF version stamped on imported records

output_dir: ./import-out      # where --dry-run writes <cid>.record.json
dry_run: true
force: false                  # true = skip name+version dedup
debug: false
```

`dry_run: true` in the file means a bare `dirctl import --config <file>` is
**already** a dry run — say so rather than implying records were pushed. Flip
it with `--dry-run=false` when the user wants the real import.

Note that `import` opens a client connection even for a dry run, so the
selected context still has to resolve.

## Common pitfalls

- Re-importing without `--force` silently skips records that already exist
  (name+version dedup) — use `--debug` to see why records were skipped.
- CID covers exact bytes: reformatting a record file changes its CID.
- `created_at` must be RFC 3339; version tags need not be semver (`latest`,
  `dev` are fine — resolution picks the newest `created_at`).

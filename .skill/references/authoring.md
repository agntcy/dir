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

Useful flags: `--limit N`, `--filter key=value` (registry: `search`,
`version`, `updated_since`), `--output-cids <file>`, `--force` (skip
name+version dedup), `--debug`, `--sign` (+ `--key` / `--oidc-token`).

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

By default import enriches records with OASF skills/domains using an LLM
(tool-calling against `dirctl mcp serve`; built-in default `azure:gpt-4o`).
This requires LLM credentials (e.g. `AZURE_OPENAI_API_KEY`). Two alternatives
via `--config <yaml>`:

```yaml
# Static taxonomy — no LLM needed:
enricher:
  skip_enricher: true
  skills:
    - name: natural_language_processing/text_completion
      id: 10201
  domains:
    - name: technology
      id: 1
```

If an import fails on enrichment, offer `skip_enricher` with skills/domains
derived from the artifact description as the no-credentials path.

## Common pitfalls

- Re-importing without `--force` silently skips records that already exist
  (name+version dedup) — use `--debug` to see why records were skipped.
- CID covers exact bytes: reformatting a record file changes its CID.
- `created_at` must be RFC 3339; version tags need not be semver (`latest`,
  `dev` are fine — resolution picks the newest `created_at`).

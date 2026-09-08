# Authoring: create and validate OASF records

Goal: produce valid OASF record JSON — written by hand — ready to push.

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

## Common pitfalls

- CID covers exact bytes: reformatting a record file changes its CID.
- `created_at` must be RFC 3339; version tags need not be semver (`latest`,
  `dev` are fine — resolution picks the newest `created_at`).

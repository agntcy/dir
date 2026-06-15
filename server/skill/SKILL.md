---
name: agntcy-dir
description: Use this skill to interact with an AGNTCY Directory (DIR) instance. DIR is a distributed peer-to-peer registry of AI agent records described in OASF. Use it to publish (push), discover (search), retrieve (pull), sign, and verify agent records.
metadata:
  author: AGNTCY Contributors
  version: 1.0.0
---

# AGNTCY Directory (DIR)

DIR is a distributed registry for AI agent records. Records describe agents using
the [Open Agentic Schema Framework (OASF)](https://github.com/agntcy/oasf) and
are content-addressed by CID.

This skill explains how to talk to a running DIR node. You can use the MCP server,
the `dirctl` CLI, the language SDKs, or the raw gRPC API — pick whichever channel
your environment supports.

## Vocabulary

- **Record** — an OASF JSON document describing one agent (name, version, skills, locators, modules, signatures).
- **CID** — Content Identifier; the SHA-based hash that uniquely names a record. Pushing the same record twice yields the same CID.
- **Skill / Domain** — OASF taxonomy nodes describing what an agent does (e.g. `natural_language_processing/text_completion`) and where it operates.
- **Module** — typed extension on a record (e.g. `core/language_model/agentskills`, `runtime/mcp`, `integration/a2a`).
- **Locator** — pointer to a runnable artifact for the agent (`container_image`, `source_code`, `helm_chart`, `package`, `binary`, `url`).

## Channels

| Operation | MCP tool | CLI | Go SDK |
|---|---|---|---|
| Push a record | `agntcy_dir_push_record` | `dirctl push <file>` | `client.Push` |
| Pull by CID | `agntcy_dir_pull_record` | `dirctl pull <cid>` | `client.Pull` |
| Search local | `agntcy_dir_search_local` | `dirctl search` | `client.Search` |
| Validate against OASF | `agntcy_oasf_validate_record` | `dirctl validate <file>` | — |
| List OASF versions | `agntcy_oasf_list_versions` | — | — |
| Sign a record | — | `dirctl sign --key=<cosign.key> <cid>` | `client.Sign` |
| Verify a signature | `agntcy_dir_verify_record` | `dirctl verify <cid>` | `client.Verify` |
| Verify name ownership | `agntcy_dir_verify_name` | `dirctl naming verify <cid>` | — |
| Import MCP / A2A / SKILL.md | `agntcy_oasf_import_record` | `dirctl import --type=<mcp\|a2a\|agent-skill>` | — |

## Workflow: publish an agent record

1. Author or generate an OASF record JSON. Required top-level fields for the
   1.0.0 schema: `name`, `schema_version`, `version`, `description`, `authors`,
   `created_at` (RFC3339), `skills`. Optional: `domains`, `locators`, `modules`,
   `annotations`.
2. Validate against the OASF schema before pushing — the server rejects
   malformed records with `InvalidArgument`.
3. Push. The server returns the CID. Pushes are idempotent: pushing the same
   bytes a second time returns the same CID without creating a duplicate.
4. Optionally sign the record with cosign — produces a signature *referrer*
   linked to the CID. Anyone can later verify signature and name ownership.

```bash
dirctl validate agent.json
CID=$(dirctl push agent.json --output=raw)
dirctl sign --key=cosign.key "$CID"
```

## Workflow: discover and pull a record

1. Search by structured filters (names, versions, skills, locators, authors, …).
   Filters use wildcards: `*` (any), `?` (one char), `[abc]` (set).
2. The search returns CIDs. Pull each CID to get the full record JSON.

```bash
dirctl search --skill-name "*text_completion*" --limit 20
dirctl pull <cid>
```

For natural-language searches via an LLM, prefer the `search_records` MCP prompt,
which translates a free-text query into structured filters automatically.

## OASF essentials

- Schema versions in active use: `0.7.0`, `0.8.0`, `1.0.0`. Pick the latest your
  tooling supports; `1.0.0` is the current default.
- Module enum is closed in 1.0.0 — only these `name`s validate:
  `language_model`, `prompt`, `core/language_model/agentskills`, `evaluation`,
  `observability`, `a2a`, `acp`, `agentspec`, `mcp`. Custom payloads belong in
  the top-level `annotations` map.
- Skills and domains are taxonomies; resolve `id` ↔ `name` with the
  `agntcy_oasf_get_schema_skills` and `agntcy_oasf_get_schema_domains` MCP tools
  (or fetch the schema JSON directly).

## Common pitfalls

- A record's CID covers its bytes — any change (whitespace, key order) yields a
  new CID. Re-push only when the record actually changes.
- Names beginning with `http://` or `https://` opt into domain-based name
  ownership verification (`/.well-known/jwks.json` on the host). Plain names
  skip that path.
- Search defaults to local-only. To find records on remote peers, use routing
  search (`dirctl routing search …` / `client.SearchRouting`).
- The server caps records at 4 MB and metadata at 100 KB. Keep large blobs in
  external storage and reference them via locators.

## Pointers

- OASF schema: <https://schema.oasf.outshift.com/1.0.0>
- DIR repo & CLI reference: <https://github.com/agntcy/dir>
- DIR MCP server: <https://github.com/agntcy/dir-mcp>
- Quickstart docs: <https://docs.agntcy.org/dir/dir-quickstart/>

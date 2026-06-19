# Import and Export

Import and export are complementary extensions that bridge Directory and external systems.
**Import** brings agent records *into* Directory from heterogeneous external sources —
remote registries as well as local files (JSON A2A Cards, MCP server definitions, Agent
Skills directories) — while **export** transforms stored records *out* into formats consumed
by external tools and agentic CLIs.

## Import

**Import** extends Directory's synchronization capabilities beyond Directory-to-Directory
[sync](dir-component-routing.md#synchronization) to support heterogeneous external sources.
It aggregates agent records into the local Directory instance from remote registries as well
as local files such as A2A AgentCards, MCP server definitions, and Agent Skills.

### How import works

The import system uses source-specific adapters to fetch records from external sources —
remote registries or local files — and transform them into OASF-compliant records. Each
source kind has its own import logic that handles authentication, pagination, filtering, and
data transformation. Records are automatically deduplicated and can be enriched with
LLM-powered skill and domain mapping to ensure consistency with the OASF schema.

### Translation and enrichment

Records are transformed from external registry data into OASF-compliant format, which
directly impacts how records are indexed and discovered across the network. Three methods are
available:

- **Basic translation** uses [OASF-SDK basic translation](https://docs.agntcy.org/oasf/translation/)
  with rule-based mapping. It is fast and deterministic but produces a record without any
  skills or domains, requiring manual or LLM-based enrichment afterwards.
- **Local LLM enrichment** runs an LLM locally for intelligent skill and domain mapping,
  requiring a local LLM runtime.
- **Remote LLM enrichment** uses external LLM services for skill and domain mapping,
  requiring API credentials.

Both LLM methods require access to an LLM with tool-calling support and the corresponding
provider credentials (for example, Azure OpenAI, or a local model via Ollama). The enrichment
pipeline is built into `dirctl` — it runs the OASF schema tools exposed by `dirctl mcp serve`
against the model using a built-in configuration that can be overridden with `--enrich-config`.
See [CLI Reference — LLM-based Enrichment](dir-cli-reference.md#llm-based-enrichment-mandatory)
for provider setup and configuration details.

### Supported import kinds

| Kind | Description | Required flag |
|------|-------------|---------------|
| `mcp-registry` | [Model Context Protocol registry v0.1](https://github.com/modelcontextprotocol/registry) | `--url` |
| `mcp` | Local MCP server JSON | `--file-path` |
| `a2a` | Local A2A AgentCard JSON | `--file-path` |
| `agent-skill` | Local Agent Skills directory with `SKILL.md` | `--file-path` |

## Export

**Export** is the inverse of import: it pulls stored OASF records and transforms them into
formats that external tools and agentic CLIs can consume directly, such as A2A AgentCards,
`SKILL.md` artifacts, or MCP configuration. This lets records published to Directory feed
back into the broader agent ecosystem without manual conversion.

### How export works

Export retrieves a record by CID or name and runs it through a format-specific transformer.
Records can be exported one at a time, or in batches driven by the same search filters used
for [discovery](dir-component-routing.md) (`--name`, `--version`, `--module`, `--skill`,
`--author`, and so on). By default only the latest semver version per name is exported;
`--all-versions` exports every version.

### Supported export formats

| Format | Output | Description |
|--------|--------|-------------|
| `oasf` | `.json` | Raw OASF record JSON (default) |
| `a2a` | `.json` | A2A AgentCard JSON for Agent-to-Agent protocol interop |
| `agent-skill` | `.md` | `SKILL.md` artifact for agentic CLIs (Cursor, Claude Code, etc.) |
| `mcp-ghcopilot` | `.json` | GitHub Copilot MCP configuration JSON |

Batch behaviour varies by format: `a2a` and `oasf` produce one file per record,
`agent-skill` produces one subdirectory per skill, and `mcp-ghcopilot` merges all matched
MCP servers into a single configuration file.

## Related documentation

- [Sync](dir-component-routing.md#synchronization) — Directory-to-Directory replication
- [Records](dir-component-records-validation.md) — the OASF record model that import and export target
- [Usage Guide — Import](dir-features-scenarios.md#import) — import CLI walkthroughs
- [Usage Guide — Export](dir-features-scenarios.md#export) — export CLI walkthroughs
- [CLI Reference — Import Operations](dir-cli-reference.md#import-operations) — `dirctl import` flags
- [CLI Reference — Export Operations](dir-cli-reference.md#export-operations) — `dirctl export` flags

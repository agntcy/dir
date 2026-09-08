---
icon: material/import
---

# Export

**Export** transforms stored Directory records *out* into formats consumed by external tools
and agentic CLIs — A2A AgentCards, `SKILL.md` artifacts, or MCP configuration.

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
| `mcp-claudecode` | `.json` | Claude Code MCP configuration JSON (`.mcp.json` `mcpServers` shape) |
| `mcp-cursor` | `.json` | Cursor IDE MCP configuration JSON (`.cursor/mcp.json` `mcpServers` shape) |

Batch behaviour varies by format: `a2a` and `oasf` produce one file per record,
`agent-skill` produces one subdirectory per skill, and `mcp-ghcopilot` / `mcp-claudecode` / `mcp-cursor`
merge all matched MCP servers into a single configuration file.

## Related documentation

- [Records](dir-component-records-validation.md) — the OASF record model that export targets
- [Usage Guide — Export](dir-features-scenarios.md#export) — export CLI walkthroughs
- [CLI Reference — Export Operations](dir-cli-reference.md#export-operations) — `dirctl export` flags

# Directory CLI (dirctl)

Command-line tools for the AGNTCY Directory.

- **Storage** — Push, pull, delete, and inspect records (info).
- **Routing** — Publish, unpublish, list, and search records across the network.
- **Integrity** — Sign, verify, and validate artifacts.
- **Import / Export** — Bulk-import from registries or local files; export to OASF, A2A, Agent Skills, and MCP formats.
- **Naming** — Domain verification (`naming verify`).
- **Sync** — Sync records between nodes.
- **Events** — Stream directory events (`events listen`).
- **MCP** — Run an MCP server for AI/agent tooling (`mcp serve`).
- **Auth** — Login, logout, and check status (for federation nodes).
- **Daemon** — Run a local Directory instance (`daemon start/stop/status`).
- **Diagnostics** — Connectivity checks (`doctor`) and version info.

Full documentation: [https://docs.agntcy.org/dir/dir-cli-reference/](https://docs.agntcy.org/dir/dir-cli-reference/)

## Quick start:

```bash
# Install (Homebrew)
brew tap agntcy/dir https://github.com/agntcy/dir
brew trust agntcy/dir
brew install dirctl

# Store, publish, search, pull
dirctl push my-record.json
dirctl routing publish <cid>
dirctl routing search --skill "natural_language_processing" --limit 10
dirctl pull <cid>
```

For federation nodes, authenticate first: `dirctl auth login`. See the documentation link above for details.

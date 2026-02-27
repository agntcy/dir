# Directory CLI (dirctl)

Command-line tools for the AGNTCY Directory.

- **Storage** — Push, pull, delete, and inspect records (info).
- **Routing** — Publish, unpublish, list, and search records across the network.
- **Integrity** — Sign, verify, and validate artifacts.
- **Import** — Bulk-import records into the store.
- **Naming** — Domain verification (verify, check, list).
- **Sync** — Sync records between nodes.
- **Events** — Stream directory events (listen).
- **MCP** — Run an MCP server for AI/agent tooling.
- **Auth** — Login, logout, and check status (for federation nodes).

Full documentation: [https://docs.agntcy.org/dir/directory-cli/](https://docs.agntcy.org/dir/directory-cli/)

## Quick start:

```bash
# Install (Homebrew)
brew tap agntcy/dir https://github.com/agntcy/dir/
brew install dirctl

# Store, publish, search, pull
dirctl push my-record.json
dirctl routing publish <cid>
dirctl routing search --skill "natural_language_processing" --limit 10
dirctl pull <cid>
```

For federation nodes, authenticate first: `dirctl auth login`. See the documentation link above for details.

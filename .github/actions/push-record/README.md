# Push OASF Record Action

Push agent records to a Directory node's OCI registry. Optionally publish to the DHT for network discovery and sign for authenticity verification.

## Features

- Push OASF records to any Directory node
- Supports multiple paths and glob patterns
- Optional DHT publish for network discovery
- Optional record signing for authenticity verification
- Returns CIDs for all pushed records
- Detailed error messages with GitHub annotations

## Usage

```yaml
name: Push Agents

on:
  push:
    branches: [main]
    paths:
      - 'record.json'
      - 'agents/**/*.json'

permissions:
  contents: read
  id-token: write  # Required for OIDC signing

jobs:
  push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Push agent records
        id: push
        uses: agntcy/dir/.github/actions/push-record@main
        with:
          record_paths: |
            record.json
            agents/**/*.json
          directory_address: ${{ vars.DIRECTORY_ADDRESS }}
          github_token: ${{ secrets.DIRECTORY_TOKEN }}
          sign: "true"
          publish: "true"
      
      - name: Output pushed CIDs
        run: |
          echo "Pushed CIDs:"
          echo '${{ steps.push.outputs.cids }}' | jq .
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `record_paths` | Paths to OASF record JSON files (one per line, supports globs) | No | `record.json` |
| `directory_address` | Directory server address (host or host:port) | Yes | - |
| `publish` | Publish to DHT for network discovery after pushing | No | `false` |
| `sign` | Sign records after pushing | No | `false` |
| `github_token` | GitHub token for Directory authentication | No | - |
| `dirctl_version` | Version of dirctl to use | No | `latest` |

## Outputs

| Output | Description |
|--------|-------------|
| `success` | Whether all records were pushed successfully (`true` or `false`) |
| `cids` | JSON object mapping file paths to their CIDs |
| `failed_files` | JSON array of file paths that failed to push |

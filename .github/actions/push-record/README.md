# Push OASF Record Action

Push agent records to a Directory node's OCI registry. Optionally publish to the DHT for network discovery and sign for authenticity verification.

## Usage

```yaml
name: Push Agents

on:
  push:
    branches: [main]
    paths:
      - 'record.json'
      - 'agents/**/*.json'

jobs:
  push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v4

      - name: Setup dirctl
        uses: agntcy/dir/.github/actions/setup-dirctl@main
      
      - name: Push agent records
        id: push
        uses: agntcy/dir/.github/actions/push-record@main
        with:
          record_paths: |
            record.json
            agents/**/*.json
          server_addr: ${{ vars.DIRECTORY_ADDRESS }}
          github_token: ${{ secrets.DIRECTORY_TOKEN }}
          sign: true
          oidc_client_id: https://github.com/${{ github.repository }}/.github/workflows/push-agents.yaml@${{ github.ref }}
          publish: true
      
      - name: Output pushed CIDs
        run: |
          echo "Pushed CIDs:"
          echo '${{ steps.push.outputs.cids }}' | jq .
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `record_paths` | Paths to OASF record JSON files (one per line, supports globs) | No | `record.json` |
| `server_addr` | Directory server address (host:port) | Yes | - |
| `github_token` | GitHub PAT for Directory authentication | Yes | - |
| `sign` | Sign records after pushing | No | `false` |
| `oidc_client_id` | Workflow URL for keyless signing (required when `sign` is true in CI). Job must have `id-token: write`. | No | `""` |
| `publish` | Publish to DHT for network discovery after pushing | No | `false` |

## Outputs

| Output | Description | Example |
|--------|-------------|--------|
| `success` | Whether all records were pushed successfully | `true` |
| `cids` | JSON object mapping file paths to CIDs | `{"record.json":"bae...","agents/a.json":"bae..."}` |
| `failed_files` | JSON array of file paths that failed to push | `["agents/bad.json"]` |

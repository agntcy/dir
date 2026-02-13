# Push OASF Record Action

Push agent records to a Directory node's OCI registry. Outputs CIDs for use with **publish-record** (DHT) and **sign-record** (keyless signing).

## Usage

```yaml
- name: Setup dirctl
  uses: agntcy/dir/.github/actions/setup-dirctl@main

- name: Push records
  id: push
  uses: agntcy/dir/.github/actions/push-record@main
  with:
    record_paths: |
      record.json
      agents/**/*.json
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    github_token: ${{ secrets.DIRECTORY_TOKEN }}
```

Compose with publish and sign:

```yaml
- name: Push records
  id: push
  uses: agntcy/dir/.github/actions/push-record@main
  with:
    record_paths: record.json
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    github_token: ${{ secrets.GITHUB_TOKEN }}

- name: Publish to DHT
  uses: agntcy/dir/.github/actions/publish-record@main
  with:
    cids: ${{ steps.push.outputs.cids }}
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    github_token: ${{ secrets.GITHUB_TOKEN }}

- name: Sign records
  uses: agntcy/dir/.github/actions/sign-record@main
  with:
    cids: ${{ steps.push.outputs.cids }}
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    github_token: ${{ secrets.GITHUB_TOKEN }}
    oidc_client_id: https://github.com/${{ github.repository }}/.github/workflows/my-workflow.yaml@${{ github.ref }}
```

(Job needs `permissions: id-token: write` for sign-record.)

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `record_paths` | Paths to OASF record JSON files (one per line, supports globs) | No | `record.json` |
| `server_addr` | Directory server address (host:port). Omit to use dirctl default. | No | `""` |
| `github_token` | Optional GitHub PAT for Directory authentication | No | `""` |

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `success` | Whether all records were pushed successfully | `true` |
| `cids` | JSON object mapping file paths to CIDs | `{"record.json":"bae...","agents/a.json":"bae..."}` |
| `failed_files` | JSON array of file paths that failed to push | `["agents/bad.json"]` |

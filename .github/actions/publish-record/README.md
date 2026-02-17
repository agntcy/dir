# Publish OASF Records Action

Publish record CIDs to the DHT for network discovery. Use after **push-record**. Accepts the `cids` output from push-record (JSON) or a newline-separated list of CIDs.

## Usage

```yaml
- name: Publish to DHT
  uses: agntcy/dir/.github/actions/publish-record@main
  with:
    cids: ${{ steps.push.outputs.cids }}
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    github_token: ${{ secrets.GITHUB_TOKEN }}
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `cids` | JSON object (file→CID) from push-record, or newline-separated CIDs | Yes | - |
| `server_addr` | Directory server address (host:port). Omit to use dirctl default. | No | `""` |
| `github_token` | Optional GitHub PAT for Directory authentication | No | `""` |

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `success` | Whether all CIDs were published successfully | `true` |
| `published` | Number of CIDs successfully published | `3` |
| `failed` | Number of CIDs that failed to publish | `0` |

# Sign OASF Records Action

Sign record CIDs using GitHub OIDC (keyless). Use after **push-record**. Only supported in GitHub Actions; the job must have `permissions: id-token: write`.

## Usage

```yaml
- name: Sign records
  uses: agntcy/dir/.github/actions/sign-record@main
  with:
    cids: ${{ steps.push.outputs.cids }}
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    github_token: ${{ secrets.GITHUB_TOKEN }}
    oidc_client_id: https://github.com/${{ github.repository }}/.github/workflows/my-workflow.yaml@${{ github.ref }}
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `cids` | JSON object (file→CID) from push-record, or newline-separated CIDs | Yes | - |
| `server_addr` | Directory server address (host:port). Omit to use dirctl default. | No | `""` |
| `github_token` | Optional GitHub PAT for Directory authentication | No | `""` |
| `oidc_client_id` | Workflow URL for keyless signing. Job must have `id-token: write`. | Yes | - |

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `success` | Whether all CIDs were signed successfully | `true` |
| `signed` | Number of CIDs successfully signed | `3` |
| `failed` | Number of CIDs that failed to sign | `0` |

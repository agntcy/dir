# Sign OASF Records Action

Sign record CIDs using GitHub OIDC (keyless). Use after **push-record** or **dirctl import**. Only supported in GitHub Actions; the job must have `permissions: id-token: write`.

## Usage

```yaml
- name: Sign records
  uses: agntcy/dir/.github/actions/sign-record@main
  with:
    cids: ${{ steps.push.outputs.cids }}
    server_addr: ${{ vars.DIRECTORY_ADDRESS }}
    auth_token: ${{ steps.oidc-dir.outputs.token }}
    oidc_client_id: https://github.com/${{ github.repository }}/.github/workflows/my-workflow.yaml@${{ github.ref }}
    max_retries: 3
    cleanup_on_failure: true
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `cids` | JSON object (file→CID) from push-record, or newline-separated CIDs | Yes | - |
| `server_addr` | Directory server address (host:port). Omit to use dirctl default. | No | `""` |
| `auth_token` | Optional OIDC bearer token for Directory authentication | No | `""` |
| `oidc_client_id` | Workflow URL for keyless signing. Job must have `id-token: write`. | Yes | - |
| `dirctl_path` | Optional absolute path to the dirctl binary | No | PATH lookup |
| `max_retries` | Sign attempts per CID (fresh Sigstore OIDC token each attempt) | No | `3` |
| `retry_delay_seconds` | Base delay between retries, multiplied by attempt number | No | `30` |
| `cleanup_on_failure` | Delete unsigned records with `dirctl delete` after all retries fail | No | `false` |

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `success` | Whether all CIDs were signed successfully | `true` |
| `signed` | Number of CIDs successfully signed | `3` |
| `cleaned` | Number of unsigned CIDs deleted after sign failure | `1` |
| `failed` | Number of CIDs still unsigned (sign and cleanup both failed) | `0` |
| `failed_cids` | Newline-separated CIDs that remain unsigned | - |
| `cleaned_cids` | Newline-separated CIDs deleted after sign failure | - |

## Retry and cleanup behavior

For each CID the action:

1. Attempts `dirctl sign` up to `max_retries` times, fetching a fresh Sigstore OIDC token before each attempt.
2. Waits `retry_delay_seconds * attempt` between retries.
3. If signing still fails and `cleanup_on_failure` is `true`, runs `dirctl delete` so a future import can push the record again.
4. Exits with code 1 if any CID remains unsigned on the server.

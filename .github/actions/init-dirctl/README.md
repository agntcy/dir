# Init dirctl

Runs `dirctl init --yes` to download and verify the local OASF taxonomy extractor
(~89 MB). Use before `dirctl import` with `enricher.extractor: {}`.

## Usage

```yaml
- name: Build dirctl
  id: dirctl
  uses: ./.github/actions/build-dirctl

- name: Init dirctl
  uses: ./.github/actions/init-dirctl
  with:
    dirctl_path: ${{ steps.dirctl.outputs.dirctl_path }}
```

## Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `dirctl_path` | Absolute path to dirctl | PATH lookup |
| `oasf_url` | OASF schema endpoint | `https://schema.oasf.outshift.com` |
| `asset_dir` | Local asset directory | `~/.agntcy/oasf-sdk/extractor` |

## Outputs

| Output | Description |
|--------|-------------|
| `asset_dir` | Path to the provisioned extractor assets |

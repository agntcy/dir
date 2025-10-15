# Setup DIRCTL in GitHub Action

An action that sets up `dirctl` CLI in GitHub Actions.

The GITHUB_TOKEN has to have "public repo" access.

## Usage

```yaml
- name: Setup dirctl
  uses: agntcy/dir/.github/actions/setup-dirctl@main
  with:
    # Default: latest
    version: v0.4.0
    # Default: linux
    os: linux
    # Default: amd64
    arch: amd64
```

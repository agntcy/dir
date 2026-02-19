# Setup dirctl Action

Install the dirctl CLI in your GitHub Actions workflow. Downloads the binary from GitHub releases and adds it to PATH.

## Usage

```yaml
- name: Setup dirctl
  uses: agntcy/dir/.github/actions/setup-dirctl@main
  with:
    version: v1.0.0-rc.4
    os: linux
    arch: amd64
```

## Inputs

| Input   | Description                             | Default  |
|---------|-----------------------------------------|----------|
| `version` | Release tag (e.g. v1.0.0) or "latest" | `latest` |
| `os`      | Operating system                      | `linux`  |
| `arch`    | Architecture                          | `amd64`  |

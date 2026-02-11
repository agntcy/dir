# Validate OASF Record Action

Validate agent records against the OASF schema in your CI/CD pipeline. This action fails the workflow with actionable error messages if validation fails.

## Usage

```yaml
name: Validate Agents

on:
  pull_request:
    paths:
      - '**/*.json'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup dirctl
        uses: agntcy/dir/.github/actions/setup-dirctl@main
      
      - name: Validate agent records
        uses: agntcy/dir/.github/actions/validate-record@main
        with:
          record_paths: |
            record.json
            agents/**/*.json
          fail_on_warning: true
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `record_paths` | Paths to OASF record JSON files (one per line, supports globs) | No | `record.json` |
| `schema_url` | URL of the OASF schema API | No | `https://schema.oasf.outshift.com` |
| `fail_on_warning` | Fail on validation warnings (not just errors) | No | `false` |

## Outputs

| Output | Description |
|--------|-------------|
| `valid` | Whether all records passed validation (`true` or `false`) |
| `validated_files` | JSON array of successfully validated file paths |
| `failed_files` | JSON array of file paths that failed validation |

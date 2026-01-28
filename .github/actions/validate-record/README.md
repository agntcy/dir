# Validate OASF Record Action

Validate agent records against the OASF schema in your CI/CD pipeline. This action fails the workflow with actionable error messages if validation fails.

## Features

- Validates OASF agent records against the official schema
- Supports multiple paths and glob patterns
- Provides detailed error messages with file annotations
- No Directory server connection required (schema validation only)

## Usage

### Basic Usage

```yaml
# Validates record.json in repo root (default)
- uses: agntcy/dir/.github/actions/validate-record@main
```

### Multiple Files

```yaml
- uses: agntcy/dir/.github/actions/validate-record@main
  with:
    record_paths: |
      agents/agent-a.json
      agents/agent-b.json
      services/*.json
```

### Full Example (PR Validation)

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
      
      - name: Validate agent records
        uses: agntcy/dir/.github/actions/validate-record@main
        with:
          record_paths: |
            record.json
            agents/**/*.json
          fail_on_warning: "true"
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `record_paths` | Paths to OASF record JSON files (one per line, supports globs) | No | `record.json` |
| `schema_url` | URL of the OASF schema API | No | `https://schema.oasf.outshift.com` |
| `fail_on_warning` | Fail on validation warnings (not just errors) | No | `false` |
| `dirctl_version` | Version of dirctl to use | No | `latest` |

## Outputs

| Output | Description |
|--------|-------------|
| `valid` | Whether all records passed validation (`true` or `false`) |
| `validated_files` | JSON array of successfully validated file paths |
| `failed_files` | JSON array of file paths that failed validation |

## Error Messages

The action provides detailed error messages as GitHub annotations:

```
Error: agent.json
  Validation failed:
    1. Field 'name' is required
    2. Field 'version' must match pattern '^v[0-9]+\.[0-9]+\.[0-9]+$'
```

These annotations appear directly on the PR diff, making it easy to identify and fix issues.

## Example Agent Record

```json
{
  "name": "my-org/my-agent",
  "version": "v1.0.0",
  "schema_version": "0.8.0",
  "description": "An example agent that does something useful",
  "authors": ["My Organization"],
  "skills": [
    {
      "name": "natural_language_processing/text_generation",
      "id": 10101
    }
  ],
  "domains": [
    {
      "name": "technology/software_development"
    }
  ]
}
```

## Local Testing

You can test the validation script locally:

```bash
RECORD_PATHS="record.json" \
SCHEMA_URL="https://schema.oasf.outshift.com" \
.github/actions/validate-record/validate.sh
```

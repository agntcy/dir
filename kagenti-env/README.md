# Kagenti Operator - Task Automation

Automate installation and management using [Task](https://taskfile.dev/).

## Prerequisites

```bash
# macOS
brew install go-task kind kubectl helm
```

## Install

```bash
# Create kind cluster + install everything (cert-manager, tekton, operator)
task install:all

# Or install on existing cluster
task install CLUSTER_NAME=my-cluster

# Install a specific chart version
task install CLUSTER_NAME=my-cluster CHART_VERSION=0.2.0-alpha.18
```

## Delete

```bash
# Uninstall operator only
task uninstall CLUSTER_NAME=my-cluster

# Uninstall everything (operator + tekton + cert-manager)
task uninstall:all CLUSTER_NAME=my-cluster

# Delete the entire kind cluster
task kind:delete CLUSTER_NAME=my-cluster
```

## Useful Commands

```bash
# Show all available tasks
task --list

# Check status
task status CLUSTER_NAME=my-cluster

# View operator logs
task logs CLUSTER_NAME=my-cluster

# Install from local chart (development)
task install:operator:local CLUSTER_NAME=my-cluster
```

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLUSTER_NAME` | `kagenti` | Kind cluster name |
| `NAMESPACE` | `kagenti-system` | Operator namespace |
| `CHART_VERSION` | `0.2.0-alpha.18` | Helm chart version |
| `CHART_REPO` | `oci://ghcr.io/kagenti/kagenti-operator` | OCI Helm chart repository |

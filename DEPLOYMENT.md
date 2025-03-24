# Deployment

To deploy the Directory, you can use the provided `Taskfile` commands to start the necessary services and deploy the Directory server.
Alternatively, you can deploy from a GitHub Helm chart release.

## Local Deployment

To start a local OCI registry server for storage and the Directory server, use the following commands:

```bash
task server:store:start
task server:start
```

These commands will set up a local environment for development and testing purposes.

## Remote Deployment

To deploy the Directory into an existing Kubernetes cluster, use a released Helm chart from GitHub with the following commands:

```bash
helm pull oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.1.3
helm upgrade --install dir oci://ghcr.io/agntcy/dir/helm-charts/dir --version v0.1.3
```

These commands will pull the latest version of the Directory Helm chart from the GitHub Container Registry and install or upgrade the Directory in your Kubernetes cluster. Ensure that your Kubernetes cluster is properly configured and accessible before running these commands. The `helm upgrade --install` command will either upgrade an existing release or install a new release if it does not exist.

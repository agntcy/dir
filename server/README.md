# Directory Server

## Configuration

The Directory server supports configuration via environment variables, YAML configuration files, or both. Environment variables follow the `DIRECTORY_SERVER_` prefix convention.

### Record validators

Record validation is configured as a YAML list. Each entry names a provider, the operations it runs on (`push`, `autosync`, `index`), and a provider-specific `config`. An empty list disables record validation. This list cannot be set via environment variables.

**Example with YAML configuration:**
```yaml
# server.config.yml
validators:
  - provider: oasf
    op: ["push", "autosync", "index"]
    config:
      schema_url: "https://schema.oasf.outshift.com"
listen_address: "0.0.0.0:8888"
```

The daemon nests the same list under `server.validators`. Helm exposes it as a top-level `apiserver.validators` value and injects it into both the apiserver and the reconciler.

#### Testing with Local OASF Server

To test with a local OASF instance deployed alongside the directory server:

1. **Enable OASF in Helm values** - Edit `install/charts/dir/values.yaml`:
   ```yaml
   apiserver:
     oasf:
       enabled: true
   ```

2. **Set schema URL to use the deployed OASF instance** - In the same file, set:
   ```yaml
   apiserver:
     validators:
       - provider: oasf
         op: ["push", "autosync", "index"]
         config:
           schema_url: "http://dir-ingress-controller.dir-server.svc.cluster.local"
   ```
   Replace `dir` with your Helm release name and `dir-server` with your namespace if different.

3. **Deploy**:
   Follow the [Kubernetes Deployment](https://docs.agntcy.org/dir/dir-deployment-kubernetes/) guide
   to create a Kind cluster and install the Directory Helm chart with the OASF subchart enabled.

The OASF instance will be deployed as a subchart in the same namespace and automatically configured for multi-version routing via ingress.

#### Using a Locally Built OASF Image

If you want to deploy with a locally built OASF image (e.g., containing `0.9.0-dev` schema files), you need to load the image into Kind **before** deploying. Recreating the Kind cluster after loading a custom OASF image will discard it. Follow these steps:

1. **Create the Kind cluster first**:
   ```bash
   kind create cluster --name agntcy-cluster
   task build
   ```

2. **Build and tag your local OASF image**:
   ```bash
   cd /path/to/oasf/server
   docker build -t ghcr.io/agntcy/oasf-server:latest .
   ```

3. **Load the OASF image into Kind**:
   ```bash
   kind load docker-image ghcr.io/agntcy/oasf-server:latest --name agntcy-cluster
   ```

4. **Configure values.yaml** to use the local image:
   ```yaml
   oasf:
     enabled: true
     image:
       repository: ghcr.io/agntcy/oasf-server
       versions:
         - server: latest
           schema: 0.9.0-dev
           default: true
   ```

5. **Deploy with Helm** (without recreating the cluster):
   ```bash
   helm upgrade --install dir ./install/charts/dir \
     -f ./install/charts/dir/values.yaml \
     -n dir-server --create-namespace
   ```

**Note**: If you update the local OASF image, reload it into Kind and restart the deployment:
```bash
kind load docker-image ghcr.io/agntcy/oasf-server:latest --name agntcy-cluster
kubectl rollout restart deployment/dir-oasf-0-9-0-dev -n dir-server
```

### Other Configuration Options

For complete server configuration including authentication, authorization, storage, routing, and database options, see the [server configuration reference](./config/config.go).

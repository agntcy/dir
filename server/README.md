# Directory Server

## Configuration

The Directory server reads the canonical dir configuration (see [`github.com/agntcy/dir/config`](../config)). Configuration can be provided via YAML files, environment variables, or both. All environment variables share the unified `DIRECTORY_` prefix; legacy `DIRECTORY_SERVER_*` and `RECONCILER_*` prefixes are no longer supported.

### OASF Validation Configuration

The server validates all records server-side. Records are validated using the configured OASF schema URL.

- **`oasf_api_validation.schema_url`** / **`DIRECTORY_OASF_API_VALIDATION_SCHEMA_URL`** - OASF schema URL for API-based validation (required)
  - **Default**: `https://schema.oasf.outshift.com`
  - URL of the OASF server to use for validation
  - This affects all record validation operations including push, sync, and import

**Example with environment variables:**
```bash
# Use default OASF API validator
./dirctl-apiserver

# Use custom OASF server
DIRECTORY_OASF_API_VALIDATION_SCHEMA_URL=http://localhost:8080 ./dirctl-apiserver

# Use custom OASF server
DIRECTORY_OASF_API_VALIDATION_SCHEMA_URL="http://localhost:8080" ./dirctl-apiserver
```

**Example with YAML configuration:**
```yaml
# /etc/agntcy/dir/dir.config.yml
oasf_api_validation:
  schema_url: "https://schema.oasf.outshift.com"
apiserver:
  listen_address: "0.0.0.0:8888"
```

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
     config:
       oasf_api_validation:
         schema_url: "http://dir-ingress-controller.dir-server.svc.cluster.local"
   ```
   Replace `dir` with your Helm release name and `dir-server` with your namespace if different.

3. **Deploy**:
   ```bash
   task build
   task deploy:local
   ```

The OASF instance will be deployed as a subchart in the same namespace and automatically configured for multi-version routing via ingress.

#### Using a Locally Built OASF Image

If you want to deploy with a locally built OASF image (e.g., containing `0.9.0-dev` schema files), you need to load the image into Kind **before** deploying. The `task deploy:local` command automatically creates a cluster and loads images, but it doesn't load custom OASF images. Follow these steps:

1. **Create the Kind cluster first**:
   ```bash
   task test-env:kubernetes:setup-cluster
   ```
   This creates the cluster and loads the Directory server images.

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

5. **Deploy with Helm** (don't use `task deploy:local` as it will recreate the cluster):
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

For the full canonical schema (authentication, authorization, storage, routing, database, reconciler tasks) see [`config/config.go`](../config/config.go) in the top-level `config` module.

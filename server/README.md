# Directory Server

## Configuration

The Directory server supports configuration via environment variables, YAML configuration files, or both. Environment variables follow the `DIRECTORY_SERVER_` prefix convention.

### OASF Validation Configuration

The server includes built-in OASF record validation with support for API-based validation:

- **`schema_url`** / **`DIRECTORY_SERVER_SCHEMA_URL`** - OASF schema URL for API-based validation
  - **Default**: `https://schema.oasf.outshift.com`
  - URL of the OASF server to use for validation
  - This affects all record validation operations including push, sync, and import

- **`disable_api_validation`** / **`DIRECTORY_SERVER_DISABLE_API_VALIDATION`** - Disable API-based validation
  - **Default**: `false` (API validation enabled)
  - When `true`, uses embedded schemas instead of the API validator
  - When `false`, uses API validation with the configured `schema_url`

**Example with environment variable:**
```bash
# Use default OASF API validator (default behavior)
./dirctl-apiserver

# Use custom OASF server
DIRECTORY_SERVER_SCHEMA_URL=http://localhost:8080 ./dirctl-apiserver

# Use embedded schemas (disable API validator)
DIRECTORY_SERVER_DISABLE_API_VALIDATION=true ./dirctl-apiserver
```

**Example with YAML configuration:**
```yaml
# server.config.yml
schema_url: "https://schema.oasf.outshift.com"
disable_api_validation: false
listen_address: "0.0.0.0:8888"
```

#### Testing with Local OASF Server

To test with a local OASF instance (e.g., for schema development or debugging):

```bash
# 1. Deploy OASF (in separate terminal/repo)
cd /path/to/agntcy/oasf
HELM_VALUES_PATH=./install/charts/oasf/values-test-versions.yaml task up

# 2. Remove host restriction from OASF ingress (allows cross-cluster access)
kubectl --context kind-test-oasf-cluster patch ingress oasf-api -p '{"spec":{"rules":[{"http":{"paths":[{"path":"/api/0.8.0(/|$)(.*)","pathType":"ImplementationSpecific","backend":{"service":{"name":"oasf-0-8-0","port":{"number":8080}}}},{"path":"/api(/|$)(.*)","pathType":"ImplementationSpecific","backend":{"service":{"name":"oasf-0-8-0","port":{"number":8080}}}}]}}]}}'

# 3. Get OASF node IP and deploy Directory
cd /path/to/agntcy/dir
OASF_IP=$(docker inspect test-oasf-cluster-control-plane -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')
task build
task deploy:local DIRECTORY_SERVER_SCHEMA_URL=http://${OASF_IP}:30080
```

**Note:** Update `oasf/install/charts/oasf/values-test-versions.yaml` with desired OASF versions before deploying. The ingress patch removes the host restriction to allow cross-cluster access.

### Other Configuration Options

For complete server configuration including authentication, authorization, storage, routing, and database options, see the [server configuration reference](./config/config.go).

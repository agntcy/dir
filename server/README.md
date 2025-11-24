# Directory Server

## Configuration

The Directory server supports configuration via environment variables, YAML configuration files, or both. Environment variables follow the `DIRECTORY_SERVER_` prefix convention.

### OASF Validation Configuration

The server validates all records server-side. By default, records are validated using API validation in strict mode. This ensures consistent, strict validation for all records regardless of their source.

- **`schema_url`** / **`DIRECTORY_SERVER_SCHEMA_URL`** - OASF schema URL for API-based validation
  - **Default**: `https://schema.oasf.outshift.com`
  - URL of the OASF server to use for validation
  - This affects all record validation operations including push, sync, and import

- **`disable_api_validation`** / **`DIRECTORY_SERVER_DISABLE_API_VALIDATION`** - Use embedded schema validation instead of API validator
  - **Default**: `false` (uses API validation)
  - When `true`, uses embedded schemas for validation (no HTTP calls to OASF server)

- **`strict_validation`** / **`DIRECTORY_SERVER_STRICT_VALIDATION`** - Use strict validation mode
  - **Default**: `true` (strict mode - fails on warnings)
  - When `false`, uses lax validation mode (allows warnings, only fails on errors)
  - Only applies when `disable_api_validation` is `false`

**Example with environment variables:**
```bash
# Use default OASF API validator with strict validation (default behavior)
./dirctl-apiserver

# Use custom OASF server
DIRECTORY_SERVER_SCHEMA_URL=http://localhost:8080 ./dirctl-apiserver

# Use embedded schema validation (no API calls)
DIRECTORY_SERVER_DISABLE_API_VALIDATION=true ./dirctl-apiserver

# Use lax API validation (allows warnings)
DIRECTORY_SERVER_STRICT_VALIDATION=false ./dirctl-apiserver
```

**Example with YAML configuration:**
```yaml
# server.config.yml
schema_url: "https://schema.oasf.outshift.com"
disable_api_validation: false
strict_validation: true
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

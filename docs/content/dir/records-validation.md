# Records and Validation

## ADS Records

Directory uses [Open Agent Schema Framework](https://schema.oasf.outshift.com) (OASF) which defines a standardized schema for representing agents and their capabilities using [OASF Record specification](https://schema.oasf.outshift.com/objects/record). This ensures interoperability and consistency across different implementations of the directory service.

### Content Identifier

The content identifier of the record is a [Content IDentifier](https://github.com/multiformats/cid) (CID) hash digest which makes it:

- Globally unique
- Content-addressable
- Collision-resistant
- Immutable

### Verifiable Names

Records must include a `name` field with a domain-based identifier that enables name verification. When a record uses a verifiable name:

- The name must include a protocol prefix: `https://domain/path` or `http://domain/path`.
- The domain must host a JWKS file at `<scheme>://<domain>/.well-known/jwks.json`.
- Records signed with a private key associated with a public key present in that JWKS file can be verified as authorized by the domain.

See the [Directory CLI documentation](directory-cli.md#name-verification) for details on name verification workflows.

### Example Email Agent

You can generate your own example records using the [OASF Record Sample generator](https://schema.oasf.outshift.com/sample/objects/record). Below is an example OASF record for an email agent that is capable of sending and receiving emails.

```json
{
  "schema_version": "1.0.0",
  "name": "https://www.cisco.com/agents/email-agent",
  "version": "v1.0.0",
  "authors": ["Cisco Systems Inc."],
  "description": "An agent that can send and receive emails.",
  "created_at": "2025-08-11T16:20:37.159072Z",
  "skills": [
    {
      "id": 10306,
      "name": "natural_language_processing/information_retrieval_synthesis/information_retrieval_synthesis_search"
    },
    {
      "id": 10202,
      "name": "natural_language_processing/natural_language_generation/summarization"
    },
    {
      "id": 60103,
      "name": "retrieval_augmented_generation/retrieval_of_information/document_retrieval"
    }
  ],
  "locators": [
    {
      "urls": [
        "https://github.com/agntcy/agentic-apps/tree/main/email_reviewer"
      ],
      "type": "source_code"
    },
    {
      "urls": [
        "https://github.com/agntcy/agentic-apps/tree/main/email_reviewer/pyproject.toml"
      ],
      "type": "package"
    }
  ]
}
```

!!! note

    The `name` field uses a verifiable domain-based format (`https://cisco.com/agents/email-agent`). When signed with a key authorized by the domain's JWKS file at `https://cisco.com/.well-known/jwks.json`, this record can be pulled using the convenient reference `cisco.com/agents/email-agent:v1.0.0` instead of its CID.

## Validation

The Directory enforces validation on all records before accepting them.
Validation is performed using the [OASF SDK](https://github.com/agntcy/oasf), which requires an OASF schema server URL for validation.

### Configuration

The Directory server validates records using an OASF schema URL. By default, it uses `https://schema.oasf.outshift.com`, but you can configure a different OASF instance:

**Using environment variables:**

```bash
# Use default OASF instance (https://schema.oasf.outshift.com)
task server:start

# Use custom OASF instance
DIRECTORY_SERVER_OASF_API_VALIDATION_SCHEMA_URL=https://your-custom-oasf.com task server:start
```

**Using YAML configuration:**

```yaml
# server.config.yml
oasf_api_validation:
  schema_url: "https://schema.oasf.outshift.com"
listen_address: "0.0.0.0:8888"
```

!!! note
    
    The server itself does not have a built-in default schema URL. Deployment tools like Helm and Taskfile set `https://schema.oasf.outshift.com` as the default. When using Docker Compose or running the server binary directly, you must explicitly set the `DIRECTORY_SERVER_OASF_API_VALIDATION_SCHEMA_URL` environment variable.

### Validation Behavior

The Directory server uses API-based validation against the configured OASF schema server:

- **Validation Method**: HTTP requests to the OASF schema server API.
- **Errors vs Warnings**: Only errors cause validation to fail. Warnings are returned but do not affect the validation result.
- **Schema Version Detection**: The schema version is automatically detected from each record's `schema_version` field.
- **Supported Versions**: The OASF SDK decoder supports specific schema versions (currently: 0.7.0, 0.8.0, and 1.0.0). Records using unsupported versions are not validated.
- **Unknown Classes**: Classes (such as modules, skills, and domains) not defined in the OASF schema are rejected with an error. Records must only use classes that are defined in the configured OASF schema instance.

### OASF Instance Configurations

While there is only one validation method (API validation), you can configure the Directory server to use different OASF instances.
The choice of OASF instance affects which records are accepted and how compatible your directory instance is with other directory instances.

The following table shows which OASF instance configurations can exchange records with each other:

| Instance Type | Can Pull From | Can Be Pulled By |
|--------------|---------------|------------------|
| **Official OASF Instance** | Official OASF instance only | Official OASF instance and custom instances with additional taxonomy |
| **Custom OASF Instance (Additional Taxonomy)** | Official OASF instance, custom instances with same extended taxonomy | Custom instances with same extended taxonomy only |
| **Custom OASF Instance (Changed Taxonomy)** | Custom instances with same changed taxonomy only | Custom instances with same changed taxonomy only |

#### Official OASF Instance

Using the official OASF instance (`https://schema.oasf.outshift.com`) provides the baseline for the official network.
Records validated here form the most strict, compatible set.

**Configuration:**

```yaml
oasf_api_validation:
  schema_url: "https://schema.oasf.outshift.com"
```

#### Custom OASF Instance (Additional Taxonomy)

Using a custom OASF instance with extensions that add to the taxonomy (with modules, skills, and domains that extend the official taxonomy but doesn't change or remove).

Records using the extended taxonomy can only be pulled by nodes using the exact same extended taxonomy.

**Configuration:**

```yaml
oasf_api_validation:
  schema_url: "https://your-custom-oasf-instance.com"
```

#### Custom OASF Instance (Changed Taxonomy)

Using a custom OASF instance with extensions that modify the taxonomy (with modules, skills, and domains that change or remove from the official taxonomy).

This approach is completely incompatible with all other options, can only work with nodes using the exact same changed taxonomy.

**Configuration:**

```yaml
oasf_api_validation:
  schema_url: "https://your-custom-oasf-instance.com"
```

### Deploying a Local OASF Instance

You can deploy a local OASF instance alongside the Directory server for testing or development purposes.

#### Testing with Local OASF Server

To test with a local OASF instance deployed alongside the directory server:

1. Enable OASF in Helm values

    Edit `install/charts/dir/values.yaml` with the following configuration:

    ```yaml
    apiserver:
      oasf:
        enabled: true
    ```

2. Set schema URL to use the deployed OASF instance

    In the same file, set the schema URL to use the deployed OASF instance:

    ```yaml
    apiserver:
      config:
        oasf_api_validation:
          schema_url: "http://dir-ingress-controller.dir-server.svc.cluster.local"
    ```

    Replace `dir` with your Helm release name and `dir-server` with your namespace if different.

3. Deploy the local OASF instance

    ```bash
    task build
    task deploy:local
    ```

The OASF instance is deployed as a subchart in the same namespace and automatically configured for multi-version routing via ingress.

#### Using a Locally Built OASF Image

If you want to deploy with a locally built OASF image (e.g., containing `0.9.0-dev` schema files), you need to load the image into Kind before deploying.
The `task deploy:local` command automatically creates a cluster and loads images, but it doesn't load custom OASF images.

Follow the steps below:

1. Create the Kind cluster

    ```bash
    task deploy:kubernetes:setup-cluster
    ```

    This creates the cluster and loads the Directory server images.

2. Build your local OASF image with the `latest` tag

    ```bash
    cd /path/to/oasf/server
    task build
    ```

3. Load the OASF image into Kind

    ```bash
    kind load docker-image ghcr.io/agntcy/oasf-server:latest --name agntcy-cluster
    ```

4. Configure `values.yaml` to use the local image:

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

5. Deploy the Directory

    Don't use `task deploy:local` as it will recreate the cluster.

    ```bash
    task deploy:kubernetes:dir
    ```

!!! note

    If you update the local OASF image, reload it into Kind and restart the deployment:

    ```bash
    kind load docker-image ghcr.io/agntcy/oasf-server:latest --name agntcy-cluster
    kubectl rollout restart deployment/dir-oasf-0-9-0-dev -n dir-server
    ```

### Related Documentation

For more information, see the following:

- [OASF Validation Service](https://github.com/agntcy/oasf) - Detailed validation service documentation
- [Validation Comparison](https://github.com/agntcy/oasf) - Comparison between API validator and JSON Schema
- [OASF Extensions](https://github.com/agntcy/oasf/blob/main/CONTRIBUTING.md#oasf-extensions) - Information about creating OASF extensions

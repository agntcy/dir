# Usage

This document defines a basic overview of main Directory features, components, and usage scenarios.
All code snippets below are tested against the Directory `v0.2.0` release.

> Although the following example is shown for a CLI-based usage scenario,
there is an effort to expose the same functionality via language-specific SDKs.

## Prerequisites

- Directory CLI client (`dirctl`), distributed via [GitHub Releases](https://github.com/agntcy/dir/releases)
- Directory API server, outlined in the [Readme Deployment](README.md#deployment) section

### Build

This example demonstrates how to define a Record using provided tooling to prepare for publication.

To start, generate an example Record that matches the data model schema defined in [Record](https://buf.build/agntcy/oasf/docs/main:objects.v3#objects.v3.Record) specification using the [OASF Record Sample generator](https://schema.oasf.outshift.com/sample/0.5.0/objects/record).

```bash
# Generate an example data model
cat << EOF > record.json
{
    "name": "record",
    "version": "v1.0.0",
    "description": "insert description here",
    "schema_version": "v0.5.0",
    "skills": [
        {
            "id": 302,
            "name": "schema.oasf.agntcy.org/skills/audio_to_audio"
        }
    ],
    "authors": [
        "Jane Doe"
    ],
    "created_at": "2025-08-11T16:20:37.159072Z",
    "locators": [
        {
            "type": "source_code",
            "url": "https://github.com/agntcy/oasf/blob/main/record"
        }
    ]
}
EOF
```

### Store

This example demonstrates the interaction with the local storage layer using the CLI client.
The storage layer is used as a content-addressable object store for Directory-specific models and serves both the local and network-based operations (if enabled).

```bash
# Push the record and store its CID to a file
dirctl push record.json > record.cid

# Set the CID as a variable for easier reference
RECORD_CID=$(cat record.cid)

# Pull the record
# Returns the same data as record.json
dirctl pull $RECORD_CID

# Lookup basic metadata about the record
# Returns annotations, creation timestamp and OASF schema version
dirctl info $RECORD_CID
```

### Signing and Verification

#### Method 1: OIDC-based Interactive

This process relies on creating and uploading to the OCI registry a signature for the record using identity-based OIDC signing flow which can later be verified.
The signing process opens a browser window to authenticate the user with an OIDC identity provider.
These operations are implemented using [Sigstore](https://www.sigstore.dev/).

```bash
# Push record with signature
dirctl push record.json --sign

# Verify record
dirctl verify $RECORD_CID
```

#### Method 2: OIDC-based Non-Interactive

This method is designed for automated environments such as CI/CD pipelines where browser-based authentication is not available. It uses OIDC tokens provided by the execution environment (like GitHub Actions) to sign records. The signing process uses a pre-obtained OIDC token along with provider-specific configuration to establish identity without user interaction.

```
      - name: Push and sign record
        run: |
          bin/dirctl push record.json --sign \
            --oidc-token ${{ steps.oidc-token.outputs.token }} \
            --oidc-provider-url "https://token.actions.githubusercontent.com" \
            --oidc-client-id "https://github.com/${{ github.repository }}/.github/workflows/demo.yaml@${{ github.ref }}"

      - name: Run verify command
        run: |
          echo "Running dir verify command"
          bin/dirctl verify $RECORD_CID
```

#### Method 3: Self-Managed Keys

This method is suitable for non-interactive use cases, such as CI/CD pipelines, where browser-based authentication is not possible or desired. Instead of OIDC, a signing keypair is generated (e.g., with Cosign), and the private key is used to sign the record.

```bash
# Generate a key-pair for signing
# This creates 'cosign.key' (private) and 'cosign.pub' (public)
cosign generate-key-pair

# Set COSIGN_PASSWORD shell variable if you password protected the private key
export COSIGN_PASSWORD=your_password_here
# Push record with signature 
dirctl push record.json --sign --key cosign.key

# Verify the signed record
dirctl verify $RECORD_CID
```

### Announce

This example demonstrates how to publish records to allow content discovery across the network.
To avoid stale data, it is recommended to republish the data periodically
as the data across the network has TTL.

Note that this operation only works for the objects already pushed to the local storage layer, i.e., it is required to first push the data before publication.

```bash
# Publish the record across the network
dirctl publish $RECORD_CID
```

If the data is not published to the network, it cannot be discovered by other peers.
For published data, peers may try to reach out over the network
to request specific objects for verification and replication.
Network publication may fail if you are not connected to the network.

### Discover

This example demonstrates how to discover published data locally or across the network.
The API supports both unicast- mode for routing to specific objects,
and multicast- mode for attribute-based matching and routing.

There are two modes of operation, a) local mode where the data is queried from the local data store, and b) network mode where the data is queried across the network.

Discovery is performed using full-set label matching, i.e., the results always fully match the requested query.
Note that it is not guaranteed that the returned data is available, valid, or up to date.

```bash
# Get a list of peers holding a specific record
dirctl list --digest $DIGEST

#> Peer 12D3KooWQffoFP8ePUxTeZ8AcfReTMo4oRPqTiN1caDeG5YW3gng
#>   Digest: sha256:<hash>
#>   Labels: /skills/Text Generation, /skills/Fact Extraction

# Discover the records in your local data store
dirctl list "/skills/Text Generation"
dirctl list "/skills/Text Generation" "/skills/Fact Extraction"

#> Peer HOST
#>   Digest: sha256:<hash>
#>   Labels: /skills/Text Generation, /skills/Fact Extraction

# Discover the records across the network
dirctl list "/skills/Text Generation" --network
dirctl list "/skills/Text Generation" "/skills/Fact Extraction" --network
```

It is also possible to get an aggregated summary of the data held in your local data store or across the network.
This is used for routing decisions when traversing the network.

```bash
# Get label summary details in your local data store
dirctl list info

#> Peer HOST | Label: /skills/Text Generation | Total: 1
#> Peer HOST | Label: /skills/Fact Extraction | Total: 1

# Get label summary details across the network
dirctl list info --network
```

### Search

This example demonstrates how to search for records in the directory using various filters and query parameters.
The search functionality allows you to find records based on specific attributes like name, version, skills, locators, and extensions using structured query filters.

Search operations support pagination and return Content Identifier (CID) values that can be used with other Directory commands like `pull`, `info`, and `verify`.

```bash
# Basic search for records by name
dirctl search --query "name=my-agent-name"

# Search for records with a specific version
dirctl search --query "version=v1.0.0"

# Search for records that have a particular skill by ID
dirctl search --query "skill-id=10201"

# Search for records with a specific skill name
dirctl search --query "skill-name=Text Generation"

# Search for records with a specific locator type and URL
dirctl search --query "locator=docker-image:https://example.com/my-agent"

# Search for records with a specific extension
dirctl search --query "extension=my-custom-extension:v1.0.0"

# Combine multiple query filters (AND operation)
dirctl search \
  --query "name=my-agent" \
  --query "version=v1.0.0" \
  --query "skill-name=Text Generation"

# Use pagination to limit results and specify offset
dirctl search \
  --query "skill-name=Text Generation" \
  --limit 10 \
  --offset 0

# Get the next page of results
dirctl search \
  --query "skill-name=Text Generation" \
  --limit 10 \
  --offset 10
```

**Available Query Types:**

- `name` - Search by record name
- `version` - Search by record version  
- `skill-id` - Search by skill ID number
- `skill-name` - Search by skill name
- `locator` - Search by locator (format: `type:url`)
- `extension` - Search by extension (format: `name:version`)

**Query Format:**

All queries use the format `field=value`. Multiple queries are combined with AND logic, meaning results must match all specified criteria.

### Sync

The sync feature enables one-way synchronization of records and other objects between remote Directory instances and your local node. This feature supports distributed AI agent ecosystems by allowing you to replicate content from multiple remote directories, creating local mirrors for offline access, backup, and cross-network collaboration.

**How Sync Works**: Directory leverages [Zot](https://zotregistry.dev/), a cloud-native OCI registry, as the underlying synchronization engine. When you create a sync operation, the system dynamically configures Zot's sync extension to pull content from remote registries. Objects are stored as OCI artifacts (manifests, blobs, and tags), enabling container-native synchronization with automatic polling, retry mechanisms, and secure credential exchange between Directory nodes.

This example demonstrates how to synchronize records between remote directories and your local instance.

```bash
# Create a sync operation to start periodic poll from remote
dirctl sync create https://remote-directory.example.com:8888

# List all sync operations
dirctl sync list

# Check the status of a specific sync operation
dirctl sync status <sync id>

# Delete a sync operation to stop periodic poll from remote
dirctl sync delete <sync id>
```

### gRPC Error Codes

The following table lists the gRPC error codes returned by the server APIs, along with a description of when each code is used:

| Error Code                 | Description                                                                                                                                                           |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `codes.InvalidArgument`    | Returned when the client provides an invalid or malformed argument, such as a missing or invalid record reference or record.                                           |
| `codes.NotFound`           | Returned when the requested object does not exist in the local store or across the network.                                                                           |
| `codes.FailedPrecondition` | Returned when the server environment or configuration is not in the required state (e.g., failed to create a directory or temp file, unsupported provider in config). |
| `codes.Internal`           | Returned for unexpected internal errors, such as I/O failures, serialization errors, or other server-side issues.                                                     |
| `codes.Canceled`           | Returned when the operation is canceled by the client or context expires.                                                                                             |

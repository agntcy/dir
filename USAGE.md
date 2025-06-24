# Usage

This document defines a basic overview of main Directory features, components, and usage scenarios.
All code snippets below are tested against the Directory `v0.2.0` release.

> Although the following example is shown for a CLI-based usage scenario,
there is an effort to expose the same functionality via language-specific SDKs.

## Prerequisites

- Directory CLI client, distributed via [GitHub Releases](https://github.com/agntcy/dir/releases)
- Directory API server, outlined in the [Readme Deployment](README.md#deployment) section.

### Build

This example demonstrates how to define an agent data model and build such models using provided tooling to prepare for publication.

To start, generate an example agent that matches the data model schema defined in [Agent Data Model](api/proto/core/v1alpha1/agent.proto) specification.

```bash
# Generate an example data model
cat << EOF > model.json
{
 "name": "my-agent",
 "skills": [
    {"category_name": "Text Generation"},
    {"category_name": "Fact Extraction"}
 ]
}
EOF
```

Alternatively, build the same agent data model using the CLI client.
The build process allows the execution of additional user-defined operations,
which is useful for data model enrichment and other custom use cases.

```bash
# Use the model above as the base model
mv model.json model.base.json

# Define the build config
cat << EOF > build.config.yml
builder:
 # Base agent model path
 base-model: "model.base.json"

 # Disable the LLMAnalyzer plugin
 llmanalyzer: false

 # Disable the runtime plugin
 runtime: false

 # Disable the pyprojectparser plugin
 pyprojectparser: false

 # Enable OASF validation
 oasf-validation: true 
EOF

# Build the agent
dirctl build . > model.json

# Preview built agent
cat model.json
```

### Signing and Verification

#### Method 1: OIDC-based Interactive

This process relies on attaching signature to the agent data model using identity-based OIDC signing flow which can be verified by other clients.
The signing process opens a browser window to authenticate the user
with an OIDC identity provider.
The verification process validates the agent signature against the identity provider and signature transparency services.
These operations are implemented using [Sigstore](https://www.sigstore.dev/).

```bash
## Sign the agent data model
cat model.json | dirctl sign --stdin > signed.model.json

## Verify agent data models
cat model.json | dirctl verify --stdin
cat signed.model.json | dirctl verify --stdin

## Verify signature using custom parameters:
# 1. Only trust users with "cisco.com" addresses
# 2. Only trust issuers from "github.com"
dirctl verify signed.model.json \
   --oidc-identity "(.*)@cisco.com" \
   --oidc-issuer "(.*)github.com(.*)"

## Replace the base agent model with a signed one
rm -rf model.json
mv signed.model.json model.json
```

#### Method 2: OIDC-based Non-Interactive

This method is designed for automated environments such as CI/CD pipelines where browser-based authentication is not available. It uses OIDC tokens provided by the execution environment (like GitHub Actions) to sign agent data models. The signing process uses a pre-obtained OIDC token along with provider-specific configuration to establish identity without user interaction. The verification process validates the agent signature against the specified OIDC issuer and identity pattern.

```
      - name: Run sign command
        run: |
          echo "Running dir sign command"
          bin/dirctl sign agent.json \
            --oidc-token ${{ steps.oidc-token.outputs.token }} \
            --oidc-provider-url "https://token.actions.githubusercontent.com" \
            --oidc-client-id "https://github.com/${{ github.repository }}/.github/workflows/demo.yaml@${{ github.ref }}" \
            --stdin > signed.model.json
          echo "Signed agent.json to signed.model.json"
          cat signed.model.json
          mv signed.model.json agent.json

      - name: Run verify command
        run: |
          echo "Running dir verify command"
          bin/dirctl verify agent.json \
            --oidc-issuer "https://token.actions.githubusercontent.com" \
            --oidc-identity "https://github.com/${{ github.repository }}-custom/.github/workflows/demo.yaml@${{ github.ref }}"
```

#### Method 3: Self-Managed Keys

This method is suitable for non-interactive use cases, such as CI/CD pipelines, where browser-based authentication is not possible or desired. Instead of OIDC, a signing keypair is generated (e.g., with Cosign), and the private key is used to sign the agent model. The corresponding public key is then required to verify the agent, therefore, it must be distributed to any party that needs to verify signed agent models.

```bash
# Generate a key-pair for signing
# This creates 'cosign.key' (private) and 'cosign.pub' (public)
cosign generate-key-pair

# Sign the agent data model using the private key:
cat model.json | dirctl sign --stdin --key cosign.key > signed.model.json

# Verify the signed agent using the public key:
cat signed.model.json | dirctl verify --stdin --key cosign.pub

# (Optional) Replace the base agent model with the signed one:
rm -rf model.json
mv signed.model.json model.json
```

### Store

This example demonstrates the interaction with the local storage layer using the CLI client.
The storage layer is used as a content-addressable object store for Directory-specific models and serves both the local and network-based operations (if enabled).

```bash
# Push and store content digest
dirctl push model.json > model.digest
DIGEST=$(cat model.digest)

# Pull the agent
# Returns the same data as model.json
dirctl pull $DIGEST

# Lookup basic metadata about the agent
dirctl info $DIGEST

#> {
#>   "digest": "sha256:<hash>",
#>   "type": "OBJECT_TYPE_AGENT",
#>   "size": 143
#> }
```

### Announce

This example demonstrates how to publish agent data models to allow content discovery across the network.
To avoid stale data, it is recommended to republish the data periodically
as the data across the network has TTL.

Note that this operation only works for the objects already pushed to the local storage layer, ie. it is required to first push the data before publication.

```bash
# Publish the data to your local data store
dirctl publish $DIGEST

# Publish the data across the network
dirctl publish $DIGEST --network
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

Discovery is performed using full-set label matching, ie. the results always fully match the requested query.
Note that it is not guaranteed that the returned data is available, valid, or up to date.

```bash
# Get a list of peers holding a specific agent data model
dirctl list --digest $DIGEST

#> Peer 12D3KooWQffoFP8ePUxTeZ8AcfReTMo4oRPqTiN1caDeG5YW3gng
#>   Digest: sha256:<hash>
#>   Labels: /skills/Text Generation, /skills/Fact Extraction

# Discover the agent data models in your local data store
dirctl list "/skills/Text Generation"
dirctl list "/skills/Text Generation" "/skills/Fact Extraction"

#> Peer HOST
#>   Digest: sha256:<hash>
#>   Labels: /skills/Text Generation, /skills/Fact Extraction

# Discover the agent data models across the network
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

### gRPC Error Codes

The following table lists the gRPC error codes returned by the server APIs, along with a description of when each code is used:

| Error Code                | Description                                                                                                                 |
|---------------------------|-----------------------------------------------------------------------------------------------------------------------------|
| `codes.InvalidArgument`   | Returned when the client provides an invalid or malformed argument, such as a missing or invalid object reference or agent. |
| `codes.NotFound`          | Returned when the requested object does not exist in the local store or across the network.                                 |
| `codes.FailedPrecondition`| Returned when the server environment or configuration is not in the required state (e.g., failed to create a directory or temp file, unsupported provider in config). |
| `codes.Internal`          | Returned for unexpected internal errors, such as I/O failures, serialization errors, or other server-side issues.           |
| `codes.Canceled`          | Returned when the operation is canceled by the client or context expires.                                                   |

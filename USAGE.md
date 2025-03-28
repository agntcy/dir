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

To start, generate an example agent that matches the data model schema defined in [Agent Data Model](api/core/v1alpha1/agent.proto) specification.

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
EOF

# Build the agent
dirctl build . > model.json

# Preview built agent
cat model.json
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
#>   "digest": "sha256:a8abbdd7403aed85abaa00e176effc212cd6b080cb161e0fac51399fd0e69c3f",
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
#>   Digest: sha256:a8abbdd7403aed85abaa00e176effc212cd6b080cb161e0fac51399fd0e69c3f
#>   Labels: /skills/Text Generation, /skills/Fact Extraction

# Discover the agent data models in your local data store
dirctl list "/skills/Text Generation"
dirctl list "/skills/Text Generation" "/skills/Fact Extraction"

#> Peer HOST
#>   Digest: sha256:a8abbdd7403aed85abaa00e176effc212cd6b080cb161e0fac51399fd0e69c3f
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

## Manifests inside OASF record (modules)

Each manifest module MUST have the following fields:

```json
{
    "module": [
        {
            "@schema": "jsonschema:https://mcp.com/v1.0.0/server.json",
            // alternatively, using DIR storage or IPFS
            "@schema": "jsonschema:/ipfs/bafybeigdyrzt5tqz5e3j6x5x5z5x5z5x5z5x5z5x",
            "@schema": "jsonschema:/dir/bafybeigdyrzt5tqz5e3j6x5x5z5x5z5x5z5x5z5x",
            "name": "integration/mcp",
            "data": {
                // full MCP data
            }
        }
    ]
}
```

Schema is a typed URL of where the schema can be found.

## SDK integration interface

```go
type ManifestModule interface {
    // Base module details
    GetSchema() string
    GetName() string

    // Serialization and validation module and OASF record
    ToOASFRecord() (Record, error) 
    FromOASFRecord() error
    Validate() error
}
```

## Packaging of modules

Each OASF record can have one or multiple manifest modules.
Linkage between different artifacts and OASF record can be expressed using links inside the OASF record.

```json
{
    // OASF record
    "name": "record/v1.0.0/my-oasf-record",
    "version": "1.0.0",

    // inner envelop - internal
    "modules": [
        {
            "@schema": "jsonschema:https://mcp.com/v1.0.0/server.json",
            "name": "manifest/mcp",
            "data": {
                // full MCP data
            },
            "links": [] // ??? this links a module to other objects, but its inner envelop
        }
    ],

    // does not have to be embedded into OASF record
    // outer envelop - external
    "links": [
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "type": "mcp.server.json.data.schema",
            "name": "manifest/mcp",
            "size": 5678
        },
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "type": "agntcy.oasf.Record",
            "name": "manifest/mcp",
            "size": 5678
        },
    ]
}
```

```
OASF Record <- (referrer API) MCP server (one or many) <- (tools/prompts/etc)
            <- (referrer API) A2A card (one or many)
            <- (referrer API) OASF card (one or many)


(current, internal envelop) vs (external envelop, linkage)
            OASF{MCP}       vs OASF <- MCP server


OASF Record <- link (MCP server)
            <- link (signature)
            <- link of {cid of MCP server} <- link (signature by producer of original OASF record)

```


## Linkage options

- Linkage via modules (one record can have multiple modules, e.g. MCP server, A2A card, etc)
- Linkage via links (one record can have multiple links to other objects)
- Linkage via external references ()












### Linkage

The parent object is the external protocol object which referrers (references) the OASF record.


**Translated OASF record links to an MCP server**
MCP Server = cid1 <- OASF Record 1 = cid11

**Translated OASF record links to an A2A card**
A2A Card = cid2  <- OASF Record 2 = cid22

**An OASF record can link to multiple other OASF records**
OASF Record (OCI) cid33 <- OASF Record 1 = cid11
                        <- OASF Record 2 = cid22

This can resolve to a graph of a single OASF record linking to other artifacts, e.g.

OASF Record (resolved)  <- MCP Server <- OASF Record 1
                        <- A2A Card   <- OASF Record 2

**All operations are available via OCI**

search("tourist guide") = {cid11, cid22, cid33}
                           |
                           v
                 OASF Record (resolved)  <- MCP Server <- OASF Record 1 = cid11
                                         <- A2A Card   <- OASF Record 2 = cid22

get_links(cid33) = {cid11, cid22}
get_links(cid11) = {cid1}
get_links(cid22) = {cid2}

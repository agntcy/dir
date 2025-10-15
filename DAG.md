# Overview

This document defines how Merkle DAGs are represented in the system.
It links to OASF for the formal specification of the DAG format,
and describes how DAGs are stored and retrieved.

# Specification

The DAG format is specified in the [OASF specification](https://oasis-open.github.io/cti-documentation/). It defines the structure and semantics of DAGs in the system.

**Object**

```proto
message Object {
    // Unique identifier for the DAG.
    // This field is populated when the object is stored.
    // It can be calculated from the content of the object.
    optional string cid = 1;

    string type = 2; // Type of the object
    string name = 3; // Name of the object
    uint64 size = 4; // Size of the object
    optional google.protobuf.Struct data = 5; // Data of the object
    repeated Object links = 6; // Links to other objects
}
```

**File Object**

```json
{
    "cid": "bafybeigdyrzt5tqz5e3j6x5x5z5x5z5x5z5x5z5x",
    "type": "file", // only A-Z, a-z, 0-9, and _./ are allowed
    "name": "example.txt", // only A-Z, a-z, 0-9, and _./ are allowed
    "size": 1234,
    "data": {
        "content": "Hello, World!"
    },
    "links": [
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "name": "example.txt",
            "size": 1234
        }
    ]
}
```

**Record Object**

```json
{
    "cid": "bafybeigdyrzt5tqz5e3j6x5x5z5x5z5x5z5x5z5x",
    "type": "agntcy.oasf.types.v1.Record",
    "name": "record/v1.0.0/my-example-record",
    "size": 1234,
    "links": [
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "type": "agntcy.dir.types.v1.Signature",
            "name": "apps/runtime/signature",
            "size": 5678
        },
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "type": "agntcy.oasf.modules.runtime.integrations",
            "name": "modules/runtime/integrations",
            "size": 5678
        },
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "type": "agntcy.oasf.types.v1.Skill",
            "name": "skills/natural_language_processing",
            "size": 5678
        }
    ]
}
```

# Overview

This document defines how Merkle DAGs are represented in the system.
It links to OASF for the formal specification of the DAG format,
and describes how DAGs are stored and retrieved.

# Specification

The specification defines the structure and semantics of the data in the system.
It is based on the [Open Agentic Schema Framework](https://schema.oasf.outshift.com/) specification.

## Structure

Each object can be expressed in the following format as per [JSON-LD 1.1](https://www.w3.org/TR/json-ld11/) specification.

OASF defines a base object structure that can be extended for different types of objects.
Namely, the `Object` structure is defined as follows:

```yaml
# hashes to Qm-oasf-base-object
---
    @artifactType: agntcy.oasf.types.Object
    @mediaType: application/ld+json; profile="https://schema.oasf.outshift.com/"
    annotations:
    data:

# hashes to Qm-mediachain-song1
---
  @type: MusicRecording
  name: Christmas Will Break Your Heart
  byArtist: 
    @type: MusicGroup
    name: LCD Soundsystem
  inAlbum: 
    link: /ipfs/Qm-mediachain-album1
  author: 
    @type: Person
    link: /ipfs/Qm-mediachain-artist1
  image: 
    @type: Photograph
    contentUrl: 
      link: /ipfs/Qm-literal-photo-bytes

# hashes to Qm-mediachain-album1
---
  @type: Album
  name: LCD Soundsystem's Triumphant Return

# hashes to Qm-mediachain-artist1
---
  @type: Person
  name: James Murphy
  bitcoinAddress: 19d3ynnuNKbgFLxp6XsyoxxGSvoAzBYPMw
```

**Object**

```json
{
    "cid": "bafybeigdyrzt5tqz5e3j6x5x5z5x5z5x5z5x5z5x",
    "type": "object_type", // only A-Z, a-z, 0-9, and _./ are allowed
    "name": "object_name", // only A-Z, a-z, 0-9, and _./ are allowed
    "size": 1234,
    "links": [
        {
            "cid": "bafybeihdwdcefgh4dqkjv67uz",
            "type": "linked_object_type", // only A-Z, a-z, 0-9, and _./ are allowed
            "name": "linked_object_name", // only A-Z, a-z, 0-9, and _./ are allowed
            "size": 5678
        }
    ]
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

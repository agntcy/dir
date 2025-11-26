# Overview

This document defines how objects are stored in the system.
It describes the structure of the objects, their fields, and how they relate to each other.
In addition, it also describes the way objects are linked together to form a graph of related objects,
aka DAG (Directed Acyclic Graph).

# Specification

The specification defines the structure and semantics of the data in the system.

## Structure

**Object**

Each object in the system is represented as a JSON object with the following fields:

```json
{
    "schema": {
      // optional, object schema
    },
    "annotations": {
      // optional, object annotations
    },
    "created_at": "2024-01-01T00:00:00Z", // optional, creation timestamp in RFC3339 
    "size": 1234, // optional, size of the data in bytes
    "data": {
      "cid": "cid-data", // cid of the data object
    },
    "links": [
      // optional, links to other objects
      { /* object link */ },
      { /* object link */ }
    ]
}
```

**Schema field**

Schema field defines the schema of an arbitrary object in the system.

```json
{
    "schema": {
        "type": "org.agntcy.dir.v1.Record", // optional, type of the object
        "version": "0.8.0", // optional, version of the object schema
        "format": "json", // optional, format of the data, usually json
    },
}
```

The schema field is optional, but it is recommended to include it for better interoperability and validation of the data by other systems and consumers.

**Annotations field**

Annotations field contains metadata about the object.
It represents a string-string map of key-value pairs.

```json
{
    "annotations": {
        "key1": "value1",
        "key2": "value2"
    }
}
```

**Data field**

Data field contains the link to the actual data of the object.
The data itself can be of any type, and its structure is defined by the schema field.
The data is stored as an independent object in the system, and is linked via its CID.

```json
{
    "data": {
        "cid": "bafybei67890"
    }
}
```

**Links field**

Links field contains references to other objects in the system.
It is represented as an array of objects.

```json
{
    "links": [
        {
            "cid": "bafybei67890",
            "schema": {
                "type": "agntcy.oasf.types.v1.Record", // optional, type of the linked object
                "version": "0.8.0" // optional, version of the linked object schema
            },
            "annotations": {
                // optional, link annotations
            },
            "size": 5678 // size of the linked object in bytes
        }
    ]
}
```

The linked object MUST contain at least the `cid` field.
The linked object SHOULD NOT contain `links` field.

## Storing objects

The objects can be stored in a CAS (Content Addressable Storage) system that supports at least:
- Storing data
- Looking up data size
- Content tagging / named references

## Calculating CIDs

The CID of an `Object` is calculated by serializing the object to JSON canonical form (with sorted keys), and then computing the CID over the serialized bytes.
The CID of raw data is calculated without serialization.

The CID is calculated using the following parameters:
- CID version: 1
- Multicodec: 1
- Multihash: sha2-256

## Mapping to storage backends 

The goal of creating a custom object format is to enable easy mapping to different storage backends.
For example, when using OCI as a storage backend, the objects and their data can be mapped to OCI artifacts.
When using IPFS as a storage backend, the objects and their data can be stored as separate IPFS objects.
When using filesystem as a storage backend, the objects and their data can be stored as separate files.
Whatever the storage backend is, the objects and their data always resolve to the same CIDs.
Otherwise, if the storage backend changes, the CIDs would change as well, breaking the whole system.

**Object**

```json
{
    "schema": {
        "type": "agntcy.oasf.types.v1.Record",
        "version": "0.8.0",
        "format": "json"
    },
    "annotations": {
        "key1": "value1",
        "key2": "value2"
    },
    "created_at": "2024-01-01T00:00:00Z",
    "size": 1234,
    "data": {
        "cid": "bafybei67890" // hash of the data
    },
    "links": [
        {
            "cid": "bafybei67890",
            "schema": {
                "type": "agntcy.oasf.types.v1.Monitoring",
                "version": "1.0.0"
            },
            "annotations": {
                "link_key": "link_value"
            },
            "created_at": "2024-01-01T00:00:00Z",
            "size": 1234,
            "data": {
                "cid": "bafybei13579"
            }
        }
    ]
}
```

**OCI Storage**

When using OCI as a storage backend, the objects and their data are mapped to OCI artifacts.
Raw data is stored as OCI blobs, while the objects are stored as OCI manifests.

When storing objects in OCI, both raw data and objects are tagged with their respective CIDs for easy retrieval, in the following way:
- Raw data blobs are tagged with their data CID.
- OCI manifests are tagged with the `Object` CID.
This ensures that both data and objects can be retrieved using their respective CIDs.

```json
{
    "schemaVersion": 2,
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "artifactType": "org.agntcy.dir.Object",
    "config": {
        "mediaType": "application/octet-stream",
        "size": 1234,
        "digest": "sha256:abcdef1234567890", // object data cid
        "artifactType": "agntcy.oasf.types.v1.Record",
        "annotations": {
            "org.agntcy.oasf.schema.type": "agntcy.oasf.types.v1.Record",
            "org.agntcy.oasf.schema.version": "1.0.0",
            "org.agntcy.oasf.schema.created_at": "2024-01-01T00:00:00Z",
            "key1": "value1",
            "key2": "value2"
        }
    },
    "layers": [
        {
            "mediaType": "application/octet-stream",
            "size": 1234,
            "digest": "sha256:abcdef1234567890", // link object data cid
            "artifactType": "agntcy.oasf.types.v1.Monitoring",
            "annotations": {
                "org.agntcy.oasf.schema.type": "agntcy.oasf.types.v1.Monitoring",
                "org.agntcy.oasf.schema.version": "1.0.0",
                "org.agntcy.oasf.schema.created_at": "2024-01-01T00:00:00Z",
                "link_key": "link_value"
            }
        }
    ]
}
```

**IPFS storage**

When using IPFS as a storage backend, the objects and their data are stored as separate IPFS objects.
Raw data is stored as raw IPFS objects, while the objects are stored as JSON objects.
This ensures that both data and objects can be retrieved using their respective CIDs.

**Filesystem storage**

When using filesystem as a storage backend, the objects and their data are stored as separate files.
Raw data is stored as files named by their data CID, while the objects are stored as JSON files named by their object CID.
This ensures that both data and objects can be retrieved using their respective CIDs.

## API operations

The system provides the following API operations for working with objects:

```
- PushData([]byte): cid.Cid
- Pull(cid.Cid): []byte
- Push(Object): cid.Cid
- Lookup(cid.Cid): Object
- Delete(cid.Cid): void
```

## OASF Integration

The object format is designed to be compatible with the OASF (Open Agentic Schema Framework) specifications.
This ensures that objects can be easily integrated with other systems and services that also adhere to the OASF standards.
For more information about OASF, please refer to the [OASF documentation](https://schema.oasf.outshift.com/).

### Record Schema

Given a Record object defined in OASF:

```json
{
  "name": "canada guitar betting",
  "version": "v2.3",
  "description": "achievement metal covering",
  "modules": [
    {
      "data": {
        "config": {
          "type": "helm_chart",
          "url": "https://www.clips.coop"
        },
        "runtime_framework": "autogen",
        "runtime_deps": [
          "captured"
        ]
      },
      "id": 204,
      "name": "integration/agentspec"
    },
    {
      "data": {
        "protocol_version": "carrying alike options",
        "card_data": "registered"
      },
      "id": 203,
      "name": "integration/a2a"
    }
  ],
  "domains": [
    {
      "id": 1205,
      "name": "energy/renewable_energy"
    },
    {
      "id": 605,
      "name": "education/pedagogy"
    },
    {
      "id": 201,
      "name": "finance_and_business/banking"
    }
  ],
  "skills": [
    {
      "id": 1002,
      "name": "agent_orchestration/role_assignment"
    },
    {
      "id": 1202,
      "name": "devops_mlops/deployment_orchestration"
    },
    {
      "id": 10103,
      "name": "natural_language_processing/natural_language_understanding/entity_recognition"
    },
    {
      "id": 10303,
      "name": "natural_language_processing/information_retrieval_synthesis/knowledge_synthesis"
    },
    {
      "id": 70101,
      "name": "multi_modal/image_processing/image_to_text"
    },
    {
      "id": 70105,
      "name": "multi_modal/image_processing/visual_qa"
    },
    {
      "id": 108,
      "name": "natural_language_processing/ethical_interaction"
    },
    {
      "id": 702,
      "name": "multi_modal/audio_processing"
    }
  ],
  "schema_version": "0.8.0",
  "authors": [
    "Ira Brandi"
  ],
  "annotations": {
    "evening identity list": "as div warriors",
    "friendship medline licence": "himself biz crime"
  },
  "created_at": "2025-11-26T08:21:51.216617Z",
  "locators": [
    {
      "type": "helm_chart",
      "url": "https://www.prominent.mobi",
      "annotations": {
        "booking daily magazines": "canberra vb encounter",
        "naval reservation alien": "celebs attribute symptoms"
      }
    }
  ]
}
```

It would be represented as a DIR Object as follows:

```json
{
  "schema": {
    "type": "org.agntcy.oasf.types.v1.Record",
    "version": "0.8.0",
    "format": "json"
  },
  "annotations": {
    "evening identity list": "as div warriors",
    "friendship medline licence": "himself biz crime"
  },
  "created_at": "2025-11-26T08:21:51.216617Z",
  "size": 4567,
  "data": {
    "cid": "bafybeihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku" // CID of the raw Record data
  },
    "links": [
        {
        "cid": "bafybeibwzif3c5tq37z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7", // CID of the first module data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Module",
            "version": "0.8.0"
        },
        "size": 1234,
        "data": {
            "cid": "bafybeibwzif3c5tq37z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7"
        }
        },
        {
        "cid": "bafybeic7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7", // CID of the second module data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Module",
            "version": "0.8.0"
        },
        "size": 1134,
        "data": {
            "cid": "bafybeic7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7z5x7x4y5g6z7"
        }
        },
        {
        "cid": "bafybeidomainscid1234567890abcdefg", // CID of the first domain data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Domain",
            "version": "0.8.0"
        },
        "size": 567,
        "data": {
            "cid": "bafybeidomainscid1234567890abcdefg"
        }
        },
        {
        "cid": "bafybeidomainscid2234567890abcdefg", // CID of the second domain data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Domain",
            "version": "0.8.0"
        },
        "size": 567,
        "data": {
            "cid": "bafybeidomainscid2234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid100234567890abcdefg", // CID of the first skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid100234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid120234567890abcdefg", // CID of the second skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid120234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid10103234567890abcdefg", // CID of the third skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid10103234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid10303234567890abcdefg", // CID of the fourth skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid10303234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid70101234567890abcdefg", // CID of the fifth skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid70101234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid70105234567890abcdefg", // CID of the sixth skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid70105234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid108234567890abcdefg", // CID of the seventh skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid108234567890abcdefg"
        }
        },
        {
        "cid": "bafybeiskillscid702234567890abcdefg", // CID of the eighth skill data
        "schema": {
            "type": "org.agntcy.oasf.types.v1.Skill",
            "version": "0.8.0"
        },
        "size": 890,
        "data": {
            "cid": "bafybeiskillscid702234567890abcdefg"
        }
        }
    ]
}
```
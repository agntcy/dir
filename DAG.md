# Overview

This document defines how OASF objects are represented in the system.
It describes the structure of the objects, their fields, and how they relate to each other.
In addition, it also describes the way objects are linked together to form a graph of related objects,
aka DAG (Directed Acyclic Graph).

# Specification

The specification defines the structure and semantics of the data in the system.
It is based on the [Open Agentic Schema Framework](https://schema.oasf.outshift.com/) specification.

## Structure

**Object**

Each object in the system is represented as a JSON object with the following fields:

```json
{
    "cid": "bafybei12345", // Content Identifier (CID) of the object
    "schema": {
      // optional, object schema
    },
    "annotations": {
      // optional, object annotations
    },
    "created_at": "2024-01-01T00:00:00Z", // creation timestamp in RFC3339 
    "size": 1234, // size of the object in bytesformat
    "data": {
      // optional, reference to the actual data of the object
    },
    "links": [
      // optional, links to other objects
    ]
}
```

**Schema field**

Schema field defines the schema of an arbitrary object in the system.

```json
{
    "schema": {
        "type": "agntcy.oasf.types.v1.Record", // optional, type of the object
        "version": "0.8.0", // optional, version of the object schema
        "format": "json", // optional, format of the data
    },
}
```

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

Data field contains the actual content of the object.
It can be of any type, and its structure is defined by the schema field.
The data field is actually stored as a separate object in the system, and the data field here contains only a reference to that object.

```json
{
    "data": {
        "cid": "bafybei67890",
        "size": 1234,
        "schema": {
          "type": "octet/raw"
        },
    }
}
```

**Links field**

Links field contains references to other objects in the system.
It is represented as an array of link objects, each containing the following fields:

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

## Mapping between OASF and underlying storage

The OASF objects are stored in an underlying storage system that supports content addressing and linking between objects.

The mapping between OASF object and OCI storage is as follows:

**Object**
```json
{
    "cid": "bafybei12345",
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
        "cid": "bafybei67890"
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
            },
        }
    ]
}
```

**OCI Manifest**

```json
{
    "schemaVersion": 2,
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "config": {
        "mediaType": "application/octet-stream",
        "size": 1234,
        "digest": "sha256:abcdef1234567890", // object data cid
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
# directory API python SDK

## Overview

The directory API python SDK provides a simple and efficient way to interact with the directory API.
It allows developers to integrate directory functionality into their Python applications with ease.

## Features

- **Store API**: The SDK includes a store API that allows developers to push the record data model to the store and
retrieve it from the store.
- **Routing API**: The SDK provides a routing API that allows developers to publish and retrieve record to and from the
network.


## Installation

Install the SDK using [uv](https://github.com/astral-sh/uv)

init the project:
```bash
uv init
```

add the SDK to your project:
```bash
uv add agntcy-dir-sdk --index https://buf.build/gen/Python
```

## Usage

### Starting the Directory Server

To start the Directory server, you can deploy your instance or use Taskfile as below.

```bash
task server:start
```

### Initializing the Client

```python
# Initialize the client
client = Client(Config())
```
### Creating and Pushing an Agent Object

```python
# Create a record object
example_record = record_pb2.Record(
    name="example-record",
    version="v3",
    skills=[
        skill_pb2.Skill(
            name="Natural Language Processing",
            id=1,
        ),
    ],
    extensions=[
        extension_pb2.Extension(
            name="schema.oasf.agntcy.org/domains/domain-1",
            version="v1",
        )
    ],
    signature=signature_pb2.Signature(),
)

r = core_record_pb2.Record(v3=example_record)

# Push the object to the store
record_reference = client.push(record=r)
```

### Pulling the Object

```python
# Pull the object from the store
record = client.pull(record_reference)
```

### Looking Up the Object

```python
# Lookup the object
metadata = client.lookup(record_reference)
```

### Publishing the Object

```python
# Publish the object
client.publish(record_reference)
```

### Listing Objects in the Store

```python
# List objects in the store
query = record_query_type.RecordQuery(
    type=record_query_type.RECORD_QUERY_TYPE_SKILL,
    value="/skills/Natural Language Processing/Text Completion",
)

list_request = routingv1.ListRequest(queries=[query])
objects = list(client.list(list_request))
```

### Unpublishing the Object

```python
# Unpublish the object
client.unpublish(record_reference)
```

### Deleting the Object

```python
# Delete the object
client.delete(record_reference)
```
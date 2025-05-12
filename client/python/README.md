# directory API python SDK

## Overview

The directory API python SDK provides a simple and efficient way to interact with the directory API.
It allows developers to integrate directory functionality into their Python applications with ease.

## Features

- **Model Bindings**: The SDK includes pre-generated Python proto stubs for all API endpoints, making it
easy to call the API without needing to manually define the request and response structures.
- **Store API**: The SDK includes a store API that allows developers to push the agent data model to the store and
retrieve it from the store.
- **Routing API**: The SDK provides a routing API that allows developers to publish and retrieve agents to and from the
network.

## Installation

Install the SDK using [uv](https://github.com/astral-sh/uv)

init the project:
```bash
uv init
```

add the SDK to your project:
```bash
uv add git+https://github.com/agntcy/dir.git@v0.2.2#subdirectory=client/python
```

## Usage

### Starting the Directory Server

To start the Directory server, you can deploy your instance or use Taskfile as below.

```bash
task server:start
```

### Initializing the Client

```python
import io
import hashlib
import json
from google.protobuf.json_format import MessageToDict
from client.client import Client, Config
from core.v1alpha1 import object_pb2, agent_pb2, skill_pb2, extension_pb2
from routing.v1alpha1 import routing_service_pb2 as routingtypes

# Initialize the client
client = Client(Config())
```
### Creating and Pushing an Agent Object

```python
# Create an agent object
agent = agent_pb2.Agent(
    name="example-agent",
    version="v1",
    skills=[
        skill_pb2.Skill(
            category_name="Natural Language Processing",
            category_uid="1",
            class_name="Text Completion",
            class_uid="10201",
        ),
    ],
    extensions=[
        extension_pb2.Extension(
            name="schema.oasf.agntcy.org/domains/domain-1",
            version="v1",
        )
    ]
)

agent_dict = MessageToDict(agent, preserving_proto_field_name=True)

# Convert the agent object to a JSON string
agent_json = json.dumps(agent_dict).encode('utf-8')
print(agent_json)

# Create a reference for the object
ref = object_pb2.ObjectRef(
    digest="sha256:" + hashlib.sha256(agent_json).hexdigest(),
    type=object_pb2.ObjectType.Name(object_pb2.ObjectType.OBJECT_TYPE_AGENT),
    size=len(agent_json),
    annotations=agent.annotations,
)

# Push the object to the store
data_stream = io.BytesIO(agent_json)
response = client.push(ref, data_stream)
print("Pushed object:", response)
```

### Pulling the Object

```python
# Pull the object from the store
data_stream = client.pull(ref)

# Deserialize the data
pulled_agent_json = data_stream.getvalue().decode('utf-8')
print("Pulled object data:", pulled_agent_json)
```

### Looking Up the Object

```python
# Lookup the object
metadata = client.lookup(ref)
print("Object metadata:", metadata)
```

### Publishing the Object

```python
# Publish the object
client.publish(ref, network=False)
print("Object published.")
```

### Listing Objects in the Store

```python
# List objects in the store
list_request = routingtypes.ListRequest(
    labels=["/skills/Natural Language Processing/Text Completion"]
)
objects = list(client.list(list_request))
print("Listed objects:", objects)
```

### Unpublishing the Object

```python
# Unpublish the object
client.unpublish(ref, network=False)
print("Object unpublished.")
```

### Deleting the Object

```python
# Delete the object
client.delete(ref)
print("Object deleted.")
```
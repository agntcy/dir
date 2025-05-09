# Python client SDK for AGNTCY

## Overview

The Python client SDK for **AGNTCY** provides a simple and efficient way to interact with the **AGNTCY** API.
It allows developers to integrate **AGNTCY** functionality into their Python applications with ease.

## Features

- **Available Python proto stubs**: The SDK includes pre-generated Python proto stubs for all API endpoints, making it
easy to call the API without needing to manually define the request and response structures.
- **Store API**: The SDK includes a store API that allows developers to push the agent data model to the store and
retrieve it from the store.
- **Routing API**: The SDK provides a routing API that allows developers to publish agent pushed upfront to the
network.

## Installation

Install the SDK using [uv](https://github.com/astral-sh/uv)

init the project:
```bash
$ uv init
# Initialized project `python-test`
```

add the SDK to your project:
```bash
$ uv add git+https://github.com/agntcy/dir.git@main#subdirectory=client/python
# Using CPython 3.12.7
# Creating virtual environment at: .venv
#     Updated https://github.com/agntcy/dir.git (ec4c0b36e7d483ac19341795398c280e7298e383)
# Resolved 12 packages in 887ms
#       Built agntcy-dir-client-sdk @ git+https://github.com/agntcy/dir.git@ec4c0b36e7d483ac19341795398c280e7298e383#subdirectory=client/python
#       Built agntcy-dir-proto-stubs @ git+https://github.com/agntcy/dir.git@ec4c0b36e7d483ac19341795398c280e7298e383#subdirectory=api/python
# Prepared 4 packages in 1.31s
# Installed 10 packages in 18ms
#  + agntcy-dir-client-sdk==0.1.0 (from git+https://github.com/agntcy/dir.git@ec4c0b36e7d483ac19341795398c280e7298e383#subdirectory=client/python)
#  + agntcy-dir-proto-stubs==0.1.0 (from git+https://github.com/agntcy/dir.git@ec4c0b36e7d483ac19341795398c280e7298e383#subdirectory=api/python)
#  + grpcio==1.71.0
#  + grpcio-tools==1.71.0
#  + iniconfig==2.1.0
#  + packaging==25.0
#  + pluggy==1.5.0
#  + protobuf==5.29.4
#  + pytest==8.3.5
#  + setuptools==80.3.1
```

## Usage

### Starting the AGNTCY Server

For these commands to work correctly, you need to have the **AGNTCY** server running and accessible.

```bash
$ task server start
# task: [server:store:start] # mount config
# cat > /tmp/config.json <<EOF
# {
#   "distSpecVersion": "1.1.1",
#   "storage": {
#     "rootDirectory": "/tmp/zot"
#   },
#   "http": {
#     "address": "127.0.0.1",
#     "port": "5000"
#   },
#   "log": {
#     "level": "debug"
#   },
#   "extensions": {
#     "search": {
#       "enable": true,
#       "cve": {
#         "updateInterval": "24h"
#       }
#     }
#   }
# }
# EOF
# 
# # run docker with attached volume
# docker run \
#       -it \
#       --rm -d -p 5000:5000 \
#       -v /tmp/config.json:/config.json:ro \
#       --name oci-registry ghcr.io/project-zot/zot-linux-arm64:v2.1.1
# 
# 1f772921467d2eaa6070ddb857500f29ace22aeea75cd75d06c67a157a0a00e1
# task: [server:start] go run main.go
# time=2025-05-08T12:35:48.942+02:00 level=INFO msg="Config file not found, use defaults." component=config
# time=2025-05-08T12:35:48.952+02:00 level=INFO msg="Ignoring announcement event for invalid object" component=routing/handler digest="sha256:̪뉍\r\xd2\xd4X\xaa\x14\xe7\xcc=\xd4y\xbb>\xd0\xc9\x04\xf8\x99]]\xaf\xc7\xfa\xdev0J"
# INFO[0000] Starting healthz server. listenAddr=0.0.0.0:8889
# time=2025-05-08T12:35:48.952+02:00 level=INFO msg="Server starting" component=server address=0.0.0.0:8888
# time=2025-05-08T12:37:48.951+02:00 level=INFO msg="Ignoring announcement event for invalid object" component=routing/handler digest="sha256:̪뉍\r\xd2\xd4X\xaa\x14\xe7\xcc=\xd4y\xbb>\xd0\xc9\x04\xf8\x99]]\xaf\xc7\xfa\xdev0J"
```

### Example Usage

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

# Pull the object from the store
data_stream = client.pull(ref)

# Deserialize the data
pulled_agent_json = data_stream.getvalue().decode('utf-8')
print("Pulled object data:", pulled_agent_json)

# Lookup the object
metadata = client.lookup(ref)
print("Object metadata:", metadata)

# Publish the object
client.publish(ref, network=False)
print("Object published.")

# List objects in the store
list_request = routingtypes.ListRequest(
    labels=["/skills/Natural Language Processing/Text Completion"]
)
objects = list(client.list(list_request))
print("Listed objects:", objects)

# Unpublish the object
client.unpublish(ref, network=False)
print("Object unpublished.")

# Delete the object
client.delete(ref)
print("Object deleted.")
```
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
uv add agntcy-dir-sdk --index https://buf.build/gen/python
uv add 'agntcy-dir-grpc-python' --index https://buf.build/gen/python
```

## Usage

### Starting the Directory Server

To start the Directory server, you can deploy your instance or use Taskfile as below.

```bash
task server:start
```

### Usage of the SDK

See `v1/example.py` to help get started.
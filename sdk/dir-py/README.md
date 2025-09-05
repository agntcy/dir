# Directory Python SDK

## Overview

Dir Python SDK provides a simple way to interact with the Directory API.
It allows developers to integrate and use Directory functionality from their Python applications with ease.

## Features

- **Store API**: The SDK provides a way to interact with the Store API which allows developers to push the record data model to the store and retrieve it from the store. It also provides a way to interact with the Store Sync API to create synchronization policies between Directory servers.
- **Search API**: The SDK provides provides a way to interact with the Search API which allows developers to search for stored records in Directory server.
- **Routing API**: The SDK provides a way to interact with Routing API which allows developers to publish, retrieve, and query record to, from, and across the network.
- **Signing and Verification**: The SDK provides a way to sign and verify your records. Signing is performed locally and requires [dirctl]() binary,
while verification is preformed remotely using Directory gRPC API that leverages OCI and Cosign verification.

## Installation

Install the SDK using [uv](https://github.com/astral-sh/uv)

1. Initialize the project:
```bash
uv init
```

2. Add the SDK to your project:
```bash
uv add agntcy-dir --index https://buf.build/gen/python
```

## Getting Started

1. Start the Directory Server

To start the Directory server, you can deploy your instance or use Taskfile as below.

```bash
task server:start
```

Alternativelly, if you already have a running Dir server, specify its location using env-variable `DIRECTORY_CLIENT_SERVER_ADDRESS`. 
By default, SDK uses `DIRECTORY_CLIENT_SERVER_ADDRESS=0.0.0.0:8888`

2. SDK Usage

See local [Example Python Project](./example) to help get started with Directory Python SDK.

Run the example using

```
uv run example.py
```

# directory API javascript SDK

## Overview

The directory API javascript SDK provides a simple and efficient way to interact with the directory API.
It allows developers to integrate directory functionality into their Javascript applications with ease.

## Features

- **Store API**: The SDK includes a store API that allows developers to push the record data model to the store and
retrieve it from the store.
- **Routing API**: The SDK provides a routing API that allows developers to publish and retrieve record to and from the
network.
- **Search API**: The SDK constains a search API which enables developers to search records inside their directory.

## Installation

Install the SDK using [npm](https://nodejs.org/en/download)

init the project:
```bash
npm init
```

add the SDK to your project:
```bash
npm config set @buf:registry https://buf.build/gen/npm/v1/
npm install agntcy-dir-sdk
npm install @buf/agntcy_dir.grpc_node
npm install @buf/agntcy_dir.grpc_web
```

## Usage

### Starting the Directory Server

To start the Directory server, you can deploy your instance or use Taskfile as below.

```bash
task server:start
```

### Usage of the SDK

See `v1/example.js` to help get started.
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
npm i agntcy-dir-sdk
```

## Usage

### Starting the Directory Server

To start the Directory server, you can deploy your instance or use Taskfile as below.

```bash
task server:start
```

### Initializing the Client

```javascript
// Initialize the client
const client = new Client(new Config());
```
### Creating and Pushing an Agent Object

```javascript
// Create a record object
const exampleRecord = new record_pb2.Record();
exampleRecord.setName('example-record');
exampleRecord.setVersion('v3');

const skill = new skill_pb2.Skill();
skill.setName('Natural Language Processing');
skill.setId(1);
exampleRecord.addSkills(skill);

const extension = new extension_pb2.Extension();
extension.setName('schema.oasf.agntcy.org/domains/domain-1');
extension.setVersion('v1');
exampleRecord.addExtensions(extension);

const signature = new signature_pb2.Signature();
exampleRecord.setSignature(signature);

record = new core_record_pb2.Record();
record.setV3(exampleRecord);

// Push the object to the store
record_reference = client.push(record)
```

### Pulling the Object

```javascript
// Pull the object from the store
record = client.pull(record_reference)
```

### Search objects

```javascript
// Set search queries
let search_query = new search_query_type.RecordQuery();
search_query.setType(search_query_type.RecordQueryType.RECORD_QUERY_TYPE_SKILL);
search_query.setValue('/skills/Natural Language Processing/Text Completion');

const queries = [search_query];

let search_request = new search_types.SearchRequest();
search_request.setQueriesList(queries);
search_request.setLimit(1);

// Search objects
search_response = client.search(search_request);
```

### Looking Up the Object

```javascript
// Lookup the object
metadata = client.lookup(record_reference)
```

### Publishing the Object

```javascript
let publish_request = new routing_types.PublishRequest();
publish_request.setRecordCid(ref.u[0]);

// Publish the object
client.publish(publish_request)
```

### Listing Objects in the Store

```javascript
// List objects in the store
const query = new record_query_type.RecordQuery();
query.setType(record_query_type.RECORD_QUERY_TYPE_SKILL);
query.setValue('/skills/Natural Language Processing/Text Completion');
const listRequest = new routing_types.ListRequest();
listRequest.addQueries(query);

list_response = client.list(listRequest)
```

### Unpublishing the Object

```javascript
let unpublish_request = new routing_types.UnpublishRequest();
unpublish_request.setRecordCid(ref.u[0]);

// Unpublish the object
client.unpublish(unpublish_request)
```

### Deleting the Object

```javascript
// Delete the object
client.delete(record_reference)
```
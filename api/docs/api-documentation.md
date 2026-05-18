# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [core/v1alpha2/extension.proto](#core_v1alpha2_extension-proto)
    - [Extension](#core-v1alpha2-Extension)
    - [Extension.AnnotationsEntry](#core-v1alpha2-Extension-AnnotationsEntry)
  
- [core/v1alpha2/locator.proto](#core_v1alpha2_locator-proto)
    - [Locator](#core-v1alpha2-Locator)
    - [Locator.AnnotationsEntry](#core-v1alpha2-Locator-AnnotationsEntry)
  
    - [LocatorType](#core-v1alpha2-LocatorType)
  
- [core/v1alpha2/record.proto](#core_v1alpha2_record-proto)
    - [Record](#core-v1alpha2-Record)
    - [Record.AnnotationsEntry](#core-v1alpha2-Record-AnnotationsEntry)
  
- [core/v1alpha2/skill.proto](#core_v1alpha2_skill-proto)
    - [Skill](#core-v1alpha2-Skill)
    - [Skill.AnnotationsEntry](#core-v1alpha2-Skill-AnnotationsEntry)
  
- [core/v1alpha2/signature.proto](#core_v1alpha2_signature-proto)
    - [Signature](#core-v1alpha2-Signature)
    - [Signature.AnnotationsEntry](#core-v1alpha2-Signature-AnnotationsEntry)
  
- [core/v1alpha1/extension.proto](#core_v1alpha1_extension-proto)
    - [Extension](#core-v1alpha1-Extension)
    - [Extension.AnnotationsEntry](#core-v1alpha1-Extension-AnnotationsEntry)
  
- [core/v1alpha1/locator.proto](#core_v1alpha1_locator-proto)
    - [Locator](#core-v1alpha1-Locator)
    - [Locator.AnnotationsEntry](#core-v1alpha1-Locator-AnnotationsEntry)
  
    - [LocatorType](#core-v1alpha1-LocatorType)
  
- [core/v1alpha1/object.proto](#core_v1alpha1_object-proto)
    - [Object](#core-v1alpha1-Object)
    - [ObjectRef](#core-v1alpha1-ObjectRef)
    - [ObjectRef.AnnotationsEntry](#core-v1alpha1-ObjectRef-AnnotationsEntry)
  
    - [ObjectType](#core-v1alpha1-ObjectType)
  
- [core/v1alpha1/skill.proto](#core_v1alpha1_skill-proto)
    - [Skill](#core-v1alpha1-Skill)
    - [Skill.AnnotationsEntry](#core-v1alpha1-Skill-AnnotationsEntry)
  
- [core/v1alpha1/signature.proto](#core_v1alpha1_signature-proto)
    - [Signature](#core-v1alpha1-Signature)
  
- [core/v1alpha1/agent.proto](#core_v1alpha1_agent-proto)
    - [Agent](#core-v1alpha1-Agent)
    - [Agent.AnnotationsEntry](#core-v1alpha1-Agent-AnnotationsEntry)
  
- [search/v1alpha2/record_query.proto](#search_v1alpha2_record_query-proto)
    - [RecordQuery](#search-v1alpha2-RecordQuery)
  
    - [RecordQueryType](#search-v1alpha2-RecordQueryType)
  
- [search/v1alpha2/search_service.proto](#search_v1alpha2_search_service-proto)
    - [SearchRequest](#search-v1alpha2-SearchRequest)
    - [SearchResponse](#search-v1alpha2-SearchResponse)
  
    - [SearchService](#search-v1alpha2-SearchService)
  
- [routing/v1alpha2/routing_service.proto](#routing_v1alpha2_routing_service-proto)
    - [ListRequest](#routing-v1alpha2-ListRequest)
    - [ListResponse](#routing-v1alpha2-ListResponse)
    - [PublishRequest](#routing-v1alpha2-PublishRequest)
    - [SearchRequest](#routing-v1alpha2-SearchRequest)
    - [SearchResponse](#routing-v1alpha2-SearchResponse)
    - [UnpublishRequest](#routing-v1alpha2-UnpublishRequest)
  
    - [RoutingService](#routing-v1alpha2-RoutingService)
  
- [routing/v1alpha2/peer.proto](#routing_v1alpha2_peer-proto)
    - [Peer](#routing-v1alpha2-Peer)
    - [Peer.AnnotationsEntry](#routing-v1alpha2-Peer-AnnotationsEntry)
  
    - [PeerConnectionType](#routing-v1alpha2-PeerConnectionType)
  
- [routing/v1alpha2/record_query.proto](#routing_v1alpha2_record_query-proto)
    - [RecordQuery](#routing-v1alpha2-RecordQuery)
  
    - [RecordQueryType](#routing-v1alpha2-RecordQueryType)
  
- [routing/v1alpha1/routing_service.proto](#routing_v1alpha1_routing_service-proto)
    - [ListRequest](#routing-v1alpha1-ListRequest)
    - [ListResponse](#routing-v1alpha1-ListResponse)
    - [ListResponse.Item](#routing-v1alpha1-ListResponse-Item)
    - [ListResponse.Item.LabelCountsEntry](#routing-v1alpha1-ListResponse-Item-LabelCountsEntry)
    - [PublishRequest](#routing-v1alpha1-PublishRequest)
    - [UnpublishRequest](#routing-v1alpha1-UnpublishRequest)
  
    - [RoutingService](#routing-v1alpha1-RoutingService)
  
- [routing/v1alpha1/peer.proto](#routing_v1alpha1_peer-proto)
    - [Peer](#routing-v1alpha1-Peer)
  
    - [ConnectionType](#routing-v1alpha1-ConnectionType)
  
- [sign/v1alpha2/sign_service.proto](#sign_v1alpha2_sign_service-proto)
    - [SignRequest](#sign-v1alpha2-SignRequest)
    - [SignRequestProvider](#sign-v1alpha2-SignRequestProvider)
    - [SignResponse](#sign-v1alpha2-SignResponse)
    - [SignWithKey](#sign-v1alpha2-SignWithKey)
    - [SignWithOIDC](#sign-v1alpha2-SignWithOIDC)
    - [SignWithOIDC.SignOpts](#sign-v1alpha2-SignWithOIDC-SignOpts)
    - [VerifyRequest](#sign-v1alpha2-VerifyRequest)
    - [VerifyRequestProvider](#sign-v1alpha2-VerifyRequestProvider)
    - [VerifyResponse](#sign-v1alpha2-VerifyResponse)
    - [VerifyWithKey](#sign-v1alpha2-VerifyWithKey)
    - [VerifyWithOIDC](#sign-v1alpha2-VerifyWithOIDC)
  
    - [SignService](#sign-v1alpha2-SignService)
  
- [sign/v1alpha1/sign_service.proto](#sign_v1alpha1_sign_service-proto)
    - [SignRequest](#sign-v1alpha1-SignRequest)
    - [SignRequestProvider](#sign-v1alpha1-SignRequestProvider)
    - [SignResponse](#sign-v1alpha1-SignResponse)
    - [SignWithKey](#sign-v1alpha1-SignWithKey)
    - [SignWithOIDC](#sign-v1alpha1-SignWithOIDC)
    - [SignWithOIDC.SignOpts](#sign-v1alpha1-SignWithOIDC-SignOpts)
    - [VerifyRequest](#sign-v1alpha1-VerifyRequest)
    - [VerifyRequestProvider](#sign-v1alpha1-VerifyRequestProvider)
    - [VerifyResponse](#sign-v1alpha1-VerifyResponse)
    - [VerifyWithKey](#sign-v1alpha1-VerifyWithKey)
    - [VerifyWithOIDC](#sign-v1alpha1-VerifyWithOIDC)
  
    - [SignService](#sign-v1alpha1-SignService)
  
- [objectmanager/record.proto](#objectmanager_record-proto)
    - [RecordObject](#objectmanager-RecordObject)
    - [RecordObjectData](#objectmanager-RecordObjectData)
  
    - [RecordObjectType](#objectmanager-RecordObjectType)
  
    - [RecordObjectConverter](#objectmanager-RecordObjectConverter)
  
- [store/v1alpha2/object.proto](#store_v1alpha2_object-proto)
    - [Object](#store-v1alpha2-Object)
    - [Object.AnnotationsEntry](#store-v1alpha2-Object-AnnotationsEntry)
    - [ObjectRef](#store-v1alpha2-ObjectRef)
  
    - [ObjectType](#store-v1alpha2-ObjectType)
  
- [store/v1alpha2/store_service.proto](#store_v1alpha2_store_service-proto)
    - [StoreService](#store-v1alpha2-StoreService)
  
- [store/v1alpha2/sync_service.proto](#store_v1alpha2_sync_service-proto)
    - [CreateSyncRequest](#store-v1alpha2-CreateSyncRequest)
    - [CreateSyncResponse](#store-v1alpha2-CreateSyncResponse)
    - [DeleteSyncRequest](#store-v1alpha2-DeleteSyncRequest)
    - [GetSyncRequest](#store-v1alpha2-GetSyncRequest)
    - [GetSyncResponse](#store-v1alpha2-GetSyncResponse)
    - [ListSyncItem](#store-v1alpha2-ListSyncItem)
    - [ListSyncsRequest](#store-v1alpha2-ListSyncsRequest)
  
    - [SyncStatus](#store-v1alpha2-SyncStatus)
  
    - [SyncService](#store-v1alpha2-SyncService)
  
- [store/v1alpha1/store_service.proto](#store_v1alpha1_store_service-proto)
    - [StoreService](#store-v1alpha1-StoreService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="core_v1alpha2_extension-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha2/extension.proto



<a name="core-v1alpha2-Extension"></a>

### Extension
Extensions provide a generic way to attach additional information
about an agent to the record. For example, application-specific 
details can be provided using an extension.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| annotations | [Extension.AnnotationsEntry](#core-v1alpha2-Extension-AnnotationsEntry) | repeated | Metadata associated with the extension. |
| name | [string](#string) |  | Name of the extension. Can be used as a fully qualified name. For example, &#34;org.agntcy.oasf.schema/features/&lt;feature-name&gt;&#34; |
| version | [string](#string) |  | Version of the extension. |
| data | [google.protobuf.Struct](#google-protobuf-Struct) |  | Data attached to the extension. Usually a JSON-embedded object. |






<a name="core-v1alpha2-Extension-AnnotationsEntry"></a>

### Extension.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="core_v1alpha2_locator-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha2/locator.proto



<a name="core-v1alpha2-Locator"></a>

### Locator
Locator points to the source where agent can be found at.
For example, a locator can be a link to a helm chart.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| annotations | [Locator.AnnotationsEntry](#core-v1alpha2-Locator-AnnotationsEntry) | repeated | Metadata associated with the locator. |
| type | [string](#string) |  | Type of the locator. Supports custom types. Native types are defined in the LocatorType. |
| url | [string](#string) |  | Location where the source can be found at. Specs: https://datatracker.ietf.org/doc/html/rfc1738 |
| size | [uint64](#uint64) | optional | Size of the source in bytes pointed by the {url} property. |
| digest | [string](#string) | optional | Digest of the source pointed by the {url} property. Specs: https://github.com/opencontainers/image-spec/blob/main/descriptor.md#digests |






<a name="core-v1alpha2-Locator-AnnotationsEntry"></a>

### Locator.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="core-v1alpha2-LocatorType"></a>

### LocatorType
LocatorType defines placeholders for supported locators.
Used in string format across APIs.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LOCATOR_TYPE_UNSPECIFIED | 0 | &#34;&#34; |
| LOCATOR_TYPE_HELM_CHART | 1 | &#34;helm-chart&#34; |
| LOCATOR_TYPE_DOCKER_IMAGE | 2 | &#34;docker-image&#34; |
| LOCATOR_TYPE_PYTHON_PACKAGE | 3 | &#34;python-package&#34; |
| LOCATOR_TYPE_SOURCE_CODE | 4 | &#34;source-code&#34; |
| LOCATOR_TYPE_BINARY | 5 | &#34;binary&#34; |


 

 

 



<a name="core_v1alpha2_record-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha2/record.proto



<a name="core-v1alpha2-Record"></a>

### Record
Record defines a schema for versioned AI agent content representation.
The schema provides a way to describe an agent in a structured format.

This is a versioned gRPC-based OASF schema.

Max size: 4 MB (or to fully fit in a single request)
It may be required to support larger record size in the future.

Records are stored in a content-addressable store.
Records can be indexed for quick lookups and searches
to avoid unnecessary data transfer.

All records are referenced by a globally-unique content identifier (CID).
Specs: https://github.com/multiformats/cid


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| annotations | [Record.AnnotationsEntry](#core-v1alpha2-Record-AnnotationsEntry) | repeated | Metadata associated with the record. |
| name | [string](#string) |  | Name of the agent. |
| version | [string](#string) |  | Version of the agent. |
| description | [string](#string) |  | Description of the agent. |
| authors | [string](#string) | repeated | List of agent’s authors, e.g. in the form of `author-name &lt;author-email&gt;`. |
| created_at | [string](#string) |  | Creation timestamp of the agent in the RFC3339 format. Specs: https://www.rfc-editor.org/rfc/rfc3339.html |
| skills | [Skill](#core-v1alpha2-Skill) | repeated | List of skills that the agent can perform. |
| locators | [Locator](#core-v1alpha2-Locator) | repeated | List of source locators where the agent can be found or used from. |
| extensions | [Extension](#core-v1alpha2-Extension) | repeated | Additional information attached to the agent. Extensions are used to generically extend the agent&#39;s functionality. |
| signature | [Signature](#core-v1alpha2-Signature) |  | Security signature of the agent. |
| previous_record_cid | [string](#string) | optional | Reference to the previous agent record, if any. Used to link the agent record to its previous versions. Field number is explicitly reserved for extenability. |






<a name="core-v1alpha2-Record-AnnotationsEntry"></a>

### Record.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="core_v1alpha2_skill-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha2/skill.proto



<a name="core-v1alpha2-Skill"></a>

### Skill
A specific skills that an agent is capable of performing.
Supported skills: https://schema.oasf.agntcy.org/skills


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| annotations | [Skill.AnnotationsEntry](#core-v1alpha2-Skill-AnnotationsEntry) | repeated | Metadata associated with the skill. |
| id | [uint32](#uint32) |  | Unique identifier of the skill. |
| name | [string](#string) |  | Human-readable name of the skill. |






<a name="core-v1alpha2-Skill-AnnotationsEntry"></a>

### Skill.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="core_v1alpha2_signature-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha2/signature.proto



<a name="core-v1alpha2-Signature"></a>

### Signature
Signature provides the signing and verification
details about the agent record.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| annotations | [Signature.AnnotationsEntry](#core-v1alpha2-Signature-AnnotationsEntry) | repeated | Metadata associated with the signature. |
| signed_at | [string](#string) |  | Signing timestamp of the agent in the RFC3339 format. Specs: https://www.rfc-editor.org/rfc/rfc3339.html |
| algorithm | [string](#string) |  | The signature algorithm used (e.g., &#34;ECDSA_P256_SHA256&#34;). |
| signature | [string](#string) |  | Base64-encoded signature. |
| certificate | [string](#string) |  | Base64-encoded signing certificate. |
| content_type | [string](#string) |  | Type of the signature content bundle. |
| content_bundle | [string](#string) |  | Base64-encoded signature bundle produced by the signer. It is up to the client to interpret the content of the bundle. |






<a name="core-v1alpha2-Signature-AnnotationsEntry"></a>

### Signature.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="core_v1alpha1_extension-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha1/extension.proto



<a name="core-v1alpha1-Extension"></a>

### Extension
Extensions provide dynamic descriptors for an agent data model.
For example, arbitrary data and third-party features can be
described using extensions.

Key := {name}/{version}

This is an immutable object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the extension attached to an agent. |
| version | [string](#string) |  | Version of the extension attached to an agent. |
| annotations | [Extension.AnnotationsEntry](#core-v1alpha1-Extension-AnnotationsEntry) | repeated | Metadata associated with this extension. |
| data | [google.protobuf.Struct](#google-protobuf-Struct) | optional | Value of the data.

Reference to the data on the content storage. This allows model producers and extension consumers to leverage the storage layer and bypass restrictions on the request size.

NOTE: currently not used optional ObjectRef data_ref = 5; |






<a name="core-v1alpha1-Extension-AnnotationsEntry"></a>

### Extension.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="core_v1alpha1_locator-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha1/locator.proto



<a name="core-v1alpha1-Locator"></a>

### Locator
Locator points to the artifact locators for an agent data model.
For example, this can include a reference to a helm chart.

Key := {type}/{url}

This is an immutable object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | Type of the locator. Can be custom or native LocatorType. |
| url | [string](#string) |  | Location URI where this source can be found/accessed. Specs: https://datatracker.ietf.org/doc/html/rfc1738 |
| annotations | [Locator.AnnotationsEntry](#core-v1alpha1-Locator-AnnotationsEntry) | repeated | Metadata associated with this locator. |
| size | [uint64](#uint64) | optional | Size of the source in bytes pointed by the {url} property. |
| digest | [string](#string) | optional | Digest of the source pointed by the {url} property. Specs: https://github.com/opencontainers/image-spec/blob/main/descriptor.md#digests |






<a name="core-v1alpha1-Locator-AnnotationsEntry"></a>

### Locator.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="core-v1alpha1-LocatorType"></a>

### LocatorType
LocatorType defines native types of locators.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LOCATOR_TYPE_UNSPECIFIED | 0 | &#34;&#34;, &#34;unspecified&#34; |
| LOCATOR_TYPE_HELM_CHART | 1 | &#34;helm-chart&#34; |
| LOCATOR_TYPE_DOCKER_IMAGE | 2 | &#34;docker-image&#34; |
| LOCATOR_TYPE_PYTHON_PACKAGE | 3 | &#34;python-package&#34; |
| LOCATOR_TYPE_SOURCE_CODE | 4 | &#34;source-code&#34; |
| LOCATOR_TYPE_BINARY | 5 | &#34;binary&#34; |


 

 

 



<a name="core_v1alpha1_object-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha1/object.proto



<a name="core-v1alpha1-Object"></a>

### Object
Object maps a given ref to an actual object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  | Opaque data held by this object. Arbitrary in size. Readers should know how to process this value. |
| ref | [ObjectRef](#core-v1alpha1-ObjectRef) | optional | This is only needed in push. |
| agent | [Agent](#core-v1alpha1-Agent) | optional | In case ref is pointing to an agent model, this can be set. This is only set on pull. |






<a name="core-v1alpha1-ObjectRef"></a>

### ObjectRef
Reference to a typed object in the content store.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| digest | [string](#string) |  | Digest of the object data. Digest is used as a globally unique ID of an object. Specs: https://github.com/opencontainers/image-spec/blob/main/descriptor.md#digests |
| type | [string](#string) |  | Type of the object. Can be looked up from digest. |
| size | [uint64](#uint64) |  | Size of the object. Can be looked up from digest. |
| annotations | [ObjectRef.AnnotationsEntry](#core-v1alpha1-ObjectRef-AnnotationsEntry) | repeated | Additional metadata associated with this object. Can be looked up from digest. |






<a name="core-v1alpha1-ObjectRef-AnnotationsEntry"></a>

### ObjectRef.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="core-v1alpha1-ObjectType"></a>

### ObjectType
ObjectType defines a list of native types.
Other types may be used but some operations
on the storage layer may not be supported.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OBJECT_TYPE_RAW | 0 | &#34;raw&#34; |
| OBJECT_TYPE_AGENT | 1 | &#34;agent&#34; |


 

 

 



<a name="core_v1alpha1_skill-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha1/skill.proto



<a name="core-v1alpha1-Skill"></a>

### Skill
A specific skills that an agent is capable of performing.
Specs: https://schema.oasf.agntcy.org/skills.

Example (https://schema.oasf.agntcy.org/skills/contextual_comprehension)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| annotations | [Skill.AnnotationsEntry](#core-v1alpha1-Skill-AnnotationsEntry) | repeated | Additional metadata for this skill. |
| category_uid | [uint64](#uint64) |  | UID of the category. |
| class_uid | [uint64](#uint64) |  | UID of the class. |
| category_name | [string](#string) | optional | Optional human-readable name of the category. |
| class_name | [string](#string) | optional | Optional human-readable name of the class. |






<a name="core-v1alpha1-Skill-AnnotationsEntry"></a>

### Skill.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="core_v1alpha1_signature-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha1/signature.proto



<a name="core-v1alpha1-Signature"></a>

### Signature



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| algorithm | [string](#string) |  | The signature algorithm used (e.g., &#34;ECDSA_P256_SHA256&#34;) |
| signature | [string](#string) |  | Base64-encoded signature |
| certificate | [string](#string) |  | Base64-encoded signing certificate |
| content_type | [string](#string) |  | Type of the signature content bundle. |
| content_bundle | [string](#string) |  | Base64-encoded signature bundle produced by the signer. It is up to the client to interpret the content of the bundle. |
| signed_at | [string](#string) |  | Timestamp when signing occurred |





 

 

 

 



<a name="core_v1alpha1_agent-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## core/v1alpha1/agent.proto



<a name="core-v1alpha1-Agent"></a>

### Agent
Data model defines a schema for versioned AI agent content representation.
The schema provides a way to describe features, constraints, artifact
locators, and other relevant details of an agent.

Key := {name} - newest release
Key := {name}:{version} - newest versioned release
Key := {name}@{digest} - exact release
Key := {name}:{version}@{digest} - exact versioned release

This is an immutable object.

Max size: 4 MB (or to fully fit in a single request)
https://opencontainers.org/posts/blog/2024-03-13-image-and-distribution-1-1/#manifest-maximum-size


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_version | [string](#string) |  | Schema version of the agent. |
| name | [string](#string) |  | Name of the agent. |
| version | [string](#string) |  | Version of the agent. |
| description | [string](#string) |  | Description of the agent. |
| authors | [string](#string) | repeated | List of agent’s authors in the form of `author-name &lt;author-email&gt;`. |
| created_at | [string](#string) |  | Creation timestamp of the agent in the RFC3339 format. Specs: https://www.rfc-editor.org/rfc/rfc3339.html |
| annotations | [Agent.AnnotationsEntry](#core-v1alpha1-Agent-AnnotationsEntry) | repeated | Additional metadata associated with this agent. |
| skills | [Skill](#core-v1alpha1-Skill) | repeated | List of skills that this agent can perform. |
| locators | [Locator](#core-v1alpha1-Locator) | repeated | List of source locators where this agent can be found or used from. |
| extensions | [Extension](#core-v1alpha1-Extension) | repeated | List of extensions that describe this agent and its capabilities and constraints more in depth. |
| signature | [Signature](#core-v1alpha1-Signature) | optional | Signature attached to this agent. |






<a name="core-v1alpha1-Agent-AnnotationsEntry"></a>

### Agent.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="search_v1alpha2_record_query-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## search/v1alpha2/record_query.proto



<a name="search-v1alpha2-RecordQuery"></a>

### RecordQuery
A query to match the record against during discovery.
For example:
 { type: RECORD_QUERY_TYPE_SKILL_NAME, value: &#34;Natural Language Processing&#34; }
 { type: RECORD_QUERY_TYPE_LOCATOR, value: &#34;docker-image:https://example.com/docker-image&#34; }


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [RecordQueryType](#search-v1alpha2-RecordQueryType) |  | The type of the query to match against. |
| value | [string](#string) |  | The query value to match against. |





 


<a name="search-v1alpha2-RecordQueryType"></a>

### RecordQueryType
Defines a list of supported record query types.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RECORD_QUERY_TYPE_UNSPECIFIED | 0 | Unspecified query type. |
| RECORD_QUERY_TYPE_NAME | 1 | Query for an agent name. |
| RECORD_QUERY_TYPE_VERSION | 2 | Query for an agent version. |
| RECORD_QUERY_TYPE_SKILL_ID | 3 | Query for a skill ID. |
| RECORD_QUERY_TYPE_SKILL_NAME | 4 | Query for a skill name. |
| RECORD_QUERY_TYPE_LOCATOR | 5 | Query for a locator. |
| RECORD_QUERY_TYPE_EXTENSION | 6 | Query for an extension. |


 

 

 



<a name="search_v1alpha2_search_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## search/v1alpha2/search_service.proto



<a name="search-v1alpha2-SearchRequest"></a>

### SearchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| queries | [RecordQuery](#search-v1alpha2-RecordQuery) | repeated | List of queries to match against the records. |
| limit | [uint32](#uint32) | optional | Optional limit on the number of results to return. |
| offset | [uint32](#uint32) | optional | Optional offset for pagination of results. |






<a name="search-v1alpha2-SearchResponse"></a>

### SearchResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_cid | [string](#string) |  | The CID of the record that matches the search criteria. |





 

 

 


<a name="search-v1alpha2-SearchService"></a>

### SearchService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Search | [SearchRequest](#search-v1alpha2-SearchRequest) | [SearchResponse](#search-v1alpha2-SearchResponse) stream | List records that this peer is currently providing that match the given parameters. This operation does not interact with the network. |

 



<a name="routing_v1alpha2_routing_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## routing/v1alpha2/routing_service.proto



<a name="routing-v1alpha2-ListRequest"></a>

### ListRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| queries | [RecordQuery](#routing-v1alpha2-RecordQuery) | repeated | List of queries to match against the records. If set, all queries must match for the record to be returned. |
| limit | [uint32](#uint32) | optional | Limit the number of results returned. If not set, it will return all records that this peer is providing. |






<a name="routing-v1alpha2-ListResponse"></a>

### ListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_cid | [string](#string) |  | The record that matches the list queries. |






<a name="routing-v1alpha2-PublishRequest"></a>

### PublishRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_cid | [string](#string) |  | Reference to the agent record to be published. |






<a name="routing-v1alpha2-SearchRequest"></a>

### SearchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| queries | [RecordQuery](#routing-v1alpha2-RecordQuery) | repeated | List of queries to match against the records. |
| min_match_score | [uint32](#uint32) | optional | Minimal target query match score. For example, if min_match_score=2, it will return records that match at least two of the queries. If not set, it will return records that match at least one query. |
| limit | [uint32](#uint32) | optional | Limit the number of results returned. If not set, it will return all discovered records. Note that this is a soft limit, as the search may return more results than the limit if there are multiple peers providing the same record. |






<a name="routing-v1alpha2-SearchResponse"></a>

### SearchResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_cid | [string](#string) |  | The record that matches the search query. |
| peer | [Peer](#routing-v1alpha2-Peer) |  | The peer that provided the record. |
| match_queries | [RecordQuery](#routing-v1alpha2-RecordQuery) | repeated | The queries that were matched. |
| match_score | [uint32](#uint32) |  | The score of the search match. |






<a name="routing-v1alpha2-UnpublishRequest"></a>

### UnpublishRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_cid | [string](#string) |  | Reference to the agent record to be unpublished. |





 

 

 


<a name="routing-v1alpha2-RoutingService"></a>

### RoutingService
Defines an interface for announcement and discovery
of records across interconnected network.

Middleware should be used to control who can perform these RPCs.
Policies for the middleware can be handled via separate service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Publish | [PublishRequest](#routing-v1alpha2-PublishRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Announce to the network that this peer is providing a given record. This enables other peers to discover this record and retrieve it from this peer. Listeners can use this event to perform custom operations, for example by cloning the record.

Items need to be periodically republished (eg. 24h) to the network to avoid stale data. Republication should be done in the background. |
| Unpublish | [UnpublishRequest](#routing-v1alpha2-UnpublishRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Stop serving this record to the network. If other peers try to retrieve this record, the peer will refuse the request. |
| Search | [SearchRequest](#routing-v1alpha2-SearchRequest) | [SearchResponse](#routing-v1alpha2-SearchResponse) stream | Search records based on the request across the network. This will search the network for the record with the given parameters.

It is possible that the records are stale or that they do not exist. Some records may be provided by multiple peers.

Results from the search can be used as an input to Pull operation to retrieve the records. |
| List | [ListRequest](#routing-v1alpha2-ListRequest) | [ListResponse](#routing-v1alpha2-ListResponse) stream | List all records that this peer is currently providing that match the given parameters. This operation does not interact with the network. |

 



<a name="routing_v1alpha2_peer-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## routing/v1alpha2/peer.proto



<a name="routing-v1alpha2-Peer"></a>

### Peer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of a given peer, typically described by a protocol. For example: - SPIFFE: &#34;spiffe://example.org/service/foo&#34; - JWT: &#34;jwt:sub=alice,iss=https://issuer.example.com&#34; - Tor: &#34;onion:abcdefghijklmno.onion&#34; - DID: &#34;did:example:123456789abcdefghi&#34; - IPFS: &#34;ipfs:QmYwAPJzv5CZsnAzt8auVZRn2E6sD1c4x8pN5o6d5cW4D5&#34; |
| addrs | [string](#string) | repeated | Multiaddrs for a given peer. For example: - &#34;/ip4/127.0.0.1/tcp/4001&#34; - &#34;/ip6/::1/tcp/4001&#34; - &#34;/dns4/example.com/tcp/443/https&#34; |
| annotations | [Peer.AnnotationsEntry](#routing-v1alpha2-Peer-AnnotationsEntry) | repeated | Additional metadata about the peer. |
| connection | [PeerConnectionType](#routing-v1alpha2-PeerConnectionType) |  | Used to signal the sender&#39;s connection capabilities to the peer. |






<a name="routing-v1alpha2-Peer-AnnotationsEntry"></a>

### Peer.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="routing-v1alpha2-PeerConnectionType"></a>

### PeerConnectionType


| Name | Number | Description |
| ---- | ------ | ----------- |
| PEER_CONNECTION_TYPE_NOT_CONNECTED | 0 | Sender does not have a connection to peer, and no extra information (default) |
| PEER_CONNECTION_TYPE_CONNECTED | 1 | Sender has a live connection to peer. |
| PEER_CONNECTION_TYPE_CAN_CONNECT | 2 | Sender recently connected to peer. |
| PEER_CONNECTION_TYPE_CANNOT_CONNECT | 3 | Sender made strong effort to connect to peer repeatedly but failed. |


 

 

 



<a name="routing_v1alpha2_record_query-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## routing/v1alpha2/record_query.proto



<a name="routing-v1alpha2-RecordQuery"></a>

### RecordQuery
A query to match the record against during discovery.
For example:
 { type: RECORD_QUERY_TYPE_SKILL, value: &#34;Natural Language Processing&#34; }
 { type: RECORD_QUERY_TYPE_LOCATOR, value: &#34;helm-chart&#34; }


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [RecordQueryType](#routing-v1alpha2-RecordQueryType) |  | The type of the query to match against. |
| value | [string](#string) |  | The query value to match against. |





 


<a name="routing-v1alpha2-RecordQueryType"></a>

### RecordQueryType
Defines a list of supported record query types.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RECORD_QUERY_TYPE_UNSPECIFIED | 0 | Unspecified query type. |
| RECORD_QUERY_TYPE_SKILL | 1 | Query for a skill name. |
| RECORD_QUERY_TYPE_LOCATOR | 2 | Query for a locator type. |


 

 

 



<a name="routing_v1alpha1_routing_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## routing/v1alpha1/routing_service.proto



<a name="routing-v1alpha1-ListRequest"></a>

### ListRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| peer | [Peer](#routing-v1alpha1-Peer) | optional | Target peer. If selected, it returns the skill details for this peer. It may use labels to only return selected labels. |
| labels | [string](#string) | repeated | Target labels. For example, labels={&#34;skill=text&#34;, &#34;skill=text/rag&#34;} |
| record | [core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | optional | Target object, if any. If set, it will return only the object with the given reference. |
| max_hops | [uint32](#uint32) | optional | Max routing depth. |
| network | [bool](#bool) | optional | Run a networked query. |






<a name="routing-v1alpha1-ListResponse"></a>

### ListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [ListResponse.Item](#routing-v1alpha1-ListResponse-Item) | repeated | Returned items that match a given request |






<a name="routing-v1alpha1-ListResponse-Item"></a>

### ListResponse.Item



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [string](#string) | repeated | Labels associated with a given object |
| label_counts | [ListResponse.Item.LabelCountsEntry](#routing-v1alpha1-ListResponse-Item-LabelCountsEntry) | repeated | Optionally sends count details about individual skill. This is only set when querying labels or our own current peer. For record requests, only returns the data about that record. |
| peer | [Peer](#routing-v1alpha1-Peer) |  | Peer that returned this object. |
| record | [core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | optional | Found object if any. If empty, then only the labels are important. |






<a name="routing-v1alpha1-ListResponse-Item-LabelCountsEntry"></a>

### ListResponse.Item.LabelCountsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [uint64](#uint64) |  |  |






<a name="routing-v1alpha1-PublishRequest"></a>

### PublishRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record | [core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) |  | Published record reference. On publish, we read from the local store and extract the neccessary labels. |
| network | [bool](#bool) | optional | Announce the publication to the network. This item will end up in the network and can be searched. |






<a name="routing-v1alpha1-UnpublishRequest"></a>

### UnpublishRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record | [core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) |  | Reference to the record to be unpublished. On unpublish, we read from the local store and remove the associated labels. |
| network | [bool](#bool) | optional | Notify the network about the unpublication. This will remove the item from the network and make it no longer searchable. |





 

 

 


<a name="routing-v1alpha1-RoutingService"></a>

### RoutingService
Defines an interface for publication and retrieval
of objects across interconnected network.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Publish | [PublishRequest](#routing-v1alpha1-PublishRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Notifies the network that the node is providing given object. Listeners should use this event to update their routing tables. They may optionally forward the request to other nodes. Items need to be periodically republished to avoid stale data.

It is the API responsibility to fully construct the routing details, these are minimal details needed for us to publish the request. |
| List | [ListRequest](#routing-v1alpha1-ListRequest) | [ListResponse](#routing-v1alpha1-ListResponse) stream | List all the available items across the network. TODO: maybe remove to search? |
| Unpublish | [UnpublishRequest](#routing-v1alpha1-UnpublishRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Unpublish a given object. This will remove the object from the network. |

 



<a name="routing_v1alpha1_peer-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## routing/v1alpha1/peer.proto



<a name="routing-v1alpha1-Peer"></a>

### Peer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | ID of a given peer. |
| addrs | [string](#string) | repeated | Multiaddrs for a given peer. |
| connection | [ConnectionType](#routing-v1alpha1-ConnectionType) |  | Used to signal the sender&#39;s connection capabilities to the peer. |





 


<a name="routing-v1alpha1-ConnectionType"></a>

### ConnectionType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CONNECTION_TYPE_NOT_CONNECTED | 0 | Sender does not have a connection to peer, and no extra information (default) |
| CONNECTION_TYPE_CONNECTED | 1 | Sender has a live connection to peer |
| CONNECTION_TYPE_CAN_CONNECT | 2 | Sender recently connected to peer |
| CONNECTION_TYPE_CANNOT_CONNECT | 3 | Sender made strong effort to connect to peer repeatedly but failed |


 

 

 



<a name="sign_v1alpha2_sign_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## sign/v1alpha2/sign_service.proto



<a name="sign-v1alpha2-SignRequest"></a>

### SignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record | [core.v1alpha2.Record](#core-v1alpha2-Record) |  | Record to be signed |
| provider | [SignRequestProvider](#sign-v1alpha2-SignRequestProvider) |  | Signing provider to use |






<a name="sign-v1alpha2-SignRequestProvider"></a>

### SignRequestProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oidc | [SignWithOIDC](#sign-v1alpha2-SignWithOIDC) |  |  |
| key | [SignWithKey](#sign-v1alpha2-SignWithKey) |  |  |






<a name="sign-v1alpha2-SignResponse"></a>

### SignResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record | [core.v1alpha2.Record](#core-v1alpha2-Record) |  | Signed record |






<a name="sign-v1alpha2-SignWithKey"></a>

### SignWithKey



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| private_key | [bytes](#bytes) |  | Private key used for signing |
| password | [bytes](#bytes) | optional | Password to unlock the private key |






<a name="sign-v1alpha2-SignWithOIDC"></a>

### SignWithOIDC



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_token | [string](#string) |  | Token for OIDC provider |
| options | [SignWithOIDC.SignOpts](#sign-v1alpha2-SignWithOIDC-SignOpts) |  | Signing options for OIDC |






<a name="sign-v1alpha2-SignWithOIDC-SignOpts"></a>

### SignWithOIDC.SignOpts
List of sign options for OIDC


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fulcio_url | [string](#string) | optional | Fulcio authority access URL (default value: https://fulcio.sigstage.dev) |
| rekor_url | [string](#string) | optional | Rekor validator access URL (default value: https://rekor.sigstage.dev) |
| timestamp_url | [string](#string) | optional | Timestamp authority access URL (default value: https://timestamp.sigstage.dev/api/v1/timestamp) |
| oidc_provider_url | [string](#string) | optional | OIDC provider access URL (default value: https://oauth2.sigstage.dev/auth) |






<a name="sign-v1alpha2-VerifyRequest"></a>

### VerifyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record | [core.v1alpha2.Record](#core-v1alpha2-Record) |  | Signed record to be verified |
| provider | [VerifyRequestProvider](#sign-v1alpha2-VerifyRequestProvider) |  | Verification provider to use |






<a name="sign-v1alpha2-VerifyRequestProvider"></a>

### VerifyRequestProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oidc | [VerifyWithOIDC](#sign-v1alpha2-VerifyWithOIDC) |  |  |
| key | [VerifyWithKey](#sign-v1alpha2-VerifyWithKey) |  |  |






<a name="sign-v1alpha2-VerifyResponse"></a>

### VerifyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  | The verify process result |






<a name="sign-v1alpha2-VerifyWithKey"></a>

### VerifyWithKey



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| public_key | [bytes](#bytes) |  | Public key to validate the signed record |






<a name="sign-v1alpha2-VerifyWithOIDC"></a>

### VerifyWithOIDC



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expected_issuer | [string](#string) |  | Expected issuer in the signed record |
| expected_signer | [string](#string) |  | Expected signer in the signed record |





 

 

 


<a name="sign-v1alpha2-SignService"></a>

### SignService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Sign | [SignRequest](#sign-v1alpha2-SignRequest) | [SignResponse](#sign-v1alpha2-SignResponse) | Sign record using keyless OIDC based provider or using PEM-encoded private key with an optional passphrase |
| Verify | [VerifyRequest](#sign-v1alpha2-VerifyRequest) | [VerifyResponse](#sign-v1alpha2-VerifyResponse) | Verify signed record using keyless OIDC based provider or using PEM-encoded formatted PEM public key encrypted |

 



<a name="sign_v1alpha1_sign_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## sign/v1alpha1/sign_service.proto



<a name="sign-v1alpha1-SignRequest"></a>

### SignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent | [core.v1alpha1.Agent](#core-v1alpha1-Agent) |  | Agent to be signed |
| provider | [SignRequestProvider](#sign-v1alpha1-SignRequestProvider) |  | Signing provider to use |






<a name="sign-v1alpha1-SignRequestProvider"></a>

### SignRequestProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oidc | [SignWithOIDC](#sign-v1alpha1-SignWithOIDC) |  |  |
| key | [SignWithKey](#sign-v1alpha1-SignWithKey) |  |  |






<a name="sign-v1alpha1-SignResponse"></a>

### SignResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent | [core.v1alpha1.Agent](#core-v1alpha1-Agent) |  | Signed agent |






<a name="sign-v1alpha1-SignWithKey"></a>

### SignWithKey



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| private_key | [bytes](#bytes) |  | Private key used for signing |
| password | [bytes](#bytes) | optional | Password to unlock the private key |






<a name="sign-v1alpha1-SignWithOIDC"></a>

### SignWithOIDC



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_token | [string](#string) |  | Token for OIDC provider |
| options | [SignWithOIDC.SignOpts](#sign-v1alpha1-SignWithOIDC-SignOpts) |  | Signing options for OIDC |






<a name="sign-v1alpha1-SignWithOIDC-SignOpts"></a>

### SignWithOIDC.SignOpts
List of sign options for OIDC


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fulcio_url | [string](#string) | optional | Fulcio authority access URL (default value: https://fulcio.sigstage.dev) |
| rekor_url | [string](#string) | optional | Rekor validator access URL (default value: https://rekor.sigstage.dev) |
| timestamp_url | [string](#string) | optional | Timestamp authority access URL (default value: https://timestamp.sigstage.dev/api/v1/timestamp) |
| oidc_provider_url | [string](#string) | optional | OIDC provider access URL (default value: https://oauth2.sigstage.dev/auth) |






<a name="sign-v1alpha1-VerifyRequest"></a>

### VerifyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| agent | [core.v1alpha1.Agent](#core-v1alpha1-Agent) |  | Signed agent to be verified |
| provider | [VerifyRequestProvider](#sign-v1alpha1-VerifyRequestProvider) |  | Verification provider to use |






<a name="sign-v1alpha1-VerifyRequestProvider"></a>

### VerifyRequestProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oidc | [VerifyWithOIDC](#sign-v1alpha1-VerifyWithOIDC) |  |  |
| key | [VerifyWithKey](#sign-v1alpha1-VerifyWithKey) |  |  |






<a name="sign-v1alpha1-VerifyResponse"></a>

### VerifyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  | The verify process result |






<a name="sign-v1alpha1-VerifyWithKey"></a>

### VerifyWithKey



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| public_key | [bytes](#bytes) |  | Public key to validate the signed agent |






<a name="sign-v1alpha1-VerifyWithOIDC"></a>

### VerifyWithOIDC



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expected_issuer | [string](#string) |  | Expected issuer in the signed agent |
| expected_signer | [string](#string) |  | Expected signer in the signed agent |





 

 

 


<a name="sign-v1alpha1-SignService"></a>

### SignService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Sign | [SignRequest](#sign-v1alpha1-SignRequest) | [SignResponse](#sign-v1alpha1-SignResponse) | Sign agents using keyless OIDC based provider or using PEM-encoded private key with an optional passphrase |
| Verify | [VerifyRequest](#sign-v1alpha1-VerifyRequest) | [VerifyResponse](#sign-v1alpha1-VerifyResponse) | Verify signed agents using keyless OIDC based provider or using PEM-encoded formatted PEM public key encrypted |

 



<a name="objectmanager_record-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## objectmanager/record.proto



<a name="objectmanager-RecordObject"></a>

### RecordObject
Unifies different agent records into a single object.
Allows to handle different versions of record objects
in a single way, without worrying about the schema.

RecordObject and Object are composable types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cid | [string](#string) |  | CID of the record. |
| type | [RecordObjectType](#objectmanager-RecordObjectType) |  | Type of the record. |
| record | [RecordObjectData](#objectmanager-RecordObjectData) |  | Data of the record. |






<a name="objectmanager-RecordObjectData"></a>

### RecordObjectData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_v1alpha1 | [core.v1alpha1.Agent](#core-v1alpha1-Agent) |  |  |
| record_v1alpha2 | [core.v1alpha2.Record](#core-v1alpha2-Record) |  |  |





 


<a name="objectmanager-RecordObjectType"></a>

### RecordObjectType


| Name | Number | Description |
| ---- | ------ | ----------- |
| RECORD_OBJECT_TYPE_UNSPECIFIED | 0 | invalid type, should not be used |
| RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON | 1001 |  |
| RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON | 1002 |  |


 

 


<a name="objectmanager-RecordObjectConverter"></a>

### RecordObjectConverter
Converts Object to an RecordObject and vice versa.
This is client-side service only.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ConvertToRecordObject | [.store.v1alpha2.Object](#store-v1alpha2-Object) | [RecordObject](#objectmanager-RecordObject) | Converts an Object to a RecordObject. |
| ConvertFromRecordObject | [RecordObject](#objectmanager-RecordObject) | [.store.v1alpha2.Object](#store-v1alpha2-Object) | Converts a RecordObject to an Object. |

 



<a name="store_v1alpha2_object-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/v1alpha2/object.proto



<a name="store-v1alpha2-Object"></a>

### Object
Object is a generic data structure that can hold
arbitrary data. It is used to store and associate
objects in a content-addressable store.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cid | [string](#string) |  | Globally-unique content identifier of the object. Encodes fully-qualified type of the object as part of &#34;codec&#34;. Specs: https://github.com/multiformats/cid |
| type | [ObjectType](#store-v1alpha2-ObjectType) |  | Type of the object. Can be extracted from CID. |
| annotations | [Object.AnnotationsEntry](#store-v1alpha2-Object-AnnotationsEntry) | repeated | Metadata associated with the object. |
| created_at | [string](#string) |  | Creation timestamp of the object in the RFC3339 format. Specs: https://www.rfc-editor.org/rfc/rfc3339.html |
| size | [uint64](#uint64) |  | Size of the object in bytes. |
| data | [bytes](#bytes) | optional | Opaque data held by this object. Clients can use {type} to handle processing. |






<a name="store-v1alpha2-Object-AnnotationsEntry"></a>

### Object.AnnotationsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="store-v1alpha2-ObjectRef"></a>

### ObjectRef
Reference to a content-addressable object.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cid | [string](#string) |  | Globally-unique content identifier (CID) of the object. Specs: https://github.com/multiformats/cid |





 


<a name="store-v1alpha2-ObjectType"></a>

### ObjectType
Defines a list of supported object data types.
Some values may be reserved for future use.
These types are used as a &#34;codec&#34; in the CID.

| Name | Number | Description |
| ---- | ------ | ----------- |
| OBJECT_TYPE_UNSPECIFIED | 0 | invalid type, should not be used |
| OBJECT_TYPE_RAW | 1 | Common Object Types |


 

 

 



<a name="store_v1alpha2_store_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/v1alpha2/store_service.proto


 

 

 


<a name="store-v1alpha2-StoreService"></a>

### StoreService
Defines an interface for content-addressable storage
service for arbitrary objects such as blobs, files, etc.
It may also store metadata for pushed objects.

Store service can be implemented by various storage backends,
such as local file system, OCI registry, etc.

Middleware should be used to control who can perform these RPCs.
Policies for the middleware can be handled via separate service.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Push | [Object](#store-v1alpha2-Object) stream | [ObjectRef](#store-v1alpha2-ObjectRef) | Push performs streamed write operation for the provided object. Objects must be sent in chunks if larger than 4MB. All objects are stored in raw format.

Some object types such as OASF records may be validated. CID is ignored and is generated by the service. |
| Pull | [ObjectRef](#store-v1alpha2-ObjectRef) | [Object](#store-v1alpha2-Object) stream | Pull performs streamed read operation for the requested object. Object is sent back in chunks if larger than 4MB. |
| Lookup | [ObjectRef](#store-v1alpha2-ObjectRef) | [Object](#store-v1alpha2-Object) | Lookup resolves basic metadata for the object. It does not return the object data. |
| Delete | [ObjectRef](#store-v1alpha2-ObjectRef) | [.google.protobuf.Empty](#google-protobuf-Empty) | Remove performs delete operation for the requested object. |

 



<a name="store_v1alpha2_sync_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/v1alpha2/sync_service.proto



<a name="store-v1alpha2-CreateSyncRequest"></a>

### CreateSyncRequest
CreateSyncRequest defines the parameters for creating a new synchronization operation.

Currently supports basic synchronization of all objects from a remote Directory.
Future versions may include additional options for filtering, authentication,
and scheduling capabilities.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| remote_directory | [string](#string) |  | URL of the remote Directory node to synchronize from.

This should be a complete URL including protocol and port if non-standard. Examples: - &#34;https://directory.example.com&#34; - &#34;http://localhost:8080&#34; - &#34;https://directory.example.com:9443&#34; |






<a name="store-v1alpha2-CreateSyncResponse"></a>

### CreateSyncResponse
CreateSyncResponse contains the result of creating a new synchronization operation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sync_id | [string](#string) |  | Unique identifier for the created synchronization operation. This ID can be used with other SyncService RPCs to monitor and manage the sync. |






<a name="store-v1alpha2-DeleteSyncRequest"></a>

### DeleteSyncRequest
DeleteSyncRequest specifies which synchronization to delete.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sync_id | [string](#string) |  | Unique identifier of the synchronization operation to delete. |






<a name="store-v1alpha2-GetSyncRequest"></a>

### GetSyncRequest
GetSyncRequest specifies which synchronization status to retrieve.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sync_id | [string](#string) |  | Unique identifier of the synchronization operation to query. |






<a name="store-v1alpha2-GetSyncResponse"></a>

### GetSyncResponse
GetSyncResponse provides detailed information about a specific synchronization operation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique identifier of the synchronization operation. |
| status | [SyncStatus](#store-v1alpha2-SyncStatus) |  | Current status of the synchronization operation. |
| remote_directory | [string](#string) |  | URL of the remote Directory node being synchronized from. |
| created_time | [string](#string) |  | Timestamp when the synchronization operation was created in the RFC3339 format. Specs: https://www.rfc-editor.org/rfc/rfc3339.html |
| last_update_time | [string](#string) |  | Timestamp of the most recent status update for this synchronization in the RFC3339 format. |






<a name="store-v1alpha2-ListSyncItem"></a>

### ListSyncItem
ListSyncItem represents a single synchronization in the list of all syncs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sync_id | [string](#string) |  | Unique identifier of the synchronization operation. |
| status | [string](#string) |  | Brief status of the sync (e.g., &#34;in_progress&#34;, &#34;completed&#34;, &#34;failed&#34;). |
| remote_directory | [string](#string) |  | URL of the remote Directory being synchronized from. |






<a name="store-v1alpha2-ListSyncsRequest"></a>

### ListSyncsRequest
ListSyncsRequest specifies parameters for listing synchronization operations.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [uint32](#uint32) | optional | Optional limit on the number of results to return. |
| offset | [uint32](#uint32) | optional | Optional offset for pagination of results. |





 


<a name="store-v1alpha2-SyncStatus"></a>

### SyncStatus
SyncStatus enumeration defines the possible states of a synchronization operation.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SYNC_STATUS_UNSPECIFIED | 0 | Default/unset status - should not be used in practice |
| SYNC_STATUS_PENDING | 1 | Sync operation has been created but not yet started |
| SYNC_STATUS_IN_PROGRESS | 2 | Sync operation is actively discovering and transferring objects |
| SYNC_STATUS_COMPLETED | 3 | Sync operation completed successfully with all objects transferred |
| SYNC_STATUS_FAILED | 4 | Sync operation encountered an error and stopped |


 

 


<a name="store-v1alpha2-SyncService"></a>

### SyncService
SyncService provides functionality for synchronizing objects between Directory nodes.

This service enables one-way synchronization from a remote Directory node to the local node,
allowing distributed Directory instances to share and replicate objects. The service supports
both on-demand synchronization and tracking of sync operations through their lifecycle.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateSync | [CreateSyncRequest](#store-v1alpha2-CreateSyncRequest) | [CreateSyncResponse](#store-v1alpha2-CreateSyncResponse) | CreateSync initiates a new synchronization operation from a remote Directory node.

The operation is non-blocking and returns immediately with a sync ID that can be used to track progress and manage the sync operation.

Returns: A unique sync ID string for tracking the operation |
| ListSyncs | [ListSyncsRequest](#store-v1alpha2-ListSyncsRequest) | [ListSyncItem](#store-v1alpha2-ListSyncItem) stream | ListSyncs returns a stream of all sync operations known to the system.

This includes active, completed, and failed synchronizations.

Returns: A stream of sync IDs for all known synchronizations |
| GetSync | [GetSyncRequest](#store-v1alpha2-GetSyncRequest) | [GetSyncResponse](#store-v1alpha2-GetSyncResponse) | GetSync retrieves detailed status information for a specific synchronization.

Args: sync_id - The unique identifier of the sync operation Returns: Detailed status information for the specified sync |
| DeleteSync | [DeleteSyncRequest](#store-v1alpha2-DeleteSyncRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | DeleteSync removes a synchronization operation from the system.

Args: sync_id - The unique identifier of the sync operation to delete |

 



<a name="store_v1alpha1_store_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/v1alpha1/store_service.proto


 

 

 


<a name="store-v1alpha1-StoreService"></a>

### StoreService
Defines an interface for content-addressable storage
service for arbitrary data such as blobs, files, etc.
It may also store metadata for pushed objects.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Push | [.core.v1alpha1.Object](#core-v1alpha1-Object) stream | [.core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | Push performs streamed write operation for provided object. |
| Pull | [.core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | [.core.v1alpha1.Object](#core-v1alpha1-Object) stream | Pull performs streamed read operation for the requested object. |
| Lookup | [.core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | [.core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | Lookup resolves ref data from digest only. |
| Delete | [.core.v1alpha1.ObjectRef](#core-v1alpha1-ObjectRef) | [.google.protobuf.Empty](#google-protobuf-Empty) | Remove performs delete operation for the requested object. |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |


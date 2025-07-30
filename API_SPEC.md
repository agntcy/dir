# API Specification

This document describes Directory API interfaces and usage scenarios.
The API specification is defined and exposed via gRPC services.
All code snippets below are tested against the Directory `v0.2.0` release.

## Models

Defines all objects used to define schema and API specification.

It is defined in [api/proto/core/v1](api/proto/core/v1).

## Storage API

This API is responsible for managing content-addressable object storage operations.

It is defined in [api/proto/store/v1/store_service.proto](api/proto/store/v1/store_service.proto).

## Routing API

This API is responsible for managing peer and content routing data.

It is defined in [api/proto/routing/v1/routing_service.proto](api/proto/routing/v1/routing_service.proto).

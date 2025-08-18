# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
import logging
import os
from typing import Iterator, List, Optional, Tuple

import core.v1.record_pb2 as core_types
import grpc
import routing.v1.routing_service_pb2 as routing_types
import routing.v1.routing_service_pb2_grpc as routing_services
import search.v1.search_service_pb2 as search_types
import search.v1.search_service_pb2_grpc as search_services
import store.v1.store_service_pb2_grpc as store_services
import store.v1.store_service_pb2 as store_types

CHUNK_SIZE = 4096  # 4KB

logger = logging.getLogger("client")


class Config:
    DEFAULT_ENV_PREFIX = "DIRECTORY_CLIENT"
    DEFAULT_SERVER_ADDRESS = "0.0.0.0:8888"

    def __init__(self, server_address: str = DEFAULT_SERVER_ADDRESS):
        self.server_address = server_address

    @classmethod
    def load_from_env(cls) -> "Config":
        """Load configuration from environment variables"""
        prefix = cls.DEFAULT_ENV_PREFIX
        server_address = os.environ.get(
            f"{prefix}_SERVER_ADDRESS", cls.DEFAULT_SERVER_ADDRESS
        )
        return cls(server_address=server_address)


class Client:
    def __init__(self, config: Config):
        """Initialize the client with the given configuration.

        Args:
            config: The client configuration
        """
        # Create gRPC channel
        channel = grpc.insecure_channel(config.server_address)

        # Initialize service clients
        self.store_client = store_services.StoreServiceStub(channel)
        self.routing_client = routing_services.RoutingServiceStub(channel)
        self.search_client = search_services.SearchServiceStub(channel)

    @classmethod
    def new(cls, config: Optional[Config] = None) -> "Client":
        """Create a new client instance.

        Args:
            config: Optional configuration, will load from environment if not provided

        Returns:
            A new Client instance
        """
        if config is None:
            config = Config.load_from_env()
        return cls(config)

    def publish(
        self,
        req: routing_types.PublishRequest,
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> None:
        """Publish an object to the routing service.

        Args:
            req: Publish request containing the cid of published object
            metadata: Optional metadata for the gRPC call
        Raises:
            Exception: If publishing fails
        """

        try:
            self.routing_client.Publish(req, metadata=metadata)
        except Exception as e:
            raise Exception(f"Failed to publish object: {e}")

    def list(
        self,
        req: routing_types.ListRequest,
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> Iterator[routing_types.ListResponse]:
        """List objects matching the criteria.

        Args:
            req: List request specifying criteria
            metadata: Optional metadata for the gRPC call

        Returns:
            Iterator yielding list response items

        Raises:
            Exception: If list operation fails
        """

        try:
            stream = self.routing_client.List(req, metadata=metadata)

            # Yield each item from the stream
            for response in stream:
                yield response
        except Exception as e:
            logger.error(f"Error receiving objects: {e}")
            raise Exception(f"Failed to list objects: {e}")

    def search(
        self,
        req: search_types.SearchRequest,
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> Iterator[routing_types.SearchResponse]:
        """Search objects matching the queries.

        Args:
            req: Search request specifying criteria
            metadata: Optional metadata for the gRPC call

        Returns:
            Search response object

        Raises: Exception if search fails
        """

        try:
            stream = self.search_client.Search(req, metadata=metadata)

            # Yield each item from the stream
            for response in stream:
                yield response
        except Exception as e:
            logger.error(f"Error receiving objects: {e}")
            raise Exception(f"Failed to search objects: {e}")

    def unpublish(
        self,
        req: routing_types.UnpublishRequest,
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> None:
        """Unpublish an object from the routing service.

        Args:
            req: Unpublish request containing the cid of unpublished object
            metadata: Optional metadata for the gRPC call
        Raises:
            Exception: If unpublishing fails
        """

        try:
            self.routing_client.Unpublish(req, metadata=metadata)
        except Exception as e:
            raise Exception(f"Failed to unpublish object: {e}")

    def push(
        self,
        records: List[core_types.Record],
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> List[core_types.RecordRef]:
        """Push an object to the store.

        Args:
            records: Records object
            metadata: Optional metadata for the gRPC call

        Returns:
            Updated object reference

        Raises:
            Exception: If push operation fails
        """

        references = []

        try:
            # Push is a client-streaming RPC - stream of requests, single response
            # Call the Push method with the request iterator

            response = self.store_client.Push(iter(records), metadata=metadata)

            for r in response:
                references.append(r)

        except Exception as e:
            raise Exception(f"Failed to push object: {e}")

        return references

    def push_referrer(
            self,
            req: List[store_types.PushReferrerRequest],
            metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> List[store_types.PushReferrerResponse]:
        """
        Push objects to the store.

        Args:
            req: PushReferrerRequest represents a record with optional OCI artifacts for push operations.
            metadata: Optional metadata for the gRPC call

        Returns:
            List of objects cid pushed to the store

        Raises:
            Exception: If push operation fails
        """

        responses = []

        try:
            response = self.store_client.PushReferrer(iter(req), metadata=metadata)

            for r in response:
                responses.append(r)

        except Exception as e:
            raise Exception(f"Failed to push object: {e}")

        return responses

    def pull(
        self,
        refs: List[core_types.RecordRef],
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> List[core_types.Record]:
        """Pull objects from the store.

        Args:
            refs: References to objects
            metadata: Optional metadata for the gRPC call

        Returns:
            BytesIO object containing the pulled data

        Raises:
            Exception: If pull operation fails
        """

        records = []

        try:
            response = self.store_client.Pull(iter(refs), metadata=metadata)

            for r in response:
                if r is not None:
                    records.append(r)

        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

        return records

    def pull_referrer(
            self,
            req: List[store_types.PullReferrerRequest],
            metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> List[store_types.PullReferrerResponse]:
        """
        Pull objects from the store.

        Args:
            req: PullReferrerRequest represents a record with optional OCI artifacts for pull operations.
            metadata: Optional metadata for the gRPC call

        Returns:
            List of record objects from the store

        Raises:
            Exception: If push operation fails
        """

        responses = []

        try:
            response = self.store_client.PullReferrer(iter(req), metadata=metadata)

            for r in response:
                responses.append(r)

        except Exception as e:
            raise Exception(f"Failed to push object: {e}")

        return responses

    def lookup(
        self,
        refs: List[core_types.RecordRef],
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> List[core_types.RecordMeta]:
        """Look up an object in the store.

        Args:
            refs: References to objects
            metadata: Optional metadata for the gRPC call

        Returns:
            Object metadata

        Raises:
            Exception: If lookup fails
        """

        metadatas = []

        try:
            response = self.store_client.Lookup(iter(refs), metadata=metadata)

            for r in response:
                if r is not None:
                    metadatas.append(r)

        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

        return metadatas

    def delete(
        self,
        refs: List[core_types.RecordRef],
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> None:
        """Delete an object from the store.

        Args:
            refs: References to objects
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If delete operation fails
        """

        try:
            self.store_client.Delete(iter(refs), metadata=metadata)

        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
import io
import logging
import os
from typing import Optional, List, Tuple, BinaryIO, Iterator

import grpc
import core.v1.record_pb2 as core_types
import routing.v1alpha2.routing_service_pb2_grpc as routing_services
import routing.v1alpha2.routing_service_pb2 as routing_types
import store.v1alpha2.store_service_pb2_grpc as store_services

CHUNK_SIZE = 4096  # 4KB

logger = logging.getLogger("client")

class Config:
    DEFAULT_ENV_PREFIX = "DIRECTORY_CLIENT"
    DEFAULT_SERVER_ADDRESS = "0.0.0.0:8888"

    def __init__(self, server_address: str = DEFAULT_SERVER_ADDRESS):
        self.server_address = server_address

    @classmethod
    def load_from_env(cls) -> 'Config':
        """Load configuration from environment variables"""
        prefix = cls.DEFAULT_ENV_PREFIX
        server_address = os.environ.get(f"{prefix}_SERVER_ADDRESS", cls.DEFAULT_SERVER_ADDRESS)
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

    @classmethod
    def new(cls, config: Optional[Config] = None) -> 'Client':
        """Create a new client instance.

        Args:
            config: Optional configuration, will load from environment if not provided

        Returns:
            A new Client instance
        """
        if config is None:
            config = Config.load_from_env()
        return cls(config)

    def publish(self, req: routing_types.PublishRequest, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
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

    def list(self, req: routing_types.ListRequest, metadata: Optional[List[Tuple[str, str]]] = None) -> Iterator[routing_types.ListResponse]:
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
            # List is a server-streaming RPC - single request, stream of responses
            stream = self.routing_client.List(req, metadata=metadata)

            # Yield each item from the stream
            for response in stream:
                yield response
        except Exception as e:
            logger.error(f"Error receiving objects: {e}")
            raise Exception(f"Failed to list objects: {e}")

    def unpublish(self, req: routing_types.UnpublishRequest, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
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

    def push(self, record: core_types.Record, metadata: Optional[List[Tuple[str, str]]] = None) -> core_types.RecordRef:
        """Push an object to the store.

        Args:
            record: Record object
            metadata: Optional metadata for the gRPC call

        Returns:
            Updated object reference

        Raises:
            Exception: If push operation fails
        """

        try:
            # Push is a client-streaming RPC - stream of requests, single response
            # Call the Push method with the request iterator

            def request_iterator():
                yield record

            response = self.store_client.Push(request_iterator(), metadata=metadata)

            return next(response, None)
        except Exception as e:
            raise Exception(f"Failed to push object: {e}")

    def pull(self, ref: core_types.RecordRef, metadata: Optional[List[Tuple[str, str]]] = None) -> core_types.Record:
        """Pull an object from the store.

        Args:
            ref: Reference to the object
            metadata: Optional metadata for the gRPC call

        Returns:
            BytesIO object containing the pulled data

        Raises:
            Exception: If pull operation fails
        """

        try:
            # Pull is a server-streaming RPC - single request, stream of responses
            def request_iterator():
                yield ref

            response = self.store_client.Pull(request_iterator(), metadata=metadata)

            for r in response:
                if r is not None:
                    return r

        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

    def lookup(self, ref: core_types.RecordRef, metadata: Optional[List[Tuple[str, str]]] = None) -> core_types.RecordMeta:
        """Look up an object in the store.

        Args:
            ref: Reference to the object
            metadata: Optional metadata for the gRPC call

        Returns:
            Object metadata

        Raises:
            Exception: If lookup fails
        """

        try:
            # Pull is a server-streaming RPC - single request, stream of responses
            def request_iterator():
                yield ref

            response = self.store_client.Lookup(request_iterator(), metadata=metadata)

            for r in response:
                if r is not None:
                    return r

        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

    def delete(self, ref: core_types.RecordRef, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Delete an object from the store.

        Args:
            ref: Reference to the object
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If delete operation fails
        """

        try:
            def request_iterator():
                yield ref

            self.store_client.Delete(request_iterator(), metadata=metadata)

        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

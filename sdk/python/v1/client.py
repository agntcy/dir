# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
import io
import logging
import os
from typing import Optional, List, Tuple, BinaryIO, Iterator

import grpc
import core.v1alpha1 as core
import routing.v1alpha1.routing_service_pb2_grpc as routing_services
import routing.v1alpha1.routing_service_pb2 as routing_types
import sign.v1alpha1.sign_service_pb2_grpc as sign_services
import sign.v1alpha1.sign_service_pb2 as sign_types
import store.v1alpha1.store_service_pb2_grpc as store_services

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

    def publish(self, ref: core.object_pb2.ObjectRef, network: bool = False, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Publish an object to the routing service.

        Args:
            ref: Reference to the object
            network: Whether to publish to the network
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If publishing fails
        """

        try:
            self.routing_client.Publish(
                routing_types.PublishRequest(record=ref, network=network),
                metadata=metadata,
            )
        except Exception as e:
            raise Exception(f"Failed to publish object: {e}")

    def list(self, req: routing_types.ListRequest, metadata: Optional[List[Tuple[str, str]]] = None) -> Iterator[routing_types.ListResponse.Item]:
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
                for item in response.items:
                    yield item
        except Exception as e:
            logger.error(f"Error receiving objects: {e}")
            raise Exception(f"Failed to list objects: {e}")

    def unpublish(self, ref: core.object_pb2.ObjectRef, network: bool = False, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Unpublish an object from the routing service.

        Args:
            ref: Reference to the object
            network: Whether to unpublish from the network
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If unpublishing fails
        """

        try:
            self.routing_client.Unpublish(
                routing_types.UnpublishRequest(record=ref, network=network),
                metadata=metadata
            )
        except Exception as e:
            raise Exception(f"Failed to unpublish object: {e}")

    def push(self, ref: core.object_pb2.ObjectRef, reader: BinaryIO, metadata: Optional[List[Tuple[str, str]]] = None) -> core.object_pb2.ObjectRef:
        """Push an object to the store.

        Args:
            ref: Reference to the object
            reader: Binary reader providing object data
            metadata: Optional metadata for the gRPC call

        Returns:
            Updated object reference

        Raises:
            Exception: If push operation fails
        """

        try:
            # Push is a client-streaming RPC - stream of requests, single response
            def request_iterator():
                while True:
                    data = reader.read(CHUNK_SIZE)
                    if not data:
                        break

                    obj = core.object_pb2.Object(ref=ref, data=data)
                    yield obj

            # Call the Push method with the request iterator
            response = self.store_client.Push(request_iterator(), metadata=metadata)
            return response
        except Exception as e:
            raise Exception(f"Failed to push object: {e}")

    def pull(self, ref: core.object_pb2.ObjectRef, metadata: Optional[List[Tuple[str, str]]] = None) -> io.BytesIO:
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
            stream = self.store_client.Pull(ref, metadata=metadata)

            buffer = io.BytesIO()
            for obj in stream:
                buffer.write(obj.data)

            buffer.seek(0)
            return buffer
        except Exception as e:
            raise Exception(f"Failed to pull object: {e}")

    def lookup(self, ref: core.object_pb2.ObjectRef, metadata: Optional[List[Tuple[str, str]]] = None) -> core.object_pb2.ObjectRef:
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
            # Lookup is a unary RPC - single request, single response
            return self.store_client.Lookup(ref, metadata=metadata)
        except Exception as e:
            raise Exception(f"Failed to lookup object: {e}")

    def delete(self, ref: core.object_pb2.ObjectRef, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Delete an object from the store.

        Args:
            ref: Reference to the object
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If delete operation fails
        """

        try:
            # Delete is a unary RPC - single request, single response
            self.store_client.Delete(ref, metadata=metadata)
        except Exception as e:
            raise Exception(f"Failed to delete object: {e}")

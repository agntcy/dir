# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import io
import logging
import os
from typing import Optional, List, Tuple, BinaryIO, Iterator

import grpc
import core.v1alpha2 as core
import routing.v1alpha2.routing_service_pb2_grpc as routing_services
import routing.v1alpha2.routing_service_pb2 as routing_types
import search.v1alpha2.search_service_pb2_grpc as search_services
import search.v1alpha2.search_service_pb2 as search_types
import sign.v1alpha2.sign_service_pb2_grpc as sign_services
import sign.v1alpha2.sign_service_pb2 as sign_types
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


    def publish(self, ref: core.object_pb2.ObjectRef, network: bool = False, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Publish an object to the routing service.

        Args:
            ref: Reference to the object
            network: Whether to publish to the network
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If publishing fails
        """


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

        raise NotImplementedError


    def unpublish(self, ref: core.object_pb2.ObjectRef, network: bool = False, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Unpublish an object from the routing service.

        Args:
            ref: Reference to the object
            network: Whether to unpublish from the network
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If unpublishing fails
        """

        raise NotImplementedError


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

        raise NotImplementedError


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

        raise NotImplementedError


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

        raise NotImplementedError


    def delete(self, ref: core.object_pb2.ObjectRef, metadata: Optional[List[Tuple[str, str]]] = None) -> None:
        """Delete an object from the store.

        Args:
            ref: Reference to the object
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If delete operation fails
        """

        raise NotImplementedError


    def sign(self, req: sign_types.SignRequest) -> sign_types.SignResponse:
        raise NotImplementedError


    def verify(self, req: sign_types.VerifyRequest) -> sign_types.VerifyResponse:
        raise NotImplementedError


    def local_search(self, req: search_types.SearchRequest) -> search_types.SearchResponse:
        raise NotImplementedError


    def network_search(self, req: routing_types.SearchRequest) -> routing_types.SearchResponse:
        raise NotImplementedError
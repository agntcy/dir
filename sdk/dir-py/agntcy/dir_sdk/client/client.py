# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""Client module for the AGNTCY Directory service.

This module provides a high-level Python client for interacting with the AGNTCY
Directory services including routing, search, store, and signing operations.
"""

import builtins
import logging
import os
from typing import List
import subprocess
import tempfile
from collections.abc import Sequence

import grpc
from cryptography.hazmat.primitives import serialization
from spiffe import WorkloadApiClient, X509Source

from agntcy.dir_sdk.client.config import Config
from agntcy.dir_sdk.models import (
    core_v1,
    events_v1,
    routing_v1,
    search_v1,
    sign_v1,
    store_v1,
)

logger = logging.getLogger("client")


class JWTAuthInterceptor(grpc.UnaryUnaryClientInterceptor, grpc.UnaryStreamClientInterceptor,
                          grpc.StreamUnaryClientInterceptor, grpc.StreamStreamClientInterceptor):
    """gRPC interceptor that adds JWT-SVID authentication to requests."""

    def __init__(self, socket_path: str, audience: str) -> None:
        """Initialize the JWT auth interceptor.

        Args:
            socket_path: Path to the SPIFFE Workload API socket
            audience: JWT audience claim for token validation

        """
        self.socket_path = socket_path
        self.audience = audience
        self._workload_client = WorkloadApiClient(socket_path=socket_path)

    def _get_jwt_token(self) -> str:
        """Fetch a JWT-SVID from the SPIRE Workload API.

        Returns:
            JWT token string

        Raises:
            RuntimeError: If unable to fetch JWT-SVID

        """
        try:
            # Fetch JWT-SVID with the configured audience
            jwt_svid = self._workload_client.fetch_jwt_svid(audience=[self.audience])
            if jwt_svid and jwt_svid.token:
                return jwt_svid.token
            msg = "Failed to fetch JWT-SVID: empty token"
            raise RuntimeError(msg)
        except Exception as e:
            msg = f"Failed to fetch JWT-SVID: {e}"
            raise RuntimeError(msg) from e

    def _add_jwt_metadata(self, client_call_details):
        """Add JWT token to request metadata."""
        token = self._get_jwt_token()
        metadata = []
        if client_call_details.metadata is not None:
            metadata = list(client_call_details.metadata)
        metadata.append(("authorization", f"Bearer {token}"))

        return grpc._interceptor._ClientCallDetails(
            method=client_call_details.method,
            timeout=client_call_details.timeout,
            metadata=metadata,
            credentials=client_call_details.credentials,
            wait_for_ready=client_call_details.wait_for_ready,
            compression=client_call_details.compression,
        )

    def intercept_unary_unary(self, continuation, client_call_details, request):
        """Intercept unary-unary RPC calls."""
        new_details = self._add_jwt_metadata(client_call_details)
        return continuation(new_details, request)

    def intercept_unary_stream(self, continuation, client_call_details, request):
        """Intercept unary-stream RPC calls."""
        new_details = self._add_jwt_metadata(client_call_details)
        return continuation(new_details, request)

    def intercept_stream_unary(self, continuation, client_call_details, request_iterator):
        """Intercept stream-unary RPC calls."""
        new_details = self._add_jwt_metadata(client_call_details)
        return continuation(new_details, request_iterator)

    def intercept_stream_stream(self, continuation, client_call_details, request_iterator):
        """Intercept stream-stream RPC calls."""
        new_details = self._add_jwt_metadata(client_call_details)
        return continuation(new_details, request_iterator)


class Client:
    """High-level client for interacting with AGNTCY Directory services.

    This client provides a unified interface for operations across Dir API.
    It handles gRPC communication and provides convenient methods for common operations.

    Example:
        >>> config = Config.load_from_env()
        >>> client = Client(config)
        >>> # Use client for operations...

    """

    def __init__(self, config: Config | None = None) -> None:
        """Initialize the client with the given configuration.

        Args:
            config: Optional client configuration. If None, loads from environment
                   variables using Config.load_from_env().

        Raises:
            grpc.RpcError: If unable to establish connection to the server
            ValueError: If configuration is invalid

        """
        # Load config if unset
        if config is None:
            config = Config.load_from_env()
        self.config = config

        # Create gRPC channel
        channel = self.__create_grpc_channel()

        # Initialize service clients
        self.store_client = store_v1.StoreServiceStub(channel)
        self.routing_client = routing_v1.RoutingServiceStub(channel)
        self.publication_client = routing_v1.PublicationServiceStub(channel)
        self.search_client = search_v1.SearchServiceStub(channel)
        self.sign_client = sign_v1.SignServiceStub(channel)
        self.sync_client = store_v1.SyncServiceStub(channel)
        self.event_client = events_v1.EventServiceStub(channel)

    def __create_grpc_channel(self) -> grpc.Channel:
        # Handle different authentication modes
        if self.config.auth_mode == "":
            return grpc.insecure_channel(self.config.server_address)
        elif self.config.auth_mode == "jwt":
            return self.__create_jwt_channel()
        elif self.config.auth_mode == "x509":
            return self.__create_x509_channel()
        elif self.config.auth_mode == "tls":
            return self.__create_tls_channel()
        else:
            msg = f"Unsupported auth mode: {self.config.auth_mode}"
            raise ValueError(msg)

    def __create_x509_channel(self) -> grpc.Channel:
        """Create a secure gRPC channel using SPIFFE X.509."""
        if self.config.spiffe_socket_path == "":
            msg = "SPIFFE socket path is required for X.509 authentication"
            raise ValueError(msg)

        # Create secure gRPC channel using SPIFFE X.509
        workload_client = WorkloadApiClient(socket_path=self.config.spiffe_socket_path)
        x509_src = X509Source(
            workload_api_client=workload_client,
            socket_path=self.config.spiffe_socket_path,
            timeout_in_seconds=60,
        )

        root_ca = b""
        for b in x509_src.bundles:
            for a in b.x509_authorities:
                root_ca += a.public_bytes(encoding=serialization.Encoding.PEM)

        private_key = x509_src.svid.private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        )

        public_leaf = x509_src.svid.leaf.public_bytes(
            encoding=serialization.Encoding.PEM
        )

        credentials = grpc.ssl_channel_credentials(
            root_certificates=root_ca,
            private_key=private_key,
            certificate_chain=public_leaf,
        )

        channel = grpc.secure_channel(
            target=self.config.server_address,
            credentials=credentials,
        )

        return channel

    def __create_jwt_channel(self) -> grpc.Channel:
        """Create a gRPC channel with JWT authentication."""
        if self.config.spiffe_socket_path == "":
            msg = "SPIFFE socket path is required for JWT authentication"
            raise ValueError(msg)

        if self.config.jwt_audience == "":
            msg = "JWT audience is required for JWT authentication"
            raise ValueError(msg)

        # Create X509Source to get the SPIFFE bundle for TLS verification
        # In JWT mode, the server presents its X.509-SVID via TLS for transport security
        # The X509Source will handle fetching the bundle from the Workload API
        workload_client = WorkloadApiClient(socket_path=self.config.spiffe_socket_path)
        x509_source = X509Source(
            workload_api_client=workload_client,
            socket_path=self.config.spiffe_socket_path,
            timeout_in_seconds=60,
        )
        
        # Extract the CA certificates from all bundles
        root_ca = b""
        for bundle in x509_source.bundles:
            for authority in bundle.x509_authorities:
                root_ca += authority.public_bytes(encoding=serialization.Encoding.PEM)
        
        if not root_ca:
            msg = "Failed to fetch X.509 bundle from SPIRE: no bundles returned"
            raise RuntimeError(msg)

        # Create JWT interceptor
        jwt_interceptor = JWTAuthInterceptor(
            socket_path=self.config.spiffe_socket_path,
            audience=self.config.jwt_audience
        )

        # Create secure channel with JWT interceptor and TLS using SPIFFE bundle
        # For JWT mode: Server presents X.509-SVID via TLS, clients authenticate with JWT-SVID
        credentials = grpc.ssl_channel_credentials(root_certificates=root_ca)
        channel = grpc.secure_channel(
            target=self.config.server_address,
            credentials=credentials,
        )
        channel = grpc.intercept_channel(channel, jwt_interceptor)

        # Close the X509Source since we only needed it to get the bundle
        x509_source.close()

        return channel
    
    def __create_tls_channel(self) -> grpc.Channel:
        if not self.config.tls_ca_file:
            msg = "TLS CA file is required for TLS authentication"
            raise ValueError(msg)
        if not self.config.tls_cert_file:
            msg = "TLS certificate file is required for TLS authentication"
            raise ValueError(msg)
        if not self.config.tls_key_file:
            msg = "TLS key file is required for TLS authentication"
            raise ValueError(msg)

        try:
            with open(self.config.tls_ca_file, "rb") as f:
                root_ca = f.read()
            with open(self.config.tls_cert_file, "rb") as f:
                cert_chain = f.read()
            with open(self.config.tls_key_file, "rb") as f:
                private_key = f.read()
        except OSError as e:
            msg = f"Failed to read TLS files: {e}"
            raise RuntimeError(msg) from e

        credentials = grpc.ssl_channel_credentials(
            root_certificates=root_ca,
            private_key=private_key,
            certificate_chain=cert_chain,
        )

        channel = grpc.secure_channel(
            target=self.config.server_address,
            credentials=credentials,
        )

        return channel

    def publish(
        self,
        req: routing_v1.PublishRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> None:
        """Publish objects to the Routing API matching the specified criteria.

        Makes the specified objects available for discovery and retrieval by other
        clients in the network. The objects must already exist in the store before
        they can be published.

        Args:
            req: Publish request containing the query for the objects to publish
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the object is not found or cannot be published

        Example:
            >>> ref = routing_v1.RecordRef(cid="QmExample123")
            >>> req = routing_v1.PublishRequest(record_refs=[ref])
            >>> client.publish(req)

        """
        try:
            self.routing_client.Publish(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during publish: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during publish: %s", e)
            msg = f"Failed to publish object: {e}"
            raise RuntimeError(msg) from e

    def list(
        self,
        req: routing_v1.ListRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> list[routing_v1.ListResponse]:
        """List objects from the Routing API matching the specified criteria.

        Returns a list of objects that match the filtering and
        query criteria specified in the request.

        Args:
            req: List request specifying filtering criteria, pagination, etc.
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[routing_v1.ListResponse]: List of items matching the criteria

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the list operation fails

        Example:
            >>> req = routing_v1.ListRequest(limit=10)
            >>> responses = client.list(req)
            >>> for response in responses:
            ...     print(f"Found object: {response.cid}")

        """
        results: list[routing_v1.ListResponse] = []

        try:
            stream = self.routing_client.List(req, metadata=metadata)
            results.extend(stream)
        except grpc.RpcError as e:
            logger.exception("gRPC error during list: %s", e)
            raise
        except Exception as e:
            logger.exception("Error receiving objects: %s", e)
            msg = f"Failed to list objects: {e}"
            raise RuntimeError(msg) from e

        return results

    def search(
        self,
        req: search_v1.SearchRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[routing_v1.SearchResponse]:
        """Search objects from the Store API matching the specified queries.

        Performs a search across the storage using the provided search queries
        and returns a list of matching results. Supports various
        search types including text, semantic, and structured queries.

        Args:
            req: Search request containing queries, filters, and search options
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[routing_v1.SearchResponse]: List of search results matching the queries

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the search operation fails

        Example:
            >>> req = search_v1.SearchRequest(query="python AI agent")
            >>> responses = client.search(req)
            >>> for response in responses:
            ...     print(f"Found: {response.record.name}")

        """
        results: list[routing_v1.SearchResponse] = []

        try:
            stream = self.search_client.Search(req, metadata=metadata)
            results.extend(stream)
        except grpc.RpcError as e:
            logger.exception("gRPC error during search: %s", e)
            raise
        except Exception as e:
            logger.exception("Error receiving search results: %s", e)
            msg = f"Failed to search objects: {e}"
            raise RuntimeError(msg) from e

        return results

    def unpublish(
        self,
        req: routing_v1.UnpublishRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> None:
        """Unpublish objects from the Routing API matching the specified criteria.

        Removes the specified objects from the public network, making them no
        longer discoverable by other clients. The objects remain in the local
        store but are not available for network discovery.

        Args:
            req: Unpublish request containing the query for the objects to unpublish
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the objects cannot be unpublished

        Example:
            >>> ref = routing_v1.RecordRef(cid="QmExample123")
            >>> req = routing_v1.UnpublishRequest(record_refs=[ref])
            >>> client.unpublish(req)

        """
        try:
            self.routing_client.Unpublish(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during unpublish: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during unpublish: %s", e)
            msg = f"Failed to unpublish object: {e}"
            raise RuntimeError(msg) from e

    def push(
        self,
        records: builtins.list[core_v1.Record],
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[core_v1.RecordRef]:
        """Push records to the Store API.

        Uploads one or more records to the content store, making them available
        for retrieval and reference. Each record is assigned a unique content
        identifier (CID) based on its content hash.

        Args:
            records: List of Record objects to push to the store
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[core_v1.RecordRef]: List of objects containing the CIDs of the pushed records

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the push operation fails

        Example:
            >>> records = [create_record("example")]
            >>> refs = client.push(records)
            >>> print(f"Pushed with CID: {refs[0].cid}")

        """
        results: list[core_v1.RecordRef] = []

        try:
            response = self.store_client.Push(iter(records), metadata=metadata)
            results.extend(response)
        except grpc.RpcError as e:
            logger.exception("gRPC error during push: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during push: %s", e)
            msg = f"Failed to push object: {e}"
            raise RuntimeError(msg) from e

        return results

    def push_referrer(
        self,
        req: builtins.list[store_v1.PushReferrerRequest],
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[store_v1.PushReferrerResponse]:
        """Push records with referrer metadata to the Store API.

        Uploads records along with optional artifacts and referrer information.
        This is useful for pushing complex objects that include additional
        metadata or associated artifacts.

        Args:
            req: List of PushReferrerRequest objects containing records and
                 optional artifacts
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[store_v1.PushReferrerResponse]: List of objects containing the details of pushed artifacts

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the push operation fails

        Example:
            >>> requests = [store_v1.PushReferrerRequest(record=record)]
            >>> responses = client.push_referrer(requests)

        """
        results: list[store_v1.PushReferrerResponse] = []

        try:
            response = self.store_client.PushReferrer(iter(req), metadata=metadata)
            results.extend(response)
        except grpc.RpcError as e:
            logger.exception("gRPC error during push_referrer: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during push_referrer: %s", e)
            msg = f"Failed to push object: {e}"
            raise RuntimeError(msg) from e

        return results

    def pull(
        self,
        refs: builtins.list[core_v1.RecordRef],
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[core_v1.Record]:
        """Pull records from the Store API by their references.

        Retrieves one or more records from the content store using their
        content identifiers (CIDs).

        Args:
            refs: List of RecordRef objects containing the CIDs to retrieve
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[core_v1.Record]: List of record objects retrieved from the store

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the pull operation fails

        Example:
            >>> refs = [core_v1.RecordRef(cid="QmExample123")]
            >>> records = client.pull(refs)
            >>> for record in records:
            ...     print(f"Retrieved record: {record}")

        """
        results: list[core_v1.Record] = []

        try:
            response = self.store_client.Pull(iter(refs), metadata=metadata)
            results.extend(response)
        except grpc.RpcError as e:
            logger.exception("gRPC error during pull: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during pull: %s", e)
            msg = f"Failed to pull object: {e}"
            raise RuntimeError(msg) from e

        return results

    def pull_referrer(
        self,
        req: builtins.list[store_v1.PullReferrerRequest],
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[store_v1.PullReferrerResponse]:
        """Pull records with referrer metadata from the Store API.

        Retrieves records along with their associated artifacts and referrer
        information. This provides access to complex objects that include
        additional metadata or associated artifacts.

        Args:
            req: List of PullReferrerRequest objects containing records and
                 optional artifacts for pull operations
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[store_v1.PullReferrerResponse]: List of objects containing the retrieved records

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the pull operation fails

        Example:
            >>> requests = [store_v1.PullReferrerRequest(ref=ref)]
            >>> responses = client.pull_referrer(requests)
            >>> for response in responses:
            ...     print(f"Retrieved: {response}")

        """
        results: list[store_v1.PullReferrerResponse] = []

        try:
            response = self.store_client.PullReferrer(iter(req), metadata=metadata)
            results.extend(response)
        except grpc.RpcError as e:
            logger.exception("gRPC error during pull_referrer: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during pull_referrer: %s", e)
            msg = f"Failed to pull referrer object: {e}"
            raise RuntimeError(msg) from e

        return results

    def lookup(
        self,
        refs: builtins.list[core_v1.RecordRef],
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[core_v1.RecordMeta]:
        """Look up metadata for records in the Store API.

        Retrieves metadata information for one or more records without
        downloading the full record content. This is useful for checking
        if records exist and getting basic information about them.

        Args:
            refs: List of RecordRef objects containing the CIDs to look up
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List[core_v1.RecordMeta]: List of objects containing metadata for the records

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the lookup operation fails

        Example:
            >>> refs = [core_v1.RecordRef(cid="QmExample123")]
            >>> metadatas = client.lookup(refs)
            >>> for meta in metadatas:
            ...     print(f"Record size: {meta.size}")

        """
        results: list[core_v1.RecordMeta] = []

        try:
            response = self.store_client.Lookup(iter(refs), metadata=metadata)
            results.extend(response)
        except grpc.RpcError as e:
            logger.exception("gRPC error during lookup: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during lookup: %s", e)
            msg = f"Failed to lookup object: {e}"
            raise RuntimeError(msg) from e

        return results

    def delete(
        self,
        refs: builtins.list[core_v1.RecordRef],
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> None:
        """Delete records from the Store API.

        Permanently removes one or more records from the content store using
        their content identifiers (CIDs). This operation cannot be undone.

        Args:
            refs: List of RecordRef objects containing the CIDs to delete
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the delete operation fails

        Example:
            >>> refs = [core_v1.RecordRef(cid="QmExample123")]
            >>> client.delete(refs)

        """
        try:
            self.store_client.Delete(iter(refs), metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during delete: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during delete: %s", e)
            msg = f"Failed to delete object: {e}"
            raise RuntimeError(msg) from e

    def create_sync(
        self,
        req: store_v1.CreateSyncRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> store_v1.CreateSyncResponse:
        """Create a new synchronization configuration.

        Creates a new sync configuration that defines how data should be
        synchronized between different Directory servers. This allows for
        automated data replication and consistency across multiple locations.

        Args:
            req: CreateSyncRequest containing the sync configuration details
                 including source, target, and synchronization parameters
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            store_v1.CreateSyncResponse: Response containing the created sync details
                                       including the sync ID and configuration

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the sync creation fails

        Example:
            >>> req = store_v1.CreateSyncRequest()
            >>> response = client.create_sync(req)
            >>> print(f"Created sync with ID: {response.sync_id}")

        """
        try:
            response = self.sync_client.CreateSync(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during create_sync: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during create_sync: %s", e)
            msg = f"Failed to create sync: {e}"
            raise RuntimeError(msg) from e

        return response

    def list_syncs(
        self,
        req: store_v1.ListSyncsRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[store_v1.ListSyncsItem]:
        """List existing synchronization configurations.

        Retrieves a list of all sync configurations that have been created,
        with optional filtering and pagination support. This allows you to
        monitor and manage multiple synchronization processes.

        Args:
            req: ListSyncsRequest containing filtering criteria, pagination options,
                 and other query parameters
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            list[store_v1.ListSyncsItem]: List of sync configuration items with
                                         their details including ID, name, status,
                                         and configuration parameters

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the list operation fails

        Example:
            >>> req = store_v1.ListSyncsRequest(limit=10)
            >>> syncs = client.list_syncs(req)
            >>> for sync in syncs:
            ...     print(f"Sync: {sync}")

        """
        results: list[store_v1.ListSyncsItem] = []

        try:
            stream = self.sync_client.ListSyncs(req, metadata=metadata)
            results.extend(stream)
        except grpc.RpcError as e:
            logger.exception("gRPC error during list_syncs: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during list_syncs: %s", e)
            msg = f"Failed to list syncs: {e}"
            raise RuntimeError(msg) from e

        return results

    def get_sync(
        self,
        req: store_v1.GetSyncRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> store_v1.GetSyncResponse:
        """Retrieve detailed information about a specific synchronization configuration.

        Gets comprehensive details about a specific sync configuration including
        its current status, configuration parameters, performance metrics,
        and any recent errors or warnings.

        Args:
            req: GetSyncRequest containing the sync ID or identifier to retrieve
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            store_v1.GetSyncResponse: Detailed information about the sync configuration
                                    including status, metrics, configuration, and logs

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the get operation fails

        Example:
            >>> req = store_v1.GetSyncRequest(sync_id="sync-123")
            >>> response = client.get_sync(req)
            >>> print(f"Sync status: {response.status}")
            >>> print(f"Last update: {response.last_update_time}")

        """
        try:
            response = self.sync_client.GetSync(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during get_sync: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during get_sync: %s", e)
            msg = f"Failed to get sync: {e}"
            raise RuntimeError(msg) from e

        return response

    def delete_sync(
        self,
        req: store_v1.DeleteSyncRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> None:
        """Delete a synchronization configuration.

        Permanently removes a sync configuration and stops any ongoing
        synchronization processes. This operation cannot be undone and
        will halt all data synchronization for the specified configuration.

        Args:
            req: DeleteSyncRequest containing the sync ID or identifier to delete
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the delete operation fails

        Example:
            >>> req = store_v1.DeleteSyncRequest(sync_id="sync-123")
            >>> client.delete_sync(req)
            >>> print(f"Sync deleted")

        """
        try:
            self.sync_client.DeleteSync(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during delete_sync: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during delete_sync: %s", e)
            msg = f"Failed to delete sync: {e}"
            raise RuntimeError(msg) from e

    def listen(
        self,
        req: events_v1.ListenRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> grpc.UnaryStreamMultiCallable:
        """
        Listen establishes a streaming connection to receive events.
        Events are only delivered while the stream is active.
        On disconnect, missed events are not recoverable.

        Args:
            req: ListenRequest specifies filters for event subscription.
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            A grpc stream which can read and closed.

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the verification operation fails
        """

        try:
            stream = self.event_client.Listen(req, metadata=metadata)
        except grpc.RpcError as e:
            if e.code() != grpc.StatusCode.CANCELLED:
                logger.exception("gRPC error during listen: %s", e)
                raise
        except Exception as e:
            logger.exception("Unexpected error during listen: %s", e)
            msg = f"Failed to listen: {e}"
            raise RuntimeError(msg) from e

        return stream
    
    def create_publication(
        self,
        req: routing_v1.PublishRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> routing_v1.CreatePublicationResponse:
        """
        Create publication creates a new publication request that will be processed by the PublicationWorker.
        The publication request can specify either a query, a list of specific CIDs,
        or all records to be announced to the DHT.

        Args:
            req: PublishRequest specifies the record references and queries for publication.
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            CreatePublicationResponse returns the result of creating a publication request.
            This includes the publication ID and any relevant metadata.

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the publication operation create fails
        """
        try:
            response = self.publication_client.CreatePublication(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during create_publication: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during create_publication: %s", e)
            msg = f"Failed to create publication: {e}"
            raise RuntimeError(msg) from e

        return response

    def get_publication(
        self,
        req: routing_v1.GetPublicationRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> routing_v1.GetPublicationResponse:
        """
        GetPublication retrieves details of a specific publication request by its identifier.
        This includes the current status and any associated metadata.

        Args:
            req: GetPublicationRequest specifies which publication to retrieve by its identifier.
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            GetPublicationResponse contains the full details of a specific publication request.
            Includes status, progress information, and any error details if applicable.

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the publication get operation fails
        """
        try:
            response = self.publication_client.GetPublication(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during get_publication: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during get_publication: %s", e)
            msg = f"Failed to create publication: {e}"
            raise RuntimeError(msg) from e

        return response

    def list_publication(
        self,
        req: routing_v1.ListPublicationsRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> builtins.list[routing_v1.ListPublicationsItem]:
        """
        ListPublications returns a stream of all publication requests in the system.
        This allows monitoring of pending, processing, and completed publication requests.

        Args:
            req: ListPublicationsRequest contains optional filters for listing publication requests.
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            List of ListPublicationsItem represents a single publication request in the list response.
            Contains publication details including ID, status, and creation timestamp.

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the publication get operation fails
        """

        results: list[routing_v1.ListPublicationsItem] = []

        try:
            stream = self.publication_client.ListPublications(req, metadata=metadata)
            results.extend(stream)
        except grpc.RpcError as e:
            logger.exception("gRPC error during list_publication: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during list_publication: %s", e)
            msg = f"Failed to list publication: {e}"
            raise RuntimeError(msg) from e

        return results


    def verify(
        self,
        req: sign_v1.VerifyRequest,
        metadata: Sequence[tuple[str, str]] | None = None,
    ) -> sign_v1.VerifyResponse:
        """Verify a cryptographic signature on a record.

        Validates the cryptographic signature of a previously signed record
        to ensure its authenticity and integrity. This operation verifies
        that the record has not been tampered with since signing.

        Args:
            req: VerifyRequest containing the record reference and verification
                 parameters
            metadata: Optional gRPC metadata headers as sequence of key-value pairs

        Returns:
            VerifyResponse containing the verification result and details

        Raises:
            grpc.RpcError: If the gRPC call fails (includes InvalidArgument, NotFound, etc.)
            RuntimeError: If the verification operation fails

        Example:
            >>> req = sign_v1.VerifyRequest(
            ...     record_ref=core_v1.RecordRef(cid="QmExample123")
            ... )
            >>> response = client.verify(req)
            >>> print(f"Signature valid: {response.valid}")

        """
        try:
            response = self.sign_client.Verify(req, metadata=metadata)
        except grpc.RpcError as e:
            logger.exception("gRPC error during verify: %s", e)
            raise
        except Exception as e:
            logger.exception("Unexpected error during verify: %s", e)
            msg = f"Failed to verify the object: {e}"
            raise RuntimeError(msg) from e

        return response

    def sign(
        self,
        req: sign_v1.SignRequest,
        oidc_client_id: str | None = "sigstore",
    ) -> None:
        """Sign a record with a cryptographic signature.

        Creates a cryptographic signature for a record using either a private
        key or OIDC-based signing. The signing process uses the external dirctl
        command-line tool to perform the actual cryptographic operations.

        Args:
            req: SignRequest containing the record reference and signing provider
                 configuration. The provider can specify either key-based signing
                 (with a private key) or OIDC-based signing
            oidc_client_id: OIDC client identifier for OIDC-based signing.
                           Defaults to "sigstore"

        Raises:
            RuntimeError: If the signing operation fails

        Example:
            >>> req = sign_v1.SignRequest(
            ...     record_ref=core_v1.RecordRef(cid="QmExample123"),
            ...     provider=sign_v1.SignProvider(key=key_config)
            ... )
            >>> client.sign(req)
            >>> print(f"Signing completed!")

        """
        try:
            if len(req.provider.key.private_key) > 0:
                self._sign_with_key(req.record_ref, req.provider.key)
            else:
                self._sign_with_oidc(req.record_ref, req.provider.oidc, oidc_client_id)
        except RuntimeError as e:
            msg = f"Failed to sign the object: {e}"
            raise RuntimeError(msg) from e
        except Exception as e:
            logger.exception("Signing operation failed: %s", e)
            msg = f"Failed to sign the object: {e}"
            raise RuntimeError(msg) from e

    def _sign_with_key(
        self,
        record_ref: core_v1.RecordRef,
        key_signer: sign_v1.SignWithKey,
    ) -> None:
        """Sign a record using a private key.

        This private method handles key-based signing by writing the private key
        to a temporary file and executing the dirctl command with the key file.

        Args:
            req: SignRequest containing the record reference and key provider

        Raises:
            RuntimeError: If any other error occurs during signing

        """
        try:
            # Create temporary file for the private key
            with tempfile.NamedTemporaryFile(delete=False) as tmp_key_file:
                tmp_key_file.write(key_signer.private_key)
                tmp_key_file.flush()

                # Set up environment with password
                shell_env = os.environ.copy()
                shell_env["COSIGN_PASSWORD"] = key_signer.password.decode("utf-8")

                # Build and execute the signing command
                command = [
                    self.config.dirctl_path,
                    "sign",
                    record_ref.cid,
                    "--key",
                    tmp_key_file.name,
                ]

                if self.config.spiffe_socket_path != "":
                    command.extend(["--spiffe-socket-path", self.config.spiffe_socket_path])
                
                subprocess.run(
                    command,
                    check=True,
                    capture_output=True,
                    env=shell_env,
                    timeout=60,  # 1 minute timeout
                )

        except OSError as e:
            msg = f"Failed to write key file to disk: {e}"
            raise RuntimeError(msg) from e
        except subprocess.CalledProcessError as e:
            msg = f"dirctl signing failed with return code {e.returncode}: {e.stderr.decode('utf-8', errors='ignore')}"
            raise RuntimeError(msg) from e
        except subprocess.TimeoutExpired as e:
            msg = "dirctl signing timed out"
            raise RuntimeError(msg) from e
        except Exception as e:
            msg = f"Unexpected error during key-based signing: {e}"
            raise RuntimeError(msg) from e

    def _sign_with_oidc(
        self,
        record_ref: core_v1.RecordRef,
        oidc_signer: sign_v1.SignWithOIDC,
        oidc_client_id: str = "sigstore",
    ) -> None:
        """Sign a record using OIDC-based authentication.

        This private method handles OIDC-based signing by building the appropriate
        dirctl command with OIDC parameters and executing it.

        Args:
            req: SignRequest containing the record reference and OIDC provider
            oidc_client_id: OIDC client identifier for authentication

        Raises:
            RuntimeError: If any other error occurs during signing

        """
        try:
            shell_env = os.environ.copy()

            # Build base command
            command = [self.config.dirctl_path, "sign", record_ref.cid]

            # Add OIDC-specific parameters
            if oidc_signer.id_token:
                command.extend(["--oidc-token", oidc_signer.id_token])
            if oidc_signer.options.oidc_provider_url:
                command.extend(
                    [
                        "--oidc-provider-url",
                        oidc_signer.options.oidc_provider_url,
                    ]
                )
            if oidc_signer.options.fulcio_url:
                command.extend(["--fulcio-url", oidc_signer.options.fulcio_url])
            if oidc_signer.options.rekor_url:
                command.extend(["--rekor-url", oidc_signer.options.rekor_url])
            if oidc_signer.options.timestamp_url:
                command.extend(["--timestamp-url", oidc_signer.options.timestamp_url])

            # Add client ID
            command.extend(["--oidc-client-id", oidc_client_id])

            if self.config.spiffe_socket_path != "":
                command.extend(["--spiffe-socket-path", self.config.spiffe_socket_path])

            # Execute the signing command
            subprocess.run(
                command,
                check=True,
                capture_output=True,
                env=shell_env,
                timeout=60,  # 1 minute timeout
            )

        except subprocess.CalledProcessError as e:
            msg = f"dirctl signing failed with return code {e.returncode}: {e.stderr.decode('utf-8', errors='ignore')}"
            raise RuntimeError(msg) from e
        except subprocess.TimeoutExpired as e:
            msg = "dirctl signing timed out"
            raise RuntimeError(msg) from e
        except Exception as e:
            msg = f"Unexpected error during OIDC signing: {e}"
            raise RuntimeError(msg) from e

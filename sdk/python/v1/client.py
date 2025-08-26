# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
import logging
import os
import subprocess
import tempfile
from subprocess import CompletedProcess
from typing import Iterator, List, Optional, Tuple

import core.v1.record_pb2 as core_types
import grpc
import routing.v1.routing_service_pb2 as routing_types
import routing.v1.routing_service_pb2_grpc as routing_services
import search.v1.search_service_pb2 as search_types
import search.v1.search_service_pb2_grpc as search_services
import sign.v1.sign_service_pb2 as sign_types
import sign.v1.sign_service_pb2_grpc as sign_services
import store.v1.store_service_pb2 as store_types
import store.v1.store_service_pb2_grpc as store_services

CHUNK_SIZE = 4096  # 4KB

logger = logging.getLogger("client")


class Config:
    DEFAULT_ENV_PREFIX = "DIRECTORY_CLIENT"
    DEFAULT_SERVER_ADDRESS = "0.0.0.0:8888"

    DEFAULT_DIRCTL_PATH = "dirctl"

    def __init__(
        self,
        server_address: str = DEFAULT_SERVER_ADDRESS,
        dirctl_path: str = DEFAULT_DIRCTL_PATH,
    ):
        self.server_address = server_address
        self.dirctl_path = dirctl_path

    @staticmethod
    def load_from_env() -> "Config":
        """Load configuration from environment variables"""
        prefix = Config.DEFAULT_ENV_PREFIX
        server_address = os.environ.get(
            f"{prefix}_SERVER_ADDRESS", Config.DEFAULT_SERVER_ADDRESS
        )

        dirctl_path = os.environ.get("DIRCTL_PATH", Config.DEFAULT_DIRCTL_PATH)

        return Config(server_address=server_address, dirctl_path=dirctl_path)


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
        self.sign_client = sign_services.SignServiceStub(channel)

        self.dirctl_path = config.dirctl_path

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

    def sign(
        self,
        req: sign_types.SignRequest,
        oidc_client_id: Optional[str] = "sigstore",
    ) -> CompletedProcess[bytes]:
        """Sign a record with a provider

        Args:
            req: Sign request contains the record reference and provider
            oidc_client_id: OIDC client id for OIDC signing
        Raises:
            Exception: If sign operation fails
        """

        try:
            if len(req.provider.key.private_key) > 0:
                result = self.__sign_with_key__(req)
            else:
                result = self.__sign_with_oidc__(req, oidc_client_id=oidc_client_id)

        except Exception as e:
            raise Exception(f"Failed to sign the object: {e}")

        return result

    def verify(
        self,
        req: sign_types.VerifyRequest,
        metadata: Optional[List[Tuple[str, str]]] = None,
    ) -> sign_types.VerifyResponse:
        """Verify a signed record

        Args:
            req: Verify request contains the record reference
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If verify operation fails
        """

        try:
            response = self.sign_client.Verify(req, metadata=metadata)
        except Exception as e:
            raise Exception(f"Failed to verify the object: {e}")

        return response

    def __sign_with_key__(
        self,
        req: sign_types.SignRequest,
    ) -> CompletedProcess[bytes]:
        process = None

        try:
            key_signer = req.provider.key

            tmp_key_file = tempfile.NamedTemporaryFile()

            with open(tmp_key_file.name, "wb") as key_file:
                key_file.write(key_signer.private_key)

            shell_env = os.environ.copy()
            shell_env["COSIGN_PASSWORD"] = key_signer.password.decode("utf8")

            command = (
                self.dirctl_path,
                "sign",
                req.record_ref.cid,
                "--key",
                tmp_key_file.name,
            )
            process = subprocess.run(
                command, check=True, capture_output=True, env=shell_env
            )

        except OSError as e:
            raise Exception(f"Failed to write file to disk: {e}")
        except subprocess.CalledProcessError as e:
            raise Exception(f"dirctl command failed: {e}")
        except Exception as e:
            raise Exception(f"Unknown error: {e}")

        return process

    def __sign_with_oidc__(
        self,
        req: sign_types.SignRequest,
        oidc_client_id: str = "sigstore",
    ) -> CompletedProcess[bytes]:
        oidc_signer = req.provider.oidc

        try:
            shell_env = os.environ.copy()

            command = (self.dirctl_path, "sign", f"{req.record_ref.cid}")
            if oidc_signer.id_token != "":
                command = (*command, "--oidc-token", f"{oidc_signer.id_token}")
            if oidc_signer.options.oidc_provider_url != "":
                command = (
                    *command,
                    "--oidc-provider-url",
                    f"{oidc_signer.options.oidc_provider_url}",
                )
            if oidc_signer.options.fulcio_url != "":
                command = (
                    *command,
                    "--fulcio-url",
                    f"{oidc_signer.options.fulcio_url}",
                )
            if oidc_signer.options.rekor_url != "":
                command = (*command, "--rekor-url", f"{oidc_signer.options.rekor_url}")
            if oidc_signer.options.timestamp_url != "":
                command = (
                    *command,
                    "--timestamp-url",
                    f"{oidc_signer.options.timestamp_url}",
                )

            result = subprocess.run(
                (*command, "--oidc-client-id", f"{oidc_client_id}"),
                check=True,
                capture_output=True,
                env=shell_env,
            )

        except subprocess.CalledProcessError as e:
            raise Exception(f"dirctl command failed: {e}")
        except Exception as e:
            raise Exception(f"Unknown error: {e}")

        return result

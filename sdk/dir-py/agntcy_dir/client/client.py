# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import builtins
import logging
import os
import subprocess
import tempfile
from collections.abc import Iterator
from subprocess import CompletedProcess

import grpc

from agntcy_dir.client.config import Config
from agntcy_dir.models import *

logger = logging.getLogger("client")


class Client:
    def __init__(self, config: Config | None = None) -> "Client":
        """Initialize the client with the given configuration.

        Args:
            config: Optional client configuration. If unset, loaded from env.

        Returns:
            A new Client instance

        """
        # Load config if unset
        if config is None:
            config = Config.load_from_env()
        self.config = config

        # Create gRPC channel
        channel = grpc.insecure_channel(config.server_address)

        # Initialize service clients
        self.store_client = store_v1.StoreServiceStub(channel)
        self.routing_client = routing_v1.RoutingServiceStub(channel)
        self.search_client = search_v1.SearchServiceStub(channel)
        self.sign_client = sign_v1.SignServiceStub(channel)

    def publish(
        self,
        req: routing_v1.PublishRequest,
        metadata: list[tuple[str, str]] | None = None,
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
            msg = f"Failed to publish object: {e}"
            raise Exception(msg)

    def list(
        self,
        req: routing_v1.ListRequest,
        metadata: list[tuple[str, str]] | None = None,
    ) -> Iterator[routing_v1.ListResponse]:
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
            yield from stream
        except Exception as e:
            logger.exception("Error receiving objects: %s", e)
            msg = f"Failed to list objects: {e}"
            raise Exception(msg)

    def search(
        self,
        req: search_v1.SearchRequest,
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> Iterator[routing_v1.SearchResponse]:
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
            yield from stream
        except Exception as e:
            logger.exception("Error receiving objects: %s", e)
            msg = f"Failed to search objects: {e}"
            raise Exception(msg)

    def unpublish(
        self,
        req: routing_v1.UnpublishRequest,
        metadata: builtins.list[tuple[str, str]] | None = None,
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
            msg = f"Failed to unpublish object: {e}"
            raise Exception(msg)

    def push(
        self,
        records: builtins.list[core_v1.Record],
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> builtins.list[core_v1.RecordRef]:
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
            msg = f"Failed to push object: {e}"
            raise Exception(msg)

        return references

    def push_referrer(
        self,
        req: builtins.list[store_v1.PushReferrerRequest],
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> builtins.list[store_v1.PushReferrerResponse]:
        """Push objects to the store.

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
            msg = f"Failed to push object: {e}"
            raise Exception(msg)

        return responses

    def pull(
        self,
        refs: builtins.list[core_v1.RecordRef],
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> builtins.list[core_v1.Record]:
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

            records.extend(r for r in response if r is not None)

        except Exception as e:
            msg = f"Failed to pull object: {e}"
            raise Exception(msg)

        return records

    def pull_referrer(
        self,
        req: builtins.list[store_v1.PullReferrerRequest],
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> builtins.list[store_v1.PullReferrerResponse]:
        """Pull objects from the store.

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
            msg = f"Failed to push object: {e}"
            raise Exception(msg)

        return responses

    def lookup(
        self,
        refs: builtins.list[core_v1.RecordRef],
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> builtins.list[core_v1.RecordMeta]:
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

            metadatas.extend(r for r in response if r is not None)

        except Exception as e:
            msg = f"Failed to pull object: {e}"
            raise Exception(msg)

        return metadatas

    def delete(
        self,
        refs: builtins.list[core_v1.RecordRef],
        metadata: builtins.list[tuple[str, str]] | None = None,
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
            msg = f"Failed to pull object: {e}"
            raise Exception(msg)

    def sign(
        self,
        req: sign_v1.SignRequest,
        oidc_client_id: str | None = "sigstore",
    ) -> CompletedProcess[bytes]:
        """Sign a record with a provider.

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
            msg = f"Failed to sign the object: {e}"
            raise Exception(msg)

        return result

    def verify(
        self,
        req: sign_v1.VerifyRequest,
        metadata: builtins.list[tuple[str, str]] | None = None,
    ) -> sign_v1.VerifyResponse:
        """Verify a signed record.

        Args:
            req: Verify request contains the record reference
            metadata: Optional metadata for the gRPC call

        Raises:
            Exception: If verify operation fails

        """
        try:
            response = self.sign_client.Verify(req, metadata=metadata)
        except Exception as e:
            msg = f"Failed to verify the object: {e}"
            raise Exception(msg)

        return response

    def __sign_with_key__(
        self,
        req: sign_v1.SignRequest,
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
                self.config.dirctl_path,
                "sign",
                req.record_ref.cid,
                "--key",
                tmp_key_file.name,
            )
            process = subprocess.run(
                command, check=True, capture_output=True, env=shell_env,
            )

        except OSError as e:
            msg = f"Failed to write file to disk: {e}"
            raise Exception(msg)
        except subprocess.CalledProcessError as e:
            msg = f"dirctl command failed: {e}"
            raise Exception(msg)
        except Exception as e:
            msg = f"Unknown error: {e}"
            raise Exception(msg)

        return process

    def __sign_with_oidc__(
        self,
        req: sign_v1.SignRequest,
        oidc_client_id: str = "sigstore",
    ) -> CompletedProcess[bytes]:
        oidc_signer = req.provider.oidc

        try:
            shell_env = os.environ.copy()

            command = (self.config.dirctl_path, "sign", f"{req.record_ref.cid}")
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
            msg = f"dirctl command failed: {e}"
            raise Exception(msg)
        except Exception as e:
            msg = f"Unknown error: {e}"
            raise Exception(msg)

        return result

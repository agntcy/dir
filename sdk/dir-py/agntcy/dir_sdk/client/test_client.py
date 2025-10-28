# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os
import pathlib
import subprocess
import time
import unittest
import uuid

from agntcy.dir_sdk.client import Client
from agntcy.dir_sdk.models import *


class TestClient(unittest.TestCase):
    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)

        # Verify that `DIRCTL_PATH` is set in the environment
        assert os.environ.get("DIRCTL_PATH") is not None

        # Initialize the client
        self.client = Client()

    def test_push(self) -> None:
        records = self.gen_records(2, "push")
        record_refs = self.client.push(records=records)

        assert record_refs is not None
        assert isinstance(record_refs, list)
        assert len(record_refs) == 2

        for ref in record_refs:
            assert isinstance(ref, core_v1.RecordRef)
            assert len(ref.cid) == 59

    def test_pull(self) -> None:
        records = self.gen_records(2, "pull")
        record_refs = self.client.push(records=records)
        pulled_records = self.client.pull(refs=record_refs)

        assert pulled_records is not None
        assert isinstance(pulled_records, list)
        assert len(pulled_records) == 2

        for index, record in enumerate(pulled_records):
            assert isinstance(record, core_v1.Record)
            assert records[index] == record

    def test_lookup(self) -> None:
        records = self.gen_records(2, "lookup")
        record_refs = self.client.push(records=records)
        metadatas = self.client.lookup(record_refs)

        assert metadatas is not None
        assert isinstance(metadatas, list)
        assert len(metadatas) == 2

        for metadata in metadatas:
            assert isinstance(metadata, core_v1.RecordMeta)

    def test_publish(self) -> None:
        records = self.gen_records(1, "publish")
        record_refs = self.client.push(records=records)
        publish_request = routing_v1.PublishRequest(
            record_refs=routing_v1.RecordRefs(refs=record_refs),
        )

        try:
            self.client.publish(publish_request)
        except Exception as e:
            assert e is None

    def test_list(self) -> None:
        records = self.gen_records(1, "list")
        record_refs = self.client.push(records=records)
        self.client.publish(routing_v1.PublishRequest(
            record_refs=routing_v1.RecordRefs(refs=record_refs),
        ))

        # Sleep to allow the publication to be indexed
        time.sleep(5)

        # Query for records in the domain
        list_query = routing_v1.RecordQuery(
            type=routing_v1.RECORD_QUERY_TYPE_DOMAIN,
            value="technology/networking",
        )

        list_request = routing_v1.ListRequest(queries=[list_query])
        objects = list(self.client.list(list_request))

        assert objects is not None
        assert len(objects) != 0

        for o in objects:
            assert isinstance(o, routing_v1.ListResponse)

    def test_search(self) -> None:
        records = self.gen_records(1, "search")
        _ = self.client.push(records=records)

        search_query = search_v1.RecordQuery(
            type=search_v1.RECORD_QUERY_TYPE_SKILL_ID,
            value="10201",
        )

        search_request = search_v1.SearchRequest(queries=[search_query], limit=2)

        objects = list(self.client.search(search_request))

        assert objects is not None
        assert len(objects) > 0

        for o in objects:
            assert isinstance(o, search_v1.SearchResponse)

    def test_unpublish(self) -> None:
        records = self.gen_records(1, "unpublish")
        record_refs = self.client.push(records=records)

        publish_record_refs = routing_v1.RecordRefs(refs=record_refs)
        _ = routing_v1.PublishRequest(record_refs=publish_record_refs)
        unpublish_request = routing_v1.UnpublishRequest(record_refs=publish_record_refs)

        try:
            self.client.unpublish(unpublish_request)
        except Exception as e:
            assert e is None

    def test_delete(self) -> None:
        records = self.gen_records(1, "delete")
        record_refs = self.client.push(records=records)
        try:
            self.client.delete(record_refs)
        except Exception as e:
            assert e is None

    def test_push_referrer(self) -> None:
        records = self.gen_records(2, "push_referrer")
        record_refs = self.client.push(records=records)

        try:
            request = [
                store_v1.PushReferrerRequest(
                    record_ref=record_refs[0],
                    referrer=core_v1.RecordReferrer(
                        type=sign_v1.Signature.DESCRIPTOR.full_name,
                        data={
                            "signature": "dGVzdC1zaWduYXR1cmU=",  # base64 encoded "test-signature"
                            "annotations": {
                                "payload": "test-payload-data"
                            }
                        }
                    ),
                ),
                store_v1.PushReferrerRequest(
                    record_ref=record_refs[1],
                    referrer=core_v1.RecordReferrer(
                        type=sign_v1.Signature.DESCRIPTOR.full_name,
                        data={
                            "signature": "dGVzdC1zaWduYXR1cmU=",  # base64 encoded "test-signature"
                            "annotations": {
                                "payload": "test-payload-data"
                            }
                        }
                    ),
                ),
            ]

            response = self.client.push_referrer(req=request)

            assert response is not None
            assert len(response) == 2

            for r in response:
                assert isinstance(r, store_v1.PushReferrerResponse)

        except Exception as e:
            assert e is None

    def test_pull_referrer(self) -> None:
        records = self.gen_records(2, "pull_referrer")
        record_refs = self.client.push(records=records)

        # Push referrers to these records
        request = [
            store_v1.PushReferrerRequest(
                record_ref=record_refs[0],
                referrer=core_v1.RecordReferrer(
                    type=sign_v1.Signature.DESCRIPTOR.full_name,
                    data={
                        "signature": "dGVzdC1zaWduYXR1cmU=",  # base64 encoded "test-signature"
                        "annotations": {
                            "payload": "test-payload-data"
                        }
                    }
                ),
            ),
            store_v1.PushReferrerRequest(
                record_ref=record_refs[1],
                referrer=core_v1.RecordReferrer(
                    type=sign_v1.Signature.DESCRIPTOR.full_name,
                    data={
                        "signature": "dGVzdC1zaWduYXR1cmU=",  # base64 encoded "test-signature"
                        "annotations": {
                            "payload": "test-payload-data"
                        }
                    }
                ),
            ),
        ]
        response = self.client.push_referrer(req=request)
        assert response is not None
        assert len(response) == 2
        for r in response:
            assert isinstance(r, store_v1.PushReferrerResponse)

        try:
            request = [
                store_v1.PullReferrerRequest(
                    record_ref=record_refs[0],
                    referrer_type=sign_v1.Signature.DESCRIPTOR.full_name,
                ),
                store_v1.PullReferrerRequest(
                    record_ref=record_refs[1],
                    referrer_type=sign_v1.Signature.DESCRIPTOR.full_name,
                ),
            ]

            response = self.client.pull_referrer(req=request)

            assert response is not None
            assert len(response) == 2

            for r in response:
                assert isinstance(r, store_v1.PullReferrerResponse)
        except Exception as e:
            assert "pull referrer not implemented" in str(
                e,
            )  # Delete when the service implemented

            # self.assertIsNone(e) # Uncomment when the service implemented

    def test_sign_and_verify(self) -> None:
        records = self.gen_records(2, "sign_verify")
        record_refs = self.client.push(records=records)

        # Remove existing cosign keys if any
        try:
            pathlib.Path("cosign.key").unlink()
            pathlib.Path("cosign.pub").unlink()
        except FileNotFoundError:
            pass # Clean state found

        # Prepare cosign key pair
        key_password = "testing-key"

        # Set environment variable for cosign password
        shell_env = os.environ.copy()
        shell_env["COSIGN_PASSWORD"] = key_password

        # Generate a key pair using cosign
        cosign_path = os.getenv("COSIGN_PATH", "cosign")
        command = (cosign_path, "generate-key-pair")
        subprocess.run(command, check=True, capture_output=True, env=shell_env)

        with open("cosign.key", "rb") as reader:
            key_file = reader.read()

        # Prepare Key signing request
        key_provider = sign_v1.SignWithKey(
            private_key=key_file,
            password=key_password.encode("utf-8"),
        )

        request_key_provider = sign_v1.SignRequestProvider(key=key_provider)
        key_request = sign_v1.SignRequest(
            record_ref=record_refs[0],
            provider=request_key_provider,
        )

        # Prepare OIDC signing request
        token = shell_env.get("OIDC_TOKEN", "")
        provider_url = shell_env.get("OIDC_PROVIDER_URL", "")
        client_id = shell_env.get("OIDC_CLIENT_ID", "sigstore")

        oidc_options = sign_v1.SignWithOIDC.SignOpts(oidc_provider_url=provider_url)
        oidc_provider = sign_v1.SignWithOIDC(id_token=token, options=oidc_options)
        request_oidc_provider = sign_v1.SignRequestProvider(oidc=oidc_provider)
        oidc_request = sign_v1.SignRequest(
            record_ref=record_refs[1],
            provider=request_oidc_provider,
        )

        try:
            # Sign and verify using Key signing
            self.client.sign(key_request)

            # Sign and verify using OIDC signing if set
            if shell_env.get("OIDC_TOKEN", "") != "" and shell_env.get("OIDC_PROVIDER_URL", "") != "":
                self.client.sign(oidc_request, client_id)
            else:
                record_refs.pop() # NOTE: Drop the unsigned record if no OIDC tested

            for ref in record_refs:
                response = self.client.verify(sign_v1.VerifyRequest(record_ref=ref))

                assert response.success is True
                
        except Exception as e:
            assert e is None
        finally:
            pathlib.Path("cosign.key").unlink()
            pathlib.Path("cosign.pub").unlink()

        # Test invalid sign request
        invalid_request = sign_v1.SignRequest(
            record_ref=core_v1.RecordRef(cid="invalid-cid"),
            provider=request_key_provider,
        )
        try:
            self.client.sign(invalid_request)
        except RuntimeError as e:
            assert "Failed to sign the object" in str(e)

    def test_sync(self) -> None:
        try:
            create_request = store_v1.CreateSyncRequest(
                remote_directory_url=os.getenv(
                    "DIRECTORY_SERVER_PEER1_ADDRESS",
                    "0.0.0.0:8891",
                ),
            )
            create_response = self.client.create_sync(create_request)

            try:
                assert uuid.UUID(create_response.sync_id)
            except ValueError:
                msg = f"Not an UUID: {create_response.sync_id}"
                raise ValueError(msg)

            list_request = store_v1.ListSyncsRequest()
            list_response = self.client.list_syncs(list_request)

            for sync_item in list_response:
                try:
                    assert isinstance(sync_item, store_v1.ListSyncsItem)
                    assert uuid.UUID(sync_item.sync_id)
                except ValueError:
                    msg = f"Not an UUID: {sync_item.sync_id}"
                    raise ValueError(msg)

            get_request = store_v1.GetSyncRequest(sync_id=create_response.sync_id)
            get_response = self.client.get_sync(get_request)

            assert isinstance(get_response, store_v1.GetSyncResponse)
            assert get_response.sync_id == create_response.sync_id

            delete_request = store_v1.DeleteSyncRequest(sync_id=create_response.sync_id)
            self.client.delete_sync(delete_request)

        except Exception as e:
            assert e is None

    def gen_records(self, count: int, test_function_name: str) -> list[core_v1.Record]:
        """
        Generate test records with unique names.
        Schema: https://schema.oasf.outshift.com/0.7.0/objects/record
        """
        records: list[core_v1.Record] = [
            core_v1.Record(
                data={
                    "name": f"agntcy-{test_function_name}-{index}-{str(uuid.uuid4())[:8]}",
                    "version": "v3.0.0",
                    "schema_version": "0.7.0",
                    "description": "Research agent for Cisco's marketing strategy.",
                    "authors": ["Cisco Systems"],
                    "created_at": "2025-03-19T17:06:37Z",
                    "skills": [
                        {
                            "name": "natural_language_processing/natural_language_generation/text_completion",
                            "id": 10201
                        },
                        {
                            "name": "natural_language_processing/analytical_reasoning/problem_solving",
                            "id": 10702
                        }
                    ],
                    "locators": [
                        {
                            "type": "docker_image",
                            "url": "https://ghcr.io/agntcy/marketing-strategy"
                        }
                    ],
                    "domains": [
                        {
                            "name": "technology/networking",
                            "id": 103
                        }
                    ],
                    "modules": []
                }
            )
            for index in range(count)
        ]

        return records


if __name__ == "__main__":
    unittest.main()

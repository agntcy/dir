# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os
import pathlib
import subprocess
import unittest
import uuid

from agntcy_dir.client import Client
from agntcy_dir.models import *


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
        _ = self.client.push(records=records)
        list_query = routing_v1.RecordQuery(
            type=routing_v1.RECORD_QUERY_TYPE_LOCATOR,
            value="docker-image",
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
            value="1",
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
            example_signature = sign_v1.Signature()
            request = [
                store_v1.PushReferrerRequest(
                    record_ref=record_refs[0],
                    signature=example_signature,
                ),
                store_v1.PushReferrerRequest(
                    record_ref=record_refs[1],
                    signature=example_signature,
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

        try:
            request = [
                store_v1.PullReferrerRequest(
                    record_ref=record_refs[0],
                    pull_signature=False,
                ),
                store_v1.PullReferrerRequest(
                    record_ref=record_refs[1],
                    pull_signature=False,
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

        shell_env = os.environ.copy()

        key_password = "testing-key"
        shell_env["COSIGN_PASSWORD"] = key_password

        # Avoid interactive question about override
        try:
            pathlib.Path("cosign.key").unlink()
            pathlib.Path("cosign.pub").unlink()
        except FileNotFoundError:
            pass  # Clean state found

        cosign_path = os.getenv("COSIGN_PATH", "cosign")
        command = (cosign_path, "generate-key-pair")
        subprocess.run(command, check=True, capture_output=True, env=shell_env)

        with open("cosign.key", "rb") as reader:
            key_file = reader.read()

        key_provider = sign_v1.SignWithKey(
            private_key=key_file,
            password=key_password.encode("utf-8"),
        )

        token = shell_env.get("OIDC_TOKEN", "")
        provider_url = shell_env.get("OIDC_PROVIDER_URL", "")
        client_id = shell_env.get("OIDC_CLIENT_ID", "sigstore")

        oidc_options = sign_v1.SignWithOIDC.SignOpts(oidc_provider_url=provider_url)
        oidc_provider = sign_v1.SignWithOIDC(id_token=token, options=oidc_options)

        request_key_provider = sign_v1.SignRequestProvider(key=key_provider)
        request_oidc_provider = sign_v1.SignRequestProvider(oidc=oidc_provider)

        key_request = sign_v1.SignRequest(
            record_ref=record_refs[0],
            provider=request_key_provider,
        )
        oidc_request = sign_v1.SignRequest(
            record_ref=record_refs[1],
            provider=request_oidc_provider,
        )

        try:
            # Sign test
            self.client.sign(key_request)
            self.client.sign(oidc_request, client_id)

            # Verify test
            for ref in record_refs:
                request = sign_v1.VerifyRequest(record_ref=ref)
                response = self.client.verify(request)

                assert response.success is True
        except Exception as e:
            assert e is None
        finally:
            pathlib.Path("cosign.key").unlink()
            pathlib.Path("cosign.pub").unlink()

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
        records: list[core_v1.Record] = [
            core_v1.Record(
                v3=objects_v3.Record(
                    name=f"{test_function_name}-{index}",
                    version="v3",
                    schema_version="v0.5.0",
                    skills=[
                        objects_v3.Skill(
                            name="Natural Language Processing",
                            id=1,
                        ),
                    ],
                    locators=[
                        objects_v3.Locator(
                            type="docker-image",
                            url="127.0.0.1",
                        ),
                    ],
                    extensions=[
                        objects_v3.Extension(
                            name="runtime/prompt",
                            version="v1",
                        ),
                    ],
                    signature=objects_v3.Signature(),
                ),
            )
            for index in range(count)
        ]

        return records


if __name__ == "__main__":
    unittest.main()

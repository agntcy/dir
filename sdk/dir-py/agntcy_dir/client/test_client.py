# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os
import pathlib
import subprocess
import unittest

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
        example_records = self.init_records(2, "push", push=False)
        records_list = list[core_v1.Record](
            record for _, record in example_records.values()
        )

        references = self.client.push(records=records_list)

        assert references is not None
        assert isinstance(references, list)
        assert len(references) == 2

        for ref in references:
            assert isinstance(ref, core_v1.RecordRef)
            assert len(ref.cid) == 59

    def test_pull(self) -> None:
        example_records = self.init_records(2, "pull")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        pulled_records = self.client.pull(refs=record_refs_list)

        assert pulled_records is not None
        assert isinstance(pulled_records, list)
        assert len(pulled_records) == 2

        for record in pulled_records:
            assert isinstance(record, core_v1.Record)

    def test_lookup(self) -> None:
        example_records = self.init_records(2, "lookup")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        metadatas = self.client.lookup(record_refs_list)

        assert metadatas is not None
        assert isinstance(metadatas, list)
        assert len(metadatas) == 2

        for metadata in metadatas:
            assert isinstance(metadata, core_v1.RecordMeta)

    def test_publish(self) -> None:
        example_records = self.init_records(1, "publish")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        record_refs = routing_v1.RecordRefs(refs=record_refs_list)
        publish_request = routing_v1.PublishRequest(record_refs=record_refs)

        try:
            self.client.publish(publish_request)
        except Exception as e:
            assert e is None

    def test_list(self) -> None:
        _ = self.init_records(2, "list", publish=True)

        list_query = routing_v1.RecordQuery(
            type=routing_v1.RECORD_QUERY_TYPE_SKILL,
            value="/skills/Natural Language Processing/Text Completion",
        )

        list_request = routing_v1.ListRequest(queries=[list_query])
        objects = list(self.client.list(list_request))

        assert objects is not None
        assert len(objects) != 0

        for o in objects:
            assert isinstance(o, routing_v1.ListResponse)

    def test_search(self) -> None:
        _ = self.init_records(2, "search", publish=True)

        search_query = search_v1.RecordQuery(
            type=search_v1.RECORD_QUERY_TYPE_SKILL_ID, value="1",
        )

        search_request = search_v1.SearchRequest(queries=[search_query], limit=2)

        objects = list(self.client.search(search_request))

        assert objects is not None
        assert len(objects) != 0

        for o in objects:
            assert isinstance(o, search_v1.SearchResponse)

    def test_unpublish(self) -> None:
        example_records = self.init_records(1, "unpublish", publish=True)
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        record_refs = routing_v1.RecordRefs(refs=record_refs_list)
        unpublish_request = routing_v1.UnpublishRequest(record_refs=record_refs)

        try:
            self.client.unpublish(unpublish_request)
        except Exception as e:
            assert e is None

    def test_delete(self) -> None:
        example_records = self.init_records(2, "delete")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        try:
            self.client.delete(record_refs_list)
        except Exception as e:
            assert e is None

    def test_push_referrer(self) -> None:
        example_records = self.init_records(2, "push_referrer")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        try:
            example_signature = sign_v1.Signature()
            request = [
                store_v1.PushReferrerRequest(
                    record_ref=record_refs_list[0], signature=example_signature,
                ),
                store_v1.PushReferrerRequest(
                    record_ref=record_refs_list[1], signature=example_signature,
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
        example_records = self.init_records(2, "pull_referrer")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

        try:
            request = [
                store_v1.PullReferrerRequest(
                    record_ref=record_refs_list[0], pull_signature=False,
                ),
                store_v1.PullReferrerRequest(
                    record_ref=record_refs_list[1], pull_signature=False,
                ),
            ]

            response = self.client.pull_referrer(req=request)

            assert response is not None
            assert len(response) == 2

            for r in response:
                assert isinstance(r, store_v1.PullReferrerResponse)
        except Exception as e:
            assert "pull referrer not implemented" in str(e)  # Delete when the service implemented

            # self.assertIsNone(e) # Uncomment when the service implemented

    def test_sign_and_verify(self) -> None:
        example_records = self.init_records(2, "sign_and_verify")
        record_refs_list = list[core_v1.RecordRef](
            ref for ref, _ in example_records.values()
        )

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
            private_key=key_file, password=key_password.encode("utf-8"),
        )

        token = shell_env.get("OIDC_TOKEN", "")
        provider_url = shell_env.get("OIDC_PROVIDER_URL", "")
        client_id = shell_env.get("OIDC_CLIENT_ID", "sigstore")

        oidc_options = sign_v1.SignWithOIDC.SignOpts(oidc_provider_url=provider_url)
        oidc_provider = sign_v1.SignWithOIDC(id_token=token, options=oidc_options)

        request_key_provider = sign_v1.SignRequestProvider(key=key_provider)
        request_oidc_provider = sign_v1.SignRequestProvider(oidc=oidc_provider)

        key_request = sign_v1.SignRequest(
            record_ref=record_refs_list[0], provider=request_key_provider,
        )
        oidc_request = sign_v1.SignRequest(
            record_ref=record_refs_list[1], provider=request_oidc_provider,
        )

        try:
            # Sign test
            result = self.client.sign(key_request)
            assert result.stderr.decode("utf-8") == ""
            assert result.stdout.decode("utf-8") == "Record signed successfully"

            result = self.client.sign(oidc_request, client_id)
            assert result.stderr.decode("utf-8") == ""
            assert result.stdout.decode("utf-8") == "Record signed successfully"

            # Verify test
            for ref in record_refs_list:
                request = sign_v1.VerifyRequest(record_ref=ref)
                response = self.client.verify(request)

                assert response.success is True
        except Exception as e:
            print(e)
            assert e is None
        finally:
            pathlib.Path("cosign.key").unlink()
            pathlib.Path("cosign.pub").unlink()

    def init_records(self, count, test_function_name, push=True, publish=False):
        example_records = {}

        for index in range(count):
            generated_record = core_v1.Record(
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
                            type="ipv4",
                            url="127.0.0.1",
                        ),
                    ],
                    extensions=[
                        objects_v3.Extension(
                            name="schema.oasf.agntcy.org/domains/domain-1",
                            version="v1",
                        ),
                    ],
                    signature=objects_v3.Signature(),
                ),
            )

            example_records[index] = (None, generated_record)

        if push:
            records_list = list[core_v1.Record](
                record for _, record in example_records.values()
            )

            for index, record in enumerate(records_list):
                # Push only one at a time to make sure of the cid pairing
                references = self.client.push(records=[record])

                example_records[index] = (references[0], record)

            if publish:
                for record_ref in example_records.values():
                    record_refs = routing_v1.RecordRefs(refs=[record_ref])
                    req = routing_v1.PublishRequest(record_refs=record_refs)
                    self.client.publish(req=req)

        return example_records


if __name__ == "__main__":
    unittest.main()

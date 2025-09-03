import os
import subprocess
import unittest
import uuid

import core.v1.record_pb2 as core_record_pb2
import store.v1.sync_service_pb2 as sync_types
from objects.v3 import extension_pb2, locator_pb2, record_pb2, signature_pb2, skill_pb2
from routing.v1 import record_query_pb2 as record_query_type
from routing.v1 import routing_service_pb2 as routingv1
from search.v1 import record_query_pb2 as search_query_type
from search.v1 import search_service_pb2 as searchv1
from sign.v1 import sign_service_pb2 as sign_types
from store.v1 import store_service_pb2 as store_types

from .client import Client, Config

client = Client(Config.load_from_env())


def init_records(count, test_function_name, push=True, publish=False):
    example_records = {}

    for index in range(count):
        generated_record = core_record_pb2.Record(
            v3=record_pb2.Record(
                name="{}-{}".format(test_function_name, index),
                version="v3",
                schema_version="v0.5.0",
                skills=[
                    skill_pb2.Skill(
                        name="Natural Language Processing",
                        id=1,
                    ),
                ],
                locators=[
                    locator_pb2.Locator(
                        type="ipv4",
                        url="127.0.0.1",
                    ),
                ],
                extensions=[
                    extension_pb2.Extension(
                        name="schema.oasf.agntcy.org/domains/domain-1",
                        version="v1",
                    )
                ],
                signature=signature_pb2.Signature(),
            )
        )

        example_records[index] = (None, generated_record)

    if push:
        records_list = list[core_record_pb2.Record](
            record for _, record in example_records.values()
        )

        for index, record in enumerate(records_list):
            # Push only one at a time to make sure of the cid pairing
            references = client.push(records=[record])

            example_records[index] = (references[0], record)

        # if publish:
        #     for record_ref, record in example_records.values():
        #         record_refs = routingv1.RecordRefs(refs=[record_ref])
        #         req = routingv1.PublishRequest(record_refs=record_refs)
        #         client.publish(req=req)

    return example_records


class TestClient(unittest.TestCase):
    def test_push(self):
        example_records = init_records(2, "push", push=False)
        records_list = list[core_record_pb2.Record](
            record for _, record in example_records.values()
        )

        references = client.push(records=records_list)

        self.assertIsNotNone(references)
        self.assertIsInstance(references, list)
        self.assertEqual(len(references), 2)

        for ref in references:
            self.assertIsInstance(ref, core_record_pb2.RecordRef)
            self.assertEqual(len(ref.cid), 59)

    def test_pull(self):
        example_records = init_records(2, "pull")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        pulled_records = client.pull(refs=record_refs_list)

        self.assertIsNotNone(pulled_records)
        self.assertIsInstance(pulled_records, list)
        self.assertEqual(len(pulled_records), 2)

        for record in pulled_records:
            self.assertIsInstance(record, core_record_pb2.Record)

    def test_lookup(self):
        example_records = init_records(2, "lookup")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        metadatas = client.lookup(record_refs_list)

        self.assertIsNotNone(metadatas)
        self.assertIsInstance(metadatas, list)
        self.assertEqual(len(metadatas), 2)

        for metadata in metadatas:
            self.assertIsInstance(metadata, core_record_pb2.RecordMeta)

    def test_publish(self):
        example_records = init_records(1, "publish")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        record_refs = routingv1.RecordRefs(refs=record_refs_list)
        publish_request = routingv1.PublishRequest(record_refs=record_refs)

        try:
            client.publish(publish_request)
        except Exception as e:
            self.assertIsNone(e)

    def test_list(self):
        _ = init_records(2, "list", publish=True)

        list_query = record_query_type.RecordQuery(
            type=record_query_type.RECORD_QUERY_TYPE_SKILL,
            value="/skills/Natural Language Processing/Text Completion",
        )

        list_request = routingv1.ListRequest(queries=[list_query])
        objects = list(client.list(list_request))

        self.assertIsNotNone(objects)
        self.assertNotEqual(len(objects), 0)

        for o in objects:
            self.assertIsInstance(o, routingv1.ListResponse)

    def test_search(self):
        _ = init_records(2, "search", publish=True)

        search_query = search_query_type.RecordQuery(
            type=search_query_type.RECORD_QUERY_TYPE_SKILL_ID, value="1"
        )

        search_request = searchv1.SearchRequest(queries=[search_query], limit=2)

        objects = list(client.search(search_request))

        self.assertIsNotNone(objects)
        self.assertNotEqual(len(objects), 0)

        for o in objects:
            self.assertIsInstance(o, searchv1.SearchResponse)

    def test_unpublish(self):
        example_records = init_records(1, "unpublish", publish=True)
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        record_refs = routingv1.RecordRefs(refs=record_refs_list)
        unpublish_request = routingv1.UnpublishRequest(record_refs=record_refs)

        try:
            client.unpublish(unpublish_request)
        except Exception as e:
            self.assertIsNone(e)

    def test_delete(self):
        example_records = init_records(2, "delete")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        try:
            client.delete(record_refs_list)
        except Exception as e:
            self.assertIsNone(e)

    def test_push_referrer(self):
        example_records = init_records(2, "push_referrer")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        try:
            example_signature = sign_types.Signature()
            request = [
                store_types.PushReferrerRequest(
                    record_ref=record_refs_list[0], signature=example_signature
                ),
                store_types.PushReferrerRequest(
                    record_ref=record_refs_list[1], signature=example_signature
                ),
            ]

            response = client.push_referrer(req=request)

            self.assertIsNotNone(response)
            self.assertEqual(len(response), 2)

            for r in response:
                self.assertIsInstance(r, store_types.PushReferrerResponse)

        except Exception as e:
            self.assertIsNone(e)

    def test_pull_referrer(self):
        example_records = init_records(2, "pull_referrer")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        try:
            request = [
                store_types.PullReferrerRequest(
                    record_ref=record_refs_list[0], pull_signature=False
                ),
                store_types.PullReferrerRequest(
                    record_ref=record_refs_list[1], pull_signature=False
                ),
            ]

            response = client.pull_referrer(req=request)

            self.assertIsNotNone(response)
            self.assertEqual(len(response), 2)

            for r in response:
                self.assertIsInstance(r, store_types.PullReferrerResponse)
        except Exception as e:
            self.assertTrue(
                "pull referrer not implemented" in str(e)
            )  # Delete when the service implemented

            # self.assertIsNone(e) # Uncomment when the service implemented

    def test_sign_and_verify(self):
        example_records = init_records(2, "sign_and_verify")
        record_refs_list = list[core_record_pb2.RecordRef](
            ref for ref, _ in example_records.values()
        )

        shell_env = os.environ.copy()

        key_password = "testing-key"
        shell_env["COSIGN_PASSWORD"] = key_password

        # Avoid interactive question about override
        try:
            os.remove("cosign.key")
            os.remove("cosign.pub")
        except FileNotFoundError:
            pass  # Clean state found

        cosign_path = os.getenv("COSIGN_PATH", "cosign")
        command = (cosign_path, "generate-key-pair")
        subprocess.run(command, check=True, capture_output=True, env=shell_env)

        with open("cosign.key", "rb") as reader:
            key_file = reader.read()

        key_provider = sign_types.SignWithKey(
            private_key=key_file, password=key_password.encode("utf-8")
        )

        token = shell_env.get("OIDC_TOKEN", "")
        provider_url = shell_env.get("OIDC_PROVIDER_URL", "")
        client_id = shell_env.get("OIDC_CLIENT_ID", "sigstore")

        oidc_options = sign_types.SignWithOIDC.SignOpts(oidc_provider_url=provider_url)
        oidc_provider = sign_types.SignWithOIDC(id_token=token, options=oidc_options)

        request_key_provider = sign_types.SignRequestProvider(key=key_provider)
        request_oidc_provider = sign_types.SignRequestProvider(oidc=oidc_provider)

        key_request = sign_types.SignRequest(
            record_ref=record_refs_list[0], provider=request_key_provider
        )
        oidc_request = sign_types.SignRequest(
            record_ref=record_refs_list[1], provider=request_oidc_provider
        )

        try:
            # Sign test
            result = client.sign(key_request)
            self.assertEqual(result.stderr.decode("utf-8"), "")
            self.assertEqual(
                result.stdout.decode("utf-8"), "Record signed successfully"
            )

            result = client.sign(oidc_request, client_id)
            self.assertEqual(
                result.stdout.decode("utf-8"), "Record signed successfully"
            )

            # Verify test
            for ref in record_refs_list:
                request = sign_types.VerifyRequest(record_ref=ref)
                response = client.verify(request)

                self.assertIs(response.success, True)
        except Exception as e:
            self.assertIsNone(e)
        finally:
            os.remove("cosign.key")
            os.remove("cosign.pub")

    def test_sync(self):
        try:
            create_request = sync_types.CreateSyncRequest(
                remote_directory_url=os.getenv(
                    "DIRECTORY_SERVER_PEER1_ADDRESS", "0.0.0.0:8891"
                )
            )
            create_response = client.create_sync(create_request)

            try:
                self.assertTrue(uuid.UUID(create_response.sync_id))
            except ValueError:
                raise ValueError("Not an UUID: {}".format(create_response.sync_id))

            list_request = sync_types.ListSyncsRequest()
            list_response = client.list_syncs(list_request)

            for sync_item in list_response:
                try:
                    self.assertIsInstance(sync_item, sync_types.ListSyncsItem)
                    self.assertTrue(uuid.UUID(sync_item.sync_id))
                except ValueError:
                    raise ValueError("Not an UUID: {}".format(sync_item.sync_id))

            get_request = sync_types.GetSyncRequest(sync_id=create_response.sync_id)
            get_response = client.get_sync(get_request)

            self.assertIsInstance(get_response, sync_types.GetSyncResponse)
            self.assertEqual(get_response.sync_id, create_response.sync_id)

            delete_request = sync_types.DeleteSyncRequest(
                sync_id=create_response.sync_id
            )
            client.delete_sync(delete_request)

        except Exception as e:
            self.assertIsNone(e)


if __name__ == "__main__":
    unittest.main()

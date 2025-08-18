import time
import unittest

import core.v1.record_pb2 as core_record_pb2
from objects.v3 import extension_pb2, locator_pb2, record_pb2, signature_pb2, skill_pb2
from routing.v1 import record_query_pb2 as record_query_type
from routing.v1 import routing_service_pb2 as routingv1
from search.v1 import record_query_pb2 as search_query_type
from search.v1 import search_service_pb2 as searchv1
from sign.v1 import sign_service_pb2 as sign_types
from store.v1 import store_service_pb2 as store_types

from .client import Client, Config

client = Client(Config())


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

        if publish:
            for record_ref, record in example_records.values():
                req = routingv1.PublishRequest(record_cid=record_ref.cid)
                client.publish(req=req)

    time.sleep(3)

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

        publish_request = routingv1.PublishRequest(record_cid=record_refs_list[0].cid)

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

        unpublish_request = routingv1.UnpublishRequest(
            record_cid=record_refs_list[0].cid
        )

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
                store_types.PushReferrerRequest(record_ref=record_refs_list[0], signature=example_signature),
                store_types.PushReferrerRequest(record_ref=record_refs_list[1], signature=example_signature),
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
            self.assertTrue("pull referrer not implemented" in str(e)) # Delete when the service implemented

            # self.assertIsNone(e) # Uncomment when the service implemented


if __name__ == "__main__":
    unittest.main()

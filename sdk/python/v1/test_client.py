import unittest

import core.v1.record_pb2 as core_record_pb2
from objects.v3 import extension_pb2, record_pb2, signature_pb2, skill_pb2, locator_pb2
from routing.v1 import record_query_pb2 as record_query_type
from routing.v1 import routing_service_pb2 as routingv1

from .client import Client, Config

client = Client(Config())

def generate_records(names):
    example_records = [];

    for name in names:
        example_record = core_record_pb2.Record(
            v3=record_pb2.Record(
                name=name,
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

        example_records.append(example_record)

    return example_records

class TestClient(unittest.TestCase):
    example_records = generate_records(["example-record", "example-record2"])

    example_record_refs = []

    def test_1_push(self):
        references = client.push(records=self.example_records)

        self.assertIsNotNone(references)
        self.assertIsInstance(references, list)
        self.assertEqual(len(references), 2)

        for ref in references:
            self.assertIsInstance(ref, core_record_pb2.RecordRef)
            self.assertEqual(len(ref.cid), 59)

        TestClient.example_record_refs = references

    def test_2_pull(self):
        pulled_records = client.pull(refs=TestClient.example_record_refs)

        self.assertIsNotNone(pulled_records)
        self.assertIsInstance(pulled_records, list)
        self.assertEqual(len(pulled_records), 2)

        for record in pulled_records:
            self.assertIsInstance(record, core_record_pb2.Record)

    def test_3_lookup(self):
        metadatas = client.lookup(TestClient.example_record_refs)

        self.assertIsNotNone(metadatas)
        self.assertIsInstance(metadatas, list)
        self.assertEqual(len(metadatas), 2)

        for metadata in metadatas:
            self.assertIsInstance(metadata, core_record_pb2.RecordMeta)

    def test_4_publish(self):
        publish_request = routingv1.PublishRequest(record_cid=TestClient.example_record_refs[0].cid)

        try:
            client.publish(publish_request)
        except Exception as e:
            self.assertIsNone(e)

    def test_5_list(self):
        list_query = record_query_type.RecordQuery(
            type=record_query_type.RECORD_QUERY_TYPE_SKILL,
            value="/skills/Natural Language Processing/Text Completion",
        )

        list_request = routingv1.ListRequest(queries=[list_query])
        objects = list(client.list(list_request))

        self.assertIsNotNone(objects)
        self.assertNotEqual(objects, 0)

        for o in objects:
            self.assertIsInstance(o, routingv1.ListResponse)

    def test_7_unpublish(self):
        unpublish_request = routingv1.UnpublishRequest(record_cid=TestClient.example_record_refs[0].cid)

        try:
            client.unpublish(unpublish_request)
        except Exception as e:
            self.assertIsNone(e)

    def test_8_delete(self):
        try:
            client.delete(TestClient.example_record_refs)
        except Exception as e:
            self.assertIsNone(e)

if __name__ == "__main__":
    unittest.main()

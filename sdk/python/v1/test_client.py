import unittest

import core.v1.record_pb2 as core_record_pb2
from objects.v3 import extension_pb2, record_pb2, signature_pb2, skill_pb2, locator_pb2
from routing.v1 import record_query_pb2 as record_query_type
from routing.v1 import routing_service_pb2 as routingv1

from .client import Client, Config

client = Client(Config())


class TestClient(unittest.TestCase):
    example_record = core_record_pb2.Record(
        v3=record_pb2.Record(
            name="example-record",
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

    example_record_ref: core_record_pb2.RecordRef = None

    def test_1_push(self):
        reference = client.push(record=self.example_record)

        self.assertIsNotNone(reference)
        self.assertIsInstance(reference, core_record_pb2.RecordRef)
        self.assertEqual(len(reference.cid), 59)

        TestClient.example_record_ref = reference

    def test_2_pull(self):
        pulled_record = client.pull(ref=TestClient.example_record_ref)

        self.assertIsNotNone(pulled_record)
        self.assertIsInstance(pulled_record, core_record_pb2.Record)

    def test_3_lookup(self):
        metadata = client.lookup(TestClient.example_record_ref)

        self.assertIsNotNone(metadata)
        self.assertIsInstance(metadata, core_record_pb2.RecordMeta)

    def test_4_publish(self):
        client.publish(TestClient.example_record_ref)

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
        client.unpublish(TestClient.example_record_ref)

    def test_8_delete(self):
        client.delete(TestClient.example_record_ref)


if __name__ == "__main__":
    unittest.main()

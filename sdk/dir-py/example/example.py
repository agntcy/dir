import json

import core.v1.record_pb2 as core_record_pb2
from google.protobuf.json_format import MessageToDict
from objects.v3 import extension_pb2, locator_pb2, record_pb2, signature_pb2, skill_pb2
from routing.v1 import record_query_pb2 as record_query_type
from routing.v1 import routing_service_pb2 as routingv1
from search.v1 import search_service_pb2 as searchv1
from search.v1 import record_query_pb2 as search_query_type

from v1.client import Client, Config


def convert_uint64_fields_to_int(d):
    """
    Recursively convert specific fields known to be uint64 from string to int.
    """
    if isinstance(d, dict):
        for key, value in d.items():
            if (
                key in ("category_uid", "class_uid")
                and isinstance(value, str)
                and value.isdigit()
            ):
                d[key] = int(value)
            elif isinstance(value, dict):
                convert_uint64_fields_to_int(value)
            elif isinstance(value, list):
                for item in value:
                    convert_uint64_fields_to_int(item)
    return d


def print_as_json(object: any):
    """
    Convert object to a serializable JSON object and print it.
    :param object: object to serialize
    :return: json_object: serialized JSON object
    """

    # Convert the record object to a JSON string
    dict_object = MessageToDict(object, preserving_proto_field_name=True)
    dict_object = convert_uint64_fields_to_int(dict_object)
    json_object = json.dumps(dict_object).encode("utf-8")

    return json_object


def generate_records(names):
    example_records = []

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


# Initialize the client
client = Client(Config())

records = generate_records(["example-record", "example-record2"])

# Push objects to the store
refs = client.push(records)

for ref in refs:
    print("Pushed object ref:", print_as_json(ref))

# Pull objects from the store
pulled_records = client.pull(refs)

for pulled_record in pulled_records:
    print("Pulled object data:", print_as_json(pulled_record))


# Lookup the object
metadatas = client.lookup(refs)

for metadata in metadatas:
    print("Lookup object metadata:", print_as_json(metadata))

# Publish the object
record_refs = routingv1.RecordRefs(refs=[refs[0]])
publish_request = routingv1.PublishRequest(record_refs=record_refs)
client.publish(publish_request)
print("Object published.")

# List objects in the store
query = record_query_type.RecordQuery(
    type=record_query_type.RECORD_QUERY_TYPE_SKILL,
    value="/skills/Natural Language Processing/Text Completion",
)

list_request = routingv1.ListRequest(queries=[query])
objects = list(client.list(list_request))

for o in objects:
    print("Listed object:", print_as_json(o))

# Search objects
search_query = search_query_type.RecordQuery(
    type=search_query_type.RECORD_QUERY_TYPE_SKILL_ID,
    value="1")

search_request = searchv1.SearchRequest(
    queries=[search_query],
    limit=3)

objects = list(client.search(search_request))

print(objects)

# Unpublish the object
record_refs = routingv1.RecordRefs(refs=[refs[0]])
unpublish_request = routingv1.UnpublishRequest(record_refs=record_refs)
client.unpublish(unpublish_request)
print("Object unpublished.")

# Delete the object
client.delete(refs)
print("Objects are deleted.")

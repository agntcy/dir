import json

import core.v1.record_pb2 as core_record_pb2
from google.protobuf.json_format import MessageToDict
from objects.v3 import extension_pb2, record_pb2, signature_pb2, skill_pb2
from routing.v1alpha2 import record_query_pb2 as record_query_type
from routing.v1alpha2 import routing_service_pb2 as routingtypes

from client import Client, Config


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


# Initialize the client
client = Client(Config())

# Create a record object
example_record = record_pb2.Record(
    name="example-record",
    version="v3",
    skills=[
        skill_pb2.Skill(
            name="Natural Language Processing",
            id=1,
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

r = core_record_pb2.Record(v3=example_record)

# Push the object to the store
ref = client.push(record=r)
print("Pushed object ref:", ref)

# Pull the object from the store
record = client.pull(ref)

# Convert the record object to a JSON string
record_dict = MessageToDict(record, preserving_proto_field_name=True)
record_dict = convert_uint64_fields_to_int(record_dict)
record_json = json.dumps(record_dict).encode("utf-8")

print("Pulled object data:", record_json)

# Lookup the object
metadata = client.lookup(ref)

# Convert the metadata object to a JSON string
metadata_dict = MessageToDict(metadata, preserving_proto_field_name=True)
metadata_dict = convert_uint64_fields_to_int(metadata_dict)
metadata_json = json.dumps(metadata_dict).encode("utf-8")

print("Object metadata:", metadata_json)

# Publish the object
client.publish(ref)
print("Object published.")

# List objects in the store
query = record_query_type.RecordQuery(
    type=record_query_type.RECORD_QUERY_TYPE_SKILL,
    value="/skills/Natural Language Processing/Text Completion",
)

list_request = routingtypes.ListRequest(queries=[query])
objects = list(client.list(list_request))
print("Listed objects:", objects)

# Unpublish the object
client.unpublish(ref)
print("Object unpublished.")

# Delete the object
client.delete(ref)
print("Object deleted.")

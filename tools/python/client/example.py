import io
import hashlib
import json

from google.protobuf.json_format import MessageToDict
from client.client import Client, Config
from core.v1alpha1 import object_pb2, agent_pb2, skill_pb2, extension_pb2, signature_pb2
from routing.v1alpha1 import routing_service_pb2 as routingtypes

def convert_uint64_fields_to_int(d):
    """
    Recursively convert specific fields known to be uint64 from string to int.
    """
    if isinstance(d, dict):
        for key, value in d.items():
            if key in ("category_uid", "class_uid") and isinstance(value, str) and value.isdigit():
                d[key] = int(value)
            elif isinstance(value, dict):
                convert_uint64_fields_to_int(value)
            elif isinstance(value, list):
                for item in value:
                    convert_uint64_fields_to_int(item)
    return d

# Initialize the client
client = Client(Config())

# Create an agent object
agent = agent_pb2.Agent(
    name="example-agent",
    version="v1",
    skills=[
        skill_pb2.Skill(
            category_name="Natural Language Processing",
            category_uid=1,
            class_name="Text Completion",
            class_uid=10201,
        ),
    ],
    extensions=[
        extension_pb2.Extension(
            name="schema.oasf.agntcy.org/domains/domain-1",
            version="v1",
        )
    ],
    signature=signature_pb2.Signature()
)

agent_dict = MessageToDict(agent, preserving_proto_field_name=True)
agent_dict = convert_uint64_fields_to_int(agent_dict)

# Convert the agent object to a JSON string
agent_json = json.dumps(agent_dict, separators=(',', ':')).encode('utf-8')
print(agent_json)

# Create a reference for the object
ref = object_pb2.ObjectRef(
    digest="sha256:" + hashlib.sha256(agent_json).hexdigest(),
    type=object_pb2.ObjectType.Name(object_pb2.ObjectType.OBJECT_TYPE_AGENT),
    size=len(agent_json),
    annotations=agent.annotations,
)

# Push the object to the store
data_stream = io.BytesIO(agent_json)
response = client.push(ref, data_stream)
print("Pushed object:", response)

# Pull the object from the store
data_stream = client.pull(ref)

# Deserialize the data
pulled_agent_json = data_stream.getvalue().decode('utf-8')
print("Pulled object data:", pulled_agent_json)

# Lookup the object
metadata = client.lookup(ref)
print("Object metadata:", metadata)

# Publish the object
client.publish(ref, network=False)
print("Object published.")

# List objects in the store
list_request = routingtypes.ListRequest(
    labels=["/skills/Natural Language Processing/Text Completion"]
)
objects = list(client.list(list_request))
print("Listed objects:", objects)

# Unpublish the object
client.unpublish(ref, network=False)
print("Object unpublished.")

# Delete the object
client.delete(ref)
print("Object deleted.")

from client import new_client, with_env_config
from client_methods import ClientMethods
from proto.core.v1alpha1.object_pb2 import core_pb2

# Create a new client
client = new_client(with_env_config())

# Create an instance of ClientMethods
client_methods = ClientMethods(client)

# Example usage
ref = core_pb2.ObjectRef(id="example_id")
client_methods.publish(ref, network=True)

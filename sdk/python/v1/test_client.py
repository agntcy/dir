import unittest
import io
import hashlib
from unittest.mock import patch, MagicMock

from .client import Client, Config
from core.v1alpha1 import object_pb2, agent_pb2, skill_pb2, signature_pb2
from routing.v1alpha1 import routing_service_pb2

class TestClient(unittest.TestCase):
    def setUp(self):
        # Set up test data
        self.agent = agent_pb2.Agent(
            name="test-agent",
            version="v1",
            skills=[
                skill_pb2.Skill(
                    category_name="test-category-1",
                    class_name="test-class-1"
                ),
                skill_pb2.Skill(
                    category_name="test-category-2",
                    class_name="test-class-2"
                )
            ],
            annotations={
                "lorem": "ipsum",
                "dolor": "sit"
            },
            signature=signature_pb2.Signature()
        )


        # Marshal the Agent struct to bytes
        self.agent_data = self.agent.SerializeToString()

        # Compute the digest
        self.digest = "sha256:" + hashlib.sha256(self.agent_data).hexdigest()

        # Create ref
        self.ref = object_pb2.ObjectRef(
            digest=self.digest,
            type=object_pb2.ObjectType.Name(object_pb2.ObjectType.OBJECT_TYPE_AGENT),
            size=len(self.agent_data),
            annotations=self.agent.annotations,
        )

        # Create patch targets for gRPC clients
        self.store_client_patch = patch('store.v1alpha1.store_service_pb2_grpc.StoreServiceStub')
        self.routing_client_patch = patch('routing.v1alpha1.routing_service_pb2_grpc.RoutingServiceStub')

        # Start patches
        self.mock_store_client = self.store_client_patch.start()
        self.mock_routing_client = self.routing_client_patch.start()

        # Create client with mocked gRPC services
        self.client = Client(Config())

        # Replace client's service clients with our mocks
        self.client.store_client = self.mock_store_client
        self.client.routing_client = self.mock_routing_client

    def tearDown(self):
        # Stop patches
        self.store_client_patch.stop()
        self.routing_client_patch.stop()

    def test_config_from_env(self):
        """Test loading configuration from environment."""
        with patch('os.environ', {'DIRECTORY_CLIENT_SERVER_ADDRESS': 'test-server:9999'}):
            config = Config.load_from_env()
            self.assertEqual(config.server_address, 'test-server:9999')

    def test_push(self):
        """Test pushing an object."""
        # Set up mock for Push method
        mock_response = object_pb2.ObjectRef(
            digest=self.digest,
            type=self.ref.type,
            size=self.ref.size,
        )
        self.mock_store_client.Push.return_value = mock_response

        # Create data stream
        data_stream = io.BytesIO(self.agent_data)

        # Call push method
        result = self.client.push(self.ref, data_stream)

        # Verify result
        self.assertEqual(result.digest, self.digest)
        self.assertEqual(result.type, self.ref.type)
        self.assertEqual(result.size, self.ref.size)

        # Verify Push was called
        self.mock_store_client.Push.assert_called_once()

    def test_pull(self):
        """Test pulling an object."""
        # Set up mock for Pull method
        mock_response_iter = [
            object_pb2.Object(
                ref=self.ref,
                data=self.agent_data[:2048]
            ),
            object_pb2.Object(
                ref=self.ref,
                data=self.agent_data[2048:]
            )
        ]
        mock_stream = MagicMock()
        mock_stream.__iter__.return_value = mock_response_iter
        self.mock_store_client.Pull.return_value = mock_stream

        # Call pull method
        result = self.client.pull(self.ref)

        # Verify result
        self.assertEqual(result.getvalue(), self.agent_data)

        # Verify Pull was called with correct parameters
        self.mock_store_client.Pull.assert_called_once_with(self.ref, metadata=None)

    def test_lookup(self):
        """Test looking up an object."""
        # Set up mock for Lookup method
        self.mock_store_client.Lookup.return_value = self.ref

        # Call lookup method
        result = self.client.lookup(self.ref)

        # Verify result
        self.assertEqual(result, self.ref)

        # Verify Lookup was called with correct parameters
        self.mock_store_client.Lookup.assert_called_once_with(self.ref, metadata=None)

    def test_delete(self):
        """Test deleting an object."""
        # Set up mock for Delete method
        self.mock_store_client.Delete.return_value = None

        # Call delete method
        self.client.delete(self.ref)

        # Verify Delete was called with correct parameters
        self.mock_store_client.Delete.assert_called_once_with(self.ref, metadata=None)

    def test_publish(self):
        """Test publishing an object."""
        # Set up mock for Publish method
        self.mock_routing_client.Publish.return_value = None

        # Call publish method
        self.client.publish(self.ref, network=True)

        # Verify Publish was called with correct parameters
        self.mock_routing_client.Publish.assert_called_once()
        args, kwargs = self.mock_routing_client.Publish.call_args
        self.assertEqual(args[0].record, self.ref)
        self.assertEqual(args[0].network, True)

    def test_unpublish(self):
        """Test unpublishing an object."""
        # Set up mock for Unpublish method
        self.mock_routing_client.Unpublish.return_value = None

        # Call unpublish method
        self.client.unpublish(self.ref, network=True)

        # Verify Unpublish was called with correct parameters
        self.mock_routing_client.Unpublish.assert_called_once()
        args, kwargs = self.mock_routing_client.Unpublish.call_args
        self.assertEqual(args[0].record, self.ref)
        self.assertEqual(args[0].network, True)

    def test_list(self):
        """Test listing objects."""
        # Create a list request
        list_request = routing_service_pb2.ListRequest()

        # Create mock items for the response
        mock_items = [
            routing_service_pb2.ListResponse.Item(record=self.ref),
            routing_service_pb2.ListResponse.Item(record=object_pb2.ObjectRef(
                digest="sha256:another_digest",
                type=self.ref.type,
                size=1024
            ))
        ]

        # Create a mock response with the items
        mock_response = routing_service_pb2.ListResponse(items=mock_items)

        # Set up the mock stream to return the response
        mock_stream = MagicMock()
        mock_stream.__iter__.return_value = [mock_response]
        self.mock_routing_client.List.return_value = mock_stream

        # Call list method
        result = list(self.client.list(list_request))

        # Verify result
        self.assertEqual(len(result), 2)
        self.assertEqual(result[0].record.digest, self.ref.digest)
        self.assertEqual(result[1].record.digest, "sha256:another_digest")

        # Verify List was called with correct parameters
        self.mock_routing_client.List.assert_called_once_with(list_request, metadata=None)
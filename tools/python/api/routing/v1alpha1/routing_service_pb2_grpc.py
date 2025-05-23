# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
"""Client and server classes corresponding to protobuf-defined services."""
import grpc

from google.protobuf import empty_pb2 as google_dot_protobuf_dot_empty__pb2
from routing.v1alpha1 import routing_service_pb2 as routing_dot_v1alpha1_dot_routing__service__pb2


class RoutingServiceStub(object):
    """Defines an interface for publication and retrieval
    of objects across interconnected network.
    """

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.Publish = channel.unary_unary(
                '/routing.v1alpha1.RoutingService/Publish',
                request_serializer=routing_dot_v1alpha1_dot_routing__service__pb2.PublishRequest.SerializeToString,
                response_deserializer=google_dot_protobuf_dot_empty__pb2.Empty.FromString,
                _registered_method=True)
        self.List = channel.unary_stream(
                '/routing.v1alpha1.RoutingService/List',
                request_serializer=routing_dot_v1alpha1_dot_routing__service__pb2.ListRequest.SerializeToString,
                response_deserializer=routing_dot_v1alpha1_dot_routing__service__pb2.ListResponse.FromString,
                _registered_method=True)
        self.Unpublish = channel.unary_unary(
                '/routing.v1alpha1.RoutingService/Unpublish',
                request_serializer=routing_dot_v1alpha1_dot_routing__service__pb2.UnpublishRequest.SerializeToString,
                response_deserializer=google_dot_protobuf_dot_empty__pb2.Empty.FromString,
                _registered_method=True)


class RoutingServiceServicer(object):
    """Defines an interface for publication and retrieval
    of objects across interconnected network.
    """

    def Publish(self, request, context):
        """Notifies the network that the node is providing given object.
        Listeners should use this event to update their routing tables.
        They may optionally forward the request to other nodes.
        Items need to be periodically republished to avoid stale data.

        It is the API responsibility to fully construct the routing details,
        these are minimal details needed for us to publish the request.
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def List(self, request, context):
        """List all the available items across the network.
        TODO: maybe remove to search?
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')

    def Unpublish(self, request, context):
        """Unpublish a given object.
        This will remove the object from the network.
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')


def add_RoutingServiceServicer_to_server(servicer, server):
    rpc_method_handlers = {
            'Publish': grpc.unary_unary_rpc_method_handler(
                    servicer.Publish,
                    request_deserializer=routing_dot_v1alpha1_dot_routing__service__pb2.PublishRequest.FromString,
                    response_serializer=google_dot_protobuf_dot_empty__pb2.Empty.SerializeToString,
            ),
            'List': grpc.unary_stream_rpc_method_handler(
                    servicer.List,
                    request_deserializer=routing_dot_v1alpha1_dot_routing__service__pb2.ListRequest.FromString,
                    response_serializer=routing_dot_v1alpha1_dot_routing__service__pb2.ListResponse.SerializeToString,
            ),
            'Unpublish': grpc.unary_unary_rpc_method_handler(
                    servicer.Unpublish,
                    request_deserializer=routing_dot_v1alpha1_dot_routing__service__pb2.UnpublishRequest.FromString,
                    response_serializer=google_dot_protobuf_dot_empty__pb2.Empty.SerializeToString,
            ),
    }
    generic_handler = grpc.method_handlers_generic_handler(
            'routing.v1alpha1.RoutingService', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))
    server.add_registered_method_handlers('routing.v1alpha1.RoutingService', rpc_method_handlers)


 # This class is part of an EXPERIMENTAL API.
class RoutingService(object):
    """Defines an interface for publication and retrieval
    of objects across interconnected network.
    """

    @staticmethod
    def Publish(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_unary(
            request,
            target,
            '/routing.v1alpha1.RoutingService/Publish',
            routing_dot_v1alpha1_dot_routing__service__pb2.PublishRequest.SerializeToString,
            google_dot_protobuf_dot_empty__pb2.Empty.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)

    @staticmethod
    def List(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_stream(
            request,
            target,
            '/routing.v1alpha1.RoutingService/List',
            routing_dot_v1alpha1_dot_routing__service__pb2.ListRequest.SerializeToString,
            routing_dot_v1alpha1_dot_routing__service__pb2.ListResponse.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)

    @staticmethod
    def Unpublish(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_unary(
            request,
            target,
            '/routing.v1alpha1.RoutingService/Unpublish',
            routing_dot_v1alpha1_dot_routing__service__pb2.UnpublishRequest.SerializeToString,
            google_dot_protobuf_dot_empty__pb2.Empty.FromString,
            options,
            channel_credentials,
            insecure,
            call_credentials,
            compression,
            wait_for_ready,
            timeout,
            metadata,
            _registered_method=True)

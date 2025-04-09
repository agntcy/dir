# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import grpc
from typing import Generator, Optional
from client import Client
from your_proto_package import core_pb2, routing_pb2

class ClientMethods:
    def __init__(self, client: Client):
        self.client = client

    def publish(self, ref: core_pb2.ObjectRef, network: bool) -> None:
        request = routing_pb2.PublishRequest(record=ref, network=network)
        self.client.routing_client.Publish(request)

    def list(self, req: routing_pb2.ListRequest) -> Generator[routing_pb2.ListResponse.Item, None, None]:
        stream = self.client.routing_client.List(req)
        for response in stream:
            for item in response.items:
                yield item

    def unpublish(self, ref: core_pb2.ObjectRef, network: bool) -> None:
        request = routing_pb2.UnpublishRequest(record=ref, network=network)
        self.client.routing_client.Unpublish(request)

    def push(self, ref: core_pb2.ObjectRef, reader: Any) -> core_pb2.ObjectRef:
        stream = self.client.store_client.Push()
        chunk_size = 4096
        while True:
            data = reader.read(chunk_size)
            if not data:
                break
            obj = core_pb2.Object(ref=ref, data=data)
            stream.send(obj)
        return stream.close_and_recv()

    def pull(self, ref: core_pb2.ObjectRef) -> bytes:
        stream = self.client.store_client.Pull(ref)
        buffer = bytearray()
        for obj in stream:
            buffer.extend(obj.data)
        return bytes(buffer)

    def lookup(self, ref: core_pb2.ObjectRef) -> core_pb2.ObjectRef:
        return self.client.store_client.Lookup(ref)

    def delete(self, ref: core_pb2.ObjectRef) -> None:
        self.client.store_client.Delete(ref)

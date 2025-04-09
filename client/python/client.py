# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import grpc
from typing import Callable, Optional, List, Any
from config import Config, load_config
from your_proto_package import store_pb2_grpc, routing_pb2_grpc, core_pb2, routing_pb2

class Client:
    def __init__(self, store_client, routing_client):
        self.store_client = store_client
        self.routing_client = routing_client

class Options:
    def __init__(self, config: Optional[Config] = None):
        self.config = config or load_config()

def with_env_config() -> Callable[[Options], None]:
    def _with_env_config(opts: Options):
        opts.config = load_config()
    return _with_env_config

def with_config(config: Config) -> Callable[[Options], None]:
    def _with_config(opts: Options):
        opts.config = config
    return _with_config

def new_client(*opts: Callable[[Options], None]) -> Client:
    options = Options()
    for opt in opts:
        opt(options)

    channel = grpc.insecure_channel(options.config.server_address)
    store_client = store_pb2_grpc.StoreServiceStub(channel)
    routing_client = routing_pb2_grpc.RoutingServiceStub(channel)

    return Client(store_client, routing_client)

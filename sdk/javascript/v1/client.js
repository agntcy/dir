// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import process from "node:process";
const grpc = require('@grpc/grpc-js');

const core = require('@buf/agntcy_dir.grpc_web/core/v1alpha1/object_pb')
const routing = require('@buf/agntcy_dir.grpc_web/routing/v1alpha1/routing_service_pb')
const sign = require('@buf/agntcy_dir.grpc_web/sign/v1alpha1/sign_service_pb')
const store = require('@buf/agntcy_dir.grpc_web/store/v1alpha1/store_service_pb')

class Config {
    static DEFAULT_ENV_PREFIX = 'DIRECTORY_CLIENT';
    static DEFAULT_SERVER_ADDRESS = '0.0.0.0:8888';

    constructor(serverAddress = Config.DEFAULT_SERVER_ADDRESS) {
        this.serverAddress = serverAddress;
    }

    static loadFromEnv() {
        const prefix = Config.DEFAULT_ENV_PREFIX;
        const serverAddress = process.env[`${prefix}_SERVER_ADDRESS`] || Config.DEFAULT_SERVER_ADDRESS;
        return new Config(serverAddress);
    }
}

class Client {
    constructor(config) {
        this.channel = new grpc.Client(config.serverAddress, grpc.credentials.createInsecure());
        this.storeClient = new storeServices.StoreServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        this.routingClient = new routingServices.RoutingServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        // Add sign client etc. as needed
    }

    static new(config = null) {
        if (!config) {
            config = Config.loadFromEnv();
        }
        return new Client(config);
    }

    publish(ref, network = false, metadata = null) {
        throw new Error('Not implemented');
    }

    list(req, metadata = null) {
        throw new Error('Not implemented');
    }

    unpublish(ref, network = false, metadata = null) {
        throw new Error('Not implemented');
    }

    push(ref, reader, metadata = null) {
        throw new Error('Not implemented');
    }

    pull(ref, metadata = null) {
        throw new Error('Not implemented');
    }

    lookup(ref, metadata = null) {
        throw new Error('Not implemented');
    }

    delete(ref, metadata = null) {
        throw new Error('Not implemented');
    }

    sign(req) {
        throw new Error('Not implemented');
    }

    verify(req) {
        throw new Error('Not implemented');
    }
}

module.exports = {
    Config,
    Client,
};
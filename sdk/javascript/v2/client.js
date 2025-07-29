// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

const process = require('node:process');
const grpc = require('@grpc/grpc-js');

const routing_service = require('@buf/agntcy_dir.grpc_node/routing/v1alpha2/routing_service_grpc_pb');
const store_service = require('@buf/agntcy_dir.grpc_node/store/v1alpha2/store_service_grpc_pb');
const search_service = require('@buf/agntcy_dir.grpc_node/search/v1alpha2/search_service_grpc_pb')
const routing_type = require('@buf/agntcy_dir.grpc_node/routing/v1alpha2/routing_service_pb')
const record_type = require('@buf/agntcy_dir.grpc_node/core/v1/record_pb');
const search_type = require('@buf/agntcy_dir.grpc_node/search/v1alpha2/search_service_pb')

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
        this.storeClient = new store_service.StoreServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        this.routingClient = new routing_service.RoutingServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        this.searchClient = new search_service.SearchServiceClient(config.serverAddress, grpc.credentials.createInsecure());

    }

    static new(config = null) {
        if (!config) {
            config = Config.loadFromEnv();
        }
        return new Client(config);
    }

    push(record, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.push();

            let ref = new record_type.RecordRef();

            call.on('data', (response) => {
                ref = response.u
            });

            call.on('end', () => {
                resolve(ref)
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error)
            });

            // Send a requests and close the stream
            call.write(record, metadata);
            call.end();
        });
    }

    pull(ref, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.pull();

            let record = new record_type.Record();

            call.on('data', (response) => {
                record = response.u
            });

            call.on('end', () => {
                resolve(record)
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error)
            });

            // Send a requests and close the stream
            call.write(ref, metadata);
            call.end();
        });
    }

    search(request, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.searchClient.search(request, metadata);

            let results = new search_type.SearchResponse();

            // Handle response stream
            call.on('data', (response) => {
                results = response.u
            });

            call.on('end', () => {
                resolve(results)
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error)
            });
        });
    }

    lookup(ref, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.lookup();

            let record_meta = new record_type.RecordMeta();

            // Handle response stream
            call.on('data', (response) => {
                record_meta = response.u
            });

            call.on('end', () => {
                resolve(record_meta)
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error)
            });

            // Send a requests and close the stream
            call.write(ref, metadata);
            call.end();
        });
    }

    list(request, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.routingClient.list(request, metadata);

            let results = new routing_type.ListResponse();

            call.on('data', (response) => {
                results = response.u
            });

            call.on('end', () => {
                resolve(results)
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error)
            });
        });
    }

    publish(request) {
        this.routingClient.publish(request, (error, _) => {
            if (error) {
                console.error('Error from server:', error);
            }
        });
    }

    unpublish(request) {
        this.routingClient.unpublish(request, (error, _) => {
            if (error) {
                console.error('Error from server:', error);
            }
        });
    }

    delete(ref) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.delete((error, _) => {
                if (error.code !== 12) {
                    console.error('Error from server:', error)
                    reject(err);
                }

                resolve(); // No response to resolve
            });

            // Send a requests and close the stream
            call.write(ref);
            call.end();
        });
    }
}

module.exports = {
    Config,
    Client,
};
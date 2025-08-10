// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

const process = require('node:process');
const grpc = require('@grpc/grpc-js');

const routing_service = require('@buf/agntcy_dir.grpc_node/routing/v1/routing_service_grpc_pb');
const store_service = require('@buf/agntcy_dir.grpc_node/store/v1/store_service_grpc_pb');
const search_service = require('@buf/agntcy_dir.grpc_node/search/v1/search_service_grpc_pb')

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

    push(records, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.push();

            let refs = [];

            call.on('data', (response) => {
                refs.push(response);
            });

            call.on('end', () => {
                resolve(refs);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
            });

            // Send a requests and close the stream
            records.forEach(record => {
                call.write(record, metadata);
            });

            call.end();
        });
    }

    pull(refs, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.pull();

            let records = [];

            call.on('data', (response) => {
                records.push(response.u);
            });

            call.on('end', () => {
                resolve(records);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
            });

            // Send a requests and close the stream
            refs.forEach(ref => {
                call.write(ref, metadata);
            });

            call.end();
        });
    }

    search(request, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.searchClient.search(request, metadata);

            let results = [];

            // Handle response stream
            call.on('data', (response) => {
                results.push(response.u[0]); // Save CIDs
            });

            call.on('end', () => {
                resolve(results);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
            });
        });
    }

    lookup(refs, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.lookup();

            let record_metas = [];

            // Handle response stream
            call.on('data', (response) => {
                record_metas.push(response.u);
            });

            call.on('end', () => {
                resolve(record_metas);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
            });

            // Send a requests and close the stream
            refs.forEach(ref => {
                call.write(ref, metadata);
            });

            call.end();
        });
    }

    list(request, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.routingClient.list(request, metadata);

            let results = [];

            call.on('data', (response) => {
                results.push(response.u);
            });

            call.on('end', () => {
                resolve(results);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
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

    delete(refs) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.delete((error, _) => {
                if (error.code !== 12) {
                    console.error('Error from server:', error)
                    reject(error);
                }

                resolve(); // No response to resolve
            });

            // Send a requests and close the stream
            refs.forEach(ref => {
                call.write(ref);
            });

            call.end();
        });
    }
}

module.exports = {
    Config,
    Client,
};
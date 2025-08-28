// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

const process = require('node:process');
const grpc = require('@grpc/grpc-js');
const { execSync } = require('child_process');
const { rmSync, writeFileSync } = require('node:fs');

const routing_service = require('@buf/agntcy_dir.grpc_node/routing/v1/routing_service_grpc_pb');
const store_service = require('@buf/agntcy_dir.grpc_node/store/v1/store_service_grpc_pb');
const search_service = require('@buf/agntcy_dir.grpc_node/search/v1/search_service_grpc_pb')
const sign_service = require('@buf/agntcy_dir.grpc_node/sign/v1/sign_service_grpc_pb')


class Config {
    static DEFAULT_ENV_PREFIX = 'DIRECTORY_CLIENT';
    static DEFAULT_SERVER_ADDRESS = '0.0.0.0:8888';

    static DEFAULT_DIRCTL_PATH = "dirctl"

    constructor(serverAddress = Config.DEFAULT_SERVER_ADDRESS, dirctl_path = Config.DEFAULT_DIRCTL_PATH) {
        this.serverAddress = serverAddress;
        this.dirctl_path = dirctl_path;
    }

    static loadFromEnv() {
        const prefix = Config.DEFAULT_ENV_PREFIX;
        const serverAddress = process.env[`${prefix}_SERVER_ADDRESS`] || Config.DEFAULT_SERVER_ADDRESS;
        const dirctl_path = process.env["DIRCTL_PATH"] || Config.DEFAULT_DIRCTL_PATH;
        return new Config(serverAddress, dirctl_path);
    }
}

class Client {
    constructor(config) {
        this.storeClient = new store_service.StoreServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        this.routingClient = new routing_service.RoutingServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        this.searchClient = new search_service.SearchServiceClient(config.serverAddress, grpc.credentials.createInsecure());
        this.signClient = new sign_service.SignServiceClient(config.serverAddress, grpc.credentials.createInsecure());

        this.dirctl_path = config.dirctl_path;
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

    push_referrer(requests, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.pushReferrer();

            let pushReferrerResponses = [];

            call.on('data', (response) => {
                pushReferrerResponses.push(response);
            });

            call.on('end', () => {
                resolve(pushReferrerResponses);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
            });

            // Send a requests and close the stream
            requests.forEach(request => {
                call.write(request, metadata);
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

    pull_referrer(requests, metadata = null) {
        return new Promise((resolve, reject) => {
            const call = this.storeClient.pullReferrer();

            let pullReferrerResponses = [];

            call.on('data', (response) => {
                pullReferrerResponses.push(response);
            });

            call.on('end', () => {
                resolve(pullReferrerResponses);
            });

            call.on('error', (stream_error) => {
                console.error('Stream error:', stream_error);

                reject(stream_error);
            });

            // Send a requests and close the stream
            requests.forEach(request => {
                call.write(request, metadata);
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

    sign(req, oidc_client_id = "sigstore") {
        const oidc_provider = req.u[1][0];
        const key_provider = req.u[1][1];

        let command_result = null;

        if (oidc_provider !== undefined) {
            command_result = this.__sign_with_oidc__(req, oidc_client_id);
        }
        else if (key_provider !== undefined) {
            command_result = this.__sign_with_key__(req);
        } else {
            throw new Error("not defined provider was supplied");
        }

        return command_result;
    }

    verify(request, metadata = new grpc.Metadata()) {
        return new Promise((resolve, reject) => {
            function callback(error, response) {
                if (error) {
                    reject(error);
                } else {
                    resolve(response.u);
                }
            }

            this.signClient.verify(request, metadata, callback);
        });
    }

    __sign_with_key__(req) {
        let ref_cid = req.u[0][0];
        let key_provider = req.u[1][1];
        let tmp_key_filename = ".p.key"
        let err = null

        try {
            writeFileSync(tmp_key_filename, key_provider[0]);
        } catch (error) {
            throw error;
        }

        shell_env = process.env
        shell_env["COSIGN_PASSWORD"] = key_provider[1];

        let output = null;

        try {
            output = execSync(
                `${this.dirctl_path} sign "${ref_cid}" --key "${tmp_key_filename}"`,
                { env: { ...shell_env }, encoding: 'utf8', stdio: 'pipe' }
            );
        } catch (error) {
            err = error;
        }

        rmSync(tmp_key_filename, { force: true });

        command_result = { output: output, error: err }

        return command_result;
    }

    __sign_with_oidc__(req, oidc_client_id) {
        let ref_cid = req.u[0][0];
        let oidc_provider = req.u[1][0];
        let oidc_token = oidc_provider[0];
        let fulcio_url = oidc_provider[1][0];
        let rekor_url = oidc_provider[1][1];
        let timestamp_url = oidc_provider[1][2];
        let oidc_provider_url = oidc_provider[1][3];
        let err = null;

        shell_env = process.env

        let output = null;

        try {
            let command = `${this.dirctl_path} sign "${ref_cid}"`

            if (typeof oidc_token !== 'undefined' && oidc_token !== "") {
                command = `${command} --oidc-token "${oidc_token}"`
            }
            if (typeof oidc_provider_url !== 'undefined' && oidc_provider_url !== "") {
                command = `${command} --oidc-provider-url "${oidc_provider_url}"`
            }
            if (typeof fulcio_url !== 'undefined' && fulcio_url !== "") {
                command = `${command} --fulcio-url "${fulcio_url}"`
            }
            if (typeof rekor_url !== 'undefined' && rekor_url !== "") {
                command = `${command} --rekor-url "${rekor_url}"`
            }
            if (typeof timestamp_url !== 'undefined' && timestamp_url !== "") {
                command = `${command} --timestamp-url "${timestamp_url}"`
            }

            output = execSync(
                `${command} --oidc-client-id "${oidc_client_id}"`,
                { env: { ...shell_env }, encoding: 'utf8', stdio: 'pipe' }
            );
        } catch (error) {
            err = error;
        }

        command_result = { output: output, error: err }

        return command_result;
    }
}

module.exports = {
    Config,
    Client,
};
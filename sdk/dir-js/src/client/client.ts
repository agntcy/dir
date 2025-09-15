// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import {writeFileSync} from 'node:fs';
import {execSync} from 'node:child_process';

import {ChannelCredentials} from '@grpc/grpc-js';
import {GrpcTransport} from '@protobuf-ts/grpc-transport';

import * as models from '../models';

/**
 * Configuration class for the AGNTCY Directory client.
 *
 * This class manages configuration settings for connecting to the Directory service
 * and provides default values and environment-based configuration loading.
 */
export class Config {
  static DEFAULT_SERVER_ADDRESS = '0.0.0.0:8888';
  static DEFAULT_DIRCTL_PATH = 'dirctl';
  serverAddress: string;
  dirctlPath: string;

  /**
   * Creates a new Config instance.
   *
   * @param serverAddress - The server address to connect to. Defaults to '0.0.0.0:8888'
   * @param dirctlPath - Path to the dirctl executable. Defaults to 'dirctl'
   */
  constructor(
    serverAddress = Config.DEFAULT_SERVER_ADDRESS,
    dirctlPath = Config.DEFAULT_DIRCTL_PATH,
  ) {
    this.serverAddress = serverAddress;
    this.dirctlPath = dirctlPath;
  }

  /**
   * Load configuration from environment variables.
   *
   * @param prefix - Environment variable prefix. Defaults to 'DIRECTORY_CLIENT_'
   * @returns A new Config instance with values loaded from environment variables
   *
   * @example
   * ```typescript
   * // Load with default prefix
   * const config = Config.loadFromEnv();
   *
   * // Load with custom prefix
   * const config = Config.loadFromEnv("MY_APP_");
   * ```
   */
  static loadFromEnv(prefix = 'DIRECTORY_CLIENT_') {
    const serverAddress =
      process.env[`${prefix}SERVER_ADDRESS`] || Config.DEFAULT_SERVER_ADDRESS;
    const dirctlPath =
      process.env[`${prefix}DIRCTL_PATH`] || Config.DEFAULT_DIRCTL_PATH;

    return new Config(serverAddress, dirctlPath);
  }
}

/**
 * High-level client for interacting with AGNTCY Directory services.
 *
 * This client provides a unified interface for operations across the Directory API.
 * It handles gRPC communication and provides convenient methods for common operations
 * including storage, routing, search, signing, and synchronization.
 *
 * @example
 * ```typescript
 * // Create client with default configuration
 * const client = new Client();
 *
 * // Create client with custom configuration
 * const config = new Config('localhost:8888', '/usr/local/bin/dirctl');
 * const client = new Client(config);
 *
 * // Use client for operations
 * const records = await client.push([record]);
 * ```
 */
export class Client {
  config: Config;
  storeClient: models.store_v1.StoreServiceClient;
  routingClient: models.routing_v1.RoutingServiceClient;
  searchClient: models.search_v1.SearchServiceClient;
  signClient: models.sign_v1.SignServiceClient;
  syncClient: models.store_v1.SyncServiceClient;

  /**
   * Initialize the client with the given configuration.
   *
   * @param config - Optional client configuration. If null, loads from environment
   *                variables using Config.loadFromEnv()
   *
   * @throws {Error} If unable to establish connection to the server or configuration is invalid
   *
   * @example
   * ```typescript
   * // Load config from environment
   * const client = new Client();
   *
   * // Use custom config
   * const config = new Config('localhost:9999');
   * const client = new Client(config);
   * ```
   */
  constructor(config?: Config) {
    // Load config from environment if not provided
    if (!config) {
      config = Config.loadFromEnv();
    }
    this.config = config;

    // Create transport settings for gRPC client
    const transport = new GrpcTransport({
      host: 'localhost:5000',
      channelCredentials: ChannelCredentials.createInsecure(),
    });

    // Set clients for all services
    this.storeClient = new models.store_v1.StoreServiceClient(transport);
    this.routingClient = new models.routing_v1.RoutingServiceClient(transport);
    this.searchClient = new models.search_v1.SearchServiceClient(transport);
    this.signClient = new models.sign_v1.SignServiceClient(transport);
    this.syncClient = new models.store_v1.SyncServiceClient(transport);
  }

  /**
   * Push records to the Store API.
   *
   * Uploads one or more records to the content store, making them available
   * for retrieval and reference. Each record is assigned a unique content
   * identifier (CID) based on its content hash.
   *
   * @param records - Array of Record objects to push to the store
   * @returns Promise that resolves to an array of RecordRef objects containing the CIDs of the pushed records
   *
   * @throws {Error} If the gRPC call fails or the push operation fails
   *
   * @example
   * ```typescript
   * const records = [createRecord("example")];
   * const refs = await client.push(records);
   * console.log(`Pushed with CID: ${refs[0].cid}`);
   * ```
   */
  async push(
    records: models.core_v1.Record[],
  ): Promise<models.core_v1.RecordRef[]> {
    const call = this.storeClient.push();
    const responses: models.core_v1.RecordRef[] = [];

    // Send records
    for (const record of records) {
      await call.requests.send(record);
    }
    await call.requests.complete();

    // Collect responses
    for await (const response of call.responses) {
      responses.push(response);
    }

    return responses;
  }

  /**
   * Push records with referrer metadata to the Store API.
   *
   * Uploads records along with optional artifacts and referrer information.
   * This is useful for pushing complex objects that include additional
   * metadata or associated artifacts.
   *
   * @param requests - Array of PushReferrerRequest objects containing records and optional artifacts
   * @returns Promise that resolves to an array of PushReferrerResponse objects containing the details of pushed artifacts
   *
   * @throws {Error} If the gRPC call fails or the push operation fails
   *
   * @example
   * ```typescript
   * const requests = [new models.store_v1.PushReferrerRequest({record: record})];
   * const responses = await client.push_referrer(requests);
   * ```
   */
  async push_referrer(
    requests: models.store_v1.PushReferrerRequest[],
  ): Promise<models.store_v1.PushReferrerResponse[]> {
    const call = this.storeClient.pushReferrer();
    const responses: models.store_v1.PushReferrerResponse[] = [];

    // Send requests
    for (const request of requests) {
      await call.requests.send(request);
    }
    await call.requests.complete();

    // Collect responses
    for await (const response of call.responses) {
      responses.push(response);
    }

    return responses;
  }

  /**
   * Pull records from the Store API by their references.
   *
   * Retrieves one or more records from the content store using their
   * content identifiers (CIDs).
   *
   * @param refs - Array of RecordRef objects containing the CIDs to retrieve
   * @returns Promise that resolves to an array of Record objects retrieved from the store
   *
   * @throws {Error} If the gRPC call fails or the pull operation fails
   *
   * @example
   * ```typescript
   * const refs = [new models.core_v1.RecordRef({cid: "QmExample123"})];
   * const records = await client.pull(refs);
   * for (const record of records) {
   *   console.log(`Retrieved record: ${record}`);
   * }
   * ```
   */
  async pull(
    refs: models.core_v1.RecordRef[],
  ): Promise<models.core_v1.Record[]> {
    const call = this.storeClient.pull();
    const records: models.core_v1.Record[] = [];

    // Send requests
    for (const ref of refs) {
      await call.requests.send(ref);
    }
    await call.requests.complete();

    // Collect responses
    for await (const response of call.responses) {
      records.push(response);
    }

    return records;
  }

  /**
   * Pull records with referrer metadata from the Store API.
   *
   * Retrieves records along with their associated artifacts and referrer
   * information. This provides access to complex objects that include
   * additional metadata or associated artifacts.
   *
   * @param requests - Array of PullReferrerRequest objects containing records and optional artifacts for pull operations
   * @returns Promise that resolves to an array of PullReferrerResponse objects containing the retrieved records
   *
   * @throws {Error} If the gRPC call fails or the pull operation fails
   *
   * @example
   * ```typescript
   * const requests = [new models.store_v1.PullReferrerRequest({ref: ref})];
   * const responses = await client.pull_referrer(requests);
   * for (const response of responses) {
   *   console.log(`Retrieved: ${response}`);
   * }
   * ```
   */
  async pull_referrer(
    requests: models.store_v1.PullReferrerRequest[],
  ): Promise<models.store_v1.PullReferrerResponse[]> {
    const call = this.storeClient.pullReferrer();
    const responses: models.store_v1.PullReferrerResponse[] = [];

    // Send requests
    for (const request of requests) {
      await call.requests.send(request);
    }
    await call.requests.complete();

    // Collect responses
    for await (const response of call.responses) {
      responses.push(response);
    }

    return responses;
  }

  /**
   * Search objects from the Store API matching the specified queries.
   *
   * Performs a search across the storage using the provided search queries
   * and returns a list of matching results. Supports various
   * search types including text, semantic, and structured queries.
   *
   * @param request - SearchRequest containing queries, filters, and search options
   * @returns Promise that resolves to an array of SearchResponse objects matching the queries
   *
   * @throws {Error} If the gRPC call fails or the search operation fails
   *
   * @example
   * ```typescript
   * const request = new models.search_v1.SearchRequest({query: "python AI agent"});
   * const responses = await client.search(request);
   * for (const response of responses) {
   *   console.log(`Found: ${response.record.name}`);
   * }
   * ```
   */
  async search(
    request: models.search_v1.SearchRequest,
  ): Promise<models.search_v1.SearchResponse[]> {
    // Send requests
    const call = this.searchClient.search(request);

    // Collect responses
    const results: models.search_v1.SearchResponse[] = [];
    for await (const response of call.responses) {
      results.push(response);
    }

    return results;
  }

  /**
   * Look up metadata for records in the Store API.
   *
   * Retrieves metadata information for one or more records without
   * downloading the full record content. This is useful for checking
   * if records exist and getting basic information about them.
   *
   * @param refs - Array of RecordRef objects containing the CIDs to look up
   * @returns Promise that resolves to an array of RecordMeta objects containing metadata for the records
   *
   * @throws {Error} If the gRPC call fails or the lookup operation fails
   *
   * @example
   * ```typescript
   * const refs = [new models.core_v1.RecordRef({cid: "QmExample123"})];
   * const metadatas = await client.lookup(refs);
   * for (const meta of metadatas) {
   *   console.log(`Record size: ${meta.size}`);
   * }
   * ```
   */
  async lookup(
    refs: models.core_v1.RecordRef[],
  ): Promise<models.core_v1.RecordMeta[]> {
    const call = this.storeClient.lookup();
    const recordMetas: models.core_v1.RecordMeta[] = [];

    // Send requests
    for (const ref of refs) {
      await call.requests.send(ref);
    }
    await call.requests.complete();

    // Collect responses
    for await (const response of call.responses) {
      recordMetas.push(response);
    }

    return recordMetas;
  }

  /**
   * List objects from the Routing API matching the specified criteria.
   *
   * Returns a list of objects that match the filtering and
   * query criteria specified in the request.
   *
   * @param request - ListRequest specifying filtering criteria, pagination, etc.
   * @returns Promise that resolves to an array of ListResponse objects matching the criteria
   *
   * @throws {Error} If the gRPC call fails or the list operation fails
   *
   * @example
   * ```typescript
   * const request = new models.routing_v1.ListRequest({limit: 10});
   * const responses = await client.list(request);
   * for (const response of responses) {
   *   console.log(`Found object: ${response.cid}`);
   * }
   * ```
   */
  async list(
    request: models.routing_v1.ListRequest,
  ): Promise<models.routing_v1.ListResponse[]> {
    // Send requests
    const call = this.routingClient.list(request);

    // Collect responses
    const results: models.routing_v1.ListResponse[] = [];
    for await (const response of call.responses) {
      results.push(response);
    }

    return results;
  }

  /**
   * Publish objects to the Routing API matching the specified criteria.
   *
   * Makes the specified objects available for discovery and retrieval by other
   * clients in the network. The objects must already exist in the store before
   * they can be published.
   *
   * @param request - PublishRequest containing the query for the objects to publish
   * @returns Promise that resolves when the publish operation is complete
   *
   * @throws {Error} If the gRPC call fails or the object cannot be published
   *
   * @example
   * ```typescript
   * const ref = new models.routing_v1.RecordRef({cid: "QmExample123"});
   * const request = new models.routing_v1.PublishRequest({recordRefs: [ref]});
   * await client.publish(request);
   * ```
   */
  async publish(request: models.routing_v1.PublishRequest): Promise<void> {
    await this.routingClient.publish(request);
  }

  /**
   * Unpublish objects from the Routing API matching the specified criteria.
   *
   * Removes the specified objects from the public network, making them no
   * longer discoverable by other clients. The objects remain in the local
   * store but are not available for network discovery.
   *
   * @param request - UnpublishRequest containing the query for the objects to unpublish
   * @returns Promise that resolves when the unpublish operation is complete
   *
   * @throws {Error} If the gRPC call fails or the objects cannot be unpublished
   *
   * @example
   * ```typescript
   * const ref = new models.routing_v1.RecordRef({cid: "QmExample123"});
   * const request = new models.routing_v1.UnpublishRequest({recordRefs: [ref]});
   * await client.unpublish(request);
   * ```
   */
  async unpublish(request: models.routing_v1.UnpublishRequest): Promise<void> {
    await this.routingClient.unpublish(request);
  }

  /**
   * Delete records from the Store API.
   *
   * Permanently removes one or more records from the content store using
   * their content identifiers (CIDs). This operation cannot be undone.
   *
   * @param refs - Array of RecordRef objects containing the CIDs to delete
   * @returns Promise that resolves when the deletion is complete
   *
   * @throws {Error} If the gRPC call fails or the delete operation fails
   *
   * @example
   * ```typescript
   * const refs = [new models.core_v1.RecordRef({cid: "QmExample123"})];
   * await client.delete(refs);
   * ```
   */
  async delete(refs: models.core_v1.RecordRef[]): Promise<void> {
    // Send requests
    const call = this.storeClient.delete();
    for (const ref of refs) {
      await call.requests.send(ref);
    }
    await call.requests.complete();

    // Wait for completion (no response data expected)
    await call;
  }

  /**
   * Sign a record with a cryptographic signature.
   *
   * Creates a cryptographic signature for a record using either a private
   * key or OIDC-based signing. The signing process uses the external dirctl
   * command-line tool to perform the actual cryptographic operations.
   *
   * @param req - SignRequest containing the record reference and signing provider
   *              configuration. The provider can specify either key-based signing
   *              (with a private key) or OIDC-based signing
   * @param oidc_client_id - OIDC client identifier for OIDC-based signing. Defaults to "sigstore"
   * @returns SignResponse containing the signature
   *
   * @throws {Error} If the signing operation fails or unsupported provider is supplied
   *
   * @example
   * ```typescript
   * const req = new models.sign_v1.SignRequest({
   *   recordRef: new models.core_v1.RecordRef({cid: "QmExample123"}),
   *   provider: new models.sign_v1.SignProvider({key: keyConfig})
   * });
   * const response = client.sign(req);
   * console.log(`Signature: ${response.signature}`);
   * ```
   */
  sign(req: models.sign_v1.SignRequest, oidc_client_id = 'sigstore') {
    switch (req.provider?.request.oneofKind) {
      case 'oidc':
        return this.__sign_with_oidc(
          req.recordRef?.cid || '',
          req.provider.request.oidc,
          oidc_client_id,
        );

      case 'key':
        return this.__sign_with_key(
          req.recordRef?.cid || '',
          req.provider.request.key,
        );

      default:
        throw new Error('unsupported provider was supplied');
    }
  }

  /**
   * Verify a cryptographic signature on a record.
   *
   * Validates the cryptographic signature of a previously signed record
   * to ensure its authenticity and integrity. This operation verifies
   * that the record has not been tampered with since signing.
   *
   * @param request - VerifyRequest containing the record reference and verification parameters
   * @returns Promise that resolves to a VerifyResponse containing the verification result and details
   *
   * @throws {Error} If the gRPC call fails or the verification operation fails
   *
   * @example
   * ```typescript
   * const request = new models.sign_v1.VerifyRequest({
   *   recordRef: new models.core_v1.RecordRef({cid: "QmExample123"})
   * });
   * const response = await client.verify(request);
   * console.log(`Signature valid: ${response.valid}`);
   * ```
   */
  async verify(
    request: models.sign_v1.VerifyRequest,
  ): Promise<models.sign_v1.VerifyResponse> {
    const response = await this.signClient.verify(request);
    return response.response;
  }

  /**
   * Create a new synchronization configuration.
   *
   * Creates a new sync configuration that defines how data should be
   * synchronized between different Directory servers. This allows for
   * automated data replication and consistency across multiple locations.
   *
   * @param request - CreateSyncRequest containing the sync configuration details
   *                  including source, target, and synchronization parameters
   * @returns Promise that resolves to a CreateSyncResponse containing the created sync details
   *          including the sync ID and configuration
   *
   * @throws {Error} If the gRPC call fails or the sync creation fails
   *
   * @example
   * ```typescript
   * const request = new models.store_v1.CreateSyncRequest();
   * const response = await client.create_sync(request);
   * console.log(`Created sync with ID: ${response.syncId}`);
   * ```
   */
  async create_sync(
    request: models.store_v1.CreateSyncRequest,
  ): Promise<models.store_v1.CreateSyncResponse> {
    const response = await this.syncClient.createSync(request);
    return response.response;
  }

  /**
   * List existing synchronization configurations.
   *
   * Retrieves a list of all sync configurations that have been created,
   * with optional filtering and pagination support. This allows you to
   * monitor and manage multiple synchronization processes.
   *
   * @param request - ListSyncsRequest containing filtering criteria, pagination options,
   *                  and other query parameters
   * @returns Promise that resolves to an array of ListSyncsItem objects with
   *          their details including ID, name, status, and configuration parameters
   *
   * @throws {Error} If the gRPC call fails or the list operation fails
   *
   * @example
   * ```typescript
   * const request = new models.store_v1.ListSyncsRequest({limit: 10});
   * const syncs = await client.list_syncs(request);
   * for (const sync of syncs) {
   *   console.log(`Sync: ${sync}`);
   * }
   * ```
   */
  async list_syncs(
    request: models.store_v1.ListSyncsRequest,
  ): Promise<models.store_v1.ListSyncsItem[]> {
    // Send requests
    const call = this.syncClient.listSyncs(request);

    // Collect responses
    const results: models.store_v1.ListSyncsItem[] = [];
    for await (const response of call.responses) {
      results.push(response);
    }

    return results;
  }

  /**
   * Retrieve detailed information about a specific synchronization configuration.
   *
   * Gets comprehensive details about a specific sync configuration including
   * its current status, configuration parameters, performance metrics,
   * and any recent errors or warnings.
   *
   * @param request - GetSyncRequest containing the sync ID or identifier to retrieve
   * @returns Promise that resolves to a GetSyncResponse with detailed information about the sync configuration
   *          including status, metrics, configuration, and logs
   *
   * @throws {Error} If the gRPC call fails or the get operation fails
   *
   * @example
   * ```typescript
   * const request = new models.store_v1.GetSyncRequest({syncId: "sync-123"});
   * const response = await client.get_sync(request);
   * console.log(`Sync status: ${response.status}`);
   * console.log(`Last update: ${response.lastUpdateTime}`);
   * ```
   */
  async get_sync(
    request: models.store_v1.GetSyncRequest,
  ): Promise<models.store_v1.GetSyncResponse> {
    const response = await this.syncClient.getSync(request);
    return response.response;
  }

  /**
   * Delete a synchronization configuration.
   *
   * Permanently removes a sync configuration and stops any ongoing
   * synchronization processes. This operation cannot be undone and
   * will halt all data synchronization for the specified configuration.
   *
   * @param request - DeleteSyncRequest containing the sync ID or identifier to delete
   * @returns Promise that resolves to a DeleteSyncResponse when the deletion is complete
   *
   * @throws {Error} If the gRPC call fails or the delete operation fails
   *
   * @example
   * ```typescript
   * const request = new models.store_v1.DeleteSyncRequest({syncId: "sync-123"});
   * await client.delete_sync(request);
   * console.log("Sync deleted");
   * ```
   */
  async delete_sync(
    request: models.store_v1.DeleteSyncRequest,
  ): Promise<models.store_v1.DeleteSyncResponse> {
    const response = await this.syncClient.deleteSync(request);
    return response;
  }

  /**
   * Sign a record using a private key.
   *
   * This private method handles key-based signing by writing the private key
   * to a temporary file and executing the dirctl command with the key file.
   *
   * @param cid - Content identifier of the record to sign
   * @param req - SignWithKey request containing the private key
   * @returns SignResponse containing the signature
   *
   * @throws {Error} If any error occurs during signing
   *
   * @private
   */
  __sign_with_key(
    cid: string,
    req: models.sign_v1.SignWithKey,
  ): models.sign_v1.SignResponse {
    // Write private key to a temporary file
    const tmp_key_filename = path.join(os.tmpdir(), '.p.key');
    writeFileSync(tmp_key_filename, String(req.privateKey));

    // Prepare environment for signing command
    const shell_env = process.env;
    shell_env['COSIGN_PASSWORD'] = String(req.privateKey);

    // Execute signing command
    const output = execSync(
      `${this.config.dirctlPath} sign "${cid}" --key "${tmp_key_filename}"`,
      {env: {...shell_env}, encoding: 'utf8', stdio: 'pipe'},
    );

    // Return signature
    return {
      signature: JSON.parse(output.trim()) as models.sign_v1.Signature,
    };
  }

  /**
   * Sign a record using OIDC-based authentication.
   *
   * This private method handles OIDC-based signing by building the appropriate
   * dirctl command with OIDC parameters and executing it.
   *
   * @param cid - Content identifier of the record to sign
   * @param req - SignWithOIDC request containing the OIDC configuration
   * @param oidc_client_id - OIDC client identifier for authentication
   * @returns SignResponse containing the signature
   *
   * @throws {Error} If any error occurs during signing
   *
   * @private
   */
  __sign_with_oidc(
    cid: string,
    req: models.sign_v1.SignWithOIDC,
    oidc_client_id: string,
  ) {
    // Prepare command for signing
    let command = `${this.config.dirctlPath} sign "${cid}"`;
    if (req.idToken !== '') {
      command = `${command} --oidc-token "${req.idToken}"`;
    }
    if (req.options?.oidcProviderUrl !== '') {
      command = `${command} --oidc-provider-url "${req.options?.oidcProviderUrl}"`;
    }
    if (req.options?.fulcioUrl !== '') {
      command = `${command} --fulcio-url "${req.options?.fulcioUrl}"`;
    }
    if (req.options?.rekorUrl !== '') {
      command = `${command} --rekor-url "${req.options?.rekorUrl}"`;
    }
    if (req.options?.timestampUrl !== '') {
      command = `${command} --timestamp-url "${req.options?.timestampUrl}"`;
    }

    // Execute signing command
    const output = execSync(`${command} --oidc-client-id "${oidc_client_id}"`, {
      env: {...process.env},
      encoding: 'utf8',
      stdio: 'pipe',
    });

    // Return signature
    return {
      signature: JSON.parse(output.trim()) as models.sign_v1.Signature,
    };
  }
}

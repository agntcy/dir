// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { env } from 'node:process';
import { readFileSync, writeFileSync } from 'node:fs';
import { spawnSync, SpawnSyncReturns } from 'node:child_process';

import {
  Client as GrpcClient,
  createClient,
  Interceptor,
  Transport,
} from '@connectrpc/connect';
import { createGrpcTransport } from '@connectrpc/connect-node';
import { createClient as createClientSpiffe, X509SVID } from 'spiffe';
import * as models from '../models';

/**
 * Configuration class for the AGNTCY Directory client.
 *
 * This class manages configuration settings for connecting to the Directory service
 * and provides default values and environment-based configuration loading.
 */
export class Config {
  static DEFAULT_SERVER_ADDRESS = '127.0.0.1:8888';
  static DEFAULT_DIRCTL_PATH = 'dirctl';
  static DEFAULT_SPIFFE_ENDPOINT_SOCKET = '';
  static DEFAULT_AUTH_MODE = '';
  static DEFAULT_JWT_AUDIENCE = '';
  static DEFAULT_TLS_CA_FILE = '';
  static DEFAULT_TLS_CERT_FILE = '';
  static DEFAULT_TLS_KEY_FILE = '';

  serverAddress: string;
  dirctlPath: string;
  spiffeEndpointSocket: string;
  authMode: '' | 'x509' | 'jwt' | 'tls';
  jwtAudience: string;
  tlsCaFile: string;
  tlsCertFile: string
  tlsKeyFile: string;

  /**
   * Creates a new Config instance.
   *
   * @param serverAddress - The server address to connect to. Defaults to '127.0.0.1:8888'
   * @param dirctlPath - Path to the dirctl executable. Defaults to 'dirctl'
   * @param spiffeEndpointSocket - Path to the spire server socket. Defaults to empty string.
   * @param authMode - Authentication mode: '' for insecure, 'x509', 'jwt' or 'tls'. Defaults to ''
   * @param jwtAudience - JWT audience for JWT authentication. Required when authMode is 'jwt'
   */
  constructor(
    serverAddress = Config.DEFAULT_SERVER_ADDRESS,
    dirctlPath = Config.DEFAULT_DIRCTL_PATH,
    spiffeEndpointSocket = Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET,
    authMode: '' | 'x509' | 'jwt' | 'tls' = Config.DEFAULT_AUTH_MODE as '' | 'x509' | 'jwt' | 'tls',
    jwtAudience = Config.DEFAULT_JWT_AUDIENCE,
    tlsCaFile = Config.DEFAULT_TLS_CA_FILE,
    tlsCertFile = Config.DEFAULT_TLS_CERT_FILE,
    tlsKeyFile = Config.DEFAULT_TLS_KEY_FILE
  ) {
    // add protocol prefix if not set
    // use unsafe http unless spire/auth is used
    if (
      !serverAddress.startsWith('http://') &&
      !serverAddress.startsWith('https://')
    ) {
      // use https protocol when X.509, JWT, or TLS auth is used
      if (authMode === 'x509' || authMode === 'jwt' || authMode === 'tls') {
        serverAddress = `https://${serverAddress}`;
      } else {
        serverAddress = `http://${serverAddress}`;
      }
    }

    this.serverAddress = serverAddress;
    this.dirctlPath = dirctlPath;
    this.spiffeEndpointSocket = spiffeEndpointSocket;
    this.authMode = authMode;
    this.jwtAudience = jwtAudience;
    this.tlsCaFile = tlsCaFile;
    this.tlsCertFile = tlsCertFile;
    this.tlsKeyFile = tlsKeyFile;
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
    // Load dirctl path from env without env prefix
    const dirctlPath = env['DIRCTL_PATH'] || Config.DEFAULT_DIRCTL_PATH;

    // Load other config values with env prefix
    const serverAddress =
      env[`${prefix}SERVER_ADDRESS`] || Config.DEFAULT_SERVER_ADDRESS;
    const spiffeEndpointSocketPath = env[`${prefix}SPIFFE_SOCKET_PATH`] || Config.DEFAULT_SPIFFE_ENDPOINT_SOCKET;
    const authMode = (env[`${prefix}AUTH_MODE`] || Config.DEFAULT_AUTH_MODE) as '' | 'x509' | 'jwt' | 'tls';
    const jwtAudience = env[`${prefix}JWT_AUDIENCE`] || Config.DEFAULT_JWT_AUDIENCE;
    const tlsCaFile = env[`${prefix}TLS_CA_FILE`] || Config.DEFAULT_TLS_CA_FILE;
    const tlsCertFile = env[`${prefix}TLS_CERT_FILE`] || Config.DEFAULT_TLS_CERT_FILE;
    const tlsKeyFile = env[`${prefix}TLS_KEY_FILE`] || Config.DEFAULT_TLS_KEY_FILE;

    return new Config(serverAddress, dirctlPath, spiffeEndpointSocketPath, authMode, jwtAudience, tlsCaFile, tlsCertFile, tlsKeyFile);
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
  storeClient: GrpcClient<typeof models.store_v1.StoreService>;
  routingClient: GrpcClient<typeof models.routing_v1.RoutingService>;
  searchClient: GrpcClient<typeof models.search_v1.SearchService>;
  signClient: GrpcClient<typeof models.sign_v1.SignService>;
  syncClient: GrpcClient<typeof models.store_v1.SyncService>;
  eventClient: GrpcClient<typeof models.events_v1.EventService>;

  /**
   * Initialize the client with the given configuration.
   *
   * @param config - Optional client configuration. If null, loads from environment
   *                variables using Config.loadFromEnv()
   * @param grpcTransport - Optional transport to use for gRPC communication.
   *                Can be created with Client.createGRPCTransport(config)
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
   * const grpcTransport = await Client.createGRPCTransport(config);
   * const client = new Client(config, grpcTransport);
   * ```
   */
  constructor();
  constructor(config?: Config);
  constructor(config?: Config, grpcTransport?: Transport);
  constructor(config?: Config, grpcTransport?: Transport) {
    // Load config from environment if not provided
    if (!config) {
      config = Config.loadFromEnv();
    }
    this.config = config;

    // if no transport provided, use insecure transport
    if (!grpcTransport) {
      grpcTransport = createGrpcTransport({
        baseUrl: config.serverAddress,
      });
    }

    // Set clients for all services
    this.storeClient = createClient(models.store_v1.StoreService, grpcTransport);
    this.routingClient = createClient(
      models.routing_v1.RoutingService,
      grpcTransport,
    );
    this.searchClient = createClient(models.search_v1.SearchService, grpcTransport);
    this.signClient = createClient(models.sign_v1.SignService, grpcTransport);
    this.syncClient = createClient(models.store_v1.SyncService, grpcTransport);
    this.eventClient = createClient(models.events_v1.EventService, grpcTransport);
  }

  private static convertToPEM(bytes: Uint8Array, label: string): string {
    // Convert Uint8Array to base64 string
    let binary = '';
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    const base64String = btoa(binary);

    // Split base64 string into 64-character lines
    const lines = base64String.match(/.{1,64}/g) || [];

    // Build PEM formatted string with headers and footers
    const pem = [
      `-----BEGIN ${label}-----`,
      ...lines,
      `-----END ${label}-----`
    ].join('\n');

    return pem;
  }

  static async createGRPCTransport(config: Config): Promise<Transport> {
    // Handle different authentication modes
    switch (config.authMode) {
      case '':
        return createGrpcTransport({
          baseUrl: config.serverAddress,
        });

      case 'jwt':
        return await this.createJWTTransport(config);

      case 'x509':
        return await this.createX509Transport(config);

      case 'tls':
        return await this.createTLSTransport(config);

      default:
        throw new Error(`Unsupported auth mode: ${config.authMode}`);
    }
  }

  private static async createX509Transport(config: Config): Promise<Transport> {
    if (config.spiffeEndpointSocket === '') {
      throw new Error('SPIFFE socket path is required for X.509 authentication');
    }

    // Create secure transport with SPIFFE X.509
    const client = createClientSpiffe(config.spiffeEndpointSocket);

    let svid: X509SVID = {
      spiffeId: '',
      hint: '',
      x509Svid: new Uint8Array(),
      x509SvidKey: new Uint8Array(),
      bundle: new Uint8Array(),
    };

    const svidStream = client.fetchX509SVID({});
    for await (const message of svidStream.responses) {
      message.svids.forEach((_svid) => {
        svid = _svid;
      })

      if (message.svids.length > 0) {
        break
      }
    }

    // Create transport settings for gRPC client
    const transport = createGrpcTransport({
      baseUrl: config.serverAddress,
      nodeOptions: {
        ca: this.convertToPEM(svid.bundle, "TRUSTED CERTIFICATE"),
        cert: this.convertToPEM(svid.x509Svid, "CERTIFICATE"),
        key: this.convertToPEM(svid.x509SvidKey, "PRIVATE KEY"),
      },
    });

    return transport;
  }

  private static async createJWTTransport(config: Config): Promise<Transport> {
    if (config.spiffeEndpointSocket === '') {
      throw new Error('SPIFFE socket path is required for JWT authentication');
    }

    if (config.jwtAudience === '') {
      throw new Error('JWT audience is required for JWT authentication');
    }

    // Create SPIFFE client
    const client = createClientSpiffe(config.spiffeEndpointSocket);

    // Fetch X.509 bundle for verifying server's TLS certificate
    // In JWT mode, the server presents its X.509-SVID via TLS for transport security
    let bundle: Uint8Array | null = null;
    const bundleStream = client.fetchX509Bundles({});
    for await (const message of bundleStream.responses) {
      // Get the first bundle from the bundles map
      // bundles is a map<string, bytes> where bytes is ASN.1 DER encoded
      for (const [_, bundleData] of Object.entries(message.bundles)) {
        // Convert to a new Uint8Array to ensure type compatibility
        bundle = new Uint8Array(bundleData);
        break;
      }
      if (bundle !== null) {
        break;
      }
    }

    if (bundle === null || bundle.length === 0) {
      throw new Error('Failed to fetch X.509 bundle from SPIRE: no bundles returned');
    }

    // Create JWT interceptor that fetches and injects JWT tokens
    const jwtInterceptor: Interceptor = (next) => async (req) => {
      // Fetch JWT-SVID from SPIRE
      // Note: spiffeId is empty string to use the workload's default identity
      const jwtCall = client.fetchJWTSVID({
        spiffeId: '',
        audience: [config.jwtAudience]
      });

      const response = await jwtCall.response;

      if (!response.svids || response.svids.length === 0) {
        throw new Error('Failed to fetch JWT-SVID from SPIRE: no SVIDs returned');
      }

      const jwtToken = response.svids[0].svid;

      // Add JWT token to request headers
      req.header.set('authorization', `Bearer ${jwtToken}`);

      return await next(req);
    };

    // Create transport with JWT interceptor and TLS using SPIFFE bundle
    // For JWT mode: Server presents X.509-SVID via TLS, clients authenticate with JWT-SVID
    const transport = createGrpcTransport({
      baseUrl: config.serverAddress,
      interceptors: [jwtInterceptor],
      nodeOptions: {
        ca: this.convertToPEM(bundle, "CERTIFICATE"),
      },
    });

    return transport;
  }

  private static async createTLSTransport(config: Config): Promise<Transport> {
    if (config.tlsCaFile === '') {
      throw new Error('TLS CA file is required for TLS authentication');
    }
    if (config.tlsCertFile === '') {
      throw new Error('TLS certificate file is required for TLS authentication');
    }
    if (config.tlsKeyFile === '') {
      throw new Error('TLS key file is required for TLS authentication');
    }

    let root_ca: string;
    let cert_chain: string;
    let private_key: string;

    try {
      root_ca = readFileSync(config.tlsCaFile).toString();
      cert_chain = readFileSync(config.tlsCertFile).toString();
      private_key = readFileSync(config.tlsKeyFile).toString();
    } catch (e) {
      console.error('Error reading file:', (e as Error).message);
      throw e;
    }

    const transport = createGrpcTransport({
      baseUrl: config.serverAddress,
      nodeOptions: {
        ca: root_ca,
        cert: cert_chain,
        key: private_key,
      },
    });

    return transport;
  }
  /**
   * Request generator helper function for streaming requests.
   */
  private async *requestGenerator<T>(reqs: T[]): AsyncIterable<T> {
    for (const req of reqs) {
      yield req;
    }
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
    const responses: models.core_v1.RecordRef[] = [];

    for await (const response of this.storeClient.push(
      this.requestGenerator(records),
    )) {
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
    const responses: models.store_v1.PushReferrerResponse[] = [];

    for await (const response of this.storeClient.pushReferrer(
      this.requestGenerator(requests),
    )) {
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
    const records: models.core_v1.Record[] = [];

    for await (const response of this.storeClient.pull(
      this.requestGenerator(refs),
    )) {
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
    const responses: models.store_v1.PullReferrerResponse[] = [];

    for await (const response of this.storeClient.pullReferrer(
      this.requestGenerator(requests),
    )) {
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
    const responses: models.search_v1.SearchResponse[] = [];

    for await (const response of this.searchClient.search(request)) {
      responses.push(response);
    }

    return responses;
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
    const recordMetas: models.core_v1.RecordMeta[] = [];

    for await (const response of this.storeClient.lookup(
      this.requestGenerator(refs),
    )) {
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
    const results: models.routing_v1.ListResponse[] = [];

    for await (const response of this.routingClient.list(request)) {
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
    await this.storeClient.delete(this.requestGenerator(refs));
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
  sign(req: models.sign_v1.SignRequest, oidc_client_id = 'sigstore'): void {

    var output;

    switch (req.provider?.request.case) {
      case 'oidc':
        output = this.__sign_with_oidc(
          req.recordRef?.cid || '',
          req.provider.request.value,
          oidc_client_id,
        );
        break;

      case 'key':
        output = this.__sign_with_key(
          req.recordRef?.cid || '',
          req.provider.request.value,
        );
        break;

      default:
        throw new Error('unsupported provider was supplied');
    }

    if (output.status !== 0) {
      throw output.error;
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
    return await this.signClient.verify(request);
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
    return await this.syncClient.createSync(request);
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
    const results: models.store_v1.ListSyncsItem[] = [];

    for await (const response of this.syncClient.listSyncs(request)) {
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
    return await this.syncClient.getSync(request);
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
    return await this.syncClient.deleteSync(request);
  }

  /**
   * Get events from the Event API matching the specified criteria.
   * 
   * Retrieves a list of events that match the filtering and query criteria
   * specified in the request.
   * 
   * @param request - ListenRequest specifying filtering criteria, pagination, etc.
   * @returns Promise that resolves to an array of ListenResponse objects matching the criteria
   * 
   * @throws {Error} If the gRPC call fails or the get events operation fails
   */
  listen(
    request: models.events_v1.ListenRequest
  ): AsyncIterable<models.events_v1.ListenResponse> {
    return this.eventClient.listen(request);
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
  private __sign_with_key(cid: string, req: models.sign_v1.SignWithKey): SpawnSyncReturns<string> {
    // Write private key to a temporary file
    const tmp_key_filename = join(tmpdir(), '.p.key');
    writeFileSync(tmp_key_filename, String(req.privateKey));

    // Prepare environment for command
    const shell_env = env;
    shell_env['COSIGN_PASSWORD'] = String(req.password);

    let commandArgs = ["sign", cid, "--key", tmp_key_filename];

    if (this.config.spiffeEndpointSocket !== '') {
      commandArgs.push(...["--spiffe-socket-path", this.config.spiffeEndpointSocket]);
    }

    // Execute command
    let output = spawnSync(
      `${this.config.dirctlPath}`, commandArgs,
      { env: { ...shell_env }, encoding: 'utf8', stdio: 'pipe' },
    );

    return output;
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
  private __sign_with_oidc(
    cid: string,
    req: models.sign_v1.SignWithOIDC,
    oidc_client_id: string,
  ): SpawnSyncReturns<string> {
    // Prepare command
    let commandArgs = ["sign", cid];
    if (req.idToken !== '') {
      commandArgs.push(...["--oidc-token", req.idToken]);
    }
    if (
      req.options?.oidcProviderUrl !== undefined &&
      req.options.oidcProviderUrl !== ''
    ) {
      commandArgs.push(...["--oidc-provider-url", req.options.oidcProviderUrl]);
    }
    if (req.options?.fulcioUrl !== undefined && req.options.fulcioUrl !== '') {
      commandArgs.push(...["--fulcio-url", req.options.fulcioUrl]);
    }
    if (req.options?.rekorUrl !== undefined && req.options.rekorUrl !== '') {
      commandArgs.push(...["--rekor-url", req.options.rekorUrl]);
    }
    if (
      req.options?.timestampUrl !== undefined &&
      req.options.timestampUrl !== ''
    ) {
      commandArgs.push(...["--timestamp-url", req.options.timestampUrl]);
    }

    if (this.config.spiffeEndpointSocket !== '') {
      commandArgs.push(...["--spiffe-socket-path", this.config.spiffeEndpointSocket]);
    }

    // Execute command
    let output = spawnSync(`${this.config.dirctlPath}`, commandArgs, {
      env: { ...env },
      encoding: 'utf8',
      stdio: 'pipe',
    });

    return output;
  }
}

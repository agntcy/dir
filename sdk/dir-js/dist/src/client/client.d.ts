import { Client as GrpcClient, Transport } from '@connectrpc/connect';
import * as models from '../models';
/**
 * Configuration class for the AGNTCY Directory client.
 *
 * This class manages configuration settings for connecting to the Directory service
 * and provides default values and environment-based configuration loading.
 */
export declare class Config {
    static DEFAULT_SERVER_ADDRESS: string;
    static DEFAULT_DIRCTL_PATH: string;
    static DEFAULT_SPIFFE_ENDPOINT_SOCKET: string;
    static DEFAULT_AUTH_MODE: string;
    static DEFAULT_JWT_AUDIENCE: string;
    static DEFAULT_TLS_CA_FILE: string;
    static DEFAULT_TLS_CERT_FILE: string;
    static DEFAULT_TLS_KEY_FILE: string;
    serverAddress: string;
    dirctlPath: string;
    spiffeEndpointSocket: string;
    authMode: '' | 'x509' | 'jwt' | 'tls';
    jwtAudience: string;
    tlsCaFile: string;
    tlsCertFile: string;
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
    constructor(serverAddress?: string, dirctlPath?: string, spiffeEndpointSocket?: string, authMode?: '' | 'x509' | 'jwt' | 'tls', jwtAudience?: string, tlsCaFile?: string, tlsCertFile?: string, tlsKeyFile?: string);
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
    static loadFromEnv(prefix?: string): Config;
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
export declare class Client {
    config: Config;
    storeClient: GrpcClient<typeof models.store_v1.StoreService>;
    routingClient: GrpcClient<typeof models.routing_v1.RoutingService>;
    publicationClient: GrpcClient<typeof models.routing_v1.PublicationService>;
    searchClient: GrpcClient<typeof models.search_v1.SearchService>;
    signClient: GrpcClient<typeof models.sign_v1.SignService>;
    syncClient: GrpcClient<typeof models.store_v1.SyncService>;
    eventClient: GrpcClient<typeof models.events_v1.EventService>;
    namingClient: GrpcClient<typeof models.naming_v1.NamingService>;
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
    private static convertToPEM;
    static createGRPCTransport(config: Config): Promise<Transport>;
    private static createX509Transport;
    private static createJWTTransport;
    private static createTLSTransport;
    /**
     * Request generator helper function for streaming requests.
     */
    private requestGenerator;
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
    push(records: models.core_v1.Record[]): Promise<models.core_v1.RecordRef[]>;
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
    push_referrer(requests: models.store_v1.PushReferrerRequest[]): Promise<models.store_v1.PushReferrerResponse[]>;
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
    pull(refs: models.core_v1.RecordRef[]): Promise<models.core_v1.Record[]>;
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
    pull_referrer(requests: models.store_v1.PullReferrerRequest[]): Promise<models.store_v1.PullReferrerResponse[]>;
    /**
     * Search objects from the Store API matching the specified queries.
     *
     * Performs a search across the storage using the provided search queries
     * and returns a list of matching CIDs. This is efficient for lookups
     * where only the CIDs are needed.
     *
     * @param request - SearchCIDsRequest containing queries, filters, and search options
     * @returns Promise that resolves to an array of SearchCIDsResponse objects matching the queries
     *
     * @throws {Error} If the gRPC call fails or the search operation fails
     *
     * @example
     * ```typescript
     * const request = create(models.search_v1.SearchCIDsRequestSchema, {queries: [query], limit: 10});
     * const responses = await client.searchCIDs(request);
     * for (const response of responses) {
     *   console.log(`Found CID: ${response.recordCid}`);
     * }
     * ```
     */
    searchCIDs(request: models.search_v1.SearchCIDsRequest): Promise<models.search_v1.SearchCIDsResponse[]>;
    /**
     * Search for full records from the Store API matching the specified queries.
     *
     * Performs a search across the storage using the provided search queries
     * and returns a list of full records with all metadata.
     *
     * @param request - SearchRecordsRequest containing queries, filters, and search options
     * @returns Promise that resolves to an array of SearchRecordsResponse objects matching the queries
     *
     * @throws {Error} If the gRPC call fails or the search operation fails
     *
     * @example
     * ```typescript
     * const request = create(models.search_v1.SearchRecordsRequestSchema, {queries: [query], limit: 10});
     * const responses = await client.searchRecords(request);
     * for (const response of responses) {
     *   console.log(`Found: ${response.record?.name}`);
     * }
     * ```
     */
    searchRecords(request: models.search_v1.SearchRecordsRequest): Promise<models.search_v1.SearchRecordsResponse[]>;
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
    lookup(refs: models.core_v1.RecordRef[]): Promise<models.core_v1.RecordMeta[]>;
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
    list(request: models.routing_v1.ListRequest): Promise<models.routing_v1.ListResponse[]>;
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
    publish(request: models.routing_v1.PublishRequest): Promise<void>;
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
    unpublish(request: models.routing_v1.UnpublishRequest): Promise<void>;
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
    delete(refs: models.core_v1.RecordRef[]): Promise<void>;
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
    sign(req: models.sign_v1.SignRequest, oidc_client_id?: string): void;
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
    verify(request: models.sign_v1.VerifyRequest): Promise<models.sign_v1.VerifyResponse>;
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
    create_sync(request: models.store_v1.CreateSyncRequest): Promise<models.store_v1.CreateSyncResponse>;
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
    list_syncs(request: models.store_v1.ListSyncsRequest): Promise<models.store_v1.ListSyncsItem[]>;
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
    get_sync(request: models.store_v1.GetSyncRequest): Promise<models.store_v1.GetSyncResponse>;
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
    delete_sync(request: models.store_v1.DeleteSyncRequest): Promise<models.store_v1.DeleteSyncResponse>;
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
    listen(request: models.events_v1.ListenRequest): AsyncIterable<models.events_v1.ListenResponse>;
    /**
     * CreatePublication creates a new publication request that will be processed by the PublicationWorker.
     * The publication request can specify either a query, a list of specific CIDs,
     * or all records to be announced to the DHT.
     *
     * @param request - PublishRequest containing record references and queries options.
     *
     * @returns CreatePublicationResponse returns the result of creating a publication request.
     * This includes the publication ID and any relevant metadata.
     *
     * @throws {Error} If the gRPC call fails or the list operation fails
     */
    create_publication(request: models.routing_v1.PublishRequest): Promise<models.routing_v1.CreatePublicationResponse>;
    /**
     * ListPublications returns a stream of all publication requests in the system.
     * This allows monitoring of pending, processing, and completed publication requests.
     *
     * @param request - ListPublicationsRequest contains optional filters for listing publication requests.
     *
     * @returns Promise that resolves to an array of ListPublicationsItem represents
     * a single publication request in the list response.
     * Contains publication details including ID, status, and creation timestamp.
     *
     * @throws {Error} If the gRPC call fails or the list operation fails
     */
    list_publication(request: models.routing_v1.ListPublicationsRequest): Promise<models.routing_v1.ListPublicationsItem[]>;
    /**
     * GetPublication retrieves details of a specific publication request by its identifier.
     * This includes the current status and any associated metadata.
     *
     * @param request - GetPublicationRequest specifies which publication to retrieve by its identifier.
     *
     * @returns GetPublicationResponse contains the full details of a specific publication request.
     * Includes status, progress information, and any error details if applicable.
     *
     * @throws {Error} If the gRPC call fails or the get operation fails
     */
    get_publication(request: models.routing_v1.GetPublicationRequest): Promise<models.routing_v1.GetPublicationResponse>;
    /**
     * Resolve a record name to CIDs.
     *
     * Resolves a record reference (name with optional version) to content identifiers (CIDs).
     * When no version is specified, returns all versions sorted by creation time (newest first).
     *
     * @param request - ResolveRequest containing the name and optional version
     * @returns Promise that resolves to a ResolveResponse containing the resolved record references
     *
     * @throws {Error} If the gRPC call fails or the resolve operation fails
     *
     * @example
     * ```typescript
     * import { create } from "@bufbuild/protobuf";
     *
     * // Resolve latest version
     * const request = create(models.naming_v1.ResolveRequestSchema, { name: "cisco.com/agent" });
     * const response = await client.resolve(request);
     * console.log(`Latest CID: ${response.records[0].cid}`);
     *
     * // Resolve specific version
     * const request = create(models.naming_v1.ResolveRequestSchema, { name: "cisco.com/agent", version: "v1.0.0" });
     * const response = await client.resolve(request);
     * ```
     */
    resolve(request: models.naming_v1.ResolveRequest): Promise<models.naming_v1.ResolveResponse>;
    /**
     * Get verification info for a record.
     *
     * Retrieves the name verification status for a record. Can look up by CID directly
     * or by name (with optional version) which will be resolved first.
     *
     * @param request - GetVerificationInfoRequest containing cid, name, and/or version
     * @returns Promise that resolves to a GetVerificationInfoResponse containing verification status
     *
     * @throws {Error} If the gRPC call fails or the operation fails
     *
     * @example
     * ```typescript
     * import { create } from "@bufbuild/protobuf";
     *
     * // Check by CID
     * const request = create(models.naming_v1.GetVerificationInfoRequestSchema, { cid: "bafyreib..." });
     * const response = await client.getVerificationInfo(request);
     *
     * // Check by name (latest version)
     * const request = create(models.naming_v1.GetVerificationInfoRequestSchema, { name: "cisco.com/agent" });
     * const response = await client.getVerificationInfo(request);
     *
     * // Check by name with specific version
     * const request = create(models.naming_v1.GetVerificationInfoRequestSchema, { name: "cisco.com/agent", version: "v1.0.0" });
     * const response = await client.getVerificationInfo(request);
     * ```
     */
    getVerificationInfo(request: models.naming_v1.GetVerificationInfoRequest): Promise<models.naming_v1.GetVerificationInfoResponse>;
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
    private __sign_with_key;
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
    private __sign_with_oidc;
}

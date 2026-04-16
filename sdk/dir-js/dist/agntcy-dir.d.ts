import { Client as Client_2 } from '@connectrpc/connect';
import type { EmptySchema } from '@bufbuild/protobuf/wkt';
import type { GenEnum } from '@bufbuild/protobuf/codegenv2';
import type { GenFile } from '@bufbuild/protobuf/codegenv2';
import type { GenMessage } from '@bufbuild/protobuf/codegenv2';
import type { GenService } from '@bufbuild/protobuf/codegenv2';
import type { JsonObject } from '@bufbuild/protobuf';
import type { Message } from '@bufbuild/protobuf';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { Transport } from '@connectrpc/connect';

/**
 * Supporting credential type definitions
 *
 * @generated from message agntcy.dir.store.v1.BasicAuthCredentials
 */
declare type BasicAuthCredentials = Message<"agntcy.dir.store.v1.BasicAuthCredentials"> & {
    /**
     * @generated from field: string username = 1;
     */
    username: string;

    /**
     * @generated from field: string password = 2;
     */
    password: string;
};

/**
 * Describes the message agntcy.dir.store.v1.BasicAuthCredentials.
 * Use `create(BasicAuthCredentialsSchema)` to create a new message.
 */
declare const BasicAuthCredentialsSchema: GenMessage<BasicAuthCredentials>;

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
    storeClient: Client_2<typeof models_2.store_v1.StoreService>;
    routingClient: Client_2<typeof models_2.routing_v1.RoutingService>;
    publicationClient: Client_2<typeof models_2.routing_v1.PublicationService>;
    searchClient: Client_2<typeof models_2.search_v1.SearchService>;
    signClient: Client_2<typeof models_2.sign_v1.SignService>;
    syncClient: Client_2<typeof models_2.store_v1.SyncService>;
    eventClient: Client_2<typeof models_2.events_v1.EventService>;
    namingClient: Client_2<typeof models_2.naming_v1.NamingService>;
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
    push(records: models_2.core_v1.Record[]): Promise<models_2.core_v1.RecordRef[]>;
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
    push_referrer(requests: models_2.store_v1.PushReferrerRequest[]): Promise<models_2.store_v1.PushReferrerResponse[]>;
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
    pull(refs: models_2.core_v1.RecordRef[]): Promise<models_2.core_v1.Record[]>;
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
    pull_referrer(requests: models_2.store_v1.PullReferrerRequest[]): Promise<models_2.store_v1.PullReferrerResponse[]>;
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
    searchCIDs(request: models_2.search_v1.SearchCIDsRequest): Promise<models_2.search_v1.SearchCIDsResponse[]>;
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
    searchRecords(request: models_2.search_v1.SearchRecordsRequest): Promise<models_2.search_v1.SearchRecordsResponse[]>;
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
    lookup(refs: models_2.core_v1.RecordRef[]): Promise<models_2.core_v1.RecordMeta[]>;
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
    list(request: models_2.routing_v1.ListRequest): Promise<models_2.routing_v1.ListResponse[]>;
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
    publish(request: models_2.routing_v1.PublishRequest): Promise<void>;
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
    unpublish(request: models_2.routing_v1.UnpublishRequest): Promise<void>;
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
    delete(refs: models_2.core_v1.RecordRef[]): Promise<void>;
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
    sign(req: models_2.sign_v1.SignRequest, oidc_client_id?: string): void;
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
    verify(request: models_2.sign_v1.VerifyRequest): Promise<models_2.sign_v1.VerifyResponse>;
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
    create_sync(request: models_2.store_v1.CreateSyncRequest): Promise<models_2.store_v1.CreateSyncResponse>;
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
    list_syncs(request: models_2.store_v1.ListSyncsRequest): Promise<models_2.store_v1.ListSyncsItem[]>;
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
    get_sync(request: models_2.store_v1.GetSyncRequest): Promise<models_2.store_v1.GetSyncResponse>;
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
    delete_sync(request: models_2.store_v1.DeleteSyncRequest): Promise<models_2.store_v1.DeleteSyncResponse>;
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
    listen(request: models_2.events_v1.ListenRequest): AsyncIterable<models_2.events_v1.ListenResponse>;
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
    create_publication(request: models_2.routing_v1.PublishRequest): Promise<models_2.routing_v1.CreatePublicationResponse>;
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
    list_publication(request: models_2.routing_v1.ListPublicationsRequest): Promise<models_2.routing_v1.ListPublicationsItem[]>;
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
    get_publication(request: models_2.routing_v1.GetPublicationRequest): Promise<models_2.routing_v1.GetPublicationResponse>;
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
    resolve(request: models_2.naming_v1.ResolveRequest): Promise<models_2.naming_v1.ResolveResponse>;
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
    getVerificationInfo(request: models_2.naming_v1.GetVerificationInfoRequest): Promise<models_2.naming_v1.GetVerificationInfoResponse>;
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

export declare namespace core_v1 {
    export {
        file_agntcy_dir_core_v1_record,
        RecordRef,
        RecordRefSchema,
        NamedRecordRef,
        NamedRecordRefSchema,
        RecordMeta,
        RecordMetaSchema,
        Record_2 as Record,
        RecordSchema,
        RecordReferrer,
        RecordReferrerSchema
    }
}

/**
 * CreatePublicationResponse returns the result of creating a publication request.
 * This includes the publication ID and any relevant metadata.
 *
 * @generated from message agntcy.dir.routing.v1.CreatePublicationResponse
 */
declare type CreatePublicationResponse = Message<"agntcy.dir.routing.v1.CreatePublicationResponse"> & {
    /**
     * Unique identifier of the publication operation.
     *
     * @generated from field: string publication_id = 1;
     */
    publicationId: string;
};

/**
 * Describes the message agntcy.dir.routing.v1.CreatePublicationResponse.
 * Use `create(CreatePublicationResponseSchema)` to create a new message.
 */
declare const CreatePublicationResponseSchema: GenMessage<CreatePublicationResponse>;

/**
 * CreateSyncRequest defines the parameters for creating a new synchronization operation.
 *
 * Currently supports basic synchronization of all objects from a remote Directory.
 * Future versions may include additional options for filtering and scheduling capabilities.
 *
 * @generated from message agntcy.dir.store.v1.CreateSyncRequest
 */
declare type CreateSyncRequest = Message<"agntcy.dir.store.v1.CreateSyncRequest"> & {
    /**
     * URL of the remote Registry to synchronize from.
     *
     * This should be a complete URL including protocol and port if non-standard.
     * Examples:
     * - "https://directory.example.com"
     * - "http://localhost:8080"
     * - "https://directory.example.com:9443"
     *
     * @generated from field: string remote_directory_url = 1;
     */
    remoteDirectoryUrl: string;

    /**
     * List of CIDs to synchronize from the remote Directory.
     * If empty, all objects will be synchronized.
     *
     * @generated from field: repeated string cids = 2;
     */
    cids: string[];
};

/**
 * Describes the message agntcy.dir.store.v1.CreateSyncRequest.
 * Use `create(CreateSyncRequestSchema)` to create a new message.
 */
declare const CreateSyncRequestSchema: GenMessage<CreateSyncRequest>;

/**
 * CreateSyncResponse contains the result of creating a new synchronization operation.
 *
 * @generated from message agntcy.dir.store.v1.CreateSyncResponse
 */
declare type CreateSyncResponse = Message<"agntcy.dir.store.v1.CreateSyncResponse"> & {
    /**
     * Unique identifier for the created synchronization operation.
     * This ID can be used with other SyncService RPCs to monitor and manage the sync.
     *
     * @generated from field: string sync_id = 1;
     */
    syncId: string;
};

/**
 * Describes the message agntcy.dir.store.v1.CreateSyncResponse.
 * Use `create(CreateSyncResponseSchema)` to create a new message.
 */
declare const CreateSyncResponseSchema: GenMessage<CreateSyncResponse>;

/**
 * DeleteSyncRequest specifies which synchronization to delete.
 *
 * @generated from message agntcy.dir.store.v1.DeleteSyncRequest
 */
declare type DeleteSyncRequest = Message<"agntcy.dir.store.v1.DeleteSyncRequest"> & {
    /**
     * Unique identifier of the synchronization operation to delete.
     *
     * @generated from field: string sync_id = 1;
     */
    syncId: string;
};

/**
 * Describes the message agntcy.dir.store.v1.DeleteSyncRequest.
 * Use `create(DeleteSyncRequestSchema)` to create a new message.
 */
declare const DeleteSyncRequestSchema: GenMessage<DeleteSyncRequest>;

/**
 * DeleteSyncResponse
 *
 * @generated from message agntcy.dir.store.v1.DeleteSyncResponse
 */
declare type DeleteSyncResponse = Message<"agntcy.dir.store.v1.DeleteSyncResponse"> & {
};

/**
 * Describes the message agntcy.dir.store.v1.DeleteSyncResponse.
 * Use `create(DeleteSyncResponseSchema)` to create a new message.
 */
declare const DeleteSyncResponseSchema: GenMessage<DeleteSyncResponse>;

/**
 * DomainVerification represents the result of verifying that a record's
 * signing key is authorized by the domain claimed in the record's name.
 *
 * @generated from message agntcy.dir.naming.v1.DomainVerification
 */
declare type DomainVerification = Message<"agntcy.dir.naming.v1.DomainVerification"> & {
    /**
     * The domain that was verified (e.g., "cisco.com").
     *
     * @generated from field: string domain = 1;
     */
    domain: string;

    /**
     * The verification method used: "wellknown".
     *
     * @generated from field: string method = 2;
     */
    method: string;

    /**
     * The identifier of the domain's public key that matched the record's signing key.
     * This is the "id" field from the well-known file.
     *
     * @generated from field: string key_id = 3;
     */
    keyId: string;

    /**
     * When the verification was performed.
     *
     * @generated from field: google.protobuf.Timestamp verified_at = 4;
     */
    verifiedAt?: Timestamp;
};

/**
 * Describes the message agntcy.dir.naming.v1.DomainVerification.
 * Use `create(DomainVerificationSchema)` to create a new message.
 */
declare const DomainVerificationSchema: GenMessage<DomainVerification>;

/**
 * Event represents a system event that occurred.
 *
 * @generated from message agntcy.dir.events.v1.Event
 */
declare type Event_2 = Message<"agntcy.dir.events.v1.Event"> & {
    /**
     * Unique event identifier (generated by the system).
     *
     * @generated from field: string id = 1;
     */
    id: string;

    /**
     * Type of event that occurred.
     *
     * @generated from field: agntcy.dir.events.v1.EventType type = 2;
     */
    type: EventType;

    /**
     * When the event occurred.
     *
     * @generated from field: google.protobuf.Timestamp timestamp = 3;
     */
    timestamp?: Timestamp;

    /**
     * Resource identifier (CID for records, sync_id for syncs, etc.).
     *
     * @generated from field: string resource_id = 4;
     */
    resourceId: string;

    /**
     * Optional labels associated with the record (for record events).
     *
     * @generated from field: repeated string labels = 5;
     */
    labels: string[];

    /**
     * Optional metadata for additional context.
     * Used for flexible event-specific data that doesn't fit standard fields.
     *
     * @generated from field: map<string, string> metadata = 7;
     */
    metadata: { [key: string]: string };
};

export declare namespace events_v1 {
    export {
        file_agntcy_dir_events_v1_event_service,
        ListenRequest,
        ListenRequestSchema,
        ListenResponse,
        ListenResponseSchema,
        Event_2 as Event,
        EventSchema,
        EventType,
        EventTypeSchema,
        EventService
    }
}

/**
 * Describes the message agntcy.dir.events.v1.Event.
 * Use `create(EventSchema)` to create a new message.
 */
declare const EventSchema: GenMessage<Event_2>;

/**
 * EventService provides real-time event streaming for all system operations.
 * Events are delivered from subscription time forward with no history or replay.
 * This service enables external applications to react to system changes in real-time.
 *
 * @generated from service agntcy.dir.events.v1.EventService
 */
declare const EventService: GenService<{
    /**
     * Listen establishes a streaming connection to receive events.
     * Events are only delivered while the stream is active.
     * On disconnect, missed events are not recoverable.
     *
     * @generated from rpc agntcy.dir.events.v1.EventService.Listen
     */
    listen: {
        methodKind: "server_streaming";
        input: typeof ListenRequestSchema;
        output: typeof ListenResponseSchema;
    },
}>;

/**
 * EventType represents all valid event types in the system.
 * Each value represents a specific operation that can occur.
 *
 * Supported Events:
 * - Store: RECORD_PUSHED, RECORD_PULLED, RECORD_DELETED
 * - Routing: RECORD_PUBLISHED, RECORD_UNPUBLISHED
 * - Sync: SYNC_CREATED, SYNC_COMPLETED, SYNC_FAILED
 * - Sign: RECORD_SIGNED
 *
 * @generated from enum agntcy.dir.events.v1.EventType
 */
declare enum EventType {
    /**
     * Unknown/unspecified event type.
     *
     * @generated from enum value: EVENT_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,

    /**
     * A record was pushed to local storage.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_PUSHED = 1;
     */
    RECORD_PUSHED = 1,

    /**
     * A record was pulled from storage.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_PULLED = 2;
     */
    RECORD_PULLED = 2,

    /**
     * A record was deleted from storage.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_DELETED = 3;
     */
    RECORD_DELETED = 3,

    /**
     * A record was published/announced to the network.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_PUBLISHED = 4;
     */
    RECORD_PUBLISHED = 4,

    /**
     * A record was unpublished from the network.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_UNPUBLISHED = 5;
     */
    RECORD_UNPUBLISHED = 5,

    /**
     * A sync operation was created/initiated.
     *
     * @generated from enum value: EVENT_TYPE_SYNC_CREATED = 6;
     */
    SYNC_CREATED = 6,

    /**
     * A sync operation completed successfully.
     *
     * @generated from enum value: EVENT_TYPE_SYNC_COMPLETED = 7;
     */
    SYNC_COMPLETED = 7,

    /**
     * A sync operation failed.
     *
     * @generated from enum value: EVENT_TYPE_SYNC_FAILED = 8;
     */
    SYNC_FAILED = 8,

    /**
     * A record was signed.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_SIGNED = 9;
     */
    RECORD_SIGNED = 9,

    /**
     * A record signature was verified.
     *
     * @generated from enum value: EVENT_TYPE_RECORD_VERIFIED = 10;
     */
    RECORD_VERIFIED = 10,

    /**
     * A public key was uploaded.
     *
     * @generated from enum value: EVENT_TYPE_PUBLIC_KEY_UPLOADED = 11;
     */
    PUBLIC_KEY_UPLOADED = 11,
}

/**
 * Describes the enum agntcy.dir.events.v1.EventType.
 */
declare const EventTypeSchema: GenEnum<EventType>;

/**
 * Describes the file agntcy/dir/core/v1/record.proto.
 */
declare const file_agntcy_dir_core_v1_record: GenFile;

/**
 * Describes the file agntcy/dir/events/v1/event_service.proto.
 */
declare const file_agntcy_dir_events_v1_event_service: GenFile;

/**
 * Describes the file agntcy/dir/naming/v1/name_verification.proto.
 */
declare const file_agntcy_dir_naming_v1_name_verification: GenFile;

/**
 * Describes the file agntcy/dir/naming/v1/naming_service.proto.
 */
declare const file_agntcy_dir_naming_v1_naming_service: GenFile;

/**
 * Describes the file agntcy/dir/routing/v1/peer.proto.
 */
declare const file_agntcy_dir_routing_v1_peer: GenFile;

/**
 * Describes the file agntcy/dir/routing/v1/publication_service.proto.
 */
declare const file_agntcy_dir_routing_v1_publication_service: GenFile;

/**
 * Describes the file agntcy/dir/routing/v1/record_query.proto.
 */
declare const file_agntcy_dir_routing_v1_record_query: GenFile;

/**
 * Describes the file agntcy/dir/routing/v1/routing_service.proto.
 */
declare const file_agntcy_dir_routing_v1_routing_service: GenFile;

/**
 * Describes the file agntcy/dir/search/v1/record_query.proto.
 */
declare const file_agntcy_dir_search_v1_record_query: GenFile;

/**
 * Describes the file agntcy/dir/search/v1/search_service.proto.
 */
declare const file_agntcy_dir_search_v1_search_service: GenFile;

/**
 * Describes the file agntcy/dir/sign/v1/public_key.proto.
 */
declare const file_agntcy_dir_sign_v1_public_key: GenFile;

/**
 * Describes the file agntcy/dir/sign/v1/sign_service.proto.
 */
declare const file_agntcy_dir_sign_v1_sign_service: GenFile;

/**
 * Describes the file agntcy/dir/sign/v1/signature.proto.
 */
declare const file_agntcy_dir_sign_v1_signature: GenFile;

/**
 * Describes the file agntcy/dir/store/v1/store_service.proto.
 */
declare const file_agntcy_dir_store_v1_store_service: GenFile;

/**
 * Describes the file agntcy/dir/store/v1/sync_service.proto.
 */
declare const file_agntcy_dir_store_v1_sync_service: GenFile;

/**
 * GetPublicationRequest specifies which publication to retrieve by its identifier.
 *
 * @generated from message agntcy.dir.routing.v1.GetPublicationRequest
 */
declare type GetPublicationRequest = Message<"agntcy.dir.routing.v1.GetPublicationRequest"> & {
    /**
     * Unique identifier of the publication operation to query.
     *
     * @generated from field: string publication_id = 1;
     */
    publicationId: string;
};

/**
 * Describes the message agntcy.dir.routing.v1.GetPublicationRequest.
 * Use `create(GetPublicationRequestSchema)` to create a new message.
 */
declare const GetPublicationRequestSchema: GenMessage<GetPublicationRequest>;

/**
 * GetPublicationResponse contains the full details of a specific publication request.
 * Includes status, progress information, and any error details if applicable.
 *
 * @generated from message agntcy.dir.routing.v1.GetPublicationResponse
 */
declare type GetPublicationResponse = Message<"agntcy.dir.routing.v1.GetPublicationResponse"> & {
    /**
     * Unique identifier of the publication operation.
     *
     * @generated from field: string publication_id = 1;
     */
    publicationId: string;

    /**
     * Current status of the publication operation.
     *
     * @generated from field: agntcy.dir.routing.v1.PublicationStatus status = 2;
     */
    status: PublicationStatus;

    /**
     * Timestamp when the publication operation was created in the RFC3339 format.
     * Specs: https://www.rfc-editor.org/rfc/rfc3339.html
     *
     * @generated from field: string created_time = 3;
     */
    createdTime: string;

    /**
     * Timestamp of the most recent status update for this publication in the RFC3339 format.
     *
     * @generated from field: string last_update_time = 4;
     */
    lastUpdateTime: string;
};

/**
 * Describes the message agntcy.dir.routing.v1.GetPublicationResponse.
 * Use `create(GetPublicationResponseSchema)` to create a new message.
 */
declare const GetPublicationResponseSchema: GenMessage<GetPublicationResponse>;

/**
 * GetSyncRequest specifies which synchronization status to retrieve.
 *
 * @generated from message agntcy.dir.store.v1.GetSyncRequest
 */
declare type GetSyncRequest = Message<"agntcy.dir.store.v1.GetSyncRequest"> & {
    /**
     * Unique identifier of the synchronization operation to query.
     *
     * @generated from field: string sync_id = 1;
     */
    syncId: string;
};

/**
 * Describes the message agntcy.dir.store.v1.GetSyncRequest.
 * Use `create(GetSyncRequestSchema)` to create a new message.
 */
declare const GetSyncRequestSchema: GenMessage<GetSyncRequest>;

/**
 * GetSyncResponse provides detailed information about a specific synchronization operation.
 *
 * @generated from message agntcy.dir.store.v1.GetSyncResponse
 */
declare type GetSyncResponse = Message<"agntcy.dir.store.v1.GetSyncResponse"> & {
    /**
     * Unique identifier of the synchronization operation.
     *
     * @generated from field: string sync_id = 1;
     */
    syncId: string;

    /**
     * Current status of the synchronization operation.
     *
     * @generated from field: agntcy.dir.store.v1.SyncStatus status = 2;
     */
    status: SyncStatus;

    /**
     * URL of the remote Directory node being synchronized from.
     *
     * @generated from field: string remote_directory_url = 3;
     */
    remoteDirectoryUrl: string;

    /**
     * Timestamp when the synchronization operation was created in the RFC3339 format.
     * Specs: https://www.rfc-editor.org/rfc/rfc3339.html
     *
     * @generated from field: string created_time = 4;
     */
    createdTime: string;

    /**
     * Timestamp of the most recent status update for this synchronization in the RFC3339 format.
     *
     * @generated from field: string last_update_time = 5;
     */
    lastUpdateTime: string;
};

/**
 * Describes the message agntcy.dir.store.v1.GetSyncResponse.
 * Use `create(GetSyncResponseSchema)` to create a new message.
 */
declare const GetSyncResponseSchema: GenMessage<GetSyncResponse>;

/**
 * GetVerificationInfoRequest is the request for retrieving verification info.
 * Either cid OR name must be provided. If name is provided, it will be resolved
 * to a CID first (using the latest version if version is not specified).
 *
 * @generated from message agntcy.dir.naming.v1.GetVerificationInfoRequest
 */
declare type GetVerificationInfoRequest = Message<"agntcy.dir.naming.v1.GetVerificationInfoRequest"> & {
    /**
     * The CID of the record to check.
     * If provided, name and version are ignored.
     *
     * @generated from field: optional string cid = 1;
     */
    cid?: string;

    /**
     * The name of the record to check (e.g., "cisco.com/agent").
     * Used when cid is not provided.
     *
     * @generated from field: optional string name = 2;
     */
    name?: string;

    /**
     * Optional version when looking up by name (e.g., "v1.0.0").
     * If not specified, the latest version is used.
     *
     * @generated from field: optional string version = 3;
     */
    version?: string;
};

/**
 * Describes the message agntcy.dir.naming.v1.GetVerificationInfoRequest.
 * Use `create(GetVerificationInfoRequestSchema)` to create a new message.
 */
declare const GetVerificationInfoRequestSchema: GenMessage<GetVerificationInfoRequest>;

/**
 * GetVerificationInfoResponse is the response for retrieving verification info.
 *
 * @generated from message agntcy.dir.naming.v1.GetVerificationInfoResponse
 */
declare type GetVerificationInfoResponse = Message<"agntcy.dir.naming.v1.GetVerificationInfoResponse"> & {
    /**
     * Whether the record has verified name ownership.
     *
     * @generated from field: bool verified = 1;
     */
    verified: boolean;

    /**
     * The verification details (only set if verified is true).
     *
     * @generated from field: agntcy.dir.naming.v1.Verification verification = 2;
     */
    verification?: Verification;

    /**
     * Error message if lookup failed.
     *
     * @generated from field: optional string error_message = 3;
     */
    errorMessage?: string;
};

/**
 * Describes the message agntcy.dir.naming.v1.GetVerificationInfoResponse.
 * Use `create(GetVerificationInfoResponseSchema)` to create a new message.
 */
declare const GetVerificationInfoResponseSchema: GenMessage<GetVerificationInfoResponse>;

/**
 * ListenRequest specifies filters for event subscription.
 *
 * @generated from message agntcy.dir.events.v1.ListenRequest
 */
declare type ListenRequest = Message<"agntcy.dir.events.v1.ListenRequest"> & {
    /**
     * Event types to subscribe to.
     * If empty, subscribes to all event types.
     *
     * @generated from field: repeated agntcy.dir.events.v1.EventType event_types = 1;
     */
    eventTypes: EventType[];

    /**
     * Optional label filters (e.g., "/skills/AI", "/domains/research").
     * Only events for records matching these labels are delivered.
     * Uses substring matching.
     *
     * @generated from field: repeated string label_filters = 2;
     */
    labelFilters: string[];

    /**
     * Optional CID filters.
     * Only events for specific CIDs are delivered.
     *
     * @generated from field: repeated string cid_filters = 3;
     */
    cidFilters: string[];
};

/**
 * Describes the message agntcy.dir.events.v1.ListenRequest.
 * Use `create(ListenRequestSchema)` to create a new message.
 */
declare const ListenRequestSchema: GenMessage<ListenRequest>;

/**
 * ListenResponse is the response message for the Listen RPC.
 * Wraps the Event message to allow for future extensions without breaking the Event structure.
 *
 * @generated from message agntcy.dir.events.v1.ListenResponse
 */
declare type ListenResponse = Message<"agntcy.dir.events.v1.ListenResponse"> & {
    /**
     * The event that occurred.
     *
     * @generated from field: agntcy.dir.events.v1.Event event = 1;
     */
    event?: Event_2;
};

/**
 * Describes the message agntcy.dir.events.v1.ListenResponse.
 * Use `create(ListenResponseSchema)` to create a new message.
 */
declare const ListenResponseSchema: GenMessage<ListenResponse>;

/**
 * ListPublicationsItem represents a single publication request in the list response.
 * Contains publication details including ID, status, and creation timestamp.
 *
 * @generated from message agntcy.dir.routing.v1.ListPublicationsItem
 */
declare type ListPublicationsItem = Message<"agntcy.dir.routing.v1.ListPublicationsItem"> & {
    /**
     * Unique identifier of the publication operation.
     *
     * @generated from field: string publication_id = 1;
     */
    publicationId: string;

    /**
     * Current status of the publication operation.
     *
     * @generated from field: agntcy.dir.routing.v1.PublicationStatus status = 2;
     */
    status: PublicationStatus;

    /**
     * Timestamp when the publication operation was created in the RFC3339 format.
     * Specs: https://www.rfc-editor.org/rfc/rfc3339.html
     *
     * @generated from field: string created_time = 3;
     */
    createdTime: string;

    /**
     * Timestamp of the most recent status update for this publication in the RFC3339 format.
     *
     * @generated from field: string last_update_time = 4;
     */
    lastUpdateTime: string;
};

/**
 * Describes the message agntcy.dir.routing.v1.ListPublicationsItem.
 * Use `create(ListPublicationsItemSchema)` to create a new message.
 */
declare const ListPublicationsItemSchema: GenMessage<ListPublicationsItem>;

/**
 * ListPublicationsRequest contains optional filters for listing publication requests.
 *
 * @generated from message agntcy.dir.routing.v1.ListPublicationsRequest
 */
declare type ListPublicationsRequest = Message<"agntcy.dir.routing.v1.ListPublicationsRequest"> & {
    /**
     * Optional limit on the number of results to return.
     *
     * @generated from field: optional uint32 limit = 2;
     */
    limit?: number;

    /**
     * Optional offset for pagination of results.
     *
     * @generated from field: optional uint32 offset = 3;
     */
    offset?: number;
};

/**
 * Describes the message agntcy.dir.routing.v1.ListPublicationsRequest.
 * Use `create(ListPublicationsRequestSchema)` to create a new message.
 */
declare const ListPublicationsRequestSchema: GenMessage<ListPublicationsRequest>;

/**
 * @generated from message agntcy.dir.routing.v1.ListRequest
 */
declare type ListRequest = Message<"agntcy.dir.routing.v1.ListRequest"> & {
    /**
     * List of queries to match against the records.
     * If set, all queries must match for the record to be returned.
     *
     * @generated from field: repeated agntcy.dir.routing.v1.RecordQuery queries = 1;
     */
    queries: RecordQuery_2[];

    /**
     * Limit the number of results returned.
     * If not set, it will return all records that this peer is providing.
     *
     * @generated from field: optional uint32 limit = 2;
     */
    limit?: number;
};

/**
 * Describes the message agntcy.dir.routing.v1.ListRequest.
 * Use `create(ListRequestSchema)` to create a new message.
 */
declare const ListRequestSchema: GenMessage<ListRequest>;

/**
 * @generated from message agntcy.dir.routing.v1.ListResponse
 */
declare type ListResponse = Message<"agntcy.dir.routing.v1.ListResponse"> & {
    /**
     * The record that matches the list queries.
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 1;
     */
    recordRef?: RecordRef;

    /**
     * Labels associated with this record (skills, domains, modules)
     * Derived from the record content for CLI display purposes
     *
     * @generated from field: repeated string labels = 2;
     */
    labels: string[];
};

/**
 * Describes the message agntcy.dir.routing.v1.ListResponse.
 * Use `create(ListResponseSchema)` to create a new message.
 */
declare const ListResponseSchema: GenMessage<ListResponse>;

/**
 * ListSyncItem represents a single synchronization in the list of all syncs.
 *
 * @generated from message agntcy.dir.store.v1.ListSyncsItem
 */
declare type ListSyncsItem = Message<"agntcy.dir.store.v1.ListSyncsItem"> & {
    /**
     * Unique identifier of the synchronization operation.
     *
     * @generated from field: string sync_id = 1;
     */
    syncId: string;

    /**
     * Current status of the synchronization operation.
     *
     * @generated from field: agntcy.dir.store.v1.SyncStatus status = 2;
     */
    status: SyncStatus;

    /**
     * URL of the remote Directory being synchronized from.
     *
     * @generated from field: string remote_directory_url = 3;
     */
    remoteDirectoryUrl: string;
};

/**
 * Describes the message agntcy.dir.store.v1.ListSyncsItem.
 * Use `create(ListSyncsItemSchema)` to create a new message.
 */
declare const ListSyncsItemSchema: GenMessage<ListSyncsItem>;

/**
 * ListSyncsRequest specifies parameters for listing synchronization operations.
 *
 * @generated from message agntcy.dir.store.v1.ListSyncsRequest
 */
declare type ListSyncsRequest = Message<"agntcy.dir.store.v1.ListSyncsRequest"> & {
    /**
     * Optional limit on the number of results to return.
     *
     * @generated from field: optional uint32 limit = 2;
     */
    limit?: number;

    /**
     * Optional offset for pagination of results.
     *
     * @generated from field: optional uint32 offset = 3;
     */
    offset?: number;
};

/**
 * Describes the message agntcy.dir.store.v1.ListSyncsRequest.
 * Use `create(ListSyncsRequestSchema)` to create a new message.
 */
declare const ListSyncsRequestSchema: GenMessage<ListSyncsRequest>;

export declare namespace models {
    export {
        core_v1,
        naming_v1,
        routing_v1,
        search_v1,
        sign_v1,
        store_v1,
        events_v1
    }
}

declare namespace models_2 {
    export {
        core_v1,
        naming_v1,
        routing_v1,
        search_v1,
        sign_v1,
        store_v1,
        events_v1
    }
}

/**
 * Represents a named reference to a Record with version information.
 *
 * @generated from message agntcy.dir.core.v1.NamedRecordRef
 */
declare type NamedRecordRef = Message<"agntcy.dir.core.v1.NamedRecordRef"> & {
    /**
     * The name of the record.
     *
     * @generated from field: string name = 1;
     */
    name: string;

    /**
     * The version of the record.
     *
     * @generated from field: string version = 2;
     */
    version: string;

    /**
     * The CID of the record.
     *
     * @generated from field: string cid = 3;
     */
    cid: string;
};

/**
 * Describes the message agntcy.dir.core.v1.NamedRecordRef.
 * Use `create(NamedRecordRefSchema)` to create a new message.
 */
declare const NamedRecordRefSchema: GenMessage<NamedRecordRef>;

export declare namespace naming_v1 {
    export {
        file_agntcy_dir_naming_v1_naming_service,
        GetVerificationInfoRequest,
        GetVerificationInfoRequestSchema,
        GetVerificationInfoResponse,
        GetVerificationInfoResponseSchema,
        ResolveRequest,
        ResolveRequestSchema,
        ResolveResponse,
        ResolveResponseSchema,
        NamingService,
        file_agntcy_dir_naming_v1_name_verification,
        Verification,
        VerificationSchema,
        DomainVerification,
        DomainVerificationSchema
    }
}

/**
 * NamingService provides methods for name resolution and verification.
 * Note: Verification is performed automatically by the backend scheduler
 * for signed records with verifiable names (http://, https:// prefixes).
 *
 * @generated from service agntcy.dir.naming.v1.NamingService
 */
declare const NamingService: GenService<{
    /**
     * GetVerificationInfo retrieves the verification info for a record.
     *
     * @generated from rpc agntcy.dir.naming.v1.NamingService.GetVerificationInfo
     */
    getVerificationInfo: {
        methodKind: "unary";
        input: typeof GetVerificationInfoRequestSchema;
        output: typeof GetVerificationInfoResponseSchema;
    },
    /**
     * Resolve resolves a record reference (name with optional version) to CIDs.
     * Supports Docker-style references:
     *   - "name" -> returns all versions (newest first)
     *   - "name:version" -> returns the specific version
     *   - "name@cid" -> hash-verified lookup (latest version)
     *   - "name:version@cid" -> hash-verified lookup (specific version)
     * Returns an error if no matching record is found.
     *
     * @generated from rpc agntcy.dir.naming.v1.NamingService.Resolve
     */
    resolve: {
        methodKind: "unary";
        input: typeof ResolveRequestSchema;
        output: typeof ResolveResponseSchema;
    },
}>;

/**
 * @generated from message agntcy.dir.routing.v1.Peer
 */
declare type Peer = Message<"agntcy.dir.routing.v1.Peer"> & {
    /**
     * ID of a given peer, typically described by a protocol.
     * For example:
     *  - SPIFFE:   "spiffe://example.org/service/foo"
     *  - JWT:      "jwt:sub=alice,iss=https://issuer.example.com"
     *  - Tor:      "onion:abcdefghijklmno.onion"
     *  - DID:      "did:example:123456789abcdefghi"
     *  - IPFS:     "ipfs:QmYwAPJzv5CZsnAzt8auVZRn2E6sD1c4x8pN5o6d5cW4D5"
     *
     * @generated from field: string id = 1;
     */
    id: string;

    /**
     * Multiaddrs for a given peer.
     * For example:
     * - "/ip4/127.0.0.1/tcp/4001"
     * - "/ip6/::1/tcp/4001"
     * - "/dns4/example.com/tcp/443/https"
     *
     * @generated from field: repeated string addrs = 2;
     */
    addrs: string[];

    /**
     * Additional metadata about the peer.
     *
     * @generated from field: map<string, string> annotations = 3;
     */
    annotations: { [key: string]: string };

    /**
     * Used to signal the sender's connection capabilities to the peer.
     *
     * @generated from field: agntcy.dir.routing.v1.PeerConnectionType connection = 4;
     */
    connection: PeerConnectionType;
};

/**
 * @generated from enum agntcy.dir.routing.v1.PeerConnectionType
 */
declare enum PeerConnectionType {
    /**
     * Sender does not have a connection to peer, and no extra information (default)
     *
     * @generated from enum value: PEER_CONNECTION_TYPE_NOT_CONNECTED = 0;
     */
    NOT_CONNECTED = 0,

    /**
     * Sender has a live connection to peer.
     *
     * @generated from enum value: PEER_CONNECTION_TYPE_CONNECTED = 1;
     */
    CONNECTED = 1,

    /**
     * Sender recently connected to peer.
     *
     * @generated from enum value: PEER_CONNECTION_TYPE_CAN_CONNECT = 2;
     */
    CAN_CONNECT = 2,

    /**
     * Sender made strong effort to connect to peer repeatedly but failed.
     *
     * @generated from enum value: PEER_CONNECTION_TYPE_CANNOT_CONNECT = 3;
     */
    CANNOT_CONNECT = 3,
}

/**
 * Describes the enum agntcy.dir.routing.v1.PeerConnectionType.
 */
declare const PeerConnectionTypeSchema: GenEnum<PeerConnectionType>;

/**
 * Describes the message agntcy.dir.routing.v1.Peer.
 * Use `create(PeerSchema)` to create a new message.
 */
declare const PeerSchema: GenMessage<Peer>;

/**
 * PublicationService manages publication requests for announcing records to the DHT.
 *
 * Publications are stored in the database and processed by a worker that runs every hour.
 * The publication workflow:
 * 1. Publications are created via routing's Publish RPC by specifying either a query, a list of CIDs, or all records
 * 2. Publication requests are added to the database
 * 3. PublicationWorker queries the data using the publication request from the database to get the list of CIDs to be published
 * 4. PublicationWorker announces the records with these CIDs to the DHT
 *
 * @generated from service agntcy.dir.routing.v1.PublicationService
 */
declare const PublicationService: GenService<{
    /**
     * CreatePublication creates a new publication request that will be processed by the PublicationWorker.
     * The publication request can specify either a query, a list of specific CIDs, or all records to be announced to the DHT.
     *
     * @generated from rpc agntcy.dir.routing.v1.PublicationService.CreatePublication
     */
    createPublication: {
        methodKind: "unary";
        input: typeof PublishRequestSchema;
        output: typeof CreatePublicationResponseSchema;
    },
    /**
     * ListPublications returns a stream of all publication requests in the system.
     * This allows monitoring of pending, processing, and completed publication requests.
     *
     * @generated from rpc agntcy.dir.routing.v1.PublicationService.ListPublications
     */
    listPublications: {
        methodKind: "server_streaming";
        input: typeof ListPublicationsRequestSchema;
        output: typeof ListPublicationsItemSchema;
    },
    /**
     * GetPublication retrieves details of a specific publication request by its identifier.
     * This includes the current status and any associated metadata.
     *
     * @generated from rpc agntcy.dir.routing.v1.PublicationService.GetPublication
     */
    getPublication: {
        methodKind: "unary";
        input: typeof GetPublicationRequestSchema;
        output: typeof GetPublicationResponseSchema;
    },
}>;

/**
 * PublicationStatus represents the current state of a publication request.
 * Publications progress from pending to processing to completed or failed states.
 *
 * @generated from enum agntcy.dir.routing.v1.PublicationStatus
 */
declare enum PublicationStatus {
    /**
     * Default/unset status - should not be used in practice
     *
     * @generated from enum value: PUBLICATION_STATUS_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,

    /**
     * Sync operation has been created but not yet started
     *
     * @generated from enum value: PUBLICATION_STATUS_PENDING = 1;
     */
    PENDING = 1,

    /**
     * Sync operation is actively discovering and transferring objects
     *
     * @generated from enum value: PUBLICATION_STATUS_IN_PROGRESS = 2;
     */
    IN_PROGRESS = 2,

    /**
     * Sync operation has been successfully completed
     *
     * @generated from enum value: PUBLICATION_STATUS_COMPLETED = 3;
     */
    COMPLETED = 3,

    /**
     * Sync operation encountered an error and stopped
     *
     * @generated from enum value: PUBLICATION_STATUS_FAILED = 4;
     */
    FAILED = 4,
}

/**
 * Describes the enum agntcy.dir.routing.v1.PublicationStatus.
 */
declare const PublicationStatusSchema: GenEnum<PublicationStatus>;

/**
 * PublicKey is the public key data associated with a Record.
 * Multiple public keys can be associated with a single Record.
 *
 * @generated from message agntcy.dir.sign.v1.PublicKey
 */
declare type PublicKey = Message<"agntcy.dir.sign.v1.PublicKey"> & {
    /**
     * PEM-encoded public key string.
     *
     * @generated from field: string key = 1;
     */
    key: string;
};

/**
 * Describes the message agntcy.dir.sign.v1.PublicKey.
 * Use `create(PublicKeySchema)` to create a new message.
 */
declare const PublicKeySchema: GenMessage<PublicKey>;

/**
 * @generated from message agntcy.dir.routing.v1.PublishRequest
 */
declare type PublishRequest = Message<"agntcy.dir.routing.v1.PublishRequest"> & {
    /**
     * @generated from oneof agntcy.dir.routing.v1.PublishRequest.request
     */
    request: {
        /**
         * References to the records to be published.
         *
         * @generated from field: agntcy.dir.routing.v1.RecordRefs record_refs = 1;
         */
        value: RecordRefs;
        case: "recordRefs";
    } | {
        /**
         * Queries to match against the records to be published.
         *
         * @generated from field: agntcy.dir.routing.v1.RecordQueries queries = 2;
         */
        value: RecordQueries;
        case: "queries";
    } | { case: undefined; value?: undefined };
};

/**
 * Describes the message agntcy.dir.routing.v1.PublishRequest.
 * Use `create(PublishRequestSchema)` to create a new message.
 */
declare const PublishRequestSchema: GenMessage<PublishRequest>;

/**
 * PullReferrerRequest represents a record with optional OCI artifacts for pull operations.
 *
 * @generated from message agntcy.dir.store.v1.PullReferrerRequest
 */
declare type PullReferrerRequest = Message<"agntcy.dir.store.v1.PullReferrerRequest"> & {
    /**
     * Record reference
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 1;
     */
    recordRef?: RecordRef;

    /**
     * Record referrer type to be pulled
     * If not provided, all referrers will be pulled
     *
     * @generated from field: optional string referrer_type = 2;
     */
    referrerType?: string;
};

/**
 * Describes the message agntcy.dir.store.v1.PullReferrerRequest.
 * Use `create(PullReferrerRequestSchema)` to create a new message.
 */
declare const PullReferrerRequestSchema: GenMessage<PullReferrerRequest>;

/**
 * PullReferrerResponse is returned after successfully fetching a record referrer.
 *
 * @generated from message agntcy.dir.store.v1.PullReferrerResponse
 */
declare type PullReferrerResponse = Message<"agntcy.dir.store.v1.PullReferrerResponse"> & {
    /**
     * RecordReferrer object associated with the record
     *
     * @generated from field: agntcy.dir.core.v1.RecordReferrer referrer = 1;
     */
    referrer?: RecordReferrer;
};

/**
 * Describes the message agntcy.dir.store.v1.PullReferrerResponse.
 * Use `create(PullReferrerResponseSchema)` to create a new message.
 */
declare const PullReferrerResponseSchema: GenMessage<PullReferrerResponse>;

/**
 * PushReferrerRequest represents a record with optional OCI artifacts for push operations.
 *
 * @generated from message agntcy.dir.store.v1.PushReferrerRequest
 */
declare type PushReferrerRequest = Message<"agntcy.dir.store.v1.PushReferrerRequest"> & {
    /**
     * Record reference
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 1;
     */
    recordRef?: RecordRef;

    /**
     * RecordReferrer object to be stored for the record
     *
     * @generated from field: agntcy.dir.core.v1.RecordReferrer referrer = 2;
     */
    referrer?: RecordReferrer;
};

/**
 * Describes the message agntcy.dir.store.v1.PushReferrerRequest.
 * Use `create(PushReferrerRequestSchema)` to create a new message.
 */
declare const PushReferrerRequestSchema: GenMessage<PushReferrerRequest>;

/**
 * PushReferrerResponse
 *
 * @generated from message agntcy.dir.store.v1.PushReferrerResponse
 */
declare type PushReferrerResponse = Message<"agntcy.dir.store.v1.PushReferrerResponse"> & {
    /**
     * The push process result
     *
     * @generated from field: bool success = 1;
     */
    success: boolean;

    /**
     * Optional error message if push failed
     *
     * @generated from field: optional string error_message = 2;
     */
    errorMessage?: string;
};

/**
 * Describes the message agntcy.dir.store.v1.PushReferrerResponse.
 * Use `create(PushReferrerResponseSchema)` to create a new message.
 */
declare const PushReferrerResponseSchema: GenMessage<PushReferrerResponse>;

/**
 * Record is a generic object that encapsulates data of different Record types.
 *
 * Supported schemas:
 *
 * v0.7.0: https://schema.oasf.outshift.com/0.7.0/objects/record
 * v0.8.0: https://schema.oasf.outshift.com/0.8.0/objects/record
 * v1.0.0-rc.1: https://schema.oasf.outshift.com/1.0.0-rc.1/objects/record
 *
 * @generated from message agntcy.dir.core.v1.Record
 */
declare type Record_2 = Message<"agntcy.dir.core.v1.Record"> & {
    /**
     * @generated from field: google.protobuf.Struct data = 1;
     */
    data?: JsonObject;
};

/**
 * Defines metadata about a record.
 *
 * @generated from message agntcy.dir.core.v1.RecordMeta
 */
declare type RecordMeta = Message<"agntcy.dir.core.v1.RecordMeta"> & {
    /**
     * CID of the record.
     *
     * @generated from field: string cid = 1;
     */
    cid: string;

    /**
     * Annotations attached to the record.
     *
     * @generated from field: map<string, string> annotations = 2;
     */
    annotations: { [key: string]: string };

    /**
     * Schema version of the record.
     *
     * @generated from field: string schema_version = 3;
     */
    schemaVersion: string;

    /**
     * Creation timestamp of the record in the RFC3339 format.
     * Specs: https://www.rfc-editor.org/rfc/rfc3339.html
     *
     * @generated from field: string created_at = 4;
     */
    createdAt: string;
};

/**
 * Describes the message agntcy.dir.core.v1.RecordMeta.
 * Use `create(RecordMetaSchema)` to create a new message.
 */
declare const RecordMetaSchema: GenMessage<RecordMeta>;

/**
 * @generated from message agntcy.dir.routing.v1.RecordQueries
 */
declare type RecordQueries = Message<"agntcy.dir.routing.v1.RecordQueries"> & {
    /**
     * @generated from field: repeated agntcy.dir.search.v1.RecordQuery queries = 1;
     */
    queries: RecordQuery[];
};

/**
 * Describes the message agntcy.dir.routing.v1.RecordQueries.
 * Use `create(RecordQueriesSchema)` to create a new message.
 */
declare const RecordQueriesSchema: GenMessage<RecordQueries>;

/**
 * A query to match the record against during discovery.
 * For example:
 *   Exact match:      { type: RECORD_QUERY_TYPE_NAME, value: "my-agent" }
 *   Wildcard match:   { type: RECORD_QUERY_TYPE_NAME, value: "web*" }
 *   Pattern match:    { type: RECORD_QUERY_TYPE_SKILL_NAME, value: "*machine*learning*" }
 *   Question mark:    { type: RECORD_QUERY_TYPE_VERSION, value: "v1.0.?" }
 *   Complex match:    { type: RECORD_QUERY_TYPE_LOCATOR, value: "docker-image:https://*.example.com/*" }
 *
 * @generated from message agntcy.dir.search.v1.RecordQuery
 */
declare type RecordQuery = Message<"agntcy.dir.search.v1.RecordQuery"> & {
    /**
     * The type of the query to match against.
     *
     * @generated from field: agntcy.dir.search.v1.RecordQueryType type = 1;
     */
    type: RecordQueryType;

    /**
     * The query value to match against.
     * Supports wildcard patterns:
     *   '*' - matches zero or more characters
     *   '?' - matches exactly one character
     *
     * @generated from field: string value = 2;
     */
    value: string;
};

/**
 * A query to match the record against during discovery.
 * For example:
 *  { type: RECORD_QUERY_TYPE_SKILL, value: "Natural Language Processing" }
 *  { type: RECORD_QUERY_TYPE_LOCATOR, value: "helm-chart" }
 *  { type: RECORD_QUERY_TYPE_DOMAIN, value: "research" }
 *  { type: RECORD_QUERY_TYPE_MODULE, value: "core/llm/model" }
 *
 * @generated from message agntcy.dir.routing.v1.RecordQuery
 */
declare type RecordQuery_2 = Message<"agntcy.dir.routing.v1.RecordQuery"> & {
    /**
     * The type of the query to match against.
     *
     * @generated from field: agntcy.dir.routing.v1.RecordQueryType type = 1;
     */
    type: RecordQueryType_2;

    /**
     * The query value to match against.
     *
     * @generated from field: string value = 2;
     */
    value: string;
};

/**
 * Describes the message agntcy.dir.routing.v1.RecordQuery.
 * Use `create(RecordQuerySchema)` to create a new message.
 */
declare const RecordQuerySchema: GenMessage<RecordQuery_2>;

/**
 * Describes the message agntcy.dir.search.v1.RecordQuery.
 * Use `create(RecordQuerySchema)` to create a new message.
 */
declare const RecordQuerySchema_2: GenMessage<RecordQuery>;

/**
 * Defines a list of supported record query types.
 *
 * @generated from enum agntcy.dir.search.v1.RecordQueryType
 */
declare enum RecordQueryType {
    /**
     * Unspecified query type.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,

    /**
     * Query for a record name.
     * Supports wildcard patterns: "web*", "*service", "api-*-v2", "???api"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_NAME = 1;
     */
    NAME = 1,

    /**
     * Query for a record version.
     * Supports wildcard patterns: "v1.*", "v2.*", "*-beta", "v1.0.?"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_VERSION = 2;
     */
    VERSION = 2,

    /**
     * Query for a skill ID.
     * Numeric field - exact match only, no wildcard support.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_SKILL_ID = 3;
     */
    SKILL_ID = 3,

    /**
     * Query for a skill name.
     * Supports wildcard patterns: "python*", "*script", "*machine*learning*", "Pytho?"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_SKILL_NAME = 4;
     */
    SKILL_NAME = 4,

    /**
     * Query for a locator type.
     * Supports wildcard patterns: "http*", "ftp*", "*docker*"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_LOCATOR = 5;
     */
    LOCATOR = 5,

    /**
     * Query for a module name.
     * Supports wildcard patterns: "*-plugin", "*-module", "core*", "mod-?"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_MODULE_NAME = 6;
     */
    MODULE_NAME = 6,

    /**
     * Query for a domain ID.
     * Numeric field - exact match only, no wildcard support.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_DOMAIN_ID = 7;
     */
    DOMAIN_ID = 7,

    /**
     * Query for a domain name.
     * Supports wildcard patterns: "*education*", "healthcare/*", "*technology"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_DOMAIN_NAME = 8;
     */
    DOMAIN_NAME = 8,

    /**
     * Query for a record's created_at timestamp.
     * Supports wildcard patterns for date strings: "2025-*", ">=2025-01-01"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_CREATED_AT = 9;
     */
    CREATED_AT = 9,

    /**
     * Query for a record author.
     * Supports wildcard patterns: "AGNTCY*", "*@example.com", "*Team*"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_AUTHOR = 10;
     */
    AUTHOR = 10,

    /**
     * Query for a schema version.
     * Supports wildcard patterns: "0.7.*", "0.*", "1.0.?"
     *
     * @generated from enum value: RECORD_QUERY_TYPE_SCHEMA_VERSION = 11;
     */
    SCHEMA_VERSION = 11,

    /**
     * Query for a module ID.
     * Numeric field - exact match only, no wildcard support.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_MODULE_ID = 12;
     */
    MODULE_ID = 12,

    /**
     * Query for verified records (name ownership verified via JWKS).
     * Boolean field - use "true" or "false" as value.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_VERIFIED = 13;
     */
    VERIFIED = 13,
}

/**
 * Defines a list of supported record query types.
 *
 * @generated from enum agntcy.dir.routing.v1.RecordQueryType
 */
declare enum RecordQueryType_2 {
    /**
     * Unspecified query type.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,

    /**
     * Query for a skill name.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_SKILL = 1;
     */
    SKILL = 1,

    /**
     * Query for a locator type.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_LOCATOR = 2;
     */
    LOCATOR = 2,

    /**
     * Query for a domain name.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_DOMAIN = 3;
     */
    DOMAIN = 3,

    /**
     * Query for a module name.
     *
     * @generated from enum value: RECORD_QUERY_TYPE_MODULE = 4;
     */
    MODULE = 4,
}

/**
 * Describes the enum agntcy.dir.routing.v1.RecordQueryType.
 */
declare const RecordQueryTypeSchema: GenEnum<RecordQueryType_2>;

/**
 * Describes the enum agntcy.dir.search.v1.RecordQueryType.
 */
declare const RecordQueryTypeSchema_2: GenEnum<RecordQueryType>;

/**
 * Defines a reference or a globally unique content identifier of a record.
 *
 * @generated from message agntcy.dir.core.v1.RecordRef
 */
declare type RecordRef = Message<"agntcy.dir.core.v1.RecordRef"> & {
    /**
     * Globally-unique content identifier (CID) of the record.
     * Specs: https://github.com/multiformats/cid
     *
     * @generated from field: string cid = 1;
     */
    cid: string;
};

/**
 * RecordReferrer represents a referrer object or an association
 * to a record. The actual structure of the referrer object can vary
 * depending on the type of referrer (e.g., signature, public key, etc.).
 *
 * RecordReferrer types in the `agntcy.dir.` namespace are reserved for
 * Directory-specific schemas and will be validated across Dir services.
 *
 * @generated from message agntcy.dir.core.v1.RecordReferrer
 */
declare type RecordReferrer = Message<"agntcy.dir.core.v1.RecordReferrer"> & {
    /**
     * The type of the referrer.
     * For example, "agntcy.dir.sign.v1.Signature" for signatures.
     *
     * @generated from field: string type = 1;
     */
    type: string;

    /**
     * Record reference to which this referrer is associated.
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 2;
     */
    recordRef?: RecordRef;

    /**
     * Annotations attached to the referrer object.
     *
     * @generated from field: map<string, string> annotations = 3;
     */
    annotations: { [key: string]: string };

    /**
     * Creation timestamp of the record in the RFC3339 format.
     * Specs: https://www.rfc-editor.org/rfc/rfc3339.html
     *
     * @generated from field: string created_at = 4;
     */
    createdAt: string;

    /**
     * The actual data of the referrer.
     *
     * @generated from field: google.protobuf.Struct data = 5;
     */
    data?: JsonObject;
};

/**
 * Describes the message agntcy.dir.core.v1.RecordReferrer.
 * Use `create(RecordReferrerSchema)` to create a new message.
 */
declare const RecordReferrerSchema: GenMessage<RecordReferrer>;

/**
 * @generated from message agntcy.dir.routing.v1.RecordRefs
 */
declare type RecordRefs = Message<"agntcy.dir.routing.v1.RecordRefs"> & {
    /**
     * @generated from field: repeated agntcy.dir.core.v1.RecordRef refs = 1;
     */
    refs: RecordRef[];
};

/**
 * Describes the message agntcy.dir.core.v1.RecordRef.
 * Use `create(RecordRefSchema)` to create a new message.
 */
declare const RecordRefSchema: GenMessage<RecordRef>;

/**
 * Describes the message agntcy.dir.routing.v1.RecordRefs.
 * Use `create(RecordRefsSchema)` to create a new message.
 */
declare const RecordRefsSchema: GenMessage<RecordRefs>;

/**
 * Describes the message agntcy.dir.core.v1.Record.
 * Use `create(RecordSchema)` to create a new message.
 */
declare const RecordSchema: GenMessage<Record_2>;

/**
 * @generated from message agntcy.dir.store.v1.RequestRegistryCredentialsRequest
 */
declare type RequestRegistryCredentialsRequest = Message<"agntcy.dir.store.v1.RequestRegistryCredentialsRequest"> & {
};

/**
 * Describes the message agntcy.dir.store.v1.RequestRegistryCredentialsRequest.
 * Use `create(RequestRegistryCredentialsRequestSchema)` to create a new message.
 */
declare const RequestRegistryCredentialsRequestSchema: GenMessage<RequestRegistryCredentialsRequest>;

/**
 * @generated from message agntcy.dir.store.v1.RequestRegistryCredentialsResponse
 */
declare type RequestRegistryCredentialsResponse = Message<"agntcy.dir.store.v1.RequestRegistryCredentialsResponse"> & {
    /**
     * Success status of the credential negotiation
     *
     * @generated from field: bool success = 1;
     */
    success: boolean;

    /**
     * Error message if negotiation failed
     *
     * @generated from field: string error_message = 2;
     */
    errorMessage: string;

    /**
     * Address of the remote Registry (e.g., "ghcr.io", "dir-zot.default.svc.cluster.local:5000")
     *
     * @generated from field: string registry_address = 3;
     */
    registryAddress: string;

    /**
     * Repository name within the registry (e.g., "user/dir-test/dir", "dir")
     *
     * @generated from field: string repository_name = 6;
     */
    repositoryName: string;

    /**
     * Registry credentials (oneof based on credential type)
     *
     * @generated from oneof agntcy.dir.store.v1.RequestRegistryCredentialsResponse.credentials
     */
    credentials: {
        /**
         * CertificateCredentials certificate = 5;
         *
         * @generated from field: agntcy.dir.store.v1.BasicAuthCredentials basic_auth = 4;
         */
        value: BasicAuthCredentials;
        case: "basicAuth";
    } | { case: undefined; value?: undefined };

    /**
     * Whether the registry uses plain HTTP (insecure) instead of HTTPS
     * When true, TLS should be disabled for connections to this registry
     *
     * @generated from field: bool insecure = 7;
     */
    insecure: boolean;
};

/**
 * Describes the message agntcy.dir.store.v1.RequestRegistryCredentialsResponse.
 * Use `create(RequestRegistryCredentialsResponseSchema)` to create a new message.
 */
declare const RequestRegistryCredentialsResponseSchema: GenMessage<RequestRegistryCredentialsResponse>;

/**
 * ResolveRequest is the request for resolving a record reference to CIDs.
 *
 * @generated from message agntcy.dir.naming.v1.ResolveRequest
 */
declare type ResolveRequest = Message<"agntcy.dir.naming.v1.ResolveRequest"> & {
    /**
     * The name of the record to resolve (e.g., "cisco.com/agent").
     *
     * @generated from field: string name = 1;
     */
    name: string;

    /**
     * Optional version to resolve to (e.g., "v1.0.0").
     *
     * @generated from field: optional string version = 2;
     */
    version?: string;
};

/**
 * Describes the message agntcy.dir.naming.v1.ResolveRequest.
 * Use `create(ResolveRequestSchema)` to create a new message.
 */
declare const ResolveRequestSchema: GenMessage<ResolveRequest>;

/**
 * ResolveResponse is the response containing the resolved records.
 *
 * @generated from message agntcy.dir.naming.v1.ResolveResponse
 */
declare type ResolveResponse = Message<"agntcy.dir.naming.v1.ResolveResponse"> & {
    /**
     * The resolved record references (newest first by created_at).
     *
     * @generated from field: repeated agntcy.dir.core.v1.NamedRecordRef records = 1;
     */
    records: NamedRecordRef[];
};

/**
 * Describes the message agntcy.dir.naming.v1.ResolveResponse.
 * Use `create(ResolveResponseSchema)` to create a new message.
 */
declare const ResolveResponseSchema: GenMessage<ResolveResponse>;

export declare namespace routing_v1 {
    export {
        file_agntcy_dir_routing_v1_peer,
        Peer,
        PeerSchema,
        PeerConnectionType,
        PeerConnectionTypeSchema,
        file_agntcy_dir_routing_v1_publication_service,
        CreatePublicationResponse,
        CreatePublicationResponseSchema,
        ListPublicationsRequest,
        ListPublicationsRequestSchema,
        ListPublicationsItem,
        ListPublicationsItemSchema,
        GetPublicationRequest,
        GetPublicationRequestSchema,
        GetPublicationResponse,
        GetPublicationResponseSchema,
        PublicationStatus,
        PublicationStatusSchema,
        PublicationService,
        file_agntcy_dir_routing_v1_record_query,
        RecordQuery_2 as RecordQuery,
        RecordQuerySchema,
        RecordQueryType_2 as RecordQueryType,
        RecordQueryTypeSchema,
        file_agntcy_dir_routing_v1_routing_service,
        PublishRequest,
        PublishRequestSchema,
        UnpublishRequest,
        UnpublishRequestSchema,
        RecordRefs,
        RecordRefsSchema,
        RecordQueries,
        RecordQueriesSchema,
        SearchRequest,
        SearchRequestSchema,
        SearchResponse,
        SearchResponseSchema,
        ListRequest,
        ListRequestSchema,
        ListResponse,
        ListResponseSchema,
        RoutingService
    }
}

/**
 * Defines an interface for announcement and discovery
 * of records across interconnected network.
 *
 * Middleware should be used to control who can perform these RPCs.
 * Policies for the middleware can be handled via separate service.
 *
 * @generated from service agntcy.dir.routing.v1.RoutingService
 */
declare const RoutingService: GenService<{
    /**
     * Announce to the network that this peer is providing a given record.
     * This enables other peers to discover this record and retrieve it
     * from this peer. Listeners can use this event to perform custom operations,
     * for example by cloning the record.
     *
     * Items need to be periodically republished (eg. 24h) to the network
     * to avoid stale data. Republication should be done in the background.
     *
     * @generated from rpc agntcy.dir.routing.v1.RoutingService.Publish
     */
    publish: {
        methodKind: "unary";
        input: typeof PublishRequestSchema;
        output: typeof EmptySchema;
    },
    /**
     * Stop serving this record to the network. If other peers try
     * to retrieve this record, the peer will refuse the request.
     *
     * @generated from rpc agntcy.dir.routing.v1.RoutingService.Unpublish
     */
    unpublish: {
        methodKind: "unary";
        input: typeof UnpublishRequestSchema;
        output: typeof EmptySchema;
    },
    /**
     * Search records based on the request across the network.
     * This will search the network for the record with the given parameters.
     *
     * It is possible that the records are stale or that they do not exist.
     * Some records may be provided by multiple peers.
     *
     * Results from the search can be used as an input
     * to Pull operation to retrieve the records.
     *
     * @generated from rpc agntcy.dir.routing.v1.RoutingService.Search
     */
    search: {
        methodKind: "server_streaming";
        input: typeof SearchRequestSchema;
        output: typeof SearchResponseSchema;
    },
    /**
     * List all records that this peer is currently providing
     * that match the given parameters.
     * This operation does not interact with the network.
     *
     * @generated from rpc agntcy.dir.routing.v1.RoutingService.List
     */
    list: {
        methodKind: "server_streaming";
        input: typeof ListRequestSchema;
        output: typeof ListResponseSchema;
    },
}>;

export declare namespace search_v1 {
    export {
        file_agntcy_dir_search_v1_record_query,
        RecordQuery,
        RecordQuerySchema_2 as RecordQuerySchema,
        RecordQueryType,
        RecordQueryTypeSchema_2 as RecordQueryTypeSchema,
        file_agntcy_dir_search_v1_search_service,
        SearchCIDsRequest,
        SearchCIDsRequestSchema,
        SearchRecordsRequest,
        SearchRecordsRequestSchema,
        SearchCIDsResponse,
        SearchCIDsResponseSchema,
        SearchRecordsResponse,
        SearchRecordsResponseSchema,
        SearchService
    }
}

/**
 * @generated from message agntcy.dir.search.v1.SearchCIDsRequest
 */
declare type SearchCIDsRequest = Message<"agntcy.dir.search.v1.SearchCIDsRequest"> & {
    /**
     * List of queries to match against the records.
     *
     * @generated from field: repeated agntcy.dir.search.v1.RecordQuery queries = 1;
     */
    queries: RecordQuery[];

    /**
     * Optional limit on the number of results to return.
     *
     * @generated from field: optional uint32 limit = 2;
     */
    limit?: number;

    /**
     * Optional offset for pagination of results.
     *
     * @generated from field: optional uint32 offset = 3;
     */
    offset?: number;
};

/**
 * Describes the message agntcy.dir.search.v1.SearchCIDsRequest.
 * Use `create(SearchCIDsRequestSchema)` to create a new message.
 */
declare const SearchCIDsRequestSchema: GenMessage<SearchCIDsRequest>;

/**
 * @generated from message agntcy.dir.search.v1.SearchCIDsResponse
 */
declare type SearchCIDsResponse = Message<"agntcy.dir.search.v1.SearchCIDsResponse"> & {
    /**
     * The CID of the record that matches the search criteria.
     *
     * @generated from field: string record_cid = 1;
     */
    recordCid: string;
};

/**
 * Describes the message agntcy.dir.search.v1.SearchCIDsResponse.
 * Use `create(SearchCIDsResponseSchema)` to create a new message.
 */
declare const SearchCIDsResponseSchema: GenMessage<SearchCIDsResponse>;

/**
 * @generated from message agntcy.dir.search.v1.SearchRecordsRequest
 */
declare type SearchRecordsRequest = Message<"agntcy.dir.search.v1.SearchRecordsRequest"> & {
    /**
     * List of queries to match against the records.
     *
     * @generated from field: repeated agntcy.dir.search.v1.RecordQuery queries = 1;
     */
    queries: RecordQuery[];

    /**
     * Optional limit on the number of results to return.
     *
     * @generated from field: optional uint32 limit = 2;
     */
    limit?: number;

    /**
     * Optional offset for pagination of results.
     *
     * @generated from field: optional uint32 offset = 3;
     */
    offset?: number;
};

/**
 * Describes the message agntcy.dir.search.v1.SearchRecordsRequest.
 * Use `create(SearchRecordsRequestSchema)` to create a new message.
 */
declare const SearchRecordsRequestSchema: GenMessage<SearchRecordsRequest>;

/**
 * @generated from message agntcy.dir.search.v1.SearchRecordsResponse
 */
declare type SearchRecordsResponse = Message<"agntcy.dir.search.v1.SearchRecordsResponse"> & {
    /**
     * The full record that matches the search criteria.
     *
     * @generated from field: agntcy.dir.core.v1.Record record = 1;
     */
    record?: Record_2;
};

/**
 * Describes the message agntcy.dir.search.v1.SearchRecordsResponse.
 * Use `create(SearchRecordsResponseSchema)` to create a new message.
 */
declare const SearchRecordsResponseSchema: GenMessage<SearchRecordsResponse>;

/**
 * @generated from message agntcy.dir.routing.v1.SearchRequest
 */
declare type SearchRequest = Message<"agntcy.dir.routing.v1.SearchRequest"> & {
    /**
     * List of queries to match against the records.
     *
     * @generated from field: repeated agntcy.dir.routing.v1.RecordQuery queries = 1;
     */
    queries: RecordQuery_2[];

    /**
     * Minimal target query match score.
     * For example, if min_match_score=2, it will return records that match
     * at least two of the queries.
     * If not set, it will return records that match at least one query.
     *
     * @generated from field: optional uint32 min_match_score = 2;
     */
    minMatchScore?: number;

    /**
     * Limit the number of results returned.
     * If not set, it will return all discovered records.
     * Note that this is a soft limit, as the search may return more results
     * than the limit if there are multiple peers providing the same record.
     *
     * @generated from field: optional uint32 limit = 3;
     */
    limit?: number;
};

/**
 * Describes the message agntcy.dir.routing.v1.SearchRequest.
 * Use `create(SearchRequestSchema)` to create a new message.
 */
declare const SearchRequestSchema: GenMessage<SearchRequest>;

/**
 * @generated from message agntcy.dir.routing.v1.SearchResponse
 */
declare type SearchResponse = Message<"agntcy.dir.routing.v1.SearchResponse"> & {
    /**
     * The record that matches the search query.
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 1;
     */
    recordRef?: RecordRef;

    /**
     * The peer that provided the record.
     *
     * @generated from field: agntcy.dir.routing.v1.Peer peer = 2;
     */
    peer?: Peer;

    /**
     * The queries that were matched.
     *
     * @generated from field: repeated agntcy.dir.routing.v1.RecordQuery match_queries = 3;
     */
    matchQueries: RecordQuery_2[];

    /**
     * The score of the search match.
     *
     * @generated from field: uint32 match_score = 4;
     */
    matchScore: number;
};

/**
 * Describes the message agntcy.dir.routing.v1.SearchResponse.
 * Use `create(SearchResponseSchema)` to create a new message.
 */
declare const SearchResponseSchema: GenMessage<SearchResponse>;

/**
 * @generated from service agntcy.dir.search.v1.SearchService
 */
declare const SearchService: GenService<{
    /**
     * Search for record CIDs that match the given parameters.
     * Returns only CIDs for efficient lookups and piping to other commands.
     * This operation does not interact with the network.
     *
     * @generated from rpc agntcy.dir.search.v1.SearchService.SearchCIDs
     */
    searchCIDs: {
        methodKind: "server_streaming";
        input: typeof SearchCIDsRequestSchema;
        output: typeof SearchCIDsResponseSchema;
    },
    /**
     * Search for full records that match the given parameters.
     * Returns complete record data including all metadata, skills, domains, etc.
     * This operation does not interact with the network.
     *
     * @generated from rpc agntcy.dir.search.v1.SearchService.SearchRecords
     */
    searchRecords: {
        methodKind: "server_streaming";
        input: typeof SearchRecordsRequestSchema;
        output: typeof SearchRecordsResponseSchema;
    },
}>;

export declare namespace sign_v1 {
    export {
        file_agntcy_dir_sign_v1_sign_service,
        SignRequest,
        SignRequestSchema,
        SignRequestProvider,
        SignRequestProviderSchema,
        SignWithOIDC,
        SignWithOIDCSchema,
        SignWithOIDC_SignOpts,
        SignWithOIDC_SignOptsSchema,
        SignWithKey,
        SignWithKeySchema,
        SignResponse,
        SignResponseSchema,
        VerifyRequest,
        VerifyRequestSchema,
        VerifyResponse,
        VerifyResponseSchema,
        SignService,
        file_agntcy_dir_sign_v1_signature,
        Signature,
        SignatureSchema,
        file_agntcy_dir_sign_v1_public_key,
        PublicKey,
        PublicKeySchema
    }
}

/**
 * Signature is the signing data associated with a Record.
 * Multiple signatures can be associated with a single Record, 
 * ie 1 record : N record signatures.
 *
 * Storage and management of signatures is provided via
 * StoreService as a RecordReferrer object.
 *
 * Signature can be encoded into RecordReferrer object as follows:
 *   type = "agntcy.dir.sign.v1.Signature"
 *   data = Signature message encoded as JSON
 *
 * @generated from message agntcy.dir.sign.v1.Signature
 */
declare type Signature = Message<"agntcy.dir.sign.v1.Signature"> & {
    /**
     * Metadata associated with the signature.
     *
     * @generated from field: map<string, string> annotations = 1;
     */
    annotations: { [key: string]: string };

    /**
     * Signing timestamp of the record in the RFC3339 format.
     * Specs: https://www.rfc-editor.org/rfc/rfc3339.html
     *
     * @generated from field: string signed_at = 2;
     */
    signedAt: string;

    /**
     * The signature algorithm used (e.g., "ECDSA_P256_SHA256").
     *
     * @generated from field: string algorithm = 3;
     */
    algorithm: string;

    /**
     * Base64-encoded signature.
     *
     * @generated from field: string signature = 4;
     */
    signature: string;

    /**
     * Base64-encoded signing certificate.
     *
     * @generated from field: string certificate = 5;
     */
    certificate: string;

    /**
     * Type of the signature content bundle.
     *
     * @generated from field: string content_type = 6;
     */
    contentType: string;

    /**
     * Base64-encoded signature bundle produced by the signer.
     * It is up to the client to interpret the content of the bundle.
     *
     * @generated from field: string content_bundle = 7;
     */
    contentBundle: string;
};

/**
 * Describes the message agntcy.dir.sign.v1.Signature.
 * Use `create(SignatureSchema)` to create a new message.
 */
declare const SignatureSchema: GenMessage<Signature>;

/**
 * @generated from message agntcy.dir.sign.v1.SignRequest
 */
declare type SignRequest = Message<"agntcy.dir.sign.v1.SignRequest"> & {
    /**
     * Record reference to be signed
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 1;
     */
    recordRef?: RecordRef;

    /**
     * Signing provider to use
     *
     * @generated from field: agntcy.dir.sign.v1.SignRequestProvider provider = 2;
     */
    provider?: SignRequestProvider;
};

/**
 * @generated from message agntcy.dir.sign.v1.SignRequestProvider
 */
declare type SignRequestProvider = Message<"agntcy.dir.sign.v1.SignRequestProvider"> & {
    /**
     * @generated from oneof agntcy.dir.sign.v1.SignRequestProvider.request
     */
    request: {
        /**
         * Sign with OIDC provider
         *
         * @generated from field: agntcy.dir.sign.v1.SignWithOIDC oidc = 1;
         */
        value: SignWithOIDC;
        case: "oidc";
    } | {
        /**
         * Sign with PEM-encoded public key
         *
         * @generated from field: agntcy.dir.sign.v1.SignWithKey key = 2;
         */
        value: SignWithKey;
        case: "key";
    } | { case: undefined; value?: undefined };
};

/**
 * Describes the message agntcy.dir.sign.v1.SignRequestProvider.
 * Use `create(SignRequestProviderSchema)` to create a new message.
 */
declare const SignRequestProviderSchema: GenMessage<SignRequestProvider>;

/**
 * Describes the message agntcy.dir.sign.v1.SignRequest.
 * Use `create(SignRequestSchema)` to create a new message.
 */
declare const SignRequestSchema: GenMessage<SignRequest>;

/**
 * @generated from message agntcy.dir.sign.v1.SignResponse
 */
declare type SignResponse = Message<"agntcy.dir.sign.v1.SignResponse"> & {
    /**
     * Cryptographic signature of the record
     *
     * @generated from field: agntcy.dir.sign.v1.Signature signature = 1;
     */
    signature?: Signature;
};

/**
 * Describes the message agntcy.dir.sign.v1.SignResponse.
 * Use `create(SignResponseSchema)` to create a new message.
 */
declare const SignResponseSchema: GenMessage<SignResponse>;

/**
 * SignService provides methods to sign and verify records.
 *
 * @generated from service agntcy.dir.sign.v1.SignService
 */
declare const SignService: GenService<{
    /**
     * Sign record using keyless OIDC based provider or using PEM-encoded private key with an optional passphrase
     *
     * @generated from rpc agntcy.dir.sign.v1.SignService.Sign
     */
    sign: {
        methodKind: "unary";
        input: typeof SignRequestSchema;
        output: typeof SignResponseSchema;
    },
    /**
     * Verify signed record using keyless OIDC based provider or using PEM-encoded formatted PEM public key encrypted
     *
     * @generated from rpc agntcy.dir.sign.v1.SignService.Verify
     */
    verify: {
        methodKind: "unary";
        input: typeof VerifyRequestSchema;
        output: typeof VerifyResponseSchema;
    },
}>;

/**
 * @generated from message agntcy.dir.sign.v1.SignWithKey
 */
declare type SignWithKey = Message<"agntcy.dir.sign.v1.SignWithKey"> & {
    /**
     * Private key used for signing
     *
     * @generated from field: bytes private_key = 1;
     */
    privateKey: Uint8Array;

    /**
     * Password to unlock the private key
     *
     * @generated from field: optional bytes password = 2;
     */
    password?: Uint8Array;
};

/**
 * Describes the message agntcy.dir.sign.v1.SignWithKey.
 * Use `create(SignWithKeySchema)` to create a new message.
 */
declare const SignWithKeySchema: GenMessage<SignWithKey>;

/**
 * @generated from message agntcy.dir.sign.v1.SignWithOIDC
 */
declare type SignWithOIDC = Message<"agntcy.dir.sign.v1.SignWithOIDC"> & {
    /**
     * Token for OIDC provider
     *
     * @generated from field: string id_token = 1;
     */
    idToken: string;

    /**
     * Signing options for OIDC
     *
     * @generated from field: agntcy.dir.sign.v1.SignWithOIDC.SignOpts options = 2;
     */
    options?: SignWithOIDC_SignOpts;
};

/**
 * List of sign options for OIDC
 *
 * @generated from message agntcy.dir.sign.v1.SignWithOIDC.SignOpts
 */
declare type SignWithOIDC_SignOpts = Message<"agntcy.dir.sign.v1.SignWithOIDC.SignOpts"> & {
    /**
     * Fulcio authority access URL (default value: https://fulcio.sigstage.dev)
     *
     * @generated from field: optional string fulcio_url = 1;
     */
    fulcioUrl?: string;

    /**
     * Rekor validator access URL (default value: https://rekor.sigstage.dev)
     *
     * @generated from field: optional string rekor_url = 2;
     */
    rekorUrl?: string;

    /**
     * Timestamp authority access URL (default value: https://timestamp.sigstage.dev/api/v1/timestamp)
     *
     * @generated from field: optional string timestamp_url = 3;
     */
    timestampUrl?: string;

    /**
     * OIDC provider access URL (default value: https://oauth2.sigstage.dev/auth)
     *
     * @generated from field: optional string oidc_provider_url = 4;
     */
    oidcProviderUrl?: string;
};

/**
 * Describes the message agntcy.dir.sign.v1.SignWithOIDC.SignOpts.
 * Use `create(SignWithOIDC_SignOptsSchema)` to create a new message.
 */
declare const SignWithOIDC_SignOptsSchema: GenMessage<SignWithOIDC_SignOpts>;

/**
 * Describes the message agntcy.dir.sign.v1.SignWithOIDC.
 * Use `create(SignWithOIDCSchema)` to create a new message.
 */
declare const SignWithOIDCSchema: GenMessage<SignWithOIDC>;

export declare namespace store_v1 {
    export {
        file_agntcy_dir_store_v1_store_service,
        PushReferrerRequest,
        PushReferrerRequestSchema,
        PushReferrerResponse,
        PushReferrerResponseSchema,
        PullReferrerRequest,
        PullReferrerRequestSchema,
        PullReferrerResponse,
        PullReferrerResponseSchema,
        StoreService,
        file_agntcy_dir_store_v1_sync_service,
        CreateSyncRequest,
        CreateSyncRequestSchema,
        CreateSyncResponse,
        CreateSyncResponseSchema,
        ListSyncsRequest,
        ListSyncsRequestSchema,
        ListSyncsItem,
        ListSyncsItemSchema,
        GetSyncRequest,
        GetSyncRequestSchema,
        GetSyncResponse,
        GetSyncResponseSchema,
        DeleteSyncRequest,
        DeleteSyncRequestSchema,
        DeleteSyncResponse,
        DeleteSyncResponseSchema,
        RequestRegistryCredentialsRequest,
        RequestRegistryCredentialsRequestSchema,
        RequestRegistryCredentialsResponse,
        RequestRegistryCredentialsResponseSchema,
        BasicAuthCredentials,
        BasicAuthCredentialsSchema,
        SyncStatus,
        SyncStatusSchema,
        SyncService
    }
}

/**
 * Defines an interface for content-addressable storage
 * service for objects.
 *
 * Max object size: 4MB (to fully fit in a single request)
 * Max metadata size: 100KB
 *
 * Store service can be implemented by various storage backends,
 * such as local file system, OCI registry, etc.
 *
 * Middleware should be used to control who can perform these RPCs.
 * Policies for the middleware can be handled via separate service.
 *
 * Each operation is performed sequentially, meaning that
 * for the N-th request, N-th response will be returned.
 * If an error occurs, the stream will be cancelled.
 *
 * @generated from service agntcy.dir.store.v1.StoreService
 */
declare const StoreService: GenService<{
    /**
     * Push performs write operation for given records.
     *
     * @generated from rpc agntcy.dir.store.v1.StoreService.Push
     */
    push: {
        methodKind: "bidi_streaming";
        input: typeof RecordSchema;
        output: typeof RecordRefSchema;
    },
    /**
     * Pull performs read operation for given records.
     *
     * @generated from rpc agntcy.dir.store.v1.StoreService.Pull
     */
    pull: {
        methodKind: "bidi_streaming";
        input: typeof RecordRefSchema;
        output: typeof RecordSchema;
    },
    /**
     * Lookup resolves basic metadata for the records.
     *
     * @generated from rpc agntcy.dir.store.v1.StoreService.Lookup
     */
    lookup: {
        methodKind: "bidi_streaming";
        input: typeof RecordRefSchema;
        output: typeof RecordMetaSchema;
    },
    /**
     * Remove performs delete operation for the records.
     *
     * @generated from rpc agntcy.dir.store.v1.StoreService.Delete
     */
    delete: {
        methodKind: "client_streaming";
        input: typeof RecordRefSchema;
        output: typeof EmptySchema;
    },
    /**
     * PushReferrer performs write operation for record referrers.
     *
     * @generated from rpc agntcy.dir.store.v1.StoreService.PushReferrer
     */
    pushReferrer: {
        methodKind: "bidi_streaming";
        input: typeof PushReferrerRequestSchema;
        output: typeof PushReferrerResponseSchema;
    },
    /**
     * PullReferrer performs read operation for record referrers.
     *
     * @generated from rpc agntcy.dir.store.v1.StoreService.PullReferrer
     */
    pullReferrer: {
        methodKind: "bidi_streaming";
        input: typeof PullReferrerRequestSchema;
        output: typeof PullReferrerResponseSchema;
    },
}>;

/**
 * SyncService provides functionality for synchronizing objects between Directory nodes.
 *
 * This service enables one-way synchronization from a remote Directory node to the local node,
 * allowing distributed Directory instances to share and replicate objects. The service supports
 * both on-demand synchronization and tracking of sync operations through their lifecycle.
 *
 * @generated from service agntcy.dir.store.v1.SyncService
 */
declare const SyncService: GenService<{
    /**
     * CreateSync initiates a new synchronization operation from a remote Directory node.
     *
     * The operation is non-blocking and returns immediately with a sync ID that can be used
     * to track progress and manage the sync operation.
     *
     * @generated from rpc agntcy.dir.store.v1.SyncService.CreateSync
     */
    createSync: {
        methodKind: "unary";
        input: typeof CreateSyncRequestSchema;
        output: typeof CreateSyncResponseSchema;
    },
    /**
     * ListSyncs returns a stream of all sync operations known to the system.
     *
     * This includes active, completed, and failed synchronizations.
     *
     * @generated from rpc agntcy.dir.store.v1.SyncService.ListSyncs
     */
    listSyncs: {
        methodKind: "server_streaming";
        input: typeof ListSyncsRequestSchema;
        output: typeof ListSyncsItemSchema;
    },
    /**
     * GetSync retrieves detailed status information for a specific synchronization.
     *
     * @generated from rpc agntcy.dir.store.v1.SyncService.GetSync
     */
    getSync: {
        methodKind: "unary";
        input: typeof GetSyncRequestSchema;
        output: typeof GetSyncResponseSchema;
    },
    /**
     * DeleteSync removes a synchronization operation from the system.
     *
     * @generated from rpc agntcy.dir.store.v1.SyncService.DeleteSync
     */
    deleteSync: {
        methodKind: "unary";
        input: typeof DeleteSyncRequestSchema;
        output: typeof DeleteSyncResponseSchema;
    },
    /**
     * RequestRegistryCredentials requests registry credentials between two Directory nodes.
     *
     * This RPC allows a requesting node to authenticate with this node and obtain
     * temporary registry credentials for secure Zot-based synchronization.
     *
     * @generated from rpc agntcy.dir.store.v1.SyncService.RequestRegistryCredentials
     */
    requestRegistryCredentials: {
        methodKind: "unary";
        input: typeof RequestRegistryCredentialsRequestSchema;
        output: typeof RequestRegistryCredentialsResponseSchema;
    },
}>;

/**
 * SyncStatus enumeration defines the possible states of a synchronization operation.
 *
 * @generated from enum agntcy.dir.store.v1.SyncStatus
 */
declare enum SyncStatus {
    /**
     * Default/unset status - should not be used in practice
     *
     * @generated from enum value: SYNC_STATUS_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,

    /**
     * Sync operation has been created but not yet started
     *
     * @generated from enum value: SYNC_STATUS_PENDING = 1;
     */
    PENDING = 1,

    /**
     * Sync operation is actively discovering and transferring objects
     *
     * @generated from enum value: SYNC_STATUS_IN_PROGRESS = 2;
     */
    IN_PROGRESS = 2,

    /**
     * Sync operation encountered an error and stopped
     *
     * @generated from enum value: SYNC_STATUS_FAILED = 3;
     */
    FAILED = 3,

    /**
     * Sync operation has been marked for deletion but cleanup not yet started
     *
     * @generated from enum value: SYNC_STATUS_DELETE_PENDING = 4;
     */
    DELETE_PENDING = 4,

    /**
     * Sync operation has been successfully deleted and cleaned up
     *
     * @generated from enum value: SYNC_STATUS_DELETED = 5;
     */
    DELETED = 5,

    /**
     * Sync operation has completed successfully
     *
     * @generated from enum value: SYNC_STATUS_COMPLETED = 6;
     */
    COMPLETED = 6,
}

/**
 * Describes the enum agntcy.dir.store.v1.SyncStatus.
 */
declare const SyncStatusSchema: GenEnum<SyncStatus>;

/**
 * @generated from message agntcy.dir.routing.v1.UnpublishRequest
 */
declare type UnpublishRequest = Message<"agntcy.dir.routing.v1.UnpublishRequest"> & {
    /**
     * @generated from oneof agntcy.dir.routing.v1.UnpublishRequest.request
     */
    request: {
        /**
         * References to the records to be unpublished.
         *
         * @generated from field: agntcy.dir.routing.v1.RecordRefs record_refs = 1;
         */
        value: RecordRefs;
        case: "recordRefs";
    } | {
        /**
         * Queries to match against the records to be unpublished.
         *
         * @generated from field: agntcy.dir.routing.v1.RecordQueries queries = 2;
         */
        value: RecordQueries;
        case: "queries";
    } | { case: undefined; value?: undefined };
};

/**
 * Describes the message agntcy.dir.routing.v1.UnpublishRequest.
 * Use `create(UnpublishRequestSchema)` to create a new message.
 */
declare const UnpublishRequestSchema: GenMessage<UnpublishRequest>;

/**
 * Verification represents the result of verifying a record's name ownership.
 * It uses a oneof to support different verification types.
 *
 * @generated from message agntcy.dir.naming.v1.Verification
 */
declare type Verification = Message<"agntcy.dir.naming.v1.Verification"> & {
    /**
     * @generated from oneof agntcy.dir.naming.v1.Verification.info
     */
    info: {
        /**
         * Domain verification details.
         *
         * Future verification types can be added here.
         *
         * @generated from field: agntcy.dir.naming.v1.DomainVerification domain = 1;
         */
        value: DomainVerification;
        case: "domain";
    } | { case: undefined; value?: undefined };
};

/**
 * Describes the message agntcy.dir.naming.v1.Verification.
 * Use `create(VerificationSchema)` to create a new message.
 */
declare const VerificationSchema: GenMessage<Verification>;

/**
 * @generated from message agntcy.dir.sign.v1.VerifyRequest
 */
declare type VerifyRequest = Message<"agntcy.dir.sign.v1.VerifyRequest"> & {
    /**
     * Record reference to be verified
     *
     * @generated from field: agntcy.dir.core.v1.RecordRef record_ref = 1;
     */
    recordRef?: RecordRef;
};

/**
 * Describes the message agntcy.dir.sign.v1.VerifyRequest.
 * Use `create(VerifyRequestSchema)` to create a new message.
 */
declare const VerifyRequestSchema: GenMessage<VerifyRequest>;

/**
 * @generated from message agntcy.dir.sign.v1.VerifyResponse
 */
declare type VerifyResponse = Message<"agntcy.dir.sign.v1.VerifyResponse"> & {
    /**
     * The verify process result
     *
     * @generated from field: bool success = 1;
     */
    success: boolean;

    /**
     * Optional error message if verification failed
     *
     * @generated from field: optional string error_message = 2;
     */
    errorMessage?: string;
};

/**
 * Describes the message agntcy.dir.sign.v1.VerifyResponse.
 * Use `create(VerifyResponseSchema)` to create a new message.
 */
declare const VerifyResponseSchema: GenMessage<VerifyResponse>;

export { }

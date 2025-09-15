const { execSync } = require('child_process');
const { readFileSync, rmSync } = require('node:fs');
const { validate: isValidUUID } = require('uuid');

const { Client, Config } = require('../../v1/client');
const core_record_pb2 = require('@buf/agntcy_dir.grpc_node/core/v1/record_pb');
const extension_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/extension_pb');
const record_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/record_pb');
const signature_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/signature_pb');
const skill_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/skill_pb');
const locator_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/locator_pb');
const record_query_type = require('@buf/agntcy_dir.grpc_web/routing/v1/record_query_pb');
const routing_types = require('@buf/agntcy_dir.grpc_node/routing/v1/routing_service_pb');
const search_types = require('@buf/agntcy_dir.grpc_node/search/v1/search_service_pb');
const store_types = require('@buf/agntcy_dir.grpc_node/store/v1/store_service_pb');
const search_query_type = require('@buf/agntcy_dir.grpc_node/search/v1/record_query_pb');
const sign_types = require('@buf/agntcy_dir.grpc_node/sign/v1/sign_service_pb');
const sync_types = require('@buf/agntcy_dir.grpc_node/store/v1/sync_service_pb');

class GeneratedRecord {
    constructor(ref, record) {
        this.ref = ref;
        this.record = record;
    }
}

describe('Client', () => {
    async function initRecords(count, test_function_name, push = true, publish = false) {
        const test_records = [];

        for (let index = 0; index < count; index++) {
            const exampleRecord = new record_pb2.Record();
            exampleRecord.setName(test_function_name + " " + index);
            exampleRecord.setVersion('v3');
            exampleRecord.setSchemaVersion("v0.5.0");

            const skill = new skill_pb2.Skill();
            skill.setName('Natural Language Processing');
            skill.setId(1);
            exampleRecord.addSkills(skill);

            const locator = new locator_pb2.Locator();
            locator.setType("ipv4")
            locator.setUrl("127.0.0.1");
            exampleRecord.addLocators(locator);

            const extension = new extension_pb2.Extension();
            extension.setName('schema.oasf.agntcy.org/domains/domain-1');
            extension.setVersion('v1');
            exampleRecord.addExtensions(extension);

            const signature = new signature_pb2.Signature();
            exampleRecord.setSignature(signature);

            const test_record = new core_record_pb2.Record();
            test_record.setV3(exampleRecord);

            test_records.push(new GeneratedRecord(null, test_record));
        }

        if (push === true) {
            for (const test_record of test_records) {
                try {
                    references = await client.push([test_record.record]);

                    test_record.ref = new core_record_pb2.RecordRef(references[0].u);
                } catch (error) {
                    throw new Error(error);
                }
            }
        }

        if (publish === true) {
            for (const test_record of test_records) {
                const publish_request = new routing_types.PublishRequest();
                publish_request.setRecordCid(test_record.ref.u[0]);

                let err = null;

                try {
                    client.publish(publish_request);
                } catch (error) {
                    err = error;
                    throw new Error(error);
                }
            }
        }

        return test_records;
    }

    const client = new Client(Config.loadFromEnv());

    afterAll(async () => {
        client.storeClient.close();
        client.routingClient.close();
        client.searchClient.close();
        client.syncClient.close();
    });

    test('push', async () => {
        const generated_records = await initRecords(2, "push", false, false);
        const test_records = [];

        generated_records.forEach(generated_record => {
            test_records.push(generated_record.record);
        });

        let references;

        try {
            references = await client.push(test_records);
        } catch (error) {
            throw new Error(error);
        }

        const test_record_ref1 = new core_record_pb2.RecordRef(references[0].u);
        const test_record_ref2 = new core_record_pb2.RecordRef(references[1].u);

        expect(references).not.toBeNull();
        expect(references).toBeInstanceOf(Array);
        expect(references[0].u[0].length).toBe(59);
        expect(references[1].u[0].length).toBe(59);
        expect(test_record_ref1).not.toBeNull();
        expect(test_record_ref2).not.toBeNull();
        expect(test_record_ref1).toBeInstanceOf(core_record_pb2.RecordRef);
        expect(test_record_ref2).toBeInstanceOf(core_record_pb2.RecordRef);
    });

    test('pull', async () => {
        const generated_records = await initRecords(2, "pull", true, false);
        const test_records_ref = [];

        generated_records.forEach(generated_record => {
            test_records_ref.push(generated_record.ref);
        });

        let pulled_records;

        try {
            pulled_records = await client.pull(test_records_ref);
        } catch (error) {
            throw new Error(error);
        }

        const pulledRecordInstance1 = new core_record_pb2.Record(pulled_records[0]);
        const pulledRecordInstance2 = new core_record_pb2.Record(pulled_records[1]);

        expect(pulled_records).not.toBeNull();
        expect(pulled_records).toBeInstanceOf(Array);
        expect(pulled_records.length).toBe(2);
        expect(pulledRecordInstance1).not.toBeNull();
        expect(pulledRecordInstance2).not.toBeNull();
        expect(pulledRecordInstance1).toBeInstanceOf(core_record_pb2.Record);
        expect(pulledRecordInstance2).toBeInstanceOf(core_record_pb2.Record);

    });

    test('search', async () => {
        _ = await initRecords(2, "search", true, false);

        const search_query = new search_query_type.RecordQuery();
        search_query.setType(search_query_type.RecordQueryType.RECORD_QUERY_TYPE_SKILL_ID);
        search_query.setValue('1');

        const queries = [search_query];

        const search_request = new search_types.SearchRequest();
        search_request.setQueriesList(queries);
        search_request.setLimit(2);

        let objects;

        try {
            objects = await client.search(search_request);
        } catch (error) {
            throw new Error(error);
        }

        objectsInstance = new search_types.SearchResponse(objects);

        expect(objects).not.toBeNull();
        expect(objects).toBeInstanceOf(Array);
        expect(objects.length).toBe(2);
        expect(objectsInstance).not.toBeNull();
        expect(objectsInstance).toBeInstanceOf(search_types.SearchResponse);
        expect(objectsInstance.u.length).toBe(2);
    });

    test('lookup', async () => {
        const generated_records = await initRecords(2, "lookup", true, false);
        const test_records_ref = [];

        generated_records.forEach(generated_record => {
            test_records_ref.push(generated_record.ref);
        });

        let metadatas;

        try {
            metadatas = await client.lookup(test_records_ref);
        } catch (error) {
            throw new Error(error);
        }

        expect(metadatas).not.toBeNull();
        expect(metadatas).toBeInstanceOf(Array);
        expect(metadatas.length).toBe(2);
    });

    test('publish', async () => {
        const generated_records = await initRecords(1, "publish", true, false);
        const test_records_ref = [];

        generated_records.forEach(generated_record => {
            test_records_ref.push(generated_record.ref);
        });

        const publish_request = new routing_types.PublishRequest();
        publish_request.setRecordCid(test_records_ref[0].u[0]);

        let err = null;

        try {
            client.publish(publish_request);
        } catch (error) {
            err = error;
            throw new Error(error);
        }

        expect(err).toBeNull();

        // no assertion needed, no response
    });

    test('list', async () => {
        _ = await initRecords(1, "list", true, true);

        const query = new record_query_type.RecordQuery();
        query.setType(record_query_type.RECORD_QUERY_TYPE_SKILL);
        query.setValue('/skills/Natural Language Processing/Text Completion');
        const listRequest = new routing_types.ListRequest();
        listRequest.addQueries(query);

        let objects;

        try {
            objects = await client.list(listRequest)
        } catch (error) {
            throw new Error(error);
        }

        const objectsInstance = new routing_types.ListResponse(objects);

        expect(objects).not.toBeNull();
        expect(objects).toBeInstanceOf(Array);
        expect(objects[1].length).not.toBe(0);
        expect(objectsInstance).not.toBeNull();
        expect(objectsInstance).toBeInstanceOf(routing_types.ListResponse);
    });

    test('unpublish', async () => {
        const generated_records = await initRecords(1, "unpublish", true, false);
        const test_records_ref = [];

        generated_records.forEach(generated_record => {
            test_records_ref.push(generated_record.ref);
        });

        let unpublish_request = new routing_types.UnpublishRequest();
        unpublish_request.setRecordCid(test_records_ref[0].u[0]);

        let err = null;

        try {
            client.unpublish(unpublish_request);
        } catch (error) {
            err = error;
            throw new Error(error);
        }

        expect(err).toBeNull();

        // no assertion needed, no response
    });

    test('delete', async () => {
        const generated_records = await initRecords(1, "delete", true, false);
        const test_records_ref = [];

        generated_records.forEach(generated_record => {
            test_records_ref.push(generated_record.ref);
        });

        let err = null;

        try {
            await client.delete(test_records_ref);
        } catch (error) {
            err = error;
            throw new Error(error);
        }

        expect(err).toBeNull();

        // no assertion needed, no response
    });

    test('pushReferrer', async () => {
        const generated_records = await initRecords(2, "pushReferrer", true, false);
        const requests = [];

        generated_records.forEach(generated_record => {
            let push_referrer_request = new store_types.PushReferrerRequest();
            push_referrer_request.setRecordRef(generated_record.ref);
            push_referrer_request.setSignature(new sign_types.Signature());

            requests.push(push_referrer_request);
        });

        let err = null;

        try {
            await client.push_referrer(requests);
        } catch (error) {
            err = error;
            throw new Error(error);
        }

        expect(err).toBeNull();
    });

    test('pullReferrer', async () => {
        const generated_records = await initRecords(2, "pullReferrer", true, false);
        const requests = [];

        generated_records.forEach(generated_record => {
            let pull_referrer_request = new store_types.PullReferrerRequest();
            pull_referrer_request.setRecordRef(generated_record.ref);
            pull_referrer_request.setPullSignature(false);

            requests.push(pull_referrer_request);
        });

        let err = null;

        try {
            await client.pull_referrer(requests);
        } catch (error) {
            err = error;

            if (error.details === "pull referrer not implemented") { err = null; return; }; // Remove when service implemented
            throw new Error(error);
        }

        expect(err).toBeNull();
    });

    test('sign_and_verify', async () => {
        const generated_records = await initRecords(2, "sign_and_verify", true, false);
        const test_records_ref = [];

        generated_records.forEach(generated_record => {
            test_records_ref.push(generated_record.ref);
        });

        rmSync("cosign.key", { force: true });
        rmSync("cosign.pub", { force: true });

        let err = null;

        const key_password = "testing-key";
        shell_env = process.env

        try {
            const cosign_path = process.env["COSIGN_PATH"] || 'cosign';
            execSync(
                `${cosign_path} generate-key-pair`,
                { env: { ...shell_env, COSIGN_PASSWORD: key_password }, encoding: 'utf8', stdio: 'pipe' }
            );
        } catch (error) {
            err = error;
        }

        expect(err).toBeNull();

        let key_file = null;

        try {
            key_file = readFileSync('cosign.key');
        } catch (error) {
            err = error;
        }

        expect(err).toBeNull();

        let key_provider = new sign_types.SignWithKey();
        key_provider.setPrivateKey(key_file);
        key_provider.setPassword(key_password);

        const token = shell_env["OIDC_TOKEN"];
        const provider_url = shell_env["OIDC_PROVIDER_URL"];
        const client_id = shell_env["OIDC_CLIENT_ID"];

        let oidc_options = new sign_types.SignWithOIDC.SignOpts();
        oidc_options.setOidcProviderUrl(provider_url);

        let oidc_provider = new sign_types.SignWithOIDC();
        oidc_provider.setIdToken(token);
        oidc_provider.setOptions(oidc_options);

        let request_key_provider = new sign_types.SignRequestProvider();
        request_key_provider.setKey(key_provider);

        let request_oidc_provider = new sign_types.SignRequestProvider();
        request_oidc_provider.setOidc(oidc_provider);

        let key_request = new sign_types.SignRequest();
        key_request.setRecordRef(test_records_ref[0]);
        key_request.setProvider(request_key_provider);

        let oidc_request = new sign_types.SignRequest();
        oidc_request.setRecordRef(test_records_ref[1]);
        oidc_request.setProvider(request_oidc_provider);

        try {
            let command_result = client.sign(key_request);
            expect(command_result.error).toBeNull();
            expect(command_result.output).toEqual('Record signed successfully');

            command_result = null;

            command_result = client.sign(oidc_request, client_id);
            expect(command_result.error).toBeNull();
            expect(command_result.output).toEqual('Record signed successfully');

            for (const ref of test_records_ref) {
                let request = new sign_types.VerifyRequest();
                request.setRecordRef(ref);

                response = await client.verify(request);

                expect(response[0]).toBe(true);
            }
        } catch (error) {
            err = error;
            throw new Error(error);
        } finally {
            rmSync("cosign.key", { force: true });
            rmSync("cosign.pub", { force: true });
        }

        expect(err).toBeNull();
    }, 30000); // NOTE: This test timeout is 30s because interactive mode

    test('sync', async () => {
        let err = null;

        try {
            let create_request = new sync_types.CreateSyncRequest();
            create_request.setRemoteDirectoryUrl(process.env["DIRECTORY_SERVER_PEER1_ADDRESS"] || "0.0.0.0:8891");

            let create_response = await client.create_sync(create_request);
            expect(create_response).toBeInstanceOf(sync_types.CreateSyncResponse);

            let sync_id = create_response.u[0];
            expect(isValidUUID(sync_id)).toBe(true);

            let list_request = new sync_types.ListSyncsRequest();
            let list_response = await client.list_syncs(list_request);
            expect(list_response).toBeInstanceOf(sync_types.ListSyncsItem);

            let get_request = new sync_types.GetSyncRequest();
            get_request.setSyncId(sync_id);

            let get_response = await client.get_sync(get_request);
            expect(get_response).toBeInstanceOf(sync_types.GetSyncResponse);
            expect(get_response.u[0]).toEqual(sync_id);

            let delete_request = new sync_types.DeleteSyncRequest();
            delete_request.setSyncId(sync_id);
            client.delete_sync(delete_request);

        } catch (error) {
            err = error;
            throw new Error(error);
        }

        expect(err).toBeNull();
    });
});

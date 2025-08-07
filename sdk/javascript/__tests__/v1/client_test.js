const { setTimeout } = require('node:timers/promises');

const { Client, Config } = require('../../v1/client');
const core_record_pb2 = require('@buf/agntcy_dir.grpc_node/core/v1/record_pb');
const extension_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/extension_pb');
const record_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/record_pb');
const signature_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/signature_pb');
const skill_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/skill_pb');
const locator_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/locator_pb');
const record_query_type = require('@buf/agntcy_dir.grpc_web/routing/v1/record_query_pb');
const routing_types = require('@buf/agntcy_dir.grpc_node/routing/v1/routing_service_pb');
const search_types = require('@buf/agntcy_dir.grpc_node/search/v1/search_service_pb')
const search_query_type = require('@buf/agntcy_dir.grpc_node/search/v1/record_query_pb');
const { afterEach } = require('node:test');

describe('Client', () => {
    const client = new Client(new Config());

    let test_record_ref = null;

    let shouldSkipTest = false;

    beforeEach(async () => {
        if (shouldSkipTest === false) {
            // Wait for directory api server to finish the test action
            await setTimeout(3000);
        }
    });

    afterEach(async () => {
        const test_name = expect.getState().currentTestName.split(" ")[1];
        if (test_name === 'push' && expect.getState().testFailing) {
            shouldSkipTest = true;
        }
    });

    test('push', async () => {
        const exampleRecord = new record_pb2.Record();
        exampleRecord.setName('example-record');
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

        test_record = new core_record_pb2.Record();
        test_record.setV3(exampleRecord);

        let reference;

        try {
            reference = await client.push(test_record);
        } catch (error) {
            throw new Error(error);
        }

        test_record_ref = new core_record_pb2.RecordRef(reference);

        expect(reference).not.toBeNull();
        expect(reference).toBeInstanceOf(Array);
        expect(reference[0].length).toBe(59);
        expect(test_record_ref).not.toBeNull();
        expect(test_record_ref).toBeInstanceOf(core_record_pb2.RecordRef);
    });

    test('pull', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };

        let pulled_record;

        try {
            pulled_record = await client.pull(test_record_ref);
        } catch (error) {
            throw new Error(error);
        }

        const pulledRecordInstance = new core_record_pb2.Record(pulled_record);

        expect(pulled_record).not.toBeNull();
        expect(pulled_record).toBeInstanceOf(Array);
        expect(pulledRecordInstance).not.toBeNull();
        expect(pulledRecordInstance).toBeInstanceOf(core_record_pb2.Record);

    });

    test('search', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };

        const search_query = new search_query_type.RecordQuery();
        search_query.setType(search_query_type.RecordQueryType.RECORD_QUERY_TYPE_SKILL);
        search_query.setValue('/skills/Natural Language Processing/Text Completion');

        const queries = [search_query];

        const search_request = new search_types.SearchRequest();
        search_request.setQueriesList(queries);
        search_request.setLimit(1);

        let objects;

        try {
            objects = await client.search(search_request);
        } catch (error) {
            throw new Error(error);
        }

        objectsInstance = new search_types.SearchResponse(objects);

        expect(objects).not.toBeNull();
        expect(objects).toBeInstanceOf(Array);
        expect(objects.length).not.toBe(0);
        expect(objectsInstance).not.toBeNull();
        expect(objectsInstance).toBeInstanceOf(search_types.SearchResponse);
    });

    test('lookup', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };


        let metadata;

        try {
            metadata = await client.lookup(test_record_ref);
        } catch (error) {
            throw new Error(error);
        }

        expect(metadata).not.toBeNull();
        expect(metadata).toBeInstanceOf(Array);
    });

    test('publish', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };

        const publish_request = new routing_types.PublishRequest();
        publish_request.setRecordCid(test_record_ref.u[0]);

        try {
            await client.publish(publish_request);
        } catch (error) {
            throw new Error(error);
        }

        // no assertion needed, no response
    });

    test('list', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };

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
        expect(objects.length).not.toBe(0);
        expect(objectsInstance).not.toBeNull();
        expect(objectsInstance).toBeInstanceOf(routing_types.ListResponse);
    });

    test('unpublish', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };


        let unpublish_request = new routing_types.UnpublishRequest();
        unpublish_request.setRecordCid(test_record_ref.u[0]);

        try {
            await client.unpublish(unpublish_request);
        } catch (error) {
            throw new Error(error);
        }

        // no assertion needed, no response
    });

    test('delete', async () => {
        if (shouldSkipTest) { throw new Error("Test is skipped") };

        try {
            await client.delete(test_record_ref);
        } catch (error) {
            throw new Error(error);
        }

        // no assertion needed, no response
    });
});

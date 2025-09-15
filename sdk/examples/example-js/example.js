// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

const core_record_pb2 = require('@buf/agntcy_dir.grpc_node/core/v1/record_pb');
const extension_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/extension_pb');
const record_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/record_pb');
const signature_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/signature_pb');
const skill_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/skill_pb');
const locator_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/locator_pb');
const record_query_type = require('@buf/agntcy_dir.grpc_web/routing/v1/record_query_pb');
const routing_types = require('@buf/agntcy_dir.grpc_node/routing/v1/routing_service_pb');
const search_types = require('@buf/agntcy_dir.grpc_node/search/v1/search_service_pb')
const search_query_type = require('@buf/agntcy_dir.grpc_node/search/v1/record_query_pb')
const { Client, Config } = require('agntcy-dir-sdk/v1/client');

function printAsJson(obj) {
    let obj_dict = protoToJson(obj);
    obj_dict = convertUint64FieldsToInt(obj_dict);
    const obj_json = JSON.stringify(obj_dict);

    return obj_json;
}


function convertUint64FieldsToInt(obj) {
    if (Array.isArray(obj)) {
        obj.forEach(convertUint64FieldsToInt);
    } else if (obj && typeof obj === 'object') {
        for (const [key, value] of Object.entries(obj)) {
            if (
                (key === 'category_uid' || key === 'class_uid') &&
                typeof value === 'string' &&
                /^\d+$/.test(value)
            ) {
                obj[key] = Number(value);
            } else if (Array.isArray(value)) {
                value.forEach(convertUint64FieldsToInt);
            } else if (typeof value === 'object' && value !== null) {
                convertUint64FieldsToInt(value);
            }
        }
    }
    return obj;
}

function protoToJson(protoMsg) {
    return protoMsg.toObject ? protoMsg.toObject() : protoMsg;
}

function generateRecords(names) {
    const test_records = [];

    names.forEach(name => {
        const exampleRecord = new record_pb2.Record();
        exampleRecord.setName(name);
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

        test_records.push(test_record);
    });

    return test_records;
}

(async () => {
    const client = new Client(new Config());

    // Create records object
    const records = generateRecords(['example-record', 'example-record2']);

    // Push objects to store
    let refs;

    try {
        response = await client.push(records);
        refs = response;

        refs.forEach(ref => {
            console.log('Pushed object ref:', printAsJson(ref));
        });
    } catch (err) {
        console.error('Push error:', err);
        return;
    }

    // Pull objects from the store
    let pulledRecords;
    try {
        response = await client.pull(refs);
        pulledRecords = response;

        pulledRecords.forEach(pulledRecord => {
            console.log('Pulled object:', printAsJson(new core_record_pb2.Record(pulledRecord)));
        });
    } catch (err) {
        console.error('Pull error:', err);
        return;
    }

    // Search object
    let search_query = new search_query_type.RecordQuery();
    search_query.setType(search_query_type.RecordQueryType.RECORD_QUERY_TYPE_SKILL);
    search_query.setValue('/skills/Natural Language Processing/Text Completion');

    const queries = [search_query];

    let search_request = new search_types.SearchRequest();
    search_request.setQueriesList(queries);
    search_request.setLimit(3);

    try {
        search_response = await client.search(search_request);
        console.log('Search result:', printAsJson(search_response));
    } catch (err) {
        console.error('Search error:', err);
        return;
    }

    // Lookup the object (gets metadata)
    let metadatas;
    try {
        metadatas = await client.lookup(refs);
        metadatas.forEach(metadata => {
            console.log('Lookup result:', printAsJson(new core_record_pb2.RecordMeta(metadata)));
        });
    } catch (err) {
        console.error('Lookup error:', err);
        return;
    }

    let publish_request = new routing_types.PublishRequest();
    publish_request.setRecordCid(refs[0].u[0]);

    // Publish the object
    try {
        client.publish(publish_request);
        console.log('Object published.');
    } catch (err) {
        console.error('Publish error:', err);
    }

    // List objects in the store
    const query = new record_query_type.RecordQuery();
    query.setType(record_query_type.RECORD_QUERY_TYPE_SKILL);
    query.setValue('/skills/Natural Language Processing/Text Completion');
    const listRequest = new routing_types.ListRequest();
    listRequest.addQueries(query);

    try {
        list_response = await client.list(listRequest);
        list_response.forEach(r => {
            console.log('Listed objects:', printAsJson(new routing_types.ListResponse(r)));
        });
    } catch (err) {
        console.error('List error:', err);
    }

    let unpublish_request = new routing_types.UnpublishRequest();
    unpublish_request.setRecordCid(refs[0].u[0]);

    // Unpublish the object
    try {
        client.unpublish(unpublish_request);

        console.log('Object unpublished.');
    } catch (err) {
        console.error('Unpublish error:', err);
    }

    // Delete the object
    try {
        await client.delete(refs);
        console.log('Object deleted.');
    } catch (err) {
        console.error('Delete error:', err);
    }
})();

const core_record_pb2 = require('@buf/agntcy_dir.grpc_node/core/v1/record_pb');
const extension_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/extension_pb');
const record_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/record_pb');
const signature_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/signature_pb');
const skill_pb2 = require('@buf/agntcy_oasf.grpc_web/objects/v3/skill_pb');
const record_query_type = require('@buf/agntcy_dir.grpc_web/routing/v1alpha2/record_query_pb');
const routing_types = require('@buf/agntcy_dir.grpc_node/routing/v1alpha2/routing_service_pb');
const search_types = require('@buf/agntcy_dir.grpc_node/search/v1alpha2/search_service_pb')
const search_query_type = require('@buf/agntcy_dir.grpc_node/search/v1alpha2/record_query_pb')


const { Client, Config } = require('./client');

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

(async () => {
    const client = new Client(new Config());

    // Create a record object
    const exampleRecord = new record_pb2.Record();
    exampleRecord.setName('example-record');
    exampleRecord.setVersion('v3');

    const skill = new skill_pb2.Skill();
    skill.setName('Natural Language Processing');
    skill.setId(1);
    exampleRecord.addSkills(skill);

    const extension = new extension_pb2.Extension();
    extension.setName('schema.oasf.agntcy.org/domains/domain-1');
    extension.setVersion('v1');
    exampleRecord.addExtensions(extension);

    const signature = new signature_pb2.Signature();
    exampleRecord.setSignature(signature);

    record = new core_record_pb2.Record();
    record.setV3(exampleRecord);

    // Push object to store
    let ref;

    try {
        response = await client.push(record)
        ref = new core_record_pb2.RecordRef(response);

        console.log('Pushed object ref:', printAsJson(ref));
    } catch (err) {
        console.error('Push error:', err);
        return;
    }

    // Pull the object from the store
    let pulledRecord;
    try {
        response = await client.pull(ref);
        pulledRecord = new core_record_pb2.Record(response);

        console.log('Pulled object:', printAsJson(pulledRecord));
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
    search_request.setLimit(1);

    let search_response;
    try {
        response = await client.search(search_request);
        search_response = new search_types.SearchResponse(response);
        console.log('Search result:', printAsJson(search_response));
    } catch (err) {
        console.error('Search error:', err);
        return;
    }

    // Lookup the object (gets metadata)
    let metadata;
    try {
        metadata = await client.lookup(ref);
    } catch (err) {
        console.error('Lookup error:', err);
        return;
    }
    let metadata_dict = protoToJson(metadata);
    metadata_dict = convertUint64FieldsToInt(metadata_dict);
    const metadata_json = JSON.stringify(metadata_dict);

    console.log('Object metadata:', metadata_json);

    let publish_request = new routing_types.PublishRequest();
    publish_request.setRecordCid(ref.u[0]);

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

    let list_response;
    try {
        response = await client.list(listRequest)
        list_response = new routing_types.ListResponse(response);

        console.log('Listed objects:', printAsJson(list_response));
    } catch (err) {
        console.error('List error:', err);
    }

    let unpublish_request = new routing_types.UnpublishRequest();
    unpublish_request.setRecordCid(ref.u[0]);

    // Unpublish the object
    try {
        client.unpublish(unpublish_request);

        console.log('Object unpublished.');
    } catch (err) {
        console.error('Unpublish error:', err);
    }

    // Delete the object
    try {
        await client.delete(ref);

        console.log('Object deleted.');
    } catch (err) {
        console.error('Delete error:', err);
    }
})();

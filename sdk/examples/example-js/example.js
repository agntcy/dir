// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

const { Client, Config, models } = await import('agntcy-dir');

function generateRecords(names) {
    return names.map(name => {
        name=`${name}`,
        version="v1.0.0",
        schema_version="v0.7.0",
        description="My example agent",
        authors=["AGNTCY"],
        created_at="2025-03-19T17:06:37Z",
        skills=[
            {
                name: "natural_language_processing/natural_language_generation/text_completion",
                id: 10201
            },
            {
                name: "natural_language_processing/analytical_reasoning/problem_solving",
                id: 10702
            }
        ],
        locators=[
            {
                type: "docker-image",
                url: "https://ghcr.io/agntcy/marketing-strategy"
            }
        ],
        domains=[
            {
                name: "technology/networking",
                id: 103
            }
        ],
        modules=[
            {
                name: "runtime/a2a",
                data: {}
            }
        ]
    });
}

(async () => {
    // Create client
    const client = new Client(new Config());

    // Create record objects
    const records = generateRecords(['example-record', 'example-record2']);

    // Push objects
    const pushed_refs = await client.push(records);
    pushed_refs.forEach(ref => {
        console.log('Pushed object ref:', ref);
    });

    // Pull objects
    const pulled_records = await client.pull(pushed_refs);
    pulled_records.forEach(pulled_record => {
        console.log('Pulled object:', pulled_record);
    });

    // Lookup objects
    const metadatas = await client.lookup(pushed_refs);
    metadatas.forEach(metadata => {
        console.log('Lookup result:', printAsJson(new core_record_pb2.RecordMeta(metadata)));
    });

    // Search objects
    const search_response = await client.search({
        queries: [{
            type: models.search_v1.RecordQueryType.SKILL_ID,
            value: "10201"
        }],
        limit: 3
    });
    console.log('Search result:', search_response);

    // Publish objects
    await client.publish({
        request: {
            case: "recordRefs",
            value: {
                refs: pushed_refs,
            }
        }
    });
    console.log('Objects published.');

    // List objects in the routing table
    const list_response = await client.list({
        queries: [
            {
                type: models.routing_v1.RecordQueryType.SKILL,
                value: 'natural_language_processing/analytical_reasoning/problem_solving'
            }
        ],
    });
    list_response.forEach(r => {
        console.log('Listed objects:', printAsJson(new routing_types.ListResponse(r)));
    });

    // Unpublish objects
    client.unpublish({
        request: {
            case: "recordRefs",
            value: {
                refs: pushed_refs,
            }
        }
    });
    console.log('Objects unpublished.');

    // Delete objects
    await client.delete(pushed_refs);
    console.log('Objects deleted.');
})();

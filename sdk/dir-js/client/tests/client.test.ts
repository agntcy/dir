// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { execSync } from 'node:child_process';
import { readFileSync, rmSync } from 'node:fs';
import process from 'node:process';

import { validate as isValidUUID } from 'uuid';
import { v4 as uuidv4 } from 'uuid';

// Import the compiled JavaScript modules for compatibility
const { models } = require('../../dist/models');
const { Client, Config } = require('../../dist/client');

/**
 * Generate test records with unique names.
 * Schema: https://schema.oasf.outshift.com/0.7.0/objects/record
 * @param count - Number of records to generate
 * @param testFunctionName - Name of the test function for record naming
 * @returns Array of generated Record objects
 */
function genRecords(count: number, testFunctionName: string): any[] {
    const records: any[] = [];
    
    for (let index = 0; index < count; index++) {
        // Generate unique record data
        const recordData = {
            name: `agntcy-${testFunctionName}-${index}-${uuidv4().substring(0, 8)}`,
            version: "v3.0.0",
            schema_version: "v0.7.0",
            description: "Research agent for Cisco's marketing strategy.",
            authors: ["Cisco Systems"],
            created_at: "2025-03-19T17:06:37Z",
            skills: [
                {
                    name: "natural_language_processing/natural_language_generation/text_completion",
                    id: 10201
                },
                {
                    name: "natural_language_processing/analytical_reasoning/problem_solving",
                    id: 10702
                }
            ],
            locators: [
                {
                    type: "docker-image",
                    url: "https://ghcr.io/agntcy/marketing-strategy"
                }
            ],
            domains: [
                {
                    name: "technology/networking",
                    id: 103
                }
            ],
            modules: []
        };

        // Create the record object
        var record = {} as any;
        record.data = recordData;

        // Append to records array
        records.push(record);
    }

    return records;
}

describe('Client', () => {
    let client: any;

    beforeAll(() => {
        // Verify that DIRCTL_PATH is set in the environment
        expect(process.env.DIRCTL_PATH).toBeDefined();
        
        // Initialize the client
        client = new Client(Config.loadFromEnv());
    });

    afterAll(async () => {
        // Clean up any resources if needed
        // Note: gRPC clients in Connect don't need explicit closing
    });

    test('push', async () => {
        const records = genRecords(2, "push");
        const recordRefs = await client.push(records);

        expect(recordRefs).not.toBeNull();
        expect(recordRefs).toBeInstanceOf(Array);
        expect(recordRefs).toHaveLength(2);

        for (const ref of recordRefs) {
            expect(ref).toBeInstanceOf(models.core_v1.RecordRef);
            expect(ref.cid).toHaveLength(59);
        }
    });

    test('pull', async () => {
        const records = genRecords(2, "pull");
        const recordRefs = await client.push(records);
        const pulledRecords = await client.pull(recordRefs);

        expect(pulledRecords).not.toBeNull();
        expect(pulledRecords).toBeInstanceOf(Array);
        expect(pulledRecords).toHaveLength(2);

        for (let index = 0; index < pulledRecords.length; index++) {
            const record = pulledRecords[index];
            expect(record).toBeInstanceOf(models.core_v1.Record);
            expect(record).toEqual(records[index]);
        }
    });

    test('search', async () => {
        const records = genRecords(1, "search");
        await client.push(records);

        const searchQuery = new models.search_v1.RecordQuery({
            type: models.search_v1.RECORD_QUERY_TYPE_SKILL_ID,
            value: '10201'
        });

        const searchRequest = new models.search_v1.SearchRequest({
            queries: [searchQuery],
            limit: 2
        });

        const objects = await client.search(searchRequest);

        expect(objects).not.toBeNull();
        expect(objects).toBeInstanceOf(Array);
        expect(objects.length).toBeGreaterThan(0);

        for (const obj of objects) {
            expect(obj).toBeInstanceOf(models.search_v1.SearchResponse);
        }
    });

    test('lookup', async () => {
        const records = genRecords(2, "lookup");
        const recordRefs = await client.push(records);
        const metadatas = await client.lookup(recordRefs);

        expect(metadatas).not.toBeNull();
        expect(metadatas).toBeInstanceOf(Array);
        expect(metadatas).toHaveLength(2);

        for (const metadata of metadatas) {
            expect(metadata).toBeInstanceOf(models.core_v1.RecordMeta);
        }
    });

    test('publish', async () => {
        const records = genRecords(1, "publish");
        const recordRefs = await client.push(records);
        
        const publishRequest = new models.routing_v1.PublishRequest({
            recordRefs: new models.routing_v1.RecordRefs({ refs: recordRefs })
        });

        try {
            await client.publish(publishRequest);
        } catch (error) {
            fail(`Publish should not throw error: ${error}`);
        }
    });

    test('list', async () => {
        const records = genRecords(1, "list");
        const recordRefs = await client.push(records);
        await client.publish(new models.routing_v1.PublishRequest({
            recordRefs: new models.routing_v1.RecordRefs({ refs: recordRefs })
        }));

        // Sleep to allow the publication to be indexed
        await new Promise(resolve => setTimeout(resolve, 5000));

        // Query for records in the domain
        const query = new models.routing_v1.RecordQuery({
            type: models.routing_v1.RECORD_QUERY_TYPE_DOMAIN,
            value: 'technology/networking'
        });

        const listRequest = new models.routing_v1.ListRequest({
            queries: [query]
        });

        const objects = await client.list(listRequest);

        expect(objects).not.toBeNull();
        expect(objects).toBeInstanceOf(Array);
        expect(objects.length).not.toBe(0);

        for (const obj of objects) {
            expect(obj).toBeInstanceOf(models.routing_v1.ListResponse);
        }
    });

    test('unpublish', async () => {
        const records = genRecords(1, "unpublish");
        const recordRefs = await client.push(records);

        const publishRecordRefs = new models.routing_v1.RecordRefs({ refs: recordRefs });
        const unpublishRequest = new models.routing_v1.UnpublishRequest({ 
            recordRefs: publishRecordRefs 
        });

        try {
            await client.unpublish(unpublishRequest);
        } catch (error) {
            fail(`Unpublish should not throw error: ${error}`);
        }
    });

    test('delete', async () => {
        const records = genRecords(1, "delete");
        const recordRefs = await client.push(records);

        try {
            await client.delete(recordRefs);
        } catch (error) {
            fail(`Delete should not throw error: ${error}`);
        }
    });

    test('pushReferrer', async () => {
        const records = genRecords(2, "pushReferrer");
        const recordRefs = await client.push(records);

        const exampleSignature = new models.sign_v1.Signature();
        const requests = recordRefs.map((recordRef: any) => 
            new models.store_v1.PushReferrerRequest({
                recordRef: recordRef,
                signature: exampleSignature
            })
        );

        try {
            const response = await client.push_referrer(requests);
            expect(response).not.toBeNull();
            expect(response).toHaveLength(2);
            
            for (const r of response) {
                expect(r).toBeInstanceOf(models.store_v1.PushReferrerResponse);
            }
        } catch (error) {
            fail(`Push referrer should not throw error: ${error}`);
        }
    });

    test('pullReferrer', async () => {
        const records = genRecords(2, "pullReferrer");
        const recordRefs = await client.push(records);

        const requests = recordRefs.map((recordRef: any) => 
            new models.store_v1.PullReferrerRequest({
                recordRef: recordRef,
                pullSignature: false
            })
        );

        try {
            const response = await client.pull_referrer(requests);
            expect(response).not.toBeNull();
            expect(response).toHaveLength(2);
            
            for (const r of response) {
                expect(r).toBeInstanceOf(models.store_v1.PullReferrerResponse);
            }
        } catch (error) {
            // Remove when service is implemented
            if (error instanceof Error && error.message.includes("pull referrer not implemented")) {
                return;
            }
            fail(`Pull referrer should not throw error: ${error}`);
        }
    });

    test('sign_and_verify', async () => {
        const records = genRecords(2, "sign_verify");
        const recordRefs = await client.push(records);

        const shellEnv = { ...process.env };
        const keyPassword = "testing-key";

        // Clean up any existing keys
        rmSync("cosign.key", { force: true });
        rmSync("cosign.pub", { force: true });

        try {
            // Generate key pair
            const cosignPath = process.env["COSIGN_PATH"] || 'cosign';
            execSync(
                `${cosignPath} generate-key-pair`,
                { 
                    env: { ...shellEnv, COSIGN_PASSWORD: keyPassword }, 
                    encoding: 'utf8', 
                    stdio: 'pipe' 
                }
            );

            // Read the private key
            const keyFile = readFileSync('cosign.key');

            // Create signing providers
            const keyProvider = new models.sign_v1.SignWithKey({
                privateKey: keyFile,
                password: Buffer.from(keyPassword, 'utf-8')
            });

            const token = shellEnv["OIDC_TOKEN"] || "";
            const providerUrl = shellEnv["OIDC_PROVIDER_URL"] || "";
            const clientId = shellEnv["OIDC_CLIENT_ID"] || "sigstore";

            const oidcOptions = new models.sign_v1.SignWithOIDC.SignOpts({
                oidcProviderUrl: providerUrl
            });

            const oidcProvider = new models.sign_v1.SignWithOIDC({
                idToken: token,
                options: oidcOptions
            });

            const requestKeyProvider = new models.sign_v1.SignRequestProvider({
                key: keyProvider
            });

            const requestOidcProvider = new models.sign_v1.SignRequestProvider({
                oidc: oidcProvider
            });

            const keyRequest = new models.sign_v1.SignRequest({
                recordRef: recordRefs[0],
                provider: requestKeyProvider
            });

            const oidcRequest = new models.sign_v1.SignRequest({
                recordRef: recordRefs[1],
                provider: requestOidcProvider
            });

            // Sign test
            const keyCommandResult = client.sign(keyRequest);
            expect(keyCommandResult.signature).toBeDefined();

            const oidcCommandResult = client.sign(oidcRequest, clientId);
            expect(oidcCommandResult.signature).toBeDefined();

            // Verify test
            for (const ref of recordRefs) {
                const request = new models.sign_v1.VerifyRequest({
                    recordRef: ref
                });

                const response = await client.verify(request);
                expect(response.success).toBe(true);
            }

            // Test invalid CID
            const invalidRequest = new models.sign_v1.SignRequest({
                recordRef: new models.core_v1.RecordRef({ cid: "invalid-cid" }),
                provider: requestKeyProvider
            });

            try {
                client.sign(invalidRequest);
                fail("Should have thrown error for invalid CID");
            } catch (error) {
                if (error instanceof Error) {
                    expect(error.message).toContain("Failed to sign the object");
                }
            }

        } catch (error) {
            fail(`Sign and verify test failed: ${error}`);
        } finally {
            // Clean up keys
            rmSync("cosign.key", { force: true });
            rmSync("cosign.pub", { force: true });
        }
    }, 30000);

    test('sync', async () => {
        try {
            const createRequest = new models.store_v1.CreateSyncRequest({
                remoteDirectoryUrl: process.env["DIRECTORY_SERVER_PEER1_ADDRESS"] || "0.0.0.0:8891"
            });

            const createResponse = await client.create_sync(createRequest);
            expect(createResponse).toBeInstanceOf(models.store_v1.CreateSyncResponse);

            const syncId = createResponse.syncId;
            expect(isValidUUID(syncId)).toBe(true);

            const listRequest = new models.store_v1.ListSyncsRequest();
            const listResponse = await client.list_syncs(listRequest);
            expect(listResponse).toBeInstanceOf(Array);

            for (const syncItem of listResponse) {
                expect(syncItem).toBeInstanceOf(models.store_v1.ListSyncsItem);
                expect(isValidUUID(syncItem.syncId)).toBe(true);
            }

            const getRequest = new models.store_v1.GetSyncRequest({
                syncId: syncId
            });

            const getResponse = await client.get_sync(getRequest);
            expect(getResponse).toBeInstanceOf(models.store_v1.GetSyncResponse);
            expect(getResponse.syncId).toEqual(syncId);

            const deleteRequest = new models.store_v1.DeleteSyncRequest({
                syncId: syncId
            });
            
            await client.delete_sync(deleteRequest);

        } catch (error) {
            fail(`Sync test should not throw error: ${error}`);
        }
    });
});
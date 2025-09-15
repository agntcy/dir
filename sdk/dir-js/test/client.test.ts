// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import {describe, test, beforeAll, afterAll, expect} from 'vitest';
import {execSync} from 'node:child_process';
import {readFileSync, rmSync} from 'node:fs';
import process from 'node:process';

import {validate as isValidUUID} from 'uuid';
import {v4 as uuidv4} from 'uuid';
import {Struct} from '@buf/agntcy_dir.community_timostamm-protobuf-ts/google/protobuf/struct_pb';

import {Client, Config, models} from '../dist/src';

/**
 * Generate test records with unique names.
 * Schema: https://schema.oasf.outshift.com/0.7.0/objects/record
 * @param count - Number of records to generate
 * @param testFunctionName - Name of the test function for record naming
 * @returns Array of generated Record objects
 */
function genRecords(
  count: number,
  testFunctionName: string,
): models.core_v1.Record[] {
  const records: models.core_v1.Record[] = [];
  for (let index = 0; index < count; index++) {
    records.push({
      data: Struct.fromJson({
        name: `agntcy-${testFunctionName}-${index}-${uuidv4().substring(0, 8)}`,
        version: 'v3.0.0',
        schema_version: 'v0.7.0',
        description: "Research agent for Cisco's marketing strategy.",
        authors: ['Cisco Systems'],
        created_at: '2025-03-19T17:06:37Z',
        skills: [
          {
            name: 'natural_language_processing/natural_language_generation/text_completion',
            id: 10201,
          },
          {
            name: 'natural_language_processing/analytical_reasoning/problem_solving',
            id: 10702,
          },
        ],
        locators: [
          {
            type: 'docker-image',
            url: 'https://ghcr.io/agntcy/marketing-strategy',
          },
        ],
        domains: [
          {
            name: 'technology/networking',
            id: 103,
          },
        ],
        modules: [],
      }),
    });
  }

  return records;
}

describe('Client', () => {
  let client: Client;

  beforeAll(async () => {
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
    const records = genRecords(2, 'push');
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
    const records = genRecords(2, 'pull');
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
    const records = genRecords(1, 'search');
    await client.push(records);

    const searchQuery: models.search_v1.RecordQuery = {
      type: models.search_v1.RecordQueryType.SKILL_ID,
      value: '10201',
    };

    const searchRequest: models.search_v1.SearchRequest = {
      queries: [searchQuery],
      limit: 2,
    };

    const objects = await client.search(searchRequest);

    expect(objects).not.toBeNull();
    expect(objects).toBeInstanceOf(Array);
    expect(objects.length).toBeGreaterThan(0);

    for (const obj of objects) {
      expect(obj).toBeInstanceOf(models.search_v1.SearchResponse);
    }
  });

  test('lookup', async () => {
    const records = genRecords(2, 'lookup');
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
    const records = genRecords(1, 'publish');
    const recordRefs = await client.push(records);

    await client.publish({
      request: {
        oneofKind: 'recordRefs',
        recordRefs: {
          refs: recordRefs,
        },
      },
    });
  });

  test('list', async () => {
    const records = genRecords(1, 'list');
    const recordRefs = await client.push(records);

    // Publish records
    await client.publish({
      request: {
        oneofKind: 'recordRefs',
        recordRefs: {
          refs: recordRefs,
        },
      },
    });

    // Sleep to allow the publication to be indexed
    await new Promise(resolve => setTimeout(resolve, 5000));

    // Query for records in the domain
    const query: models.routing_v1.RecordQuery = {
      type: models.routing_v1.RecordQueryType.DOMAIN,
      value: 'technology/networking',
    };
    const objects = await client.list({
      queries: [query],
    });

    expect(objects).not.toBeNull();
    expect(objects).toBeInstanceOf(Array);
    expect(objects.length).not.toBe(0);

    for (const obj of objects) {
      expect(obj).toBeInstanceOf(models.routing_v1.ListResponse);
    }
  });

  test('unpublish', async () => {
    const records = genRecords(1, 'unpublish');
    const recordRefs = await client.push(records);

    // Publish records
    await client.publish({
      request: {
        oneofKind: 'recordRefs',
        recordRefs: {
          refs: recordRefs,
        },
      },
    });

    // Unpublish
    await client.unpublish({
      request: {
        oneofKind: 'recordRefs',
        recordRefs: {
          refs: recordRefs,
        },
      },
    });
  });

  test('delete', async () => {
    const records = genRecords(1, 'delete');
    const recordRefs = await client.push(records);

    await client.delete(recordRefs);
  });

  test('pushReferrer', async () => {
    const records = genRecords(2, 'pushReferrer');
    const recordRefs = await client.push(records);

    const requests: models.store_v1.PushReferrerRequest[] = recordRefs.map(
      (
        recordRef: models.core_v1.RecordRef,
      ): models.store_v1.PushReferrerRequest => {
        return {
          recordRef: recordRef,
          options: {
            oneofKind: 'signature',
            signature: {} as models.sign_v1.Signature,
          },
        };
      },
    );

    const response = await client.push_referrer(requests);
    expect(response).not.toBeNull();
    expect(response).toHaveLength(2);

    for (const r of response) {
      expect(r).toBeInstanceOf(models.store_v1.PushReferrerResponse);
    }
  });

  test('pullReferrer', async () => {
    const records = genRecords(2, 'pullReferrer');
    const recordRefs = await client.push(records);

    const requests: models.store_v1.PullReferrerRequest[] = recordRefs.map(
      (
        recordRef: models.core_v1.RecordRef,
      ): models.store_v1.PullReferrerRequest => {
        return {
          recordRef: recordRef,
          options: {
            oneofKind: 'pullSignature',
            pullSignature: true,
          },
        };
      },
    );

    const response = await client.pull_referrer(requests);
    expect(response).not.toBeNull();
    expect(response).toHaveLength(2);

    for (const r of response) {
      expect(r).toBeInstanceOf(models.store_v1.PullReferrerResponse);
    }
  });

  test('sign_and_verify', async () => {
    const records = genRecords(2, 'sign_verify');
    const recordRefs = await client.push(records);

    const shellEnv = {...process.env};
    const keyPassword = 'testing-key';

    // Clean up any existing keys
    rmSync('cosign.key', {force: true});
    rmSync('cosign.pub', {force: true});

    try {
      // Generate key pair
      const cosignPath = process.env['COSIGN_PATH'] || 'cosign';
      execSync(`${cosignPath} generate-key-pair`, {
        env: {...shellEnv, COSIGN_PASSWORD: keyPassword},
        encoding: 'utf8',
        stdio: 'pipe',
      });

      // Read configuration data
      const keyFile = readFileSync('cosign.key');
      const token = shellEnv['OIDC_TOKEN'] || '';
      const providerUrl = shellEnv['OIDC_PROVIDER_URL'] || '';
      const clientId = shellEnv['OIDC_CLIENT_ID'] || 'sigstore';

      // Create signing providers
      const keyRequest: models.sign_v1.SignRequest = {
        recordRef: recordRefs[0],
        provider: {
          request: {
            oneofKind: 'key',
            key: {
              privateKey: keyFile,
              password: Buffer.from(keyPassword, 'utf-8'),
            },
          },
        },
      };

      const oidcRequest: models.sign_v1.SignRequest = {
        recordRef: recordRefs[1],
        provider: {
          request: {
            oneofKind: 'oidc',
            oidc: {
              idToken: token,
              options: {
                oidcProviderUrl: providerUrl,
              },
            },
          },
        },
      };

      // Sign test
      const keyCommandResult = client.sign(keyRequest);
      expect(keyCommandResult.signature).toBeDefined();

      const oidcCommandResult = client.sign(oidcRequest, clientId);
      expect(oidcCommandResult.signature).toBeDefined();

      // Verify test
      for (const ref of recordRefs) {
        const response = await client.verify({
          recordRef: ref,
        });
        expect(response.success).toBe(true);
      }

      // Test invalid CID
      const invalidRequest: models.sign_v1.SignRequest = {
        recordRef: {cid: 'invalid-cid'},
        provider: {
          request: {
            oneofKind: 'key',
            key: {
              privateKey: Uint8Array.from([]),
              password: Uint8Array.from([]),
            },
          },
        },
      };

      try {
        client.sign(invalidRequest);
        expect.fail('Should have thrown error for invalid CID');
      } catch (error) {
        if (error instanceof Error) {
          expect(error.message).toContain('Failed to sign the object');
        }
      }
    } catch (error) {
      expect.fail(`Sign and verify test failed: ${error}`);
    } finally {
      // Clean up keys
      rmSync('cosign.key', {force: true});
      rmSync('cosign.pub', {force: true});
    }
  }, 30000);

  test('sync', async () => {
    // Create sync
    const createResponse = await client.create_sync({
      remoteDirectoryUrl:
        process.env['DIRECTORY_SERVER_PEER1_ADDRESS'] || '0.0.0.0:8891',
    });
    expect(createResponse).toBeInstanceOf(models.store_v1.CreateSyncResponse);

    const syncId = createResponse.syncId;
    expect(isValidUUID(syncId)).toBe(true);

    // List syncs
    const listRequest: models.store_v1.ListSyncsRequest = {};
    const listResponse = await client.list_syncs(listRequest);
    expect(listResponse).toBeInstanceOf(Array);

    for (const syncItem of listResponse) {
      expect(syncItem).toBeInstanceOf(models.store_v1.ListSyncsItem);
      expect(isValidUUID(syncItem.syncId)).toBe(true);
    }

    // Get sync
    const getResponse = await client.get_sync({
      syncId: syncId,
    });
    expect(getResponse).toBeInstanceOf(models.store_v1.GetSyncResponse);
    expect(getResponse.syncId).toEqual(syncId);

    // Delete sync
    await client.delete_sync({
      syncId: syncId,
    });
  });
});

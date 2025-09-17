// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import {describe, test, beforeAll, afterAll, expect} from 'vitest';

import {execSync} from 'node:child_process';
import {readFileSync, rmSync} from 'node:fs';
import {env} from 'node:process';
import {create} from '@bufbuild/protobuf';

import {validate as isValidUUID} from 'uuid';
import {v4 as uuidv4} from 'uuid';

import {Client, Config, models} from '../src';

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
    records.push(
      create(models.core_v1.RecordSchema, {
        data: {
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
        },
      }),
    );
  }

  return records;
}

describe('Client', () => {
  let client: Client;

  beforeAll(async () => {
    // Verify that DIRCTL_PATH is set in the environment
    expect(env.DIRCTL_PATH).toBeDefined();

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
      expect(ref).toBeTypeOf(typeof models.core_v1.RecordRefSchema);
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
      expect(record).toBeTypeOf(typeof models.core_v1.RecordSchema);
      expect(record).toEqual(records[index]);
    }
  });

  test('search', async () => {
    const records = genRecords(1, 'search');
    await client.push(records);

    const searchRequest = create(models.search_v1.SearchRequestSchema, {
      queries: [
        {
          type: models.search_v1.RecordQueryType.SKILL_ID,
          value: '10201',
        },
      ],
      limit: 2,
    });

    const objects = await client.search(searchRequest);

    expect(objects).not.toBeNull();
    expect(objects).toBeInstanceOf(Array);
    expect(objects.length).toBeGreaterThan(0);

    for (const obj of objects) {
      expect(obj).toBeTypeOf(typeof models.search_v1.SearchResponseSchema);
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
      expect(metadata).toBeTypeOf(typeof models.core_v1.RecordMetaSchema);
    }
  });

  test('publish', async () => {
    const records = genRecords(1, 'publish');
    const recordRefs = await client.push(records);

    await client.publish(
      create(models.routing_v1.PublishRequestSchema, {
        request: {
          case: 'recordRefs',
          value: {
            refs: recordRefs,
          },
        },
      }),
    );
  });

  test('list', async () => {
    const records = genRecords(1, 'list');
    const recordRefs = await client.push(records);

    // Publish records
    await client.publish(
      create(models.routing_v1.PublishRequestSchema, {
        request: {
          case: 'recordRefs',
          value: {
            refs: recordRefs,
          },
        },
      }),
    );

    // Sleep to allow the publication to be indexed
    await new Promise(resolve => setTimeout(resolve, 5000));

    // Query for records in the domain
    const objects = await client.list(
      create(models.routing_v1.ListRequestSchema, {
        queries: [
          {
            type: models.routing_v1.RecordQueryType.DOMAIN,
            value: 'technology/networking',
          },
        ],
      }),
    );

    expect(objects).not.toBeNull();
    expect(objects).toBeInstanceOf(Array);
    expect(objects.length).not.toBe(0);

    for (const obj of objects) {
      expect(obj).toBeTypeOf(typeof models.routing_v1.ListResponseSchema);
    }
  }, 30000);

  test('unpublish', async () => {
    const records = genRecords(1, 'unpublish');
    const recordRefs = await client.push(records);

    // Publish records
    await client.publish(
      create(models.routing_v1.PublishRequestSchema, {
        request: {
          case: 'recordRefs',
          value: {
            refs: recordRefs,
          },
        },
      }),
    );

    // Unpublish
    await client.unpublish(
      create(models.routing_v1.UnpublishRequestSchema, {
        request: {
          case: 'recordRefs',
          value: {
            refs: recordRefs,
          },
        },
      }),
    );
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
        return create(models.store_v1.PushReferrerRequestSchema, {
          recordRef: recordRef,
          options: {
            case: 'signature',
            value: {},
          },
        });
      },
    );

    const response = await client.push_referrer(requests);
    expect(response).not.toBeNull();
    expect(response).toHaveLength(2);

    for (const r of response) {
      expect(r).toBeTypeOf(typeof models.store_v1.PushReferrerResponseSchema);
    }
  });

  test('pullReferrer', async () => {
    const records = genRecords(2, 'pullReferrer');
    const recordRefs = await client.push(records);

    const requests: models.store_v1.PullReferrerRequest[] = recordRefs.map(
      (
        recordRef: models.core_v1.RecordRef,
      ): models.store_v1.PullReferrerRequest => {
        return create(models.store_v1.PullReferrerRequestSchema, {
          recordRef: recordRef,
          options: {
            case: 'pullSignature',
            value: true,
          },
        });
      },
    );

    const response = await client.pull_referrer(requests);
    expect(response).not.toBeNull();
    expect(response).toHaveLength(2);

    for (const r of response) {
      expect(r).toBeTypeOf(typeof models.store_v1.PullReferrerResponseSchema);
    }
  });

  test('sign_and_verify', async () => {
    const records = genRecords(2, 'sign_verify');
    const recordRefs = await client.push(records);

    const shellEnv = {...env};
    const keyPassword = 'testing-key';

    // Clean up any existing keys
    rmSync('cosign.key', {force: true});
    rmSync('cosign.pub', {force: true});

    try {
      // Generate key pair
      const cosignPath = env['COSIGN_PATH'] || 'cosign';
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
      const keyRequest = create(models.sign_v1.SignRequestSchema, {
        recordRef: recordRefs[0],
        provider: {
          request: {
            case: 'key',
            value: {
              privateKey: keyFile,
              password: Buffer.from(keyPassword, 'utf-8'),
            },
          },
        },
      });

      const oidcRequest = create(models.sign_v1.SignRequestSchema, {
        recordRef: recordRefs[1],
        provider: {
          request: {
            case: 'oidc',
            value: {
              idToken: token,
              options: {
                oidcProviderUrl: providerUrl,
              },
            },
          },
        },
      });

      // Sign test
      client.sign(keyRequest);
      client.sign(oidcRequest, clientId);

      // Verify test
      for (const ref of recordRefs) {
        const response = await client.verify(
          create(models.sign_v1.VerifyRequestSchema, {
            recordRef: ref,
          }),
        );
        expect(response.success).toBe(true);
      }

      // Test invalid CID
      try {
        client.sign(
          create(models.sign_v1.SignRequestSchema, {
            recordRef: {cid: 'invalid-cid'},
            provider: {
              request: {
                case: 'key',
                value: {
                  privateKey: Uint8Array.from([]),
                  password: Uint8Array.from([]),
                },
              },
            },
          }),
        );
        expect.fail('Should have thrown error for invalid CID');
      } catch (error) {
        if (error instanceof Error) {
          expect(error.message).toContain('failed to decode CID invalid-cid');
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
    const createResponse = await client.create_sync(
      create(models.store_v1.CreateSyncRequestSchema, {
        remoteDirectoryUrl:
          env['DIRECTORY_SERVER_PEER1_ADDRESS'] || '0.0.0.0:8891',
      }),
    );
    expect(createResponse).toBeTypeOf(
      typeof models.store_v1.CreateSyncResponseSchema,
    );

    const syncId = createResponse.syncId;
    expect(isValidUUID(syncId)).toBe(true);

    // List syncs
    const listResponse = await client.list_syncs(
      create(models.store_v1.ListSyncsRequestSchema, {}),
    );
    expect(listResponse).toBeInstanceOf(Array);

    for (const syncItem of listResponse) {
      expect(syncItem).toBeTypeOf(typeof models.store_v1.ListSyncsItemSchema);
      expect(isValidUUID(syncItem.syncId)).toBe(true);
    }

    // Get sync
    const getResponse = await client.get_sync(
      create(models.store_v1.GetSyncRequestSchema, {
        syncId: syncId,
      }),
    );
    expect(getResponse).toBeTypeOf(
      typeof models.store_v1.GetSyncResponseSchema,
    );
    expect(getResponse.syncId).toEqual(syncId);

    // Delete sync
    await client.delete_sync(
      create(models.store_v1.DeleteSyncRequestSchema, {
        syncId: syncId,
      }),
    );
  });
});

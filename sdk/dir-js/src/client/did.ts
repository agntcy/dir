// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { randomUUID } from 'crypto';

/**
 * Authentication message structure for DID authentication.
 */
interface AuthMessage {
  method: string;
  timestamp: number;
  nonce: string;
}

/**
 * Create a DID authentication interceptor for Connect.
 * 
 * This interceptor extracts DID and signature from request headers for each request.
 * The DID and signature should be added to headers before making the request.
 * 
 * This allows using the same client/interceptor with different DIDs per request.
 * 
 * @example
 * ```typescript
 * // Create interceptor once (no DID needed!)
 * const didInterceptor = createDIDAuthInterceptor();
 * 
 * const transport = createGrpcTransport({
 *   baseUrl: 'localhost:8888',
 *   interceptors: [didInterceptor],
 * });
 * 
 * // Use with different OASF records - add DID and signature to headers
 * const client = createClient(StoreService, transport);
 * 
 * await client.push(
 *   { records: [record1] },
 *   {
 *     headers: {
 *       'did': oasfRecord1.uid,  // e.g., "did:cheqd:testnet:abc#key-1"
 *       'did-signature': oasfRecord1.signature.signature
 *     }
 *   }
 * );
 * ```
 */
export function createDIDAuthInterceptor(): Interceptor {
  return (next) => async (req) => {
    // Create authentication message
    const authMsg: AuthMessage = {
      method: req.method.name,
      timestamp: Math.floor(Date.now() / 1000),
      nonce: randomUUID(),
    };
    const messageBytes = Buffer.from(JSON.stringify(authMsg), 'utf-8');

    // Extract DID and signature from request headers
    const didUrl = req.header.get('did');
    const signatureB64 = req.header.get('did-signature');

    if (!didUrl) {
      throw new Error(
        'No DID provided. Add DID URL to headers with key "did". ' +
        'Example: headers: { "did": "did:cheqd:testnet:abc123#key-1" }'
      );
    }

    if (!signatureB64) {
      throw new Error(
        'No signature provided. Add signature to headers with key "did-signature". ' +
        `Message to sign: ${messageBytes.toString('utf-8')}`
      );
    }

    // Remove DID and signature from headers (clean up)
    req.header.delete('did');
    req.header.delete('did-signature');

    // Encode message to base64
    const messageB64 = messageBytes.toString('base64');

    // Create DID-Auth header
    // Format: DID-Auth <did>#<vm_id>;<base64_message>;<base64_signature>
    const authValue = `DID-Auth ${didUrl};${messageB64};${signatureB64}`;
    req.header.set('authorization', authValue);

    // Continue with modified headers
    return await next(req);
  };
}
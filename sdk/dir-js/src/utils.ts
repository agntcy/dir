// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

/**
 * Utility functions for the AGNTCY Directory SDK.
 *
 * This module provides helper functions for encoding and decoding various
 * Directory objects, such as signatures and public keys, to and from RecordReferrer format.
 */

import { create } from '@bufbuild/protobuf';
import type { Signature } from '@buf/agntcy_dir.bufbuild_es/agntcy/dir/sign/v1/signature_pb';
import {
    RecordReferrer,
    RecordReferrerSchema,
} from '@buf/agntcy_dir.bufbuild_es/agntcy/dir/core/v1/record_pb';
import type { JsonObject } from '@bufbuild/protobuf';

// Referrer type constants
export const SIGNATURE_REFERRER_TYPE = 'agntcy.dir.sign.v1.Signature';
export const PUBLIC_KEY_REFERRER_TYPE = 'agntcy.dir.sign.v1.PublicKey';

/**
 * Encode a Signature object into a RecordReferrer.
 *
 * This function converts a Signature object into the RecordReferrer format
 * that can be used with PushReferrerRequest.
 *
 * @param signature - The Signature object to encode
 * @returns RecordReferrer object containing the encoded signature
 * @throws Error if the signature is invalid
 *
 * @example
 * ```typescript
 * const signature = create(SignatureSchema, {
 *   signature: 'dGVzdC1zaWduYXR1cmU=',
 *   annotations: { payload: 'test-payload-data' }
 * });
 * const referrer = encodeSignatureToReferrer(signature);
 * // Use referrer with PushReferrerRequest
 * ```
 */
export function encodeSignatureToReferrer(
    signature: Signature,
): RecordReferrer {
    // Marshal annotations map to JSON string for struct compatibility
    const annotationsJson =
        signature.annotations && Object.keys(signature.annotations).length > 0
            ? JSON.stringify(signature.annotations)
            : '';

    // Create the data object with signature fields
    const dataObject: Record<string, unknown> = {
        annotations: annotationsJson,
        signed_at: signature.signedAt || '',
        algorithm: signature.algorithm || '',
        signature: signature.signature || '',
        certificate: signature.certificate || '',
        content_type: signature.contentType || '',
        content_bundle: signature.contentBundle || '',
    };

    // Create and return the RecordReferrer
    // The data field accepts JsonObject directly
    return create(RecordReferrerSchema, {
        type: SIGNATURE_REFERRER_TYPE,
        data: dataObject as JsonObject,
    });
}

/**
 * Decode a Signature object from a RecordReferrer.
 *
 * This function extracts a Signature object from the RecordReferrer format
 * received from PullReferrerResponse.
 *
 * @param referrer - The RecordReferrer containing the signature data
 * @returns Signature object decoded from the referrer
 * @throws Error if the referrer data is invalid or missing
 *
 * @example
 * ```typescript
 * // Assume we got a referrer from PullReferrerResponse
 * const signature = decodeSignatureFromReferrer(response.referrer);
 * console.log(signature.signature);
 * ```
 */
export function decodeSignatureFromReferrer(
    referrer: RecordReferrer,
): Signature {
    if (!referrer.data) {
        throw new Error('Referrer data is empty');
    }

    // The data field is already a JsonObject
    const data = referrer.data as Record<string, unknown>;

    // Initialize signature object with empty values
    const signature: Partial<Signature> = {
        annotations: {},
        signedAt: '',
        algorithm: '',
        signature: '',
        certificate: '',
        contentType: '',
        contentBundle: '',
    };

    // Handle annotations - they can be either a JSON string or an object
    if ('annotations' in data) {
        const annotationsData = data.annotations;
        if (typeof annotationsData === 'string' && annotationsData) {
            // Annotations stored as JSON string
            try {
                signature.annotations = JSON.parse(annotationsData) as Record<
                    string,
                    string
                >;
            } catch {
                // Ignore invalid JSON
            }
        } else if (
            typeof annotationsData === 'object' &&
            annotationsData !== null
        ) {
            // Legacy format - annotations stored as object
            signature.annotations = {};
            for (const [k, v] of Object.entries(annotationsData)) {
                if (typeof v === 'string') {
                    signature.annotations[k] = v;
                }
            }
        }
    }

    // Extract other signature fields
    if ('signed_at' in data && typeof data.signed_at === 'string') {
        signature.signedAt = data.signed_at;
    }

    if ('algorithm' in data && typeof data.algorithm === 'string') {
        signature.algorithm = data.algorithm;
    }

    if ('signature' in data && typeof data.signature === 'string') {
        signature.signature = data.signature;
    }

    if ('certificate' in data && typeof data.certificate === 'string') {
        signature.certificate = data.certificate;
    }

    if ('content_type' in data && typeof data.content_type === 'string') {
        signature.contentType = data.content_type;
    }

    if ('content_bundle' in data && typeof data.content_bundle === 'string') {
        signature.contentBundle = data.content_bundle;
    }

    return signature as Signature;
}

/**
 * Encode a public key string into a RecordReferrer.
 *
 * This function converts a PEM-encoded public key string into the RecordReferrer
 * format that can be used with PushReferrerRequest.
 *
 * @param publicKey - PEM-encoded public key string
 * @returns RecordReferrer object containing the encoded public key
 * @throws Error if the public key is empty or invalid
 *
 * @example
 * ```typescript
 * const publicKey = '-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----';
 * const referrer = encodePublicKeyToReferrer(publicKey);
 * ```
 */
export function encodePublicKeyToReferrer(publicKey: string): RecordReferrer {
    if (!publicKey) {
        throw new Error('Public key cannot be empty');
    }

    // Create the data object with public key
    const dataObject: Record<string, unknown> = {
        key: publicKey,
    };

    // Create and return the RecordReferrer
    // The data field accepts JsonObject directly
    return create(RecordReferrerSchema, {
        type: PUBLIC_KEY_REFERRER_TYPE,
        data: dataObject as JsonObject,
    });
}

/**
 * Decode a public key string from a RecordReferrer.
 *
 * This function extracts a PEM-encoded public key from the RecordReferrer format.
 *
 * @param referrer - The RecordReferrer containing the public key data
 * @returns PEM-encoded public key string
 * @throws Error if the referrer data is invalid or missing the public key
 *
 * @example
 * ```typescript
 * const publicKey = decodePublicKeyFromReferrer(response.referrer);
 * console.log(publicKey);
 * ```
 */
export function decodePublicKeyFromReferrer(referrer: RecordReferrer): string {
    if (!referrer.data) {
        throw new Error('Referrer data is empty');
    }

    // The data field is already a JsonObject
    const data = referrer.data as Record<string, unknown>;

    if ('key' in data) {
        throw new Error('Public key not found in referrer data');
    }

    const publicKey = data.key;
    if (typeof publicKey !== 'string') {
        throw new Error('Public key must be a string');
    }

    return publicKey;
}


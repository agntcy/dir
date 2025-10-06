# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""Utility functions for the AGNTCY Directory SDK.

This module provides helper functions for encoding and decoding various
Directory objects, such as signatures and public keys, to and from RecordReferrer format.
"""

import json

from agntcy.dir_sdk.models import core_v1, sign_v1


# Referrer type constants using proto full names (for high-level API)
SIGNATURE_REFERRER_TYPE = "agntcy.dir.sign.v1.Signature"
PUBLIC_KEY_REFERRER_TYPE = "agntcy.dir.sign.v1.PublicKey"


def encode_signature_to_referrer(signature: sign_v1.Signature) -> core_v1.RecordReferrer:
    """Encode a Signature object into a RecordReferrer.
    
    This function converts a Signature object into the RecordReferrer format
    that can be used with PushReferrerRequest.
    
    Args:
        signature: The Signature object to encode
        
    Returns:
        RecordReferrer object containing the encoded signature
        
    Raises:
        ValueError: If the signature is invalid
        
    Example:
        >>> signature = sign_v1.Signature(
        ...     signature="dGVzdC1zaWduYXR1cmU=",
        ...     annotations={"payload": "test-payload-data"}
        ... )
        >>> referrer = encode_signature_to_referrer(signature)
        >>> # Use referrer with PushReferrerRequest
    """
    # Marshal annotations map to JSON string for struct compatibility
    annotations_json = ""
    if signature.annotations:
        annotations_json = json.dumps(dict(signature.annotations))
    
    # Create the data dict with signature fields
    # When passed to RecordReferrer, Python protobuf will automatically convert dict to Struct
    data_dict = {
        "annotations": annotations_json,
        "signed_at": signature.signed_at if signature.signed_at else "",
        "algorithm": signature.algorithm if signature.algorithm else "",
        "signature": signature.signature if signature.signature else "",
        "certificate": signature.certificate if signature.certificate else "",
        "content_type": signature.content_type if signature.content_type else "",
        "content_bundle": signature.content_bundle if signature.content_bundle else "",
    }
    
    # Create and return the RecordReferrer
    # Python protobuf automatically converts dict to Struct for data field
    return core_v1.RecordReferrer(
        type=SIGNATURE_REFERRER_TYPE,
        data=data_dict,
    )


def decode_signature_from_referrer(referrer: core_v1.RecordReferrer) -> sign_v1.Signature:
    """Decode a Signature object from a RecordReferrer.
    
    This function extracts a Signature object from the RecordReferrer format
    received from PullReferrerResponse.
    
    Args:
        referrer: The RecordReferrer containing the signature data
        
    Returns:
        Signature object decoded from the referrer
        
    Raises:
        ValueError: If the referrer data is invalid or missing
        
    Example:
        >>> # Assume we got a referrer from PullReferrerResponse
        >>> signature = decode_signature_from_referrer(response.referrer)
        >>> print(signature.signature)
    """
    if not referrer.data:
        raise ValueError("Referrer data is empty")
    
    # Convert Struct to dict
    data = dict(referrer.data)
    
    # Initialize signature object
    signature = sign_v1.Signature()
    
    # Handle annotations - they can be either a JSON string or a dict
    if "annotations" in data:
        annotations_data = data["annotations"]
        if isinstance(annotations_data, str) and annotations_data:
            # Annotations stored as JSON string
            try:
                signature.annotations.update(json.loads(annotations_data))
            except json.JSONDecodeError:
                pass  # Ignore invalid JSON
        elif isinstance(annotations_data, dict):
            # Legacy format - annotations stored as dict
            for k, v in annotations_data.items():
                if isinstance(v, str):
                    signature.annotations[k] = v
    
    # Extract other signature fields
    if "signed_at" in data and isinstance(data["signed_at"], str):
        signature.signed_at = data["signed_at"]
    
    if "algorithm" in data and isinstance(data["algorithm"], str):
        signature.algorithm = data["algorithm"]
    
    if "signature" in data and isinstance(data["signature"], str):
        signature.signature = data["signature"]
    
    if "certificate" in data and isinstance(data["certificate"], str):
        signature.certificate = data["certificate"]
    
    if "content_type" in data and isinstance(data["content_type"], str):
        signature.content_type = data["content_type"]
    
    if "content_bundle" in data and isinstance(data["content_bundle"], str):
        signature.content_bundle = data["content_bundle"]
    
    return signature


def encode_public_key_to_referrer(public_key: str) -> core_v1.RecordReferrer:
    """Encode a public key string into a RecordReferrer.
    
    This function converts a PEM-encoded public key string into the RecordReferrer
    format that can be used with PushReferrerRequest.
    
    Args:
        public_key: PEM-encoded public key string
        
    Returns:
        RecordReferrer object containing the encoded public key
        
    Raises:
        ValueError: If the public key is empty or invalid
        
    Example:
        >>> public_key = "-----BEGIN PUBLIC KEY-----\\n...\\n-----END PUBLIC KEY-----"
        >>> referrer = encode_public_key_to_referrer(public_key)
    """
    if not public_key:
        raise ValueError("Public key cannot be empty")
    
    # Create the data dict with public key
    # Python protobuf automatically converts dict to Struct for data field
    data_dict = {
        "key": public_key,
    }
    
    # Create and return the RecordReferrer
    return core_v1.RecordReferrer(
        type=PUBLIC_KEY_REFERRER_TYPE,
        data=data_dict,
    )


def decode_public_key_from_referrer(referrer: core_v1.RecordReferrer) -> str:
    """Decode a public key string from a RecordReferrer.
    
    This function extracts a PEM-encoded public key from the RecordReferrer format.
    
    Args:
        referrer: The RecordReferrer containing the public key data
        
    Returns:
        PEM-encoded public key string
        
    Raises:
        ValueError: If the referrer data is invalid or missing the public key
        
    Example:
        >>> public_key = decode_public_key_from_referrer(response.referrer)
        >>> print(public_key)
    """
    if not referrer.data:
        raise ValueError("Referrer data is empty")
    
    # Convert Struct to dict
    data = dict(referrer.data)
    
    if "key" not in data:
        raise ValueError("Public key not found in referrer data")
    
    public_key = data["key"]
    if not isinstance(public_key, str):
        raise ValueError("Public key must be a string")
    
    return public_key


// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package referrer provides encoding and decoding utilities for RecordReferrer objects.
package referrer

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// PublicKeyArtifactMediaType defines the media type for public key blobs.
	PublicKeyArtifactMediaType = "application/vnd.agntcy.dir.publickey.v1+pem"

	// SignatureArtifactType defines the media type for cosign signature layers.
	SignatureArtifactType = "application/vnd.dev.cosign.simplesigning.v1+json"

	// ReferrerArtifactMediaType defines the media type for referrer blobs.
	DefaultReferrerArtifactMediaType = "application/vnd.agntcy.dir.referrer.v1+json"
)

// EncodePublicKeyToReferrer creates a RecordReferrer for a public key.
func EncodePublicKeyToReferrer(publicKey string) (*corev1.RecordReferrer, error) {
	publicKeyStruct, err := structpb.NewStruct(map[string]any{
		"publicKey": publicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert public key to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: PublicKeyArtifactMediaType,
		Data: publicKeyStruct,
	}, nil
}

// DecodePublicKeyFromReferrer decodes a public key from a referrer object.
func DecodePublicKeyFromReferrer(referrer *corev1.RecordReferrer) (string, error) {
	if referrer.GetData() == nil {
		return "", errors.New("data struct is nil")
	}

	dataMap := referrer.GetData().AsMap()

	publicKeyValue, ok := dataMap["publicKey"]
	if !ok {
		return "", errors.New("publicKey field not found in data")
	}

	publicKey, ok := publicKeyValue.(string)
	if !ok {
		return "", errors.New("publicKey field is not a string")
	}

	if publicKey == "" {
		return "", errors.New("publicKey field is empty")
	}

	return publicKey, nil
}

// EncodeSignatureToReferrer creates a RecordReferrer for a signature.
func EncodeSignatureToReferrer(signature *signv1.Signature) (*corev1.RecordReferrer, error) {
	// Marshal annotations map[string]string into JSON string for structpb compatibility
	annotationsJSON := ""

	if annotations := signature.GetAnnotations(); len(annotations) > 0 {
		annotationsBytes, err := json.Marshal(annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations to JSON: %w", err)
		}

		annotationsJSON = string(annotationsBytes)
	}

	// Convert signature to struct for encoding
	signatureStruct, err := structpb.NewStruct(map[string]any{
		"annotations":    annotationsJSON,
		"signed_at":      signature.GetSignedAt(),
		"algorithm":      signature.GetAlgorithm(),
		"signature":      signature.GetSignature(),
		"certificate":    signature.GetCertificate(),
		"content_type":   signature.GetContentType(),
		"content_bundle": signature.GetContentBundle(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert signature to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: SignatureArtifactType,
		Data: signatureStruct,
	}, nil
}

// DecodeSignatureFromReferrer decodes a signature from a referrer object.
func DecodeSignatureFromReferrer(referrer *corev1.RecordReferrer) (*signv1.Signature, error) {
	if referrer.GetData() == nil {
		return nil, errors.New("data struct is nil")
	}

	data := referrer.GetData().AsMap()
	signature := &signv1.Signature{}

	// Handle annotations - they can be either a JSON string or a map
	if annotationsData, ok := data["annotations"]; ok {
		signature.Annotations = make(map[string]string)

		switch v := annotationsData.(type) {
		case string:
			// Annotations stored as JSON string
			if v != "" {
				var annotationsMap map[string]string
				if err := json.Unmarshal([]byte(v), &annotationsMap); err == nil {
					signature.Annotations = annotationsMap
				}
			}
		case map[string]interface{}:
			// Legacy format - annotations stored as map
			for k, val := range v {
				if str, ok := val.(string); ok {
					signature.Annotations[k] = str
				}
			}
		}
	}

	if signedAt, ok := data["signed_at"].(string); ok {
		signature.SignedAt = signedAt
	}

	if algorithm, ok := data["algorithm"].(string); ok {
		signature.Algorithm = algorithm
	}

	if sig, ok := data["signature"].(string); ok {
		signature.Signature = sig
	}

	if certificate, ok := data["certificate"].(string); ok {
		signature.Certificate = certificate
	}

	if contentType, ok := data["content_type"].(string); ok {
		signature.ContentType = contentType
	}

	if contentBundle, ok := data["content_bundle"].(string); ok {
		signature.ContentBundle = contentBundle
	}

	return signature, nil
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package publisher

import (
	"context"

	adscorev1 "github.com/agntcy/dir/api/core/v1"
	adsroutingv1 "github.com/agntcy/dir/api/routing/v1"
	adssignv1 "github.com/agntcy/dir/api/sign/v1"
	adsclient "github.com/agntcy/dir/client"
	"github.com/sigstore/cosign/v3/pkg/cosign"
)

// Publish an OASF record to ADS, sign it, then announce to DHT for federated discovery.
// Local node indexes it and can serve it locally over ADS or ARD right away.
func Publish(ctx context.Context, client *adsclient.Client) error {
	// Push locally
	ref, err := client.Push(ctx, AgentRecord)
	if err != nil {
		return err
	}

	// Sign the record to prove authenticity and integrity.
	if _, err := client.Sign(ctx, &adssignv1.SignRequest{
		RecordRef: ref,
		Provider:  getSigningProvider(),
	}); err != nil {
		return err
	}

	// Publish to DHT for federated discovery, so other nodes can find it.
	// Optional if only local discovery is needed.
	if err := client.Publish(ctx, &adsroutingv1.PublishRequest{
		Request: &adsroutingv1.PublishRequest_RecordRefs{
			RecordRefs: &adsroutingv1.RecordRefs{
				Refs: []*adscorev1.RecordRef{ref},
			},
		},
	}); err != nil {
		return err
	}

	return nil
}

// getSigningProvider creates a private key encoded in PEM format to sign the records with.
// Use OIDC-based signing if you want to use a third-party signing service instead of a key.
func getSigningProvider() *adssignv1.SignRequestProvider {
	key, err := cosign.GenerateKeyPair(nil)
	if err != nil {
		return nil
	}

	return &adssignv1.SignRequestProvider{Request: &adssignv1.SignRequestProvider_Key{
		Key: &adssignv1.SignWithKey{
			PrivateKey: string(key.PrivateBytes),
		},
	}}
}

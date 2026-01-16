// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/signing/eventswrap"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/registry"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/zot"
)

var logger = logging.Logger("signing")

// sign handles signature operations for records.
type sign struct {
	store     types.ReferrerStoreAPI
	ociConfig *ociconfig.Config
	zotConfig *zot.VerifyConfig
}

// New creates a new signing service.
func New(storeAPI types.StoreAPI, opts types.APIOptions) (types.SigningAPI, error) {
	// Check if store supports referrers (required for signing service)
	referrerStore, ok := storeAPI.(types.ReferrerStoreAPI)
	if !ok {
		return nil, errors.New("store does not support referrers, signing service unavailable")
	}

	cfg := opts.Config()

	s := &sign{
		store:     referrerStore,
		ociConfig: &cfg.Store.OCI,
	}

	// Configure Zot verification if using Zot registry
	if cfg.Store.OCI.GetType() == registry.RegistryTypeZot {
		s.zotConfig = &zot.VerifyConfig{
			RegistryAddress: cfg.Store.OCI.RegistryAddress,
			RepositoryName:  cfg.Store.OCI.RepositoryName,
			Username:        cfg.Store.OCI.Username,
			Password:        cfg.Store.OCI.Password,
			AccessToken:     cfg.Store.OCI.AccessToken,
			Insecure:        cfg.Store.OCI.Insecure,
		}

		logger.Debug("Signing service configured with Zot verification")
	}

	logger.Info("Signing service created")

	// Wrap with event emitter
	return eventswrap.Wrap(s, opts.EventBus()), nil
}

// ConvertCosignSignatureToReferrer converts cosign signature data to a referrer.
// This is used when reading signature referrers from the OCI registry.
func ConvertCosignSignatureToReferrer(blobAnnotations map[string]string, payload []byte) (*corev1.RecordReferrer, error) {
	// Extract the signature from the layer annotations
	var signatureValue string

	if blobAnnotations != nil {
		if sig, exists := blobAnnotations["dev.cosignproject.cosign/signature"]; exists {
			signatureValue = sig
		}
	}

	if signatureValue == "" {
		return nil, errors.New("no signature value found in annotations")
	}

	signature := &signv1.Signature{
		Signature: signatureValue,
		Annotations: map[string]string{
			"payload": string(payload),
		},
	}

	referrer, err := signature.MarshalReferrer()
	if err != nil {
		return nil, fmt.Errorf("failed to encode signature to referrer: %w", err)
	}

	return referrer, nil
}

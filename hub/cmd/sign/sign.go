// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
package sign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/service"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/spf13/cobra"
)

type SignOpts struct {
	// *hubOptions.HubOptions
	FulcioURL       string
	RekorURL        string
	TimestampURL    string
	OIDCProviderURL string
	OIDCClientID    string
	OIDCToken       string
	Key             string
}

var (
	signOpts SignOpts
	hubOpts  *hubOptions.HubOptions
)

func NewCommand(hubOptions *hubOptions.HubOptions) *cobra.Command {
	hubOpts = hubOptions

	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign record using identity-based OIDC or key-based signing",
		Long: `This command signs the record using identity-based signing.
It uses a short-lived signing certificate issued by Sigstore Fulcio
along with a local ephemeral signing key and OIDC identity.

Verification data is attached to the signed record,
and the transparency log is pushed to Sigstore Rekor.

This command opens a browser window to authenticate the user
with the default OIDC provider.

Usage examples:

1. Sign a record using OIDC:

	dirctl hub sign <org ID | org name> <record-cid>

2. Sign a record using key:

	dirctl hub sign <org ID | org name> <record-cid> --key <key-file>
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 { //nolint:mnd
				return errors.New("organization and record CID are required")
			}

			organization := args[0]
			recordCID := args[1]

			currentSession, err := authUtils.GetOrCreateSession(cmd, hubOpts.ServerAddress, hubOpts.APIKeyFile, false)
			if err != nil {
				return fmt.Errorf("failed to get or create session: %w", err)
			}

			hc, err := hubClient.New(currentSession.HubBackendAddress)
			if err != nil {
				return fmt.Errorf("failed to create hub client: %w", err)
			}

			signature, publicKey, err := sign(recordCID)
			if err != nil {
				return fmt.Errorf("failed to sign record: %w", err)
			}

			err = service.PushRecordSignature(cmd.Context(), hc, organization, recordCID, signature, publicKey, currentSession)
			if err != nil {
				return fmt.Errorf("failed to push record signature: %w", err)
			}

			_ = presenter.PrintMessage(cmd, "signature", "Record is", "signed")
			_ = presenter.PrintMessage(cmd, "signature", "Signature", signature)
			_ = presenter.PrintMessage(cmd, "signature", "Public Key", publicKey)

			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringVar(&signOpts.FulcioURL, "fulcio-url", cosign.DefaultFulcioURL,
		"Sigstore Fulcio URL")
	flags.StringVar(&signOpts.RekorURL, "rekor-url", cosign.DefaultRekorURL,
		"Sigstore Rekor URL")
	flags.StringVar(&signOpts.TimestampURL, "timestamp-url", cosign.DefaultTimestampURL,
		"Sigstore Timestamp URL")
	flags.StringVar(&signOpts.OIDCProviderURL, "oidc-provider-url", cosign.DefaultOIDCProviderURL,
		"OIDC Provider URL")
	flags.StringVar(&signOpts.OIDCClientID, "oidc-client-id", cosign.DefaultOIDCClientID,
		"OIDC Client ID")
	flags.StringVar(&signOpts.OIDCToken, "oidc-token", "",
		"OIDC Token for non-interactive signing. ")
	flags.StringVar(&signOpts.Key, "key", "",
		"Path to the private key file to use for signing (e.g., a Cosign key generated with a GitHub token). Use this option to sign with a self-managed keypair instead of OIDC identity-based signing.")

	return cmd
}

func sign(recordCID string) (string, string, error) {
	ctx := context.Background()

	var (
		signature string
		publicKey string
		err       error
	)

	switch {
	case signOpts.Key != "":
		rawKey, err := os.ReadFile(filepath.Clean(signOpts.Key))
		if err != nil {
			return "", "", fmt.Errorf("failed to read key file: %w", err)
		}

		pw, err := cosign.ReadPrivateKeyPassword()()
		if err != nil {
			return "", "", fmt.Errorf("failed to read password: %w", err)
		}

		signature, publicKey, err = signWithKey(ctx, recordCID, rawKey, pw)
		if err != nil {
			return "", "", fmt.Errorf("failed to sign record with key: %w", err)
		}

	case signOpts.OIDCToken != "":
		signature, publicKey, err = signWithOIDCToken(ctx, recordCID, signOpts.OIDCToken)
		if err != nil {
			return "", "", fmt.Errorf("failed to sign record with OIDC token: %w", err)
		}

	default: // Retrieve the token from the OIDC provider using interactive flow
		token, err := oauthflow.OIDConnect(signOpts.OIDCProviderURL, signOpts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return "", "", fmt.Errorf("failed to get OIDC token: %w", err)
		}

		signature, publicKey, err = signWithOIDCToken(ctx, recordCID, token.RawString)
		if err != nil {
			return "", "", fmt.Errorf("failed to sign record with OIDC: %w", err)
		}
	}

	return signature, publicKey, nil
}

func signWithKey(ctx context.Context, recordCID string, privateKey []byte, password []byte) (string, string, error) {
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return "", "", fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	payloadBytes, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate payload: %w", err)
	}

	result, err := cosign.SignBlobWithKey(ctx, &cosign.SignBlobKeyOptions{
		Payload:    payloadBytes,
		PrivateKey: privateKey,
		Password:   password,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to sign with key: %w", err)
	}

	return result.Signature, result.PublicKey, nil
}

func signWithOIDCToken(ctx context.Context, recordCID string, token string) (string, string, error) {
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return "", "", fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	payloadBytes, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate payload: %w", err)
	}

	result, err := cosign.SignBlobWithOIDC(ctx, &cosign.SignBlobOIDCOptions{
		Payload:         payloadBytes,
		IDToken:         token,
		FulcioURL:       signOpts.FulcioURL,
		RekorURL:        signOpts.RekorURL,
		TimestampURL:    signOpts.TimestampURL,
		OIDCProviderURL: signOpts.OIDCProviderURL,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to sign with OIDC: %w", err)
	}

	return result.Signature, result.PublicKey, nil
}

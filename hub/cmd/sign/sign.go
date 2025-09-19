// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive,mnd
package sign

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/utils/cosign"
	corev1alpha1 "github.com/agntcy/dirhub/backport/api/core/v1alpha1"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

var Command = &cobra.Command{
	Use:   "sign",
	Short: "Sign agent model using identity-based OIDC signing",
	Long: `This command signs the agent data model using identity-based signing.
It uses a short-lived signing certificate issued by Sigstore Fulcio
along with a local ephemeral signing key and OIDC identity.

Verification data is attached to the signed agent model,
and the transparency log is pushed to Sigstore Rekor.

This command opens a browser window to authenticate the user 
with the default OIDC provider.

Usage examples:

1. Sign an agent model from file:

	dirctl sign agent.json

2. Sign an agent model from standard input:

	cat agent.json | dirctl sign --stdin

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var fpath string
		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			fpath = args[0]
		}

		// get source
		source, err := GetCmdReader(fpath, opts.FromStdin)
		if err != nil {
			return err //nolint:wrapcheck
		}

		return runCommand(cmd, source)
	},
}

func runCommand(cmd *cobra.Command, source io.ReadCloser) error {
	// Load data from source
	agent := &corev1alpha1.Agent{}
	if _, err := agent.LoadFromReader(source); err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	defer source.Close() //nolint:errcheck

	var agentSigned *corev1alpha1.Agent

	//nolint:nestif,gocritic
	if opts.Key != "" {
		// Load the key from file
		rawKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		// Read password from environment variable
		pw, err := cosign.ReadPrivateKeyPassword()()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		// Sign the agent using the provided key
		agentSigned, err = SignWithKey(cmd.Context(), rawKey, pw, agent)
		if err != nil {
			return fmt.Errorf("failed to sign agent with key: %w", err)
		}
	} else if opts.OIDCToken != "" {
		// Sign the agent using the OIDC provider
		var err error

		agentSigned, err = SignOIDC(cmd.Context(), agent, opts.OIDCToken)
		if err != nil {
			return fmt.Errorf("failed to sign agent: %w", err)
		}
	} else {
		// Retrieve the token from the OIDC provider
		token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, "", "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}

		// Sign the agent using the OIDC provider
		agentSigned, err = SignOIDC(cmd.Context(), agent, token.RawString)
		if err != nil {
			return fmt.Errorf("failed to sign agent: %w", err)
		}
	}

	// Print signed agent
	signedAgentJSON, err := json.MarshalIndent(agentSigned, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(signedAgentJSON))

	return nil
}

func SignOIDC(ctx context.Context, agent *corev1alpha1.Agent, idToken string) (*corev1alpha1.Agent, error) {
	// Validate request.
	if agent == nil {
		return nil, errors.New("agent must be set")
	}

	// Load signing options.
	var signOpts sign.BundleOptions
	{
		// Define config to use for signing.
		signingConfig, err := root.NewSigningConfig(
			root.SigningConfigMediaType02,
			// Fulcio URLs
			[]root.Service{
				{
					URL:                 opts.FulcioURL,
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// OIDC Provider URLs
			// Usage and requirements: https://docs.sigstore.dev/certificate_authority/oidc-in-fulcio/
			[]root.Service{
				{
					URL:                 opts.OIDCProviderURL,
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			// Rekor URLs
			[]root.Service{
				{
					URL:                 opts.RekorURL,
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			root.ServiceConfiguration{
				Selector: v1.ServiceSelector_ANY,
			},
			[]root.Service{
				{
					URL:                 opts.TimestampURL,
					MajorAPIVersion:     1,
					ValidityPeriodStart: time.Now().Add(-time.Hour),
					ValidityPeriodEnd:   time.Now().Add(time.Hour),
				},
			},
			root.ServiceConfiguration{
				Selector: v1.ServiceSelector_ANY,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create signing config: %w", err)
		}

		// Use fulcio to sign the agent.
		fulcioURL, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), []uint32{1}, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to select fulcio URL: %w", err)
		}

		fulcioOpts := &sign.FulcioOptions{
			BaseURL: fulcioURL,
			Timeout: 30 * time.Second,
			Retries: 1,
		}
		signOpts.CertificateProvider = sign.NewFulcio(fulcioOpts)
		signOpts.CertificateProviderOptions = &sign.CertificateProviderOptions{
			IDToken: idToken,
		}

		// Use timestamp authortiy to sign the agent.
		tsaURLs, err := root.SelectServices(signingConfig.TimestampAuthorityURLs(),
			signingConfig.TimestampAuthorityURLsConfig(), []uint32{1}, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to select timestamp authority URL: %w", err)
		}

		for _, tsaURL := range tsaURLs {
			tsaOpts := &sign.TimestampAuthorityOptions{
				URL:     tsaURL,
				Timeout: 30 * time.Second,
				Retries: 1,
			}
			signOpts.TimestampAuthorities = append(signOpts.TimestampAuthorities, sign.NewTimestampAuthority(tsaOpts))
		}

		// Use rekor to sign the agent.
		rekorURLs, err := root.SelectServices(signingConfig.RekorLogURLs(),
			signingConfig.RekorLogURLsConfig(), []uint32{1}, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to select rekor URL: %w", err)
		}

		for _, rekorURL := range rekorURLs {
			rekorOpts := &sign.RekorOptions{
				BaseURL: rekorURL,
				Timeout: 90 * time.Second,
				Retries: 1,
			}
			signOpts.TransparencyLogs = append(signOpts.TransparencyLogs, sign.NewRekor(rekorOpts))
		}
	}

	// Generate an ephemeral keypair for signing.
	signKeypair, err := sign.NewEphemeralKeypair(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ephemeral keypair: %w", err)
	}

	return Sign(ctx, agent, signKeypair, signOpts)
}

func SignWithKey(ctx context.Context, privKey []byte, pw []byte, agent *corev1alpha1.Agent) (*corev1alpha1.Agent, error) {
	// Generate a keypair from the provided private key bytes.
	// The keypair hint is derived from the public key and will be used for verification.
	signKeypair, err := cosign.LoadKeypair(privKey, pw)
	if err != nil {
		return nil, fmt.Errorf("failed to create keypair: %w", err)
	}

	return Sign(ctx, agent, signKeypair, sign.BundleOptions{})
}

func Sign(_ context.Context, agent *corev1alpha1.Agent, signKeypair sign.Keypair, signOpts sign.BundleOptions) (*corev1alpha1.Agent, error) {
	// Reset the signature field in the agent.
	// This is required as the agent may have been signed before,
	// but also because this ensures signing idempotency.
	agent.Signature = nil

	// Convert the agent to JSON.
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent: %w", err)
	}

	// Sign the agent JSON data.
	sigBundle, err := sign.Bundle(&sign.PlainData{Data: agentJSON}, signKeypair, signOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to sign agent: %w", err)
	}

	certData := sigBundle.GetVerificationMaterial()
	sigData := sigBundle.GetMessageSignature()

	// Extract data from the signature bundle.
	sigBundleJSON, err := protojson.Marshal(sigBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bundle: %w", err)
	}

	// Update the agent with the signature details.
	agent.Signature = &corev1alpha1.Signature{
		Algorithm:     sigData.GetMessageDigest().GetAlgorithm().String(),
		Signature:     base64.StdEncoding.EncodeToString(sigData.GetSignature()),
		Certificate:   base64.StdEncoding.EncodeToString(certData.GetCertificate().GetRawBytes()),
		ContentType:   sigBundle.GetMediaType(),
		ContentBundle: base64.StdEncoding.EncodeToString(sigBundleJSON),
		SignedAt:      time.Now().Format(time.RFC3339),
	}

	return agent, nil
}

func GetCmdReader(fpath string, fromStdin bool) (io.ReadCloser, error) {
	if fpath == "" && !fromStdin {
		return nil, errors.New("if no path defined --stdin flag must be set")
	}

	if fpath != "" {
		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", fpath, err)
		}

		return file, nil
	}

	return os.Stdin, nil
}

// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package sign

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/cosign/v3/pkg/cosign/env"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "sign",
	Short: "Sign record using identity-based OIDC or key-based signing",
	Long: `This command signs the record using identity-based signing.
It uses a short-lived signing certificate issued by Sigstore Fulcio
along with a local ephemeral signing key and OIDC identity.

Verification data is attached to the signed record,
and the transparency log is pushed to Sigstore Rekor.

This command opens a browser window to authenticate the user
with the default OIDC provider.

Password for encrypted private keys:
When using an encrypted private key, the password can be provided via:
  1. COSIGN_PASSWORD environment variable
  2. --password-stdin to explicitly read it from standard input
  3. Interactive terminal prompt (if running in a terminal)

In a non-interactive environment, standard input is read only when
--password-stdin is set. Otherwise, COSIGN_PASSWORD is used when present and an
empty password is used when it is absent. Local and inline PEM keys must use the
encrypted Cosign/Sigstore private-key format. Generate a compatible key pair
with "cosign generate-key-pair".

Usage examples:

1. Sign a record using OIDC:

	dirctl sign <record-cid>

2. Sign a record using key file:

	dirctl sign <record-cid> --key /path/to/cosign.key

3. Sign with encrypted key (password from env):

	COSIGN_PASSWORD=mypassword dirctl sign <record-cid> --key cosign.key

4. Sign with password from standard input:

	printf '%s' "$KEY_PASSWORD" | dirctl sign <record-cid> --key cosign.key --password-stdin

5. Output formats:

	# Get signing result as JSON
	dirctl sign <record-cid> --output json
	
	# Sign with key and JSON output
	dirctl sign <record-cid> --key <key-file> --output json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var recordCID string

		if len(args) > 1 {
			return errors.New("only one record CID is allowed")
		} else if len(args) == 1 {
			recordCID = args[0]
		} else {
			return errors.New("record CID is required")
		}

		return runCommand(cmd, recordCID)
	},
}

func runCommand(cmd *cobra.Command, recordCID string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	err := Sign(cmd.Context(), c, recordCID)
	if err != nil {
		return fmt.Errorf("failed to sign record: %w", err)
	}

	// Output in the appropriate format
	return presenter.PrintMessage(cmd, "signature", "Record is", "signed")
}

func Sign(ctx context.Context, c *client.Client, recordCID string) error {
	// Construct the sign request with the provided options
	var provider *signv1.SignRequestProvider

	switch {
	case opts.Key != "":
		// Read password from environment variable or terminal
		pw, err := readPrivateKeyPassword()()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		// Sign the record using the provided key reference
		// The key can be a file path, URL, KMS URI, etc.
		provider = &signv1.SignRequestProvider{
			Request: &signv1.SignRequestProvider_Key{
				Key: &signv1.SignWithKey{
					PrivateKey: opts.Key,
					Password:   pw,
				},
			},
		}

	case opts.OIDCToken != "":
		// Sign the record using the OIDC provider
		provider = &signv1.SignRequestProvider{
			Request: &signv1.SignRequestProvider_Oidc{
				Oidc: &signv1.SignWithOIDC{
					IdToken: opts.OIDCToken,
					Options: &signv1.SignOptionsOIDC{
						FulcioUrl:        opts.FulcioURL,
						RekorUrl:         opts.RekorURL,
						TimestampUrl:     opts.TimestampURL,
						OidcProviderUrl:  opts.OIDCProviderURL,
						OidcClientId:     opts.OIDCClientID,
						OidcClientSecret: opts.OIDCClientSecret,
						SkipTlog:         opts.SkipTlog,
					},
				},
			},
		}

	default:
		// Retrieve the token from the OIDC provider
		token, err := oauthflow.OIDConnect(opts.OIDCProviderURL, opts.OIDCClientID, opts.OIDCClientSecret, "", oauthflow.DefaultIDTokenGetter)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}

		// Sign the record using the OIDC provider
		provider = &signv1.SignRequestProvider{
			Request: &signv1.SignRequestProvider_Oidc{
				Oidc: &signv1.SignWithOIDC{
					IdToken: token.RawString,
					Options: &signv1.SignOptionsOIDC{
						FulcioUrl:        opts.FulcioURL,
						RekorUrl:         opts.RekorURL,
						TimestampUrl:     opts.TimestampURL,
						OidcProviderUrl:  opts.OIDCProviderURL,
						OidcClientId:     opts.OIDCClientID,
						OidcClientSecret: opts.OIDCClientSecret,
						SkipTlog:         opts.SkipTlog,
					},
				},
			},
		}
	}

	// Sign the record using given provider
	_, err := c.Sign(ctx, &signv1.SignRequest{
		RecordRef: &corev1.RecordRef{Cid: recordCID},
		Provider:  provider,
	})
	if err != nil {
		if opts.Key != "" {
			err = formatPrivateKeyError(err)
		}

		return fmt.Errorf("failed to sign record: %w", err)
	}

	return nil
}

func formatPrivateKeyError(err error) error {
	// cosign.LoadPrivateKey currently returns unsupported PEM types as an
	// untyped error, so preserve the original error while adding actionable
	// guidance for dirctl users.
	if strings.Contains(err.Error(), "unsupported pem type") { //nolint:errorlint // cosign exposes no typed/sentinel error
		return fmt.Errorf(
			"unsupported private key format: expected an encrypted Cosign/Sigstore private key "+
				"(run \"cosign generate-key-pair\"): %w",
			err,
		)
	}

	if strings.Contains(err.Error(), "decrypt:") { //nolint:errorlint // cosign exposes no typed/sentinel error
		return fmt.Errorf(
			"failed to decrypt private key: set COSIGN_PASSWORD or use --password-stdin: %w",
			err,
		)
	}

	return err
}

type privateKeyPasswordReader struct {
	lookupPassword func() (string, bool)
	passwordStdin  bool
	stdin          io.Reader
	isTerminal     func() bool
	readTerminal   func(bool) ([]byte, error)
}

func (r privateKeyPasswordReader) read() ([]byte, error) {
	pw, ok := r.lookupPassword()

	switch {
	case ok:
		return []byte(pw), nil
	case r.passwordStdin:
		return io.ReadAll(r.stdin)
	case r.isTerminal():
		return r.readTerminal(true)
	default:
		// Match the previous EOF behavior without blocking indefinitely on an
		// unrelated or permanently open stdin stream. This also allows passwordless
		// local keys and key references that do not consume a password (for example KMS).
		return []byte{}, nil
	}
}

func readPrivateKeyPassword() func() ([]byte, error) {
	return privateKeyPasswordReader{
		lookupPassword: func() (string, bool) {
			return env.LookupEnv(env.VariablePassword)
		},
		passwordStdin: opts.PasswordStdin,
		stdin:         os.Stdin,
		isTerminal:    cosign.IsTerminal,
		readTerminal:  cosign.GetPassFromTerm,
	}.read
}

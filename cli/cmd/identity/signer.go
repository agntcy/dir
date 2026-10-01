// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/agntcy/dir/client/utils/jws"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/cosign/v3/pkg/cosign/env"
)

// encryptedKeyPEMType is the PEM block type of a password-protected PKCS#8 key.
const encryptedKeyPEMType = "ENCRYPTED PRIVATE KEY"

func readFile(path, what string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s %q: %w", what, path, err)
	}

	return data, nil
}

// loadSigner reads the PEM private key at path. The password is only asked for
// when the key is encrypted, so an unencrypted key never prompts.
func loadSigner(path string, passwordStdin bool) (jws.Signer, error) {
	keyPEM, err := readFile(path, "private key")
	if err != nil {
		return nil, err
	}

	var password []byte

	if block, _ := pem.Decode(keyPEM); block != nil && block.Type == encryptedKeyPEMType {
		password, err = readPassword(passwordSource{
			lookupEnv:     func() (string, bool) { return env.LookupEnv(env.VariablePassword) },
			passwordStdin: passwordStdin,
			stdin:         os.Stdin,
			isTerminal:    cosign.IsTerminal,
			readTerminal:  cosign.GetPassFromTerm,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to read private key password: %w", err)
		}
	}

	signer, err := jws.NewKeySigner(keyPEM, password)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key %q: %w", path, err)
	}

	return signer, nil
}

type passwordSource struct {
	lookupEnv     func() (string, bool)
	passwordStdin bool
	stdin         io.Reader
	isTerminal    func() bool
	readTerminal  func(confirm bool) ([]byte, error)
}

// readPassword takes the password from, in order: the COSIGN_PASSWORD
// environment variable, standard input when asked to, or a terminal prompt.
func readPassword(src passwordSource) ([]byte, error) {
	if pw, ok := src.lookupEnv(); ok {
		return []byte(pw), nil
	}

	switch {
	case src.passwordStdin:
		pw, err := io.ReadAll(src.stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read password from stdin: %w", err)
		}

		// "echo secret |" must work: a trailing line break is not part of a password.
		return bytes.TrimRight(pw, "\r\n"), nil
	case src.isTerminal():
		pw, err := src.readTerminal(false)
		if err != nil {
			return nil, fmt.Errorf("%w: set COSIGN_PASSWORD or use --password-stdin", err)
		}

		return pw, nil
	default:
		return nil, errors.New("the private key is encrypted: set COSIGN_PASSWORD or use --password-stdin")
	}
}

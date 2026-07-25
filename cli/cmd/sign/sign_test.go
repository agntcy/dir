// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

type trackingReader struct {
	read bool
}

func (r *trackingReader) Read([]byte) (int, error) {
	r.read = true

	return 0, io.EOF
}

func TestAddSigningFlagsRegistersPasswordStdin(t *testing.T) {
	original := opts.PasswordStdin

	t.Cleanup(func() {
		opts.PasswordStdin = original
	})

	flags := pflag.NewFlagSet("sign", pflag.ContinueOnError)
	AddSigningFlags(flags)

	if err := flags.Parse([]string{"--password-stdin"}); err != nil {
		t.Fatalf("parse --password-stdin: %v", err)
	}

	if !opts.PasswordStdin {
		t.Fatal("--password-stdin did not enable stdin password reading")
	}
}

func TestPrivateKeyPasswordReaderDefaultsToEmptyWithoutTerminal(t *testing.T) {
	t.Parallel()

	stdin := &trackingReader{}
	reader := privateKeyPasswordReader{
		lookupPassword: func() (string, bool) { return "", false },
		passwordStdin:  false,
		stdin:          stdin,
		isTerminal:     func() bool { return false },
		readTerminal: func(bool) ([]byte, error) {
			t.Fatal("terminal reader must not be called")

			return nil, errors.New("unreachable")
		},
	}

	password, err := reader.read()
	if err != nil {
		t.Fatalf("read password: %v", err)
	}

	if len(password) != 0 {
		t.Fatalf("password = %q, want empty password", password)
	}

	if stdin.read {
		t.Fatal("password reader attempted to read non-interactive stdin")
	}
}

func TestPrivateKeyPasswordReaderUsesEnvironmentBeforeStdin(t *testing.T) {
	t.Parallel()

	stdin := &trackingReader{}
	reader := privateKeyPasswordReader{
		lookupPassword: func() (string, bool) { return "", true },
		passwordStdin:  true,
		stdin:          stdin,
		isTerminal: func() bool {
			t.Fatal("terminal detection must not be called")

			return false
		},
		readTerminal: func(bool) ([]byte, error) {
			t.Fatal("terminal reader must not be called")

			return nil, errors.New("unreachable")
		},
	}

	password, err := reader.read()
	if err != nil {
		t.Fatalf("read password: %v", err)
	}

	if len(password) != 0 {
		t.Fatalf("password = %q, want empty password", password)
	}

	if stdin.read {
		t.Fatal("password reader ignored COSIGN_PASSWORD precedence")
	}
}

func TestPrivateKeyPasswordReaderReadsStdinOnlyWhenRequested(t *testing.T) {
	t.Parallel()

	const want = "secret"

	reader := privateKeyPasswordReader{
		lookupPassword: func() (string, bool) { return "", false },
		passwordStdin:  true,
		stdin:          strings.NewReader(want),
		isTerminal: func() bool {
			t.Fatal("terminal detection must not be called")

			return false
		},
		readTerminal: func(bool) ([]byte, error) {
			t.Fatal("terminal reader must not be called")

			return nil, errors.New("unreachable")
		},
	}

	password, err := reader.read()
	if err != nil {
		t.Fatalf("read password: %v", err)
	}

	if string(password) != want {
		t.Fatalf("password = %q, want %q", password, want)
	}
}

func TestPrivateKeyPasswordReaderUsesTerminal(t *testing.T) {
	t.Parallel()

	const want = "terminal-secret"

	reader := privateKeyPasswordReader{
		lookupPassword: func() (string, bool) { return "", false },
		passwordStdin:  false,
		stdin:          &trackingReader{},
		isTerminal:     func() bool { return true },
		readTerminal: func(confirm bool) ([]byte, error) {
			if !confirm {
				t.Fatal("terminal password must request confirmation")
			}

			return []byte(want), nil
		},
	}

	password, err := reader.read()
	if err != nil {
		t.Fatalf("read password: %v", err)
	}

	if string(password) != want {
		t.Fatalf("password = %q, want %q", password, want)
	}
}

func TestFormatPrivateKeyErrorAddsCosignGuidance(t *testing.T) {
	t.Parallel()

	cause := errors.New("unsupported pem type: PRIVATE KEY")
	err := formatPrivateKeyError(cause)

	if !errors.Is(err, cause) {
		t.Fatalf("formatted error does not preserve cause: %v", err)
	}

	if !strings.Contains(err.Error(), "cosign generate-key-pair") {
		t.Fatalf("formatted error %q does not include key-generation guidance", err)
	}
}

func TestFormatPrivateKeyErrorAddsPasswordGuidance(t *testing.T) {
	t.Parallel()

	cause := errors.New("decrypt: invalid password")
	err := formatPrivateKeyError(cause)

	if !errors.Is(err, cause) {
		t.Fatalf("formatted error does not preserve cause: %v", err)
	}

	if !strings.Contains(err.Error(), "COSIGN_PASSWORD") || !strings.Contains(err.Error(), "--password-stdin") {
		t.Fatalf("formatted error %q does not include password guidance", err)
	}
}

func TestFormatPrivateKeyErrorLeavesOtherErrorsUnchanged(t *testing.T) {
	t.Parallel()

	cause := errors.New("key file not found")
	got := formatPrivateKeyError(cause)

	if !errors.Is(got, cause) {
		t.Fatalf("formatted error does not preserve cause: %v", got)
	}

	if strings.Contains(got.Error(), "unsupported private key format") {
		t.Fatalf("formatPrivateKeyError(%v) added unrelated guidance: %v", cause, got)
	}
}

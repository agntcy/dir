// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// GenerateCosignKeyPair generates a cosign key pair in the specified directory.
// Helper function for signature testing.
func GenerateCosignKeyPair(dir, password string) {
	// Prepare cosign generate-key-pair command
	cmd := exec.CommandContext(context.Background(), "cosign", "generate-key-pair")
	cmd.Dir = dir

	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+password)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cosign generate-key-pair failed: %v\nOutput: %s", err, string(output)))
	}
}

const (
	certValidity    = 24 * time.Hour
	filePermPrivate = 0o600
)

// GenerateSpiffeTestCert generates a self-signed ECDSA P256 X.509 certificate
// with the given SPIFFE ID as a URI SAN. Returns paths to the PEM-encoded
// private key and certificate written into dir.
func GenerateSpiffeTestCert(dir, spiffeID string) (string, string) {
	spiffeURI, err := url.Parse(spiffeID)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "invalid SPIFFE ID URI: %s", spiffeID)

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: spiffeID},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(certValidity),
		URIs:         []*url.URL{spiffeURI},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	keyDER, err := x509.MarshalECPrivateKey(privKey)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	privKeyPath := filepath.Join(dir, "spiffe.key")
	certPath := filepath.Join(dir, "spiffe.crt")

	err = os.WriteFile(privKeyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), filePermPrivate)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), filePermPrivate)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return privKeyPath, certPath
}

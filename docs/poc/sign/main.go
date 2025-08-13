package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	registry   = "localhost:5000"
	repo       = "demo"
	keyFile    = "cosign.key"
	pubKeyFile = "cosign.pub"
)

func main() {
	ctx := context.Background()

	// Wait for zot to be ready
	fmt.Println("Waiting for zot to be ready...")
	time.Sleep(10 * time.Second)

	// create client to registry
	client, err := remote.NewRepository(fmt.Sprintf("%s/%s", registry, repo))
	if err != nil {
		fmt.Printf("failed to connect to registry: %v\n", err)
		return
	}
	client.PlainHTTP = true

	// push artifacts to source registry
	artifacts := []string{"data/artifact1.json", "data/artifact2.json", "data/artifact3.json"}
	for _, artifact := range artifacts {
		err = pushArtifact(ctx, client, artifact)
		if err != nil {
			fmt.Printf("failed to push %s: %v\n", artifact, err)
			return
		}
	}

	// Generate key pair with cosign using cosign command
	err = generateCosignKeyPair()
	if err != nil {
		fmt.Printf("failed to generate key pair: %v\n", err)
		return
	}

	// Upload the public key to zot for signature verification
	err = uploadPublicKeyToZot()
	if err != nil {
		fmt.Printf("failed to upload public key: %v\n", err)
		// Continue anyway as this is optional
	}

	// Sign artifact 1 with key pair using cosign sign command
	err = signArtifact1("artifact1")
	if err != nil {
		fmt.Printf("failed to sign %s: %v\n", "artifact1", err)
		return
	}

	// Sign artifact 2 with key pair using cosign sign and attach commands
	err = signArtifact2("artifact2")
	if err != nil {
		fmt.Printf("failed to sign %s: %v\n", "artifact2", err)
		return
	}

	// Sign artifact 3 with key pair using cosign
	err = signArtifact3("data/artifact3.json", client)
	if err != nil {
		fmt.Printf("failed to sign %s: %v\n", "artifact3", err)
		return
	}

	// Sign artifact 4 with key pair using cosign sign-blob and attach commands
	err = signArtifact4("data/artifact4.json", client)
	if err != nil {
		fmt.Printf("failed to sign %s: %v\n", "artifact4", err)
		return
	}

	// Wait for zot to process signatures asynchronously
	fmt.Println("Waiting for zot to process signatures...")
	time.Sleep(30 * time.Second)

	// Verify signatures with zot using GraphQL API for detailed information
	err = verifySignatureWithZot("artifact1")
	if err != nil {
		fmt.Printf("failed to verify signature for artifact 1: %v\n", err)
		// Continue anyway as this is optional
	}

	err = verifySignatureWithZot("artifact2")
	if err != nil {
		fmt.Printf("Successfully failed to verify signature not found for artifact 2: %v\n", err)
	}

	err = verifySignatureWithZot("artifact3")
	if err != nil {
		fmt.Printf("failed to verify signature for artifact 3: %v\n", err)
		// Continue anyway as this is optional
	}

	err = verifySignatureWithZot("artifact4")
	if err != nil {
		fmt.Printf("failed to verify signature for artifact 4: %v\n", err)
		// Continue anyway as this is optional
	}
}

// generateCosignKeyPair generates a cosign key pair using the cosign command
func generateCosignKeyPair() error {
	// Check if cosign is available
	if _, err := exec.LookPath("cosign"); err != nil {
		return fmt.Errorf("cosign not found in PATH: %v", err)
	}

	// Set environment variable for empty password
	cmd := exec.Command("cosign", "generate-key-pair")
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign generate-key-pair failed: %v\nOutput: %s", err, string(output))
	}

	// Verify that key files were created
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("private key file %s was not created", keyFile)
	}
	if _, err := os.Stat(pubKeyFile); os.IsNotExist(err) {
		return fmt.Errorf("public key file %s was not created", pubKeyFile)
	}

	fmt.Printf("Key pair generated successfully: %s and %s\n", keyFile, pubKeyFile)
	return nil
}

// signArtifact signs an artifact using cosign with the generated key pair
func signArtifact1(artifactName string) error {
	// Build the artifact reference
	artifactRef := fmt.Sprintf("%s/%s:%s", registry, repo, artifactName)

	// Prepare cosign sign command with OCI 1.1 referrers support
	cmd := exec.Command("cosign", "sign", "-y",
		"--key", keyFile,
		artifactRef,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign sign failed for %s: %v\nOutput: %s", artifactName, err, string(output))
	}

	fmt.Printf("Cosign sign output: %s\n", string(output))

	return nil
}

func signArtifact2(artifactName string) error {
	// Build the artifact reference
	artifactRef := fmt.Sprintf("%s/%s:%s", registry, repo, artifactName)

	cmd := exec.Command("cosign", "sign", "-y",
		"--key", keyFile,
		"--upload=false",
		"--output-signature", "signature-art2.sig",
		"--output-payload", "payload-art2.json",
		artifactRef,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign sign failed for %s: %v\nOutput: %s", artifactName, err, string(output))
	}

	fmt.Printf("Cosign sign output: %s\n", string(output))

	// Attach the signature to the artifact
	cmd = exec.Command("cosign", "attach", "signature",
		"--signature", "signature-art2.sig",
		"--payload", "payload-art2.json",
		artifactRef,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")

	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign attach signature failed for %s: %v\nOutput: %s", artifactName, err, string(output))
	}

	fmt.Printf("Cosign attach signature output: %s\n", string(output))

	// Remove the payload and signature files
	_ = os.Remove("payload-art2.json")
	_ = os.Remove("signature-art2.sig")

	return nil
}

func signArtifact3(artifactName string, client *remote.Repository) error {
	// Push the artifact to the registry first
	err := pushArtifact(context.Background(), client, artifactName)
	if err != nil {
		return fmt.Errorf("failed to push artifact: %v\n", err)
	}

	// Get reference to the artifact
	baseName := filepath.Base(artifactName)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	artifactRef := fmt.Sprintf("%s/%s:%s", registry, repo, baseName)

	// Get digest of the artifact from the registry
	digest, err := getArtifactDigest(context.Background(), client, baseName)
	if err != nil {
		return fmt.Errorf("failed to get artifact digest: %v", err)
	}

	// Create the payload that both signing methods will use
	payload := fmt.Sprintf(`{"critical":{"identity":{"docker-reference":"%s/%s"},"image":{"docker-manifest-digest":"%s"},"type":"cosign container image signature"},"optional":null}`, registry, repo, digest)
	err = os.WriteFile("payload-art3.json", []byte(payload), 0644)
	if err != nil {
		return fmt.Errorf("failed to write payload: %v\n", err)
	}

	// Sign the payload using sign-blob (this will now match cosign sign)
	cmd := exec.Command("cosign", "sign-blob", "-y",
		"--key", keyFile,
		"--output-signature", "signature-art3.sig",
		"payload-art3.json", // Sign the payload file, not the original artifact
	)

	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign sign-blob failed for payload: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("Cosign sign-blob output: %s\n", string(output))

	// Attach the signature to the artifact
	cmd = exec.Command("cosign", "attach", "signature",
		"--signature", "signature-art3.sig",
		"--payload", "payload-art3.json",
		artifactRef,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")

	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign attach signature failed for %s: %v\nOutput: %s", artifactName, err, string(output))
	}

	fmt.Printf("Cosign attach signature output: %s\n", string(output))

	// Remove the payload and signature files
	_ = os.Remove("payload-art3.json")
	_ = os.Remove("signature-art3.sig")

	return nil
}

func signArtifact4(artifactName string, client *remote.Repository) error {
	// Push the artifact to the registry first
	err := pushArtifact(context.Background(), client, artifactName)
	if err != nil {
		return fmt.Errorf("failed to push artifact: %v\n", err)
	}

	// Get reference to the artifact
	baseName := filepath.Base(artifactName)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	artifactRef := fmt.Sprintf("%s/%s:%s", registry, repo, baseName)

	// Get digest of the artifact from the registry
	digest, err := getArtifactDigest(context.Background(), client, baseName)
	if err != nil {
		return fmt.Errorf("failed to get artifact digest: %v", err)
	}

	// Create the payload that both signing methods will use
	payload := fmt.Sprintf(`{"critical":{"identity":{"docker-reference":"%s/%s"},"image":{"docker-manifest-digest":"%s"},"type":"cosign container image signature"},"optional":null}`, registry, repo, digest)
	err = os.WriteFile("payload-art4.json", []byte(payload), 0644)
	if err != nil {
		return fmt.Errorf("failed to write payload: %v\n", err)
	}

	// Sign the payload using sign-blob (this will now match cosign sign)
	cmd := exec.Command("cosign", "sign-blob", "-y",
		"--fulcio-url", "https://fulcio.sigstage.dev",
		"--rekor-url", "https://rekor.sigstage.dev",
		"--timestamp-server-url", "https://timestamp.sigstage.dev/api/v1/timestamp",
		"--oidc-client-id", "sigstore",
		"--oidc-issuer", "https://oauth2.sigstage.dev/auth",
		"--insecure-skip-verify", // Skip SCT verification for staging environment
		"--output-signature", "signature-art4.sig",
		"--output-certificate", "certificate-art4.crt",
		"--bundle", "bundle-art4.json",
		"--new-bundle-format",
		"payload-art4.json", // Sign the payload file, not the original artifact
	)

	// Set up interactive command execution to allow browser authentication
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")
	cmd.Stdin = os.Stdin   // Allow interactive input
	cmd.Stdout = os.Stdout // Allow output to be displayed to user
	cmd.Stderr = os.Stderr // Allow error output to be displayed to user

	err = cmd.Run() // Use Run() instead of CombinedOutput() for interactive execution
	if err != nil {
		return fmt.Errorf("cosign sign-blob failed for payload: %v", err)
	}

	fmt.Printf("Cosign sign-blob completed successfully\n")

	// Extract public key from certificate and upload to zot
	err = uploadPublicKeyFromCertificateToZot("certificate-art4.crt")
	if err != nil {
		fmt.Printf("failed to upload certificate: %v\n", err)
		return err
	}

	fmt.Printf("Public key extracted from certificate and uploaded to zot successfully\n")

	// Attach the signature to the artifact using the new bundle format
	cmd = exec.Command("cosign", "attach", "signature",
		"--signature", "signature-art4.sig",
		"--payload", "payload-art4.json",
		"--rekor-response", "bundle-art4.json",
		artifactRef,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD=")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign attach signature failed for %s: %v\nOutput: %s", artifactName, err, string(output))
	}

	fmt.Printf("Cosign attach signature output: %s\n", string(output))

	// Remove the payload and signature files
	_ = os.Remove("payload-art4.json")
	_ = os.Remove("signature-art4.sig")
	_ = os.Remove("certificate-art4.crt")
	_ = os.Remove("bundle-art4.json")

	return nil
}

// getArtifactDigest retrieves the digest of an artifact from the OCI registry
func getArtifactDigest(ctx context.Context, repo *remote.Repository, tag string) (string, error) {
	// Resolve the tag to get the manifest descriptor
	desc, err := repo.Resolve(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("failed to resolve tag %s: %w", tag, err)
	}

	return desc.Digest.String(), nil
}

// GraphQL query structures for signature verification
type GraphQLImageQuery struct {
	Query string `json:"query"`
}

type GraphQLImageResponse struct {
	Data struct {
		Image struct {
			Digest        string `json:"Digest"`
			IsSigned      bool   `json:"IsSigned"`
			Tag           string `json:"Tag"`
			SignatureInfo []struct {
				Tool      string `json:"Tool"`
				IsTrusted bool   `json:"IsTrusted"`
				Author    string `json:"Author"`
			} `json:"SignatureInfo"`
		} `json:"Image"`
	} `json:"data"`
}

// uploadPublicKeyToZot uploads the public key to zot's cosign endpoint
func uploadPublicKeyToZot() error {
	// Upload public key to zot's cosign endpoint for verification
	cosignEndpoint := fmt.Sprintf("http://%s/v2/_zot/ext/cosign", registry)

	// Get the absolute path to the public key file
	pubKeyPath, err := filepath.Abs(pubKeyFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for public key: %w", err)
	}

	// Execute the curl command with absolute path
	cmd := exec.Command("curl", "--data-binary", "@"+pubKeyPath, "-X", "POST", cosignEndpoint)

	// Set the working directory to ensure we're in the right location
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
		fmt.Printf("Working directory: %s\n", wd)
	}

	// Debug: Display the exact command being executed
	fmt.Printf("Executing command: %s\n", strings.Join(cmd.Args, " "))
	fmt.Printf("Public key file path: %s\n", pubKeyPath)
	fmt.Printf("Endpoint: %s\n", cosignEndpoint)

	// Check if the file exists
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("public key file does not exist at path: %s", pubKeyPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to upload public key: %w, output: %s", err, string(output))
	}
	fmt.Printf("Curl command output: %s\n", string(output))

	return nil
}

// extractPublicKeyFromCertificateFile extracts the public key from a base64-encoded certificate file
func extractPublicKeyFromCertificateFile(certificateFile string) (string, error) {
	// Read the base64-encoded certificate file
	certData, err := os.ReadFile(certificateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Clean up the base64 data - remove URL encoding and whitespace
	certDataStr := strings.TrimSpace(string(certData))
	certDataStr = strings.TrimSuffix(certDataStr, "%")      // Remove URL encoding artifacts at the end
	certDataStr = strings.ReplaceAll(certDataStr, "\n", "") // Remove any newlines
	certDataStr = strings.ReplaceAll(certDataStr, "\r", "") // Remove any carriage returns

	// Decode the base64 certificate (this gives us PEM data)
	pemBytes, err := base64.StdEncoding.DecodeString(certDataStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 certificate: %w", err)
	}

	// Parse the PEM-encoded certificate
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("failed to decode PEM certificate")
	}

	// Parse the X.509 certificate from the PEM block
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse X.509 certificate: %w", err)
	}

	// Extract the public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Encode the public key as PEM
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

func uploadPublicKeyFromCertificateToZot(certificateFile string) error {
	// Extract public key from certificate and upload to zot
	cosignEndpoint := fmt.Sprintf("http://%s/v2/_zot/ext/cosign", registry)

	// Extract public key from the certificate programmatically
	publicKeyPEM, err := extractPublicKeyFromCertificateFile(certificateFile)
	if err != nil {
		return fmt.Errorf("failed to extract public key from certificate: %w", err)
	}

	fmt.Printf("Extracted public key:\n%s\n", publicKeyPEM)

	// Create HTTP request with public key as body
	req, err := http.NewRequest(http.MethodPost, cosignEndpoint, strings.NewReader(publicKeyPEM))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	// Create HTTP client and execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload public key: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload public key, status: %d, response: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully uploaded public key to zot, response: %s\n", string(body))

	return nil
}

// verifySignatureWithZot verifies the signature using zot's GraphQL API
func verifySignatureWithZot(artifactName string) error {
	// GraphQL query to get detailed signature information
	// Try different repository/image reference formats that zot might expect
	query := fmt.Sprintf(`{
			Image(image: "%s:%s") {
				Digest
				IsSigned
				Tag
				SignatureInfo {
					Tool
					IsTrusted
					Author
				}
			}
		}`, repo, artifactName)

	graphqlQuery := GraphQLImageQuery{
		Query: query,
	}
	jsonData, err := json.Marshal(graphqlQuery)
	if err != nil {
		fmt.Printf("Failed to marshal GraphQL query: %v\n", err)
	}

	endpoint := fmt.Sprintf("http://%s/v2/_zot/ext/search", registry) // Zot search extension
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("  Response body: %s\n", string(body))

		var graphqlResp GraphQLImageResponse
		if err := json.Unmarshal(body, &graphqlResp); err != nil {
			fmt.Printf("  Failed to decode response: %v\n", err)
		}

		// Check if we got valid data (no errors)
		if graphqlResp.Data.Image.Digest != "" || graphqlResp.Data.Image.IsSigned {
			// Display the signature information
			image := graphqlResp.Data.Image

			if image.IsSigned {
				fmt.Printf("✅ Artifact %s is verified as signed\n", artifactName)
			} else {
				fmt.Printf("⚠️ Artifact %s appears to not be signed\n", artifactName)
			}

			if len(image.SignatureInfo) > 0 {
				if image.SignatureInfo[0].IsTrusted {
					fmt.Printf("✅ Artifact %s is verified as trusted\n", artifactName)
				} else {
					fmt.Printf("⚠️ Artifact %s appears to not be trusted\n", artifactName)
				}
			}

			return nil
		} else {
			fmt.Printf("Query returned empty data, trying next format...\n")
		}
	} else {
		fmt.Printf("GraphQL request failed with status %d: %s\n", resp.StatusCode, string(body))
	}

	return nil
}

func pushArtifact(ctx context.Context, repo *remote.Repository, artifactFile string) error {
	// Create sample content for the artifact
	content, err := os.ReadFile(artifactFile)
	if err != nil {
		return fmt.Errorf("failed to read artifact file: %w", err)
	}
	contentBytes := []byte(content)

	artifactName := filepath.Base(artifactFile)
	artifactName = strings.TrimSuffix(artifactName, filepath.Ext(artifactName))

	// Create blob descriptor
	blobDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    ocidigest.FromBytes(contentBytes),
		Size:      int64(len(contentBytes)),
	}

	// Push the blob with retry logic
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := repo.Push(ctx, blobDesc, strings.NewReader(string(contentBytes)))
		if err == nil {
			break
		}

		if i == maxRetries-1 {
			return fmt.Errorf("failed to push blob after %d attempts: %v", maxRetries, err)
		}

		fmt.Printf("Push attempt %d failed, retrying... (%v)\n", i+1, err)
		time.Sleep(time.Second * time.Duration(i+1))
	}

	// Create manifest with annotations
	annotations := map[string]string{
		"org.opencontainers.artifact.created":     "2024-01-01T00:00:00Z",
		"org.opencontainers.artifact.description": "Sample artifact: " + artifactName,
	}

	// Pack manifest
	manifestDesc, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: annotations,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		fmt.Printf("Failed to pack manifest: %v\n", err)

		return err
	}

	// Tag the manifest
	_, err = oras.Tag(ctx, repo, manifestDesc.Digest.String(), artifactName)
	if err != nil {
		fmt.Printf("Failed to tag manifest: %v\n", err)

		return err
	}

	fmt.Printf("Pushed artifact: %s\n", artifactName)

	return nil
}

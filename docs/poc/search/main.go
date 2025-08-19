package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	registry        = "localhost:5000"
	repo            = "other"
	artifactsToPush = 100
)

func main() {
	syncOCI()
}

func syncOCI() {
	// do other stuff
	ctx := context.Background()

	// create client to registry
	client, err := remote.NewRepository(fmt.Sprintf("%s/%s", registry, repo))
	if err != nil {
		fmt.Printf("failed to connect to registry: %v\n", err)
		return
	}
	client.PlainHTTP = true

	// push artifacts to source registry
	for artifactID := 0; artifactID < artifactsToPush; artifactID++ {
		err = pushArtifact(ctx, client, fmt.Sprintf("artifact-%d", artifactID))
		if err != nil {
			fmt.Printf("failed to push %d: %v\n", artifactID, err)
			return
		}
	}

	// run query
	// if err := queryWithGQL("artifact1"); err != nil {
	// 	fmt.Printf("failed to query with GQL: %v\n", err)
	// 	return
	// }
}

// GraphQL query structures for signature verification
func getQuery(repo, artifactName string) string {
	return fmt.Sprintf(`
{
  GlobalSearch(query: "%s:%s") {
    Page {
      ItemCount
      TotalCount
    }
    Images {
      RepoName
      Tag
      LastUpdated
      Manifests {
        Digest
        Layers {
          Size
          Digest
        }
      }
    }
  }
}
`, repo, artifactName)
}

// queryWithGQL runs a GraphQL API
func queryWithGQL(artifactName string) error {
	type GQL struct {
		Query string `json:"query"`
	}
	jsquery, err := json.Marshal(GQL{
		Query: getQuery(repo, artifactName),
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("http://%s/v2/_zot/ext/search", registry) // Zot search extension
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(jsquery))
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL request failed with status %d: %s\n", resp.StatusCode, string(body))
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %v\n", err)
	}

	// write to a json file
	file, err := os.Create(fmt.Sprintf("%s.json", artifactName))
	if err != nil {
		return fmt.Errorf("failed to create json file: %v\n", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to write json file: %v\n", err)
	}

	return nil
}

func pushArtifact(ctx context.Context, repo *remote.Repository, artifactID string) error {
	// Create sample content for the artifact
	contentBytes := []byte(fmt.Sprintf(`{"id": "%s"}`, artifactID))

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
		"org.opencontainers.image.created":        "2024-01-01T00:00:00Z",
		"org.opencontainers.artifact.created":     "2024-01-01T00:00:00Z",
		"org.opencontainers.artifact.description": "Sample artifact: " + artifactID,
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
	_, err = oras.Tag(ctx, repo, manifestDesc.Digest.String(), artifactID)
	if err != nil {
		fmt.Printf("Failed to tag manifest: %v\n", err)

		return err
	}

	fmt.Printf("Pushed artifact: %s\n", artifactID)

	return nil
}

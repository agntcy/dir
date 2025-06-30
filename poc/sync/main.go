package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/opencontainers/go-digest"
	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	// Used for dir-specific annotations.
	manifestDirObjectKeyPrefix = "org.agntcy.dir"
	manifestDirObjectTypeKey   = manifestDirObjectKeyPrefix + "/type"
)

//go:embed agent1.json
var agent1JSON []byte

//go:embed agent2.json
var agent2JSON []byte

//go:embed agent3.json
var agent3JSON []byte

func main() {
	ctx := context.Background()

	// Configure two local registries
	registry1Config := ociconfig.Config{
		RegistryAddress: "localhost:5000",
		RepositoryName:  "test-repo",
		AuthConfig: ociconfig.AuthConfig{
			Insecure: true, // Allow insecure connections for local testing
		},
	}

	registry2Config := ociconfig.Config{
		RegistryAddress: "localhost:5001",
		RepositoryName:  "test-repo",
		AuthConfig: ociconfig.AuthConfig{
			Insecure: true, // Allow insecure connections for local testing
		},
	}

	fmt.Println("Starting OCI Sync PoC")
	fmt.Println("Registry 1 (source):", "address", registry1Config.RegistryAddress)
	fmt.Println("Registry 2 (target):", "address", registry2Config.RegistryAddress)

	repo1, err := createRemoteStore(registry1Config)
	if err != nil {
		log.Fatalf("Failed to create remote store for registry 1: %v", err)
	}

	repo2, err := createRemoteStore(registry2Config)
	if err != nil {
		log.Fatalf("Failed to create remote store for registry 2: %v", err)
	}

	// Step 2: Push sample objects to registry 1
	fmt.Println("Pushing sample objects to registry 1...")

	// Sample object 1
	object1Data := []byte(agent1JSON)
	object1Ref := &coretypes.ObjectRef{
		Digest:      digest.FromBytes(object1Data).String(),
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Size:        uint64(len(object1Data)),
		Annotations: map[string]string{"name": "test-object-1", "description": "First test object for sync", "version": "1.0"},
	}

	_, err = push(ctx, repo1, object1Ref, bytes.NewReader(object1Data))
	if err != nil {
		log.Fatalf("Failed to push object 1 to registry 1: %v", err)
	}
	fmt.Println("Successfully pushed object 1", "ref", object1Ref.Digest)

	// Sample object 2
	object2Data := []byte(agent2JSON)
	object2Ref := &coretypes.ObjectRef{
		Digest:      digest.FromBytes(object2Data).String(),
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Size:        uint64(len(object2Data)),
		Annotations: map[string]string{"name": "test-object-2", "description": "Second test object for sync", "version": "1.0"},
	}

	_, err = push(ctx, repo1, object2Ref, bytes.NewReader(object2Data))
	if err != nil {
		log.Fatalf("Failed to push object 2 to registry 1: %v", err)
	}
	fmt.Println("Successfully pushed object 2", "ref", object2Ref.Digest)

	// Step 3: Verify objects exist in registry 1
	fmt.Println("Verifying objects in registry 1...")

	lookupRef1, err := lookup(ctx, repo1, object1Ref)
	if err != nil {
		log.Fatalf("Failed to lookup object 1 in registry 1: %v", err)
	}
	fmt.Println("Found object 1 in registry 1", "type", lookupRef1.Type, "size", lookupRef1.Size)

	lookupRef2, err := lookup(ctx, repo1, object2Ref)
	if err != nil {
		log.Fatalf("Failed to lookup object 2 in registry 1: %v", err)
	}
	fmt.Println("Found object 2 in registry 1", "type", lookupRef2.Type, "size", lookupRef2.Size)

	// Step 4: Verify objects don't exist in registry 2 (before sync)
	fmt.Println("Checking registry 2 before sync...")

	_, err = lookup(ctx, repo2, object1Ref)
	if err == nil {
		fmt.Println("Object 1 already exists in registry 2 (unexpected)")
	} else {
		fmt.Println("Object 1 not found in registry 2 (expected before sync)")
	}

	_, err = lookup(ctx, repo2, object2Ref)
	if err == nil {
		fmt.Println("Object 2 already exists in registry 2 (unexpected)")
	} else {
		fmt.Println("Object 2 not found in registry 2 (expected before sync)")
	}

	// Step 5: Perform sync from registry 1 to registry 2
	fmt.Println("Starting sync from registry 1 to registry 2...")
	err = sync(ctx, repo1, repo2)
	if err != nil {
		log.Fatalf("Failed to sync from registry 1 to registry 2: %v", err)
	}
	fmt.Println("Sync completed successfully")

	// Step 6: Verify objects now exist in registry 2 (after sync)
	fmt.Println("Verifying objects in registry 2 after sync...")

	syncedRef1, err := lookup(ctx, repo2, object1Ref)
	if err != nil {
		log.Fatalf("Failed to find object 1 in registry 2 after sync: %v", err)
	}
	fmt.Println("Found synced object 1 in registry 2", "type", syncedRef1.Type, "size", syncedRef1.Size)

	syncedRef2, err := lookup(ctx, repo2, object2Ref)
	if err != nil {
		log.Fatalf("Failed to find object 2 in registry 2 after sync: %v", err)
	}
	fmt.Println("Found synced object 2 in registry 2", "type", syncedRef2.Type, "size", syncedRef2.Size)

	// Step 7: Verify content integrity by pulling and comparing
	fmt.Println("Verifying content integrity...")

	// Pull object 1 from registry 2 and verify content
	reader1, err := pull(ctx, repo2, object1Ref)
	if err != nil {
		log.Fatalf("Failed to pull object 1 from registry 2: %v", err)
	}
	defer reader1.Close()

	content1, err := io.ReadAll(reader1)
	if err != nil {
		log.Fatalf("Failed to read object 1 content: %v", err)
	}

	if string(content1) != string(object1Data) {
		log.Fatalf("Object 1 content mismatch!\nExpected: %s\nGot: %s", object1Data, string(content1))
	}
	fmt.Println("Object 1 content verified successfully")

	// Pull object 2 from registry 2 and verify content
	reader2, err := pull(ctx, repo2, object2Ref)
	if err != nil {
		log.Fatalf("Failed to pull object 2 from registry 2: %v", err)
	}
	defer reader2.Close()

	content2, err := io.ReadAll(reader2)
	if err != nil {
		log.Fatalf("Failed to read object 2 content: %v", err)
	}

	if string(content2) != string(object2Data) {
		log.Fatalf("Object 2 content mismatch!\nExpected: %s\nGot: %s", object2Data, string(content2))
	}
	fmt.Println("Object 2 content verified successfully")

	// Step 8: Test sync idempotency (running sync again should be safe)
	fmt.Println("Testing sync idempotency...")

	err = sync(ctx, repo1, repo2)
	if err != nil {
		log.Fatalf("Failed second sync (idempotency test): %v", err)
	}
	fmt.Println("Second sync completed successfully (idempotency confirmed)")

	// Sync object 1 again to check what happens
	fmt.Println("HERE! Syncing object 1 again to check what happens...")
	if err := syncObject(ctx, repo1, repo2, object1Ref.GetShortRef()); err != nil {
		log.Fatalf("Failed to sync object 1 again: %v", err)
	}
	fmt.Println("Object 1 synced again successfully")

	// Step 9: Start a continuous sync pipe
	fmt.Println("Starting continuous sync pipe...")
	go func() {
		if err := syncPipe(ctx, repo1, repo2, 10*time.Second); err != nil {
			log.Printf("Sync pipe error: %v", err)
		}
	}()

	// Push a new object to registry 1 after starting the sync pipe
	object3Data := []byte(agent3JSON)
	object3Ref := &coretypes.ObjectRef{
		Digest:      digest.FromBytes(object3Data).String(),
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Size:        uint64(len(object3Data)),
		Annotations: map[string]string{"name": "test-object-2", "description": "Second test object for sync", "version": "1.0"},
	}

	_, err = push(ctx, repo1, object3Ref, bytes.NewReader(object3Data))
	if err != nil {
		log.Fatalf("Failed to push new object to registry 1: %v", err)
	}
	fmt.Println("Successfully pushed new object 3", "ref", object3Ref.Digest)

	// Wait for a while to allow the sync pipe to process the new object
	time.Sleep(15 * time.Second)

	// Verify the new object is now in registry 2
	syncedNewRef, err := lookup(ctx, repo2, object3Ref)
	if err != nil {
		log.Fatalf("Failed to find new object in registry 2 after sync: %v", err)
	}
	fmt.Println("Found synced new object in registry 2", "type", syncedNewRef.Type)

	// TODO check if remote changed by computing hash of all tags
}

// syncPipe creates a continuous sync pipe between two repositories
func syncPipe(ctx context.Context, sourceRepo, destRepo *remote.Repository, interval time.Duration) error {
	fmt.Println("Starting sync pipe with interval:", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track last known tags to detect changes
	var lastKnownTags []string

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Sync pipe stopped due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			if err := syncWithChangeDetection(ctx, sourceRepo, destRepo, &lastKnownTags); err != nil {
				fmt.Printf("Sync pipe error: %v\n", err)
			}
		}
	}
}

// syncWithChangeDetection only syncs when changes are detected
func syncWithChangeDetection(ctx context.Context, sourceRepo, destRepo *remote.Repository, lastKnownTags *[]string) error {
	// Get current tags from source
	currentTags, err := listRemoteTags(ctx, sourceRepo)
	if err != nil {
		return fmt.Errorf("failed to list current tags: %w", err)
	}

	// Detect new or changed tags
	newTags := detectNewTags(*lastKnownTags, currentTags)
	if len(newTags) == 0 {
		fmt.Println("No new changes detected")
		return nil
	}

	fmt.Printf("Detected %d new/changed tags: %v\n", len(newTags), newTags)

	// Sync only the new/changed objects
	var syncedCount int
	for _, tag := range newTags {
		if err := syncObject(ctx, sourceRepo, destRepo, tag); err != nil {
			fmt.Printf("Failed to sync new object tag=%s error=%v\n", tag, err)
			continue
		}
		syncedCount++
	}

	// Update last known tags
	*lastKnownTags = currentTags

	fmt.Printf("Sync pipe iteration completed: synced=%d new=%d\n", syncedCount, len(newTags))
	return nil
}

// detectNewTags finds tags that are new or have changed
func detectNewTags(oldTags, newTags []string) []string {
	oldTagsMap := make(map[string]bool)
	for _, tag := range oldTags {
		oldTagsMap[tag] = true
	}

	var changedTags []string
	for _, tag := range newTags {
		if !oldTagsMap[tag] {
			changedTags = append(changedTags, tag)
		}
	}

	return changedTags
}

func push(ctx context.Context, repo *remote.Repository, ref *coretypes.ObjectRef, contents io.Reader) (any, error) {
	// push raw data
	blobRef, blobDesc, err := pushData(ctx, repo, ref, contents)
	if err != nil {
		return nil, fmt.Errorf("failed to push data: %w", err)
	}

	// set annotations for manifest
	annotations := cleanMeta(ref.GetAnnotations())
	annotations[manifestDirObjectTypeKey] = ref.GetType()

	// push manifest
	manifestDesc, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: annotations,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// tag manifest
	// tag => resolves manifest to object which can be looked up (lookup)
	// tag => allows to pull object directly (pull)
	// tag => allows listing and filtering tags (list)
	_, err = oras.Tag(ctx, repo, manifestDesc.Digest.String(), ref.GetShortRef())
	if err != nil {
		return nil, err
	}

	return &coretypes.ObjectRef{
		Digest:      blobRef.GetDigest(),
		Type:        ref.GetType(),
		Size:        ref.GetSize(),
		Annotations: cleanMeta(ref.GetAnnotations()),
	}, nil
}

func pull(ctx context.Context, repo *remote.Repository, ref *coretypes.ObjectRef) (io.ReadCloser, error) {
	return repo.Fetch(ctx, ocispec.Descriptor{ //nolint:wrapcheck
		Digest: ocidigest.Digest(ref.GetDigest()),
		Size:   int64(ref.GetSize()), //nolint:gosec
	})
}

func lookup(ctx context.Context, repo *remote.Repository, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	// check if blob exists on remote
	{
		exists, err := repo.Exists(ctx, ocispec.Descriptor{Digest: ocidigest.Digest(ref.GetDigest())})
		if err != nil {
			return nil, fmt.Errorf("failed to check if object exists: %w", err)
		}

		if !exists {
			return nil, fmt.Errorf("object not found in OCI store: %s", ref.GetDigest())
		}
	}

	// read manifest data from remote
	var manifest ocispec.Manifest
	{
		shortRef := ref.GetShortRef()

		// resolve manifest from remote tag
		manifestDesc, err := repo.Resolve(ctx, shortRef)
		if err != nil {
			fmt.Println("Failed to resolve manifest", "error", err)

			// do not error here, as we may have a raw object stored but not tagged with
			// the manifest. only agents are tagged with manifests
			return ref, nil
		}

		// TODO: validate manifest by size

		// fetch manifest from remote
		manifestRd, err := repo.Fetch(ctx, manifestDesc)
		if err != nil {
			return ref, fmt.Errorf("failed to fetch manifest: %w", err)
		}

		// read manifest
		manifestData, err := io.ReadAll(manifestRd)
		if err != nil {
			return nil, fmt.Errorf("failed to read manifest data: %w", err)
		}

		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
		}
	}

	// read object size from manifest
	var objectSize uint64
	if len(manifest.Layers) > 0 {
		objectSize = uint64(manifest.Layers[0].Size) //nolint:gosec
	}

	// read object type from manifest metadata
	objectType, ok := manifest.Annotations[manifestDirObjectTypeKey]
	if !ok {
		return nil, fmt.Errorf("object type not found in manifest annotations: %s", manifestDirObjectTypeKey)
	}

	// return clean ref
	return &coretypes.ObjectRef{
		Digest:      ref.GetDigest(),
		Type:        objectType,
		Size:        objectSize,
		Annotations: cleanMeta(manifest.Annotations),
	}, nil
}

// cleanMeta returns metadata without OCI- or Dir- annotations.
func cleanMeta(meta map[string]string) map[string]string {
	if meta == nil {
		return map[string]string{}
	}

	// delete all OCI-specific metadata
	delete(meta, "org.opencontainers.image.created")

	// delete all Dir-specific metadata
	delete(meta, manifestDirObjectTypeKey)
	// TODO: clean all with dir prefix

	return meta
}

// pushData pushes raw data to OCI.
func pushData(ctx context.Context, repo *remote.Repository, ref *coretypes.ObjectRef, rd io.Reader) (*coretypes.ObjectRef, ocispec.Descriptor, error) {
	// push blob
	blobDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    ocidigest.Digest(ref.GetDigest()),
		Size:      int64(ref.GetSize()),
	}

	fmt.Println("Pushing blob to OCI store", "ref", ref, "blobDesc", blobDesc)

	err := repo.Push(ctx, blobDesc, rd)
	if err != nil {
		// return only for non-valid errors
		if !strings.Contains(err.Error(), "already exists") {
			return nil, ocispec.Descriptor{}, fmt.Errorf("failed to push blob: %w", err)
		}
	}

	// return ref
	return &coretypes.ObjectRef{
		Digest: ref.GetDigest(),
		Type:   ref.GetType(),
		Size:   uint64(blobDesc.Size),
	}, blobDesc, nil
}

// createRemoteStore creates a remote repository client from config
func createRemoteStore(cfg ociconfig.Config) (*remote.Repository, error) {
	if cfg.RegistryAddress == "" || cfg.RepositoryName == "" {
		return nil, fmt.Errorf("remote registry address and repository name are required")
	}

	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", cfg.RegistryAddress, cfg.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// Configure client
	repo.PlainHTTP = cfg.Insecure
	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-sync-client"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			cfg.RegistryAddress,
			auth.Credential{
				Username:     cfg.Username,
				Password:     cfg.Password,
				RefreshToken: cfg.RefreshToken,
				AccessToken:  cfg.AccessToken,
			},
		),
	}

	return repo, nil
}

func sync(ctx context.Context, repo1, repo2 *remote.Repository) error {
	// List all tags from remote repository 1
	tags, err := listRemoteTags(ctx, repo1)
	if err != nil {
		return fmt.Errorf("failed to list tags from registry 1: %w", err)
	}

	fmt.Println("Found tags in registry 1:", tags)

	// Sync each tagged object
	var syncedCount int
	for _, tag := range tags {
		if err := syncObject(ctx, repo1, repo2, tag); err != nil {
			fmt.Println("Failed to sync object", "tag", tag, "error", err)
			continue // Continue with other objects
		}
		syncedCount++
	}

	fmt.Println("Sync completed", "synced", syncedCount, "total", len(tags))
	return nil
}

// listRemoteTags lists all available tags from the remote repository
func listRemoteTags(ctx context.Context, remoteRepo *remote.Repository) ([]string, error) {
	var tags []string

	// Use ORAS to list tags from the remote repository
	err := remoteRepo.Tags(ctx, "", func(tagsPage []string) error {
		tags = append(tags, tagsPage...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// syncObject syncs a single object from remote to local store
func syncObject(ctx context.Context, repo1, repo2 *remote.Repository, tag string) error {
	fmt.Println("Syncing object", "tag", tag)

	// Resolve manifest from repo1 to get object metadata
	manifestDesc, err := repo1.Resolve(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to resolve manifest for tag %s: %w", tag, err)
	}

	// Fetch and parse manifest to get object reference information
	manifestReader, err := repo1.Fetch(ctx, manifestDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer manifestReader.Close()

	manifestData, err := io.ReadAll(manifestReader)
	if err != nil {
		return fmt.Errorf("failed to read manifest data: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// Extract object information from manifest
	if len(manifest.Layers) == 0 {
		return fmt.Errorf("manifest has no layers for tag %s", tag)
	}

	layer := manifest.Layers[0] // Assuming single layer per object
	objectType, ok := manifest.Annotations[manifestDirObjectTypeKey]
	if !ok {
		return fmt.Errorf("object type not found in manifest annotations")
	}

	// Create ObjectRef for the object
	objectRef := &coretypes.ObjectRef{
		Digest:      layer.Digest.String(),
		Type:        objectType,
		Size:        uint64(layer.Size),
		Annotations: cleanMeta(manifest.Annotations),
	}

	// Check if object already exists in repo2
	exists, err := repo2.Exists(ctx, ocispec.Descriptor{
		Digest: ocidigest.Digest(objectRef.GetDigest()),
		Size:   int64(objectRef.GetSize()),
	})
	if err != nil {
		return fmt.Errorf("failed to check if object exists in destination: %w", err)
	}

	if exists {
		fmt.Println("Object already exists in destination, skipping", "tag", tag, "digest", objectRef.GetDigest())
		return nil
	}

	// Pull object data from repo1 using the pull function
	fmt.Println("Pulling object from source registry", "digest", objectRef.GetDigest())
	objectReader, err := pull(ctx, repo1, objectRef)
	if err != nil {
		return fmt.Errorf("failed to pull object from source: %w", err)
	}
	defer objectReader.Close()

	// Push object data to repo2 using the push function
	fmt.Println("Pushing object to destination registry", "digest", objectRef.GetDigest())
	_, err = push(ctx, repo2, objectRef, objectReader)
	if err != nil {
		return fmt.Errorf("failed to push object to destination: %w", err)
	}

	fmt.Println("Successfully synced object", "tag", tag, "digest", objectRef.GetDigest())
	return nil
}

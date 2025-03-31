package secretstore

import (
	"testing"

	"github.com/agntcy/dir/cli/util/dir"
)

func TestFileSecretStore_SaveHubSecret(t *testing.T) {
	secretStore := NewFileSecretStore(dir.GetAppDir() + "/secrets.json")

	if err := secretStore.SaveHubSecret("http://localhost:8080", &HubSecret{
		BackendUrl: "http://localhost:8080",
		TokenSecret: &TokenSecret{
			IdpToken:     "idpToken",
			RefreshToken: "RefreshToken",
			AuthToken:    "authToken",
		},
	}); err != nil {
		t.Fatalf("failed to save hub secret: %v", err)
	}
}

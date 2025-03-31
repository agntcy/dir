package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/docker/builder/remotecontext/urlutil"
)

var (
	ErrInvalidFrontendURL = fmt.Errorf("invalid frontend URL")
	ErrFetchingConfig     = fmt.Errorf("error fetching config")
	ErrorParsingConfig    = fmt.Errorf("error parsing config")
)

type AuthConfig struct {
	ClientId   string `json:"IAM_OIDC_CLIENT_ID"`
	IdpAddress string `json:"IAM_OIDC_ISSUER"`
}

func FetchAuthConfig(frontedUrl string) (*AuthConfig, error) {
	if !urlutil.IsURL(frontedUrl) {
		return nil, ErrInvalidFrontendURL
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/idp_config.json", frontedUrl), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %w: (%d) %s", ErrFetchingConfig, fmt.Errorf("unexpected status code"), resp.StatusCode, resp.Body)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetchingConfig, err)
	}

	var authConfig AuthConfig
	if err = json.Unmarshal(body, &authConfig); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrorParsingConfig, err)
	}

	return &authConfig, nil
}

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
	IdpProductId       string `json:"IAM_PRODUCT_ID"`
	ClientId           string `json:"IAM_OIDC_CLIENT_ID"`
	IdpIssuerAddress   string `json:"IAM_OIDC_ISSUER"`
	IdpBackendAddress  string `json:"IAM_API"`
	IdpFrontendAddress string `json:"IAM_UI"`
	HubBackendAddress  string `json:"HUB_API"`
}

func FetchAuthConfig(frontedUrl string) (*AuthConfig, error) {
	if !urlutil.IsURL(frontedUrl) {
		return nil, ErrInvalidFrontendURL
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/config.json", frontedUrl), nil)
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

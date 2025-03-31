package secretstore

import "github.com/docker/docker/builder/remotecontext/urlutil"

type HubSecrets struct {
	HubSecrets map[string]*HubSecret `json:"hubSecrets"`
}
type HubSecret struct {
	ClientId     string `json:"clientId"`
	IdpAddress   string `json:"idpAddress"`
	*TokenSecret `json:"tokens"`
}

type TokenSecret struct {
	IdpToken     string `json:"idpToken"`
	RefreshToken string `json:"refreshToken"`
	AuthToken    string `json:"authToken"`
}

func (h *HubSecret) Validate() error {

	if !urlutil.IsURL(h.IdpAddress) {
		return ErrInvalidSecret
	}

	if h.ClientId == "" {
		return ErrInvalidSecret
	}

	if h.TokenSecret == nil {
		return ErrInvalidSecret
	}

	if h.TokenSecret.IdpToken == "" {
		return ErrInvalidSecret
	}
	if h.TokenSecret.RefreshToken == "" {
		return ErrInvalidSecret
	}

	if h.TokenSecret.AuthToken == "" {
		return ErrInvalidSecret
	}

	return nil
}

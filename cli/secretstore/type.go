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
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
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

	if h.TokenSecret.IdToken == "" {
		return ErrInvalidSecret
	}
	if h.TokenSecret.RefreshToken == "" {
		return ErrInvalidSecret
	}

	if h.TokenSecret.AccessToken == "" {
		return ErrInvalidSecret
	}

	return nil
}

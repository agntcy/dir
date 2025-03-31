package webserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/agntcy/dir/cli/cmd/hub/login/webserver/utils"
)

const (
	failedLoginMessage     = "Failed to login."
	successfulLoginMessage = "Successfully logged in. You can close this tab."

	paramClientId            = "client_id"
	paramCodeChallenge       = "code_challenge"
	paramCodeChallengeMethod = "code_challenge_method"
	paramNonce               = "nonce"
	paramRedirectUri         = "redirect_uri"
	paramResponseType        = "response_type"
	paramState               = "state"
	paramScope               = "scope"
	paramGrantType           = "grant_type"
	paramCode                = "code"
	paramCodeVerifier        = "code_verifier"

	headerAccept       = "Accept"
	headerContentType  = "Content-Type"
	headerCacheControl = "Cache-Control"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

type SessionStore struct {
	verifier string
	Tokens   Tokens
}

type Config struct {
	ClientId           string
	FrontendUrl        string
	IdpUrl             string
	LocalWebserverPort int
	SessionStore       *SessionStore
	ErrChan            chan error
}

type Handler struct {
	clientId           string
	frontendUrl        string
	idpUrl             string
	localWebserverPort int
	localWebserverUrl  string

	sessionStore *SessionStore

	Err chan error
}

func NewHandler(config *Config) *Handler {
	var errChan chan error
	if config.ErrChan == nil {
		errChan = make(chan error, 1)
	} else {
		errChan = config.ErrChan
	}

	return &Handler{
		clientId:          config.ClientId,
		frontendUrl:       config.FrontendUrl,
		idpUrl:            config.IdpUrl,
		localWebserverUrl: fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort),

		sessionStore: config.SessionStore,

		Err: errChan,
	}
}

func (h *Handler) HandleRequestRedirect(w http.ResponseWriter, r *http.Request) {
	requestId := r.URL.Query().Get("request")

	var challenge string
	h.sessionStore.verifier, challenge = utils.GenerateChallenge()

	nonce, err := utils.GenerateNonce()
	if err != nil {
		h.handleError(w, err)
	}

	params := url.Values{}
	params.Add(paramClientId, h.clientId)
	params.Add(paramCodeChallenge, challenge)
	params.Add(paramCodeChallengeMethod, "S256")
	params.Add(paramNonce, nonce)
	params.Add(paramRedirectUri, h.localWebserverUrl)
	params.Add(paramResponseType, "code")
	params.Add(paramState, fmt.Sprintf(`{"sessionRequest":"%s"}`, requestId))
	params.Add(paramScope, "openid offline_access")

	redirectUrl := fmt.Sprintf("%s/v1/authorize?%s", h.idpUrl, params.Encode())
	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func (h *Handler) HandleCodeRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	// Exchange the authorization code for tokens
	data := url.Values{}
	data.Set(paramGrantType, "authorization_code")
	data.Set(paramClientId, h.clientId)
	data.Set(paramRedirectUri, h.localWebserverUrl)
	data.Set(paramCode, code)
	data.Set(paramCodeVerifier, h.sessionStore.verifier)

	tokenReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/token", h.idpUrl), strings.NewReader(data.Encode()))
	if err != nil {
		h.handleError(w, fmt.Errorf("failed to create token request: %w", err))
	}

	tokenReq.Header.Add(headerAccept, "application/json")
	tokenReq.Header.Add(headerContentType, "application/x-www-form-urlencoded")
	tokenReq.Header.Add(headerCacheControl, "no-cache")

	resp, err := http.DefaultClient.Do(tokenReq)
	if resp.StatusCode != http.StatusOK {
		h.handleError(w, fmt.Errorf("unexpected error code while getting tokens: %w", err))
	}
	if err != nil {
		h.handleError(w, fmt.Errorf("failed to send token request: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.handleError(w, fmt.Errorf("failed to read token response: %w", err))
	}

	var t Tokens
	err = json.Unmarshal(body, &t)
	if err != nil {
		h.handleError(w, fmt.Errorf("failed to unmarshal token response: %w", err))
	}

	h.sessionStore.Tokens = t

	h.handleSuccess(w)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(failedLoginMessage))
	h.Err <- err
}

func (h *Handler) handleSuccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successfulLoginMessage))
	h.Err <- nil
}

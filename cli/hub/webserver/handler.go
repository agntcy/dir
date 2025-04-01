package webserver

import (
	"fmt"
	"net/http"

	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/hub/webserver/utils"
)

const (
	failedLoginMessage     = "Failed to login."
	successfulLoginMessage = "Successfully logged in. You can close this tab."
)

type SessionStore struct {
	verifier string
	Tokens   *idp.Token
}

type Config struct {
	ClientId           string
	FrontendUrl        string
	IdpUrl             string
	LocalWebserverPort int

	SessionStore *SessionStore
	IdpClient    idp.IdpClient
	ErrChan      chan error
}

type Handler struct {
	clientId           string
	frontendUrl        string
	idpUrl             string
	localWebserverPort int
	localWebserverUrl  string

	sessionStore *SessionStore
	idpClient    idp.IdpClient

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
		idpClient:    config.IdpClient,

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

	redirectUrl := h.idpClient.AuthorizeUrl(&idp.AuthorizeRequest{
		ClientId:      h.clientId,
		S256Challenge: challenge,
		Nonce:         nonce,
		RedirectUri:   h.localWebserverUrl,
		RequestId:     requestId,
	})

	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func (h *Handler) HandleCodeRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	resp, err := h.idpClient.RequestToken(&idp.RequestTokenRequest{
		ClientId:    h.clientId,
		RedirectUri: h.localWebserverUrl,
		Verifier:    h.sessionStore.verifier,
		Code:        code,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	if resp.Response.StatusCode != http.StatusOK {
		h.handleError(w, fmt.Errorf("unexpected status code: %d: %s", resp.Response.StatusCode, resp.Body))
		return
	}

	h.sessionStore.Tokens = resp.Token

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

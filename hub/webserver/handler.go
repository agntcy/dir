// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package webserver

import (
	"fmt"
	"net/http"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/webserver/utils"
)

const (
	failedLoginMessage     = "Failed to login."
	successfulLoginMessage = "Successfully logged in. You can close this tab."
)

type SessionStore struct {
	verifier string
	Tokens   *okta.Token
}

type Config struct {
	ClientID           string
	IdpFrontendURL     string
	IdpBackendURL      string
	LocalWebserverPort int

	SessionStore *SessionStore
	OktaClient   okta.Client
	ErrChan      chan error
}

type Handler struct {
	clientID          string
	frontendURL       string
	idpURL            string
	localWebserverURL string

	sessionStore *SessionStore
	idpClient    okta.Client

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
		clientID:          config.ClientID,
		frontendURL:       config.IdpFrontendURL,
		idpURL:            config.IdpBackendURL,
		localWebserverURL: fmt.Sprintf("http://localhost:%d", config.LocalWebserverPort),

		sessionStore: config.SessionStore,
		idpClient:    config.OktaClient,

		Err: errChan,
	}
}

func (h *Handler) HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK")) //nolint:errcheck
}

func (h *Handler) HandleRequestRedirect(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request")

	var challenge string
	h.sessionStore.verifier, challenge = utils.GenerateChallenge()

	nonce, err := utils.GenerateNonce()
	if err != nil {
		h.handleError(w, err)
	}

	redirectURL := h.idpClient.AuthorizeURL(&okta.AuthorizeRequest{
		ClientID:      h.clientID,
		S256Challenge: challenge,
		Nonce:         nonce,
		RedirectURI:   h.localWebserverURL,
		RequestID:     requestID,
	})

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) HandleCodeRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	resp, err := h.idpClient.RequestToken(&okta.RequestTokenRequest{
		ClientID:    h.clientID,
		RedirectURI: h.localWebserverURL,
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
	w.Write([]byte(failedLoginMessage)) //nolint:errcheck
	h.Err <- err
}

func (h *Handler) handleSuccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successfulLoginMessage)) //nolint:errcheck
	h.Err <- nil
}

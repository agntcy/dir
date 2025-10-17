// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package webserver

import (
	"fmt"
	"net/http"

	"github.com/agntcy/dir/hub/auth/internal/webserver/utils"
	"github.com/agntcy/dir/hub/client/okta"
)

const (
	failedLoginMessage     = "Failed to login."
	successfulLoginMessage = "Successfully logged in. You can close this tab."
)

// SessionStore holds the PKCE verifier and the resulting tokens from the OAuth flow.
type SessionStore struct {
	Verifier  string
	Challenge string
	Tokens    *okta.Token
}

// Config contains the configuration for the local webserver and OAuth handler.
type Config struct {
	ClientID           string
	IdpFrontendURL     string
	IdpBackendURL      string
	LocalWebserverPort int

	SessionStore *SessionStore
	OktaClient   okta.Client
	ErrChan      chan error
}

// Handler implements the HTTP handlers for the OAuth flow, including redirects and token exchange.
type Handler struct {
	clientID          string
	frontendURL       string
	idpURL            string
	localWebserverURL string

	sessionStore *SessionStore
	idpClient    okta.Client

	Err chan error
}

// NewHandler creates a new Handler with the given configuration.
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

// HandleHealthz responds to health check requests.
func (h *Handler) HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK")) //nolint:errcheck
}

// HandleRequestRedirect handles the initial OAuth redirect, generating a PKCE challenge and redirecting to the IdP.
func (h *Handler) HandleRequestRedirect(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request")

	var challenge string

	h.sessionStore.Verifier, challenge = utils.GenerateChallenge()

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

// HandleCodeRedirect handles the redirect from the IdP with the authorization code, exchanges it for tokens, and stores them.
func (h *Handler) HandleCodeRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	resp, err := h.idpClient.RequestToken(&okta.RequestTokenRequest{
		ClientID:    h.clientID,
		RedirectURI: h.localWebserverURL,
		Verifier:    h.sessionStore.Verifier,
		Code:        code,
	})
	if err != nil { //nolint:wsl
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

// handleError writes an error message to the response and sends the error to the error channel.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(failedLoginMessage)) //nolint:errcheck

	h.Err <- err
}

// handleSuccess writes a success message to the response and signals success on the error channel.
func (h *Handler) handleSuccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successfulLoginMessage)) //nolint:errcheck

	h.Err <- nil
}

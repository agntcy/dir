# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""OAuth 2.0 Authorization Code flow with PKCE and OIDC discovery (loopback callback)."""

from __future__ import annotations

import logging
import secrets
import threading
import webbrowser
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Any
from urllib.parse import parse_qs, urlencode, urlparse

import httpx
from authlib.oauth2.rfc7636 import create_s256_code_challenge

from agntcy.dir_sdk.client.config import Config

logger = logging.getLogger(__name__)


class OAuthPkceError(RuntimeError):
    """Raised when PKCE loopback login or token exchange fails."""


def normalize_issuer(issuer: str) -> str:
    u = issuer.rstrip("/")
    if not u.startswith("https://") and not u.startswith("http://"):
        msg = "oidc_issuer must be an absolute URL (https:// recommended)"
        raise ValueError(msg)
    return u


def fetch_openid_configuration(
    issuer: str,
    *,
    verify: bool = True,
    timeout: float = 30.0,
) -> dict[str, Any]:
    base = normalize_issuer(issuer)
    url = f"{base}/.well-known/openid-configuration"
    with httpx.Client(verify=verify, timeout=timeout) as client:
        response = client.get(url)
        response.raise_for_status()
        data = response.json()
    if "authorization_endpoint" not in data or "token_endpoint" not in data:
        msg = "OpenID configuration missing authorization_endpoint or token_endpoint"
        raise OAuthPkceError(msg)
    return data


def _form_post(
    url: str,
    body: dict[str, str],
    *,
    verify: bool = True,
    timeout: float = 30.0,
) -> dict[str, Any]:
    with httpx.Client(verify=verify, timeout=timeout) as client:
        response = client.post(
            url,
            data=body,
            headers={"Content-Type": "application/x-www-form-urlencoded"},
        )
        try:
            response.raise_for_status()
        except httpx.HTTPStatusError as e:
            detail = response.text[:500] if response.text else ""
            msg = f"Token HTTP {response.status_code}: {detail}"
            raise OAuthPkceError(msg) from e
        return response.json()


class OAuthTokenHolder:
    """Holds the active OAuth access token for gRPC bearer auth."""

    def __init__(self) -> None:
        self._lock = threading.RLock()
        self._access_token: str | None = None

    def set_tokens(self, access_token: str) -> None:
        with self._lock:
            self._access_token = access_token

    def update_from_token_response(self, payload: dict[str, Any]) -> None:
        access = payload.get("access_token")
        if not access or not isinstance(access, str):
            msg = "Token response missing access_token"
            raise OAuthPkceError(msg)
        self.set_tokens(access)

    def get_access_token(self) -> str:
        with self._lock:
            if self._access_token is None:
                msg = (
                    "No OAuth access token: set DIRECTORY_CLIENT_AUTH_TOKEN "
                    "or call Client.authenticate_oauth_pkce()"
                )
                raise RuntimeError(msg)
            return self._access_token  # type: ignore[return-value]


def exchange_authorization_code(
    token_endpoint: str,
    *,
    code: str,
    redirect_uri: str,
    client_id: str,
    code_verifier: str,
    client_secret: str = "",
    verify: bool = True,
    timeout: float = 30.0,
) -> dict[str, Any]:
    body: dict[str, str] = {
        "grant_type": "authorization_code",
        "code": code,
        "redirect_uri": redirect_uri,
        "client_id": client_id,
        "code_verifier": code_verifier,
    }
    if client_secret:
        body["client_secret"] = client_secret
    return _form_post(token_endpoint, body, verify=verify, timeout=timeout)


def run_loopback_pkce_login(
    config: Config,
    metadata: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Run browser authorization with PKCE; local HTTP server receives redirect.

    Args:
        config: Must include oidc_issuer, oidc_client_id, oidc_redirect_uri,
            oidc_callback_port (if redirect has no port), scopes, etc.
        metadata: Optional OpenID configuration dict (skips a second discovery GET).

    Returns:
        Raw token JSON from the token endpoint.

    """
    if not config.oidc_issuer:
        msg = "oidc_issuer is required for OAuth PKCE"
        raise ValueError(msg)
    if not config.oidc_client_id:
        msg = "oidc_client_id is required for OAuth PKCE"
        raise ValueError(msg)

    redirect_uri = config.oidc_redirect_uri.strip()
    parsed = urlparse(redirect_uri)
    if parsed.scheme not in ("http", "https") or not parsed.hostname:
        msg = "oidc_redirect_uri must be an absolute http(s) URL"
        raise ValueError(msg)
    if parsed.hostname not in ("localhost", "127.0.0.1"):
        msg = "loopback PKCE requires redirect host localhost or 127.0.0.1"
        raise ValueError(msg)
    path = parsed.path or "/"
    if not path.startswith("/"):
        path = "/" + path

    port = config.oidc_callback_port

    verify_http = not config.tls_skip_verify
    if metadata is None:
        metadata = fetch_openid_configuration(
            config.oidc_issuer,
            verify=verify_http,
            timeout=min(30.0, config.oidc_auth_timeout),
        )
    auth_ep = str(metadata["authorization_endpoint"])
    token_ep = str(metadata["token_endpoint"])

    code_verifier = secrets.token_urlsafe(64)
    code_challenge = create_s256_code_challenge(code_verifier)
    state = secrets.token_urlsafe(32)

    result: dict[str, Any] = {}
    error_holder: list[str] = []
    done = threading.Event()

    class _Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:  # noqa: N802
            try:
                req = urlparse(self.path)
                if req.path != path:
                    error_holder.append("redirect path does not match oidc_redirect_uri")
                    self.send_error(404, "Not Found")
                    return
                qs = parse_qs(req.query)
                if qs.get("error"):
                    err = qs.get("error", ["unknown"])[0]
                    desc = (qs.get("error_description") or [""])[0]
                    error_holder.append(f"{err}: {desc}")
                    self._ok_page("Authorization failed. You may close this window.")
                    done.set()
                    return
                if qs.get("state", [None])[0] != state:
                    error_holder.append("state mismatch")
                    self._ok_page("Invalid state. You may close this window.")
                    done.set()
                    return
                code = qs.get("code", [None])[0]
                if not code:
                    error_holder.append("missing code")
                    self._ok_page("Missing code. You may close this window.")
                    done.set()
                    return
                result["code"] = code
                self._ok_page("Login successful. You may close this window.")
            finally:
                done.set()

        def _ok_page(self, message: str) -> None:
            body = (
                "<!DOCTYPE html><html><body><p>"
                + message
                + "</p></body></html>"
            ).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(body)

    server: HTTPServer | None = None
    try:
        server = HTTPServer(("127.0.0.1", port), _Handler)
    except OSError as e:
        msg = f"Cannot bind loopback callback on 127.0.0.1:{port}: {e}"
        raise OAuthPkceError(msg) from e

    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()

    scope_str = " ".join(config.oidc_scopes) if config.oidc_scopes else "openid"
    auth_params = {
        "response_type": "code",
        "client_id": config.oidc_client_id,
        "redirect_uri": redirect_uri,
        "scope": scope_str,
        "state": state,
        "code_challenge": code_challenge,
        "code_challenge_method": "S256",
    }
    sep = "&" if "?" in auth_ep else "?"
    authorize_url = f"{auth_ep}{sep}{urlencode(auth_params)}"

    webbrowser.open(authorize_url)

    try:
        if not done.wait(timeout=config.oidc_auth_timeout):
            msg = f"OAuth callback timed out after {config.oidc_auth_timeout}s"
            raise OAuthPkceError(msg)

        if error_holder:
            msg = error_holder[0]
            raise OAuthPkceError(msg)

        code = result.get("code")
        if not code:
            msg = "Authorization did not return a code"
            raise OAuthPkceError(msg)

        return exchange_authorization_code(
            token_ep,
            code=code,
            redirect_uri=redirect_uri,
            client_id=config.oidc_client_id,
            code_verifier=code_verifier,
            client_secret=config.oidc_client_secret,
            verify=verify_http,
            timeout=min(30.0, config.oidc_auth_timeout),
        )
    finally:
        server.shutdown()
        thread.join(timeout=5.0)
        server.server_close()

# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import time
from unittest import mock

import httpx
import pytest

from agntcy.dir_sdk.client.config import Config
from agntcy.dir_sdk.client.oauth_pkce import (
    OAuthPkceError,
    OAuthTokenHolder,
    exchange_authorization_code,
    fetch_openid_configuration,
    normalize_issuer,
    run_loopback_pkce_login,
)


def test_normalize_issuer_strips_slash() -> None:
    assert normalize_issuer("https://idp.example/") == "https://idp.example"


def test_normalize_issuer_requires_absolute_url() -> None:
    with pytest.raises(ValueError, match="absolute URL"):
        normalize_issuer("relative/path")


def test_fetch_openid_configuration_mocked() -> None:
    body = {
        "authorization_endpoint": "https://idp.example/oauth/authorize",
        "token_endpoint": "https://idp.example/oauth/token",
    }
    transport = httpx.MockTransport(
        lambda r: httpx.Response(200, json=body)
        if r.url.path.endswith("/.well-known/openid-configuration")
        else httpx.Response(404),
    )
    client = httpx.Client(transport=transport, timeout=5.0)
    cm = mock.Mock()
    cm.__enter__ = mock.Mock(return_value=client)
    cm.__exit__ = mock.Mock(return_value=False)
    with mock.patch("agntcy.dir_sdk.client.oauth_pkce.httpx.Client", return_value=cm):
        meta = fetch_openid_configuration("https://idp.example", verify=True, timeout=5.0)
    assert meta["token_endpoint"] == body["token_endpoint"]
    client.close()


def test_fetch_openid_configuration_missing_endpoints() -> None:
    transport = httpx.MockTransport(lambda r: httpx.Response(200, json={}))
    client = httpx.Client(transport=transport, timeout=5.0)
    cm = mock.Mock()
    cm.__enter__ = mock.Mock(return_value=client)
    cm.__exit__ = mock.Mock(return_value=False)
    with mock.patch("agntcy.dir_sdk.client.oauth_pkce.httpx.Client", return_value=cm):
        with pytest.raises(OAuthPkceError, match="missing"):
            fetch_openid_configuration("https://idp.example", verify=True, timeout=5.0)
    client.close()


def test_exchange_authorization_code_mocked() -> None:
    def transport_handler(request: httpx.Request) -> httpx.Response:
        assert b"grant_type=authorization_code" in request.content
        return httpx.Response(
            200,
            json={"access_token": "at1", "token_type": "Bearer", "expires_in": 3600},
        )

    transport = httpx.MockTransport(transport_handler)
    client = httpx.Client(transport=transport, timeout=5.0)
    cm = mock.Mock()
    cm.__enter__ = mock.Mock(return_value=client)
    cm.__exit__ = mock.Mock(return_value=False)
    with mock.patch("agntcy.dir_sdk.client.oauth_pkce.httpx.Client", return_value=cm):
        tok = exchange_authorization_code(
            "https://idp.example/token",
            code="c",
            redirect_uri="http://127.0.0.1:9/cb",
            client_id="cid",
            code_verifier="ver",
            client_secret="sec",
            verify=True,
            timeout=5.0,
        )
    assert tok["access_token"] == "at1"
    client.close()


def test_oauth_token_holder_refresh() -> None:
    def transport_handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={"access_token": "new", "expires_in": 3600},
        )

    transport = httpx.MockTransport(transport_handler)
    client = httpx.Client(transport=transport, timeout=5.0)
    cm = mock.Mock()
    cm.__enter__ = mock.Mock(return_value=client)
    cm.__exit__ = mock.Mock(return_value=False)
    with mock.patch("agntcy.dir_sdk.client.oauth_pkce.httpx.Client", return_value=cm):
        h = OAuthTokenHolder(
            "https://idp.example/token",
            "cid",
            "sec",
            verify_http=True,
            http_timeout=5.0,
        )
        h.set_tokens("old", refresh_token="r1", expires_in=3600)
        h._expires_at = time.monotonic() - 1.0
        assert h.get_access_token() == "new"
    client.close()


def test_run_loopback_pkce_state_mismatch() -> None:
    port = 18766
    meta = {
        "authorization_endpoint": "http://127.0.0.1:1/auth",
        "token_endpoint": "http://127.0.0.1:1/token",
    }
    cfg = Config(
        oidc_issuer="https://idp.example",
        oidc_client_id="cid",
        oidc_redirect_uri=f"http://127.0.0.1:{port}/callback",
        oidc_callback_port=port,
        oidc_auth_timeout=5.0,
        oidc_scopes=["openid"],
    )

    def open_browser(_url: str) -> bool:
        httpx.get(
            f"http://127.0.0.1:{port}/callback?code=abc&state=wrong-state",
            timeout=2.0,
        )
        return True

    with mock.patch("webbrowser.open", open_browser):
        with pytest.raises(OAuthPkceError, match="state mismatch"):
            run_loopback_pkce_login(cfg, metadata=meta)


def test_run_loopback_pkce_success() -> None:
    port = 18767
    meta = {
        "authorization_endpoint": "http://127.0.0.1:1/auth",
        "token_endpoint": "http://127.0.0.1:1/token",
    }
    cfg = Config(
        oidc_issuer="https://idp.example",
        oidc_client_id="cid",
        oidc_client_secret="",
        oidc_redirect_uri=f"http://127.0.0.1:{port}/callback",
        oidc_callback_port=port,
        oidc_auth_timeout=10.0,
        oidc_scopes=["openid"],
    )

    token_json = {
        "access_token": "final",
        "token_type": "Bearer",
        "expires_in": 60,
    }

    def token_transport(request: httpx.Request) -> httpx.Response:
        if request.method == "POST" and request.url.path.endswith("/token"):
            return httpx.Response(200, json=token_json)
        return httpx.Response(404)

    t = httpx.MockTransport(token_transport)

    def open_browser(url: str) -> bool:
        from urllib.parse import parse_qs, urlparse

        q = parse_qs(urlparse(url).query)
        state = q["state"][0]
        httpx.get(
            f"http://127.0.0.1:{port}/callback?code=thecode&state={state}",
            timeout=2.0,
        )
        return True

    real_client = httpx.Client

    def make_client(**kw: object) -> httpx.Client:
        return real_client(
            transport=t,
            timeout=float(kw.get("timeout", 30.0)),
            verify=kw.get("verify", True),
        )

    with mock.patch("webbrowser.open", open_browser):
        with mock.patch(
            "agntcy.dir_sdk.client.oauth_pkce.httpx.Client",
            make_client,
        ):
            out = run_loopback_pkce_login(cfg, metadata=meta)
    assert out["access_token"] == "final"

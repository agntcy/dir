# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from __future__ import annotations

import json
import os
import tempfile
import unittest
import unittest.mock

from datetime import UTC, datetime, timedelta

from agntcy.dir_sdk.client import Client, Config
from agntcy.dir_sdk.client.oauth_pkce import OAuthTokenHolder
from agntcy.dir_sdk.client.token_cache import TOKEN_CACHE_FILE, TokenCache


class OIDCAuthConfigTests(unittest.TestCase):
    def test_load_from_env_uses_auth_token(self) -> None:
        with unittest.mock.patch.dict(
            "os.environ",
            {
                "DIRECTORY_CLIENT_AUTH_TOKEN": "primary-token",
            },
            clear=True,
        ):
            config = Config.load_from_env()

        self.assertEqual(config.auth_token, "primary-token")
        self.assertEqual(config.oidc_access_token, "primary-token")

    def test_load_from_env_ignores_legacy_token_names(self) -> None:
        with unittest.mock.patch.dict(
            "os.environ",
            {
                "DIRECTORY_CLIENT_OIDC_ACCESS_TOKEN": "legacy-token",
                "DIRECTORY_CLIENT_OAUTH_ACCESS_TOKEN": "older-legacy-token",
            },
            clear=True,
        ):
            config = Config.load_from_env()

        self.assertEqual(config.auth_token, "")
        self.assertEqual(config.oidc_access_token, "")

    def test_machine_flow_config_is_removed(self) -> None:
        config = Config()

        self.assertFalse(hasattr(config, "oidc_machine_client_id"))
        self.assertFalse(hasattr(config, "oidc_machine_client_secret"))

    def test_token_cache_uses_dirctl_path(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            with unittest.mock.patch.dict("os.environ", {"XDG_CONFIG_HOME": tmp_dir}, clear=True):
                cache = TokenCache()

        self.assertEqual(
            cache.get_cache_path(),
            TokenCache(os.path.join(tmp_dir, "dirctl")).get_cache_path(),
        )


class OIDCAuthClientTests(unittest.TestCase):
    def test_constructor_uses_preissued_token_without_pkce(self) -> None:
        config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
            auth_token="preissued-token",
        )

        with (
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.fetch_openid_configuration",
            ) as fetch_mock,
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.run_loopback_pkce_login",
            ) as login_mock,
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.TokenCache.get_valid_token",
                return_value=None,
            ),
        ):
            client = Client(config)

        self.assertEqual(client._oauth_holder.get_access_token(), "preissued-token")
        fetch_mock.assert_not_called()
        login_mock.assert_not_called()

    def test_constructor_without_token_does_not_start_pkce(self) -> None:
        config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
        )

        with (
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.fetch_openid_configuration",
            ) as fetch_mock,
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.run_loopback_pkce_login",
            ) as login_mock,
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.TokenCache.get_valid_token",
                return_value=None,
            ),
        ):
            client = Client(config)

        with self.assertRaisesRegex(RuntimeError, "DIRECTORY_CLIENT_AUTH_TOKEN"):
            client._oauth_holder.get_access_token()
        fetch_mock.assert_not_called()
        login_mock.assert_not_called()

    def test_constructor_uses_cached_token_without_pkce(self) -> None:
        config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
        )

        with tempfile.TemporaryDirectory() as tmp_dir:
            cache_dir = os.path.join(tmp_dir, "dirctl")
            os.makedirs(cache_dir, exist_ok=True)
            cache_path = os.path.join(cache_dir, TOKEN_CACHE_FILE)
            payload = {
                "access_token": "cached-token",
                "token_type": "bearer",
                "provider": "oidc",
                "issuer": "https://issuer.example.com",
                "refresh_token": "cached-refresh-token",
                "expires_at": (
                    datetime.now(UTC) + timedelta(hours=1)
                ).isoformat().replace("+00:00", "Z"),
                "created_at": datetime.now(UTC).isoformat().replace("+00:00", "Z"),
            }
            with open(cache_path, "w", encoding="utf-8") as f:
                json.dump(payload, f)

            with (
                unittest.mock.patch.dict("os.environ", {"XDG_CONFIG_HOME": tmp_dir}, clear=True),
                unittest.mock.patch(
                    "agntcy.dir_sdk.client.client.fetch_openid_configuration",
                ) as fetch_mock,
                unittest.mock.patch(
                    "agntcy.dir_sdk.client.client.run_loopback_pkce_login",
                ) as login_mock,
            ):
                client = Client(config)

        self.assertEqual(client._oauth_holder.get_access_token(), "cached-token")
        fetch_mock.assert_not_called()
        login_mock.assert_not_called()

    def test_authenticate_oauth_pkce_updates_access_token(self) -> None:
        config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
            oidc_issuer="https://issuer.example.com",
            oidc_client_id="client-id",
        )
        with tempfile.TemporaryDirectory() as tmp_dir:
            with unittest.mock.patch.dict("os.environ", {"XDG_CONFIG_HOME": tmp_dir}, clear=True):
                client = Client(config)

                with (
                    unittest.mock.patch(
                        "agntcy.dir_sdk.client.client.fetch_openid_configuration",
                        return_value={
                            "authorization_endpoint": "https://issuer.example.com/auth",
                            "token_endpoint": "https://issuer.example.com/token",
                        },
                    ) as fetch_mock,
                    unittest.mock.patch(
                        "agntcy.dir_sdk.client.client.run_loopback_pkce_login",
                        return_value={
                            "access_token": "fresh-token",
                            "refresh_token": "ignored-refresh-token",
                            "expires_in": 3600,
                        },
                    ) as login_mock,
                ):
                    client.authenticate_oauth_pkce()

        self.assertEqual(client._oauth_holder.get_access_token(), "fresh-token")
        fetch_mock.assert_called_once()
        login_mock.assert_called_once()

    def test_authenticate_oauth_pkce_saves_go_compatible_cache_entry(self) -> None:
        config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
            oidc_issuer="https://issuer.example.com",
            oidc_client_id="client-id",
        )

        with tempfile.TemporaryDirectory() as tmp_dir:
            with unittest.mock.patch.dict("os.environ", {"XDG_CONFIG_HOME": tmp_dir}, clear=True):
                client = Client(config)
                with (
                    unittest.mock.patch(
                        "agntcy.dir_sdk.client.client.fetch_openid_configuration",
                        return_value={
                            "authorization_endpoint": "https://issuer.example.com/auth",
                            "token_endpoint": "https://issuer.example.com/token",
                        },
                    ),
                    unittest.mock.patch(
                        "agntcy.dir_sdk.client.client.run_loopback_pkce_login",
                        return_value={
                            "access_token": "fresh-token",
                            "refresh_token": "refresh-token",
                            "token_type": "bearer",
                            "expires_in": 3600,
                        },
                    ),
                ):
                    client.authenticate_oauth_pkce()

                cached_token = TokenCache().load()

        self.assertIsNotNone(cached_token)
        self.assertEqual(cached_token.access_token, "fresh-token")
        self.assertEqual(cached_token.refresh_token, "refresh-token")
        self.assertEqual(cached_token.provider, "oidc")
        self.assertEqual(cached_token.issuer, "https://issuer.example.com")
        self.assertEqual(cached_token.token_type, "bearer")
        self.assertIsNotNone(cached_token.created_at)
        self.assertIsNotNone(cached_token.expires_at)

    def test_oauth_channel_uses_configured_tls_ca(self) -> None:
        client = Client.__new__(Client)
        client.config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
            tls_ca_file="",
        )
        client._oauth_holder = OAuthTokenHolder()
        client._oauth_holder.set_tokens("token")

        with tempfile.NamedTemporaryFile() as ca_file:
            ca_file.write(b"test-ca")
            ca_file.flush()
            client.config.tls_ca_file = ca_file.name

            with (
                unittest.mock.patch(
                    "agntcy.dir_sdk.client.client.grpc.ssl_channel_credentials",
                    return_value="creds",
                ) as creds_mock,
                unittest.mock.patch(
                    "agntcy.dir_sdk.client.client.grpc.secure_channel",
                    return_value="channel",
                ) as secure_mock,
                unittest.mock.patch(
                    "agntcy.dir_sdk.client.client.grpc.intercept_channel",
                    return_value="intercepted-channel",
                ) as intercept_mock,
            ):
                channel = client._Client__create_oauth_pkce_channel()

        self.assertEqual(channel, "intercepted-channel")
        creds_mock.assert_called_once_with(root_certificates=b"test-ca")
        secure_mock.assert_called_once_with(
            target="directory.example.com:443",
            credentials="creds",
            options=[],
        )
        intercept_mock.assert_called_once()

    def test_oauth_channel_uses_tls_server_name_override(self) -> None:
        client = Client.__new__(Client)
        client.config = Config(
            server_address="directory.example.com:443",
            auth_mode="oidc",
            tls_server_name="override.example.com",
        )
        client._oauth_holder = OAuthTokenHolder()
        client._oauth_holder.set_tokens("token")

        with (
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.grpc.ssl_channel_credentials",
                return_value="creds",
            ) as creds_mock,
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.grpc.secure_channel",
                return_value="channel",
            ) as secure_mock,
            unittest.mock.patch(
                "agntcy.dir_sdk.client.client.grpc.intercept_channel",
                return_value="intercepted-channel",
            ) as intercept_mock,
        ):
            channel = client._Client__create_oauth_pkce_channel()

        self.assertEqual(channel, "intercepted-channel")
        creds_mock.assert_called_once_with(root_certificates=None)
        secure_mock.assert_called_once_with(
            target="directory.example.com:443",
            credentials="creds",
            options=[
                ("grpc.ssl_target_name_override", "override.example.com"),
                ("grpc.default_authority", "override.example.com"),
            ],
        )
        intercept_mock.assert_called_once()

    def test_client_credentials_flow_method_is_removed(self) -> None:
        self.assertFalse(hasattr(Client, "run_client_credentials_flow"))


if __name__ == "__main__":
    unittest.main()

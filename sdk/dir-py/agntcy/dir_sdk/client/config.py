# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os


def _parse_bool_env(value: str | None, default: bool) -> bool:
    if value is None or value == "":
        return default
    return value.strip().lower() in ("1", "true", "yes", "on")


def _parse_int_env(value: str | None, default: int) -> int:
    if value is None or value == "":
        return default
    return int(value)


def _parse_float_env(value: str | None, default: float) -> float:
    if value is None or value == "":
        return default
    return float(value)


def _parse_comma_scopes(value: str | list[str] | None, default: list[str]) -> list[str]:
    if value is None or value == "":
        return list(default)
    if isinstance(value, list):
        return list(value)
    return [s.strip() for s in value.split(",") if s.strip()]


class Config:
    DEFAULT_SERVER_ADDRESS = "127.0.0.1:8888"
    DEFAULT_DIRCTL_PATH = "dirctl"
    DEFAULT_SPIFFE_SOCKET_PATH = ""
    DEFAULT_AUTH_MODE = ""
    DEFAULT_AUTH_TOKEN: str = ""
    DEFAULT_JWT_AUDIENCE = ""
    DEFAULT_TLS_CA_FILE = ""
    DEFAULT_TLS_CERT_FILE = ""
    DEFAULT_TLS_KEY_FILE = ""
    DEFAULT_TLS_SERVER_NAME = ""
    DEFAULT_TLS_SKIP_VERIFY = False

    DEFAULT_OIDC_ISSUER: str = ""
    DEFAULT_OIDC_CLIENT_ID: str = ""
    DEFAULT_OIDC_CLIENT_SECRET: str = ""
    DEFAULT_OIDC_REDIRECT_URI: str = "http://localhost:8484/callback"
    DEFAULT_OIDC_CALLBACK_PORT: int = 8484
    DEFAULT_OIDC_AUTH_TIMEOUT: float = 300.0
    DEFAULT_OIDC_SCOPES: list[str] = ["openid", "profile", "email"]

    def __init__(
        self,
        server_address: str = DEFAULT_SERVER_ADDRESS,
        dirctl_path: str = DEFAULT_DIRCTL_PATH,
        spiffe_socket_path: str = DEFAULT_SPIFFE_SOCKET_PATH,
        auth_mode: str = DEFAULT_AUTH_MODE,
        auth_token: str = DEFAULT_AUTH_TOKEN,
        jwt_audience: str = DEFAULT_JWT_AUDIENCE,
        tls_ca_file: str = DEFAULT_TLS_CA_FILE,
        tls_cert_file: str = DEFAULT_TLS_CERT_FILE,
        tls_key_file: str = DEFAULT_TLS_KEY_FILE,
        tls_server_name: str = DEFAULT_TLS_SERVER_NAME,
        tls_skip_verify: bool = DEFAULT_TLS_SKIP_VERIFY,
        oidc_issuer: str = DEFAULT_OIDC_ISSUER,
        oidc_client_id: str = DEFAULT_OIDC_CLIENT_ID,
        oidc_client_secret: str = DEFAULT_OIDC_CLIENT_SECRET,
        oidc_redirect_uri: str = DEFAULT_OIDC_REDIRECT_URI,
        oidc_callback_port: int = DEFAULT_OIDC_CALLBACK_PORT,
        oidc_auth_timeout: float = DEFAULT_OIDC_AUTH_TIMEOUT,
        oidc_scopes: list[str] | None = None,
        oidc_access_token: str | None = None,
    ) -> None:
        self.server_address = server_address
        self.dirctl_path = dirctl_path
        self.spiffe_socket_path = spiffe_socket_path
        self.auth_mode = auth_mode  # '', insecure, x509, jwt, tls, oidc
        resolved_auth_token = auth_token or oidc_access_token or ""
        self.auth_token = resolved_auth_token
        # Backward-compatible alias for older callers.
        self.oidc_access_token = resolved_auth_token
        self.jwt_audience = jwt_audience
        self.tls_ca_file = tls_ca_file
        self.tls_cert_file = tls_cert_file
        self.tls_key_file = tls_key_file
        self.tls_server_name = tls_server_name
        self.tls_skip_verify = tls_skip_verify
        self.oidc_issuer = oidc_issuer
        self.oidc_client_id = oidc_client_id
        self.oidc_client_secret = oidc_client_secret
        self.oidc_redirect_uri = oidc_redirect_uri
        self.oidc_callback_port = oidc_callback_port
        self.oidc_auth_timeout = oidc_auth_timeout
        self.oidc_scopes = (
            list(oidc_scopes)
            if oidc_scopes is not None
            else list(Config.DEFAULT_OIDC_SCOPES)
        )

    @staticmethod
    def load_from_env(env_prefix: str = "DIRECTORY_CLIENT_") -> "Config":
        """Load configuration from environment variables."""
        dirctl_path = os.environ.get(
            "DIRCTL_PATH",
            Config.DEFAULT_DIRCTL_PATH,
        )

        server_address = os.environ.get(
            f"{env_prefix}SERVER_ADDRESS",
            Config.DEFAULT_SERVER_ADDRESS,
        )
        spiffe_socket_path = os.environ.get(
            f"{env_prefix}SPIFFE_SOCKET_PATH",
            Config.DEFAULT_SPIFFE_SOCKET_PATH,
        )
        auth_mode = os.environ.get(
            f"{env_prefix}AUTH_MODE",
            Config.DEFAULT_AUTH_MODE,
        )
        auth_token = os.environ.get(
            f"{env_prefix}AUTH_TOKEN",
            Config.DEFAULT_AUTH_TOKEN,
        )
        jwt_audience = os.environ.get(
            f"{env_prefix}JWT_AUDIENCE",
            Config.DEFAULT_JWT_AUDIENCE,
        )
        tls_ca_file = os.environ.get(
            f"{env_prefix}TLS_CA_FILE",
            Config.DEFAULT_TLS_CA_FILE,
        )
        tls_cert_file = os.environ.get(
            f"{env_prefix}TLS_CERT_FILE",
            Config.DEFAULT_TLS_CERT_FILE,
        )
        tls_key_file = os.environ.get(
            f"{env_prefix}TLS_KEY_FILE",
            Config.DEFAULT_TLS_KEY_FILE,
        )
        tls_server_name = os.environ.get(
            f"{env_prefix}TLS_SERVER_NAME",
            Config.DEFAULT_TLS_SERVER_NAME,
        )
        tls_skip_verify = _parse_bool_env(
            os.environ.get(f"{env_prefix}TLS_SKIP_VERIFY"),
            Config.DEFAULT_TLS_SKIP_VERIFY,
        )
        oidc_issuer = os.environ.get(
            f"{env_prefix}OIDC_ISSUER",
            Config.DEFAULT_OIDC_ISSUER,
        )
        oidc_client_id = os.environ.get(
            f"{env_prefix}OIDC_CLIENT_ID",
            Config.DEFAULT_OIDC_CLIENT_ID,
        )
        oidc_client_secret = os.environ.get(
            f"{env_prefix}OIDC_CLIENT_SECRET",
            Config.DEFAULT_OIDC_CLIENT_SECRET,
        )
        oidc_redirect_uri = os.environ.get(
            f"{env_prefix}OIDC_REDIRECT_URI",
            Config.DEFAULT_OIDC_REDIRECT_URI,
        )
        oidc_callback_port = _parse_int_env(
            os.environ.get(f"{env_prefix}OIDC_CALLBACK_PORT"),
            Config.DEFAULT_OIDC_CALLBACK_PORT,
        )
        oidc_auth_timeout = _parse_float_env(
            os.environ.get(f"{env_prefix}OIDC_AUTH_TIMEOUT"),
            Config.DEFAULT_OIDC_AUTH_TIMEOUT,
        )
        oidc_scopes = _parse_comma_scopes(
            os.environ.get(f"{env_prefix}OIDC_SCOPES"),
            Config.DEFAULT_OIDC_SCOPES,
        )

        return Config(
            server_address=server_address,
            dirctl_path=dirctl_path,
            spiffe_socket_path=spiffe_socket_path,
            auth_mode=auth_mode,
            auth_token=auth_token,
            jwt_audience=jwt_audience,
            tls_ca_file=tls_ca_file,
            tls_cert_file=tls_cert_file,
            tls_key_file=tls_key_file,
            tls_server_name=tls_server_name,
            tls_skip_verify=tls_skip_verify,
            oidc_issuer=oidc_issuer,
            oidc_client_id=oidc_client_id,
            oidc_client_secret=oidc_client_secret,
            oidc_redirect_uri=oidc_redirect_uri,
            oidc_callback_port=oidc_callback_port,
            oidc_auth_timeout=oidc_auth_timeout,
            oidc_scopes=oidc_scopes,
        )

# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os


class Config:
    DEFAULT_SERVER_ADDRESS = "127.0.0.1:8888"
    DEFAULT_DIRCTL_PATH = "dirctl"
    DEFAULT_SPIFFE_SOCKET_PATH = ""
    DEFAULT_AUTH_MODE = "insecure"
    DEFAULT_JWT_AUDIENCE = ""
    DEFAULT_TLS_CA_FILE = ""
    DEFAULT_TLS_CERT_FILE = ""
    DEFAULT_TLS_KEY_FILE = ""
    DEFAULT_TLS_SKIP_VERIFY = False
    DEFAULT_SPIFFE_TOKEN = ""
    DEFAULT_SERVER_SPIFFE_ID = ""
    DEFAULT_SKIP_HOSTNAME_VERIFY = False

    def __init__(
        self,
        server_address: str = DEFAULT_SERVER_ADDRESS,
        dirctl_path: str = DEFAULT_DIRCTL_PATH,
        spiffe_socket_path: str = DEFAULT_SPIFFE_SOCKET_PATH,
        auth_mode: str = DEFAULT_AUTH_MODE,
        jwt_audience: str = DEFAULT_JWT_AUDIENCE,
        spiffe_token: str = DEFAULT_SPIFFE_TOKEN,
        server_spiffe_id: str = DEFAULT_SERVER_SPIFFE_ID,
        skip_hostname_verify: bool = DEFAULT_SKIP_HOSTNAME_VERIFY,
        tls_ca_file: str = DEFAULT_TLS_CA_FILE,
        tls_cert_file: str = DEFAULT_TLS_CERT_FILE,
        tls_key_file: str = DEFAULT_TLS_KEY_FILE,
        tls_skip_verify: bool = DEFAULT_TLS_SKIP_VERIFY,
    ) -> None:
        self.server_address = server_address
        self.dirctl_path = dirctl_path
        self.spiffe_socket_path = spiffe_socket_path
        self.auth_mode = auth_mode  # 'insecure', 'x509', or 'jwt'
        self.jwt_audience = jwt_audience
        self.spiffe_token = spiffe_token
        self.server_spiffe_id = server_spiffe_id
        self.skip_hostname_verify = skip_hostname_verify
        self.tls_ca_file = tls_ca_file
        self.tls_cert_file = tls_cert_file
        self.tls_key_file = tls_key_file
        self.tls_skip_verify = tls_skip_verify

    @staticmethod
    def load_from_env(env_prefix: str = "DIRECTORY_CLIENT_") -> "Config":
        """Load configuration from environment variables."""
        # Get dirctl path from environment variable without prefix
        dirctl_path = os.environ.get(
            "DIRCTL_PATH",
            Config.DEFAULT_DIRCTL_PATH,
        )

        # Use prefixed environment variables for other settings
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
        jwt_audience = os.environ.get(
            f"{env_prefix}JWT_AUDIENCE",
            Config.DEFAULT_JWT_AUDIENCE,
        )
        spiffe_token = os.environ.get(
            f"{env_prefix}SPIFFE_TOKEN",
            Config.DEFAULT_SPIFFE_TOKEN,
        )
        server_spiffe_id = os.environ.get(
            f"{env_prefix}SERVER_SPIFFE_ID",
            Config.DEFAULT_SERVER_SPIFFE_ID,
        )
        skip_hostname_verify = os.environ.get(
            f"{env_prefix}SKIP_HOSTNAME_VERIFY",
            str(Config.DEFAULT_SKIP_HOSTNAME_VERIFY),
        ).lower() in ("1", "true", "yes", "on")
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
        tls_skip_verify = os.environ.get(
            f"{env_prefix}TLS_SKIP_VERIFY",
            str(Config.DEFAULT_TLS_SKIP_VERIFY),
        ).lower() in ("1", "true", "yes", "on")

        return Config(
            server_address=server_address,
            dirctl_path=dirctl_path,
            spiffe_socket_path=spiffe_socket_path,
            auth_mode=auth_mode,
            jwt_audience=jwt_audience,
            spiffe_token=spiffe_token,
            server_spiffe_id=server_spiffe_id,
            skip_hostname_verify=skip_hostname_verify,
            tls_ca_file=tls_ca_file,
            tls_cert_file=tls_cert_file,
            tls_key_file=tls_key_file,
            tls_skip_verify=tls_skip_verify,
        )

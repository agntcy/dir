# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os


class Config:
    DEFAULT_SERVER_ADDRESS = "127.0.0.1:8888"
    DEFAULT_DIRCTL_PATH = "dirctl"
    DEFAULT_SPIFFE_SOCKET_PATH = ""
    DEFAULT_AUTH_MODE = "insecure"
    DEFAULT_JWT_AUDIENCE = ""

    def __init__(
        self,
        server_address: str = DEFAULT_SERVER_ADDRESS,
        dirctl_path: str = DEFAULT_DIRCTL_PATH,
        spiffe_socket_path: str = DEFAULT_SPIFFE_SOCKET_PATH,
        auth_mode: str = DEFAULT_AUTH_MODE,
        jwt_audience: str = DEFAULT_JWT_AUDIENCE,
    ) -> None:
        self.server_address = server_address
        self.dirctl_path = dirctl_path
        self.spiffe_socket_path = spiffe_socket_path
        self.auth_mode = auth_mode  # 'insecure', 'mtls', or 'jwt'
        self.jwt_audience = jwt_audience

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

        return Config(
            server_address=server_address,
            dirctl_path=dirctl_path,
            spiffe_socket_path=spiffe_socket_path,
            auth_mode=auth_mode,
            jwt_audience=jwt_audience,
        )

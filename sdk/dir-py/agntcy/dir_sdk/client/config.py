# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os


class Config:
    DEFAULT_SERVER_ADDRESS = "0.0.0.0:8888"
    DEFAULT_DIRCTL_PATH = "dirctl"

    def __init__(
        self,
        server_address: str = DEFAULT_SERVER_ADDRESS,
        dirctl_path: str = DEFAULT_DIRCTL_PATH,
    ) -> None:
        self.server_address = server_address
        self.dirctl_path = dirctl_path

    @staticmethod
    def load_from_env(env_prefix: str = "DIRECTORY_CLIENT_") -> "Config":
        """Load configuration from environment variables."""
        server_address = os.environ.get(
            f"{env_prefix}SERVER_ADDRESS",
            Config.DEFAULT_SERVER_ADDRESS,
        )
        dirctl_path = os.environ.get(
            "DIRCTL_PATH",
            Config.DEFAULT_DIRCTL_PATH,
        )

        return Config(
            server_address=server_address,
            dirctl_path=dirctl_path,
        )

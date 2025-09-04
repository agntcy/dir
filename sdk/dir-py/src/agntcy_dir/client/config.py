import os

class Config:
    DEFAULT_ENV_PREFIX = "DIRECTORY_CLIENT"
    DEFAULT_SERVER_ADDRESS = "0.0.0.0:8888"

    DEFAULT_DIRCTL_PATH = "dirctl"

    def __init__(
        self,
        server_address: str = DEFAULT_SERVER_ADDRESS,
        dirctl_path: str = DEFAULT_DIRCTL_PATH,
    ):
        self.server_address = server_address
        self.dirctl_path = dirctl_path

    @staticmethod
    def load_from_env() -> "Config":
        """Load configuration from environment variables"""
        prefix = Config.DEFAULT_ENV_PREFIX
        server_address = os.environ.get(
            f"{prefix}_SERVER_ADDRESS", Config.DEFAULT_SERVER_ADDRESS
        )

        dirctl_path = os.environ.get("DIRCTL_PATH", Config.DEFAULT_DIRCTL_PATH)

        return Config(server_address=server_address, dirctl_path=dirctl_path)

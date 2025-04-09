# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import os
from pydantic import BaseSettings

class Config(BaseSettings):
    server_address: str = "0.0.0.0:8888"

    class Config:
        env_prefix = "DIRECTORY_CLIENT_"
        env_file = ".env"
        env_file_encoding = 'utf-8'

def load_config() -> Config:
    return Config()

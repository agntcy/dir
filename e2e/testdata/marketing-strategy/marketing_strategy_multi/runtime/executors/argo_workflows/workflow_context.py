# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from typing import Optional

from pydantic import BaseModel, Field


class WorkflowContext(BaseModel):
    generate_name: str = Field(
        description="Name of the workflow", alias="generate_name"
    )
    image_pull_secrets: Optional[list[str]] = Field(
        description="Image pull secrets", alias="image_pull_secrets", default=[]
    )
    program: str = Field(description="Program to run", alias="program")
    input_message: str = Field(description="Input message", alias="input_message")
    agent_runner_image: str = Field(
        description="Agent runner image", alias="agent_runner_image"
    )
    additional_env_vars: Optional[dict[str, str]] = Field(
        description="Additional environment variables",
        alias="additional_env_vars",
        default={},
    )
    additional_env_vars_from_secret: Optional[dict[str, list[str]]] = Field(
        description="Additional environment variables from secret",
        alias="additional_env_vars_from_secret",
        default={},
    )

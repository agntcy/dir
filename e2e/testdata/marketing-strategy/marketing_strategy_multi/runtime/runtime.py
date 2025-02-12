# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from abc import ABC
from concurrent.futures import Future
from enum import StrEnum
from typing import Optional

from marketing_strategy_multi.runtime.observability.observer import Observable
from pydantic import BaseModel, Field


class ExecutorType(StrEnum):
    local = "local"
    argo = "argo-workflow"


class Metadata(BaseModel):
    agent_name: str = Field(description="Name of the agent", alias="agent_name")
    name: str = Field(description="Name of the task", alias="name")
    description: Optional[str] = Field(
        description="Description of the task", alias="description", default=None
    )


class ExecutionContext(BaseModel):
    # Inspired by k8s. Execution context might be local or remote.
    # Examples of local execution context: LocalExecutor, ArgoExecutor
    # Examples of remote execution context: RPCExecutor
    metadata: Metadata = Field(
        description="Metadata for the execution context", alias="metadata"
    )
    spec: dict[str, str | dict | list] = Field(
        description="Spec for the execution context", alias="spec"
    )
    status: Optional[dict[str, str | dict]] = Field(
        description="Status of the execution context", alias="status", default=None
    )


class Executor(ABC,Observable):
    
    def run(
        self,
        context: ExecutionContext,
        async_mode: bool = False,
        timeout_ms: int = 60000,
    ):
        pass

    def get_results(self) -> dict:
        pass

    def get_result(self, key):
        pass

# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import os
from pathlib import Path
from typing import Optional, TypedDict

from marketing_strategy_multi.runtime.runtime import ExecutorType
from pydantic import BaseModel, Field


class TaskState(BaseModel):
    name: str = Field(description="Name of the task", alias="name")
    description: str = Field(description="Description of the task", alias="description")
    summary: str = Field(description="Summary of the task", alias="summary")
    output_raw: str = Field(description="Raw output of the task", alias="output_raw")
    agent: Optional[str] = Field(
        description="Agent of the task", default=None, alias="agent"
    )
    expected_output: Optional[str] = Field(
        description="Expected output of the task", default=None, alias="expected_output"
    )


class State(BaseModel):
    executor: Optional[ExecutorType] = Field(
        description="Executor for the agents",
        default=ExecutorType.argo,
        alias="executor",
    )
    inputs: dict = Field(description="Inputs for the crew", alias="inputs")
    tasks: Optional[list[TaskState]] = Field(
        description="Tasks for the crew", default=None, alias="tasks"
    )
    outputs: Optional[str] = Field(
        description="Outputs of the crew", default=None, alias="output"
    )

    @property
    def json(self) -> Optional[str]:
        return json.dumps(self.model_dump())

    def from_crew_result(inputs: dict, previous_tasks: list[TaskState], crew, output):
        state = State(
            inputs=inputs,
            tasks=previous_tasks,
            output=output,
        )

        if state.tasks is None:
            state.tasks = []

        state.tasks += [
            TaskState(
                name=task.output.name,
                description=task.output.description,
                summary=task.output.summary,
                output_raw=task.output.raw,
                agent=task.output.agent,
                expected_output=task.output.expected_output,
            )
            for task in crew.tasks
        ]

        return state

    def to_json_file(self, output_file: str):
        print("Saving output")
        try:
            output_path = Path(output_file)
            output_path.parent.mkdir(parents=True, exist_ok=True)

            with output_path.open("w", encoding="utf-8") as f:
                json.dump(self.model_dump(), f, ensure_ascii=False, indent=4)
            print(f"File saved successfully to {output_file}")
        except Exception as e:
            print(f"Error saving file {output_file}: {e}")

class MarketingStrategyState(TypedDict):
    state: State

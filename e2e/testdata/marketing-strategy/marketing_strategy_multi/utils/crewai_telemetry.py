# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import crewai.telemetry

from crewai import Crew, Task
from typing import Any, Optional
from opentelemetry.trace import Span


class TelemetryDisabled:
    def __init__(self):
        pass

    def set_tracer(self):
        pass

    def crew_creation(self, crew: Crew, inputs: dict[str, Any] | None):
        pass

    def task_started(self, crew: Crew, task: Task) -> Span | None:
        return None

    def task_ended(self, span: Span, task: Task, crew: Crew):
        pass

    def tool_repeated_usage(self, llm: Any, tool_name: str, attempts: int):
        pass

    def tool_usage(self, llm: Any, tool_name: str, attempts: int):
        pass

    def tool_usage_error(self, llm: Any):
        pass

    def individual_test_result_span(
        self, crew: Crew, quality: float, exec_time: int, model_name: str
    ):
        pass

    def test_execution_span(
        self,
        crew: Crew,
        iterations: int,
        inputs: dict[str, Any] | None,
        model_name: str,
    ):
        pass

    def deploy_signup_error_span(self):
        pass

    def start_deployment_span(self, uuid: Optional[str] = None):
        pass

    def create_crew_deployment_span(self):
        pass

    def get_crew_logs_span(self, uuid: Optional[str], log_type: str = "deployment"):
        pass

    def remove_crew_span(self, uuid: Optional[str] = None):
        pass

    def crew_execution_span(self, crew: Crew, inputs: dict[str, Any] | None):
        pass

    def end_crew(self, crew, final_string_output):
        pass

    def _add_attribute(self, span, key, value):
        pass


def disable_crewai_telemetry():
    crewai.telemetry.Telemetry = TelemetryDisabled

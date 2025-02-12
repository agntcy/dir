# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ

from crewai import Crew, Task
from crewai.project import crew
from crewai.task import TaskOutput
from crewai_tools import tool
from duckduckgo_search import DDGS
from langchain_ollama import ChatOllama
from langchain_openai import ChatOpenAI
from marketing_strategy_multi.utils.crewai_telemetry import disable_crewai_telemetry
from marketing_strategy_multi.utils.parser import parse_args, parse_inputs
from marketing_strategy_multi.utils.state import State
from marketing_strategy_multi.utils.telemetry import instrument


class CrewAiBaseClass:
    def __init__(self, *args, **kwargs):
        self.openai_host = environ.get("OPENAI_HOST", "http://localhost:11434")
        self.openai_model = environ.get("OPENAI_MODEL", "mistral")
        self.openai_key = environ.get("OPENAI_API_KEY", "NA")

        temperature = 0.2
        if self.openai_key == "NA":
            self.llm = ChatOllama(
                base_url=self.openai_host,
                model=self.openai_model,
                temperature=temperature,
            )
        else:
            self.llm = ChatOpenAI(
                base_url=self.openai_host,
                model=self.openai_model,
                api_key=self.openai_key,
                temperature=temperature,
            )

        self.received_tasks = []

        if "tasks" in kwargs:
            for tsk in kwargs["tasks"]:
                # Make sure all tasks are Task objects
                t = Task(
                    name=tsk.name,
                    description=tsk.description,
                    expected_output=tsk.expected_output,
                )

                t.output = TaskOutput(
                    description=tsk.description,
                    raw=tsk.output_raw,
                    agent="na",
                )

                self.received_tasks += [t]

    def get_received_tasks(self) -> list:
        return self.received_tasks

    @tool("DuckDuckGoSearchResults")
    def search_tool(search_query: str):
        """Search the web for information on a given topic"""
        return DDGS().text(search_query)

    @crew
    def crew(self, telemetry_enabled: bool) -> Crew:
        """Creates the MarketingPosts crew"""

        return Crew(
            agents=self.agents,
            tasks=self.tasks,
            verbose=True,
            share_crew=telemetry_enabled,
        )


def run_crew(app_name: str, crew_instance: Crew):
    disable_crewai_telemetry()

    args = parse_args()
    in_state = parse_inputs(args)

    enabled = instrument(app_name)

    crew_args = {}
    if in_state.tasks:
        crew_args["tasks"] = in_state.tasks

    crew = crew_instance(**crew_args).crew(enabled)
    output = crew.kickoff(inputs=in_state.inputs)

    out_state: State = State.from_crew_result(
        in_state.inputs, in_state.tasks, crew, output.raw
    )
    out_state.to_json_file(args.output_file)

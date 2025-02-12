# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import os
import uuid
from functools import partial

from crewai import Crew
from marketing_strategy_multi.runtime.executors.argo_workflows.workflow_agent import ArgoExecutor
from marketing_strategy_multi.runtime.executors.local.local_executor import LocalExecutor
from marketing_strategy_multi.runtime.observability.observer import Observable
from marketing_strategy_multi.runtime.runtime import ExecutionContext, ExecutorType, Metadata
from marketing_strategy_multi.runtime.utils import common_envs
from marketing_strategy_multi.utils.state import MarketingStrategyState, State, TaskState


class Nodes(Observable):
    def __init__(self):
        super().__init__()
        self.telemetry_enabled = None
        # Create common ID for the workflow
        self.workflow_name = str(uuid.uuid4())
        
    def _get_args(self, state):
        args = {}
        if state.tasks:
            args["tasks"] = state.tasks

        return args

    def _run_node(self, state: MarketingStrategyState, attributes: dict):
        executor = state["state"].executor

        print (f"Executor: {executor}")
        self.notify_observers("Nodes", {
            "agentName": attributes['name'],
            "agentPhase": "Started"
        })

        if executor == ExecutorType.local:
            self._run_node_local(state, attributes["class"],attributes["name"],attributes['execution_id'])
        elif executor == ExecutorType.argo:
            self._run_node_remote(state, attributes["program"],attributes["name"],attributes['execution_id'],attributes['image'],attributes['image_pull_secrets'])
        else:
            raise Exception(f"Executor not supported: {executor}")
        
        self.notify_observers("Nodes", {
            "agentName": attributes['name'],
            "agentPhase": "Finished"
        })

    def _run_node_remote(self, state: MarketingStrategyState, program: str, agent_name: str, execution_id: str, image: str, image_pull_secrets: str):
        in_state = state["state"]

        if hasattr(self, "executor") is False:
            self.executor = ArgoExecutor()
            for observer in self.observers:
                self.executor.add_observer(observer)
        
        env = os.environ.copy()
        
        # Create a common ID for the workflow
        env["WORKFLOW_NAME"] = self.workflow_name
        
        context = ExecutionContext(
            metadata=Metadata(
                agent_name = agent_name,
                name = execution_id
            ),
            spec={
                "program": program,
                "image_pull_secrets": image_pull_secrets,
                "input_message": json.dumps(in_state.model_dump()),
                "agent_runner_image": image,
                "additional_env_vars": {
                    n: v
                    for n, v in env.items()
                    if n.startswith("OVAL_") or n in common_envs
                },
                "additional_env_vars_from_secret": {
                    "OPENAI_API_KEY": ("openai", "api-key"),
                },
            },
        )

        self.executor.run(context=context, timeout_ms=3600000)
        out_state = self.executor.get_results(context)

        state["state"].outputs = out_state["outputs"]
        state["state"].tasks = [TaskState(**t) for t in out_state["tasks"]]

        return state

    def _run_node_local(self, state: State, agent_class: Crew,agent_name: str,execution_id: str):
        if hasattr(self, "executor") is False:
            self.executor = LocalExecutor()

        self.executor.run(
            ExecutionContext(
                metadata=Metadata(
                    agent_name = agent_name,
                    name=execution_id
                ),
                spec={
                    "node_functions": [
                        partial(self._run_node_local_2, state, agent_class)
                    ],
                },
            )
        )

    def _run_node_local_2(self, state: State, agent_class: Crew):
        in_state = state["state"]

        args = self._get_args(in_state)
        agent = agent_class(**args)

        crew = agent.crew(self.telemetry_enabled)
        output = crew.kickoff(inputs=in_state.inputs)
        out_state: State = State.from_crew_result(
            in_state.inputs, in_state.tasks, crew, output.raw
        )
        out_state.executor = ExecutorType.local

        state["state"] = out_state
        return state

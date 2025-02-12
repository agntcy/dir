# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from crewai import Agent, Task
from crewai.project import CrewBase, agent, task

from crewai_tools import ScrapeWebsiteTool

from marketing_strategy_multi.utils.crew import CrewAiBaseClass


@CrewBase
class ChiefMarketingStrategistCrew(CrewAiBaseClass):
    """ChiefMarketingStrategistCrew crew"""

    agents_config = "config/agents.yaml"
    tasks_config = "config/tasks.yaml"

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    @agent
    def chief_marketing_strategist(self) -> Agent:
        return Agent(
            config=self.agents_config["chief_marketing_strategist"],
            tools=[self.search_tool, ScrapeWebsiteTool()],
            verbose=True,
            memory=False,
            allow_delegation=False,
            llm=self.llm,
        )

    @task
    def project_understanding_task(self) -> Task:
        # Get latest context
        context = self.get_received_tasks()[-1:]

        return Task(
            name="project_understanding_task",
            config=self.tasks_config["project_understanding_task"],
            agent=self.chief_marketing_strategist(),
            context=context,
        )

    @task
    def marketing_strategy_task(self) -> Task:
        return Task(
            name="marketing_strategy_task",
            config=self.tasks_config["marketing_strategy_task"],
            agent=self.chief_marketing_strategist(),
        )

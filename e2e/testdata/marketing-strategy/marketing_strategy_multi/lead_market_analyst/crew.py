# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from crewai import Agent, Crew, Task
from crewai.project import CrewBase, agent, crew, task
from crewai.task import TaskOutput

# Check our tools documentations for more information on how to use them
from crewai_tools import ScrapeWebsiteTool
from marketing_strategy_multi.utils.crew import CrewAiBaseClass


@CrewBase
class LeadMarketAnalystCrew(CrewAiBaseClass):
    """MarketingPosts crew"""

    agents_config = "config/agents.yaml"
    tasks_config = "config/tasks.yaml"

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    @agent
    def lead_market_analyst(self) -> Agent:
        return Agent(
            config=self.agents_config["lead_market_analyst"],
            tools=[self.search_tool, ScrapeWebsiteTool()],
            verbose=True,
            memory=False,
            allow_delegation=False,
            llm=self.llm,
        )

    @task
    def research_task(self) -> Task:
        return Task(
            name="research_task",
            config=self.tasks_config["research_task"],
            agent=self.lead_market_analyst(),
        )

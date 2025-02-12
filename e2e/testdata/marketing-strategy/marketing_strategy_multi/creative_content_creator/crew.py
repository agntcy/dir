# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ

from crewai import Agent, Crew, Task
from crewai.project import CrewBase, agent, crew, task
from crewai_tools import ScrapeWebsiteTool
from marketing_strategy_multi.utils.crew import CrewAiBaseClass


@CrewBase
class CreativeContentCreatorCrew(CrewAiBaseClass):
    """CreativeContentCreator crew"""

    agents_config = "config/agents.yaml"
    tasks_config = "config/tasks.yaml"

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    @agent
    def creative_content_creator(self) -> Agent:
        return Agent(
            config=self.agents_config["creative_content_creator"],
            tools=[self.search_tool, ScrapeWebsiteTool()],
            verbose=True,
            memory=False,
            allow_delegation=False,
            llm=self.llm,
        )

    @task
    def campaign_idea_task(self) -> Task:
        context = self.get_received_tasks()[-1:]

        return Task(
            name="campaign_idea_task",
            config=self.tasks_config["campaign_idea_task"],
            agent=self.creative_content_creator(),
            context=context,
        )

    @task
    def copy_creation_task(self) -> Task:
        context = [self.campaign_idea_task()]
        for task in self.get_received_tasks():
            context += [task] if task.name == "marketing_strategy_task" else []

        return Task(
            config=self.tasks_config["copy_creation_task"],
            agent=self.creative_content_creator(),
            context=context,
        )

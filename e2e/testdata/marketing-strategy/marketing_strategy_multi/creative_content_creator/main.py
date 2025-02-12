#!/usr/bin/env python
# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ
import logging
from marketing_strategy_multi.creative_content_creator.crew import (
    CreativeContentCreatorCrew,
)
from marketing_strategy_multi.utils.crew import run_crew

app_name = environ.get("APP_NAME", "creative-content-creator")
workflow_name = environ.get("WORKFLOW_NAME", "no-name")


def run():
    run_crew(app_name + "_" + workflow_name, CreativeContentCreatorCrew)


if __name__ == "__main__":
    run()

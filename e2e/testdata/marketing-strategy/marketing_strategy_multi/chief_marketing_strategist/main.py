#!/usr/bin/env python
# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from os import environ
from marketing_strategy_multi.chief_marketing_strategist.crew import (
    ChiefMarketingStrategistCrew,
)
from marketing_strategy_multi.utils.crew import run_crew

app_name = environ.get("APP_NAME", "chief-marketing-strategist")
workflow_name = environ.get("WORKFLOW_NAME", "no-name")


def run():
    run_crew(app_name + "_" + workflow_name, ChiefMarketingStrategistCrew)


if __name__ == "__main__":
    run()

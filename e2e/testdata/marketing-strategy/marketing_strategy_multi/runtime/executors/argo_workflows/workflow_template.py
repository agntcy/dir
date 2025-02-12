# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

from pathlib import Path

import yaml
from jinja2 import Template
from marketing_strategy_multi.runtime.executors.argo_workflows.workflow_context import WorkflowContext


def workflow_template_from_file(context: WorkflowContext):
    template_file_path = Path(__file__).with_name("workflow_agent.yaml.tpl")

    with open(template_file_path, "r") as template_file:
        template_content = template_file.read()

    template = Template(template_content)
    rendered_content = template.render(context.model_dump())

    # Replace {- with {{ and  replace -} with }} because using argo template
    rendered_content = rendered_content.replace("{-", "{{")
    rendered_content = rendered_content.replace("-}", "}}")

    return yaml.safe_load(rendered_content)

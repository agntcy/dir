# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import argparse
import json
import os

import yaml
from marketing_strategy_multi.runtime.runtime import ExecutorType
from marketing_strategy_multi.utils.state import State

default_input = """
{
    "inputs": {
        "customer_domain": "lavazza.com",
        "project_description": "Espresso Lavazza, a renowned provider of premium coffee products, aims to revolutionize the coffee experience for its global customers. This project involves developing an innovative marketing strategy to showcase Espresso Lavazza high-quality, authentic Italian coffee, emphasizing richness in flavor, sustainability, and convenience. The campaign will target coffee enthusiasts and connoisseurs worldwide, highlighting success stories and the transformative potential of Espresso Lavazza offerings. Customer Domain: Premium Coffee Products. Project Overview: Creating a comprehensive marketing campaign to boost awareness and adoption of Espresso Lavazza products among global coffee lovers."
    }
}
"""

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input-json", type=str, required=False, help="Inputs json", default=default_input)
    parser.add_argument("--input-file", type=str, required=False, help="Inputs file")
    parser.add_argument("--output-file", type=str, required=True, help="Output file")
    parser.add_argument(
        "--executor",
        type=ExecutorType,
        choices=list(ExecutorType),
        required=False,
        default=ExecutorType.argo,
        help="Executor",
    )
    args = parser.parse_args()
    return args


def parse_inputs_json(inputs_file: str) -> dict:
    with open(inputs_file, "r") as f:
        inputs = json.load(f)
    return inputs


def parse_inputs_yaml(inputs_file: str) -> dict:
    with open(inputs_file, "r") as f:
        inputs = yaml.safe_load(f)
    return inputs


def parse_inputs(args: argparse.Namespace) -> State:
    dict = json.loads(args.input_json)

    input_file = os.environ.get("INPUTS_FILE", None)
    if args.input_file:
        input_file = args.input_file

    if input_file and args.input_json == default_input:
        if input_file.endswith(".json"):
            dict = parse_inputs_json(input_file)
        elif input_file.endswith(".yaml") or input_file.endswith(".yml"):
            dict = parse_inputs_yaml(input_file)
        else:
            raise ValueError(f"Invalid file format: {input_file}")

    ret = State.model_validate(dict)
    if args.executor:
        ret.executor = args.executor

    return ret

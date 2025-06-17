#!/bin/bash

######################################################################
## Example using Directory as a way to integrate Agentic workflows
## with **VSCode Copilot** and **Continue VSCode extension** natively
######################################################################

## Get the path of script directory
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

#### Hub/Directory flow
## Push record to ADS
# DIGEST=$(dirctl push $SCRIPT_DIR/record.json)

## Pull record from ADS
# dirctl pull $DIGEST > $SCRIPT_DIR/record.json

#### Integration support
# Requirements: Python3, venv, pyyaml
python3 $SCRIPT_DIR/poc.py \
        -record=$SCRIPT_DIR/record.json \
        -vscode_path=$SCRIPT_DIR/../../../.vscode \
        -continue_path=$SCRIPT_DIR/../../../.continue/assistants

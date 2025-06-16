#!/bin/bash

## Example using Directory as a way to describe LLM support information
##
## This POC can be used with **Continue VSCode Extension** natively

## Push record to ADS
# DIGEST=$(dirctl push record.json)

## Pull record from ADS
# dirctl pull $DIGEST > record.json

## Extract LLM information from the record
cat record.json | jq '.llm_info' > llm.json

## Add the LLM information to VSCode
mv llm.json ./../../../.vscode

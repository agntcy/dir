#!/bin/bash

## Example using Directory as a way to distribute A2A information
##
## This POC can be used with any tooling that supports A2A

## Push record to ADS
# DIGEST=$(dirctl push record.json)

## Pull record from ADS
# dirctl pull $DIGEST > record.json

## Extract A2A information from the record
cat record.json | jq '.extensions[] | select(.name == "a2a-card") | .data' > a2a.json

## Add the A2A information to VSCode
mv a2a.json ./../../../.vscode

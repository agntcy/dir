#!/bin/bash

## VARIABLES
DIR_TRUST_DOMAIN="dir-server.com"
DIRCTL_TRUST_DOMAIN="dir-client.com"

# Deploy everything
task deploy:k8s:spire:cleanup
task deploy:k8s:spire:cluster
task deploy:k8s:spire:chart
task deploy:k8s:spire:dir
task deploy:k8s:spire:dirctl

# Check logs
wait 60
task logs:k8s:spire:dir
task logs:k8s:spire:dirctl

#!/usr/bin/env python3

##############################################################################
## This script processes an Agent Record JSON file and generates 
## output files for VSCode and Continue. The output files enable 
## native integration of agentic workflows with VSCode Copilot and
## Continue VSCode Extension.
##
## Usage: 
##
##    ./importer.py -record=./record.json -vscode_path=./.vscode -continue_path=./.continue/assistants
##
##############################################################################

import argparse
import json
import yaml
import os
from pathlib import Path

def parse_arguments():
    parser = argparse.ArgumentParser(description='Process record JSON file and generate output files')
    parser.add_argument('-record', help='Path to the input JSON file', required=True)
    parser.add_argument('-vscode_path', help='Output path for VSCode directory', required=True)
    parser.add_argument('-continue_path', help='Output path for Continue directory', required=True)
    args = parser.parse_args()
    
    # Convert all paths to absolute paths
    args.record = os.path.abspath(args.record)
    args.vscode_path = os.path.abspath(args.vscode_path)
    args.continue_path = os.path.abspath(args.continue_path)
    
    return args

def read_json_file(file_path):
    try:
        with open(file_path, 'r') as f:
            return json.load(f)
    except json.JSONDecodeError as e:
        print(f"Error decoding JSON file: {e}")
        exit(1)
    except FileNotFoundError:
        print(f"Input file not found: {file_path}")
        exit(1)

def write_json_file(data, file_path):
    # Ensure the directory exists
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    
    try:
        with open(file_path, 'w') as f:
            json.dump(data, f, indent=2)
        print(f"Successfully wrote to: {file_path}")
    except Exception as e:
        print(f"Error writing to {file_path}: {e}")
        exit(1)

def write_yaml_file(data, file_path):
    # Ensure the directory exists
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    
    try:
        with open(file_path, 'w') as f:
            yaml.dump(data, f, default_flow_style=False)
        print(f"Successfully wrote to: {file_path}")
    except Exception as e:
        print(f"Error writing to {file_path}: {e}")
        exit(1)

def extract_vscode_data(record_data):
    # Find the MCP extension in the extensions list
    mcp_extension = None
    for extension in record_data.get('extensions', []):
        if extension['name'] == 'schema.oasf.agntcy.org/features/runtime/mcp':
            mcp_extension = extension
            break
    
    if not mcp_extension:
        print("Warning: No MCP extension found in the record")
        return {}
    
    # Extract servers data from the MCP extension
    if 'data' not in mcp_extension or 'servers' not in mcp_extension['data']:
        print("Warning: No servers data found in the MCP extension")
        return {}
    
    mcp_server_data = mcp_extension['data']['servers']

    # Extract inputs data from the MCP servers
    server_inputs = {}  # Use a set to avoid duplicates
    for server_name, server_data in mcp_server_data.items():
        if 'env' in server_data:
            for env_key, env_value in server_data['env'].items():
                # Check if the value is a reference to an environment variable
                if isinstance(env_value, str) and env_value.startswith('${input:'):
                    # Extract the env var name from ${env:NAME}
                    env_name = env_value.replace('${input:', '').replace('}', '')
                    server_inputs[env_name] = {
                        'id': env_name,
                        'type': 'promptString',
                        'password': True,
                        'description': f"Secret value for {env_name}",
                    }

    # Return MCP data
    return {
        'servers': mcp_server_data,
        'inputs': list(server_inputs.values()),
    }

def extract_continue_model_data(record_data):
    # Find the model extension
    model_extension = None
    for extension in record_data.get('extensions', []):
        if extension['name'] == 'schema.oasf.agntcy.org/features/runtime/model':
            model_extension = extension
            break
    
    if not model_extension or 'models' not in model_extension['data']:
        return []

    transformed_models = []
    for model in model_extension['data']['models']:
        transformed_model = {
            'name': f"{model['provider'].title()} {model['model']}",
            'provider': model['provider'],
            'model': model['model']
        }
        
        # Add API key or base URL if present
        if 'api_key' in model:
            transformed_model['apiKey'] = model['api_key']\
                .replace(' ', '')\
                .replace('${input:', '${{secrets.')\
                .replace('}', '}}')
        if 'api_base' in model:
            transformed_model['apiBase'] = model['api_base']\
                .replace(' ', '')\
                .replace('${input:', '${{secrets.')\
                .replace('}', '}}')
        
        # Add roles if present
        if 'roles' in model:
            transformed_model['roles'] = model['roles']
        
        # Add completion options if present
        if 'completion_options' in model:
            transformed_model['defaultCompletionOptions'] = {
                'contextLength': model['completion_options'].get('context_length'),
                'maxTokens': model['completion_options'].get('max_tokens')
            }

        transformed_models.append(transformed_model)
    
    return transformed_models

def extract_continue_prompt_data(record_data):
    # Find the model extension
    model_extension = None
    for extension in record_data.get('extensions', []):
        if extension['name'] == 'schema.oasf.agntcy.org/features/runtime/prompt':
            model_extension = extension
            break
    
    if not model_extension or 'prompts' not in model_extension['data']:
        return []

    transformed_prompts = []
    for prompt in model_extension['data']['prompts']:
        transformed_prompts.append({
            'name': prompt['name'],
            'description': prompt['description'],
            'prompt': prompt['prompt']
        })
    
    return transformed_prompts

def extract_continue_mcp_data(record_data):
    # Find the MCP extension
    mcp_extension = None
    for extension in record_data.get('extensions', []):
        if extension['name'] == 'schema.oasf.agntcy.org/features/runtime/mcp':
            mcp_extension = extension
            break
    
    if not mcp_extension or 'servers' not in mcp_extension['data']:
        return []

    transformed_servers = []
    for server_name, server_data in mcp_extension['data']['servers'].items():
        transformed_server = {
            'name': server_name.title(),
            'command': server_data['command'],
            'args': server_data['args']
        }
        
        # Transform environment variables to match Continue's format
        if 'env' in server_data:
            transformed_server['env'] = {
                key: value.replace('${input:', '${{secrets.').replace('}', '}}')
                for key, value in server_data['env'].items()
            }

        transformed_servers.append(transformed_server)
    
    return transformed_servers

def extract_continue_data(record_data):
    continue_data = {}
    
    # Get the assistant name that is a valid filename
    continue_assistant_name = record_data['name'] + '-' + record_data['version']
    continue_assistant_filename = continue_assistant_name.replace(' ', '-').replace('/', '-')

    # Get basic configuration
    continue_data['name'] = continue_assistant_filename
    continue_data['version'] = record_data['version']
    continue_data['schema'] = "v1"

    # Get models data
    models = extract_continue_model_data(record_data)
    if models:
        continue_data['models'] = models
    
    # Get MCP servers data
    mcp_servers = extract_continue_mcp_data(record_data)
    if mcp_servers:
        continue_data['mcpServers'] = mcp_servers
    
    # Get prompt data
    prompt_data = extract_continue_prompt_data(record_data)
    if prompt_data:
        continue_data['prompts'] = prompt_data
    
    return continue_data

def main():
    args = parse_arguments()
    
    # Read the record JSON file
    record_data = read_json_file(args.record)

    # Write to VSCode path
    vscode_data = extract_vscode_data(record_data)
    vscode_output = Path(args.vscode_path) / 'mcp.json'
    write_json_file(vscode_data, str(vscode_output))
    
    # Write to continue path
    continue_data = extract_continue_data(record_data)
    continue_output = Path(args.continue_path) / (continue_data['name'] + '.yaml')
    write_yaml_file(continue_data, str(continue_output))

if __name__ == '__main__':
    main()

# Reusable Taskfile modules

This directory contains reusable Taskfile modules that can be imported into other Taskfiles. 
Each module is named after the functionality it provides, e.g. `Taskfile.<module-name>.yml`.

To import a module into a Taskfile, use the `includes` in the Taskfile, specifying the path to the module file. For example:

```yaml
includes:
  setup-dir-node:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"
```

## Module requirements

- Must be self-contained and reusable in different contexts. No dependencies/includes to modules outside of this directory
- Should not have hardcoded values that limit its reusability. Instead, use variables that can be overridden when importing the module.
- Should not use public vars/envs as they can cause conflicts when imported into other Taskfiles. Use vars/envs only in tasks.
- Should provide clear documentation on how to use the module, including any required variables and expected behavior.

## Available modules

- `Taskfile.setup-dir-node.yml`: Reusable module to setup a Dir node for testing purposes. Can be used as a standalone module via `task up/down` or imported into other Taskfiles.
- `Taskfile.setup-test-env.yml`: Reusable module to set up a k8s environment for testing purposes. Can be used as a standalone module via `task up/down` or imported into other Taskfiles.
- `Taskfile.deploy-chart.yml`: Reusable module to deploy a Helm chart. Can only be used as an internal module by other Taskfiles.
- `Taskfile.test-kind.yml`: Reusable module to setup a Kind cluster. Can only be used as an internal module by other Taskfiles.

## Notes on importing modules

**Variable Configuration**

When importing a module, you can override variables defined in the module by specifying them in the `vars` section of the import. For example:

```yaml
includes:
  setup-dir-node:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"
    vars:
      DIRCTL_DATA_PATH: "/custom/data/path"
```

**Taskfile commands**

When importing a module, you can decide if you want to expose imported module's tasks or override and expose your own.

If you want to expose the tasks from the imported module with a prefix, you can simply import the module:

```yaml
# my-custom-taskfile.yml
includes:
  setup-dir-node-custom:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"

# This task now exposes the `setup-dir-node-custom:up/down` tasks from the imported module
```

For example, to expose the same tasks as the imported module without the prefix, you can use the `flatten` option:

```yaml
# my-custom-taskfile.yml
includes:
  setup-dir-node:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"
    flatten: true # Expose module's tasks in the current Taskfile

# This task now exposes the `up/down` tasks from the imported module
```

If you wish to define your own tasks that only use the imported module's tasks internally, you can do so with `internal: true`. This can be useful to create higher-level tasks that orchestrate multiple tasks from the imported module without exposing them directly.

```yaml
# my-custom-taskfile.yml
includes:
  setup-dir-node:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"
    internal: true # Do not expose module's tasks directly

tasks:
  up:
    cmds:
      - echo "Setting up Dir node with custom task"
      - task: setup-dir-node:up # Use the imported module's task internally

  down:
    cmds:
      - echo "Tearing down Dir node with custom task"
      - task: setup-dir-node:down # Use the imported module's task internally
```

## Usage as testenv modules

Some of these modules can also be used as test environment modules to set up environments for tests.

The goal is that testenv modules can be used both as standalone modules via `task up/down` and as imported modules in other Taskfiles, providing flexibility in how you set up your testing environments.

```yaml
# my-import-testenv-taskfile.yml -> exposes `task up/down` from imported module
includes:
  setup-dir-node:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"
    vars:
      DIRCTL_DATA_PATH: "/custom/data/path"

# my-override-testenv-taskfile.yml -> exposes implemented `task up/down`
includes:
  setup-dir-node:
    taskfile: "tests/modules/Taskfile.setup-dir-node.yml"
    internal: true

tasks:
  up:
    cmds:
      - echo "Setting up test environment with custom task"
      - task: setup-dir-node:up # Use the imported module's task internally
    
  down:
    cmds:
      - echo "Tearing down test environment with custom task"
      - task: setup-dir-node:down # Use the imported module's task internally
```

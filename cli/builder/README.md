# Model Builder

This component is responsible for generating an agent data model from source code
based on provided build configuration.

Builder appends additional data to the agent data model via `Extensions` property.

It supports plugins which extend the build process with custom logic.
The plugins can be registered through Golang interfaces.

## Plugins

- [llmanalyzer](./plugins/llmanalyzer/) - Extract semantic details from source code
- [runtime](./plugins/runtime/) - Extract runtime details from source code
- [pyprojectparse](./plugins/pyprojectparse) - Parse and extract metadata from pyproject.toml

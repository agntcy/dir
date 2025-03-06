# Runtime extension

When the agent data model is created from the source code, it is possible to extract more information in order to describe agent's capabilities and constraints.

For example, it may be needed to extract programming language details and packages for a given agent to describe its runtime _constraints_.

Or, details about the agent's taxonomy may be needed to properly describe its _capabilities_, for example indicating whether an agent can perform network analysis or text summarization tasks.

You can read more about agent's capabilities in the [Taxonomy of tasks](https://schema.oasf.agntcy.org/?extensions=) section.

Information contained in this extension can be leveraged by a custom search service implemented in the application layer to provide more detailed search results.

## Python analyzer

The Python analyzer is a tool that extracts the python version and an SBOM (Software Bill of Materials) from an agent written in Python.
The extension looks for the following 3 files in decreasing order of priority and stops at the first one found:

### pyproject.toml (Poetry)

1. The standard **requires-python** in the **[project]** section
2. The **python** in the **[tool.poetry.dependencies]** section

### Pipfile (Pipenv)

1. The **python_version** in the **[requires]** section

### setup.py

1. Find the python version in **python_requires** with the following regex pattern:
```go
regexPattern := `python_requires\s*=\s*['"]([^'"]+)['"]`
```

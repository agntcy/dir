---
name: code-review
description: Toolkit for performing structured code reviews. Covers identifying bugs, security issues, performance problems, and style violations. Use when asked to review code changes, pull requests, or individual files.
metadata:
  author: AGNTCY Contributors
  version: 1.0.0
---

# Code Review

Perform structured code reviews that identify bugs, security issues, performance problems,
and style violations in source code.

## Decision Tree: Choosing Your Approach

```
Review request → Is it a full PR / diff?
    ├─ Yes → Read the diff, then review file-by-file
    │         ├─ Security issues → Flag as critical
    │         ├─ Bugs / logic errors → Flag as high
    │         └─ Style / naming → Flag as suggestion
    │
    └─ No (single file) → Read the file in context
        ├─ Check imports and dependencies
        ├─ Verify error handling
        └─ Look for common anti-patterns
```

## Review Checklist

1. **Correctness** — Does the code do what it claims?
2. **Security** — Are inputs validated? Are secrets handled safely?
3. **Performance** — Are there unnecessary allocations, N+1 queries, or unbounded loops?
4. **Readability** — Is the intent clear without excessive comments?
5. **Tests** — Are edge cases covered?

## Examples

- "Review this Go function for potential nil pointer dereferences"
- "Check this PR diff for SQL injection vulnerabilities"
- "Suggest improvements for readability in this Python module"

## Guidelines

- Prefer actionable feedback over vague suggestions
- Always explain *why* something is a problem, not just *what*
- Group findings by severity: critical, high, medium, suggestion
- When suggesting a fix, show a concrete code snippet

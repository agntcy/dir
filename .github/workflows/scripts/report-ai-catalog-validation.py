# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""Create one tracking issue for a failing scheduled AI Catalog validation."""

import argparse
import json
import subprocess
import tempfile
from pathlib import Path

ISSUE_TITLE = "AI Catalog well-known validation failed"
ISSUE_MARKER = "<!-- ai-catalog-validation-tracking -->"
MAX_OUTPUT = 12000


def find_tracking_issue(issues: list[dict]) -> int | None:
    for issue in issues:
        if issue.get("title") == ISSUE_TITLE or ISSUE_MARKER in (issue.get("body") or ""):
            return issue["number"]
    return None


def render_report(version: str, url: str, run_url: str, output: str) -> str:
    excerpt = output[:MAX_OUTPUT]
    if len(output) > MAX_OUTPUT:
        excerpt += "\n... validation output truncated ..."
    # A CLI message containing backticks must not close the Markdown fence.
    excerpt = excerpt.replace("```", "` ` `")
    return (
        f"{ISSUE_MARKER}\n"
        "The latest AI Catalog CLI rejected the document served by this "
        "repository's local directory daemon.\n\n"
        f"- Validator: `{version}`\n"
        f"- Well-known URL: `{url}`\n"
        f"- Workflow run: {run_url}\n\n"
        "Validation output:\n\n"
        f"```text\n{excerpt}\n```\n"
    )


def gh(*args: str) -> str:
    return subprocess.run(
        ["gh", *args], check=True, text=True, capture_output=True
    ).stdout


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--url", required=True)
    parser.add_argument("--run-url", required=True)
    parser.add_argument("--result", type=Path, required=True)
    args = parser.parse_args()

    output = args.result.read_text(encoding="utf-8", errors="replace")
    report = render_report(args.version, args.url, args.run_url, output)
    issues = json.loads(gh(
        "issue", "list", "--repo", args.repo, "--state", "open",
        "--limit", "1000", "--json", "number,title,body",
    ))
    existing = find_tracking_issue(issues)

    with tempfile.TemporaryDirectory() as temp_dir:
        body_file = Path(temp_dir) / "body.md"
        body_file.write_text(report, encoding="utf-8")
        if existing is None:
            print(gh(
                "issue", "create", "--repo", args.repo,
                "--title", ISSUE_TITLE, "--body-file", str(body_file),
            ))
        else:
            print(gh(
                "issue", "comment", str(existing), "--repo", args.repo,
                "--body-file", str(body_file),
            ))


if __name__ == "__main__":
    main()

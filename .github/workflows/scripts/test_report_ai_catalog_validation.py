# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import importlib.util
import io
import json
import sys
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).with_name("report-ai-catalog-validation.py")
SPEC = importlib.util.spec_from_file_location("report_ai_catalog_validation", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
reporter = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(reporter)


class ReportTests(unittest.TestCase):
    def test_finds_only_marked_open_issue(self):
        issues = [
            {"number": 1, "title": "unrelated", "body": "another report"},
            {"number": 2, "title": reporter.ISSUE_TITLE, "body": "older tracking issue"},
        ]
        self.assertEqual(reporter.find_tracking_issue(issues), 2)
        self.assertIsNone(reporter.find_tracking_issue(issues[:1]))
        self.assertEqual(reporter.find_tracking_issue([
            {"number": 3, "title": "renamed", "body": reporter.ISSUE_MARKER},
        ]), 3)

    def test_report_includes_evidence_and_bounds_output(self):
        output = "```\n" + "x" * (reporter.MAX_OUTPUT + 10)
        report = reporter.render_report("ai-catalog 0.4.0", "http://localhost/doc", "https://github.com/run/1", output)
        self.assertIn("ai-catalog 0.4.0", report)
        self.assertIn("http://localhost/doc", report)
        self.assertIn("https://github.com/run/1", report)
        self.assertIn("truncated", report)
        self.assertNotIn("```\n```", report)
        self.assertEqual(report.count("```"), 2)

    def test_main_creates_or_comments_without_duplicate(self):
        for existing in (None, 42):
            with self.subTest(existing=existing), tempfile.TemporaryDirectory() as temp_dir:
                result = Path(temp_dir) / "result.json"
                result.write_text('{"valid": false}', encoding="utf-8")
                writes = []

                def fake_gh(*args):
                    if args[:2] == ("issue", "list"):
                        return json.dumps([] if existing is None else [{
                            "number": existing, "title": reporter.ISSUE_TITLE, "body": "",
                        }])
                    writes.append((args, Path(args[-1]).read_text(encoding="utf-8")))
                    return "https://github.com/agntcy/dir/issues/42"

                argv = [
                    str(SCRIPT), "--repo", "agntcy/dir", "--version", "ai-catalog 0.4.0",
                    "--url", "http://localhost/doc", "--run-url", "https://github.com/run/1",
                    "--result", str(result),
                ]
                with patch.object(reporter, "gh", side_effect=fake_gh), patch.object(sys, "argv", argv), redirect_stdout(io.StringIO()):
                    reporter.main()

                self.assertEqual(len(writes), 1)
                self.assertEqual(writes[0][0][:2], ("issue", "create" if existing is None else "comment"))
                self.assertIn('{"valid": false}', writes[0][1])


if __name__ == "__main__":
    unittest.main()

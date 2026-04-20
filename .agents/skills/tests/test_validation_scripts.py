"""Tests for validation / helper scripts across skills.

Each script is invoked via subprocess so that exit-code semantics (0 = success,
1 = failure) are verified end-to-end.  Tests that depend on a script that does
not yet exist are automatically skipped.

Run with:
    python3 -m pytest .agents/skills/tests/test_validation_scripts.py -v
"""

from __future__ import annotations

import json
import os
import subprocess
import tempfile
import textwrap
import unittest
from typing import List, Optional

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------

_HERE = os.path.dirname(os.path.abspath(__file__))
_SKILLS = os.path.normpath(os.path.join(_HERE, ".."))


def _script(skill: str, name: str) -> str:
    """Return the absolute path to *skill*/scripts/*name*."""
    return os.path.join(_SKILLS, skill, "scripts", name)


def _exists(skill: str, name: str) -> bool:
    return os.path.isfile(_script(skill, name))


def _run(script_path: str, args: list[str] | None = None,
         stdin_data: str | None = None) -> subprocess.CompletedProcess:
    """Run a Python script and return the CompletedProcess."""
    cmd = ["python3", script_path] + (args or [])
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        stdin=subprocess.PIPE if stdin_data else subprocess.DEVNULL,
        input=stdin_data,
        timeout=30,
    )


# ===================================================================
# 1. validate-confluence-target.py  (confluence-changelog-publisher)
# ===================================================================

_CONFLUENCE_SCRIPT = "validate-confluence-target.py"
_CONFLUENCE_SKILL = "confluence-changelog-publisher"


@unittest.skipUnless(
    _exists(_CONFLUENCE_SKILL, _CONFLUENCE_SCRIPT),
    f"{_CONFLUENCE_SCRIPT} not found",
)
class TestValidateConfluenceTarget(unittest.TestCase):
    """validate-confluence-target.py -- validates --space, --title, --mode."""

    script = _script(_CONFLUENCE_SKILL, _CONFLUENCE_SCRIPT)

    def test_success_with_valid_args(self):
        result = _run(self.script, [
            "--space", "ENG",
            "--title", "Release Notes v1.0",
            "--mode", "append",
        ])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_missing_required_args(self):
        result = _run(self.script, [])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_invalid_mode(self):
        result = _run(self.script, [
            "--space", "ENG",
            "--title", "Notes",
            "--mode", "invalid-mode",
        ])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 2. classify-github-target.py  (github-diff-changelog-publisher)
# ===================================================================

_CLASSIFY_SCRIPT = "classify-github-target.py"
_CLASSIFY_SKILL = "github-diff-changelog-publisher"


@unittest.skipUnless(
    _exists(_CLASSIFY_SKILL, _CLASSIFY_SCRIPT),
    f"{_CLASSIFY_SCRIPT} not found",
)
class TestClassifyGithubTarget(unittest.TestCase):
    """classify-github-target.py -- classifies GitHub URL/refs."""

    script = _script(_CLASSIFY_SKILL, _CLASSIFY_SCRIPT)

    def test_success_with_github_url(self):
        result = _run(self.script, [
            "https://github.com/owner/repo/compare/v1.0.0...v1.1.0",
        ])
        self.assertEqual(result.returncode, 0, result.stderr)
        output = json.loads(result.stdout)
        self.assertIn("type", output)

    def test_success_with_refs(self):
        result = _run(self.script, ["v1.0.0", "v1.1.0"])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_no_args(self):
        result = _run(self.script, [])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 3. normalize_pr_comments.py  (github-pr-comment-triage)
# ===================================================================

_NORMALIZE_SCRIPT = "normalize_pr_comments.py"
_NORMALIZE_SKILL = "github-pr-comment-triage"


@unittest.skipUnless(
    _exists(_NORMALIZE_SKILL, _NORMALIZE_SCRIPT),
    f"{_NORMALIZE_SCRIPT} not found",
)
class TestNormalizePrComments(unittest.TestCase):
    """normalize_pr_comments.py -- normalizes PR comment JSON."""

    script = _script(_NORMALIZE_SKILL, _NORMALIZE_SCRIPT)

    def test_success_valid_json(self):
        comments = json.dumps([
            {
                "id": 1,
                "user": {"login": "reviewer"},
                "body": "Please fix the typo on line 42.",
                "path": "main.go",
                "line": 42,
            }
        ])
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        ) as f:
            f.write(comments)
            f.flush()
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertEqual(result.returncode, 0, result.stderr)
            output = json.loads(result.stdout)
            self.assertIsInstance(output, list)
        finally:
            os.unlink(tmp)

    def test_failure_invalid_json(self):
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        ) as f:
            f.write("{not valid json")
            f.flush()
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)


# ===================================================================
# 4. render_pr_reply.py  (github-pr-comment-triage)
# ===================================================================

_RENDER_SCRIPT = "render_pr_reply.py"
_RENDER_SKILL = "github-pr-comment-triage"


@unittest.skipUnless(
    _exists(_RENDER_SKILL, _RENDER_SCRIPT),
    f"{_RENDER_SCRIPT} not found",
)
class TestRenderPrReply(unittest.TestCase):
    """render_pr_reply.py -- renders PR reply text."""

    script = _script(_RENDER_SKILL, _RENDER_SCRIPT)

    def test_success_valid_input(self):
        payload = json.dumps({
            "comment_id": 1,
            "action": "resolved",
            "message": "Fixed the typo.",
        })
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        ) as f:
            f.write(payload)
            f.flush()
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertTrue(len(result.stdout.strip()) > 0)
        finally:
            os.unlink(tmp)

    def test_failure_missing_input(self):
        result = _run(self.script, [])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 5. validate-publication-flow.py  (github-release-publication-flow)
# ===================================================================

_PUBFLOW_SCRIPT = "validate-publication-flow.py"
_PUBFLOW_SKILL = "github-release-publication-flow"


@unittest.skipUnless(
    _exists(_PUBFLOW_SKILL, _PUBFLOW_SCRIPT),
    f"{_PUBFLOW_SCRIPT} not found",
)
class TestValidatePublicationFlow(unittest.TestCase):
    """validate-publication-flow.py -- validates --target, --destination."""

    script = _script(_PUBFLOW_SKILL, _PUBFLOW_SCRIPT)

    def test_success_valid_args(self):
        result = _run(self.script, [
            "--target", "v1.2.0",
            "--destination", "github",
        ])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_missing_target(self):
        result = _run(self.script, ["--destination", "github"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_missing_destination(self):
        result = _run(self.script, ["--target", "v1.2.0"])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 6. validate-task-bundle.py  (jira-tasks)
# ===================================================================

_TASKBUNDLE_SCRIPT = "validate-task-bundle.py"
_TASKBUNDLE_SKILL = "jira-tasks"


@unittest.skipUnless(
    _exists(_TASKBUNDLE_SKILL, _TASKBUNDLE_SCRIPT),
    f"{_TASKBUNDLE_SCRIPT} not found",
)
class TestValidateTaskBundle(unittest.TestCase):
    """validate-task-bundle.py -- validates task bundle directory."""

    script = _script(_TASKBUNDLE_SKILL, _TASKBUNDLE_SCRIPT)

    def test_success_valid_bundle(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            tasks_file = os.path.join(tmpdir, "tasks.md")
            with open(tasks_file, "w") as f:
                f.write(textwrap.dedent("""\
                    # Tasks
                    - [ ] RF-01: Implement login endpoint
                    - [ ] RF-02: Add JWT validation
                """))
            result = _run(self.script, ["--dir", tmpdir])
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_empty_directory(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            result = _run(self.script, ["--dir", tmpdir])
            self.assertNotEqual(result.returncode, 0)

    def test_failure_nonexistent_directory(self):
        result = _run(self.script, ["--dir", "/tmp/nonexistent-task-bundle-xyz"])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 7. validate-dashboard.py  (otel-grafana-dashboards)
# ===================================================================

_DASHBOARD_SCRIPT = "validate-dashboard.py"
_DASHBOARD_SKILL = "otel-grafana-dashboards"


@unittest.skipUnless(
    _exists(_DASHBOARD_SKILL, _DASHBOARD_SCRIPT),
    f"{_DASHBOARD_SCRIPT} not found",
)
class TestValidateDashboard(unittest.TestCase):
    """validate-dashboard.py -- validates Grafana dashboard JSON."""

    script = _script(_DASHBOARD_SKILL, _DASHBOARD_SCRIPT)

    def _write_json(self, data: dict) -> str:
        f = tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        )
        json.dump(data, f)
        f.close()
        return f.name

    def test_success_valid_dashboard(self):
        dashboard = {
            "uid": "abc123",
            "title": "My Service Dashboard",
            "panels": [
                {
                    "type": "graph",
                    "title": "Request Rate",
                    "targets": [
                        {"expr": "rate(http_requests_total[5m])"}
                    ],
                }
            ],
            "schemaVersion": 30,
        }
        tmp = self._write_json(dashboard)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertEqual(result.returncode, 0, result.stderr)
        finally:
            os.unlink(tmp)

    def test_failure_missing_title(self):
        dashboard = {"panels": [], "schemaVersion": 30}
        tmp = self._write_json(dashboard)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)

    def test_failure_invalid_json_file(self):
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        ) as f:
            f.write("not json at all")
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)


# ===================================================================
# 8. validate-collection.py  (postman-collection-generator)
# ===================================================================

_COLLECTION_SCRIPT = "validate-collection.py"
_COLLECTION_SKILL = "postman-collection-generator"


@unittest.skipUnless(
    _exists(_COLLECTION_SKILL, _COLLECTION_SCRIPT),
    f"{_COLLECTION_SCRIPT} not found",
)
class TestValidateCollection(unittest.TestCase):
    """validate-collection.py -- validates Postman collection JSON."""

    script = _script(_COLLECTION_SKILL, _COLLECTION_SCRIPT)

    def _write_json(self, data: dict) -> str:
        f = tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        )
        json.dump(data, f)
        f.close()
        return f.name

    def test_success_valid_collection(self):
        collection = {
            "info": {
                "name": "My API",
                "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
            },
            "item": [
                {
                    "name": "Get Users",
                    "request": {
                        "method": "GET",
                        "url": "{{base_url}}/users",
                    },
                }
            ],
        }
        tmp = self._write_json(collection)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertEqual(result.returncode, 0, result.stderr)
        finally:
            os.unlink(tmp)

    def test_failure_missing_info(self):
        collection = {"item": []}
        tmp = self._write_json(collection)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)

    def test_failure_not_json(self):
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        ) as f:
            f.write("---\nyaml: true\n")
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)


# ===================================================================
# 9. validate-prompt.py  (prompt-enricher)
# ===================================================================

_PROMPT_SCRIPT = "validate-prompt.py"
_PROMPT_SKILL = "prompt-enricher"


@unittest.skipUnless(
    _exists(_PROMPT_SKILL, _PROMPT_SCRIPT),
    f"{_PROMPT_SCRIPT} not found",
)
class TestValidatePrompt(unittest.TestCase):
    """validate-prompt.py -- validates prompt files."""

    script = _script(_PROMPT_SKILL, _PROMPT_SCRIPT)

    def test_success_valid_prompt_file(self):
        content = textwrap.dedent("""\
            ---
            name: test-prompt
            version: 1.0.0
            description: A test prompt for validation.
            ---

            # Test Prompt

            You are a helpful assistant.
        """)
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".md", delete=False
        ) as f:
            f.write(content)
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertEqual(result.returncode, 0, result.stderr)
        finally:
            os.unlink(tmp)

    def test_failure_empty_file(self):
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".md", delete=False
        ) as f:
            f.write("")
            tmp = f.name
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)


# ===================================================================
# 10. resolve_pr_base.py  (pull-request)
# ===================================================================

_PRBASE_SCRIPT = "resolve_pr_base.py"
_PRBASE_SKILL = "pull-request"


@unittest.skipUnless(
    _exists(_PRBASE_SKILL, _PRBASE_SCRIPT),
    f"{_PRBASE_SCRIPT} not found",
)
class TestResolvePrBase(unittest.TestCase):
    """resolve_pr_base.py -- resolves PR base branch."""

    script = _script(_PRBASE_SKILL, _PRBASE_SCRIPT)

    def test_success_explicit_base(self):
        result = _run(self.script, ["--base", "main"])
        self.assertEqual(result.returncode, 0, result.stderr)
        output = result.stdout.strip()
        self.assertIn("main", output)

    def test_failure_invalid_ref(self):
        result = _run(self.script, [
            "--base", "nonexistent-branch-xyz-999",
        ])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 11. validate-commit-header.py  (semantic-commit)
# ===================================================================

_COMMIT_SCRIPT = "validate-commit-header.py"
_COMMIT_SKILL = "semantic-commit"


@unittest.skipUnless(
    _exists(_COMMIT_SKILL, _COMMIT_SCRIPT),
    f"{_COMMIT_SCRIPT} not found",
)
class TestValidateCommitHeader(unittest.TestCase):
    """validate-commit-header.py -- validates commit header format."""

    script = _script(_COMMIT_SKILL, _COMMIT_SCRIPT)

    def test_success_feat(self):
        result = _run(self.script, ["--header", "feat(auth): add JWT validation"])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_success_fix_no_scope(self):
        result = _run(self.script, ["--header", "fix: correct null pointer on startup"])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_success_breaking_change(self):
        result = _run(self.script, [
            "--header", "feat(api)!: remove deprecated endpoints",
        ])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_invalid_type(self):
        result = _run(self.script, ["--header", "foobar: something"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_empty_description(self):
        result = _run(self.script, ["--header", "feat:"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_no_colon(self):
        result = _run(self.script, ["--header", "this is not conventional"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_too_long(self):
        header = "feat(scope): " + "x" * 200
        result = _run(self.script, ["--header", header])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 12. validate-issue-key.py  (us-to-prd)
# ===================================================================

_ISSUEKEY_SCRIPT = "validate-issue-key.py"
_ISSUEKEY_SKILL = "us-to-prd"


@unittest.skipUnless(
    _exists(_ISSUEKEY_SKILL, _ISSUEKEY_SCRIPT),
    f"{_ISSUEKEY_SCRIPT} not found",
)
class TestValidateIssueKey(unittest.TestCase):
    """validate-issue-key.py -- validates JIRA issue key format."""

    script = _script(_ISSUEKEY_SKILL, _ISSUEKEY_SCRIPT)

    def test_success_standard_key(self):
        result = _run(self.script, ["--key", "PROJ-123"])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_success_long_project(self):
        result = _run(self.script, ["--key", "MYPROJECT-9999"])
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failure_missing_number(self):
        result = _run(self.script, ["--key", "PROJ-"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_lowercase_project(self):
        result = _run(self.script, ["--key", "proj-123"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_no_dash(self):
        result = _run(self.script, ["--key", "PROJ123"])
        self.assertNotEqual(result.returncode, 0)

    def test_failure_empty(self):
        result = _run(self.script, ["--key", ""])
        self.assertNotEqual(result.returncode, 0)


# ===================================================================
# 13. validate-bug-input.py  (bugfix)
# ===================================================================

_BUGINPUT_SCRIPT = "validate-bug-input.py"
_BUGINPUT_SKILL = "bugfix"


@unittest.skipUnless(
    _exists(_BUGINPUT_SKILL, _BUGINPUT_SCRIPT),
    f"{_BUGINPUT_SCRIPT} not found",
)
class TestValidateBugInput(unittest.TestCase):
    """validate-bug-input.py -- validates bug list JSON."""

    script = _script(_BUGINPUT_SKILL, _BUGINPUT_SCRIPT)

    def _write_json(self, data):
        f = tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False
        )
        json.dump(data, f)
        f.close()
        return f.name

    def test_success_valid_bugs(self):
        bugs = [
            {
                "id": "BUG-001",
                "severity": "critical",
                "file": "main.go",
                "line": 42,
                "reproduction": "run make test",
                "expected": "tests pass",
                "actual": "nil pointer",
            }
        ]
        tmp = self._write_json(bugs)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("SUCCESS", result.stdout)
        finally:
            os.unlink(tmp)

    def test_failure_missing_fields(self):
        bugs = [{"id": "BUG-001", "severity": "critical"}]
        tmp = self._write_json(bugs)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)

    def test_failure_invalid_severity(self):
        bugs = [
            {
                "id": "BUG-001",
                "severity": "low",
                "file": "main.go",
                "line": 42,
                "reproduction": "run test",
                "expected": "pass",
                "actual": "fail",
            }
        ]
        tmp = self._write_json(bugs)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)

    def test_failure_invalid_id_format(self):
        bugs = [
            {
                "id": "ISSUE-1",
                "severity": "minor",
                "file": "main.go",
                "line": 10,
                "reproduction": "steps",
                "expected": "ok",
                "actual": "not ok",
            }
        ]
        tmp = self._write_json(bugs)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)

    def test_failure_empty_list(self):
        tmp = self._write_json([])
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)

    def test_failure_extra_fields(self):
        bugs = [
            {
                "id": "BUG-001",
                "severity": "major",
                "file": "main.go",
                "line": 5,
                "reproduction": "run",
                "expected": "ok",
                "actual": "fail",
                "extra_field": "not allowed",
            }
        ]
        tmp = self._write_json(bugs)
        try:
            result = _run(self.script, ["--input", tmp])
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(tmp)


# ---------------------------------------------------------------------------
# Allow running directly: python3 test_validation_scripts.py
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    unittest.main()

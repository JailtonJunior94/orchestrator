#!/usr/bin/env python3
import argparse
import json
import re
import sys


def classify_branch(branch: str) -> dict:
    patterns = [
        (r"^(feat|feature|fix|bugfix|refactor|chore|docs|test|perf|style|build|ci)/", ["develop", "main"], "feature-like branch"),
        (r"^(release/.+|release-candidate)$", ["main"], "release branch"),
        (r"^hotfix/", ["main"], "hotfix branch"),
    ]

    for pattern, candidates, reason in patterns:
        if re.match(pattern, branch):
            return {
                "status": "ok",
                "branch": branch,
                "candidates": candidates,
                "reason": reason,
                "confidence": "medium",
            }

    return {
        "status": "ambiguous",
        "branch": branch,
        "candidates": [],
        "reason": "branch name does not match known conventions",
        "confidence": "low",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Resolve base branch candidates from a branch name.")
    parser.add_argument("--branch", required=True, help="Current branch name")
    args = parser.parse_args()

    branch = args.branch.strip()
    if not branch or branch == "HEAD":
        print("invalid branch name", file=sys.stderr)
        return 1

    result = classify_branch(branch)
    print(json.dumps(result, ensure_ascii=False))
    return 0 if result["status"] == "ok" else 2


if __name__ == "__main__":
    raise SystemExit(main())

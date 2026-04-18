#!/usr/bin/env python3
import re
import sys


ALLOWED_TYPES = (
    "feat",
    "fix",
    "refactor",
    "perf",
    "docs",
    "test",
    "chore",
    "build",
    "ci",
    "style",
)

HEADER_RE = re.compile(
    r"^(?P<type>feat|fix|refactor|perf|docs|test|chore|build|ci|style)"
    r"(?P<breaking>!)?"
    r"(?:\((?P<scope>[a-z0-9][a-z0-9-]*)\))?"
    r": (?P<description>.+)$"
)


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: validate-commit-header.py '<type>(scope): description>'", file=sys.stderr)
        return 2

    header = sys.argv[1].strip()
    match = HEADER_RE.match(header)
    if not match:
        print("invalid header: expected Conventional Commit format", file=sys.stderr)
        return 1

    commit_type = match.group("type")
    description = match.group("description").strip()

    if commit_type not in ALLOWED_TYPES:
        print(f"invalid type: {commit_type}", file=sys.stderr)
        return 1

    if not description:
        print("invalid description: empty", file=sys.stderr)
        return 1

    if description.lower() in {"update", "ajustes", "ajustes gerais", "melhorias", "changes"}:
        print("invalid description: too vague", file=sys.stderr)
        return 1

    print(header)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
import re
import sys


ISSUE_KEY_RE = re.compile(r"^[A-Z][A-Z0-9]*-\d+$")


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: validate-issue-key.py <ISSUE_KEY>", file=sys.stderr)
        return 2

    issue_key = sys.argv[1].strip()
    if not ISSUE_KEY_RE.match(issue_key):
        print("invalid issue key: expected format PROJ-123", file=sys.stderr)
        return 1

    print(issue_key)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

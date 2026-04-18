#!/usr/bin/env python3
import argparse
import json
import re
import sys


DESTINATIONS = {"github", "confluence", "draft-only"}


def fail(message: str) -> int:
    print(message, file=sys.stderr)
    return 1


def main() -> int:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--target", required=True)
    parser.add_argument("--destination", required=True)
    args = parser.parse_args()

    target = args.target.strip()
    destination = args.destination.strip()

    if len(target) < 3:
        return fail("ERRO DE VALIDACAO: 'target' deve ter pelo menos 3 caracteres.")

    if destination not in DESTINATIONS:
        return fail("ERRO DE VALIDACAO: 'destination' deve ser github, confluence ou draft-only.")

    looks_like_supported_target = any(
        token in target.lower()
        for token in ["github.com", "release", "pull", "pr ", "branch", "compare", "..."]
    ) or bool(re.match(r"^[^/]+/[^/]+\s+\S+", target))

    if not looks_like_supported_target:
        return fail("ERRO DE VALIDACAO: informe uma origem GitHub suportada, como release, PR, branch ou compare.")

    print(
        json.dumps(
            {"target": target, "destination": destination, "status": "ok"},
            ensure_ascii=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

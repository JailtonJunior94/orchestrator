#!/usr/bin/env python3
"""Valida metadata da skill de forma autonoma."""
from __future__ import annotations

import argparse
import re
import sys


FORBIDDEN = {"i", "me", "my", "we", "our", "you", "your"}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--name", required=True)
    parser.add_argument("--description", required=True)
    args = parser.parse_args()

    errors: list[str] = []
    if not (1 <= len(args.name) <= 64):
        errors.append("NAME ERROR: name deve ter entre 1 e 64 caracteres.")
    if not re.match(r"^[a-z0-9]+(-[a-z0-9]+)*$", args.name):
        errors.append("NAME ERROR: usar apenas lowercase, numeros e hifens simples.")
    if len(args.description) > 1024:
        errors.append("DESCRIPTION ERROR: description deve ter no maximo 1024 caracteres.")

    found = FORBIDDEN & set(re.findall(r"\b\w+\b", args.description.lower()))
    if found:
        errors.append(f"STYLE ERROR: description contem termos proibidos: {sorted(found)}")

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    print("SUCCESS: metadata valida.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

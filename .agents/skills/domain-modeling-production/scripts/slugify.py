#!/usr/bin/env python3
"""Normaliza texto para slug kebab-case ASCII."""
from __future__ import annotations

import re
import sys
import unicodedata


def slugify(value: str) -> str:
    normalized = unicodedata.normalize("NFKD", value)
    ascii_text = normalized.encode("ascii", "ignore").decode("ascii")
    lowered = ascii_text.lower()
    replaced = re.sub(r"[^a-z0-9]+", "-", lowered)
    return re.sub(r"-{2,}", "-", replaced).strip("-")


def main() -> int:
    if len(sys.argv) != 2:
        print("USO: slugify.py <texto>", file=sys.stderr)
        return 2

    slug = slugify(sys.argv[1])
    if not slug:
        print("SLUG VAZIO: informe um titulo com letras ou numeros.", file=sys.stderr)
        return 1

    print(slug)
    return 0


if __name__ == "__main__":
    sys.exit(main())

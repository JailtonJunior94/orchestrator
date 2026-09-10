#!/usr/bin/env python3
"""Inicializa a estrutura do bundle de modelagem de dominio.

Uso:
    python3 init-bundle.py <slug> [--root <diretorio-raiz>]
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

SLUG_PATTERN = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def read_asset(script_dir: Path, name: str) -> str:
    asset_path = script_dir.parent / "assets" / name
    return asset_path.read_text(encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("slug")
    parser.add_argument("--root", default=".", help="Diretorio raiz (default: cwd)")
    args = parser.parse_args()

    slug = args.slug.strip()
    if not SLUG_PATTERN.match(slug):
        print(
            f"SLUG INVALIDO: '{slug}'. Use kebab-case lowercase com letras, numeros e hifen.",
            file=sys.stderr,
        )
        return 1

    root = Path(args.root).resolve()
    bundle_dir = root / "discoveries" / f"domain-{slug}"
    if bundle_dir.exists():
        print(f"DIRETORIO JA EXISTE: {bundle_dir}", file=sys.stderr)
        return 1

    script_dir = Path(__file__).resolve().parent
    bundle_dir.mkdir(parents=True, exist_ok=False)

    bundle = json.loads(read_asset(script_dir, "bundle-template.json"))
    bundle["slug"] = slug
    bundle["created_at"] = utc_now_iso()

    (bundle_dir / "bundle.json").write_text(
        json.dumps(bundle, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (bundle_dir / "domain-model.md").write_text(
        read_asset(script_dir, "domain-model-template.md"),
        encoding="utf-8",
    )
    (bundle_dir / "transcript.md").write_text(
        read_asset(script_dir, "transcript-template.md"),
        encoding="utf-8",
    )

    print(str(bundle_dir))
    return 0


if __name__ == "__main__":
    sys.exit(main())

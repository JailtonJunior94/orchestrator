#!/usr/bin/env python3
"""Inicializa bundle canonico de decisao de design pattern."""
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


def read_asset(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("slug")
    parser.add_argument("--root", default=".", help="Diretorio raiz do projeto")
    parser.add_argument("--title", default="", help="Titulo inicial do bundle")
    parser.add_argument(
        "--context-mode",
        choices=["codebase", "greenfield"],
        default="codebase",
        help="Modo do contexto",
    )
    args = parser.parse_args()

    slug = args.slug.strip()
    if not SLUG_PATTERN.match(slug):
        print(f"SLUG INVALIDO: '{slug}'. Use kebab-case lowercase.", file=sys.stderr)
        return 1

    root = Path(args.root).resolve()
    skill_root = Path(__file__).resolve().parents[1]
    bundle_dir = root / "pattern-decisions" / slug
    if bundle_dir.exists():
        print(f"DIRETORIO JA EXISTE: {bundle_dir}", file=sys.stderr)
        return 1

    bundle_dir.mkdir(parents=True, exist_ok=False)

    bundle = json.loads(read_asset(skill_root / "assets" / "bundle-template.json"))
    bundle["slug"] = slug
    bundle["title"] = args.title.strip()
    bundle["created_at"] = utc_now_iso()
    bundle["context_mode"] = args.context_mode
    bundle["decision_bundle"]["title"] = args.title.strip()

    (bundle_dir / "bundle.json").write_text(
        json.dumps(bundle, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (bundle_dir / "decision.md").write_text(
        read_asset(skill_root / "assets" / "pattern-decision-template.md"),
        encoding="utf-8",
    )
    (bundle_dir / "implementation.md").write_text(
        read_asset(skill_root / "assets" / "pattern-implementation-template.md"),
        encoding="utf-8",
    )
    (bundle_dir / "transcript.md").write_text(
        read_asset(skill_root / "assets" / "transcript-template.md"),
        encoding="utf-8",
    )
    (bundle_dir / "selector-input.json").write_text(
        read_asset(skill_root / "assets" / "select-pattern-input.example.json"),
        encoding="utf-8",
    )
    (bundle_dir / "selector-output.json").write_text("{}\n", encoding="utf-8")

    print(str(bundle_dir))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

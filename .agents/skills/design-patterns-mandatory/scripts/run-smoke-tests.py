#!/usr/bin/env python3
"""Executa smoke tests determinísticos da skill."""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SELECTOR = ROOT / "scripts" / "select_pattern.py"
VALIDATOR = ROOT / "scripts" / "validate_pattern_bundle.py"
INIT_BUNDLE = ROOT / "scripts" / "init-bundle.py"


def run_command(args: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, text=True, capture_output=True, cwd=ROOT)


def expect_selector(asset_name: str, expected_status: str, expected_pattern: str | None) -> list[str]:
    result = run_command([sys.executable, str(SELECTOR), "--input", str(ROOT / "assets" / asset_name)])
    errors: list[str] = []
    if result.returncode != 0:
        errors.append(f"{asset_name}: seletor retornou code {result.returncode}: {result.stderr.strip()}")
        return errors
    try:
        payload = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        errors.append(f"{asset_name}: seletor retornou JSON invalido: {exc}")
        return errors
    if payload.get("status") != expected_status:
        errors.append(f"{asset_name}: status esperado {expected_status}, obtido {payload.get('status')}")
    pattern = (payload.get("primary_pattern") or {}).get("pattern")
    if expected_pattern != pattern:
        errors.append(f"{asset_name}: pattern esperado {expected_pattern}, obtido {pattern}")
    return errors


def expect_validator(asset_name: str) -> list[str]:
    result = run_command([sys.executable, str(VALIDATOR), str(ROOT / "assets" / asset_name)])
    if result.returncode != 0:
        return [f"{asset_name}: bundle deveria validar, mas falhou: {result.stderr.strip()}"]
    if "SUCCESS" not in result.stdout:
        return [f"{asset_name}: bundle nao retornou SUCCESS"]
    return []


def expect_bundle_directory() -> list[str]:
    tmp_root = Path("/tmp/design-pattern-skill-smoke")
    if tmp_root.exists():
        subprocess.run(["rm", "-rf", str(tmp_root)], check=False)
    tmp_root.mkdir(parents=True, exist_ok=True)

    init = run_command([sys.executable, str(INIT_BUNDLE), "pricing-strategy", "--root", str(tmp_root), "--title", "Pricing Strategy"])
    if init.returncode != 0:
        return [f"init-bundle falhou: {init.stderr.strip()}"]

    bundle_dir = Path(init.stdout.strip())
    (bundle_dir / "decision.md").write_text((ROOT / "assets" / "pattern-bundle-valid.md").read_text(encoding="utf-8"), encoding="utf-8")
    (bundle_dir / "implementation.md").write_text((ROOT / "assets" / "pattern-implementation-valid.md").read_text(encoding="utf-8"), encoding="utf-8")
    (bundle_dir / "selector-output.json").write_text((ROOT / "assets" / "select-pattern-output.example.json").read_text(encoding="utf-8"), encoding="utf-8")

    bundle_json = json.loads((bundle_dir / "bundle.json").read_text(encoding="utf-8"))
    bundle_json["primary_pattern"] = "Strategy"
    bundle_json["rejected_patterns"] = ["State"]
    bundle_json["status"] = "done"
    bundle_json["readiness"]["status"] = "done"
    bundle_json["readiness"]["blockers"] = []
    (bundle_dir / "bundle.json").write_text(json.dumps(bundle_json, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    result = run_command([sys.executable, str(VALIDATOR), str(bundle_dir)])
    if result.returncode != 0:
        return [f"bundle_dir: deveria validar, mas falhou: {result.stderr.strip()}"]
    return []


def main() -> int:
    errors: list[str] = []
    errors.extend(expect_selector("select-pattern-strategy.json", "ok", "Strategy"))
    errors.extend(expect_selector("select-pattern-ambiguous.json", "ambiguous", None))
    errors.extend(expect_selector("select-pattern-reject.json", "reject", None))
    errors.extend(expect_selector("select-pattern-singleton-reject.json", "reject", None))
    errors.extend(expect_selector("select-pattern-adapter-proxy-ambiguous.json", "ambiguous", None))
    errors.extend(expect_selector("select-pattern-facade-proxy-ambiguous.json", "ambiguous", None))
    errors.extend(expect_selector("select-pattern-adapter-reject.json", "reject", None))
    errors.extend(expect_selector("select-pattern-composite-reject.json", "ok", "Iterator"))
    errors.extend(expect_validator("pattern-bundle-valid.md"))
    errors.extend(expect_bundle_directory())

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    print("SUCCESS: smoke tests da skill passaram.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

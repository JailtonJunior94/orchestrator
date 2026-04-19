#!/usr/bin/env python3
"""
Verificacao reproduzivel do validate-prompt.py.
Executa tres fixtures e valida os codigos de saida esperados.
"""
import subprocess
import sys
import tempfile
import os

SCRIPT = os.path.join(os.path.dirname(__file__), "..", "scripts", "validate-prompt.py")

FIXTURES = [
    {
        "name": "prompt_valido",
        "content": (
            "[PAPEL] Atue como engenheiro de dados.\n"
            "[OBJETIVO] Extrair registros do banco e consolidar no data lake.\n"
            "[RESTRICOES] Nao exceda 500 tokens. Evite joins desnecessarios.\n"
            "[SAIDA] JSON com campos: id, timestamp, valor.\n"
        ),
        "expect_exit": 0,
    },
    {
        "name": "secoes_ausentes",
        "content": "Reescreva este texto de forma mais clara.",
        "expect_exit": 1,
    },
    {
        "name": "diretiva_vaga",
        "content": (
            "[OBJETIVO] Melhorar o pipeline.\n"
            "[RESTRICOES] Sem restricoes.\n"
            "[SAIDA] JSON. Do your best.\n"
        ),
        "expect_exit": 0,
    },
]


def run_fixture(fixture: dict) -> bool:
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".md", delete=False, encoding="utf-8"
    ) as f:
        f.write(fixture["content"])
        path = f.name

    try:
        result = subprocess.run(
            [sys.executable, SCRIPT, path],
            capture_output=True,
            text=True,
        )
    finally:
        os.unlink(path)

    passed = result.returncode == fixture["expect_exit"]
    status = "OK" if passed else "FALHA"
    print(f"[{status}] {fixture['name']} (esperado={fixture['expect_exit']}, obtido={result.returncode})")
    if not passed:
        if result.stdout:
            print("  stdout:", result.stdout.strip())
        if result.stderr:
            print("  stderr:", result.stderr.strip())
    return passed


def main() -> int:
    results = [run_fixture(f) for f in FIXTURES]
    total = len(results)
    passed = sum(results)
    print(f"\n{passed}/{total} fixtures passaram.")
    return 0 if passed == total else 1


if __name__ == "__main__":
    raise SystemExit(main())

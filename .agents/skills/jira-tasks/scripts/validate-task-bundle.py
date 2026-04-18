#!/usr/bin/env python3
import argparse
import pathlib
import re
import sys


TASK_FILE_RE = re.compile(r"^(?P<order>\d+)_task\.md$")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Valida a estrutura mínima de um diretório de tasks."
    )
    parser.add_argument("bundle_dir", help="Diretório com tasks.md e arquivos [num]_task.md")
    args = parser.parse_args()

    bundle_dir = pathlib.Path(args.bundle_dir)
    if not bundle_dir.exists():
        print(f"ERRO: diretório não encontrado: {bundle_dir}", file=sys.stderr)
        return 1

    if not bundle_dir.is_dir():
        print(f"ERRO: caminho não é diretório: {bundle_dir}", file=sys.stderr)
        return 1

    index_file = bundle_dir / "tasks.md"
    if not index_file.exists():
        print(f"ERRO: arquivo obrigatório ausente: {index_file}", file=sys.stderr)
        return 1
    if index_file.stat().st_size == 0:
        print(f"ERRO: arquivo vazio: {index_file}", file=sys.stderr)
        return 1

    task_files = sorted(bundle_dir.glob("*_task.md"))
    numbered_task_files = []
    duplicate_numbers = set()
    seen_numbers = set()

    for path in task_files:
        match = TASK_FILE_RE.match(path.name)
        if not match:
            continue
        order = int(match.group("order"))
        if order in seen_numbers:
            duplicate_numbers.add(order)
        seen_numbers.add(order)
        numbered_task_files.append((order, path))

    if not numbered_task_files:
        print(
            f"ERRO: nenhum arquivo no formato [num]_task.md encontrado em {bundle_dir}",
            file=sys.stderr,
        )
        return 1
    if duplicate_numbers:
        duplicate_list = ", ".join(str(number) for number in sorted(duplicate_numbers))
        print(f"ERRO: numeração duplicada encontrada: {duplicate_list}", file=sys.stderr)
        return 1

    empty_task_files = [path.name for _, path in numbered_task_files if path.stat().st_size == 0]
    if empty_task_files:
        print(
            "ERRO: arquivos de task vazios encontrados: "
            + ", ".join(sorted(empty_task_files)),
            file=sys.stderr,
        )
        return 1

    ordered_numbers = [order for order, _ in sorted(numbered_task_files)]
    expected_numbers = list(range(ordered_numbers[0], ordered_numbers[-1] + 1))
    missing_numbers = [number for number in expected_numbers if number not in seen_numbers]
    if missing_numbers:
        print(
            "ERRO: sequência de tasks incompleta; faltam os prefixos: "
            + ", ".join(str(number) for number in missing_numbers),
            file=sys.stderr,
        )
        return 1

    print(f"OK: diretório validado: {bundle_dir}")
    print(f"OK: índice encontrado: {index_file.name}")
    print(f"OK: tasks detalhadas válidas: {len(numbered_task_files)}")
    print("OK: tasks detalhadas encontradas:")
    for _, path in sorted(numbered_task_files):
        print(f"- {path.name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
import argparse
import json
import re
import sys


SPACE_RE = re.compile(r"^[A-Z0-9][A-Z0-9_-]{1,49}$")
MODES = {"create", "update", "decide-after-search"}


def fail(message: str) -> int:
    print(message, file=sys.stderr)
    return 1


def main() -> int:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--space", required=True)
    parser.add_argument("--title", required=True)
    parser.add_argument("--mode", required=True)
    parser.add_argument("--page-id")
    parser.add_argument("--parent-title")
    parser.add_argument("--root", action="store_true")
    args = parser.parse_args()

    space = args.space.strip()
    title = args.title.strip()
    mode = args.mode.strip()
    page_id = (args.page_id or "").strip()
    parent_title = (args.parent_title or "").strip()

    if not SPACE_RE.match(space):
        return fail("ERRO DE VALIDACAO: 'space' deve usar letras maiusculas, numeros, '_' ou '-'.")

    if len(title) < 3:
        return fail("ERRO DE VALIDACAO: 'title' deve ter pelo menos 3 caracteres.")

    if mode not in MODES:
        return fail("ERRO DE VALIDACAO: 'mode' deve ser create, update ou decide-after-search.")

    if sum(bool(x) for x in [page_id, parent_title, args.root]) > 1:
        return fail("ERRO DE VALIDACAO: informe apenas um localizador entre --page-id, --parent-title ou --root.")

    if mode == "update" and not page_id:
        return fail("ERRO DE VALIDACAO: modo update exige --page-id ou URL previamente resolvida.")

    result = {
        "space": space,
        "title": title,
        "mode": mode,
        "page_id": page_id or None,
        "parent_title": parent_title or None,
        "root": args.root,
        "status": "ok",
    }
    print(json.dumps(result, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

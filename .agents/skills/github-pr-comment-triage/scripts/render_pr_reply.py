#!/usr/bin/env python3
import argparse
import json
import sys
from typing import Any, Dict


def load_item(path: str) -> Dict[str, Any]:
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def approved_reply(item: Dict[str, Any], change_summary: str, how: str, validation: str) -> str:
    lines = [
        f"Obrigado pelo comentário sobre `{item['item_id']}`.",
        "",
        "Ajuste aplicado.",
        f"- Resumo: {change_summary}",
        f"- Como foi feito: {how}",
    ]
    if validation:
        lines.append(f"- Validação: {validation}")
    if item.get("path"):
        location = f"`{item['path']}`"
        if item.get("line"):
            location += f":{item['line']}"
        lines.append(f"- Contexto original: {location}")
    return "\n".join(lines).strip() + "\n"


def rejected_reply(item: Dict[str, Any], reason: str) -> str:
    lines = [
        f"Obrigado pelo comentário sobre `{item['item_id']}`.",
        "",
        "A sugestão não foi implementada nesta rodada.",
        f"- Motivo: {reason}",
        "- Decisão: manutenção da implementação atual com rastreabilidade deste retorno.",
    ]
    if item.get("path"):
        location = f"`{item['path']}`"
        if item.get("line"):
            location += f":{item['line']}"
        lines.append(f"- Contexto original: {location}")
    return "\n".join(lines).strip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description="Gera resposta padronizada em pt-BR para comentario de PR.")
    parser.add_argument("--decision", required=True, choices=["approved", "rejected"])
    parser.add_argument("--item", required=True, help="Arquivo JSON de um item normalizado.")
    parser.add_argument("--change-summary", default="", help="Resumo do ajuste aplicado.")
    parser.add_argument("--how", default="", help="Como o ajuste foi implementado.")
    parser.add_argument("--validation", default="", help="Validacao executada.")
    parser.add_argument("--reason", default="", help="Motivo da rejeicao.")
    args = parser.parse_args()

    try:
        item = load_item(args.item)
    except FileNotFoundError:
        print(f"INPUT ERROR: arquivo nao encontrado: {args.item}", file=sys.stderr)
        return 1
    except json.JSONDecodeError as exc:
        print(f"INPUT ERROR: JSON invalido: {exc}", file=sys.stderr)
        return 1

    if args.decision == "approved":
        if not args.change_summary or not args.how:
            print("ARG ERROR: --change-summary e --how sao obrigatorios para approved.", file=sys.stderr)
            return 1
        body = approved_reply(item, args.change_summary, args.how, args.validation)
    else:
        if not args.reason:
            print("ARG ERROR: --reason e obrigatorio para rejected.", file=sys.stderr)
            return 1
        body = rejected_reply(item, args.reason)

    sys.stdout.write(body)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

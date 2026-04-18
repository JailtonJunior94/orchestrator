#!/usr/bin/env python3
import argparse
import json
import re
import sys
from typing import Any, Dict, List, Optional, Tuple


CLASSIFICATION_RULES: List[Tuple[str, List[str]]] = [
    ("bug", ["bug", "broken", "erro", "falha", "quebra", "incorrect", "wrong", "fix", "corrigir"]),
    ("teste", ["test", "teste", "coverage", "cobertura", "assert", "unit test", "integration test"]),
    ("documentacao", ["docs", "documentation", "documentacao", "readme", "comentario", "comment"]),
    ("duvida", ["?", "why", "por que", "why not", "pode explicar", "nao entendi", "não entendi", "clarify"]),
    ("melhoria", ["improve", "improvement", "melhoria", "better", "refactor", "refatorar", "performance"]),
    ("sugestao", ["suggest", "suggestion", "sugest", "consider", "talvez", "poderia", "seria melhor"]),
    ("nit", ["nit", "nits", "style", "naming", "minor", "small", "pequeno ajuste"]),
    ("risco", ["security", "segur", "race", "rollback", "risk", "risco", "edge case", "corner case"]),
]


def load_payload(path: str) -> Dict[str, Any]:
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def clean_text(text: Optional[str]) -> str:
    if not text:
        return ""
    return re.sub(r"\s+", " ", text).strip()


def summarize(text: str, limit: int = 180) -> str:
    normalized = clean_text(text)
    if len(normalized) <= limit:
        return normalized
    truncated = normalized[: limit - 3].rstrip()
    if " " in truncated:
        truncated = truncated.rsplit(" ", 1)[0]
    return truncated + "..."


def classify(text: str) -> str:
    lowered = clean_text(text).lower()
    for label, keywords in CLASSIFICATION_RULES:
        for keyword in keywords:
            if keyword in lowered:
                return label
    return "outro"


def recommended_action(classification: str, path: Optional[str], line: Optional[int]) -> str:
    location = ""
    if path:
        location = f" em `{path}`"
        if line:
            location += f":{line}"
    mapping = {
        "bug": f"Investigar a causa e corrigir o comportamento apontado{location}.".strip(),
        "melhoria": f"Ajustar a implementacao para reduzir complexidade ou melhorar legibilidade{location}.".strip(),
        "sugestao": f"Avaliar a sugestao, aplicar a menor mudanca util{location} e validar impacto.".strip(),
        "duvida": "Esclarecer a intencao da implementacao e responder de forma objetiva.",
        "nit": f"Aplicar ajuste pontual de estilo ou nomenclatura{location} se houver ganho real.".strip(),
        "documentacao": f"Atualizar documentacao, comentario ou contexto associado{location}.".strip(),
        "teste": f"Adicionar ou ajustar teste relacionado{location}.".strip(),
        "risco": f"Validar cenario de risco citado{location} e registrar mitigacao.".strip(),
        "outro": "Avaliar manualmente e definir se a resposta exige codigo, contexto ou ambos.",
    }
    return mapping[classification]


def make_item(source_type: str, comment: Dict[str, Any], index: int) -> Dict[str, Any]:
    body = clean_text(comment.get("body"))
    path = comment.get("path")
    line = comment.get("line") or comment.get("original_line")
    comment_id = comment.get("id")
    item_id = f"{source_type}-{comment_id or index}"

    return {
        "item_id": item_id,
        "comment_id": comment_id,
        "source_type": source_type,
        "author": (comment.get("user") or {}).get("login"),
        "created_at": comment.get("created_at"),
        "url": comment.get("html_url") or comment.get("url"),
        "path": path,
        "line": line,
        "summary": summarize(body),
        "raw_excerpt": summarize(body, limit=320),
        "classification": classify(body),
        "recommended_action": recommended_action(classify(body), path, line),
        "decision_status": "pending",
    }


def dedupe(items: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    seen = set()
    result = []
    for item in items:
        key = (
            item.get("source_type"),
            item.get("author"),
            item.get("path"),
            item.get("line"),
            item.get("summary"),
        )
        if key in seen:
            continue
        seen.add(key)
        result.append(item)
    return result


def sort_key(item: Dict[str, Any]) -> Tuple[int, str, int]:
    source_rank = 0 if item["source_type"] == "review_comment" else 1
    created_at = item.get("created_at") or ""
    comment_id = item.get("comment_id") or 0
    return (source_rank, created_at, comment_id)


def main() -> int:
    parser = argparse.ArgumentParser(description="Normaliza comentarios de PR em uma fila estruturada.")
    parser.add_argument("--input", required=True, help="Arquivo JSON com issue_comments e review_comments.")
    args = parser.parse_args()

    try:
        payload = load_payload(args.input)
    except FileNotFoundError:
        print(f"INPUT ERROR: arquivo nao encontrado: {args.input}", file=sys.stderr)
        return 1
    except json.JSONDecodeError as exc:
        print(f"INPUT ERROR: JSON invalido: {exc}", file=sys.stderr)
        return 1

    repo = payload.get("repo")
    pr_number = payload.get("pr_number")
    issue_comments = payload.get("issue_comments") or []
    review_comments = payload.get("review_comments") or []

    items: List[Dict[str, Any]] = []
    for index, comment in enumerate(review_comments, start=1):
        if clean_text(comment.get("body")):
            items.append(make_item("review_comment", comment, index))
    for index, comment in enumerate(issue_comments, start=1):
        if clean_text(comment.get("body")):
            items.append(make_item("issue_comment", comment, index))

    items = dedupe(items)
    items.sort(key=sort_key)

    output = {
        "repo": repo,
        "pr_number": pr_number,
        "items": items,
        "counts": {
            "total": len(items),
            "review_comments": len([item for item in items if item["source_type"] == "review_comment"]),
            "issue_comments": len([item for item in items if item["source_type"] == "issue_comment"]),
        },
    }
    json.dump(output, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Valida o bundle de modelagem de dominio production-ready.

Uso:
    python3 validate-bundle.py <caminho-do-bundle>

Exit 0: SUCCESS
Exit 1: erros estruturais ou de conteudo
Exit 2: uso incorreto
"""
from __future__ import annotations

import json
import re
import sys
from datetime import datetime
from pathlib import Path

REQUIRED_FILES = {
    "bundle.json",
    "domain-model.md",
    "transcript.md",
}

REQUIRED_SECTIONS = [
    "## Titulo",
    "## Resumo Executivo",
    "## Problema e Objetivo",
    "## Materiais e Evidencias",
    "## Escopo e Fora de Escopo",
    "## Linguagem Ubiqua",
    "## Bounded Contexts e Fronteiras",
    "## Workflow Principal",
    "## Comandos",
    "## Eventos de Dominio",
    "## Regras, Politicas e Invariantes",
    "## Estados e Transicoes",
    "## Tipos Conceituais",
    "## Erros de Dominio",
    "## Fronteiras Externas e Traducao",
    "## Persistencia, Consistencia e Auditoria",
    "## Observabilidade e Operacao",
    "## Economia, Eficiencia e Custos",
    "## Trade-offs e Decisoes",
    "## Itens em Aberto",
    "## Proximo Passo Recomendado",
]

CRITICAL_SECTIONS = {
    "## Titulo",
    "## Resumo Executivo",
    "## Problema e Objetivo",
    "## Materiais e Evidencias",
    "## Escopo e Fora de Escopo",
    "## Linguagem Ubiqua",
    "## Bounded Contexts e Fronteiras",
    "## Workflow Principal",
    "## Comandos",
    "## Eventos de Dominio",
    "## Regras, Politicas e Invariantes",
    "## Estados e Transicoes",
    "## Tipos Conceituais",
    "## Erros de Dominio",
    "## Fronteiras Externas e Traducao",
    "## Observabilidade e Operacao",
    "## Economia, Eficiencia e Custos",
    "## Trade-offs e Decisoes",
}

MANDATORY_MARKERS = {
    "## Resumo Executivo": ["Contexto:", "Decisao central:", "Status de prontidao:"],
    "## Problema e Objetivo": ["Problema atual:", "Objetivo de negocio:", "Objetivo de modelagem:"],
    "## Materiais e Evidencias": [
        "Materiais usados:",
        "Confronto com codebase:",
        "Escopo analisado:",
        "Status do confronto:",
        "Evidencias:",
        "Riscos de compatibilidade:",
    ],
    "## Escopo e Fora de Escopo": ["Inclui:", "Exclui:"],
    "## Bounded Contexts e Fronteiras": ["Contextos:", "Mapa de contexto:"],
    "## Workflow Principal": ["Gatilho:", "Passos:", "Ponto de decisao:"],
    "## Regras, Politicas e Invariantes": ["Regras de negocio:", "Politicas:", "Invariantes:"],
    "## Estados e Transicoes": [
        "Estado inicial:",
        "Estados validos:",
        "Transicoes permitidas:",
        "Transicoes proibidas:",
    ],
    "## Fronteiras Externas e Traducao": [
        "Entradas externas:",
        "Saidas externas:",
        "Traducao entre dominio e contrato externo:",
    ],
    "## Persistencia, Consistencia e Auditoria": [
        "Persistencia necessaria:",
        "Consistencia requerida:",
        "Auditoria/rastreabilidade:",
    ],
    "## Observabilidade e Operacao": ["Sinais minimos:", "Falhas operacionais relevantes:", "Rollback/contingencia:"],
    "## Economia, Eficiencia e Custos": [
        "Decisoes para reduzir custo:",
        "Custo cognitivo:",
        "Drivers de custo residual:",
    ],
    "## Trade-offs e Decisoes": ["Alternativas rejeitadas:", "Trade-offs aceitos:", "Decisoes consolidadas:"],
}

PLACEHOLDER_EXACT = {
    "TBD",
    "A DEFINIR",
    "A CONFIRMAR",
    "PENDENTE",
    "NAO DEFINIDO",
    "NAO DEFINIDO",
    "N/A",
    "?",
    "-",
    "...",
}

BULLET_STRIP = re.compile(r"^\s*(?:[-*+]|\d+\.)?\s*")
BRACKET_ONLY = re.compile(r"^\s*(?:[-*+]|\d+\.)?\s*\[(?![ xX]?\s*\]).+\]\s*$")
PATH_LINE_PATTERN = re.compile(r"(?m)(?:^|\s)(?:\.{0,2}/)?(?:[\w@.-]+/)*[\w@.-]+:\d+\b")
NO_CODEBASE_PATTERN = re.compile(r"\b(greenfield|sem codebase|nao aplicavel|nao se aplica)\b", re.IGNORECASE)
WEAK_EVIDENCE_TERMS = re.compile(
    r"\b(confirmado|compativel|evidencia confirmada)\b",
    re.IGNORECASE,
)
URL_PATTERN = re.compile(r"https?://[^\s)]+", re.IGNORECASE)
UTC_Z_PATTERN = re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$")
VALID_STATUS = {"draft", "done", "needs_input", "blocked", "failed"}


def split_sections(text: str) -> dict[str, str]:
    sections: dict[str, str] = {}
    current: str | None = None
    buffer: list[str] = []
    for line in text.splitlines():
        if line.startswith("## "):
            if current is not None:
                sections[current] = "\n".join(buffer).strip()
            current = line.strip()
            buffer = []
        else:
            buffer.append(line)
    if current is not None:
        sections[current] = "\n".join(buffer).strip()
    return sections


def detect_placeholders(body: str) -> list[str]:
    issues: list[str] = []
    for line in body.splitlines():
        if not line.strip():
            continue
        stripped = BULLET_STRIP.sub("", line).strip()
        if stripped.upper() in PLACEHOLDER_EXACT:
            issues.append(f"placeholder proibido: '{line.strip()}'")
        elif BRACKET_ONLY.match(line):
            issues.append(f"bracket-only nao preenchido: '{line.strip()}'")
    return issues


def is_meaningful(value: object) -> bool:
    return isinstance(value, str) and bool(value.strip()) and not detect_placeholders(value)


def normalize_name(value: str) -> str:
    return re.sub(r"\s+", " ", value.strip()).casefold()


def extract_list_items(body: str) -> list[str]:
    items: list[str] = []
    for line in body.splitlines():
        stripped = line.strip()
        if stripped.startswith("- "):
            items.append(stripped[2:].strip())
    return [item for item in items if item]


def extract_marker_items(section_body: str, marker: str) -> list[str]:
    items: list[str] = []
    in_marker = False
    for line in section_body.splitlines():
        stripped = line.strip()
        if stripped.endswith(":") and not stripped.startswith("- "):
            in_marker = stripped == marker
            continue
        if in_marker and stripped.startswith("- "):
            items.append(stripped[2:].strip())
        elif in_marker and stripped and not stripped.startswith("- "):
            in_marker = False
    return [item for item in items if item]


def parse_markdown_rows(text: str) -> list[list[str]]:
    rows: list[list[str]] = []
    for line in text.splitlines():
        stripped = line.strip()
        if not stripped.startswith("|") or not stripped.endswith("|"):
            continue
        cells = [cell.strip() for cell in stripped.strip("|").split("|")]
        rows.append(cells)
    return rows


def validate_file_set(bundle_dir: Path) -> list[str]:
    errors: list[str] = []
    existing = {path.name for path in bundle_dir.iterdir()} if bundle_dir.is_dir() else set()
    missing = REQUIRED_FILES - existing
    if missing:
        errors.extend([f"arquivo ausente: {name}" for name in sorted(missing)])
    return errors


def validate_bundle_json(path: Path) -> list[str]:
    if not path.is_file():
        return ["bundle.json: arquivo ausente"]

    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return [f"bundle.json: JSON invalido - {exc}"]

    errors: list[str] = []
    required_top = {
        "version",
        "slug",
        "title",
        "created_at",
        "language",
        "status",
        "domain_model",
        "transcript",
        "bounded_contexts",
        "primary_workflow",
        "commands",
        "events",
        "decisions",
        "readiness",
    }
    missing = required_top - data.keys()
    if missing:
        return [f"bundle.json: campos ausentes - {sorted(missing)}"]

    if data["version"] != 1:
        errors.append(f"bundle.json: versao nao suportada - {data['version']}")
    if data["language"] != "pt-BR":
        errors.append("bundle.json: language deve ser 'pt-BR'")
    if data["status"] not in VALID_STATUS:
        errors.append(f"bundle.json: status invalido - '{data['status']}'")
    if not re.match(r"^[a-z0-9]+(?:-[a-z0-9]+)*$", str(data.get("slug", ""))):
        errors.append(f"bundle.json: slug invalido - '{data.get('slug')}'")
    if not str(data.get("title", "")).strip():
        errors.append("bundle.json: title vazio")
    created_at = str(data.get("created_at", "")).strip()
    if not UTC_Z_PATTERN.match(created_at):
        errors.append("bundle.json: created_at deve estar em ISO-8601 UTC com sufixo Z")
    else:
        try:
            datetime.strptime(created_at, "%Y-%m-%dT%H:%M:%SZ")
        except ValueError:
            errors.append("bundle.json: created_at invalido")

    if data.get("domain_model", {}).get("file") != "domain-model.md":
        errors.append("bundle.json: domain_model.file deve ser 'domain-model.md'")
    if data.get("transcript", {}).get("file") != "transcript.md":
        errors.append("bundle.json: transcript.file deve ser 'transcript.md'")
    if not str(data.get("domain_model", {}).get("title", "")).strip():
        errors.append("bundle.json: domain_model.title vazio")

    bounded_contexts = data.get("bounded_contexts", [])
    if not isinstance(bounded_contexts, list) or not bounded_contexts:
        errors.append("bundle.json: bounded_contexts vazio ou invalido")
    elif any(not is_meaningful(item) for item in bounded_contexts):
        errors.append("bundle.json: bounded_contexts contem item vazio ou placeholder")
    elif len({normalize_name(str(item)) for item in bounded_contexts}) != len(bounded_contexts):
        errors.append("bundle.json: bounded_contexts contem itens duplicados")

    primary_workflow = data.get("primary_workflow", "")
    if not is_meaningful(primary_workflow):
        errors.append("bundle.json: primary_workflow vazio ou placeholder")

    for key in ("commands", "events", "decisions"):
        value = data.get(key, [])
        if not isinstance(value, list) or not value:
            errors.append(f"bundle.json: {key} vazio ou invalido")
        elif any(not is_meaningful(item) for item in value):
            errors.append(f"bundle.json: {key} contem item vazio ou placeholder")

    readiness = data.get("readiness", {})
    if "status" not in readiness or "blockers" not in readiness:
        errors.append("bundle.json: readiness deve conter 'status' e 'blockers'")
    elif readiness.get("status") not in VALID_STATUS:
        errors.append(f"bundle.json: readiness.status invalido - '{readiness.get('status')}'")
    if not isinstance(readiness.get("blockers", []), list):
        errors.append("bundle.json: readiness.blockers deve ser lista")
    elif any(not isinstance(item, str) for item in readiness.get("blockers", [])):
        errors.append("bundle.json: readiness.blockers deve conter apenas strings")

    if data.get("status") == "done" and readiness.get("status") != "done":
        errors.append("bundle.json: status done exige readiness.status done")

    return errors


def validate_codebase_confrontation(body: str) -> list[str]:
    errors: list[str] = []
    has_path_line = bool(PATH_LINE_PATTERN.search(body))
    has_url = bool(URL_PATTERN.search(body))
    no_codebase = bool(NO_CODEBASE_PATTERN.search(body))
    claims_confirmed = bool(WEAK_EVIDENCE_TERMS.search(body))

    if not has_path_line and not has_url and not no_codebase:
        errors.append(
            "evidencia `path:linha` ou URL ausente; use greenfield/nao aplicavel com justificativa quando nao houver codebase"
        )
    if claims_confirmed and not has_path_line and not has_url:
        errors.append("compatibilidade confirmada sem evidencia citavel")
    return errors


def validate_domain_model(path: Path) -> list[str]:
    if not path.is_file():
        return ["domain-model.md: arquivo ausente"]

    text = path.read_text(encoding="utf-8")
    if not text.strip():
        return ["domain-model.md: arquivo vazio"]

    sections = split_sections(text)
    errors: list[str] = []

    for required in REQUIRED_SECTIONS:
        if required not in sections:
            errors.append(f"domain-model.md: secao ausente '{required}'")
            continue
        body = sections[required]
        if required in CRITICAL_SECTIONS and not body:
            errors.append(f"domain-model.md: secao critica vazia '{required}'")
            continue
        for issue in detect_placeholders(body):
            errors.append(f"domain-model.md -> {required}: {issue}")
        for marker in MANDATORY_MARKERS.get(required, []):
            if marker not in body:
                errors.append(f"domain-model.md -> {required}: marcador obrigatorio ausente '{marker}'")

    evidence_body = sections.get("## Materiais e Evidencias", "")
    for issue in validate_codebase_confrontation(evidence_body):
        errors.append(f"domain-model.md -> ## Materiais e Evidencias: {issue}")

    language_body = sections.get("## Linguagem Ubiqua", "")
    rows = parse_markdown_rows(language_body)
    data_rows = [
        row for row in rows
        if len(row) == 4 and row[0].lower() != "termo" and set("".join(row)) != {"-"}
    ]
    if not data_rows:
        errors.append("domain-model.md -> ## Linguagem Ubiqua: tabela sem termos preenchidos")
    for row in data_rows:
        if any(not cell or cell == "-" for cell in row[:2]):
            errors.append("domain-model.md -> ## Linguagem Ubiqua: linha com termo/definicao vazios")

    contexts_body = sections.get("## Bounded Contexts e Fronteiras", "")
    if contexts_body.count("Contexto:") < 1:
        errors.append("domain-model.md -> ## Bounded Contexts e Fronteiras: nenhum contexto preenchido")

    commands_body = sections.get("## Comandos", "")
    if commands_body.count("Comando:") < 1:
        errors.append("domain-model.md -> ## Comandos: nenhum comando preenchido")

    events_body = sections.get("## Eventos de Dominio", "")
    if events_body.count("Evento:") < 1:
        errors.append("domain-model.md -> ## Eventos de Dominio: nenhum evento preenchido")

    invariants_body = sections.get("## Regras, Politicas e Invariantes", "")
    if invariants_body.count("Invariante:") < 1:
        errors.append("domain-model.md -> ## Regras, Politicas e Invariantes: nenhuma invariante preenchida")
    if invariants_body.count("Politica:") < 1:
        errors.append("domain-model.md -> ## Regras, Politicas e Invariantes: nenhuma politica preenchida")

    states_body = sections.get("## Estados e Transicoes", "")
    if "Transicoes proibidas:" in states_body and states_body.count("->") < 2:
        errors.append("domain-model.md -> ## Estados e Transicoes: transicoes permitidas/proibidas insuficientes")

    types_body = sections.get("## Tipos Conceituais", "")
    if "Entidades:" not in types_body or "Value Objects:" not in types_body or "Agregados:" not in types_body:
        errors.append("domain-model.md -> ## Tipos Conceituais: blocos obrigatorios ausentes")

    errors_body = sections.get("## Erros de Dominio", "")
    if errors_body.count("Erro:") < 1:
        errors.append("domain-model.md -> ## Erros de Dominio: nenhum erro de dominio preenchido")

    tradeoffs_body = sections.get("## Trade-offs e Decisoes", "")
    if tradeoffs_body.count("Trade-offs aceitos:") != 1:
        errors.append("domain-model.md -> ## Trade-offs e Decisoes: bloco de trade-offs ausente")

    next_step = sections.get("## Proximo Passo Recomendado", "")
    if not re.search(r"\b(technical-discovery-production|epic-story-discovery|implementacao guiada)\b", next_step):
        errors.append(
            "domain-model.md -> ## Proximo Passo Recomendado: valor deve mencionar technical-discovery-production, epic-story-discovery ou implementacao guiada"
        )

    return errors


def validate_transcript(path: Path) -> list[str]:
    if not path.is_file():
        return ["transcript.md: arquivo ausente"]

    text = path.read_text(encoding="utf-8")
    if not text.strip():
        return ["transcript.md: arquivo vazio"]

    required_headings = [
        "## Contexto Inicial",
        "## Confronto com Codebase",
        "## Rodada 1 - Linguagem e Fronteiras",
        "## Rodada 2 - Workflow e Comportamento",
        "## Rodada 3 - Regras e Invariantes",
        "## Rodada 4 - Tipos e Integracoes",
        "## Decisoes Registradas",
    ]
    sections = split_sections(text)
    errors: list[str] = []
    for heading in required_headings:
        if heading not in sections:
            errors.append(f"transcript.md: heading ausente '{heading}'")
            continue
        if not sections[heading].strip():
            errors.append(f"transcript.md: secao vazia '{heading}'")
    for issue in detect_placeholders(text):
        errors.append(f"transcript.md: {issue}")
    return errors


def extract_title(text: str) -> str:
    sections = split_sections(text)
    return sections.get("## Titulo", "").splitlines()[0].strip() if sections.get("## Titulo", "").strip() else ""


def extract_tagged_values(section_body: str, tag: str) -> list[str]:
    values: list[str] = []
    prefix = f"{tag}:"
    for line in section_body.splitlines():
        stripped = line.strip()
        if stripped.startswith("- "):
            stripped = stripped[2:].strip()
        if stripped.startswith(prefix):
            values.append(stripped[len(prefix):].strip())
    return [value for value in values if value]


def validate_cross_consistency(bundle_path: Path, model_path: Path) -> list[str]:
    if not bundle_path.is_file() or not model_path.is_file():
        return []

    bundle = json.loads(bundle_path.read_text(encoding="utf-8"))
    model_text = model_path.read_text(encoding="utf-8")
    sections = split_sections(model_text)
    errors: list[str] = []

    model_title = extract_title(model_text)
    if model_title:
        if normalize_name(str(bundle.get("title", ""))) != normalize_name(model_title):
            errors.append("consistencia: bundle.json.title difere de ## Titulo")
        if normalize_name(str(bundle.get("domain_model", {}).get("title", ""))) != normalize_name(model_title):
            errors.append("consistencia: bundle.json.domain_model.title difere de ## Titulo")

    context_values = extract_tagged_values(sections.get("## Bounded Contexts e Fronteiras", ""), "Contexto")
    if context_values:
        declared_contexts = [normalize_name(str(item)) for item in bundle.get("bounded_contexts", [])]
        materialized_contexts = [normalize_name(item) for item in context_values]
        if declared_contexts != materialized_contexts:
            errors.append("consistencia: bounded_contexts do bundle.json difere dos contextos materializados")

    command_values = extract_tagged_values(sections.get("## Comandos", ""), "Comando")
    if command_values:
        declared_commands = [normalize_name(str(item)) for item in bundle.get("commands", [])]
        materialized_commands = [normalize_name(item) for item in command_values]
        if declared_commands != materialized_commands:
            errors.append("consistencia: commands do bundle.json difere dos comandos materializados")

    event_values = extract_tagged_values(sections.get("## Eventos de Dominio", ""), "Evento")
    if event_values:
        declared_events = [normalize_name(str(item)) for item in bundle.get("events", [])]
        materialized_events = [normalize_name(item) for item in event_values]
        if declared_events != materialized_events:
            errors.append("consistencia: events do bundle.json difere dos eventos materializados")

    decision_values = extract_marker_items(sections.get("## Trade-offs e Decisoes", ""), "Decisoes consolidadas:")
    if decision_values:
        declared_decisions = [normalize_name(str(item)) for item in bundle.get("decisions", [])]
        materialized_decisions = [normalize_name(item) for item in decision_values]
        if declared_decisions != materialized_decisions:
            errors.append("consistencia: decisions do bundle.json difere das decisoes materializadas")

    workflow_body = sections.get("## Workflow Principal", "")
    primary_workflow = normalize_name(str(bundle.get("primary_workflow", "")))
    if primary_workflow and primary_workflow not in normalize_name(workflow_body):
        errors.append("consistencia: primary_workflow do bundle.json nao aparece em ## Workflow Principal")

    return errors


def main() -> int:
    if len(sys.argv) != 2:
        print("USO: validate-bundle.py <caminho-do-bundle>", file=sys.stderr)
        return 2

    bundle_dir = Path(sys.argv[1]).resolve()
    if not bundle_dir.is_dir():
        print(f"DIRETORIO INVALIDO: {bundle_dir}", file=sys.stderr)
        return 1

    errors: list[str] = []
    errors.extend(validate_file_set(bundle_dir))

    if not errors:
        errors.extend(validate_bundle_json(bundle_dir / "bundle.json"))
        errors.extend(validate_domain_model(bundle_dir / "domain-model.md"))
        errors.extend(validate_transcript(bundle_dir / "transcript.md"))
        errors.extend(validate_cross_consistency(bundle_dir / "bundle.json", bundle_dir / "domain-model.md"))

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    print("SUCCESS")
    return 0


if __name__ == "__main__":
    sys.exit(main())

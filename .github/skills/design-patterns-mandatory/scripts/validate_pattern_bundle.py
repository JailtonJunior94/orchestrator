#!/usr/bin/env python3
"""Valida bundle de decisao de design pattern.

Uso:
    python3 scripts/validate_pattern_bundle.py <bundle.md|bundle_dir>

Exit 0: SUCCESS
Exit 1: erro estrutural ou de conteudo
Exit 2: uso incorreto
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path


DECISION_REQUIRED_SECTIONS = [
    "## Contexto",
    "## Diagnostico do problema",
    "## Evidencias",
    "## Alternativa mais simples rejeitada",
    "## Padrao primario",
    "## Padroes rejeitados",
    "## Justificativa de economia",
    "## Justificativa de eficiencia",
    "## Justificativa de robustez",
    "## Estrutura minima",
    "## Fluxo",
    "## Pseudocodigo canonico",
    "## Mapeamento por paradigma",
    "## Plano de implementacao ou refatoracao",
    "## Plano de testes",
    "## Criterios de aceite",
    "## Riscos e criterios de nao uso",
]

DECISION_MANDATORY_MARKERS = {
    "## Alternativa mais simples rejeitada": ["Alternativa:", "Motivo da rejeicao:"],
    "## Padrao primario": ["Recomendar:"],
    "## Justificativa de economia": ["Custo evitado:", "Retorno esperado:"],
    "## Justificativa de eficiencia": ["Impacto em execucao:", "Impacto em manutencao:"],
    "## Justificativa de robustez": ["Falhas mitigadas:", "Invariantes protegidos:"],
    "## Estrutura minima": ["Participantes:", "Responsabilidades:"],
    "## Plano de testes": ["Teste positivo:", "Teste negativo:", "Teste de regressao:"],
    "## Riscos e criterios de nao uso": ["Riscos:", "Nao usar quando:"],
}

IMPLEMENTATION_REQUIRED_SECTIONS = [
    "## Objetivo",
    "## Contratos preservados",
    "## Participantes concretos",
    "## Adaptacao para a linguagem",
    "## Pseudocodigo adaptado",
    "## Plano de mudanca",
    "## Testes obrigatorios",
    "## Rollback mental",
]

IMPLEMENTATION_MANDATORY_MARKERS = {
    "## Objetivo": ["Resultado esperado:"],
    "## Contratos preservados": ["- API publica:", "- Invariantes:", "- Comportamentos que nao podem mudar:"],
    "## Participantes concretos": ["- Papel do pattern:", "- Tipo ou modulo real:"],
    "## Adaptacao para a linguagem": ["Paradigma:", "Escolha estrutural:"],
    "## Plano de mudanca": ["1."],
    "## Testes obrigatorios": [
        "Teste positivo:",
        "Teste negativo:",
        "Teste de regressao:",
        "Teste de falha:",
    ],
    "## Rollback mental": ["Sinal de que a implementacao ficou cara demais:", "Acao corretiva:"],
}

PLACEHOLDER_EXACT = {
    "",
    "TBD",
    "A DEFINIR",
    "A CONFIRMAR",
    "PENDENTE",
    "N/A",
    "?",
    "-",
    "...",
}

VALID_STATUS = {"draft", "done", "needs_input", "blocked", "failed"}
VALID_CONTEXT_MODE = {"codebase", "greenfield"}

BULLET_STRIP = re.compile(r"^\s*(?:[-*+]|\d+\.)?\s*")
BRACKET_ONLY = re.compile(r"^\s*(?:[-*+]|\d+\.)?\s*\[(?![ xX]?\s*\]).+\]\s*$")
RECOMMEND_LINE = re.compile(r"(?m)^Recomendar:\s*(.+)$")
PATH_LINE = re.compile(r"(?m)\b[./A-Za-z0-9_-]+(?:/[A-Za-z0-9._-]+)+:\d+\b")
ALLOWED_PATTERNS = {
    "Factory Method",
    "Abstract Factory",
    "Builder",
    "Prototype",
    "Singleton",
    "Adapter",
    "Bridge",
    "Composite",
    "Decorator",
    "Facade",
    "Flyweight",
    "Proxy",
    "Chain of Responsibility",
    "Command",
    "Iterator",
    "Mediator",
    "Memento",
    "Observer",
    "State",
    "Strategy",
    "Template Method",
    "Visitor",
    "nao aplicar padrao",
}


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


def detect_placeholders(text: str) -> list[str]:
    issues: list[str] = []
    for line in text.splitlines():
        if not line.strip():
            continue
        stripped = BULLET_STRIP.sub("", line).strip()
        if stripped.upper() in PLACEHOLDER_EXACT:
            issues.append(f"placeholder proibido: '{line.strip()}'")
        elif BRACKET_ONLY.match(line):
            issues.append(f"bracket-only nao preenchido: '{line.strip()}'")
    return issues


def validate_markdown_sections(
    text: str,
    required_sections: list[str],
    mandatory_markers: dict[str, list[str]],
    pseudocode_heading: str,
) -> tuple[list[str], dict[str, str]]:
    errors: list[str] = []
    sections = split_sections(text)

    for section in required_sections:
        if section not in sections:
            errors.append(f"secao ausente: {section}")
            continue
        body = sections[section]
        if not body:
            errors.append(f"secao vazia: {section}")
            continue
        errors.extend([f"{section}: {issue}" for issue in detect_placeholders(body)])
        for marker in mandatory_markers.get(section, []):
            if marker not in body:
                errors.append(f"{section}: marcador obrigatorio ausente '{marker}'")

    pseudocode_section = sections.get(pseudocode_heading, "")
    if "```text" not in pseudocode_section:
        errors.append(f"{pseudocode_heading}: usar bloco ```text")

    return errors, sections


def validate_decision_text(text: str) -> list[str]:
    errors, sections = validate_markdown_sections(
        text,
        DECISION_REQUIRED_SECTIONS,
        DECISION_MANDATORY_MARKERS,
        "## Pseudocodigo canonico",
    )

    match = RECOMMEND_LINE.search(text)
    if not match:
        errors.append("Padrao primario: linha 'Recomendar:' ausente")
    else:
        recommendation = match.group(1).strip()
        if recommendation not in ALLOWED_PATTERNS:
            errors.append(f"Padrao primario: valor invalido em 'Recomendar:' - '{recommendation}'")

    plan_section = sections.get("## Plano de implementacao ou refatoracao", "")
    if "Passos:" not in plan_section and "1." not in plan_section:
        errors.append("## Plano de implementacao ou refatoracao: incluir passos executaveis")

    criteria_section = sections.get("## Criterios de aceite", "")
    if "-" not in criteria_section:
        errors.append("## Criterios de aceite: incluir pelo menos um item em lista")

    evidence_section = sections.get("## Evidencias", "")
    if "greenfield" not in evidence_section.lower() and not PATH_LINE.search(evidence_section):
        errors.append("## Evidencias: informar ao menos um path:line ou declarar greenfield")

    rejected_section = sections.get("## Padroes rejeitados", "")
    if not rejected_section.strip():
        errors.append("## Padroes rejeitados: listar ao menos uma alternativa rejeitada")

    return errors


def validate_implementation_text(text: str) -> list[str]:
    errors, _ = validate_markdown_sections(
        text,
        IMPLEMENTATION_REQUIRED_SECTIONS,
        IMPLEMENTATION_MANDATORY_MARKERS,
        "## Pseudocodigo adaptado",
    )
    return errors


def validate_bundle_json(bundle_dir: Path) -> tuple[list[str], dict | None]:
    path = bundle_dir / "bundle.json"
    if not path.is_file():
        return ["bundle.json: arquivo ausente"], None

    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return [f"bundle.json: JSON invalido - {exc}"], None

    errors: list[str] = []
    required_top = {
        "version",
        "slug",
        "title",
        "created_at",
        "language",
        "status",
        "context_mode",
        "selector_input",
        "selector_output",
        "decision_bundle",
        "implementation_bundle",
        "transcript",
        "primary_pattern",
        "complementary_pattern",
        "rejected_patterns",
        "readiness",
    }
    missing = required_top - data.keys()
    if missing:
        return [f"bundle.json: campos ausentes - {sorted(missing)}"], None

    if data.get("version") != 1:
        errors.append(f"bundle.json: versao nao suportada - {data.get('version')}")
    if data.get("language") != "pt-BR":
        errors.append("bundle.json: language deve ser 'pt-BR'")
    if data.get("status") not in VALID_STATUS:
        errors.append(f"bundle.json: status invalido - '{data.get('status')}'")
    if data.get("context_mode") not in VALID_CONTEXT_MODE:
        errors.append(f"bundle.json: context_mode invalido - '{data.get('context_mode')}'")
    if not re.match(r"^[a-z0-9]+(?:-[a-z0-9]+)*$", str(data.get("slug", ""))):
        errors.append(f"bundle.json: slug invalido - '{data.get('slug')}'")

    file_expectations = {
        "selector_input": "selector-input.json",
        "selector_output": "selector-output.json",
        "decision_bundle": "decision.md",
        "implementation_bundle": "implementation.md",
        "transcript": "transcript.md",
    }
    for key, expected_file in file_expectations.items():
        block = data.get(key, {})
        if block.get("file") != expected_file:
            errors.append(f"bundle.json: {key}.file deve ser '{expected_file}'")

    primary = str(data.get("primary_pattern", "")).strip()
    if primary and primary not in ALLOWED_PATTERNS:
        errors.append(f"bundle.json: primary_pattern invalido - '{primary}'")

    rejected = data.get("rejected_patterns", [])
    if not isinstance(rejected, list):
        errors.append("bundle.json: rejected_patterns deve ser lista")
    elif any(not isinstance(item, str) or not item.strip() for item in rejected):
        errors.append("bundle.json: rejected_patterns deve conter apenas strings nao vazias")

    readiness = data.get("readiness", {})
    if readiness.get("status") not in VALID_STATUS:
        errors.append(f"bundle.json: readiness.status invalido - '{readiness.get('status')}'")
    blockers = readiness.get("blockers")
    if not isinstance(blockers, list):
        errors.append("bundle.json: readiness.blockers deve ser lista")
    elif any(not isinstance(item, str) or not item.strip() for item in blockers):
        errors.append("bundle.json: readiness.blockers deve conter apenas strings nao vazias")

    if data.get("status") == "done" and blockers:
        errors.append("bundle.json: status done exige readiness.blockers vazio")

    return errors, data


def validate_selector_output(bundle_dir: Path, bundle_json: dict) -> list[str]:
    output_path = bundle_dir / bundle_json["selector_output"]["file"]
    if not output_path.is_file():
        return [f"{output_path.name}: arquivo ausente"]

    try:
        data = json.loads(output_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return [f"{output_path.name}: JSON invalido - {exc}"]

    errors: list[str] = []
    if data and data.get("status") == "ok":
        selector_primary = (data.get("primary_pattern") or {}).get("pattern")
        bundle_primary = str(bundle_json.get("primary_pattern", "")).strip()
        if bundle_primary and selector_primary and bundle_primary != selector_primary:
            errors.append(
                f"{output_path.name}: primary_pattern diverge entre seletor ('{selector_primary}') e bundle ('{bundle_primary}')"
            )
    return errors


def validate_bundle_directory(bundle_dir: Path) -> list[str]:
    errors, bundle_json = validate_bundle_json(bundle_dir)
    if bundle_json is None:
        return errors

    decision_path = bundle_dir / bundle_json["decision_bundle"]["file"]
    implementation_path = bundle_dir / bundle_json["implementation_bundle"]["file"]
    transcript_path = bundle_dir / bundle_json["transcript"]["file"]
    selector_input_path = bundle_dir / bundle_json["selector_input"]["file"]

    if not decision_path.is_file():
        errors.append(f"{decision_path.name}: arquivo ausente")
    else:
        errors.extend([f"{decision_path.name}: {err}" for err in validate_decision_text(decision_path.read_text(encoding="utf-8"))])

    if not implementation_path.is_file():
        errors.append(f"{implementation_path.name}: arquivo ausente")
    else:
        errors.extend(
            [f"{implementation_path.name}: {err}" for err in validate_implementation_text(implementation_path.read_text(encoding="utf-8"))]
        )

    if not transcript_path.is_file():
        errors.append(f"{transcript_path.name}: arquivo ausente")
    else:
        transcript_text = transcript_path.read_text(encoding="utf-8")
        if "## Contexto Inicial" not in transcript_text:
            errors.append(f"{transcript_path.name}: secao '## Contexto Inicial' ausente")

    if not selector_input_path.is_file():
        errors.append(f"{selector_input_path.name}: arquivo ausente")

    errors.extend(validate_selector_output(bundle_dir, bundle_json))
    return errors


def main() -> int:
    if len(sys.argv) != 2:
        print("USAGE ERROR: python3 scripts/validate_pattern_bundle.py <bundle.md|bundle_dir>", file=sys.stderr)
        return 2

    path = Path(sys.argv[1])
    if path.is_dir():
        errors = validate_bundle_directory(path)
    elif path.is_file():
        errors = validate_decision_text(path.read_text(encoding="utf-8"))
    else:
        print(f"INPUT ERROR: caminho ausente - {path}", file=sys.stderr)
        return 1

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    print("SUCCESS: bundle valido e consistente.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

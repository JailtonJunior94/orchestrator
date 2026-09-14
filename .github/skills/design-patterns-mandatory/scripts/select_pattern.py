#!/usr/bin/env python3
"""Seleciona design pattern GoF de forma deterministica e conservadora.

Uso:
    python3 scripts/select_pattern.py --input signals.json
    cat signals.json | python3 scripts/select_pattern.py --input -

Entrada JSON:
{
  "problem": "texto livre",
  "signals": ["runtime_algorithm_swap", "..."],
  "constraints": ["avoid_inheritance", "..."],
  "force_reject": ["singleton"]
}
"""
from __future__ import annotations

import argparse
import json
import sys
from dataclasses import dataclass
from pathlib import Path


ALLOWED_SIGNALS = {
    "single_product_variation",
    "family_of_related_products",
    "stepwise_construction",
    "clone_template",
    "single_process_shared_resource",
    "external_interface_mismatch",
    "dual_axis_variation",
    "recursive_tree_structure",
    "uniform_component_contract",
    "add_responsibilities_dynamically",
    "subsystem_too_complex",
    "high_memory_duplication",
    "access_control_or_lazy_loading",
    "sequential_conditional_handlers",
    "request_as_data",
    "custom_traversal",
    "dense_colleague_coordination",
    "snapshot_and_restore",
    "event_fanout",
    "state_transition_driven_behavior",
    "runtime_algorithm_swap",
    "fixed_workflow_with_variable_steps",
    "stable_structure_many_operations",
    "prefer_composition",
    "prefer_direct_solution",
    "low_change_frequency",
    "single_variant_only",
    "performance_hot_path",
    "memory_pressure",
    "strict_test_isolation",
    "multi_tenant_context",
    "remote_boundary",
    "undo_or_replay",
    "cross_product_consistency",
    "inheritance_already_natural",
}

ALLOWED_CONSTRAINTS = {
    "avoid_global_state",
    "avoid_inheritance",
    "minimize_class_count",
    "minimize_indirection",
    "preserve_public_contract",
    "tight_latency_budget",
    "tight_memory_budget",
    "high_change_frequency",
    "team_needs_low_cognitive_load",
    "must_support_runtime_switch",
    "must_support_undo",
    "must_support_broadcast",
    "must_support_checkpoints",
    "must_support_remote_access",
}


@dataclass(frozen=True)
class PatternRule:
    key: str
    name: str
    group: str
    simpler_alternative: str
    strong: frozenset[str]
    weak: frozenset[str]
    blockers: frozenset[str]
    high_bar: bool = False


PATTERNS = [
    PatternRule(
        "factory-method",
        "Factory Method",
        "criacional",
        "Usar construtor ou funcao fabrica simples",
        frozenset({"single_product_variation"}),
        frozenset({"prefer_composition"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "abstract-factory",
        "Abstract Factory",
        "criacional",
        "Usar funcoes fabrica simples por produto",
        frozenset({"family_of_related_products", "cross_product_consistency"}),
        frozenset({"single_product_variation"}),
        frozenset({"single_variant_only", "prefer_direct_solution"}),
        high_bar=True,
    ),
    PatternRule(
        "builder",
        "Builder",
        "criacional",
        "Usar construtor simples, literal de objeto ou funcao com defaults",
        frozenset({"stepwise_construction"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"single_variant_only", "prefer_direct_solution"}),
        high_bar=True,
    ),
    PatternRule(
        "prototype",
        "Prototype",
        "criacional",
        "Usar funcao fabrica simples que monte o objeto do zero",
        frozenset({"clone_template"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "singleton",
        "Singleton",
        "criacional",
        "Usar injecao de dependencia com ownership explicito",
        frozenset({"single_process_shared_resource"}),
        frozenset({"tight_latency_budget"}),
        frozenset({"avoid_global_state", "strict_test_isolation", "multi_tenant_context"}),
        high_bar=True,
    ),
    PatternRule(
        "adapter",
        "Adapter",
        "estrutural",
        "Usar funcao tradutora local",
        frozenset({"external_interface_mismatch"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"prefer_direct_solution", "single_variant_only", "low_change_frequency"}),
    ),
    PatternRule(
        "bridge",
        "Bridge",
        "estrutural",
        "Usar composicao direta sem dupla hierarquia",
        frozenset({"dual_axis_variation"}),
        frozenset({"prefer_composition"}),
        frozenset({"single_variant_only", "prefer_direct_solution"}),
        high_bar=True,
    ),
    PatternRule(
        "composite",
        "Composite",
        "estrutural",
        "Usar colecao simples ou recursao local",
        frozenset({"recursive_tree_structure", "uniform_component_contract"}),
        frozenset({"custom_traversal"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "decorator",
        "Decorator",
        "estrutural",
        "Usar composicao direta ou wrapper unico",
        frozenset({"add_responsibilities_dynamically"}),
        frozenset({"prefer_composition"}),
        frozenset({"prefer_direct_solution"}),
    ),
    PatternRule(
        "facade",
        "Facade",
        "estrutural",
        "Usar funcao de alto nivel ou modulo simplificado",
        frozenset({"subsystem_too_complex"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"low_change_frequency", "single_variant_only"}),
    ),
    PatternRule(
        "flyweight",
        "Flyweight",
        "estrutural",
        "Usar deduplicacao simples ou cache local",
        frozenset({"high_memory_duplication", "memory_pressure"}),
        frozenset({"performance_hot_path"}),
        frozenset({"prefer_direct_solution"}),
        high_bar=True,
    ),
    PatternRule(
        "proxy",
        "Proxy",
        "estrutural",
        "Usar wrapper unico especializado",
        frozenset({"access_control_or_lazy_loading"}),
        frozenset({"remote_boundary", "performance_hot_path"}),
        frozenset({"prefer_direct_solution"}),
    ),
    PatternRule(
        "chain-of-responsibility",
        "Chain of Responsibility",
        "comportamental",
        "Usar sequencia explicita de funcoes",
        frozenset({"sequential_conditional_handlers"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "command",
        "Command",
        "comportamental",
        "Usar chamada direta ou funcao callback",
        frozenset({"request_as_data", "undo_or_replay"}),
        frozenset({"must_support_undo"}),
        frozenset({"prefer_direct_solution"}),
    ),
    PatternRule(
        "iterator",
        "Iterator",
        "comportamental",
        "Usar iteracao nativa da linguagem",
        frozenset({"custom_traversal"}),
        frozenset({"recursive_tree_structure"}),
        frozenset({"prefer_direct_solution"}),
    ),
    PatternRule(
        "mediator",
        "Mediator",
        "comportamental",
        "Usar orquestrador simples ou modulo de servico",
        frozenset({"dense_colleague_coordination"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"prefer_direct_solution"}),
        high_bar=True,
    ),
    PatternRule(
        "memento",
        "Memento",
        "comportamental",
        "Usar clone controlado ou event log simples",
        frozenset({"snapshot_and_restore"}),
        frozenset({"must_support_checkpoints"}),
        frozenset({"prefer_direct_solution"}),
    ),
    PatternRule(
        "observer",
        "Observer",
        "comportamental",
        "Usar callback direto ou chamada sequencial explicita",
        frozenset({"event_fanout"}),
        frozenset({"must_support_broadcast"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "state",
        "State",
        "comportamental",
        "Usar tabela de transicao ou enum com regras localizadas",
        frozenset({"state_transition_driven_behavior"}),
        frozenset({"preserve_public_contract"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "strategy",
        "Strategy",
        "comportamental",
        "Usar funcao parametrizada ou dispatch table",
        frozenset({"runtime_algorithm_swap"}),
        frozenset({"must_support_runtime_switch", "prefer_composition"}),
        frozenset({"single_variant_only"}),
    ),
    PatternRule(
        "template-method",
        "Template Method",
        "comportamental",
        "Usar funcao de alto nivel com callbacks ou Strategy",
        frozenset({"fixed_workflow_with_variable_steps"}),
        frozenset({"inheritance_already_natural"}),
        frozenset({"avoid_inheritance"}),
    ),
    PatternRule(
        "visitor",
        "Visitor",
        "comportamental",
        "Usar metodos diretos na estrutura ou funcoes por tipo",
        frozenset({"stable_structure_many_operations"}),
        frozenset({"recursive_tree_structure"}),
        frozenset({"high_change_frequency", "prefer_direct_solution"}),
        high_bar=True,
    ),
]

PATTERN_BY_KEY = {item.key: item for item in PATTERNS}

OVERENGINEERING_HINTS = {
    "abstract-factory",
    "builder",
    "bridge",
    "flyweight",
    "mediator",
    "visitor",
    "singleton",
}

AMBIGUITY_RULES = [
    (
        "strategy",
        "state",
        "Falta separar troca de algoritmo de comportamento guiado por transicoes de estado.",
    ),
    (
        "decorator",
        "proxy",
        "Falta separar extensao dinamica de responsabilidade de controle de acesso ou lazy load.",
    ),
    (
        "factory-method",
        "abstract-factory",
        "Falta provar se varia um produto isolado ou uma familia inteira consistente.",
    ),
    (
        "strategy",
        "template-method",
        "Falta provar se a variacao deve acontecer por composicao em runtime ou por esqueleto fixo com heranca.",
    ),
    (
        "adapter",
        "facade",
        "Falta separar incompatibilidade de interface de simplificacao de subsistema.",
    ),
    (
        "adapter",
        "proxy",
        "Falta separar traducao de interface de governanca de acesso, lazy load ou fronteira remota.",
    ),
    (
        "facade",
        "proxy",
        "Falta separar simplificacao de subsistema de controle de acesso a recurso.",
    ),
    (
        "command",
        "chain-of-responsibility",
        "Falta separar requisicao como dado de pipeline sequencial de handlers.",
    ),
    (
        "observer",
        "mediator",
        "Falta separar fanout de eventos de coordenacao central entre colegas.",
    ),
    (
        "composite",
        "visitor",
        "Falta separar necessidade de arvore recursiva de extensao de operacoes sobre estrutura estavel.",
    ),
    (
        "bridge",
        "strategy",
        "Falta separar variacao em duas dimensoes independentes de simples troca de algoritmo ou politica.",
    ),
    (
        "composite",
        "decorator",
        "Falta separar arvore parte-todo de adicao de responsabilidade sobre um unico componente.",
    ),
    (
        "iterator",
        "visitor",
        "Falta separar necessidade de travessia da necessidade de adicionar operacoes a uma estrutura estavel.",
    ),
    (
        "flyweight",
        "singleton",
        "Falta separar economia de memoria por compartilhamento estrutural de unicidade controlada de recurso.",
    ),
]


def load_input(raw: str) -> dict:
    if raw == "-":
        payload = sys.stdin.read()
    else:
        payload = Path(raw).read_text(encoding="utf-8")
    data = json.loads(payload)
    if not isinstance(data, dict):
        raise ValueError("Entrada JSON deve ser um objeto.")
    return data


def normalize_list(values: object, allowed: set[str], label: str) -> tuple[set[str], list[str]]:
    if values is None:
        return set(), []
    if not isinstance(values, list):
        raise ValueError(f"Campo '{label}' deve ser lista.")
    normalized: set[str] = set()
    unknown: list[str] = []
    for item in values:
        if not isinstance(item, str):
            raise ValueError(f"Campo '{label}' deve conter apenas strings.")
        value = item.strip()
        if not value:
            continue
        if value in allowed:
            normalized.add(value)
        else:
            unknown.append(value)
    return normalized, sorted(set(unknown))


def score_pattern(rule: PatternRule, signals: set[str], constraints: set[str]) -> dict:
    strong_hits = sorted(rule.strong & signals)
    weak_hits = sorted(rule.weak & (signals | constraints))
    blocker_hits = sorted(rule.blockers & (signals | constraints))
    score = len(strong_hits) * 3 + len(weak_hits) - len(blocker_hits) * 4
    if rule.high_bar and len(strong_hits) < 2:
        score -= 2
    if "minimize_class_count" in constraints and rule.high_bar:
        score -= 2
    if "minimize_indirection" in constraints and rule.key in {
        "abstract-factory",
        "builder",
        "bridge",
        "decorator",
        "mediator",
        "proxy",
        "visitor",
    }:
        score -= 1
    if "tight_latency_budget" in constraints and rule.key in {"decorator", "proxy", "mediator"}:
        score -= 1
    if "tight_memory_budget" in constraints and rule.key == "flyweight":
        score += 1
    if "avoid_inheritance" in constraints and rule.key in {"template-method", "factory-method"}:
        score -= 2
    return {
        "key": rule.key,
        "name": rule.name,
        "group": rule.group,
        "simpler_alternative": rule.simpler_alternative,
        "score": score,
        "strong_hits": strong_hits,
        "weak_hits": weak_hits,
        "blocker_hits": blocker_hits,
        "high_bar": rule.high_bar,
    }


def direct_solution_preferred(signals: set[str], constraints: set[str]) -> bool:
    direct = {"prefer_direct_solution", "single_variant_only", "low_change_frequency"}
    return len(direct & signals) >= 2 or (
        "team_needs_low_cognitive_load" in constraints and "single_variant_only" in signals
    )


def candidate_is_valid(candidate: dict) -> bool:
    if candidate["score"] < 4:
        return False
    if not candidate["strong_hits"]:
        return False
    if candidate["key"] == "composite" and "uniform_component_contract" not in candidate["strong_hits"]:
        return False
    if candidate["high_bar"] and len(candidate["strong_hits"]) < 2:
        return False
    if len(candidate["blocker_hits"]) >= 2:
        return False
    return True


def find_ambiguity(top: list[dict]) -> str | None:
    keys = {item["key"] for item in top}
    scores = {item["key"]: item["score"] for item in top}
    for left, right, reason in AMBIGUITY_RULES:
        if left in keys and right in keys and abs(scores[left] - scores[right]) <= 1:
            return reason
    return None


def build_reason_sections(candidate: dict, constraints: set[str]) -> tuple[list[str], list[str], list[str]]:
    economy = [
        f"Reduz acoplamento ou branching usando os sinais fortes: {', '.join(candidate['strong_hits'])}."
    ]
    efficiency = []
    robustness = []

    if "performance_hot_path" in candidate["strong_hits"] or "performance_hot_path" in candidate["weak_hits"]:
        efficiency.append("Ataca um ponto quente de execucao com mecanismo estrutural plausivel.")
    if candidate["key"] in {"strategy", "state", "factory-method", "abstract-factory", "decorator", "proxy"}:
        efficiency.append("Reduz custo de manutencao ao isolar variacao e concrete classes.")
    else:
        efficiency.append("Reduz friccao de manutencao ao organizar responsabilidade estrutural recorrente.")

    if "preserve_public_contract" in constraints:
        robustness.append("Permite reorganizar a estrutura preservando contrato publico.")
    if candidate["key"] in {"proxy", "chain-of-responsibility", "state", "template-method", "command"}:
        robustness.append("Deixa o fluxo ou a governanca de execucao mais explicitos e testaveis.")
    else:
        robustness.append("Isola a variacao em fronteiras pequenas, reduzindo efeito colateral.")

    return economy, efficiency, robustness


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, help="Arquivo JSON ou '-' para stdin")
    args = parser.parse_args()

    try:
        data = load_input(args.input)
        problem = str(data.get("problem", "")).strip()
        signals, unknown_signals = normalize_list(data.get("signals"), ALLOWED_SIGNALS, "signals")
        constraints, unknown_constraints = normalize_list(
            data.get("constraints"), ALLOWED_CONSTRAINTS, "constraints"
        )
        force_reject, unknown_force_reject = normalize_list(
            data.get("force_reject"), set(PATTERN_BY_KEY), "force_reject"
        )
    except FileNotFoundError as exc:
        print(f"INPUT ERROR: arquivo nao encontrado - {exc}", file=sys.stderr)
        return 1
    except json.JSONDecodeError as exc:
        print(f"INPUT ERROR: JSON invalido - {exc}", file=sys.stderr)
        return 1
    except ValueError as exc:
        print(f"INPUT ERROR: {exc}", file=sys.stderr)
        return 1

    evidence_gaps: list[str] = []
    if not problem:
        evidence_gaps.append("Campo 'problem' ausente ou vazio.")
    if not signals:
        evidence_gaps.append("Nenhum sinal canonico informado.")
    if unknown_signals:
        evidence_gaps.append(f"Sinais desconhecidos: {', '.join(unknown_signals)}.")
    if unknown_constraints:
        evidence_gaps.append(f"Restricoes desconhecidas: {', '.join(unknown_constraints)}.")
    if unknown_force_reject:
        evidence_gaps.append(f"Patterns invalidos em force_reject: {', '.join(unknown_force_reject)}.")
    if evidence_gaps:
        print(
            json.dumps(
                {
                    "status": "needs_more_evidence",
                    "primary_pattern": None,
                    "complementary_pattern": None,
                    "simpler_alternative": "Coletar evidencias adicionais",
                    "rejected_patterns": [],
                    "candidate_scores": [],
                    "economy_case": [],
                    "efficiency_case": [],
                    "robustness_case": [],
                    "evidence_gaps": evidence_gaps,
                },
                ensure_ascii=True,
                indent=2,
            )
        )
        return 0

    if direct_solution_preferred(signals, constraints):
        print(
            json.dumps(
                {
                    "status": "reject",
                    "primary_pattern": None,
                    "complementary_pattern": None,
                    "simpler_alternative": "Usar solucao direta, refactor local ou composicao simples",
                    "rejected_patterns": [],
                    "candidate_scores": [],
                    "economy_case": [
                        "A combinacao de prefer_direct_solution, single_variant_only ou low_change_frequency indica retorno insuficiente para formalizar um pattern."
                    ],
                    "efficiency_case": [
                        "A opcao simples reduz indirecao e custo cognitivo."
                    ],
                    "robustness_case": [
                        "Menos tipos e menos acoplamento estrutural diminuem a superficie de falha."
                    ],
                    "evidence_gaps": [],
                },
                ensure_ascii=True,
                indent=2,
            )
        )
        return 0

    scored = [score_pattern(rule, signals, constraints) for rule in PATTERNS if rule.key not in force_reject]
    valid = [item for item in scored if candidate_is_valid(item)]
    valid.sort(key=lambda item: (-item["score"], item["name"]))

    if not valid:
        print(
            json.dumps(
                {
                    "status": "reject",
                    "primary_pattern": None,
                    "complementary_pattern": None,
                    "simpler_alternative": "Usar solucao direta, modulo pequeno ou refactor local",
                    "rejected_patterns": [],
                    "candidate_scores": scored,
                    "economy_case": [
                        "Nenhum pattern atingiu score e evidencia suficientes para pagar o custo estrutural."
                    ],
                    "efficiency_case": [
                        "A indirecao adicional seria maior que o beneficio provavel."
                    ],
                    "robustness_case": [
                        "A ausencia de sinais fortes torna a recomendacao arriscada."
                    ],
                    "evidence_gaps": [],
                },
                ensure_ascii=True,
                indent=2,
            )
        )
        return 0

    top_candidates = valid[:3]
    ambiguity = find_ambiguity(top_candidates)
    if ambiguity:
        print(
            json.dumps(
                {
                    "status": "ambiguous",
                    "primary_pattern": None,
                    "complementary_pattern": None,
                    "simpler_alternative": "Coletar a menor evidencia adicional para separar os candidatos",
                    "rejected_patterns": [],
                    "candidate_scores": top_candidates,
                    "economy_case": [],
                    "efficiency_case": [],
                    "robustness_case": [],
                    "evidence_gaps": [ambiguity],
                },
                ensure_ascii=True,
                indent=2,
            )
        )
        return 0

    primary = valid[0]
    complementary = None
    if len(valid) > 1:
        second = valid[1]
        allowed_pairs = {
            ("facade", "adapter"),
            ("composite", "iterator"),
            ("composite", "visitor"),
            ("proxy", "decorator"),
            ("state", "strategy"),
        }
        if (primary["key"], second["key"]) in allowed_pairs and primary["score"] - second["score"] <= 2:
            complementary = {"pattern": second["name"], "reason": "Complementar e nao substituto do pattern primario."}

    economy_case, efficiency_case, robustness_case = build_reason_sections(primary, constraints)
    rejected_patterns = []
    for item in valid[1:4]:
        rejected_patterns.append(
            {
                "pattern": item["name"],
                "reason": "Pontuou abaixo do primario ou adiciona custo estrutural desnecessario neste contexto.",
            }
        )
    for key in sorted(force_reject):
        rejected_patterns.append({"pattern": PATTERN_BY_KEY[key].name, "reason": "Bloqueado explicitamente por force_reject."})

    if primary["key"] in OVERENGINEERING_HINTS:
        economy_case.append("O pattern escolhido tem barra alta de uso; a recomendacao so e valida porque ha sinais fortes suficientes.")

    print(
        json.dumps(
            {
                "status": "ok",
                "primary_pattern": {
                    "pattern": primary["name"],
                    "group": primary["group"],
                    "score": primary["score"],
                    "strong_hits": primary["strong_hits"],
                    "weak_hits": primary["weak_hits"],
                    "blockers_considered": primary["blocker_hits"],
                },
                "complementary_pattern": complementary,
                "simpler_alternative": primary["simpler_alternative"],
                "rejected_patterns": rejected_patterns,
                "candidate_scores": valid[:5],
                "economy_case": economy_case,
                "efficiency_case": efficiency_case,
                "robustness_case": robustness_case,
                "evidence_gaps": [],
            },
            ensure_ascii=True,
            indent=2,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

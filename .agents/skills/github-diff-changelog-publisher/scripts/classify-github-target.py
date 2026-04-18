#!/usr/bin/env python3
import json
import re
import sys


PATTERNS = [
    ("pull_request", re.compile(r"github\.com/([^/]+/[^/]+)/pull/(\d+)", re.IGNORECASE)),
    ("release", re.compile(r"github\.com/([^/]+/[^/]+)/(?:releases/tag|releases)/([^/?#]+)", re.IGNORECASE)),
    ("compare", re.compile(r"github\.com/([^/]+/[^/]+)/compare/([^?#]+)", re.IGNORECASE)),
    ("branch", re.compile(r"github\.com/([^/]+/[^/]+)/tree/([^/?#]+)", re.IGNORECASE)),
]


def fail(message: str) -> int:
    print(message, file=sys.stderr)
    return 1


def classify(raw: str) -> dict:
    target = raw.strip()
    if not target:
        raise ValueError("ERRO: informe um alvo GitHub nao vazio.")

    for source_type, pattern in PATTERNS:
        match = pattern.search(target)
        if match:
            repository = match.group(1)
            identifier = match.group(2)
            return {
                "source_type": source_type,
                "repository": repository,
                "identifier": identifier,
                "target": target,
                "status": "ok",
            }

    compare_ref_match = re.match(r"([^/]+/[^/]+)\s+([^.\s]+)\.\.\.([^.\s]+)$", target)
    if compare_ref_match:
        return {
            "source_type": "compare",
            "repository": compare_ref_match.group(1),
            "identifier": f"{compare_ref_match.group(2)}...{compare_ref_match.group(3)}",
            "target": target,
            "status": "ok",
        }

    branch_ref_match = re.match(r"([^/]+/[^/]+)\s+(?:branch|ramo|filial)\s+(.+)$", target, re.IGNORECASE)
    if branch_ref_match:
        return {
            "source_type": "branch",
            "repository": branch_ref_match.group(1),
            "identifier": branch_ref_match.group(2).strip(),
            "target": target,
            "status": "ok",
        }

    pr_ref_match = re.match(r"([^/]+/[^/]+)\s+(?:pr|pull-request|pull request)\s+(\d+)$", target, re.IGNORECASE)
    if pr_ref_match:
        return {
            "source_type": "pull_request",
            "repository": pr_ref_match.group(1),
            "identifier": pr_ref_match.group(2),
            "target": target,
            "status": "ok",
        }

    release_ref_match = re.match(r"([^/]+/[^/]+)\s+(?:release|tag)\s+(.+)$", target, re.IGNORECASE)
    if release_ref_match:
        return {
            "source_type": "release",
            "repository": release_ref_match.group(1),
            "identifier": release_ref_match.group(2).strip(),
            "target": target,
            "status": "ok",
        }

    return {
        "source_type": "unknown",
        "repository": None,
        "identifier": None,
        "target": target,
        "status": "needs-input",
    }


def main() -> int:
    if len(sys.argv) != 2:
        return fail("ERRO DE USO: execute com exatamente um alvo GitHub.")

    try:
        result = classify(sys.argv[1])
    except ValueError as exc:
        return fail(str(exc))

    if result["status"] != "ok":
        return fail(
            "ERRO DE CLASSIFICACAO: alvo nao suportado. Informe uma URL de release, PR, compare, branch, ou '{owner}/{repo}' com identificador."
        )

    print(json.dumps(result, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

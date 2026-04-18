#!/usr/bin/env python3
import re
import sys
from pathlib import Path


REQUIRED_PATTERNS = {
    "objective": r"\bobjective\b|\bobjetivo\b",
    "constraints": r"\bconstraints\b|\brestricoes\b|\brestrições\b",
    "output": r"\boutput contract\b|\bsaida\b|\bsaída\b|\bformat\b|\bjson\b",
}

VAGUE_PATTERNS = [
    r"\bdo your best\b",
    r"\bbe creative\b",
    r"\bhandle everything\b",
    r"\bwhatever is needed\b",
    r"\bimprove as much as possible\b",
]


def fail(message: str) -> int:
    print(message, file=sys.stderr)
    return 1


def main() -> int:
    if len(sys.argv) != 2:
        return fail("ERRO DE USO: informe exatamente um caminho de arquivo de prompt.")

    path = Path(sys.argv[1])
    if not path.exists():
        return fail(f"ERRO DE ARQUIVO: '{path}' nao existe.")
    if not path.is_file():
        return fail(f"ERRO DE ARQUIVO: '{path}' nao e um arquivo regular.")

    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return fail("ERRO DE CODIFICACAO: o arquivo do prompt deve estar em UTF-8.")

    lowered = text.lower()
    errors = []
    warnings = []

    for label, pattern in REQUIRED_PATTERNS.items():
        if not re.search(pattern, lowered):
            errors.append(f"ERRO DE ESTRUTURA: falta secao ou sinal para '{label}'.")

    if len(text.strip()) < 120:
        warnings.append("AVISO DE QUALIDADE: o prompt pode estar curto demais para codificar restricoes com seguranca.")

    if len(text) > 4000:
        warnings.append("AVISO DE ECONOMIA: o prompt esta longo; remova contexto nao essencial.")

    for pattern in VAGUE_PATTERNS:
        match = re.search(pattern, lowered)
        if match:
            warnings.append(f"AVISO DE PRECISAO: diretiva vaga detectada: '{match.group(0)}'.")

    if "must" in lowered and "do not" not in lowered and "avoid" not in lowered:
        warnings.append("AVISO DE ROBUSTEZ: o prompt tem instrucoes obrigatorias, mas nao tem guarda de negativa.")

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    print("SUCESSO: o prompt passou na validacao estrutural.")
    if warnings:
        print("\n".join(warnings))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

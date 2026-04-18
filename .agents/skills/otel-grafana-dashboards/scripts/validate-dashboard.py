#!/usr/bin/env python3
"""Valida arquivos JSON de dashboards Grafana para corretude e boas práticas."""

import json
import sys
import argparse
import os


def validate_dashboard(file_path: str) -> list[str]:
    errors = []
    warnings = []

    # 1. Verificar se o arquivo existe e é JSON válido
    if not os.path.exists(file_path):
        errors.append(f"ERRO DE ARQUIVO: '{file_path}' não existe.")
        return errors

    try:
        with open(file_path, "r") as f:
            dashboard = json.load(f)
    except json.JSONDecodeError as e:
        errors.append(f"ERRO DE JSON: JSON inválido em '{file_path}': {e}")
        return errors

    # 2. Verificar campos obrigatórios de nível superior
    required_fields = ["uid", "title", "panels", "templating", "schemaVersion"]
    for field in required_fields:
        if field not in dashboard:
            errors.append(f"ERRO DE ESTRUTURA: Campo obrigatório '{field}' ausente.")

    # 3. Verificar formato do UID
    uid = dashboard.get("uid", "")
    if not uid or uid == "DASHBOARD_UID":
        errors.append("ERRO DE UID: O UID do dashboard deve ser definido (não placeholder).")

    # 4. Verificar título
    title = dashboard.get("title", "")
    if not title or title == "DASHBOARD_TITLE":
        errors.append("ERRO DE TITULO: O título do dashboard deve ser definido (não placeholder).")

    # 5. Validar painéis
    panels = dashboard.get("panels", [])
    if not panels:
        warnings.append("AVISO: Dashboard não possui painéis.")

    panel_ids = set()
    for i, panel in enumerate(panels):
        panel_id = panel.get("id")
        panel_type = panel.get("type", "unknown")

        # Verificar ID do painel
        if panel_id is None:
            errors.append(f"ERRO DE PAINEL: Painel {i} ('{panel.get('title', 'sem título')}') não possui ID.")
        elif panel_id in panel_ids:
            errors.append(f"ERRO DE PAINEL: ID duplicado {panel_id} no painel '{panel.get('title', 'sem título')}'.")
        else:
            panel_ids.add(panel_id)

        # Verificar se o painel possui título
        if not panel.get("title") and panel_type != "row":
            warnings.append(f"AVISO: Painel {panel_id} não possui título.")

        # Verificar gridPos
        grid_pos = panel.get("gridPos")
        if not grid_pos:
            errors.append(f"ERRO DE PAINEL: Painel {panel_id} ('{panel.get('title', 'sem título')}') não possui gridPos.")
        else:
            if grid_pos.get("w", 0) > 24:
                errors.append(f"ERRO DE GRID: Largura do painel {panel_id} ({grid_pos['w']}) excede o grid de 24 colunas.")
            if grid_pos.get("x", 0) + grid_pos.get("w", 0) > 24:
                errors.append(f"ERRO DE GRID: Painel {panel_id} ultrapassa o grid de 24 colunas (x={grid_pos['x']}, w={grid_pos['w']}).")

        # Verificar se referências de datasource usam variáveis
        if panel_type not in ("row",):
            ds = panel.get("datasource", {})
            if isinstance(ds, dict):
                ds_uid = ds.get("uid", "")
            elif isinstance(ds, str):
                ds_uid = ds
            else:
                ds_uid = ""

            if ds_uid and not ds_uid.startswith("${DS_") and ds_uid != "-- Grafana --":
                errors.append(
                    f"ERRO DE DATASOURCE: Painel {panel_id} ('{panel.get('title', 'sem título')}') usa "
                    f"datasource hardcoded '{ds_uid}'. Use variáveis de template (${{DS_PROMETHEUS}}, etc.)."
                )

        # Verificar targets com intervalos de rate hardcoded
        targets = panel.get("targets", [])
        for t_idx, target in enumerate(targets):
            expr = target.get("expr", "")
            if "rate(" in expr and "[5m]" in expr:
                warnings.append(
                    f"AVISO: Painel {panel_id}, target {t_idx}: Usa intervalo hardcoded '[5m]'. "
                    "Considere usar '$__rate_interval'."
                )
            if "rate(" in expr and "[1m]" in expr:
                warnings.append(
                    f"AVISO: Painel {panel_id}, target {t_idx}: Usa intervalo hardcoded '[1m]'. "
                    "Considere usar '$__rate_interval'."
                )

        # Verificar painéis aninhados em linhas
        if panel_type == "row":
            nested = panel.get("panels", [])
            for np in nested:
                np_id = np.get("id")
                if np_id is not None:
                    if np_id in panel_ids:
                        errors.append(f"ERRO DE PAINEL: ID duplicado {np_id} no painel aninhado '{np.get('title', 'sem título')}'.")
                    else:
                        panel_ids.add(np_id)

    # 6. Verificar variáveis de template
    templating = dashboard.get("templating", {})
    template_list = templating.get("list", [])
    template_names = {t.get("name") for t in template_list}

    required_vars = {"DS_PROMETHEUS", "service", "environment"}
    missing_vars = required_vars - template_names
    if missing_vars:
        errors.append(f"ERRO DE TEMPLATE: Variáveis de template obrigatórias ausentes: {missing_vars}")

    # 7. Verificar se tags incluem 'opentelemetry'
    tags = dashboard.get("tags", [])
    if "opentelemetry" not in tags:
        warnings.append("AVISO: Tags do dashboard devem incluir 'opentelemetry' para descoberta.")

    # 8. Verificar intervalo de atualização
    refresh = dashboard.get("refresh")
    if not refresh:
        warnings.append("AVISO: Nenhum intervalo de atualização automática definido.")

    # Imprimir resultados
    if warnings:
        for w in warnings:
            print(w, file=sys.stderr)

    return errors


def main():
    parser = argparse.ArgumentParser(description="Valida arquivos JSON de dashboards Grafana.")
    parser.add_argument("--file", required=True, help="Caminho para o arquivo JSON do dashboard a ser validado.")
    args = parser.parse_args()

    errors = validate_dashboard(args.file)

    if errors:
        print("\n".join(errors), file=sys.stderr)
        sys.exit(1)
    else:
        print(f"SUCESSO: '{args.file}' é um dashboard Grafana válido.")
        sys.exit(0)


if __name__ == "__main__":
    main()

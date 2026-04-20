# ADR-005: Skills externas com lock file e verificacao SHA-256

**Status:** Aceita  
**Data:** 2024-01-01  
**Autores:** JailtonJunior94

---

## Contexto

O harness suporta skills externas — artefatos de governanca hospedados fora do repositorio principal (ex: no repositorio de skills da organizacao). Essas skills sao instaladas em `.agents/skills/` e precisam ser gerenciadas ao longo do ciclo de vida do projeto.

O problema e: como garantir que skills externas instaladas sao autenticas, integras e reproduziveis entre ambientes?

Restricoes relevantes:
- Skills externas podem ser atualizadas no repositorio de origem sem aviso.
- Reproducibilidade entre CI, maquinas de desenvolvimento e producao e critica.
- Nao e desejavel adicionar um sistema de gerenciamento de pacotes externo.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Lock file com SHA-256 (escolhida) | Integridade verificavel; reproducivel; sem dependencia extra; auditavel | Upgrade manual; hash deve ser recomputado a cada mudanca |
| Git submodule | Versionamento nativo do Git; diff claro | Complexidade de clone recursivo; problemas frequentes em CI |
| Download sem verificacao | Simples; sempre a versao mais recente | Sem integridade; risco de atualizacao silenciosa quebrar governanca |
| Vendoring completo | Controle total; sem dependencia de rede | Repositorio maior; conflitos de merge em atualizacoes |

## Decisao

Decidimos usar `skills-lock.json` na raiz do repositorio como arquivo de lock para skills externas. Cada entrada registra o nome da skill, a URL de origem e o hash SHA-256 do conteudo no momento da instalacao.

O comando `upgrade` do harness verifica o hash atual contra o registrado; divergencias sao reportadas como violacoes de integridade.

## Consequencias

### Positivas
- Qualquer instalacao com o mesmo `skills-lock.json` produz exatamente os mesmos arquivos.
- Alteracoes nao autorizadas em skills instaladas sao detectadas imediatamente.
- O lock file e um registro auditavel de quais versoes de skills estao em uso.
- Sem dependencias externas alem do proprio harness.

### Negativas / Riscos
- Upgrades de skills requerem acao manual: baixar nova versao, recomputar hash, atualizar lock file.
- Se o repositorio de origem remover ou renomear uma skill, o lock file fica invalido.
- Hash SHA-256 nao garante autenticidade (nao e assinatura criptografica); garante apenas integridade pos-download.

### Neutras / Observacoes
- O modelo e analogo ao `go.sum` do ecossistema Go e ao `package-lock.json` do Node.js.
- Skills embarcadas no baseline (`internal/embedded/`) nao sao gerenciadas pelo lock file; apenas skills externas o sao.

## Referencias

- `skills-lock.json` — arquivo de lock na raiz do repositorio
- `internal/skills/` — tipos de dominio e logica de verificacao de skills
- `cmd/ai_spec_harness/` — comandos `install` e `upgrade` que consomem o lock file

# Fluxo de Comandos com `gh`

Usar estes comandos como referência operacional. Adaptar somente os placeholders.

## Descobrir Repositório e PR

```bash
gh repo view --json nameWithOwner -q .nameWithOwner
gh pr view <pr> --json number,url,title
```

## Coletar Comentários Gerais da PR

```bash
gh api repos/<owner>/<repo>/issues/<pr>/comments --paginate
```

## Coletar Comentários Inline de Review

```bash
gh api repos/<owner>/<repo>/pulls/<pr>/comments --paginate
```

## Publicar Resposta como Comentário Geral na PR

```bash
gh pr comment <pr> --repo <owner>/<repo> --body-file <reply-file>
```

## Publicar Resposta Encadeada a Comentário Inline

Criar a resposta usando a rota REST de criação de review comment com `in_reply_to`, conforme a documentação oficial do GitHub para review comments.

```bash
gh api repos/<owner>/<repo>/pulls/<pr>/comments \
  --method POST \
  --input <reply-payload.json>
```

Payload esperado em `<reply-payload.json>`:

```json
{
  "body": "texto da resposta em pt-BR",
  "in_reply_to": 123456
}
```

## Observações

1. Preferir `--body-file` quando o comando suportar arquivo diretamente.
2. Quando a rota exigir corpo JSON, preferir `--input <arquivo.json>` para evitar escaping frágil.
3. Não publicar nada antes da confirmação explícita do usuário para o item correspondente.

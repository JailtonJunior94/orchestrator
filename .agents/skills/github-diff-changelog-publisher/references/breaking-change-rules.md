# Regras para Breaking Changes

Use este arquivo somente ao avaliar risco de compatibilidade e migração.

Trate uma mudança como provável breaking change quando o diff mostrar um ou mais destes sinais:

- remove endpoint público de API, rota, comando CLI, evento, export ou membro de interface
- renomeia contrato público sem camada de compatibilidade
- altera request, response, payload, schema ou formato de config de forma incompatível com versões anteriores
- endurece validação, auth, permissões ou campos obrigatórios de forma que passe a rejeitar comportamentos antes válidos
- remove variáveis de ambiente, feature flags, defaults ou pré-requisitos de deploy
- altera schema de banco, ordem de migração ou formato de dados com impacto de mão única
- remove suporte a runtime, framework, biblioteca ou plataforma

Trate uma mudança como possível breaking change quando:

- o diff sugere alterações de contrato, mas o código ou a documentação ao redor não estão disponíveis
- uma refatoração grande move arquivos que parecem públicos, mas sem chamadas visíveis
- arquivos gerados mudam, mas o contrato de origem não está disponível

Não rotule uma mudança como breaking change com base apenas em:

- refatorações internas com interfaces preservadas
- diffs apenas de formatação
- mudanças apenas em testes
- edições apenas de documentação
- atualização de dependências sem impacto contratual visível

Quando um breaking change for identificado, inclua:

1. o que mudou
2. quem é impactado
3. qual migração ou verificação é necessária
4. quão certa é a conclusão

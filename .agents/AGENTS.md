# Stock Pulse - Agent Rules & Guidelines

Este arquivo define as diretrizes arquiteturais, regras de negócio e padrões de código do **Stock Pulse**. Qualquer agente de IA deve seguir essas instruções estritamente para manter a consistência do projeto.

## 1. Comparações de Ponto Flutuante (Valores Financeiros)
Nunca comparar valores monetários (`float64`) usando operadores de igualdade (`==`, `!=`). Utilize sempre uma margem de tolerância (ex: `< 1e-6` para "igual a zero", ou a constante de negócio apropriada para comparações de proximidade). Isso se aplica a qualquer cálculo financeiro no projeto.

## 2. Otimização de Workers: Skip quando dados são idênticos
Todo worker de sincronização de dados (como o `DividendWorker`) deve implementar uma lógica de **Skip**: se o dado lido da fonte externa for matematicamente idêntico ao existente no banco de dados (respeitando tolerância para floats e comparação correta de datas), a operação de `UPDATE` deve ser suprimida. Objetivo: evitar escritas desnecessárias no banco de dados.

## 3. Modelo de Dados: Asset Events (Proventos)
- A tabela `asset_events` armazena proventos (dividendos, JCP, rendimentos, amortizações).
- O campo `cum_date` representa a **"Data Com / Data Base"** (data em que o investidor precisava ter o ativo para receber o provento).
- A chave de unicidade de um provento na tabela é a combinação exata: `(asset_id, cum_date, type, gross_amount)`.
- Ao integrar com qualquer fonte de dados (ex: scraper Fundamentus), mapear a data de referência/aprovação do provento para `cum_date`.

## 4. Resolução e Fuzzy Matching de Proventos (`internal/dividend`)
Toda a coleta, normalização, deduplicação, validação e resolução (Fuzzy Match) de proventos é centralizada exclusivamente no pacote `backend/internal/dividend`, seguindo o pipeline:
1. **Normalize**: Traduz tipos para `"Dividendo"`, `"JCP"`, `"Rendimento"` ou `"Amortização"` e datas para `00:00:00 UTC`.
2. **Deduplicate**: Remove registros duplicados dentro da lista entregue pela fonte externa.
3. **Validate**: Exige `cum_date` preenchida, `payment_date` preenchida e `amount > 1e-6` (proventos sem `payment_date` são descartados).
4. **Resolve (Fuzzy Match contra o banco)**:
   - Busca proventos existentes com o mesmo `asset_id` e `cum_date` (e mesmo `type`, ou conciliando `"Dividendo"` genérico com específico quando possuem mesma `payment_date`).
   - Diferença **≤ R$ 0,05**: considera como o mesmo provento.
   - Diferença **< 1e-6**, mesma `payment_date` e mesmo `type`: faz **skip** (não atualiza).
   - Diferença **≤ 0,05** (ou atualização de `payment_date`/`type`): **atualiza** o registro existente.
   - Diferença **> 0,05** (ou sem par): **insere** um novo registro.
   - A leitura em `portfolio/dividend_service.go` e no Frontend (`DividendsHistory.tsx`) **não** deve reexecutar deduplicação de proventos reais.

## 5. Bot do Telegram: Resolver carteira ativa corretamente
- A carteira ativa de um usuário no Telegram é armazenada e controlada de forma independente no **Redis** (chave `telegram_active_portfolio:<chatID>`).
- Todo handler do Telegram que precisa acessar ou manipular dados da carteira do usuário **deve** usar o método `resolveActivePortfolio()`.
- **Nunca** use `portfolios[0]` diretamente ou assuma uma carteira hardcoded, mesmo dentro de callbacks de botões inline (como paginação e filtros de agrupamento).

## 6. Workflow de Git: Branches e Pull Requests
- Toda alteração de código ou documentação deve ser feita em uma branch separada, seguindo os padrões: `feat/`, `fix/`, `chore/` ou `docs/`.
- Após commitar suas alterações, faça push para o remote e abra uma Pull Request via `gh pr create`.
- **Nunca** faça push direto na `master`. Aguarde o merge da sua PR pelo usuário antes de prosseguir com tarefas dependentes.
- Antes de iniciar qualquer novo trabalho, sempre sincronize o repositório local com a branch principal usando `git fetch origin master && git rebase origin/master`.

## 7. Privacidade: Sem Dados Reais da Carteira no Código ou PRs
- Nunca inclua tickers específicos, nomes de ativos, quantidades exatas, preços médios ou datas reais informadas pelo usuário em conversas dentro de código, comentários, testes, mensagens de commit ou títulos/descrições de Pull Requests.
- Utilize sempre fixtures genéricas e fictícias (como `ORIG11.SA`, `DEST11.SA`, `ABCD11`, `PETR4`, `AAPL`) e valores numéricos redondos e meramente ilustrativos.


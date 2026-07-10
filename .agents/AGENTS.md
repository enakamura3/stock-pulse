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

## 4. Fuzzy Matching de Proventos (Regra de Negócio)
Ao sincronizar proventos da fonte externa, antes de inserir um novo registro:
1. Busque proventos existentes no banco com o mesmo `asset_id` e `cum_date`.
2. Filtre pelo mesmo `type`.
3. Se houver um provento existente cuja diferença absoluta de `gross_amount` seja **≤ R$ 0,05**, considere como o mesmo provento (trata-se de um ajuste de centavos da fonte).
4. Se a diferença for **zero** (< 1e-6) e a `payment_date` também for idêntica, faça **skip** (não atualize).
5. Se a diferença for ≤ 0,05 mas não for zero, **atualize** o registro existente com o novo valor.
6. Se a diferença for > 0,05, considere como um **provento diferente** e insira um novo registro.

## 5. Bot do Telegram: Resolver carteira ativa corretamente
- A carteira ativa de um usuário no Telegram é armazenada e controlada de forma independente no **Redis** (chave `telegram_active_portfolio:<chatID>`).
- Todo handler do Telegram que precisa acessar ou manipular dados da carteira do usuário **deve** usar o método `resolveActivePortfolio()`.
- **Nunca** use `portfolios[0]` diretamente ou assuma uma carteira hardcoded, mesmo dentro de callbacks de botões inline (como paginação e filtros de agrupamento).

## 6. Workflow de Git: Branches e Pull Requests
- Toda alteração de código ou documentação deve ser feita em uma branch separada, seguindo os padrões: `feat/`, `fix/`, `chore/` ou `docs/`.
- Após commitar suas alterações, faça push para o remote e abra uma Pull Request via `gh pr create`.
- **Nunca** faça push direto na `master`. Aguarde o merge da sua PR pelo usuário antes de prosseguir com tarefas dependentes.
- Antes de iniciar qualquer novo trabalho, sempre sincronize o repositório local com a branch principal usando `git fetch origin master && git rebase origin/master`.

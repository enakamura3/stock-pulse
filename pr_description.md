### 🚀 Resumo
Este Pull Request implementa a tão requisitada funcionalidade de **Exportação de Backup (CSV)** da carteira de investimentos e corrige um bug crítico envolvendo a **taxa de câmbio** para ativos globais. Também expande significativamente a documentação do projeto.

### ✨ Novas Funcionalidades
- **Exportação de Carteira (Frontend):** Adicionado o botão "📥 Exportar Backup" na interface do usuário (aba de Carteiras), integrado para receber e descompactar buffers ZIP em memória.
- **Endpoint de Backup em ZIP (Backend):** O backend agora expõe a rota `GET /api/portfolios/{id}/export` encarregada de orquestrar a extração de dados brutos das camadas de Renda Variável e Renda Fixa num único arquivo compactado (`backup-carteira.zip`).
  - *Formatos Adotados:* Formato internacional de datas (`YYYY-MM-DD`), separador de colunas com `;` (ponto e vírgula) e separador decimal com `.` (ponto), assegurando legibilidade perfeita nas suítes office do Brasil e do mundo.

### 🐛 Correções de Bugs (Bug Fixes)
- **Correção Cambial Automática (Renda Variável):** 
  - *Problema:* Ativos de moeda estrangeira (ex: `IVV`) registrados sem uma taxa de câmbio manual recebiam o valor default `1.0`, distorcendo fatalmente o cálculo de Custo de Aquisição.
  - *Solução:* O serviço do portfólio agora intercepta transações globais onde o campo `exchange_rate` venha vazio ou zerado, disparando automaticamente a busca pelo histórico cambial exato (`GetHistoricalExchangeRate`) atrelado à data da compra. A trava que engessava todos os registros em `1.0` no Handler foi sumariamente removida, restando válida apenas para ativos de mesma moeda nativa (ex: BRL -> BRL).

### 📚 Atualizações na Documentação (README.md)
- Adicionada a subseção **Mecânica de Cálculos Internos e Câmbio** detalhando o papel universal do multiplicador da taxa de câmbio para suporte nativo Multi-Moedas.
- Adicionada a subseção **Mecânica Interna de Desdobramentos e Agrupamentos** detalhando matematicamente o ajuste do Preço Médio (`SPLIT` e `REVERSE_SPLIT`) e o comportamento de Backtesting retroativo ("Look-Ahead") usado no gráfico de evolução histórica patrimonial.

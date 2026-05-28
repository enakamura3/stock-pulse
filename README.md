# stock-pulse 📈

stock-pulse é uma plataforma moderna e completa para acompanhamento de portfólios de investimentos, listas de favoritos (watchlists) e configuração de alertas de preços em tempo real. O sistema possui arquitetura baseada em micro-serviços orquestrados via Docker, com um backend robusto em Golang e um frontend moderno em Next.js.

## 🚀 Funcionalidades

- **Monitoramento em Tempo Real:** Conexões via WebSocket garantem que cotações de ativos pisquem na tela sem necessidade de recarregar a página.
- **Gestão de Portfólio:** Acompanhe rentabilidade, histórico de transações e custo médio dos seus ativos globais ou da B3. Suporta edição de transações, Desdobramentos/Agrupamentos (Splits), Bonificações e Importação em Lote (Bulk Import via CSV).
- **Watchlists Múltiplas:** Crie listas de favoritos customizadas para separar ativos por estratégia.
- **Valuation e Indicadores (P/VP, P/L, Yield):** Calcule o Preço Justo de ações segundo as metodologias de Benjamin Graham e Décio Bazin. Acompanhe em tempo real na sua tabela de posições ativas os múltiplos P/VP, P/L e Dividend Yield atualizados via scraping (suporte inteligente a Ações e FIIs da B3 via Fundamentus, e ativos globais via Finviz). Receba alertas visuais diretamente na tabela identificando oportunidades (ex: P/VP < 1.0 ou preços descontados ficam destacados em verde).
- **Visualização Avançada de Histórico:** Histórico de transações em layout fluido (single-line), com filtros inteligentes por Ticker para auditar operações e custos passados com facilidade.
- **Alertas de Preço (E-mail):** Configure alertas disparados automaticamente em background quando um preço atinge uma meta, recebendo um e-mail HTML estilizado.
- **Segurança Sólida:** Autenticação usando JWT armazenado exclusivamente em cookies `HttpOnly` com criptografia e regras de CORS restritas.
- **Observabilidade Total:** Telemetria integrada com Prometheus, Grafana e Loki para métricas e logs em tempo real.

---

## 🛠️ Stack Tecnológico

### Backend (Golang 1.24)
- **Roteamento & HTTP:** `go-chi`
- **Banco de Dados Relacional:** PostgreSQL 16 (driver `pgx/v5` via pool de conexões)
- **Cache & Sessão:** Redis 7 (`go-redis/v9`)
- **Autenticação:** JWT (JSON Web Tokens)
- **Migrações de DB:** `golang-migrate`
- **Fornecedor de Dados de Mercado:** Yahoo Finance API (Cotações e Busca), Fundamentus & Finviz (Scraping de Fundamentos)
- **Background Workers:** Goroutines para verificação de alertas e rotinas de portfólio.

### Frontend (Next.js 14)
- **Framework:** React 18 com TypeScript
- **Estilização:** CSS puro ("Glassmorphism", interfaces dark mode premium)
- **Gráficos:** Lightweight Charts (TradingView)
- **Testes Unitários:** Vitest & React Testing Library (100% de cobertura)
- **Testes E2E:** Playwright

### Infraestrutura & DevOps
- **Orquestração:** Docker Compose
- **Proxy Reverso:** Caddy (Roteamento local e compressão gzip/zstd)
- **Mensageria SMTP:** Mailpit (Para captura de e-mails em desenvolvimento)
- **Monitoramento:** Prometheus, Grafana, Loki e Promtail.

---

## 📡 Fornecedores de Dados (Data Providers)

O stock-pulse não possui uma base de dados interna estática de ativos financeiros. Ele atua de forma dinâmica buscando informações atualizadas (Cotações e Fundamentos) através de integrações com APIs e Web Scraping:

### 1. Yahoo Finance API (Cotações e Busca)
Responsável por entregar as cotações em tempo real e fornecer a busca (autocomplete) de ativos.
- **Busca de Tickers (Search):**
  - `GET https://query1.finance.yahoo.com/v1/finance/search?q={query}`
- **Cotação Atual e Preço de Fechamento (Chart/Quote):**
  - `GET https://query1.finance.yahoo.com/v8/finance/chart/{symbol}?interval=1d&range=1d`
  
*(Nota: Para evitar bloqueios do Yahoo Finance, o backend injeta rotineiramente cabeçalhos `User-Agent` customizados nas requisições).*

### 2. Fundamentus (Scraping de Fundamentos - Brasil)
Como as APIs gratuitas do Yahoo não fornecem indicadores fundamentalistas estruturados e confiáveis para o Brasil, o backend faz o web scraping das páginas do Fundamentus para ativos com o sufixo `.SA` (Ações e FIIs da B3).
- **Endpoint Analisado:** 
  - `GET https://www.fundamentus.com.br/detalhes.php?papel={symbol}`
- **Métricas Extraídas via Regex:** Lucro Por Ação (LPA), Valor Patrimonial por Ação (VPA), VP/Cota (para FIIs) e Dividend Yield.

### 3. Finviz (Scraping de Fundamentos - Global)
Para ativos americanos ou globais (sem o sufixo `.SA`), o sistema roteia o scraping para o portal Finviz, que possui uma tabela rica de indicadores de mercado internacional.
- **Endpoint Analisado:**
  - `GET https://finviz.com/quote.ashx?t={symbol}`
- **Métricas Extraídas via Regex:** EPS (ttm), Book/sh e Dividend %.

---

### Importação de Transações em Lote (CSV)
O stock-pulse permite a importação massiva de histórico de operações através de um arquivo `.csv` ou `.txt`. 
O arquivo deve conter as colunas na seguinte ordem exata (o cabeçalho na primeira linha é ignorado):

`DATE, TICKER, TYPE, QUANTITY, PRICE`

- **DATE**: Formato internacional (`YYYY-MM-DD`) ou brasileiro (`DD/MM/YYYY`).
- **TICKER**: Código do ativo (ex: `PETR4.SA`, `AAPL`).
- **TYPE**: Define o comportamento das colunas Quantidade e Preço:
  - **`BUY` (Compra):** Quantidade e Preço devem ser maiores que zero.
  - **`SELL` (Venda):** Quantidade e Preço devem ser maiores que zero.
  - **`BONUS` (Bonificação):** Quantidade (ações recebidas) > 0. O preço deve ser o **Custo Atribuído** (declarado pela empresa no Fato Relevante). O sistema usará este valor para aumentar o seu custo total e recalcular o Preço Médio (se preferir não alterar o custo, informe `0.00`).
  - **`SPLIT` (Desdobramento):** Quantidade representa o fator de multiplicação (ex: `2` para 1 virar 2). O sistema ignora o preço (pode informar `0.00`).
  - **`REVERSE_SPLIT` (Agrupamento):** Quantidade representa o fator de divisão (ex: `10` para 10 virar 1). O sistema ignora o preço (pode informar `0.00`).

**Exemplo Completo de Arquivo CSV:**
```csv
DATE, TICKER, TYPE, QUANTITY, PRICE
2024-01-10, WEGE3, BUY, 100, 32.50
2024-02-15, WEGE3, SELL, 50, 38.00
2024-03-01, PETR4, BUY, 200, 35.10
2024-04-10, PETR4, BONUS, 20, 0.00
2024-05-20, ITUB4, SPLIT, 2, 0.00
2024-06-15, COGN3, REVERSE_SPLIT, 10, 0.00
```

---

## 📊 Arquitetura e Fluxos de Dados

Para entender melhor como os serviços se comunicam sob o capô, abaixo estão os diagramas de arquitetura e dos fluxos principais.

### 1. Diagrama de Blocos (Alto Nível)
Representa a orquestração via Docker Compose e como o tráfego externo é roteado até os provedores de dados.

```mermaid
graph TD
    Client[Navegador do Usuário] -->|HTTPS| Caddy[Caddy Proxy Reverso]
    Caddy -->|/api/*| GoAPI[Backend Go]
    Caddy -->|/| NextJS[Frontend Next.js]
    
    subgraph Data Layer
        GoAPI -->|Leitura/Escrita| PG[(PostgreSQL)]
        GoAPI -->|Cache Rápido| Redis[(Redis)]
    end
    
    subgraph Integrações
        GoAPI -->|Disparo de Alertas| Mailpit[Servidor SMTP Local]
        GoAPI -->|Cotações e Busca| Yahoo[Yahoo Finance API]
        GoAPI -->|Fundamentos B3| Fundamentus[Fundamentus]
        GoAPI -->|Fundamentos Globais| Finviz[Finviz]
    end
```

### 2. Fluxo de Cotações em Tempo Real (WebSockets)
Como o sistema entrega piscadas na tela instantaneamente ao cliente.

```mermaid
sequenceDiagram
    participant User as Frontend (React)
    participant API as Backend (Go Hub)
    participant Redis as Redis Cache
    participant Yahoo as Yahoo Finance

    User->>API: Conecta via WebSocket (/ws)
    User->>API: Envia "subscribe" para PETR4
    API->>Redis: Verifica cache de PETR4
    alt Não está no cache
        API->>Yahoo: GET /v8/finance/chart/PETR4.SA
        Yahoo-->>API: Retorna JSON
        API->>Redis: Salva novo preço no cache (TTL 3m)
    end
    API-->>User: Dispara BroadCast (Cotação PETR4)
    Note over User: Gráfico do Dashboard é atualizado!
```

### 3. Fluxograma de Alertas (Background Workers)
Goroutines rodando infinitamente em background para checar se o preço atingiu o alvo configurado.

```mermaid
flowchart TD
    Start[Worker Inicializado] --> FetchAlerts[Busca Alertas 'Ativos' no Banco]
    FetchAlerts --> Loop[Itera sobre cada Alerta]
    Loop --> CheckCache{Preço Atual no Cache?}
    
    CheckCache -- Sim --> Compare[Compara Preço Atual vs Preço Alvo]
    CheckCache -- Não --> FetchAPI[Busca preço na API e Salva no Cache] --> Compare
    
    Compare -- Atingiu Meta! --> Dispara[Gera Email e Envia ao Mailpit]
    Compare -- Não Atingiu --> Skip[Ignora]
    
    Dispara --> MarcaInativo[Atualiza Alerta para 'Disparado' no DB]
    MarcaInativo --> Loop
    Skip --> Loop
    
    Loop --> Sleep[Dorme 60s] --> FetchAlerts
```

---

## 📂 Arquitetura do Repositório (Monorepo)

```text
.
├── backend/          # Backend em Go (Domain-Driven Design)
│   ├── cmd/api/      # Ponto de entrada (main.go)
│   ├── internal/     # Regras de negócio (auth, market, portfolio, alert, websocket, etc.)
│   ├── migrations/   # Scripts SQL de versionamento do banco
│   └── Dockerfile    # Imagem Go com Air para Live Reload
│
├── frontend/         # Interface Web em Next.js
│   ├── src/app/      # Páginas (Login, Dashboard, Portfólio)
│   ├── tests/        # Testes End-to-End com Playwright
│   └── Dockerfile    # Imagem Node.js
│
├── docker-compose.yml # Arquivo principal que sobe 9 containers integrados
├── Makefile          # Atalhos para comandos comuns
└── Caddyfile         # Configuração de rotas para proxy reverso
```

---

## ⚙️ Como Executar Localmente

### Pré-requisitos
- Docker e Docker Compose instalados.
- Make instalado (Opcional, mas recomendado).

### Subindo o Ambiente

Apenas clone o repositório e utilize o Makefile na raiz do projeto:

```bash
# Para compilar e subir todos os containers (DB, Redis, Go, Next, Grafana, Mailpit...)
make build

# Para subir sem recompilar:
make up

# Para acompanhar os logs de todos os serviços:
make logs

# Para derrubar o ambiente:
make down
```

### Acessos Locais Pós-Deploy

O `Caddy` vai expor os serviços de forma elegante:

- **Frontend (Interface do Usuário):** [http://stock-pulse.localhost](http://stock-pulse.localhost) ou [http://localhost:3000](http://localhost:3000)
- **Backend (API Base):** [http://api.stock-pulse.localhost](http://api.stock-pulse.localhost) ou [http://localhost:8080](http://localhost:8080)
- **Mailpit (Caixa de Entrada Local para Alertas):** [http://localhost:8025](http://localhost:8025)
- **Grafana (Dashboards de Monitoramento):** [http://localhost:3001](http://localhost:3001) (Usuário/Senha Padrão: admin / admin)

---

## 🏗️ Como Criar Novas Migrações do Banco de Dados

Se você modificar a estrutura do banco de dados, crie uma nova migração utilizando o atalho do Makefile:

```bash
make migrate-create
```
*O console pedirá o nome da migração (ex: `add_users_table`) e gerará os arquivos `.up.sql` e `.down.sql` na pasta `backend/migrations`.*

---

## 🛡️ Segurança e Boas Práticas Implementadas

- Proteção JWT imune a ataques XSS através de Cookies `HttpOnly`.
- Validação CSRF com Strict/Lax SameSite modes.
- Fallback elegante se os provedores externos (Yahoo Finance) aplicarem Rate Limits, utilizando cache em Redis.
- Graceful Shutdown no Go para desligar os Background Workers e encerrar conexões com o PostgreSQL com segurança.

---

## 🧪 Testes e Cobertura (Unit Testing)

A plataforma stock-pulse foca em **alta qualidade de código**, visando 100% de cobertura nos testes unitários tanto no backend quanto no frontend.

### Backend (Golang)
O backend possui um conjunto rigoroso de testes simulando casos de sucesso e tratamento de erros avançados no banco de dados com `pgxmock` (simulação de erros em scan de rows, indisponibilidade, etc).
```bash
# Rodar todos os testes de backend e exibir a porcentagem de cobertura
cd backend
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Frontend (Next.js)
O frontend conta com cobertura de testes utilizando `Vitest` em conjunto com a `React Testing Library`. A suíte realiza _smoke testing_, testes de layout, validações de fluxo de formulários (Login e Registro) e _mocking_ de providers e contextos.
```bash
# Rodar testes de frontend com geração de relatório de coverage
cd frontend
npm run test:coverage
```
_Nota: Se você não possuir o Node instalado localmente, os testes podem ser executados dentro de um container Node isolado._

## ☁️ Arquitetura de Deploy (Cloud Gratuita)

O **stock-pulse** foi desenhado para ser facilmente distribuído em serviços de nuvem gratuitos (Free Tiers), permitindo que você hospede seu próprio ambiente de produção com **custo zero**:

- **Frontend:** [Vercel](https://vercel.com/) (Hospedagem nativa Next.js, Serverless & CDN Global)
- **Backend (API/Workers):** [Koyeb](https://koyeb.com/) (Serviço Eco gratuito para containers Docker rodando as rotinas 24/7) ou Google Cloud Platform (VM `e2-micro` Always Free)
- **Banco de Dados:** [Supabase](https://supabase.com/) (PostgreSQL dedicado gratuito de 500MB, backups diários)
- **Cache & WebSockets:** [Redis Cloud](https://redis.com/try-free/) (Cluster gerenciado de 30MB gratuito, ideal para cache temporário de cotações)

Com essa distribuição, o sistema evita gargalos de memória e ganha resiliência, separando a persistência (Supabase) da lógica computacional (Go) e da entrega de interface (Vercel).

---

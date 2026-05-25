# StockPulse 📈

StockPulse é uma plataforma moderna e completa para acompanhamento de portfólios de investimentos, listas de favoritos (watchlists) e configuração de alertas de preços em tempo real. O sistema possui arquitetura baseada em micro-serviços orquestrados via Docker, com um backend robusto em Golang e um frontend moderno em Next.js.

## 🚀 Funcionalidades

- **Monitoramento em Tempo Real:** Conexões via WebSocket garantem que cotações de ativos pisquem na tela sem necessidade de recarregar a página.
- **Gestão de Portfólio:** Acompanhe rentabilidade, histórico de transações e custo médio dos seus ativos globais ou da B3.
- **Watchlists Múltiplas:** Crie listas de favoritos customizadas para separar ativos por estratégia.
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
- **Fornecedor de Dados de Mercado:** Yahoo Finance API (Cotações e Busca)
- **Background Workers:** Goroutines para verificação de alertas e rotinas de portfólio.

### Frontend (Next.js 14)
- **Framework:** React 18 com TypeScript
- **Estilização:** CSS puro ("Glassmorphism", interfaces dark mode premium)
- **Gráficos:** Lightweight Charts (TradingView)
- **Testes E2E:** Playwright

### Infraestrutura & DevOps
- **Orquestração:** Docker Compose
- **Proxy Reverso:** Caddy (Roteamento local e compressão gzip/zstd)
- **Mensageria SMTP:** Mailpit (Para captura de e-mails em desenvolvimento)
- **Monitoramento:** Prometheus, Grafana, Loki e Promtail.

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

- **Frontend (Interface do Usuário):** [http://stockpulse.localhost](http://stockpulse.localhost) ou [http://localhost:3000](http://localhost:3000)
- **Backend (API Base):** [http://api.stockpulse.localhost](http://api.stockpulse.localhost) ou [http://localhost:8080](http://localhost:8080)
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

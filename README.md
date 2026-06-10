# stock-pulse 📈

stock-pulse is a comprehensive portfolio management and financial monitoring platform. The system features a microservices architecture orchestrated via Docker, composed of a robust Golang backend and a modern Next.js frontend, providing real-time pricing, watchlist tracking, and automated alert systems.

## 🚀 Core Features

- **Real-Time Data Streaming:** WebSocket connections ensure real-time asset price updates without page reloads.
- **Modular Portfolio Management:** Track profitability, transaction history, and average costs for global and B3 (Brazilian) assets. The interface is modularized into Variable Income, Fixed Income, Transactions, Dividends, and Journal views. Supports native transaction editing, stock splits, reverse splits, bonuses, and bulk import via CSV.
- **Dedicated Fixed Income Engine:** An isolated module for Fixed Income tracking (e.g., CDBs, Treasury Bonds) featuring exclusive compound interest evolution charts and standardized net yield tables. Includes a daily yield simulator that integrates **Accumulated Monthly Yields** (discounting progressive tax and IOF) directly into the Dividends view, displaying accrued interest as stacked payments.
- **Mathematical Precision & Backtesting:** A forward-looking profitability engine retroactively calculates future splits and reverse splits on historical quantities, aligning perfectly with "Split-Adjusted" data from Yahoo Finance to prevent false profit/loss spikes.
- **Multiple Watchlists:** Create customized watchlists for diverse investment strategies.
- **Integrated Telegram Bot:** Two-way Telegram interaction allowing users to fetch full financial reports, charts, and filter dynamic views natively within the chat. Supports multi-portfolio management by securely storing session context via Redis.
- **Valuation & Fundamentals (P/B, P/E, Yield):** Real-time calculation of intrinsic value based on Benjamin Graham and Décio Bazin models. Live fundamental indicators are retrieved via web scraping (Fundamentus for B3, Finviz for Global).
- **Unified Transaction Ledger:** A single-line layout consolidating Variable Income and Fixed Income operations with advanced filtering by module, ticker, and date. Features a native "Daily Real Impact" column to instantly measure the daily P&L contribution of each asset.
- **Asynchronous Price Alerts:** Background workers continuously monitor user-defined price targets, triggering instant email and Telegram notifications.
- **Robust Security:** JWT-based authentication stored exclusively in `HttpOnly` and `Secure` cookies, alongside strict CORS and CSRF configurations.
- **Full Observability:** Integrated telemetry using Prometheus, Grafana, and Loki for real-time metrics and log aggregation.

---

## 🛠️ Technology Stack

### Backend (Golang 1.24)
- **Routing & HTTP:** `go-chi`
- **Relational Database:** PostgreSQL 16 (`pgx/v5` connection pool)
- **Cache & Session State:** Redis 7 (`go-redis/v9`)
- **Authentication:** JWT (JSON Web Tokens) & Argon2id Hashing
- **Migrations:** `golang-migrate`
- **Market Data Providers:** Yahoo Finance API (Quotes/Search), Fundamentus & Finviz (Scraping)
- **Concurrency:** Goroutine-based background workers for alerts, dividends, and portfolio backfills.

### Frontend (Next.js 14)
- **Framework:** React 18 with TypeScript
- **Styling:** Vanilla CSS (Glassmorphism, Dark Mode)
- **Charting:** Lightweight Charts (TradingView)
- **Unit Testing:** Vitest & React Testing Library (100% Coverage)
- **E2E Testing:** Playwright

### Infrastructure & DevOps
- **Orchestration:** Docker Compose
- **Reverse Proxy:** Caddy (Local routing, gzip/zstd compression)
- **SMTP Testing:** Mailpit
- **Monitoring:** Prometheus, Grafana, Loki, and Promtail

---

## 📡 Data Providers Integration

stock-pulse operates dynamically, fetching updated market data (Quotes and Fundamentals) via external API integrations and Web Scraping:

### 1. Yahoo Finance API (Quotes and Search)
Provides real-time quotes and asset autocomplete functionality.
- **Search:** `GET https://query1.finance.yahoo.com/v1/finance/search?q={query}`
- **Quote/Chart:** `GET https://query1.finance.yahoo.com/v8/finance/chart/{symbol}?interval=1d&range=1d`

### 2. Fundamentus (B3 Fundamentals & Dividends)
Used to extract structured fundamentals for Brazilian assets (`.SA` suffix).
- **Fundamentals:** Regex extraction of P/B, EPS, and Yield (`/detalhes.php`).
- **Dividends History:** Extraction and sanitization via a **Heuristic Deduplication Engine** to clean raw source data.

### 3. Finviz (Global Fundamentals)
Routes global assets to Finviz for comprehensive international market indicators.
- **Endpoint:** `GET https://finviz.com/quote.ashx?t={symbol}`

### 4. StockAnalysis (Fallback for Global & B3 ETFs)
Primary provider for global dividends and critical fallback for Brazilian ETFs missing from Fundamentus.
- Prioritizes the **Record Date** over the Ex-Dividend Date for Brazilian assets to comply with national financial legislation.

---

## 💱 Multi-Currency Architecture & Exchange Rates

To support native multi-currency portfolios (e.g., a USD portfolio purchasing BRL assets), the system relies on an Exchange Rate multiplier:

`Total Transaction Cost = Quantity × Unit Price × Exchange Rate`

- **Domestic Assets (Same currency as portfolio):** Exchange rate is enforced as `1.0`.
- **International Assets (Different currency):** The system automatically fetches historical exchange rates at the exact transaction date, permanently anchoring the asset's total cost to the portfolio's base currency and shielding the average price from future FX volatility.

---

## 📦 Bulk CSV Import Specification

stock-pulse supports massive historical data ingestion via `.csv` or `.txt`.
Required column format (header row is ignored):

`DATE, TICKER, TYPE, QUANTITY, PRICE`

- **DATE**: `YYYY-MM-DD` or `DD/MM/YYYY`.
- **TYPE**:
  - `BUY` / `SELL`: Standard transactions.
  - `BONUS`: Stock bonuses. Quantity represents shares received; Price defines assigned cost.
  - `SPLIT` / `REVERSE_SPLIT`: Corporate actions. Quantity defines the multiplier/divisor.

---

## 📊 Architecture & Data Workflows

For an in-depth understanding of the internal microservices communication and architectural patterns, refer to the detailed documentation:

- 👉 [🔐 Authentication & Security (Auth, JWT, HttpOnly Cookies)](docs/architecture/auth.md)
- 👉 [🚑 Portfolio Systemic Auto-Healing (BackfillGap & LOCF)](docs/architecture/portfolio_healing.md)
- 👉 [📦 Transactional Bulk Import Workflow](docs/architecture/bulk_import.md)
- 👉 [🤖 Telegram Bot Bidirectional Integration & State Management](docs/architecture/telegram_bot.md)

### 1. High-Level Block Diagram
Illustrates Docker Compose orchestration and external routing.

```mermaid
graph TD
    Client[User Browser] -->|HTTPS| Caddy[Caddy Reverse Proxy]
    Caddy -->|/api/*| GoAPI[Go Backend API]
    Caddy -->|/| NextJS[Next.js Frontend]
    
    subgraph Data Layer
        GoAPI -->|Read/Write| PG[(PostgreSQL)]
        GoAPI -->|Session/Cache| Redis[(Redis)]
    end
    
    subgraph External Integrations
        GoAPI -->|Alerts| Mailpit[Local SMTP]
        GoAPI -->|Quotes| Yahoo[Yahoo Finance]
        GoAPI -->|B3 Data| Fundamentus[Fundamentus]
        GoAPI -->|Global Data| Finviz[Finviz]
    end
```

### 2. Real-Time Quotes Flow (WebSockets)

```mermaid
sequenceDiagram
    participant User as Frontend (React)
    participant API as Go Hub (Backend)
    participant Redis as Redis Cache
    participant Yahoo as Yahoo Finance

    User->>API: Connects via WebSocket (/ws)
    User->>API: Sends "subscribe" payload for PETR4
    API->>Redis: Check PETR4 cache TTL
    alt Cache Miss
        API->>Yahoo: GET /v8/finance/chart/PETR4.SA
        Yahoo-->>API: JSON Response
        API->>Redis: Set new price (3m TTL)
    end
    API-->>User: BroadCast (PETR4 Quote)
    Note over User: Dashboard chart updates instantly
```

### 3. Asynchronous Alert Workers

```mermaid
flowchart TD
    Start[Worker Init] --> FetchAlerts[Fetch Active Alerts from DB]
    FetchAlerts --> Loop[Iterate over Alerts]
    Loop --> CheckCache{Price in Cache?}
    
    CheckCache -- Yes --> Compare[Compare Current Price vs Target]
    CheckCache -- No --> FetchAPI[Fetch API & Cache] --> Compare
    
    Compare -- Target Hit --> Dispara[Dispatch SMTP Email]
    Compare -- No Hit --> Skip[Continue]
    
    Dispara --> MarcaInativo[Update Alert Status to 'Fired']
    MarcaInativo --> Loop
    Skip --> Loop
    
    Loop --> Sleep[Sleep 60s] --> FetchAlerts
```

### 4. Fundamentals Scraping & Valuation Engine

```mermaid
sequenceDiagram
    participant User as Frontend (React)
    participant API as Go API (Backend)
    participant Redis as Redis Cache
    participant Fund as Fundamentus (B3)
    participant Finviz as Finviz (Global)

    User->>API: GET /api/portfolio/fundamentals?symbol=MXRF11.SA
    API->>Redis: Check 'fundamentals:v2:MXRF11.SA'
    alt Cache Hit (24h TTL)
        Redis-->>API: Return cached fundamentals
    else Cache Miss
        alt Brazilian Asset (.SA)
            API->>Fund: Scrape HTML (detalhes.php)
            Fund-->>API: Return HTML
            Note over API: Regex Execution (P/B, EPS, Yield)
        else Global Asset
            API->>Finviz: Scrape HTML (quote.ashx)
            Finviz-->>API: Return HTML
            Note over API: Regex Execution (EPS, Book Value)
        end
        API->>API: Calculate Graham & Bazin Formulas
        API->>Redis: Store payload (24h TTL)
    end
    API-->>User: JSON Response
```

### 5. Dividend Processing Workflow

```mermaid
sequenceDiagram
    participant User as Frontend (React)
    participant API as Go API (Backend)
    participant Worker as Dividend Worker
    participant DB as PostgreSQL (asset_event)
    participant Scrapers as Fundamentus / StockAnalysis
    participant Yahoo as Yahoo Finance

    Note over Worker,Scrapers: Async Cron (Every 24h)
    Worker->>DB: Fetch all registered assets
    Worker->>Scrapers: Smart Scraping
    alt Scraping Success
        Scrapers-->>Worker: Deduplicated History
    else Failure
        Worker->>Yahoo: Trigger Fallback
        Yahoo-->>Worker: Basic History
    end
    Worker->>DB: Safe Upsert into 'asset_event'
    Note over Worker,DB: UNIQUE constraint prevents Ex-Date clones

    Note over User,API: Real-time Portfolio Load
    User->>API: GET /api/portfolios/{id}/dividends
    API->>DB: Fetch user transactions and events
    Note over API: Evaluate Custody on Record Date
    Note over API: Apply Taxes (US: 30%, JCP: 15%) & FX Rate
    API-->>User: Consolidated Yield Array
```

### 6. Fixed Income Monthly Yield Simulation

```mermaid
flowchart TD
    Start[GET /monthly-yields] --> FetchTX[Fetch Applications/Redemptions]
    FetchTX --> Sort[Chronological Sort]
    Sort --> LoopDays[Iterate Daily: T0 to Today]
    LoopDays --> IsWeekend{Weekend/Holiday?}
    
    IsWeekend -- Yes --> LoopDays
    IsWeekend -- No --> CalcPrincipal[Calculate Active Principal]
    
    CalcPrincipal --> ApplyRate[Apply Daily Interest Rate Factor]
    ApplyRate --> AccrueInterest[Accrue Gross Yield for Month]
    AccrueInterest --> EndOfMonth{Last Day of Month?}
    
    EndOfMonth -- No --> LoopDays
    EndOfMonth -- Yes --> CalcTaxes[Evaluate Retention for Progressive Tax]
    CalcTaxes --> Deduct[Deduct Taxes from Gross Yield]
    Deduct --> GenYield[Generate Synthetic Accrued Interest Event]
    GenYield --> ResetMonth[Reset Accumulator]
    ResetMonth --> LoopDays
    
    LoopDays -- "Reached Today" --> Return[Return Yield Array]
```

---

## 📂 Monorepo Architecture

```text
.
├── backend/          # Golang Backend (Domain-Driven Design)
│   ├── cmd/api/      # Entry point
│   ├── internal/     # Core logic (auth, market, portfolio, alerts, etc.)
│   ├── migrations/   # SQL Schema definitions
│   └── Dockerfile    # Go image with Air (Live Reload)
│
├── frontend/         # Next.js Web Interface
│   ├── src/app/      # Application Routing
│   ├── tests/        # Playwright E2E Tests
│   └── Dockerfile    # Node.js build image
│
├── docker-compose.yml # 9-container orchestrated environment
├── Makefile          # Automation shortcuts
└── Caddyfile         # Reverse proxy configurations
```

---

## 🤖 Telegram Bot Configuration

1. Locate **@BotFather** on Telegram.
2. Send `/newbot` and follow prompts to retrieve an **HTTP API Token**.
3. Create a `.env` file based on `.env.example`.
4. Inject the token: `TELEGRAM_BOT_TOKEN=your_token_here`.
5. Upon application startup, the Telegram module engages automatically. Sending `/start` to the bot will initialize the secure binding process.

---

## ⚙️ Local Development Setup

### Prerequisites
- Docker and Docker Compose.
- Make (Recommended).

### Deployment

```bash
# Build and orchestrate all containers (DB, Redis, Go, Next, Grafana, Mailpit)
make build

# Start environment
make up

# Tail logs
make logs

# Execute E2E automated test suite
make e2e

# Tear down environment
make down
```

### Local Endpoints
- **Frontend:** [http://stock-pulse.localhost](http://stock-pulse.localhost) or `localhost:3000`
- **Backend API:** [http://api.stock-pulse.localhost](http://api.stock-pulse.localhost) or `localhost:8080`
- **Mailpit:** [http://localhost:8025](http://localhost:8025)
- **Grafana:** [http://localhost:3001](http://localhost:3001) (admin / admin)

---

## 🏗️ Database Migrations

Generate new schema migrations via Makefile:
```bash
make migrate-create
```
*(Prompts for migration name and generates `.up.sql` / `.down.sql` in `backend/migrations`).*

---

## 🧪 Testing & Coverage

### Backend (Golang)
Rigorous unit testing covering success and database failure states via `pgxmock`.
```bash
cd backend
go test -v -coverprofile=coverage.out ./...
# Or via Docker: make test-backend
```

### Frontend (Next.js)
UI component testing and flow validation using `Vitest` and `React Testing Library`.
```bash
cd frontend
npm run test:coverage
# Or via Docker: make test-frontend
```

### End-to-End (E2E)
Robust validation using **Playwright**.
- **Dynamic Database Isolation:** `scripts/run-e2e.sh` automatically provisions an ephemeral `stockpulse_test` PostgreSQL schema. It restarts the backend with `MOCK_EXTERNAL_APIS=true` to ensure market data immutability during test cycles, tearing down the environment upon completion.
```bash
make e2e
```

---

## ☁️ Free-Tier Cloud Deployment Architecture

Designed for zero-cost deployment across globally distributed free-tier platforms:
- **Frontend:** [Vercel](https://vercel.com/) (Serverless Next.js, Global CDN)
- **Backend:** [Koyeb](https://koyeb.com/) (Free Eco Docker container) or GCP `e2-micro`.
- **Database:** [Supabase](https://supabase.com/) (500MB Dedicated PostgreSQL)
- **Cache & WebSockets:** [Redis Cloud](https://redis.com/try-free/) (30MB Managed Cluster)

---

## ⚖️ License

Licensed under the **AGPLv3 (GNU Affero General Public License v3.0)**.
Source code must remain open. Any modifications or derivative works hosted as web services must strictly open-source their underlying code under identical terms.

See the [`LICENSE`](LICENSE) file for comprehensive details.

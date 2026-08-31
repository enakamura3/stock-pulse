# 📱 Backlog de Melhorias — Bot Telegram (Stock Pulse)

> **Para agentes de IA:** Leia este documento integralmente antes de começar qualquer tarefa.
> Siga estritamente as regras do `AGENTS.md` (branch separada, PR via `gh pr create`, nunca push direto na `master`).

---

## Contexto Arquitetural

### Localização do código
```
backend/
  cmd/api/main.go                    ← wiring/injeção de dependências
  internal/telegram/
    bot.go                           ← BotRunner, SendAlertMessage, rate limiter
    handlers_base.go                 ← Handlers struct, Register(), getUserID()
    handlers_menu.go                 ← /start, /menu, resolveActivePortfolio()
    handlers_portfolio.go            ← btn_resumo, btn_change_portfolio
    handlers_dividends.go            ← btn_proventos, btn_divs_year, btn_divs_month
    handlers_history.go              ← btn_history (paginado)
    handlers_fixedincome.go          ← btn_renda_fixa
    handlers_operations.go           ← btn_operacao (state machine multi-step)
    service.go                       ← Service interface + Redis keys
    repository.go                    ← DB: user_telegram_link
    middleware.go                    ← AuthMiddleware
  internal/alert/
    service.go                       ← AlertService: CreateAlert, GetAlerts, DeleteAlert, ToggleAlert
  internal/market/
    service.go                       ← MarketService: GetQuote, GetBenchmarks
  internal/portfolio/
    service.go                       ← PortfolioService: GetPortfolios, GetPortfolioDetails, ...
```

### Interfaces já disponíveis no `Handlers`
```go
type Handlers struct {
    svc          Service           // Redis: estado de conversa, active portfolio, link/unlink
    portfolioSvc PortfolioService  // DB: carteiras, posições, transações, proventos
    marketSvc    MarketService     // Cache+Yahoo: cotações, busca, benchmarks
    fiSvc        FixedIncomeService // DB+BCB: renda fixa, tesouro
    // ← alertSvc AlertService NÃO existe ainda, precisa ser adicionado (Tarefa T3)
}
```

### Redis keys já usadas
| Chave | TTL | Propósito |
|---|---|---|
| `telegram_link:<token>` | 10 min | Token de vinculação |
| `telegram_state:<chatID>` | 1 hora | Estado da máquina de estados |
| `telegram_active_portfolio:<chatID>` | 365 dias | Carteira ativa |

### `ConversationState` atual
```go
type ConversationState struct {
    Step        string  // "EXPECT_TICKER" | "EXPECT_TYPE" | "EXPECT_QTY" | "EXPECT_PRICE"
    Ticker      string
    Type        string  // "BUY" | "SELL"
    Quantity    float64
    PortfolioID string
}
```

### Wiring em `main.go` (linhas relevantes)
```go
alertService := alert.NewService(alertRepo, marketProvider)   // já existe
telegramHandlers := telegram.NewHandlers(telegramService, portfolioService, marketService, fiService)
// alertService NÃO é passado para telegram.NewHandlers() — precisa ser adicionado em T3
```

### Regras de negócio obrigatórias (AGENTS.md)
1. **Nunca comparar `float64` com `==`** — usar tolerância `< 1e-6`
2. **`resolveActivePortfolio()`** deve ser chamado em todo handler que acessa carteira — nunca `portfolios[0]` diretamente
3. **Git workflow**: branch `feat/`, `fix/` ou `chore/` → commit → push → `gh pr create` → aguardar merge

---

## Ordenação de Execução

Execute as tarefas na ordem abaixo. Cada tarefa deve estar em sua própria branch e PR.

```
T1 → T2 → T3 → T4 → T5 → T6
```

**Dependências:**
- T1, T2, T5, T6 — independentes entre si, podem ir em uma PR juntas (`feat/telegram-ux-improvements`)
- T3 — independente, deve ir em PR própria (`feat/telegram-alerts-management`)
- T4 — depende de T3 (AlertService injetado no Handlers); aguardar merge de T3 antes de abrir T4

---

## T1 — Botão "🔄 Atualizar" no Resumo e nos Proventos

**Branch sugerida:** `feat/telegram-ux-improvements`
**Arquivos a modificar:** `handlers_portfolio.go`, `handlers_dividends.go`, `handlers_fixedincome.go`
**Esforço estimado:** Baixo (~30 min)

### Descrição
Adicionar um botão inline `🔄 Atualizar` que re-executa o mesmo handler, dando ao usuário a possibilidade de ver dados atualizados sem precisar voltar ao menu.

### Implementação

**Em `handlers_portfolio.go` — `HandlePortfolioSummary`:**

Trocar o menu final:
```go
// ANTES:
menu := &telebot.ReplyMarkup{}
btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
menu.Inline(menu.Row(btnBack))

// DEPOIS:
menu := &telebot.ReplyMarkup{}
btnRefresh := menu.Data("🔄 Atualizar", "btn_resumo")
btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
menu.Inline(menu.Row(btnRefresh), menu.Row(btnBack))
```

**Em `handlers_dividends.go` — `HandleDividends`:**

Trocar o menu final quando há dividendos:
```go
// ANTES:
menu.Inline(menu.Row(btnAno, btnMes), menu.Row(btnBack))

// DEPOIS:
btnRefresh := menu.Data("🔄 Atualizar", "btn_proventos")
menu.Inline(menu.Row(btnRefresh), menu.Row(btnAno, btnMes), menu.Row(btnBack))
```

**Em `handlers_fixedincome.go` — `HandleFixedIncome`:**
```go
// ANTES:
btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
menu.Inline(menu.Row(btnBack))

// DEPOIS:
btnRefresh := menu.Data("🔄 Atualizar", "btn_renda_fixa")
btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
menu.Inline(menu.Row(btnRefresh), menu.Row(btnBack))
```

### Atenção
Se os dados não mudaram desde o último envio, o Telegram retorna erro "message is not modified" ao usar `c.Edit()`. Tratar silenciosamente:
```go
err := c.Edit(msg, telebot.ModeMarkdown, menu)
if err != nil && strings.Contains(err.Error(), "message is not modified") {
    return nil
}
return err
```

---

## T2 — Paginar a Lista de Ativos no Resumo

**Branch sugerida:** `feat/telegram-ux-improvements` (mesma branch de T1)
**Arquivos a modificar:** `handlers_portfolio.go`, `handlers_base.go`
**Esforço estimado:** Médio (~2h)

### Descrição
O `HandlePortfolioSummary` imprime **todos** os ativos numa única mensagem. Em carteiras com 15+ ativos isso pode ultrapassar o limite de 4096 caracteres do Telegram.

### Solução
Remover o bloco de posições individuais do resumo principal e criar um novo handler `HandleAssetList` paginado, acessível por um botão `📋 Todos os Ativos`.

### Passo 1 — Remover bloco de posições do `HandlePortfolioSummary`

Localizar e remover as linhas do bloco abaixo em `handlers_portfolio.go`:
```go
// REMOVER este bloco completo (começa em ~linha 212):
if len(positions) > 0 {
    msg += p.Sprintf("\n📋 *Resumo Completo (Ativos)*\n")
    posByValue := make([]portfolio.Position, len(positions))
    copy(posByValue, positions)
    sort.Slice(posByValue, func(i, j int) bool {
        return posByValue[i].CurrentValue > posByValue[j].CurrentValue
    })
    for _, pos := range posByValue {
        // ... todo o loop de formatação
    }
}
```

### Passo 2 — Adicionar botão no menu do resumo

```go
// Substituir o menu ao final de HandlePortfolioSummary:
menu := &telebot.ReplyMarkup{}
btnRefresh := menu.Data("🔄 Atualizar", "btn_resumo")
btnAtivos := menu.Data("📋 Todos os Ativos", "btn_ativos")
btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
menu.Inline(menu.Row(btnRefresh), menu.Row(btnAtivos), menu.Row(btnBack))
```

### Passo 3 — Criar `HandleAssetList` em `handlers_portfolio.go`

```go
func (h *Handlers) HandleAssetList(c telebot.Context) error {
    defer c.Respond()

    pageStr := c.Data()
    page := 0
    if pageStr != "" {
        fmt.Sscanf(pageStr, "%d", &page)
    }

    userIDStr, err := h.getUserID(c)
    if err != nil {
        return err
    }

    portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
    if err != nil || len(portfolios) == 0 {
        return c.Edit("⚠️ Nenhuma carteira encontrada.")
    }
    portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)
    _, positions, err := h.portfolioSvc.GetPortfolioDetails(context.Background(), portfolioID, userIDStr)
    if err != nil {
        return c.Edit("❌ Ocorreu um erro ao buscar seus ativos.")
    }

    sort.Slice(positions, func(i, j int) bool {
        return positions[i].CurrentValue > positions[j].CurrentValue
    })

    pageSize := 10
    start := page * pageSize
    if start >= len(positions) {
        start = len(positions)
    }
    end := start + pageSize
    if end > len(positions) {
        end = len(positions)
    }

    totalPages := (len(positions) + pageSize - 1) / pageSize

    p := message.NewPrinter(language.BrazilianPortuguese)
    msg := p.Sprintf("📋 *Ativos: %s*\n_Página %d de %d_\n\n", portfolioName, page+1, totalPages)

    for _, pos := range positions[start:end] {
        symbol := "⚪"
        if pos.DailyChangePercent > 0 {
            symbol = "🟢"
        } else if pos.DailyChangePercent < 0 {
            symbol = "🔴"
        }

        totalReturn := 0.0
        if pos.TotalCost > 1e-6 {
            totalReturn = ((pos.CurrentValue - pos.TotalCost) / pos.TotalCost) * 100
        }

        msg += p.Sprintf("%s `%s`: *R$ %.2f* | Dia: %+.2f%% | L/P: %+.2f%%\n",
            symbol, pos.Ticker, pos.CurrentValue, pos.DailyChangePercent, totalReturn)
    }

    menu := &telebot.ReplyMarkup{}
    var navBtns []telebot.Btn
    if start > 0 {
        navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_ativos", fmt.Sprintf("%d", page-1)))
    }
    if end < len(positions) {
        navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_ativos", fmt.Sprintf("%d", page+1)))
    }

    var rows []telebot.Row
    if len(navBtns) > 0 {
        rows = append(rows, menu.Row(navBtns...))
    }
    btnBack := menu.Data("⬅️ Voltar ao Resumo", "btn_resumo")
    btnMenu := menu.Data("🏠 Menu", "btn_menu")
    rows = append(rows, menu.Row(btnBack, btnMenu))
    menu.Inline(rows...)

    return c.Edit(msg, telebot.ModeMarkdown, menu)
}
```

### Passo 4 — Registrar em `handlers_base.go`

```go
// Adicionar dentro de Register():
bot.Handle("\fbtn_ativos", h.HandleAssetList)
```

---

## T3 — Gestão de Alertas de Preço via Bot

**Branch sugerida:** `feat/telegram-alerts-management`
**Arquivos a modificar:** `handlers_base.go`, `handlers_menu.go`, `handlers_operations.go`, `main.go`; criar `handlers_alerts.go`
**Esforço estimado:** Alto (~4h)

### Passo 1 — Injetar `AlertService` no `Handlers`

**Em `handlers_base.go`:**

Adicionar a interface (importar o pacote `alert` se necessário, ou definir a interface localmente para evitar import circular):
```go
// Adicionar antes de "type Handlers struct":
type AlertService interface {
    CreateAlert(ctx context.Context, userID, ticker string, targetPrice float64, condition string) (*alert.Alert, error)
    GetAlerts(ctx context.Context, userID string) ([]*alert.Alert, error)
    DeleteAlert(ctx context.Context, id string, userID string) error
    ToggleAlert(ctx context.Context, id string, userID string) (string, error)
}
```

**Atenção:** Se importar o pacote `alert` criar import circular, definir a interface com os campos da struct retornada inline (ou usar um DTO local). Verificar antes de implementar.

Adicionar campo e atualizar construtor:
```go
type Handlers struct {
    svc          Service
    portfolioSvc PortfolioService
    marketSvc    MarketService
    fiSvc        FixedIncomeService
    alertSvc     AlertService  // NOVO
}

func NewHandlers(svc Service, pSvc PortfolioService, mSvc MarketService, fiSvc FixedIncomeService, alertSvc AlertService) *Handlers {
    return &Handlers{
        svc:          svc,
        portfolioSvc: pSvc,
        marketSvc:    mSvc,
        fiSvc:        fiSvc,
        alertSvc:     alertSvc,
    }
}
```

**Em `main.go`** — atualizar a chamada (linha ~157):
```go
// ANTES:
telegramHandlers := telegram.NewHandlers(telegramService, portfolioService, marketService, fiService)

// DEPOIS:
telegramHandlers := telegram.NewHandlers(telegramService, portfolioService, marketService, fiService, alertService)
```

### Passo 2 — Adicionar botão no menu

**Em `handlers_menu.go` — `sendOrEditMenu`:**
```go
btnAlertas := menu.Data("🔔 Meus Alertas", "btn_alerts")

rows := []telebot.Row{
    menu.Row(btnResumo),
    menu.Row(btnProventos),
    menu.Row(btnHistory),
    menu.Row(btnRendaFixa),
    menu.Row(btnOperacao),
    menu.Row(btnAlertas),  // NOVO
}
```

(O botão `🔄 Trocar Carteira` continua sendo adicionado condicionalmente depois.)

### Passo 3 — Criar `handlers_alerts.go`

Criar o arquivo `backend/internal/telegram/handlers_alerts.go` com os handlers:

**`HandleAlerts`** — lista todos os alertas do usuário:
```go
func (h *Handlers) HandleAlerts(c telebot.Context) error {
    defer c.Respond()
    if h.alertSvc == nil {
        return c.Edit("⚠️ Módulo de alertas não está ativo.")
    }

    userIDStr, err := h.getUserID(c)
    if err != nil {
        return err
    }

    alerts, err := h.alertSvc.GetAlerts(context.Background(), userIDStr)
    if err != nil {
        slog.Error("Failed to fetch alerts for telegram bot", "error", err)
        return c.Edit("❌ Ocorreu um erro ao buscar seus alertas.")
    }

    p := message.NewPrinter(language.BrazilianPortuguese)
    msg := "🔔 *Meus Alertas de Preço*\n\n"

    if len(alerts) == 0 {
        msg += "Você não possui alertas cadastrados.\n\nUse o botão abaixo para criar o seu primeiro alerta!"
    } else {
        for _, a := range alerts {
            condStr := "acima de"
            if a.Condition == "BELOW" {
                condStr = "abaixo de"
            }
            curr := a.Currency
            if curr == "" {
                curr = "BRL"
            }

            var emoji, statusLabel string
            switch a.Status {
            case "ACTIVE":
                emoji = "🟢"
            case "TRIGGERED":
                emoji = "🔔"
                statusLabel = " _(disparado)_"
            case "DISABLED":
                emoji = "⚪"
                statusLabel = " _(pausado)_"
            default:
                emoji = "⚪"
            }

            msg += p.Sprintf("%s `%s` — %s %s %.2f%s\n",
                emoji, a.Ticker, condStr, getCurrencySymbol(curr), a.TargetPrice, statusLabel)
        }
        msg += p.Sprintf("\n_Total: %d alerta(s)_", len(alerts))
    }

    menu := &telebot.ReplyMarkup{}
    var rows []telebot.Row

    // Botões de ação por alerta (máximo 5 para não sobrecarregar o inline keyboard)
    limit := 5
    if len(alerts) < limit {
        limit = len(alerts)
    }
    for i := 0; i < limit; i++ {
        a := alerts[i]
        toggleLabel := "⏸️ Pausar " + a.Ticker
        if a.Status == "DISABLED" || a.Status == "TRIGGERED" {
            toggleLabel = "▶️ Ativar " + a.Ticker
        }
        btnToggle := menu.Data(toggleLabel, "btn_alert_toggle_"+a.ID)
        btnDel := menu.Data("🗑️ "+a.Ticker, "btn_alert_del_"+a.ID)
        rows = append(rows, menu.Row(btnToggle, btnDel))
    }

    btnCreate := menu.Data("➕ Criar Alerta", "btn_alert_create")
    btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
    rows = append(rows, menu.Row(btnCreate), menu.Row(btnBack))
    menu.Inline(rows...)

    return c.Edit(msg, telebot.ModeMarkdown, menu)
}
```

**`HandleAlertCreate`** — inicia o fluxo de criação:
```go
func (h *Handlers) HandleAlertCreate(c telebot.Context) error {
    defer c.Respond()
    if h.alertSvc == nil {
        return c.Edit("⚠️ Módulo de alertas não está ativo.")
    }

    err := h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
        Step: "ALERT_EXPECT_TICKER",
    })
    if err != nil {
        return c.Edit("❌ Erro interno ao iniciar criação de alerta.")
    }

    menu := &telebot.ReplyMarkup{}
    btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
    menu.Inline(menu.Row(btnCancel))

    return c.Edit("🔔 *Criar Alerta de Preço*\n\nDigite o código do ativo:\n_(ex: VALE3.SA, AAPL, BTC-USD)_", telebot.ModeMarkdown, menu)
}
```

**`handleAlertToggle`**, **`handleAlertDelete`**, **`handleAlertCondition`**:
```go
func (h *Handlers) handleAlertToggle(c telebot.Context, alertID string) error {
    defer c.Respond()
    userIDStr, err := h.getUserID(c)
    if err != nil {
        return err
    }
    _, err = h.alertSvc.ToggleAlert(context.Background(), alertID, userIDStr)
    if err != nil {
        slog.Error("Erro ao fazer toggle de alerta via Telegram", "error", err, "alert_id", alertID)
        return c.Edit("❌ Erro ao alterar status do alerta.")
    }
    return h.HandleAlerts(c)
}

func (h *Handlers) handleAlertDelete(c telebot.Context, alertID string) error {
    defer c.Respond()
    userIDStr, err := h.getUserID(c)
    if err != nil {
        return err
    }
    err = h.alertSvc.DeleteAlert(context.Background(), alertID, userIDStr)
    if err != nil {
        slog.Error("Erro ao deletar alerta via Telegram", "error", err, "alert_id", alertID)
        return c.Edit("❌ Erro ao excluir alerta.")
    }
    return h.HandleAlerts(c)
}

func (h *Handlers) handleAlertCondition(c telebot.Context, condition string) error {
    defer c.Respond()
    state, err := h.svc.GetConversationState(context.Background(), c.Chat().ID)
    if err != nil || state == nil {
        return c.Edit("⚠️ Nenhuma criação de alerta em andamento. Use /menu.")
    }
    if condition != "ABOVE" && condition != "BELOW" {
        return c.Edit("❌ Condição inválida.")
    }

    state.Type = condition  // reutilizar campo Type para a condição do alerta
    state.Step = "ALERT_EXPECT_PRICE"
    _ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

    menu := &telebot.ReplyMarkup{}
    btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
    menu.Inline(menu.Row(btnCancel))

    condStr := "acima de"
    if condition == "BELOW" {
        condStr = "abaixo de"
    }
    msg := fmt.Sprintf("Ativo: `%s`\nCondição: *%s*\n\nAgora envie o preço alvo (ex: 38.50):", state.Ticker, condStr)
    return c.Edit(msg, telebot.ModeMarkdown, menu)
}
```

### Passo 4 — Adicionar cases ao `HandleText` em `handlers_operations.go`

Dentro do `switch state.Step`, adicionar **antes** do `default`:

```go
case "ALERT_EXPECT_TICKER":
    ticker := strings.ToUpper(strings.TrimSpace(text))
    _, err := h.marketSvc.GetQuote(context.Background(), ticker)
    if err != nil {
        return c.Send("⚠️ Ativo não encontrado. Verifique o código e tente novamente:", menu)
    }

    state.Ticker = ticker
    state.Step = "ALERT_EXPECT_CONDITION"
    _ = h.svc.SetConversationState(context.Background(), c.Chat().ID, *state)

    condMenu := &telebot.ReplyMarkup{}
    btnAbove := condMenu.Data("▲ Sobe acima de", "btn_alert_cond_ABOVE")
    btnBelow := condMenu.Data("▼ Cai abaixo de", "btn_alert_cond_BELOW")
    btnCancelAlert := condMenu.Data("❌ Cancelar", "btn_cancel_op")
    condMenu.Inline(condMenu.Row(btnAbove, btnBelow), condMenu.Row(btnCancelAlert))
    return c.Send(fmt.Sprintf("Ativo: `%s`\n\nQual a condição do alerta?", ticker), telebot.ModeMarkdown, condMenu)

case "ALERT_EXPECT_PRICE":
    text = strings.ReplaceAll(text, ",", ".")
    var price float64
    if _, sscanErr := fmt.Sscanf(text, "%f", &price); sscanErr != nil || price < 1e-6 {
        return c.Send("⚠️ Preço inválido. Envie apenas o número (ex: 38.50):", menu)
    }

    userIDStr, err := h.getUserID(c)
    if err != nil {
        return err
    }

    _, err = h.alertSvc.CreateAlert(context.Background(), userIDStr, state.Ticker, price, state.Type)
    if err != nil {
        slog.Error("Erro ao criar alerta via Telegram", "error", err)
        return c.Send("❌ Não foi possível criar o alerta. Tente novamente.", menu)
    }

    _ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)

    condStr := "acima de"
    if state.Type == "BELOW" {
        condStr = "abaixo de"
    }
    p := message.NewPrinter(language.BrazilianPortuguese)
    successMsg := p.Sprintf("✅ *Alerta criado!*\n\nAtivo: `%s`\nCondição: %s R$ %.2f\n\nVocê será notificado aqui quando o alerta for disparado.", state.Ticker, condStr, price)

    successMenu := &telebot.ReplyMarkup{}
    btnAlerts := successMenu.Data("🔔 Meus Alertas", "btn_alerts")
    btnMenuSuccess := successMenu.Data("🏠 Menu", "btn_menu")
    successMenu.Inline(successMenu.Row(btnAlerts, btnMenuSuccess))
    return c.Send(successMsg, telebot.ModeMarkdown, successMenu)
```

### Passo 5 — Registrar handlers e callbacks em `handlers_base.go`

```go
// Adicionar dentro de Register():
bot.Handle("\fbtn_alerts", h.HandleAlerts)
bot.Handle("\fbtn_alert_create", h.HandleAlertCreate)
```

**Em `HandleDynamicCallback` em `handlers_operations.go`**, adicionar antes do `return nil` final:
```go
if strings.HasPrefix(data, "btn_alert_toggle_") {
    alertID := strings.TrimPrefix(data, "btn_alert_toggle_")
    return h.handleAlertToggle(c, alertID)
}

if strings.HasPrefix(data, "btn_alert_del_") {
    alertID := strings.TrimPrefix(data, "btn_alert_del_")
    return h.handleAlertDelete(c, alertID)
}

if strings.HasPrefix(data, "btn_alert_cond_") {
    condition := strings.TrimPrefix(data, "btn_alert_cond_")
    return h.handleAlertCondition(c, condition)
}
```

---

## T4 — Cotação Rápida de Ativo

**Branch sugerida:** `feat/telegram-quick-quote`
**Dependência:** Aguardar merge de T3 (pois T3 altera `NewHandlers`)
**Arquivos a modificar:** `handlers_base.go`, `handlers_menu.go`, `handlers_operations.go`; criar `handlers_market.go`
**Esforço estimado:** Baixo (~1h)

### Passo 1 — Verificar a struct `market.Quote`

Antes de codificar, ler `backend/internal/market/service.go` e identificar os campos disponíveis na struct `Quote` (ou equivalente). Confirmar: `Price`, `Change`, `ChangePercent`, `Name`, `Currency`, e opcionalmente `Open`, `High`, `Low`, `Volume`.

### Passo 2 — Adicionar botão no menu

**Em `handlers_menu.go` — `sendOrEditMenu`** (após o botão de alertas adicionado em T3):
```go
btnCotacao := menu.Data("📈 Cotação Rápida", "btn_cotacao")
// adicionar em rows
```

### Passo 3 — Criar `handlers_market.go`

```go
package telegram

import (
    "context"
    "fmt"
    "strings"

    "golang.org/x/text/language"
    "golang.org/x/text/message"
    "gopkg.in/telebot.v3"
)

func (h *Handlers) HandleQuoteStart(c telebot.Context) error {
    defer c.Respond()

    err := h.svc.SetConversationState(context.Background(), c.Chat().ID, ConversationState{
        Step: "QUOTE_EXPECT_TICKER",
    })
    if err != nil {
        return c.Edit("❌ Erro interno.")
    }

    menu := &telebot.ReplyMarkup{}
    btnCancel := menu.Data("❌ Cancelar", "btn_cancel_op")
    menu.Inline(menu.Row(btnCancel))

    return c.Edit("📈 *Cotação Rápida*\n\nDigite o código do ativo:\n_(ex: VALE3.SA, AAPL, BTC-USD)_", telebot.ModeMarkdown, menu)
}
```

### Passo 4 — Adicionar case `QUOTE_EXPECT_TICKER` ao `HandleText`

```go
case "QUOTE_EXPECT_TICKER":
    ticker := strings.ToUpper(strings.TrimSpace(text))
    quote, err := h.marketSvc.GetQuote(context.Background(), ticker)
    if err != nil {
        return c.Send("⚠️ Ativo não encontrado. Verifique o código e tente novamente:", menu)
    }

    _ = h.svc.ClearConversationState(context.Background(), c.Chat().ID)

    p := message.NewPrinter(language.BrazilianPortuguese)

    changeEmoji := "⚪"
    changeSign := ""
    if quote.Change > 1e-6 {
        changeEmoji = "🟢"
        changeSign = "+"
    } else if quote.Change < -1e-6 {
        changeEmoji = "🔴"
    }

    curr := getCurrencySymbol(quote.Currency)
    // Ajuste: usar os campos reais da struct quote conforme verificado no Passo 1

    msg := p.Sprintf("📈 *%s*\n_%s_\n\n", strings.ToUpper(ticker), quote.Name)
    msg += p.Sprintf("💵 *Preço:* %s %.2f\n", curr, quote.Price)
    msg += p.Sprintf("%s *Variação:* %s%.2f (%s%.2f%%)\n",
        changeEmoji, changeSign, quote.Change, changeSign, quote.ChangePercent)

    // Adicionar Open, High, Low, Volume SOMENTE SE existirem na struct:
    // if quote.Open > 1e-6 { msg += ... }
    // if quote.Volume > 1e-6 { ... }

    replyMenu := &telebot.ReplyMarkup{}
    btnNew := replyMenu.Data("🔍 Consultar Outro", "btn_cotacao")
    btnMenuBtn := replyMenu.Data("🏠 Menu", "btn_menu")
    replyMenu.Inline(replyMenu.Row(btnNew, btnMenuBtn))

    return c.Send(msg, telebot.ModeMarkdown, replyMenu)
```

### Passo 5 — Registrar em `handlers_base.go`

```go
bot.Handle("\fbtn_cotacao", h.HandleQuoteStart)
```

---

## T5 — Benchmarks no Resumo da Carteira

**Branch sugerida:** `feat/telegram-ux-improvements` (mesma de T1 e T2)
**Arquivos a modificar:** `handlers_portfolio.go`
**Esforço estimado:** Baixo (~45 min)

### Passo 1 — Verificar assinatura de `GetBenchmarks`

Ler `backend/internal/market/service.go` e identificar:
- A assinatura: `GetBenchmarks(ctx context.Context) ([]SomeStruct, error)` (verificar o nome e struct real)
- Os campos da struct retornada (name, change_percent, etc.)

### Passo 2 — Adicionar bloco de benchmarks no `HandlePortfolioSummary`

Em `handlers_portfolio.go`, localizar onde as maiores altas/baixas terminam e adicionar **antes** da construção do menu:

```go
// Após o bloco de maiores baixas do dia:
if benchmarks, bErr := h.marketSvc.GetBenchmarks(context.Background()); bErr == nil && len(benchmarks) > 0 {
    msg += "\n📊 *Benchmarks do Dia*\n"
    for _, b := range benchmarks {
        // Adaptar campos conforme a struct real retornada por GetBenchmarks:
        bmEmoji := "⚪"
        bmSign := ""
        if b.ChangePercent > 1e-6 {
            bmEmoji = "🟢"
            bmSign = "+"
        } else if b.ChangePercent < -1e-6 {
            bmEmoji = "🔴"
        }
        msg += p.Sprintf("• %s *%s:* %s%.2f%%\n", bmEmoji, b.Name, bmSign, b.ChangePercent)
    }
}
```

**Regra:** Se `GetBenchmarks` retornar erro, omitir o bloco silenciosamente (nunca quebrar o resumo principal por causa de benchmarks).

---

## T6 — Histórico com Filtro por Tipo (Compra/Venda)

**Branch sugerida:** `feat/telegram-ux-improvements` (mesma de T1, T2, T5)
**Arquivos a modificar:** `handlers_history.go`
**Esforço estimado:** Médio (~1.5h)

### Descrição
Adicionar botões de filtro rápido: `📋 Todas` | `🟢 Compras` | `🔴 Vendas`. O filtro é codificado na callback data no formato `PAGE:FILTER`.

### Implementação completa de `HandleHistory`

Substituir o handler atual pelo seguinte (preservar toda a lógica de paginação, acrescentar o filtro):

```go
func (h *Handlers) HandleHistory(c telebot.Context) error {
    defer c.Respond()
    userIDStr, err := h.getUserID(c)
    if err != nil {
        return err
    }

    portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
    if err != nil || len(portfolios) == 0 {
        return c.Edit("⚠️ Nenhuma carteira encontrada.")
    }
    portfolioID, portfolioName := h.resolveActivePortfolio(context.Background(), c.Chat().ID, portfolios)

    txs, err := h.portfolioSvc.GetPortfolioTransactions(context.Background(), portfolioID, userIDStr)
    if err != nil {
        slog.Error("Failed to fetch transactions for telegram bot", "error", err, "user_id", userIDStr)
        return c.Edit("❌ Ocorreu um erro ao buscar o histórico.")
    }

    // Parse callback data: "PAGE:FILTER" (ex: "0:ALL", "1:BUY", "0:SELL")
    rawData := c.Data()
    page := 0
    filter := "ALL"
    if rawData != "" {
        parts := strings.SplitN(rawData, ":", 2)
        if len(parts) == 2 {
            fmt.Sscanf(parts[0], "%d", &page)
            filter = strings.ToUpper(parts[1])
        } else {
            // backward compat: apenas número sem filtro
            fmt.Sscanf(rawData, "%d", &page)
        }
    }

    // Aplicar filtro
    var filtered []portfolio.Transaction
    for _, tx := range txs {
        if filter == "ALL" || tx.Type == filter {
            filtered = append(filtered, tx)
        }
    }

    pageSize := 10
    start := page * pageSize
    if start >= len(filtered) {
        start = 0
        page = 0
    }
    end := start + pageSize
    if end > len(filtered) {
        end = len(filtered)
    }

    filterEmoji := "📋"
    filterLabel := "Todas"
    if filter == "BUY" {
        filterEmoji = "🟢"
        filterLabel = "Compras"
    } else if filter == "SELL" {
        filterEmoji = "🔴"
        filterLabel = "Vendas"
    }

    menu := &telebot.ReplyMarkup{}
    btnAll := menu.Data("📋 Todas", "btn_history", "0:ALL")
    btnBuy := menu.Data("🟢 Compras", "btn_history", "0:BUY")
    btnSell := menu.Data("🔴 Vendas", "btn_history", "0:SELL")

    if len(filtered) == 0 {
        msg := fmt.Sprintf("📜 *Histórico: %s*\n_%s %s — Nenhuma transação encontrada._", portfolioName, filterEmoji, filterLabel)
        btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
        menu.Inline(menu.Row(btnAll, btnBuy, btnSell), menu.Row(btnBack))
        return c.Edit(msg, telebot.ModeMarkdown, menu)
    }

    p := message.NewPrinter(language.BrazilianPortuguese)
    msg := p.Sprintf("📜 *Histórico: %s*\n_%s %s — Página %d_\n\n", portfolioName, filterEmoji, filterLabel, page+1)

    for _, tx := range filtered[start:end] {
        tipoStr := "🟢 C"
        if tx.Type == "SELL" {
            tipoStr = "🔴 V"
        }
        msg += p.Sprintf("%s | `%s`\n", tipoStr, tx.Ticker)
        msg += p.Sprintf("Data: %s\n", tx.ExecutedAt.Format("2006-01-02"))
        if tx.Fee > 1e-6 {
            msg += p.Sprintf("Qtd: %.4f | Preço: %.2f | Total: %.2f (Taxas: R$ %.2f)\n\n",
                tx.Quantity, tx.UnitPrice, tx.TotalCost, tx.Fee)
        } else {
            msg += p.Sprintf("Qtd: %.4f | Preço: %.2f | Total: %.2f\n\n",
                tx.Quantity, tx.UnitPrice, tx.TotalCost)
        }
    }

    var rows []telebot.Row
    rows = append(rows, menu.Row(btnAll, btnBuy, btnSell))

    var navBtns []telebot.Btn
    if start > 0 {
        navBtns = append(navBtns, menu.Data("⬅️ Anterior", "btn_history", fmt.Sprintf("%d:%s", page-1, filter)))
    }
    if end < len(filtered) {
        navBtns = append(navBtns, menu.Data("Próxima ➡️", "btn_history", fmt.Sprintf("%d:%s", page+1, filter)))
    }
    if len(navBtns) > 0 {
        rows = append(rows, menu.Row(navBtns...))
    }

    btnBack := menu.Data("⬅️ Voltar ao Menu", "btn_menu")
    rows = append(rows, menu.Row(btnBack))
    menu.Inline(rows...)

    return c.Edit(msg, telebot.ModeMarkdown, menu)
}
```

**Verificar:** O `menu.Data("label", "unique", extraParams...)` do `telebot.v3` aceita parâmetros extras que são passados como `c.Data()`. Confirmar na documentação da lib antes de usar. Se não suportar parâmetros extras, usar callback IDs compostos (ex: `"btn_hist_0_ALL"`) e tratar no `HandleDynamicCallback`.

---

## Checklist Final por PR

Para cada PR, verificar antes de abrir:

- [ ] `git fetch origin master && git rebase origin/master` realizado antes de começar
- [ ] Branch segue o padrão `feat/`, `fix/` ou `chore/`
- [ ] **Nenhuma** comparação `float64 == 0` ou `float64 != 0` sem tolerância `1e-6`
- [ ] Todo handler de carteira usa `h.resolveActivePortfolio()` — nunca `portfolios[0]`
- [ ] Todo handler de callback tem `defer c.Respond()` no início
- [ ] `c.Edit()` para respostas a callbacks inline, `c.Send()` para texto do usuário
- [ ] Erros de "message is not modified" tratados silenciosamente nos botões de Atualizar
- [ ] Nenhuma mensagem individual ultrapassa ~3800 caracteres (margem de segurança abaixo dos 4096 do Telegram)
- [ ] `go build ./...` e `go test ./...` passam sem erros
- [ ] PR criada com `gh pr create` incluindo descrição do que foi feito
- [ ] **Nunca** fazer push direto na `master`

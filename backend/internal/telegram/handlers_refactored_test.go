package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestHandlers_AuthMiddleware(t *testing.T) {
	h, svc, _, _, _, _ := setupHandlersTest()

	t.Run("ignore /start message", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Message").Return(&telebot.Message{})
		mCtx.On("Text").Return("/start token123")

		called := false
		next := func(c telebot.Context) error {
			called = true
			return nil
		}

		err := h.AuthMiddleware(next)(mCtx)
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("user not linked (message)", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Message").Return(&telebot.Message{})
		mCtx.On("Text").Return("/menu")
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("GetUserIDByChatID", mock.Anything, int64(123)).Return(uuid.Nil, errors.New("not found")).Once()
		mCtx.On("Send", "⚠️ Sua conta não está vinculada. Gere um link no painel do Stock Pulse.", mock.Anything).Return(nil)

		called := false
		next := func(c telebot.Context) error {
			called = true
			return nil
		}

		err := h.AuthMiddleware(next)(mCtx)
		assert.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("user not linked (callback)", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Message").Return((*telebot.Message)(nil))
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{})
		svc.On("GetUserIDByChatID", mock.Anything, int64(123)).Return(uuid.Nil, errors.New("not found")).Once()
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Send", "⚠️ Sua conta não está vinculada. Gere um link no painel do Stock Pulse.", mock.Anything).Return(nil)

		called := false
		next := func(c telebot.Context) error {
			called = true
			return nil
		}

		err := h.AuthMiddleware(next)(mCtx)
		assert.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("user linked successfully", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Message").Return(&telebot.Message{})
		mCtx.On("Text").Return("/menu")
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		uID := uuid.New()
		svc.On("GetUserIDByChatID", mock.Anything, int64(123)).Return(uID, nil).Once()
		mCtx.On("Set", "user_id", uID.String()).Return()

		called := false
		next := func(c telebot.Context) error {
			called = true
			return nil
		}

		err := h.AuthMiddleware(next)(mCtx)
		assert.NoError(t, err)
		assert.True(t, called)
	})
}

func TestHandlers_HandleMenuAndCallback(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("HandleMenu - no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Send", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil)

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleMenuCallback - no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil)

		err := h.HandleMenuCallback(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleMenu - 1 portfolio success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "My Portfolio"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleMenu(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleMenuCallback - 2 portfolios success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "My Portfolio 1"},
			{ID: "p2", Name: "My Portfolio 2"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("", errors.New("not set")).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleMenuCallback(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_PortfolioSummaryAndSelection(t *testing.T) {
	h, svc, pSvc, _, fiSvc, _ := setupHandlersTest()

	t.Run("HandlePortfolioSummary - no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandlePortfolioSummary - error details", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("err")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar sua carteira.", mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandlePortfolioSummary - success with FI", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		pDetails := &portfolio.Portfolio{ID: "p1", Name: "P1"}
		positions := []portfolio.Position{
			{Ticker: "AAPL", Quantity: 10, CurrentPrice: 150, CurrentValue: 1500, TotalCost: 1400, DailyChange: 5, DailyChangePercent: 3.33},
			{Ticker: "MSFT", Quantity: 5, CurrentPrice: 300, CurrentValue: 1500, TotalCost: 1600, DailyChange: -10, DailyChangePercent: -3.23},
		}
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(pDetails, positions, nil).Once()

		fiPositions := []fixedincome.Position{
			{GrossValue: 1050, NetValue: 1000, TotalInvested: 950, DaysToMaturity: 15, IsMatured: false, Asset: fixedincome.Asset{Institution: "Banco X", Type: "CDB", Rate: 12.5, DebtType: "PRE"}},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPositions, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()

		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleChangePortfolio - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
			{ID: "p2", Name: "P2"},
		}, nil).Once()
		mCtx.On("Edit", "Qual carteira você deseja definir como Ativa?", mock.Anything).Return(nil)

		err := h.HandleChangePortfolio(mCtx)
		assert.NoError(t, err)
	})

	t.Run("handleSelectedPortfolio - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
			{ID: "p2", Name: "P2"},
		}, nil).Once()
		svc.On("SetActivePortfolio", mock.Anything, int64(123), "p2").Return(nil).Once()

		// For returning back to menu
		mCtx.On("Callback").Return(&telebot.Callback{})
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
			{ID: "p2", Name: "P2"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p2", nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p2")
		assert.NoError(t, err)
	})
}

func TestHandlers_Dividends(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("HandleDividends - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()

		pastDate := now.Add(-1 * time.Hour)
		futureDate := now.Add(1 * time.Hour)

		divs := []portfolio.CalculatedDividend{
			{Ticker: "AAPL", NetAmount: 10.0, PaymentDate: pastDate, Type: "DIVIDENDO", Currency: "USD"},
			{Ticker: "MSFT", NetAmount: 15.0, PaymentDate: futureDate, Type: "JCP", Currency: "BRL"},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			hasTitle := strings.Contains(msg, "💸 *Proventos: P1*")
			hasUSD := strings.Contains(msg, "US$ 10,00")
			hasBRL := strings.Contains(msg, "R$ 15,00")
			hasAAPL := strings.Contains(msg, "`AAPL` • US$ 10,00 • "+pastDate.Format("2006-01-02"))
			hasAAPLSub := strings.Contains(msg, "   ↳ _DIV_")
			hasMSFT := strings.Contains(msg, "`MSFT` • R$ 15,00 • "+futureDate.Format("2006-01-02"))
			hasMSFTSub := strings.Contains(msg, "   ↳ _JCP_")
			return hasTitle && hasUSD && hasBRL && hasAAPL && hasAAPLSub && hasMSFT && hasMSFTSub
		}), mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividends - success with BR assets and position details", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		pastDate := now.Add(-1 * time.Hour)
		futureDate := now.Add(1 * time.Hour)

		divs := []portfolio.CalculatedDividend{
			{Ticker: "PETR4.SA", NetAmount: 50.0, PaymentDate: pastDate, Type: "JCP", Currency: "BRL", Quantity: 100, PerShareAmount: 0.5, AssetType: "STOCK_BR"},
			{Ticker: "MXRF11.SA", NetAmount: 12.0, PaymentDate: futureDate, Type: "RENDIMENTO", Currency: "BRL", Quantity: 120, PerShareAmount: 0.1, AssetType: "FII"},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			hasTitle := strings.Contains(msg, "💸 *Proventos: P1*")
			hasTotalAcumulado := strings.Contains(msg, "💰 *Total Acumulado:* R$ 50,00")
			hasBRLRecebidos := strings.Contains(msg, "✅ *Recebidos no Mês:* R$ 50,00")

			expectedFutureStr := "⏳ *A Receber no Mês:* R$ 0,00"
			if futureDate.Month() == now.Month() && futureDate.Year() == now.Year() {
				expectedFutureStr = "⏳ *A Receber no Mês:* R$ 12,00"
			}
			hasBRLAReceber := strings.Contains(msg, expectedFutureStr)

			hasPETR4 := strings.Contains(msg, "`PETR4` • R$ 50,00 • "+pastDate.Format("2006-01-02"))
			hasPETR4Sub := strings.Contains(msg, "   ↳ _JCP • 100 un x R$ 0,50_")
			hasMXRF11 := strings.Contains(msg, "`MXRF11` • R$ 12,00 • "+futureDate.Format("2006-01-02"))
			hasMXRF11Sub := strings.Contains(msg, "   ↳ _REND • 120 un x R$ 0,10_")
			return hasTitle && hasTotalAcumulado && hasBRLRecebidos && hasBRLAReceber && hasPETR4 && hasPETR4Sub && hasMXRF11 && hasMXRF11Sub
		}), mock.Anything).Return(nil)

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByYear - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		divs := []portfolio.CalculatedDividend{
			{Ticker: "AAPL", NetAmount: 10.0, PaymentDate: time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC), Type: "DIVIDENDO", Currency: "USD"},
			{Ticker: "MSFT", NetAmount: 15.0, PaymentDate: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), Type: "JCP", Currency: "BRL"},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			hasTitle := strings.Contains(msg, "📅 *Proventos por Ano: P1*")
			has2026 := strings.Contains(msg, "📅 *Ano 2026*\n• *Total:* R$ 15,00") && strings.Contains(msg, "• *Média Mensal:* R$")
			has2025 := strings.Contains(msg, "📅 *Ano 2025*\n• *Total:* US$ 10,00\n• *Média Mensal:* US$ 0,83/mês")
			hasAcumulado := strings.Contains(msg, "💰 *Acumulado Geral:* R$ 15,00 | US$ 10,00")
			hasTipos := strings.Contains(msg, "• *Tipos:* JCP: R$ 15,00") && strings.Contains(msg, "• *Tipos:* DIV: US$ 10,00")
			hasTop := strings.Contains(msg, "• *Top Ativos:* MSFT (R$ 15,00)") && strings.Contains(msg, "• *Top Ativos:* AAPL (US$ 10,00)")
			return hasTitle && has2026 && has2025 && hasAcumulado && hasTipos && hasTop
		}), mock.Anything).Return(nil)

		err := h.HandleDividendsByYear(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("0").Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		divs := []portfolio.CalculatedDividend{
			{Ticker: "AAPL", NetAmount: 10.0, PaymentDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), Type: "DIVIDENDO", Currency: "USD"},
			{Ticker: "MSFT", NetAmount: 15.0, PaymentDate: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), Type: "JCP", Currency: "BRL"},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			hasTitle := strings.Contains(msg, "📆 *Proventos por Mês: P1*")
			hasMonthTotal := strings.Contains(msg, "• *2026-05*: R$ 15,00 | US$ 10,00")
			hasAAPL := strings.Contains(msg, "↳ `AAPL` (DIV) • US$ 10,00 • Dia 15")
			hasMSFT := strings.Contains(msg, "↳ `MSFT` (JCP) • R$ 15,00 • Dia 10")
			correctOrder := strings.Index(msg, "MSFT") < strings.Index(msg, "AAPL")
			return hasTitle && hasMonthTotal && hasAAPL && hasMSFT && correctOrder
		}), mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HistoryAndFixedIncome(t *testing.T) {
	h, svc, pSvc, _, fiSvc, _ := setupHandlersTest()

	t.Run("HandleHistory - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("0").Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		txs := []portfolio.Transaction{
			{Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, TotalCost: 1500, ExecutedAt: time.Now()},
			{Ticker: "MSFT", Type: "SELL", Quantity: 5, UnitPrice: 300, TotalCost: 1500, ExecutedAt: time.Now()},
		}
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		fiPositions := []fixedincome.Position{
			{GrossValue: 1050, NetValue: 1000, TotalInvested: 950, DaysToMaturity: 15, IsMatured: false, Asset: fixedincome.Asset{Institution: "Banco X", Type: "CDB", Rate: 12.5, DebtType: "PRE"}},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPositions, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome - with treasury and matured positions", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		fiPositions := []fixedincome.Position{
			{GrossValue: 1050, NetValue: 1000, TotalInvested: 950, DaysToMaturity: 15, IsMatured: true, Asset: fixedincome.Asset{Institution: "Banco Y", Type: "LCI", Rate: 100, DebtType: "POS", Indexer: "CDI"}},
		}
		trPositions := []fixedincome.TreasuryPosition{
			{TreasuryType: "SELIC", GrossValue: 5200, NetValue: 5000, TotalInvested: 4500, Quantity: 0.5, DaysToMaturity: 10, IsMatured: true},
		}

		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPositions, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return(trPositions, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Tesouro SELIC")
		}), mock.Anything).Return(nil).Once()

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "Tesouro SELIC")
	})
}

func TestHandlers_Operations(t *testing.T) {
	h, svc, pSvc, mSvc, _, _ := setupHandlersTest()

	t.Run("HandleCancelOperation - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Times(2)

		// For sending menu inside sendOrEditMenu
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleCancelOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleLaunchOperation - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{{Ticker: "AAPL"}}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_TICKER", PortfolioID: "p1"}).Return(nil).Once()

		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDynamicCallback - dispatch btn_ticker_", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fbtn_ticker_AAPL"})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TICKER", PortfolioID: "p1"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_TYPE", PortfolioID: "p1", Ticker: "AAPL"}).Return(nil).Once()

		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleNewAsset - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TICKER", PortfolioID: "p1"}, nil).Once()
		mCtx.On("Edit", "Qual o código do ativo? (ex: AAPL, PETR4.SA)", mock.Anything).Return(nil)

		err := h.HandleNewAsset(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleSetType - BUY and SELL", func(t *testing.T) {
		// Test BUY
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TYPE", PortfolioID: "p1", Ticker: "AAPL"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_QTY", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY"}).Return(nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleSetTypeBuy(mCtx)
		assert.NoError(t, err)

		// Test SELL
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TYPE", PortfolioID: "p1", Ticker: "AAPL"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_QTY", PortfolioID: "p1", Ticker: "AAPL", Type: "SELL"}).Return(nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err = h.HandleSetTypeSell(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("handleSelectedQty - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10}).Return(nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.handleSelectedQty(mCtx, "10")
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_TICKER", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("AAPL")
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TICKER", PortfolioID: "p1"}, nil).Times(2)
		mSvc.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Symbol: "AAPL"}, nil).Once()

		// Expected inside handleSelectedTicker
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_TYPE", PortfolioID: "p1", Ticker: "AAPL"}).Return(nil).Once()
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_QTY", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("10,5")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10.5}).Return(nil).Once()
		mCtx.On("Send", "Qual o preço unitário da transação? (ex: 15.50)", mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_PRICE success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("150.50")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10}, nil).Once()
		pSvc.On("AddTransaction", mock.Anything, "00000000-0000-0000-0000-000000000000", mock.MatchedBy(func(tx *portfolio.Transaction) bool {
			return tx.Ticker == "AAPL" && tx.UnitPrice == 150.50 && tx.TotalCost == 1505.0 && tx.Type == "BUY"
		})).Return(&portfolio.Transaction{}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})
}

func TestGetCurrencySymbolAndAbbreviate(t *testing.T) {
	assert.Equal(t, "US$", getCurrencySymbol("USD"))
	assert.Equal(t, "€", getCurrencySymbol("EUR"))
	assert.Equal(t, "R$", getCurrencySymbol("BRL"))
	assert.Equal(t, "R$", getCurrencySymbol("OTHER"))

	assert.Equal(t, "DIV", abbreviateDividendType("DIVIDENDO"))
	assert.Equal(t, "DIV", abbreviateDividendType("DIVIDENDOS"))
	assert.Equal(t, "DIV", abbreviateDividendType("DIV"))
	assert.Equal(t, "JCP", abbreviateDividendType("JUROS SOBRE CAPITAL PRÓPRIO"))
	assert.Equal(t, "JCP", abbreviateDividendType("JUROS SOBRE CAPITAL PROPRIO"))
	assert.Equal(t, "JCP", abbreviateDividendType("JCP"))
	assert.Equal(t, "REND", abbreviateDividendType("RENDIMENTO"))
	assert.Equal(t, "REND", abbreviateDividendType("RENDIMENTOS"))
	assert.Equal(t, "REND", abbreviateDividendType("REND"))
	assert.Equal(t, "AMORT", abbreviateDividendType("AMORTIZAÇÃO"))
	assert.Equal(t, "AMORT", abbreviateDividendType("AMORTIZACAO"))
	assert.Equal(t, "ABCD", abbreviateDividendType("ABCD"))
	assert.Equal(t, "TEST", abbreviateDividendType("TESTING"))
}

func TestHandlers_ExtraErrors(t *testing.T) {
	h, svc, pSvc, _, fiSvc, _ := setupHandlersTest()

	t.Run("fetchDividends - no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()

		divs, pName, err := h.fetchDividends(mCtx)
		assert.Error(t, err)
		assert.Nil(t, divs)
		assert.Equal(t, "", pName)
	})

	t.Run("fetchDividends - portfolios error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, errors.New("db error")).Once()

		divs, pName, err := h.fetchDividends(mCtx)
		assert.Error(t, err)
		assert.Nil(t, divs)
		assert.Equal(t, "", pName)
	})

	t.Run("fetchDividends - dividends error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.CalculatedDividend{}, errors.New("db error")).Once()

		divs, pName, err := h.fetchDividends(mCtx)
		assert.Error(t, err)
		assert.Nil(t, divs)
		assert.Equal(t, "", pName)
	})

	t.Run("HandleDividendsByYear - error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "❌ Erro ao buscar proventos.", mock.Anything).Return(nil)

		err := h.HandleDividendsByYear(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth - error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "❌ Erro ao buscar proventos.", mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth - empty", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("0").Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.CalculatedDividend{}, nil).Once()
		mCtx.On("Edit", "📆 Nenhum provento encontrado.", mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory - no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada.", mock.Anything).Return(nil)

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory - error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{}, errors.New("db error")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar o histórico.", mock.Anything).Return(nil)

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory - empty", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("0").Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.Transaction{}, nil).Once()
		mCtx.On("Edit", "📜 Nenhuma operação encontrada na sua carteira.", mock.Anything).Return(nil)

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome - no portfolios", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada.", mock.Anything).Return(nil)

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome - error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, errors.New("db error")).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Maybe()
		mCtx.On("Edit", "❌ Erro ao buscar posições de Renda Fixa.", mock.Anything).Return(nil)

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome - empty", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()
		mCtx.On("Edit", "🏛️ Você ainda não possui ativos de Renda Fixa ou Tesouro Direto cadastrados.", mock.Anything).Return(nil)

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleLaunchOperation - error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("error")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar seus ativos.", mock.Anything).Return(nil)

		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)
	})

	t.Run("handleSelectedPortfolio - invalid portfolio", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		mCtx.On("Edit", "❌ Carteira inválida.", mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p_invalid")
		assert.NoError(t, err)
	})

	t.Run("handleSelectedPortfolio - set active error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("SetActivePortfolio", mock.Anything, int64(123), "p1").Return(errors.New("db error")).Once()
		mCtx.On("Edit", "❌ Erro interno ao salvar carteira ativa.", mock.Anything).Return(nil)

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})
}

func TestHandlers_TextErrors(t *testing.T) {
	h, svc, pSvc, mSvc, _, _ := setupHandlersTest()

	t.Run("HandleText - state nil", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()

		// For menu redirection
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mCtx.On("Send", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_TICKER quote error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("AAPL_INVALID")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TICKER", PortfolioID: "p1"}, nil).Once()
		mSvc.On("GetQuote", mock.Anything, "AAPL_INVALID").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mCtx.On("Send", "⚠️ Ativo não encontrado na bolsa. Verifique se há erros de digitação e envie o código novamente:", mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_QTY error format", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("invalid_number")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY"}, nil).Once()
		mCtx.On("Send", "⚠️ Quantidade inválida. Por favor, envie apenas o número (ex: 10):", mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_PRICE error format", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("invalid_price")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10}, nil).Once()
		mCtx.On("Send", "⚠️ Preço inválido. Por favor, envie apenas o número (ex: 15.50):", mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText - EXPECT_PRICE db error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("150.50")

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10}, nil).Once()
		pSvc.On("AddTransaction", mock.Anything, "00000000-0000-0000-0000-000000000000", mock.Anything).Return((*portfolio.Transaction)(nil), errors.New("db error")).Once()
		mCtx.On("Send", "❌ Ocorreu um erro ao salvar a transação. Tente novamente mais tarde.", mock.Anything).Return(nil)

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_DynamicCallbackExtra(t *testing.T) {
	h, svc, pSvc, _, _, _ := setupHandlersTest()

	t.Run("HandleDynamicCallback - dispatch btn_sel_port_", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fbtn_sel_port_p2"})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
			{ID: "p2", Name: "P2"},
		}, nil).Once()
		svc.On("SetActivePortfolio", mock.Anything, int64(123), "p2").Return(nil).Once()

		// For menu redirection
		mCtx.On("Callback").Return(&telebot.Callback{})
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
			{ID: "p2", Name: "P2"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p2", nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDynamicCallback - dispatch btn_qty_", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\fbtn_qty_10"})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10}).Return(nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})
}

func TestSortCurrencies(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{"USD", "BRL"},
			expected: []string{"BRL", "USD"},
		},
		{
			input:    []string{"EUR", "USD", "BRL"},
			expected: []string{"BRL", "USD", "EUR"},
		},
		{
			input:    []string{"USD", "EUR"},
			expected: []string{"USD", "EUR"},
		},
		{
			input:    []string{"EUR", "GBP"},
			expected: []string{"EUR", "GBP"},
		},
		{
			input:    []string{"GBP", "EUR"},
			expected: []string{"EUR", "GBP"},
		},
		{
			input:    []string{"BRL", "BRL"},
			expected: []string{"BRL", "BRL"},
		},
	}

	for _, tc := range tests {
		inputCopy := make([]string, len(tc.input))
		copy(inputCopy, tc.input)
		sortCurrencies(inputCopy)
		assert.Equal(t, tc.expected, inputCopy)
	}
}

func TestGetMacroCategoryKey(t *testing.T) {
	assert.Equal(t, "FII", getMacroCategoryKey("FII", "HGLG11"))
	assert.Equal(t, "FII", getMacroCategoryKey("OTHER", "KNRI11.SA"))
	assert.Equal(t, "ETF", getMacroCategoryKey("ETF_BR", "BOVA11"))
	assert.Equal(t, "CRYPTO", getMacroCategoryKey("CRYPTO", "BTC-USD"))
	assert.Equal(t, "BDR", getMacroCategoryKey("BDR", "AAPL34"))
	assert.Equal(t, "RF", getMacroCategoryKey("RF", "CDB Banco X"))
	assert.Equal(t, "STOCK", getMacroCategoryKey("STOCK_BR", "PETR4"))
}

func TestHandleHistory_WithFee(t *testing.T) {
	h, svc, portSvc, _, _, _ := setupHandlersTest()

	t.Run("renders transaction history with fee when fee > 0", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0")

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Carteira 1", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		txs := []portfolio.Transaction{
			{ID: "t1", Ticker: "PETR4", Type: "BUY", ExecutedAt: now, Quantity: 10, UnitPrice: 30, TotalCost: 300, Fee: 10.5},
		}
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Taxas: R$ 10,50")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "Taxas: R$ 10,50")
	})
}

func TestHandlePortfolioSummary_IncludesTreasury(t *testing.T) {
	h, svc, portSvc, _, fiSvc, _ := setupHandlersTest()

	t.Run("includes treasury positions in portfolio total value and breakdown", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "PETR4", Quantity: 100, CurrentPrice: 30, CurrentValue: 3000, TotalCost: 2500, Type: "STOCK_BR"},
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()

		fiPositions := []fixedincome.Position{
			{GrossValue: 2100, NetValue: 2000, TotalInvested: 1900, Asset: fixedincome.Asset{Institution: "Banco Y", Type: "CDB"}},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPositions, nil).Once()

		trPositions := []fixedincome.TreasuryPosition{
			{TreasuryType: "SELIC", GrossValue: 5100, NetValue: 5000, TotalInvested: 4800, Quantity: 0.5},
		}
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return(trPositions, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "10.000,00") // 3000 (stock) + 2000 (cdb) + 5000 (treasury) = 10000
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "10.000,00")
	})
}

func TestHandlePortfolioSummary_WithBenchmarks(t *testing.T) {
	h, svc, portSvc, mSvc, fiSvc, _ := setupHandlersTest()

	t.Run("displays benchmarks when available", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "PETR4", Quantity: 100, CurrentPrice: 30, CurrentValue: 3000, TotalCost: 2500, Type: "STOCK_BR"},
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()

		benchmarks := &market.MarketBenchmarks{
			IBOV:   &market.BenchmarkItem{Symbol: "^BVSP", Name: "Ibovespa", ChangePercent: 1.25},
			IFIX:   &market.BenchmarkItem{Symbol: "IFIX.SA", Name: "IFIX", ChangePercent: -0.45},
			SP500:  &market.BenchmarkItem{Symbol: "^GSPC", Name: "S&P 500", ChangePercent: 0.0},
			USDBRL: &market.BenchmarkItem{Symbol: "BRL=X", Name: "Dólar", ChangePercent: 0.85},
		}
		mSvc.On("GetBenchmarks", mock.Anything).Return(benchmarks, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Benchmarks do Dia") &&
				strings.Contains(msg, "IBOV") &&
				strings.Contains(msg, "IFIX") &&
				strings.Contains(msg, "S&P 500") &&
				strings.Contains(msg, "Dólar")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "Benchmarks do Dia")
		assert.Contains(t, sentMsg, "+1,25%")
		assert.Contains(t, sentMsg, "-0,45%")
	})
}

func TestHandleAssetList(t *testing.T) {
	h, svc, portSvc, _, _, _ := setupHandlersTest()

	t.Run("success with pagination", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0")

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "PETR4", Quantity: 100, CurrentPrice: 30, CurrentValue: 3000, TotalCost: 2500, DailyChangePercent: 2.5},
			{Ticker: "VALE3", Quantity: 50, CurrentPrice: 60, CurrentValue: 3000, TotalCost: 3200, DailyChangePercent: -1.2},
			{Ticker: "WEGE3", Quantity: 10, CurrentPrice: 40, CurrentValue: 400, TotalCost: 400, DailyChangePercent: 0.0},
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Ativos: Minha Carteira") &&
				strings.Contains(msg, "PETR4") &&
				strings.Contains(msg, "VALE3")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "Página 1 de 1")
		assert.Contains(t, sentMsg, "PETR4")
	})

	t.Run("empty positions", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0")

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], []portfolio.Position{}, nil).Once()

		mCtx.On("Edit", "📋 *Ativos: Minha Carteira*\n\nNenhum ativo encontrado nesta carteira.", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandleHistory_Filter(t *testing.T) {
	h, svc, portSvc, _, _, _ := setupHandlersTest()

	t.Run("filter BUY only", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0:BUY")

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		txs := []portfolio.Transaction{
			{ID: "t1", Ticker: "PETR4", Type: "BUY", ExecutedAt: now, Quantity: 10, UnitPrice: 30, TotalCost: 300},
			{ID: "t2", Ticker: "VALE3", Type: "SELL", ExecutedAt: now, Quantity: 5, UnitPrice: 60, TotalCost: 300},
		}
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Compras") && strings.Contains(msg, "PETR4") && !strings.Contains(msg, "VALE3")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "PETR4")
		assert.NotContains(t, sentMsg, "VALE3")
	})

	t.Run("filter SELL only", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0:SELL")

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		txs := []portfolio.Transaction{
			{ID: "t1", Ticker: "PETR4", Type: "BUY", ExecutedAt: now, Quantity: 10, UnitPrice: 30, TotalCost: 300},
			{ID: "t2", Ticker: "VALE3", Type: "SELL", ExecutedAt: now, Quantity: 5, UnitPrice: 60, TotalCost: 300},
		}
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()

		var sentMsg string
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			sentMsg = msg
			return strings.Contains(msg, "Vendas") && strings.Contains(msg, "VALE3") && !strings.Contains(msg, "PETR4")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
		assert.Contains(t, sentMsg, "VALE3")
		assert.NotContains(t, sentMsg, "PETR4")
	})

	t.Run("filter SELL with no results", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0:SELL")

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		txs := []portfolio.Transaction{
			{ID: "t1", Ticker: "PETR4", Type: "BUY", ExecutedAt: now, Quantity: 10, UnitPrice: 30, TotalCost: 300},
		}
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Nenhuma operação encontrada")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_EdgeCases_FullCoverage(t *testing.T) {
	h, svc, portSvc, _, fiSvc, _ := setupHandlersTest()

	t.Run("resolveActivePortfolio all branches", func(t *testing.T) {
		ctx := context.Background()
		// 1. len(portfolios) == 0
		id, name := h.resolveActivePortfolio(ctx, 123, nil)
		assert.Empty(t, id)
		assert.Empty(t, name)

		// 2. ActiveID found in redis and matches
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p2", nil).Once()
		pList := []portfolio.Portfolio{{ID: "p1", Name: "P1"}, {ID: "p2", Name: "P2"}}
		id, name = h.resolveActivePortfolio(ctx, 123, pList)
		assert.Equal(t, "p2", id)
		assert.Equal(t, "P2", name)

		// 3. ActiveID not found, fallback to IsDefault
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("", errors.New("not found")).Once()
		pListDefault := []portfolio.Portfolio{{ID: "p1", Name: "P1"}, {ID: "p2", Name: "P2", IsDefault: true}}
		id, name = h.resolveActivePortfolio(ctx, 123, pListDefault)
		assert.Equal(t, "p2", id)
		assert.Equal(t, "P2", name)

		// 4. ActiveID not found, no IsDefault, fallback to portfolios[0]
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("", errors.New("not found")).Once()
		pListFirst := []portfolio.Portfolio{{ID: "p1", Name: "P1"}, {ID: "p2", Name: "P2"}}
		id, name = h.resolveActivePortfolio(ctx, 123, pListFirst)
		assert.Equal(t, "p1", id)
		assert.Equal(t, "P1", name)
	})

	t.Run("HandleLaunchOperation empty portfolios and state error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{}, nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil).Once()

		err := h.HandleLaunchOperation(mCtx)
		assert.NoError(t, err)

		// State error
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(errors.New("redis err")).Once()
		mCtx2.On("Edit", "❌ Erro interno ao iniciar operação.", mock.Anything).Return(nil).Once()

		err = h.HandleLaunchOperation(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("HandleNewAsset state nil", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mCtx.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err := h.HandleNewAsset(mCtx)
		assert.NoError(t, err)
	})

	t.Run("handleSelectedTicker and handleSetType state nil and Send path", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mCtx.On("Send", "⚠️ Nenhuma operação em andamento. Envie /menu e clique em Lançar Operação.", mock.Anything).Return(nil).Once()

		err := h.handleSelectedTicker(mCtx, "AAPL")
		assert.NoError(t, err)

		// handleSelectedTicker Send path (c.Callback() == nil)
		mCtxSend := new(MockTelebotContext)
		mCtxSend.On("Respond", mock.Anything).Return(nil).Once()
		mCtxSend.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxSend.On("Callback").Return((*telebot.Callback)(nil))
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_TICKER"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		mCtxSend.On("Send", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.handleSelectedTicker(mCtxSend, "AAPL")
		assert.NoError(t, err)

		// handleSetType state nil
		mCtx3 := new(MockTelebotContext)
		mCtx3.On("Respond", mock.Anything).Return(nil).Once()
		mCtx3.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mCtx3.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err = h.handleSetType(mCtx3, "BUY")
		assert.NoError(t, err)

		// handleSelectedQty state nil
		mCtx4 := new(MockTelebotContext)
		mCtx4.On("Respond", mock.Anything).Return(nil).Once()
		mCtx4.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mCtx4.On("Edit", "⚠️ Nenhuma operação em andamento.", mock.Anything).Return(nil).Once()

		err = h.handleSelectedQty(mCtx4, "10")
		assert.NoError(t, err)
	})

	t.Run("HandleDynamicCallback unmatched and with prefix", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Callback").Return(&telebot.Callback{Data: "\funknown_btn"}).Once()
		err := h.HandleDynamicCallback(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText SELL success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("50,00")

		state := &ConversationState{
			Step:        "EXPECT_PRICE",
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Type:        "SELL",
			Quantity:    10,
		}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(state, nil).Once()
		portSvc.On("AddTransaction", mock.Anything, "00000000-0000-0000-0000-000000000000", mock.Anything).Return(&portfolio.Transaction{}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mCtx.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "VENDA")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleText EXPECT_QTY success with comma", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("15,5")

		state := &ConversationState{
			Step:        "EXPECT_QTY",
			PortfolioID: "p1",
			Ticker:      "PETR4",
			Type:        "BUY",
		}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(state, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(nil).Once()
		mCtx.On("Send", "Qual o preço unitário da transação? (ex: 15.50)", mock.Anything).Return(nil).Once()

		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandlePortfolioSummary negative change and fallers", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})

		portfolios := []portfolio.Portfolio{
			{ID: "p1", Name: "Minha Carteira", IsDefault: true},
		}
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		positions := []portfolio.Position{
			{Ticker: "VALE3", Quantity: 100, CurrentPrice: 50, CurrentValue: 5000, TotalCost: 6000, DailyChange: -2, DailyChangePercent: -3.85},
			{Ticker: "MGLU3", Quantity: 100, CurrentPrice: 10, CurrentValue: 1000, TotalCost: 1500, DailyChange: -1, DailyChangePercent: -9.09},
			{Ticker: "B3SA3", Quantity: 100, CurrentPrice: 12, CurrentValue: 1200, TotalCost: 1200, DailyChange: 0, DailyChangePercent: 0.0},
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], positions, nil).Once()

		fiPositions := []fixedincome.Position{
			{GrossValue: 1000, NetValue: 900, TotalInvested: 1000, DaysToMaturity: 10, IsMatured: false, Asset: fixedincome.Asset{Institution: "Banco Y", Type: "CDB"}},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPositions, nil).Once()

		trPositions := []fixedincome.TreasuryPosition{
			{TreasuryType: "IPCA", GrossValue: 1000, NetValue: 900, TotalInvested: 1000, DaysToMaturity: 5, IsMatured: false, Quantity: 0.2},
		}
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return(trPositions, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Maiores Baixas do Dia") &&
				strings.Contains(msg, "Vencimentos Próximos")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAssetList errors and pagination", func(t *testing.T) {
		// 1. Portfolios error
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Data").Return("0")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(([]portfolio.Portfolio)(nil), errors.New("db err")).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)

		// 2. Details error
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx2.On("Data").Return("1")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("db err")).Once()
		mCtx2.On("Edit", "❌ Ocorreu um erro ao buscar sua carteira.", mock.Anything).Return(nil).Once()

		err = h.HandleAssetList(mCtx2)
		assert.NoError(t, err)

		// 3. Multi-page pagination (15 positions)
		mCtx3 := new(MockTelebotContext)
		mCtx3.On("Respond", mock.Anything).Return(nil).Once()
		mCtx3.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx3.On("Data").Return("1") // Page 2
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		var positions15 []portfolio.Position
		for i := 1; i <= 15; i++ {
			positions15 = append(positions15, portfolio.Position{
				Ticker:             fmt.Sprintf("TICK%d", i),
				CurrentValue:       float64(100 * i),
				TotalCost:          float64(90 * i),
				DailyChangePercent: float64(i - 8),
			})
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, positions15, nil).Once()
		mCtx3.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Página 2 de 2")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleAssetList(mCtx3)
		assert.NoError(t, err)
	})

	t.Run("HandleChangePortfolio error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(([]portfolio.Portfolio)(nil), errors.New("err")).Once()
		mCtx.On("Edit", "⚠️ Nenhuma carteira encontrada na sua conta.", mock.Anything).Return(nil).Once()

		err := h.HandleChangePortfolio(mCtx)
		assert.NoError(t, err)
	})

	t.Run("handleSelectedPortfolio get portfolios error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(([]portfolio.Portfolio)(nil), errors.New("err")).Once()
		mCtx.On("Edit", "❌ Erro ao buscar carteiras.", mock.Anything).Return(nil).Once()

		err := h.handleSelectedPortfolio(mCtx, "p1")
		assert.NoError(t, err)
	})

	t.Run("HandleDividends and HandleFixedIncome message not modified", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return([]portfolio.CalculatedDividend{}, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)

		// Fixed income message not modified
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{
			{NetValue: 100, TotalInvested: 100, DaysToMaturity: 100, Asset: fixedincome.Asset{Institution: "B1", Type: "CDB", DebtType: "PRE"}},
		}, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err = h.HandleFixedIncome(mCtx2)
		assert.NoError(t, err)
	})
}

func TestHandlers_DeepBranchCoverage(t *testing.T) {
	h, svc, portSvc, mSvc, fiSvc, _ := setupHandlersTest()

	t.Run("HandleDividends with multiple currencies, >5 items, empty currency, zero qty", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		now := time.Now()
		past := now.AddDate(0, 0, -5)
		future := now.AddDate(0, 0, 5)

		var divs []portfolio.CalculatedDividend
		// Add 6 past divs to trigger past limit and currency == ""
		for i := 1; i <= 6; i++ {
			curr := "USD"
			if i == 1 {
				curr = ""
			}
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:         fmt.Sprintf("PAST%d", i),
				AssetType:      "STOCK_BR",
				Currency:       curr,
				PaymentDate:    past,
				NetAmount:      10.0,
				Quantity:       0, // trigger quantity <= 0 branch
				PerShareAmount: 0,
			})
		}
		// Add 6 future divs to trigger future limit
		for i := 1; i <= 6; i++ {
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:         fmt.Sprintf("FUT%d", i),
				AssetType:      "CRYPTO",
				Currency:       "", // trigger currency == "" branch
				PaymentDate:    future,
				NetAmount:      20.0,
				Quantity:       10,
				PerShareAmount: 2.0,
			})
		}

		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByYear with YoY negative growth, year <= 1, year > currentYear, >3 shares", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		currentYear := time.Now().Year()

		divs := []portfolio.CalculatedDividend{
			// Year 0 (A Definir)
			{Ticker: "UNDEFINED", Currency: "", PaymentDate: time.Time{}, NetAmount: 100, Type: "DIVIDENDO"},
			// Year 2022 (higher)
			{Ticker: "T1", Currency: "BRL", PaymentDate: time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 1000, Type: "JCP"},
			// Year 2023 (lower -> negative YoY)
			{Ticker: "T1", Currency: "BRL", PaymentDate: time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 500, Type: "JCP"},
			// Year 2024 (higher -> positive YoY)
			{Ticker: "T1", Currency: "BRL", PaymentDate: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 1000, Type: "JCP"},
			// Year 2025 (with multiple types and tickers)
			{Ticker: "T1", Currency: "BRL", PaymentDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 500, Type: "RENDIMENTO"},
			{Ticker: "T2", Currency: "BRL", PaymentDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 300, Type: "AMORTIZACAO"},
			{Ticker: "T3", Currency: "BRL", PaymentDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 200, Type: "CUSTOM"},
			{Ticker: "T4", Currency: "BRL", PaymentDate: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC), NetAmount: 100, Type: "CUSTOM"},
			// Year currentYear + 1 (future year)
			{Ticker: "TFUT", Currency: "USD", PaymentDate: time.Date(currentYear+1, 1, 1, 0, 0, 0, 0, time.UTC), NetAmount: 50, Type: "DIV"},
		}

		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividendsByYear(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth with A Definir on page 0", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0") // page 0 -> displays A Definir
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		divs := []portfolio.CalculatedDividend{
			{Ticker: "UNDEF", Currency: "", PaymentDate: time.Time{}, NetAmount: 100},
		}
		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "A Definir")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth page 0 with next button", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0") // page 0 -> 4 items, has next button
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		divs := []portfolio.CalculatedDividend{
			{Ticker: "M1", Currency: "BRL", PaymentDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), NetAmount: 50},
			{Ticker: "M2", Currency: "BRL", PaymentDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), NetAmount: 50},
			{Ticker: "M3", Currency: "BRL", PaymentDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), NetAmount: 50},
			{Ticker: "M4", Currency: "BRL", PaymentDate: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), NetAmount: 50},
		}
		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth multi-page nav with start > 0", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("1") // page 1 -> start > 0
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		var divs []portfolio.CalculatedDividend
		for i := 1; i <= 6; i++ {
			divs = append(divs, portfolio.CalculatedDividend{
				Ticker:      fmt.Sprintf("M%d", i),
				Currency:    "BRL",
				PaymentDate: time.Date(2025, time.Month(i), 15, 0, 0, 0, 0, time.UTC),
				NetAmount:   50,
			})
		}

		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome errors and status branches", func(t *testing.T) {
		// Both errors
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(([]fixedincome.Position)(nil), errors.New("err1")).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return(([]fixedincome.TreasuryPosition)(nil), errors.New("err2")).Once()
		mCtx.On("Edit", "❌ Erro ao buscar posições de Renda Fixa.", mock.Anything).Return(nil).Once()

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)

		// Matured and near maturity positions
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		fiPos := []fixedincome.Position{
			{GrossValue: 100, NetValue: 90, TotalInvested: 0, IsMatured: true, Asset: fixedincome.Asset{Institution: "B1", Type: "CDB", DebtType: "PRE", Rate: 10.0}},
			{GrossValue: 200, NetValue: 180, TotalInvested: 150, DaysToMaturity: 15, IsMatured: false, Asset: fixedincome.Asset{Institution: "B2", Type: "LCI", DebtType: "POS", Rate: 95.0, Indexer: "CDI"}},
		}
		trPos := []fixedincome.TreasuryPosition{
			{TreasuryType: "SELIC", GrossValue: 100, NetValue: 90, TotalInvested: 0, IsMatured: true, Quantity: 0.1},
			{TreasuryType: "IPCA", GrossValue: 200, NetValue: 180, TotalInvested: 150, DaysToMaturity: 20, IsMatured: false, Quantity: 0.2},
		}
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return(fiPos, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return(trPos, nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleFixedIncome(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("HandlePortfolioSummary with risers limit and benchmarks negative/zero", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		var positions6 []portfolio.Position
		for i := 1; i <= 6; i++ {
			positions6 = append(positions6, portfolio.Position{
				Ticker:             fmt.Sprintf("RISE%d", i),
				Quantity:           10,
				CurrentPrice:       0, // rate = 1.0
				CurrentValue:       float64(100 * i),
				TotalCost:          float64(80 * i),
				DailyChange:        5.0,
				DailyChangePercent: float64(5 * i),
				Type:               "STOCK",
			})
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, positions6, nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()

		benchmarks := &market.MarketBenchmarks{
			IBOV:   &market.BenchmarkItem{Symbol: "^BVSP", Name: "Ibovespa", ChangePercent: -1.5},
			IFIX:   &market.BenchmarkItem{Symbol: "IFIX.SA", Name: "IFIX", ChangePercent: 0.0},
			SP500:  &market.BenchmarkItem{Symbol: "^GSPC", Name: "S&P 500", ChangePercent: -0.5},
			USDBRL: &market.BenchmarkItem{Symbol: "BRL=X", Name: "Dólar", ChangePercent: 0.0},
		}
		mSvc.On("GetBenchmarks", mock.Anything).Return(benchmarks, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandlePortfolioSummary message not modified", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, []portfolio.Position{}, nil).Once()
		fiSvc.On("GetPortfolioPositions", mock.Anything, "p1").Return([]fixedincome.Position{}, nil).Once()
		fiSvc.On("GetTreasuryPositions", mock.Anything, "p1").Return([]fixedincome.TreasuryPosition{}, nil).Once()
		mSvc.On("GetBenchmarks", mock.Anything).Return((*market.MarketBenchmarks)(nil), errors.New("err")).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandlePortfolioSummary details error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return((*portfolio.Portfolio)(nil), ([]portfolio.Position)(nil), errors.New("err")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar sua carteira.", mock.Anything).Return(nil).Once()

		err := h.HandlePortfolioSummary(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAssetList message not modified and zero change", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pos := []portfolio.Position{
			{Ticker: "FLAT", CurrentValue: 100, TotalCost: 0, DailyChangePercent: 0.0},
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, pos, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory message not modified and multi-page nav", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("1:ALL") // page 1 -> start > 0 and end < len(filtered)
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		var txs []portfolio.Transaction
		for i := 1; i <= 25; i++ {
			txs = append(txs, portfolio.Transaction{
				ID:          fmt.Sprintf("t%d", i),
				Ticker:      "PETR4",
				Type:        "BUY",
				Quantity:    10,
				UnitPrice:   30,
				TotalCost:   300,
				ExecutedAt:  time.Now(),
				Fee:         0,
			})
		}
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividends fetch error and empty currency", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(([]portfolio.Portfolio)(nil), errors.New("err")).Once()
		mCtx.On("Edit", "❌ Ocorreu um erro ao buscar os proventos da sua carteira.", mock.Anything).Return(nil).Once()

		err := h.HandleDividends(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleDividendsByMonth item sorting and empty currency", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		date1 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
		divs := []portfolio.CalculatedDividend{
			{Ticker: "BBAS3", Type: "DIV", Currency: "", PaymentDate: date1, NetAmount: 10},
			{Ticker: "BBAS3", Type: "JCP", Currency: "BRL", PaymentDate: date1, NetAmount: 10},
			{Ticker: "BBAS3", Type: "JCP", Currency: "USD", PaymentDate: date1, NetAmount: 10},
			{Ticker: "PETR4", Type: "DIV", Currency: "BRL", PaymentDate: date1, NetAmount: 10},
		}
		portSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleFixedIncome nil service", func(t *testing.T) {
		hNilFI := NewHandlers(svc, portSvc, mSvc, nil, nil)
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Edit", "⚠️ Módulo de Renda Fixa não está ativo.", mock.Anything).Return(nil).Once()

		err := hNilFI.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleHistory unknown filter, negative page, page out of bounds, and empty filter msg not modified", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("-1:UNKNOWN") // negative page & unknown filter
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		txs := []portfolio.Transaction{
			{ID: "t1", Ticker: "PETR4", Type: "BUY", Quantity: 10, UnitPrice: 30, TotalCost: 300, ExecutedAt: time.Now()},
		}
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleHistory(mCtx)
		assert.NoError(t, err)

		// Empty filtered message not modified
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx2.On("Data").Return("0:SELL")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioTransactions", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(txs, nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("telegram: message is not modified")).Once()

		err = h.HandleHistory(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("HandleText default step", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("some text")

		state := &ConversationState{
			Step: "UNKNOWN_STEP",
		}
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(state, nil).Once()
		err := h.HandleText(mCtx)
		assert.NoError(t, err)
	})

	t.Run("HandleAssetList negative page and page out of bounds", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("-1")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pos := []portfolio.Position{
			{Ticker: "PETR4", CurrentValue: 100, TotalCost: 90, DailyChangePercent: 2.0},
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, pos, nil).Once()
		mCtx.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)

		// Page 99 (start >= len(posByValue))
		mCtx2 := new(MockTelebotContext)
		mCtx2.On("Respond", mock.Anything).Return(nil).Once()
		mCtx2.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx2.On("Data").Return("99")
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, pos, nil).Once()
		mCtx2.On("Edit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err = h.HandleAssetList(mCtx2)
		assert.NoError(t, err)
	})

	t.Run("HandleAssetList getUserID error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil)
		mCtx.On("Data").Return("0")
		mCtx.On("Callback").Return(&telebot.Callback{})
		mCtx.On("Set", "user_id", nil).Return()
		mCtx.Set("user_id", nil)
		mCtx.On("Edit", "⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.", mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.Error(t, err)
	})

	t.Run("HandleText EXPECT_PRICE getUserID error", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Text").Return("150.50")
		mCtx.On("Callback").Return((*telebot.Callback)(nil))
		mCtx.On("Set", "user_id", nil).Return()
		mCtx.Set("user_id", nil)
		mCtx.On("Send", "⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.", mock.Anything).Return(nil).Once()

		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_PRICE", PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10}, nil).Once()

		err := h.HandleText(mCtx)
		assert.Error(t, err)
	})

	t.Run("HandleAssetList page 0 next button", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Respond", mock.Anything).Return(nil).Once()
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Data").Return("0") // page 0
		portSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{{ID: "p1", Name: "P1"}}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		var positions15 []portfolio.Position
		for i := 1; i <= 15; i++ {
			positions15 = append(positions15, portfolio.Position{
				Ticker:             fmt.Sprintf("TICK%d", i),
				CurrentValue:       float64(100 * i),
				TotalCost:          float64(90 * i),
				DailyChangePercent: float64(i - 8),
			})
		}
		portSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolio.Portfolio{ID: "p1"}, positions15, nil).Once()
		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Página 1 de 2")
		}), mock.Anything, mock.Anything).Return(nil).Once()

		err := h.HandleAssetList(mCtx)
		assert.NoError(t, err)
	})
}

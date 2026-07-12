package telegram

import (
	"errors"
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
	h, svc, _, _, _ := setupHandlersTest()

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
	h, svc, pSvc, _, _ := setupHandlersTest()

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
	h, svc, pSvc, _, fiSvc := setupHandlersTest()

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
	h, svc, pSvc, _, _ := setupHandlersTest()

	t.Run("HandleDividends - success", func(t *testing.T) {
		mCtx := new(MockTelebotContext)
		mCtx.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtx.On("Respond", mock.Anything).Return(nil).Once()

		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return([]portfolio.Portfolio{
			{ID: "p1", Name: "P1"},
		}, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()

		// Set fixed day in middle of month to prevent test failures near month boundaries
		now := time.Date(time.Now().Year(), time.Now().Month(), 15, 12, 0, 0, 0, time.Local)
		divs := []portfolio.CalculatedDividend{
			{Ticker: "AAPL", NetAmount: 10.0, PaymentDate: now.AddDate(0, 0, -5), Type: "DIVIDENDO", Currency: "USD"},
			{Ticker: "MSFT", NetAmount: 15.0, PaymentDate: now.AddDate(0, 0, 5), Type: "JCP", Currency: "BRL"},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			hasTitle := strings.Contains(msg, "💸 *Proventos: P1*")
			hasUSD := strings.Contains(msg, "US$ 10,00")
			hasBRL := strings.Contains(msg, "R$ 15,00")
			hasAAPL := strings.Contains(msg, "✅ `AAPL` (DIV) • US$ 10,00 • "+now.AddDate(0, 0, -5).Format("2006-01-02"))
			hasMSFT := strings.Contains(msg, "⏳ `MSFT` (JCP) • R$ 15,00 • "+now.AddDate(0, 0, 5).Format("2006-01-02"))
			return hasTitle && hasUSD && hasBRL && hasAAPL && hasMSFT
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
			has2026 := strings.Contains(msg, "• *2026*: R$ 15,00")
			has2025 := strings.Contains(msg, "• *2025*: US$ 10,00")
			return hasTitle && has2026 && has2025
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
			{Ticker: "AAPL", NetAmount: 10.0, PaymentDate: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), Type: "DIVIDENDO", Currency: "USD"},
			{Ticker: "MSFT", NetAmount: 15.0, PaymentDate: time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC), Type: "JCP", Currency: "BRL"},
		}
		pSvc.On("GetPortfolioDividends", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(divs, nil).Once()

		mCtx.On("Edit", mock.MatchedBy(func(msg string) bool {
			hasTitle := strings.Contains(msg, "📆 *Proventos por Mês: P1*")
			hasMonthTotal := strings.Contains(msg, "• *2026-05*: R$ 15,00 | US$ 10,00")
			hasAAPL := strings.Contains(msg, "↳ `AAPL` (DIV) • US$ 10,00 • Dia 10")
			hasMSFT := strings.Contains(msg, "↳ `MSFT` (JCP) • R$ 15,00 • Dia 12")
			return hasTitle && hasMonthTotal && hasAAPL && hasMSFT
		}), mock.Anything).Return(nil)

		err := h.HandleDividendsByMonth(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_HistoryAndFixedIncome(t *testing.T) {
	h, svc, pSvc, _, fiSvc := setupHandlersTest()

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
		mCtx.On("Edit", mock.Anything, mock.Anything).Return(nil)

		err := h.HandleFixedIncome(mCtx)
		assert.NoError(t, err)
	})
}

func TestHandlers_Operations(t *testing.T) {
	h, svc, pSvc, mSvc, _ := setupHandlersTest()

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
	h, svc, pSvc, _, fiSvc := setupHandlersTest()

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
		mCtx.On("Edit", "🏛️ Você ainda não possui ativos de Renda Fixa cadastrados.", mock.Anything).Return(nil)

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
	h, svc, pSvc, mSvc, _ := setupHandlersTest()

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
	h, svc, pSvc, _, _ := setupHandlersTest()

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

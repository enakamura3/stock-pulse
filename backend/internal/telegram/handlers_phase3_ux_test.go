package telegram

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/telebot.v3"
)

func TestPhase3_OperationBreadcrumbAndParsers(t *testing.T) {
	t.Run("formatOperationBreadcrumb progressive steps and nil guard", func(t *testing.T) {
		assert.Equal(t, "", formatOperationBreadcrumb(nil, 1))

		state := &ConversationState{
			PortfolioName: "Minha_Carteira",
			Ticker:        "PETR4.SA",
			Type:          "SELL",
			Quantity:      100,
			UnitPrice:     36.50,
			ExecutedAt:    "2026-03-15",
		}

		// Step 1 shows portfolio and ticker, hides type/qty/price/date
		s1 := formatOperationBreadcrumb(state, 1)
		assert.Contains(t, s1, `Minha\_Carteira`)
		assert.Contains(t, s1, "`PETR4`")
		assert.NotContains(t, s1, "Venda")
		assert.NotContains(t, s1, "Qtd:")

		// Step 2 adds type, still hides qty/price/date
		s2 := formatOperationBreadcrumb(state, 2)
		assert.Contains(t, s2, "🔴 Venda")
		assert.NotContains(t, s2, "Qtd:")

		// Step 3 adds quantity, hides price/date
		s3 := formatOperationBreadcrumb(state, 3)
		assert.Contains(t, s3, "Qtd: `100`")
		assert.NotContains(t, s3, "Subtotal:")

		// Step 4 adds price and subtotal, hides date
		s4 := formatOperationBreadcrumb(state, 4)
		assert.Contains(t, s4, "R$ 36,50")
		assert.Contains(t, s4, "Subtotal: R$ 3.650,00")
		assert.NotContains(t, s4, "Data:")

		// Step 5 adds date in AAAA-MM-DD
		s5 := formatOperationBreadcrumb(state, 5)
		assert.Contains(t, s5, "Data: `2026-03-15`")
	})

	t.Run("parseOperationDate supports AAAA-MM-DD, hoje, ontem and legacy DD/MM/AAAA fallback", func(t *testing.T) {
		dISO, err := parseOperationDate("2026-03-15")
		assert.NoError(t, err)
		assert.Equal(t, "2026-03-15", dISO.Format("2006-01-02"))

		dBR, err := parseOperationDate("15/03/2026")
		assert.NoError(t, err)
		assert.Equal(t, "2026-03-15", dBR.Format("2006-01-02"))

		dDash, err := parseOperationDate("15-03-2026")
		assert.NoError(t, err)
		assert.Equal(t, "2026-03-15", dDash.Format("2006-01-02"))

		dToday, err := parseOperationDate("hoje")
		assert.NoError(t, err)
		assert.Equal(t, time.Now().Format("2006-01-02"), dToday.Format("2006-01-02"))

		dYest, err := parseOperationDate("ontem")
		assert.NoError(t, err)
		assert.Equal(t, time.Now().AddDate(0, 0, -1).Format("2006-01-02"), dYest.Format("2006-01-02"))

		_, err = parseOperationDate("not-a-date")
		assert.Error(t, err)
	})

	t.Run("parseQuickOperationArgs valid and invalid inputs", func(t *testing.T) {
		// Valid buy with date and fee
		opType, ticker, qty, price, dateStr, fee, errMsg := parseQuickOperationArgs([]string{"c", "petr4", "100", "36,50", "2026-03-15", "4,50"})
		assert.Empty(t, errMsg)
		assert.Equal(t, "BUY", opType)
		assert.Equal(t, "PETR4", ticker)
		assert.InDelta(t, 100.0, qty, 1e-6)
		assert.InDelta(t, 36.50, price, 1e-6)
		assert.Equal(t, "2026-03-15", dateStr)
		assert.InDelta(t, 4.50, fee, 1e-6)

		// Valid sell with zero fee normalized
		opType, ticker, qty, price, _, fee, errMsg = parseQuickOperationArgs([]string{"VENDA", "VALE3", "50", "60.00", "2026-03-15", "0"})
		assert.Empty(t, errMsg)
		assert.Equal(t, "SELL", opType)
		assert.Equal(t, "VALE3", ticker)
		assert.InDelta(t, 50.0, qty, 1e-6)
		assert.InDelta(t, 60.0, price, 1e-6)
		assert.InDelta(t, 0.0, fee, 1e-6)

		// Error cases
		_, _, _, _, _, _, errMsg = parseQuickOperationArgs([]string{"X", "PETR4", "10", "30"})
		assert.Contains(t, errMsg, "Tipo inválido")

		_, _, _, _, _, _, errMsg = parseQuickOperationArgs([]string{"C", "PETR4", "0", "30"})
		assert.Contains(t, errMsg, "Quantidade inválida")

		_, _, _, _, _, _, errMsg = parseQuickOperationArgs([]string{"C", "PETR4", "10", "-5"})
		assert.Contains(t, errMsg, "Preço unitário inválido")

		_, _, _, _, _, _, errMsg = parseQuickOperationArgs([]string{"C", "PETR4", "10", "30", "invalida"})
		assert.Contains(t, errMsg, "Data inválida")

		_, _, _, _, _, _, errMsg = parseQuickOperationArgs([]string{"C", "PETR4", "10", "30", "2026-03-15", "-1"})
		assert.Contains(t, errMsg, "Taxa inválida")
	})
}

func TestPhase3_OperationShortcutsAndConfirmation(t *testing.T) {
	h, svc, pSvc, mSvc, _, _ := setupHandlersTest()
	portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Principal"}}

	t.Run("/operacao PETR4 single-arg shortcut branches", func(t *testing.T) {
		// 1. Quote not found
		mCtxErr := new(MockTelebotContext)
		mCtxErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxErr.On("Args").Return([]string{"INVALID"})
		mCtxErr.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "INVALID").Return((*market.Quote)(nil), errors.New("not found")).Once()
		mCtxErr.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo não encontrado")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxErr))

		// 2. State save error
		mCtxStErr := new(MockTelebotContext)
		mCtxStErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxStErr.On("Args").Return([]string{"PETR4"})
		mCtxStErr.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(errors.New("redis err")).Once()
		mCtxStErr.On("Send", "❌ Erro interno ao iniciar operação.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxStErr))

		// 3. Success -> jumps to EXPECT_TYPE with breadcrumb
		mCtxOk := new(MockTelebotContext)
		mCtxOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxOk.On("Args").Return([]string{"petr4"})
		mCtxOk.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{
			Step:          "EXPECT_TYPE",
			PortfolioID:   "p1",
			PortfolioName: "Principal",
			Ticker:        "PETR4.SA",
		}).Return(nil).Once()
		mCtxOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Nova Operação") && strings.Contains(msg, "Passo 1/5") && strings.Contains(msg, "PETR4")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxOk))
	})

	t.Run("/operacao 1-line shortcut validation, SELL custody, confirmation card, confirm & edit", func(t *testing.T) {
		// 1. Invalid arg count (2 args) -> usage hint
		mCtxUsage := new(MockTelebotContext)
		mCtxUsage.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxUsage.On("Args").Return([]string{"C", "PETR4"})
		mCtxUsage.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mCtxUsage.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Formato inválido") && strings.Contains(msg, "AAAA-MM-DD")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxUsage))

		// 2. Parse error in 4 args
		mCtxBadArg := new(MockTelebotContext)
		mCtxBadArg.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxBadArg.On("Args").Return([]string{"X", "PETR4", "10", "35"})
		mCtxBadArg.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mCtxBadArg.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Tipo inválido")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxBadArg))

		// 3. Quote not found
		mCtxNoQuote := new(MockTelebotContext)
		mCtxNoQuote.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxNoQuote.On("Args").Return([]string{"C", "NOEXIST", "10", "35"})
		mCtxNoQuote.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "NOEXIST").Return((*market.Quote)(nil), errors.New("404")).Once()
		mCtxNoQuote.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Ativo não encontrado")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxNoQuote))

		// 4. SELL exceeds custody
		mCtxNoCust := new(MockTelebotContext)
		mCtxNoCust.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxNoCust.On("Args").Return([]string{"V", "PETR4", "100", "38,00"})
		mCtxNoCust.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 40},
		}, nil).Once()
		mCtxNoCust.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Quantidade indisponível para venda")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxNoCust))

		// 5. State save error on 1-line BUY
		mCtxStErr := new(MockTelebotContext)
		mCtxStErr.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxStErr.On("Args").Return([]string{"C", "PETR4", "100", "36,50"})
		mCtxStErr.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Currency: "BRL"}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), mock.Anything).Return(errors.New("redis err")).Once()
		mCtxStErr.On("Send", "❌ Erro interno ao iniciar operação.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxStErr))

		// 6. Success 1-line SELL with date AAAA-MM-DD and fee -> renders confirmation card
		mCtxSellOk := new(MockTelebotContext)
		mCtxSellOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxSellOk.On("Args").Return([]string{"V", "PETR4", "50", "38,00", "2026-03-15", "5,00"})
		mCtxSellOk.On("Callback").Return((*telebot.Callback)(nil))
		pSvc.On("GetPortfolios", mock.Anything, "00000000-0000-0000-0000-000000000000").Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		mSvc.On("GetQuote", mock.Anything, "PETR4").Return(&market.Quote{Ticker: "PETR4.SA", Currency: "BRL"}, nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", "00000000-0000-0000-0000-000000000000").Return(&portfolios[0], []portfolio.Position{
			{Ticker: "PETR4.SA", Quantity: 100},
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{
			Step:          "CONFIRM_OP",
			PortfolioID:   "p1",
			PortfolioName: "Principal",
			Ticker:        "PETR4.SA",
			Type:          "SELL",
			Quantity:      50,
			UnitPrice:     38.0,
			ExecutedAt:    "2026-03-15",
			Fee:           5.0,
		}).Return(nil).Once()
		mCtxSellOk.On("Send", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Confirmar Operação") &&
				strings.Contains(msg, "🔴 VENDA") &&
				strings.Contains(msg, "Data: `2026-03-15`") &&
				strings.Contains(msg, "R$ 1.895,00")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleLaunchOperation(mCtxSellOk))

		// 7. HandleOperationConfirm: invalid state & success
		mCtxConfErr := new(MockTelebotContext)
		mCtxConfErr.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return((*ConversationState)(nil), nil).Once()
		mCtxConfErr.On("Edit", "⚠️ Nenhuma operação pendente de confirmação.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleOperationConfirm(mCtxConfErr))

		mCtxConfOk := new(MockTelebotContext)
		mCtxConfOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxConfOk.On("Callback").Return(&telebot.Callback{})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:          "CONFIRM_OP",
			PortfolioID:   "p1",
			PortfolioName: "Principal",
			Ticker:        "PETR4.SA",
			Type:          "BUY",
			Quantity:      100,
			UnitPrice:     36.50,
			ExecutedAt:    "2026-03-15",
			Fee:           4.50,
		}, nil).Once()
		pSvc.On("AddTransaction", mock.Anything, "00000000-0000-0000-0000-000000000000", mock.Anything).Return(&portfolio.Transaction{ID: "tx-ok", Currency: "BRL"}, nil).Once()
		svc.On("ClearConversationState", mock.Anything, int64(123)).Return(nil).Once()
		mCtxConfOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Operação Lançada com Sucesso!")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleOperationConfirm(mCtxConfOk))

		// 8. HandleOperationEditDate: invalid state & success
		mCtxEdErr := new(MockTelebotContext)
		mCtxEdErr.On("Chat").Return(&telebot.Chat{ID: 123})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{Step: "EXPECT_QTY"}, nil).Once()
		mCtxEdErr.On("Edit", "⚠️ Nenhuma operação pendente de confirmação.", mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleOperationEditDate(mCtxEdErr))

		mCtxEdOk := new(MockTelebotContext)
		mCtxEdOk.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxEdOk.On("Callback").Return(&telebot.Callback{})
		svc.On("GetConversationState", mock.Anything, int64(123)).Return(&ConversationState{
			Step:          "CONFIRM_OP",
			PortfolioID:   "p1",
			PortfolioName: "Principal",
			Ticker:        "PETR4.SA",
			Type:          "BUY",
			Quantity:      100,
			UnitPrice:     36.50,
			ExecutedAt:    "2026-03-15",
			Fee:           4.50,
		}, nil).Once()
		svc.On("SetConversationState", mock.Anything, int64(123), ConversationState{
			Step:          "EXPECT_DATE",
			PortfolioID:   "p1",
			PortfolioName: "Principal",
			Ticker:        "PETR4.SA",
			Type:          "BUY",
			Quantity:      100,
			UnitPrice:     36.50,
			ExecutedAt:    "",
			Fee:           0,
		}).Return(nil).Once()
		mCtxEdOk.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Passo 4/5") && strings.Contains(msg, "AAAA-MM-DD")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleOperationEditDate(mCtxEdOk))
	})
}

func TestPhase3_AssetListDynamicCategoryFilter(t *testing.T) {
	t.Run("getMacroCategoryDisplay covers all macro categories", func(t *testing.T) {
		e, n := getMacroCategoryDisplay("STOCK")
		assert.Equal(t, "📈", e)
		assert.Equal(t, "Ações", n)

		e, n = getMacroCategoryDisplay("FII")
		assert.Equal(t, "🏢", e)
		assert.Equal(t, "FIIs", n)

		e, n = getMacroCategoryDisplay("ETF")
		assert.Equal(t, "🌐", e)
		assert.Equal(t, "ETFs", n)

		e, n = getMacroCategoryDisplay("BDR")
		assert.Equal(t, "📦", e)
		assert.Equal(t, "BDRs", n)

		e, n = getMacroCategoryDisplay("CRYPTO")
		assert.Equal(t, "₿", e)
		assert.Equal(t, "Cripto", n)

		e, n = getMacroCategoryDisplay("ALL")
		assert.Equal(t, "📋", e)
		assert.Equal(t, "Todos", n)
	})

	t.Run("HandleAssetList dynamic filter with >=2 classes, class subtotal/pct, invalid & absent category fallback", func(t *testing.T) {
		h, svc, pSvc, _, _, _ := setupHandlersTest()
		portfolios := []portfolio.Portfolio{{ID: "p1", Name: "Completa", BaseCurrency: "BRL"}}
		allPositions := []portfolio.Position{
			{Ticker: "PETR4.SA", Type: "STOCK_BR", Quantity: 100, CurrentValue: 3500, TotalCost: 3000, Currency: "BRL"},
			{Ticker: "HGLG11.SA", Type: "FII", Quantity: 20, CurrentValue: 3000, TotalCost: 2800, Currency: "BRL"},
			{Ticker: "BOVA11.SA", Type: "ETF_BR", Quantity: 10, CurrentValue: 1500, TotalCost: 1400, Currency: "BRL"},
			{Ticker: "AAPL34.SA", Type: "BDR", Quantity: 20, CurrentValue: 1000, TotalCost: 900, Currency: "BRL"},
			{Ticker: "BTC-USD", Type: "CRYPTO", Quantity: 0.01, CurrentValue: 1000, TotalCost: 800, Currency: "USD"},
		}

		// 1. Filter by FII (3000 / 10000 = 30,0% da carteira)
		mCtxFII := new(MockTelebotContext)
		mCtxFII.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxFII.On("Data").Return("0:VAL:FII")
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolios[0], allPositions, nil).Once()

		var fiiMenu *telebot.ReplyMarkup
		mCtxFII.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "🏢 *FIIs:* *R$ 3.000,00* (30,0% da carteira") &&
				strings.Contains(msg, "HGLG11") &&
				!strings.Contains(msg, "PETR4")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					fiiMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleAssetList(mCtxFII))
		assert.NotNil(t, fiiMenu)
		// 6 category buttons (Todos + 5 classes) split into 2 rows of 3
		assert.Equal(t, 3, len(fiiMenu.InlineKeyboard[0]))
		assert.Equal(t, 3, len(fiiMenu.InlineKeyboard[1]))
		// Sort buttons preserve :FII
		assert.Equal(t, "0:VAL:FII", fiiMenu.InlineKeyboard[2][0].Data)

		// 2. Invalid category string falls back to ALL
		mCtxInv := new(MockTelebotContext)
		mCtxInv.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxInv.On("Data").Return("0:VAL:NONEXISTENT")
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolios[0], allPositions, nil).Once()
		mCtxInv.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Patrimônio em Ativos:") && strings.Contains(msg, "PETR4") && strings.Contains(msg, "HGLG11")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleAssetList(mCtxInv))

		// 3. Valid category (CRYPTO) absent in portfolio with 3 classes (STOCK + FII + ETF -> 4 filter buttons = 3+1 rows) falls back to ALL
		threeClassPos := []portfolio.Position{
			{Ticker: "PETR4.SA", Type: "STOCK_BR", Quantity: 100, CurrentValue: 3500, TotalCost: 3000, Currency: "BRL"},
			{Ticker: "HGLG11.SA", Type: "FII", Quantity: 20, CurrentValue: 3000, TotalCost: 2800, Currency: "BRL"},
			{Ticker: "BOVA11.SA", Type: "ETF_BR", Quantity: 10, CurrentValue: 1500, TotalCost: 1400, Currency: "BRL"},
		}
		mCtxAbsent := new(MockTelebotContext)
		mCtxAbsent.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxAbsent.On("Data").Return("0:VAL:CRYPTO")
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolios[0], threeClassPos, nil).Once()
		mCtxAbsent.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Patrimônio em Ativos:")
		}), mock.Anything).Return(nil).Once()
		assert.NoError(t, h.HandleAssetList(mCtxAbsent))

		// 4. Single-class portfolio omits category filter row; also verify >10 items in 2-class preserves filter in pagination
		var manyPos []portfolio.Position
		for i := 1; i <= 11; i++ {
			manyPos = append(manyPos, portfolio.Position{
				Ticker: fmt.Sprintf("STK%d.SA", i), Type: "STOCK_BR", Quantity: 10, CurrentValue: float64(i * 100), TotalCost: float64(i * 80), Currency: "BRL",
			})
		}
		manyPos = append(manyPos, portfolio.Position{Ticker: "HGLG11.SA", Type: "FII", Quantity: 10, CurrentValue: 1500, TotalCost: 1400, Currency: "BRL"})
		mCtxPag := new(MockTelebotContext)
		mCtxPag.On("Chat").Return(&telebot.Chat{ID: 123})
		mCtxPag.On("Data").Return("0:VAL:STOCK")
		pSvc.On("GetPortfolios", mock.Anything, mock.Anything).Return(portfolios, nil).Once()
		svc.On("GetActivePortfolio", mock.Anything, int64(123)).Return("p1", nil).Once()
		pSvc.On("GetPortfolioDetails", mock.Anything, "p1", mock.Anything).Return(&portfolios[0], manyPos, nil).Once()
		var pagMenu *telebot.ReplyMarkup
		mCtxPag.On("Edit", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Página 1 de 2") && strings.Contains(msg, "📈 *Ações:*")
		}), mock.MatchedBy(func(opts []interface{}) bool {
			for _, opt := range opts {
				if m, ok := opt.(*telebot.ReplyMarkup); ok {
					pagMenu = m
					return true
				}
			}
			return false
		})).Return(nil).Once()
		assert.NoError(t, h.HandleAssetList(mCtxPag))
		assert.NotNil(t, pagMenu)
		// Nav row preserves :STOCK
		foundNextWithCat := false
		for _, row := range pagMenu.InlineKeyboard {
			for _, btn := range row {
				if strings.Contains(btn.Text, "Próxima") && btn.Data == "1:VAL:STOCK" {
					foundNextWithCat = true
				}
			}
		}
		assert.True(t, foundNextWithCat)
	})
}

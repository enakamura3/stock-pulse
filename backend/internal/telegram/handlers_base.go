package telegram

import (
	"context"
	"errors"

	"github.com/onigiri/stock-pulse/backend/internal/alert"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"gopkg.in/telebot.v3"
)

type PortfolioService interface {
	GetPortfolios(ctx context.Context, userID string) ([]portfolio.Portfolio, error)
	GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*portfolio.Portfolio, []portfolio.Position, error)
	AddTransaction(ctx context.Context, userID string, tx *portfolio.Transaction) (*portfolio.Transaction, error)
	GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]portfolio.CalculatedDividend, error)
	GetPortfolioTransactions(ctx context.Context, portfolioID, userID string) ([]portfolio.Transaction, error)
}

type MarketService interface {
	GetQuote(ctx context.Context, ticker string) (*market.Quote, error)
	GetBenchmarks(ctx context.Context) (*market.MarketBenchmarks, error)
}

type FixedIncomeService interface {
	GetPortfolioPositions(ctx context.Context, portfolioID string) ([]fixedincome.Position, error)
	GetTreasuryPositions(ctx context.Context, portfolioID string) ([]fixedincome.TreasuryPosition, error)
}

type AlertService interface {
	CreateAlert(ctx context.Context, userID, ticker string, targetPrice float64, condition string) (*alert.Alert, error)
	GetAlerts(ctx context.Context, userID string) ([]*alert.Alert, error)
	DeleteAlert(ctx context.Context, id string, userID string) error
	ToggleAlert(ctx context.Context, id string, userID string) (string, error)
}

type Handlers struct {
	svc          Service
	portfolioSvc PortfolioService
	marketSvc    MarketService
	fiSvc        FixedIncomeService
	alertSvc     AlertService
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

func (h *Handlers) Register(bot *telebot.Bot) {
	// Add auth middleware for all routes globally, wait, telebot allows group or Use.
	// If we use bot.Use(), it applies to all. The middleware handles /start explicitly.
	bot.Use(h.AuthMiddleware)

	bot.Handle("/start", h.HandleStart)
	bot.Handle("/menu", h.HandleMenu)
	bot.Handle("/cotacao", h.HandleQuote)

	// Callback dos Inline Keyboards estáticos
	bot.Handle("\fbtn_resumo", h.HandlePortfolioSummary)
	bot.Handle("\fbtn_ativos", h.HandleAssetList)
	bot.Handle("\fbtn_proventos", h.HandleDividends)
	bot.Handle("\fbtn_history", h.HandleHistory)
	bot.Handle("\fbtn_renda_fixa", h.HandleFixedIncome)
	bot.Handle("\fbtn_divs_year", h.HandleDividendsByYear)
	bot.Handle("\fbtn_divs_month", h.HandleDividendsByMonth)
	bot.Handle("\fbtn_operacao", h.HandleLaunchOperation)
	bot.Handle("\fbtn_alerts", h.HandleAlerts)
	bot.Handle("\fbtn_alert_create", h.HandleAlertCreate)
	bot.Handle("\fbtn_alert_cond_above", h.HandleAlertConditionAbove)
	bot.Handle("\fbtn_alert_cond_below", h.HandleAlertConditionBelow)
	bot.Handle("\fbtn_cotacao", h.HandleQuoteStart)
	bot.Handle("\fbtn_change_portfolio", h.HandleChangePortfolio)
	bot.Handle("\fbtn_menu", h.HandleMenuCallback)
	bot.Handle("\fbtn_cancel_op", h.HandleCancelOperation)

	bot.Handle("\fbtn_new_asset", h.HandleNewAsset)
	bot.Handle("\fbtn_buy", h.HandleSetTypeBuy)
	bot.Handle("\fbtn_sell", h.HandleSetTypeSell)

	// Intercepta todos os callbacks para capturar a seleção dinâmica de ticker e portfólio
	bot.Handle(telebot.OnCallback, h.HandleDynamicCallback)

	// Intercepta todas as mensagens de texto para a máquina de estados
	bot.Handle(telebot.OnText, h.HandleText)
}

// getUserID extrai o ID do usuário autenticado do contexto telebot com segurança contra nil ou types incorretos.
func (h *Handlers) getUserID(c telebot.Context) (string, error) {
	val := c.Get("user_id")
	if val == nil {
		if c.Callback() != nil {
			_ = c.Respond()
			_ = c.Edit("⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.")
		} else {
			_ = c.Send("⚠️ Sessão não encontrada ou expirada. Por favor, envie /start para reconectar.")
		}
		return "", errors.New("sessão não autenticada")
	}

	userIDStr, ok := val.(string)
	if !ok || userIDStr == "" {
		if c.Callback() != nil {
			_ = c.Respond()
			_ = c.Edit("⚠️ Sessão inválida. Por favor, envie /start para reconectar.")
		} else {
			_ = c.Send("⚠️ Sessão inválida. Por favor, envie /start para reconectar.")
		}
		return "", errors.New("user_id inválido")
	}

	return userIDStr, nil
}

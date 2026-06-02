package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"gopkg.in/telebot.v3"
)

type PortfolioService interface {
	GetPortfolios(ctx context.Context, userID string) ([]portfolio.Portfolio, error)
	GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*portfolio.Portfolio, []portfolio.Position, error)
}

type Handlers struct {
	svc          Service
	portfolioSvc PortfolioService
}

func NewHandlers(svc Service, pSvc PortfolioService) *Handlers {
	return &Handlers{
		svc:          svc,
		portfolioSvc: pSvc,
	}
}

func (h *Handlers) Register(bot *telebot.Bot) {
	bot.Handle("/start", h.HandleStart)
	bot.Handle("/menu", h.HandleMenu)
	
	// Callback dos Inline Keyboards
	bot.Handle("\fbtn_resumo", h.HandlePortfolioSummary)
	bot.Handle("\fbtn_operacao", h.HandleLaunchOperation)
}

func (h *Handlers) HandleStart(c telebot.Context) error {
	args := c.Args()
	if len(args) > 0 {
		token := args[0]
		err := h.svc.LinkAccountWithToken(context.Background(), token, c.Chat().ID)
		if err != nil {
			if strings.Contains(err.Error(), "inválido ou expirado") {
				return c.Send("❌ O link de vinculação é inválido ou expirou. Gere um novo no Stock Pulse.")
			}
			slog.Error("Erro ao vincular conta telegram", "error", err, "chat_id", c.Chat().ID)
			return c.Send("❌ Ocorreu um erro interno ao vincular sua conta. Tente novamente.")
		}
		return c.Send("✅ Conta vinculada com sucesso! Bem-vindo ao Stock Pulse.\n\nEnvie /menu para ver as opções.")
	}

	return c.Send("Bem-vindo ao bot do Stock Pulse! Para usar este bot, vá até as configurações no sistema web e clique em 'Vincular Telegram'.")
}

func (h *Handlers) HandleMenu(c telebot.Context) error {
	// Verifica se a conta está vinculada
	_, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada. Gere um link no painel do Stock Pulse.")
	}

	menu := &telebot.ReplyMarkup{}
	btnResumo := menu.Data("📊 Resumo da Carteira", "btn_resumo")
	btnOperacao := menu.Data("💵 Lançar Operação", "btn_operacao")

	menu.Inline(
		menu.Row(btnResumo),
		menu.Row(btnOperacao),
	)

	return c.Send("Escolha uma opção:", menu)
}

func (h *Handlers) HandlePortfolioSummary(c telebot.Context) error {
	userID, err := h.svc.GetUserIDByChatID(context.Background(), c.Chat().ID)
	if err != nil {
		return c.Send("⚠️ Sua conta não está vinculada.")
	}

	userIDStr := userID.String()
	portfolios, err := h.portfolioSvc.GetPortfolios(context.Background(), userIDStr)
	if err != nil || len(portfolios) == 0 {
		return c.Send("⚠️ Nenhuma carteira encontrada na sua conta.")
	}

	portfolioID := portfolios[0].ID
	_, positions, err := h.portfolioSvc.GetPortfolioDetails(context.Background(), portfolioID, userIDStr)
	if err != nil {
		slog.Error("Failed to fetch portfolio for telegram bot", "error", err, "user_id", userIDStr)
		return c.Send("❌ Ocorreu um erro ao buscar sua carteira.")
	}

	var totalValue, totalCost, totalDailyChange float64
	for _, pos := range positions {
		totalValue += pos.CurrentValue
		totalCost += pos.TotalCost
		// Simplificação: Assumimos que o CurrentValue embute a taxa de câmbio. 
		// O DailyChange é na moeda original. Pra simplificar, vamos assumir que pos.DailyChange * pos.Quantity * taxa não temos a taxa exposta aqui na struct Position facilmente, mas podemos inferir ou apenas não somar.
		// Vamos estimar baseando em CurrentValue / CurrentPrice para a taxa
		rate := 1.0
		if pos.CurrentPrice > 0 && pos.Quantity > 0 {
			rate = pos.CurrentValue / (pos.CurrentPrice * pos.Quantity)
		}
		totalDailyChange += pos.DailyChange * pos.Quantity * rate
	}

	totalProfitLoss := totalValue - totalCost
	totalReturnPercent := 0.0
	if totalCost > 0 {
		totalReturnPercent = (totalProfitLoss / totalCost) * 100
	}

	msg := fmt.Sprintf("📊 *Resumo da sua Carteira*\n\n")
	msg += fmt.Sprintf("💰 Valor Total: *%.2f BRL*\n", totalValue)
	msg += fmt.Sprintf("📈 Variação Total Diária: *%.2f BRL*\n", totalDailyChange)
	
	var lucroPrejuizo string
	if totalProfitLoss >= 0 {
		lucroPrejuizo = fmt.Sprintf("🟢 +%.2f BRL (%.2f%%)", totalProfitLoss, totalReturnPercent)
	} else {
		lucroPrejuizo = fmt.Sprintf("🔴 %.2f BRL (%.2f%%)", totalProfitLoss, totalReturnPercent)
	}
	msg += fmt.Sprintf("⚖️ Lucro/Prejuízo Total: %s\n", lucroPrejuizo)

	// Acknowledge the callback to remove the loading state on the button
	c.Respond()
	
	return c.Send(msg, telebot.ModeMarkdown)
}

func (h *Handlers) HandleLaunchOperation(c telebot.Context) error {
	// Acknowledge the callback
	c.Respond()
	return c.Send("⏳ A funcionalidade de Lançar Operação pelo Telegram ainda está em desenvolvimento. Por favor, utilize o painel web!")
}

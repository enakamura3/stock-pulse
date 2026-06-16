package alert

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
)

// TelegramProvider define as operações necessárias para envio de alertas.
type TelegramProvider interface {
	SendAlertMessage(chatID int64, userName, ticker, assetName string, currentVal, targetVal float64, condition, currency string) error
}

// AlertWorker gerencia o monitoramento periódico de alertas de preço ativos.
type AlertWorker struct {
	repo          AlertRepository
	marketService market.QuoteProvider
	tgService     TelegramProvider
	interval      time.Duration
}

// NewAlertWorker inicializa o Worker com intervalo customizável (Padrão: 1 minuto).
func NewAlertWorker(repo AlertRepository, marketService market.QuoteProvider, tgService TelegramProvider) *AlertWorker {
	intervalStr := os.Getenv("ALERT_CHECK_INTERVAL")
	interval := 1 * time.Minute // Valor padrão aprovado (Opção 1A)

	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil {
			interval = d
		}
	}

	return &AlertWorker{
		repo:          repo,
		marketService: marketService,
		tgService:     tgService,
		interval:      interval,
	}
}

func (w *AlertWorker) Interval() time.Duration {
	return w.interval
}

// CheckActiveAlerts processa a lista de alertas ativos e avalia as condições de preço de mercado.
func (w *AlertWorker) CheckActiveAlerts(ctx context.Context) {
	// 1. Busca todos os alertas ativos globalmente no banco
	alerts, err := w.repo.GetActiveAlerts(ctx)
	if err != nil {
		slog.Error("Erro ao buscar alertas ativos do banco de dados", "error", err)
		return
	}

	if len(alerts) == 0 {
		return
	}

	slog.Info("Verificando alertas ativos em background", "count", len(alerts))

	// 2. Para cada alerta, verifica o preço atual do ativo
	for _, a := range alerts {
		// Busca a cotação (respeita o cache Redis de 60s conforme aprovado na Opção 2B)
		quote, err := w.marketService.GetQuote(ctx, a.Ticker)
		if err != nil {
			slog.Warn("Falha ao obter cotação para o ticker do alerta", "ticker", a.Ticker, "error", err)
			continue
		}

		// 3. Avalia as regras de acionamento
		triggered := false
		if a.Condition == "ABOVE" && quote.Price >= a.TargetPrice {
			triggered = true
		} else if a.Condition == "BELOW" && quote.Price <= a.TargetPrice {
			triggered = true
		}

		// 4. Se disparado, atualiza o status do banco e envia o e-mail
		if triggered {
			slog.Warn("ALERTA DE PREÇO DISPARADO!", "ticker", a.Ticker, "target", a.TargetPrice, "current", quote.Price, "condition", a.Condition)

			// Grava o disparo no banco de dados primeiro.
			// Como o banco bloqueia status != ACTIVE em MarkAlertTriggered, garantimos de forma concorrente
			// que o e-mail será disparado UMA única vez por alerta.
			err = w.repo.MarkAlertTriggered(ctx, a.ID)
			if err != nil {
				// Outro worker ou processo já marcou o alerta como disparado
				slog.Warn("Alerta já havia sido disparado ou desativado por concorrência", "id", a.ID, "ticker", a.Ticker)
				continue
			}

			// Dispara a mensagem do telegram de forma assíncrona
			go func(aAlert *Alert, currentVal float64, currency string) {
				if aAlert.TelegramChatID == nil {
					slog.Info("Alerta disparado mas o usuário não possui Telegram vinculado", "user", aAlert.UserName, "ticker", aAlert.Ticker)
					return
				}
				tgErr := w.tgService.SendAlertMessage(
					*aAlert.TelegramChatID,
					aAlert.UserName,
					aAlert.Ticker,
					aAlert.AssetName,
					currentVal,
					aAlert.TargetPrice,
					aAlert.Condition,
					currency,
				)
				if tgErr != nil {
					slog.Error("Erro ao disparar mensagem de telegram de alerta", "user", aAlert.UserName, "chat_id", *aAlert.TelegramChatID, "ticker", aAlert.Ticker, "error", tgErr)
				}
			}(a, quote.Price, quote.Currency)
		}
	}
}

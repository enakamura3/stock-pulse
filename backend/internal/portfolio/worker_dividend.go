package portfolio

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/calculator"
	"github.com/redis/go-redis/v9"
)

// DividendNotificationItem representa um provento pago que deve ser notificado ao usuário.
type DividendNotificationItem struct {
	AssetID        string    `json:"asset_id"`
	Ticker         string    `json:"ticker"`
	AssetName      string    `json:"asset_name"`
	Type           string    `json:"type"`
	Quantity       float64   `json:"quantity"`
	PerShareAmount float64   `json:"per_share_amount"`
	GrossAmount    float64   `json:"gross_amount"`
	NetAmount      float64   `json:"net_amount"`
	Currency       string    `json:"currency"`
	PaymentDate    time.Time `json:"payment_date"`
}

// DividendTelegramNotifier envia alertas de proventos para usuários vinculados no Telegram.
type DividendTelegramNotifier interface {
	SendDividendPaymentAlert(chatID int64, userName, portfolioName string, items []DividendNotificationItem) error
}

// DividendCalculator calcula proventos apurados de uma carteira.
type DividendCalculator interface {
	GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]CalculatedDividend, error)
}

// DividendDeduplicator evita o reenvio de notificações repetidas no mesmo dia.
type DividendDeduplicator interface {
	IsDividendNotified(ctx context.Context, key string) (bool, error)
	MarkDividendNotified(ctx context.Context, key string, ttl time.Duration) error
}

// RedisDividendDeduplicator implementa DividendDeduplicator usando Redis.
type RedisDividendDeduplicator struct {
	rdb redis.Cmdable
}

// NewRedisDividendDeduplicator cria uma nova instância de RedisDividendDeduplicator.
func NewRedisDividendDeduplicator(rdb redis.Cmdable) *RedisDividendDeduplicator {
	return &RedisDividendDeduplicator{rdb: rdb}
}

func (d *RedisDividendDeduplicator) IsDividendNotified(ctx context.Context, key string) (bool, error) {
	if d == nil || d.rdb == nil {
		return false, nil
	}
	val, err := d.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

func (d *RedisDividendDeduplicator) MarkDividendNotified(ctx context.Context, key string, ttl time.Duration) error {
	if d == nil || d.rdb == nil {
		return nil
	}
	return d.rdb.Set(ctx, key, "1", ttl).Err()
}

type DividendWorker struct {
	repo          PortfolioRepository
	marketService MarketService
	calc          DividendCalculator
	notifier      DividendTelegramNotifier
	dedup         DividendDeduplicator
}

func NewDividendWorker(repo PortfolioRepository, ms MarketService) *DividendWorker {
	return &DividendWorker{
		repo:          repo,
		marketService: ms,
	}
}

// WithNotifier configura o serviço de cálculo, notificador do Telegram e deduplicador opcional.
func (w *DividendWorker) WithNotifier(calc DividendCalculator, notifier DividendTelegramNotifier, dedup ...DividendDeduplicator) *DividendWorker {
	w.calc = calc
	w.notifier = notifier
	if len(dedup) > 0 {
		w.dedup = dedup[0]
	}
	return w
}

func (w *DividendWorker) SyncAllDividends(ctx context.Context) {
	assets, err := w.repo.GetAllAssets(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "erro ao buscar ativos para sincronização de dividendos", slog.Any("error", err))
		return
	}

	slog.InfoContext(ctx, "iniciando sincronização de dividendos", slog.Int("assets_count", len(assets)))

	for _, asset := range assets {
		if asset.Ticker == "" {
			continue
		}

		assetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		events, err := w.marketService.GetDividends(assetCtx, asset.Ticker, asset.AssetType)
		if err != nil {
			slog.WarnContext(assetCtx, "erro ao buscar dividendos para ativo", slog.String("ticker", asset.Ticker), slog.Any("error", err))
			cancel()
			continue
		}

		var successCount int
		for i, ev := range events {
			existingEvents, err := w.repo.GetAssetEventsByDate(assetCtx, asset.ID, ev.Date)
			if err != nil {
				slog.ErrorContext(assetCtx, "erro ao buscar dividendos existentes", slog.String("ticker", asset.Ticker), slog.Time("date", ev.Date), slog.Any("error", err))
				continue
			}

			var bestMatch *AssetEvent
			var minDiff float64 = -1

			for j := range existingEvents {
				existing := &existingEvents[j]
				if existing.Type != ev.Type {
					continue
				}

				diff := math.Abs(existing.GrossAmount - ev.Amount)
				if diff <= calculator.FuzzyMatchGrossAmountThreshold {
					if minDiff == -1 || diff < minDiff {
						minDiff = diff
						bestMatch = existing
					}
				}
			}

			if bestMatch != nil {
				if minDiff < calculator.FinancialEpsilon && bestMatch.PaymentDate.Equal(ev.PaymentDate) {
					successCount++
					continue
				}

				// Update existing
				err = w.repo.UpdateAssetEventValueByID(assetCtx, bestMatch.ID, ev.Amount, ev.Amount, ev.PaymentDate)
				if err != nil {
					slog.ErrorContext(assetCtx, "erro ao atualizar dividendo (Fuzzy Match)",
						slog.Int("current", i+1),
						slog.Int("total", len(events)),
						slog.String("id", bestMatch.ID),
						slog.String("ticker", asset.Ticker),
						slog.Any("error", err),
					)
				} else {
					successCount++
				}
			} else {
				// Insert new
				err = w.repo.UpsertAssetEvent(assetCtx, AssetEvent{
					AssetID:     asset.ID,
					Type:        ev.Type,
					GrossAmount: ev.Amount,
					NetAmount:   ev.Amount, // We store gross in both places, taxes are applied per-portfolio later
					CumDate:     ev.Date,
					PaymentDate: ev.PaymentDate,
				})
				if err != nil {
					slog.ErrorContext(assetCtx, "erro ao salvar novo dividendo",
						slog.Int("current", i+1),
						slog.Int("total", len(events)),
						slog.String("cum_date", ev.Date.Format("2006-01-02")),
						slog.String("type", ev.Type),
						slog.Float64("amount", ev.Amount),
						slog.String("ticker", asset.Ticker),
						slog.Any("error", err),
					)
				} else {
					successCount++
				}
			}
		}

		if successCount > 0 {
			slog.InfoContext(assetCtx, "proventos sincronizados para ativo", slog.Int("count", successCount), slog.String("ticker", asset.Ticker))
		}

		cancel()

		// Small sleep to avoid hammering the scrapers
		time.Sleep(2 * time.Second)
	}

	slog.InfoContext(ctx, "sincronização de dividendos finalizada")

	// Dispara notificação de proventos pagos hoje ("Caiu na conta!") para usuários com Telegram
	w.NotifyPaidDividends(ctx)
}

// NotifyPaidDividends verifica se há proventos pagos hoje para as carteiras vinculadas ao Telegram e notifica os usuários.
func (w *DividendWorker) NotifyPaidDividends(ctx context.Context) {
	if w.notifier == nil || w.calc == nil {
		return
	}

	linkedPortfolios, err := w.repo.GetTelegramLinkedPortfolios(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "erro ao buscar carteiras vinculadas ao Telegram para alerta de proventos", slog.Any("error", err))
		return
	}

	if len(linkedPortfolios) == 0 {
		return
	}

	loc, err := loadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("BRT", -3*60*60)
	}
	today := time.Now().In(loc)
	todayStr := today.Format("2006-01-02")

	for _, lp := range linkedPortfolios {
		divs, err := w.calc.GetPortfolioDividends(ctx, lp.PortfolioID, lp.UserID)
		if err != nil {
			slog.WarnContext(ctx, "erro ao calcular proventos da carteira para alerta",
				slog.String("portfolio_id", lp.PortfolioID),
				slog.String("user_id", lp.UserID),
				slog.Any("error", err),
			)
			continue
		}

		var dueItems []DividendNotificationItem
		for _, d := range divs {
			if d.PaymentDate.IsZero() {
				continue
			}
			payDate := d.PaymentDate.In(loc)
			if payDate.Format("2006-01-02") != todayStr {
				continue
			}

			// Regra 1: Comparação de floats monetários com tolerância epsilon
			if d.Quantity <= 1e-6 || d.NetAmount <= 1e-6 {
				continue
			}

			dedupKey := fmt.Sprintf("notified_dividend:%s:%s:%s:%s:%s",
				lp.UserID, lp.PortfolioID, d.AssetID, d.Type, todayStr,
			)

			if w.dedup != nil {
				alreadyNotified, err := w.dedup.IsDividendNotified(ctx, dedupKey)
				if err == nil && alreadyNotified {
					continue
				}
			}

			dueItems = append(dueItems, DividendNotificationItem{
				AssetID:        d.AssetID,
				Ticker:         d.Ticker,
				AssetName:      d.AssetName,
				Type:           d.Type,
				Quantity:       d.Quantity,
				PerShareAmount: d.PerShareAmount,
				GrossAmount:    d.GrossAmount,
				NetAmount:      d.NetAmount,
				Currency:       d.Currency,
				PaymentDate:    d.PaymentDate,
			})
		}

		if len(dueItems) == 0 {
			continue
		}

		err = w.notifier.SendDividendPaymentAlert(lp.TelegramChatID, lp.UserName, lp.PortfolioName, dueItems)
		if err != nil {
			slog.ErrorContext(ctx, "erro ao enviar notificação de proventos via Telegram",
				slog.Int64("chat_id", lp.TelegramChatID),
				slog.String("user", lp.UserName),
				slog.String("portfolio", lp.PortfolioName),
				slog.Any("error", err),
			)
			continue
		}

		if w.dedup != nil {
			for _, item := range dueItems {
				dedupKey := fmt.Sprintf("notified_dividend:%s:%s:%s:%s:%s",
					lp.UserID, lp.PortfolioID, item.AssetID, item.Type, todayStr,
				)
				_ = w.dedup.MarkDividendNotified(ctx, dedupKey, 48*time.Hour)
			}
		}

		slog.InfoContext(ctx, "notificação de proventos enviada com sucesso via Telegram",
			slog.Int64("chat_id", lp.TelegramChatID),
			slog.String("user", lp.UserName),
			slog.Int("items_count", len(dueItems)),
		)
	}
}

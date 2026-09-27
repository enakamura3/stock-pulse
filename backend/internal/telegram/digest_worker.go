package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/redis/go-redis/v9"
)

// DigestTelegramNotifier define o contrato para envio do daily digest no Telegram.
type DigestTelegramNotifier interface {
	SendDailyDigest(chatID int64, userName, portfolioName, currency string, totalValue, dailyChange, returnPercent float64, todayDividends, upcomingDividends []portfolio.CalculatedDividend) error
}

// DailyDigestWorker gerencia o disparo do resumo matinal diário de investimentos.
type DailyDigestWorker struct {
	repo         Repository
	svc          Service
	portfolioSvc PortfolioService
	fiSvc        FixedIncomeService
	notifier     DigestTelegramNotifier
	rdb          redis.Cmdable
	targetHour   int
	targetMinute int
	nowFunc      func() time.Time
	locFunc      func(name string) (*time.Location, error)
}

// NewDailyDigestWorker instancia o worker de resumo diário matinal.
func NewDailyDigestWorker(repo Repository, svc Service, pSvc PortfolioService, fiSvc FixedIncomeService, notifier DigestTelegramNotifier, rdb redis.Cmdable) *DailyDigestWorker {
	return &DailyDigestWorker{
		repo:         repo,
		svc:          svc,
		portfolioSvc: pSvc,
		fiSvc:        fiSvc,
		notifier:     notifier,
		rdb:          rdb,
		targetHour:   8,
		targetMinute: 30,
		nowFunc:      time.Now,
		locFunc:      time.LoadLocation,
	}
}

// ProcessDailyDigests avalia todas as contas vinculadas e envia o resumo matinal.
func (w *DailyDigestWorker) ProcessDailyDigests(ctx context.Context) {
	if w.notifier == nil || w.portfolioSvc == nil || w.repo == nil {
		return
	}

	locFunc := w.locFunc
	if locFunc == nil {
		locFunc = time.LoadLocation
	}

	loc, err := locFunc("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("BRT", -3*60*60)
	}

	now := w.nowFunc().In(loc)
	todayStr := now.Format("2006-01-02")

	// Verifica se já atingiu o horário alvo da manhã (padrão: 08:30)
	if now.Hour() < w.targetHour || (now.Hour() == w.targetHour && now.Minute() < w.targetMinute) {
		return
	}

	users, err := w.repo.GetLinkedUsers(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "erro ao buscar usuários vinculados ao Telegram para daily digest", slog.Any("error", err))
		return
	}

	if len(users) == 0 {
		return
	}

	for _, u := range users {
		dedupKey := fmt.Sprintf("daily_digest_sent:%s:%s", u.UserID.String(), todayStr)
		if w.rdb != nil {
			val, err := w.rdb.Get(ctx, dedupKey).Result()
			if err == nil && val == "1" {
				continue // Já enviado hoje para este usuário
			}
		}

		portfolios, err := w.portfolioSvc.GetPortfolios(ctx, u.UserID.String())
		if err != nil || len(portfolios) == 0 {
			continue
		}

		portfolioID, portfolioName := resolveActivePortfolio(ctx, w.svc, u.TelegramChatID, portfolios)
		pDetails, positions, err := w.portfolioSvc.GetPortfolioDetails(ctx, portfolioID, u.UserID.String())
		if err != nil {
			slog.WarnContext(ctx, "erro ao buscar detalhes da carteira para daily digest",
				slog.String("portfolio_id", portfolioID),
				slog.String("user_id", u.UserID.String()),
				slog.Any("error", err),
			)
			continue
		}

		var totalValue, totalCost, totalDailyChange float64
		for _, pos := range positions {
			totalValue += pos.CurrentValue
			totalCost += pos.TotalCost
			rate := 1.0
			if pos.CurrentPrice > 1e-6 && pos.Quantity > 1e-6 {
				rate = pos.CurrentValue / (pos.CurrentPrice * pos.Quantity)
			}
			totalDailyChange += pos.DailyChange * pos.Quantity * rate
		}

		if w.fiSvc != nil {
			fiPositions, err := w.fiSvc.GetPortfolioPositions(ctx, portfolioID)
			if err == nil {
				for _, fi := range fiPositions {
					totalValue += fi.NetValue
					totalCost += fi.TotalInvested
				}
			}
		}

		var returnPercent float64
		if totalCost > 1e-6 {
			returnPercent = ((totalValue - totalCost) / totalCost) * 100.0
		}

		currency := "BRL"
		if pDetails != nil && pDetails.BaseCurrency != "" {
			currency = pDetails.BaseCurrency
		}

		var todayDividends []portfolio.CalculatedDividend
		var upcomingDividends []portfolio.CalculatedDividend
		divs, err := w.portfolioSvc.GetPortfolioDividends(ctx, portfolioID, u.UserID.String())
		if err == nil {
			sevenDaysLater := now.AddDate(0, 0, 7)
			for _, d := range divs {
				if d.PaymentDate.IsZero() {
					continue
				}
				pDate := d.PaymentDate.In(loc)
				pDateStr := pDate.Format("2006-01-02")
				if pDateStr == todayStr {
					todayDividends = append(todayDividends, d)
				} else if pDate.After(now) && !pDate.After(sevenDaysLater) {
					upcomingDividends = append(upcomingDividends, d)
				}
			}
		}

		err = w.notifier.SendDailyDigest(
			u.TelegramChatID,
			u.UserName,
			portfolioName,
			currency,
			totalValue,
			totalDailyChange,
			returnPercent,
			todayDividends,
			upcomingDividends,
		)
		if err != nil {
			slog.ErrorContext(ctx, "erro ao enviar daily digest via Telegram",
				slog.Int64("chat_id", u.TelegramChatID),
				slog.String("user", u.UserName),
				slog.Any("error", err),
			)
			continue
		}

		if w.rdb != nil {
			_ = w.rdb.Set(ctx, dedupKey, "1", 48*time.Hour).Err()
		}

		slog.InfoContext(ctx, "daily digest enviado com sucesso via Telegram",
			slog.Int64("chat_id", u.TelegramChatID),
			slog.String("user", u.UserName),
		)
	}
}

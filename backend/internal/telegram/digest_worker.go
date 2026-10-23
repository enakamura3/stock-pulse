package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/redis/go-redis/v9"
)

// DailyMover representa um ativo com variação intradiária relevante no pregão.
type DailyMover struct {
	Ticker             string
	DailyChangePercent float64
	Currency           string
	OpenPrice          float64
	CurrentPrice       float64
}

// DigestTelegramNotifier define o contrato para envio do daily digest no Telegram.
type DigestTelegramNotifier interface {
	SendDailyDigest(
		chatID int64,
		userName, portfolioName, currency string,
		totalValue, dailyChange, dailyChangePercent, returnPercent float64,
		topGainers, topLosers []DailyMover,
		todayDividends, upcomingDividends []portfolio.CalculatedDividend,
		extraAlerts ...string,
	) error
}

// DailyDigestWorker gerencia o disparo do resumo de fechamento de mercado de investimentos.
type DailyDigestWorker struct {
	repo         Repository
	svc          Service
	portfolioSvc PortfolioService
	fiSvc        FixedIncomeService
	notifier     DigestTelegramNotifier
	rdb          redis.Cmdable
	targetHour   int
	targetMinute int
	pacingDelay  time.Duration
	nowFunc      func() time.Time
	locFunc      func(name string) (*time.Location, error)
}

// NewDailyDigestWorker instancia o worker de resumo diário de fechamento de mercado.
func NewDailyDigestWorker(repo Repository, svc Service, pSvc PortfolioService, fiSvc FixedIncomeService, notifier DigestTelegramNotifier, rdb redis.Cmdable) *DailyDigestWorker {
	hour := 19
	minute := 0
	if hStr := os.Getenv("TELEGRAM_DIGEST_HOUR"); hStr != "" {
		if h, err := strconv.Atoi(hStr); err == nil && h >= 0 && h <= 23 {
			hour = h
		}
	}
	if mStr := os.Getenv("TELEGRAM_DIGEST_MINUTE"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m >= 0 && m <= 59 {
			minute = m
		}
	}

	pacingDelay := 50 * time.Millisecond
	if pStr := os.Getenv("TELEGRAM_DIGEST_PACING_MS"); pStr != "" {
		if ms, err := strconv.Atoi(pStr); err == nil && ms >= 0 {
			pacingDelay = time.Duration(ms) * time.Millisecond
		}
	}

	return &DailyDigestWorker{
		repo:         repo,
		svc:          svc,
		portfolioSvc: pSvc,
		fiSvc:        fiSvc,
		notifier:     notifier,
		rdb:          rdb,
		targetHour:   hour,
		targetMinute: minute,
		pacingDelay:  pacingDelay,
		nowFunc:      time.Now,
		locFunc:      time.LoadLocation,
	}
}

// ProcessDailyDigests avalia todas as contas vinculadas e envia o resumo de fechamento de mercado.
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

	// Não envia aos fins de semana (sábado e domingo), pois a B3 não opera
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return
	}

	// Não envia em feriados nacionais da B3/ANBIMA
	if w.fiSvc != nil {
		holidays, err := w.fiSvc.GetAnbimaHolidays(ctx)
		if err == nil && holidays != nil && holidays[todayStr] {
			slog.InfoContext(ctx, "daily digest suprimido: feriado B3/ANBIMA", slog.String("date", todayStr))
			return
		}
	}

	// Verifica se já atingiu o horário alvo de fechamento (padrão: 19:00)
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
		userDedupKey := fmt.Sprintf("daily_digest_sent:%s:%s", u.UserID.String(), todayStr)
		if w.rdb != nil {
			val, err := w.rdb.Get(ctx, userDedupKey).Result()
			if err == nil && val == "1" {
				continue // Já enviado hoje para este usuário
			}
		}

		portfolios, err := w.portfolioSvc.GetPortfolios(ctx, u.UserID.String())
		if err != nil || len(portfolios) == 0 {
			continue
		}

		activeID, _ := resolveActivePortfolio(ctx, w.svc, u.TelegramChatID, portfolios)
		if activeID != "" && len(portfolios) > 1 {
			sort.SliceStable(portfolios, func(i, j int) bool {
				return portfolios[i].ID == activeID
			})
		}

		allPortfoliosSent := true
		for _, port := range portfolios {
			portfolioID := port.ID
			portfolioName := port.Name
			if portfolioName == "" {
				portfolioName = "Principal"
			}

			dedupKey := fmt.Sprintf("daily_digest_sent:%s:%s:%s", u.UserID.String(), portfolioID, todayStr)
			if w.rdb != nil {
				val, err := w.rdb.Get(ctx, dedupKey).Result()
				if err == nil && val == "1" {
					continue
				}
			}

			pDetails, positions, err := w.portfolioSvc.GetPortfolioDetails(ctx, portfolioID, u.UserID.String())
			if err != nil {
				slog.WarnContext(ctx, "erro ao buscar detalhes da carteira para daily digest",
					slog.String("portfolio_id", portfolioID),
					slog.String("user_id", u.UserID.String()),
					slog.Any("error", err),
				)
				allPortfoliosSent = false
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

			var extraAlerts []string
			if w.fiSvc != nil {
				fiPositions, err := w.fiSvc.GetPortfolioPositions(ctx, portfolioID)
				if err == nil {
					for _, fi := range fiPositions {
						totalValue += fi.NetValue
						totalCost += fi.TotalInvested
						if fi.DaysToMaturity >= 0 && fi.DaysToMaturity <= 7 && !fi.IsMatured && (fi.Asset.Institution != "" || fi.Asset.Type != "") {
							fiName := strings.TrimSpace(fi.Asset.Institution + " " + fi.Asset.Type)
							extraAlerts = append(extraAlerts, fmt.Sprintf("%s vencendo em %d dia(s)", fiName, fi.DaysToMaturity))
						}
					}
				}

				treasuryPositions, err := w.fiSvc.GetTreasuryPositions(ctx, portfolioID)
				if err == nil {
					for _, tp := range treasuryPositions {
						totalValue += tp.NetValue
						totalCost += tp.TotalInvested
						if tp.TreasuryType != "" && tp.DaysToMaturity >= 0 && tp.DaysToMaturity <= 7 && !tp.IsMatured {
							extraAlerts = append(extraAlerts, fmt.Sprintf("Tesouro %s vencendo em %d dia(s)", tp.TreasuryType, tp.DaysToMaturity))
						}
					}
				}
			}

			var returnPercent float64
			if totalCost > 1e-6 {
				returnPercent = ((totalValue - totalCost) / totalCost) * 100.0
			}

			var dailyChangePercent float64
			prevValue := totalValue - totalDailyChange
			if prevValue > 1e-6 {
				dailyChangePercent = (totalDailyChange / prevValue) * 100.0
			}

			currency := "BRL"
			if pDetails != nil && pDetails.BaseCurrency != "" {
				currency = pDetails.BaseCurrency
			}

			var gainers []DailyMover
			var losers []DailyMover
			for _, pos := range positions {
				if pos.Quantity > 1e-6 {
					posCurr := pos.Currency
					if posCurr == "" {
						posCurr = currency
					}
					mover := DailyMover{
						Ticker:             pos.Ticker,
						DailyChangePercent: pos.DailyChangePercent,
						Currency:           posCurr,
						OpenPrice:          pos.OpenPrice,
						CurrentPrice:       pos.CurrentPrice,
					}
					if pos.DailyChangePercent > 1e-6 {
						gainers = append(gainers, mover)
					} else if pos.DailyChangePercent < -1e-6 {
						losers = append(losers, mover)
					}
				}
			}

			sort.Slice(gainers, func(i, j int) bool {
				return gainers[i].DailyChangePercent > gainers[j].DailyChangePercent
			})
			if len(gainers) > 3 {
				gainers = gainers[:3]
			}

			sort.Slice(losers, func(i, j int) bool {
				return losers[i].DailyChangePercent < losers[j].DailyChangePercent
			})
			if len(losers) > 3 {
				losers = losers[:3]
			}

			var todayDividends []portfolio.CalculatedDividend
			var upcomingDividends []portfolio.CalculatedDividend
			divs, err := w.portfolioSvc.GetPortfolioDividends(ctx, portfolioID, u.UserID.String())
			if err == nil {
				sevenDaysLaterStr := now.AddDate(0, 0, 7).Format("2006-01-02")
				for _, d := range divs {
					if d.PaymentDate.IsZero() {
						continue
					}
					pDateStr := d.PaymentDate.Format("2006-01-02")
					if pDateStr == todayStr {
						todayDividends = append(todayDividends, d)
					} else if pDateStr > todayStr && pDateStr <= sevenDaysLaterStr {
						upcomingDividends = append(upcomingDividends, d)
					}
					if !d.CumDate.IsZero() && d.CumDate.Format("2006-01-02") == todayStr {
						extraAlerts = append(extraAlerts, fmt.Sprintf("Data Com hoje para %s (%s)", d.Ticker, d.Type))
					}
				}
			}

			// Se houver múltiplas carteiras e esta carteira secundária estiver vazia, não poluir o chat do usuário
			if len(portfolios) > 1 && len(positions) == 0 && totalValue < 1e-6 && len(todayDividends) == 0 && len(upcomingDividends) == 0 && len(extraAlerts) == 0 {
				continue
			}

			err = w.notifier.SendDailyDigest(
				u.TelegramChatID,
				u.UserName,
				portfolioName,
				currency,
				totalValue,
				totalDailyChange,
				dailyChangePercent,
				returnPercent,
				gainers,
				losers,
				todayDividends,
				upcomingDividends,
				extraAlerts...,
			)
			if err != nil {
				allPortfoliosSent = false
				slog.ErrorContext(ctx, "erro ao enviar daily digest via Telegram",
					slog.Int64("chat_id", u.TelegramChatID),
					slog.String("user", u.UserName),
					slog.String("portfolio", portfolioName),
					slog.Any("error", err),
				)
				if isBlockedByUser(err) {
					slog.WarnContext(ctx, "usuário bloqueou o bot no Telegram; gravando dedup para suprimir retentativas",
						slog.Int64("chat_id", u.TelegramChatID),
						slog.String("user", u.UserName),
					)
					if w.rdb != nil {
						_ = w.rdb.Set(ctx, dedupKey, "1", 48*time.Hour).Err()
						_ = w.rdb.Set(ctx, userDedupKey, "1", 48*time.Hour).Err()
					}
					break
				}
				continue
			}

			if w.rdb != nil {
				_ = w.rdb.Set(ctx, dedupKey, "1", 48*time.Hour).Err()
			}

			slog.InfoContext(ctx, "daily digest enviado com sucesso via Telegram",
				slog.Int64("chat_id", u.TelegramChatID),
				slog.String("user", u.UserName),
				slog.String("portfolio", portfolioName),
			)

			// Throttling / Pacing entre envios (F6)
			if w.pacingDelay > 0 {
				time.Sleep(w.pacingDelay)
			}
		}

		if allPortfoliosSent && w.rdb != nil {
			_ = w.rdb.Set(ctx, userDedupKey, "1", 48*time.Hour).Err()
		}
	}
}

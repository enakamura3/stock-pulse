package fixedincome

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

func countTreasuryBusinessDays(start, end time.Time, holidays map[string]bool) int {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	if !start.Before(end) {
		return 0
	}
	businessDays := 0
	curr := start
	for curr.Before(end) {
		curr = curr.AddDate(0, 0, 1)
		if curr.Weekday() != time.Saturday && curr.Weekday() != time.Sunday {
			dateStr := curr.Format("2006-01-02")
			if !holidays[dateStr] {
				businessDays++
			}
		}
	}
	return businessDays
}

func getTreasuryIOFRate(days int) float64 {
	iofRates := []float64{
		96, 93, 90, 86, 83, 80, 76, 73, 70, 66, 63, 60, 56, 53, 50, 46, 43, 40, 36, 33, 30, 26, 23, 20, 16, 13, 10, 6, 3, 0,
	}
	if days <= 0 {
		return 96.0
	}
	if days >= 30 {
		return 0.0
	}
	return iofRates[days-1]
}

func getTreasuryIRRate(days int) float64 {
	if days <= 180 {
		return 22.5
	} else if days <= 360 {
		return 20.0
	} else if days <= 720 {
		return 17.5
	}
	return 15.0
}

func (s *service) GetTreasuryPositions(ctx context.Context, portfolioID string) ([]TreasuryPosition, error) {
	lots, err := s.repo.GetActiveSubscriptionLots(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	holidays, err := s.repo.GetAnbimaHolidays(ctx)
	if err != nil {
		return nil, err
	}

	today := time.Now()

	type tempPos struct {
		p                TreasuryPosition
		holdingDays      int
		busDays          int
		accruedFeeFactor float64
		grossYield       float64
	}

	var tempPositions []tempPos
	var totalSelicGross float64

	for _, l := range lots {
		ticker, treasuryType, maturityDate, hasCoupons, err := s.repo.GetTreasuryAssetDetails(ctx, l.AssetID)
		if err != nil {
			return nil, err
		}

		var p TreasuryPosition
		p.TransactionID = l.ID
		p.AssetID = l.AssetID
		p.Ticker = ticker
		p.TreasuryType = treasuryType
		p.MaturityDate = maturityDate
		p.HasCoupons = hasCoupons
		p.StartDate = l.TransactionDate
		p.Quantity = l.Quantity
		p.UnitPrice = l.UnitPrice
		p.ContractedRate = l.ContractedRate
		p.TotalInvested = l.RemainingQuantity * l.UnitPrice

		holdingDays := int(today.Sub(l.TransactionDate).Hours() / 24)
		busDays := countTreasuryBusinessDays(l.TransactionDate, today, holidays)

		dailyRate := math.Pow(1.0+l.ContractedRate/100.0, 1.0/252.0) - 1.0
		factor := math.Pow(1.0+dailyRate, float64(busDays))
		p.GrossValue = p.TotalInvested * factor

		grossYield := p.GrossValue - p.TotalInvested
		if grossYield < 0 {
			grossYield = 0
		}

		if p.TreasuryType == "SELIC" {
			totalSelicGross += p.GrossValue
		}

		dailyB3Rate := math.Pow(1.0+0.0020, 1.0/252.0) - 1.0
		accruedFeeFactor := math.Pow(1.0+dailyB3Rate, float64(busDays)) - 1.0

		tempPositions = append(tempPositions, tempPos{
			p:                p,
			holdingDays:      holdingDays,
			busDays:          busDays,
			accruedFeeFactor: accruedFeeFactor,
			grossYield:       grossYield,
		})
	}

	var positions []TreasuryPosition
	for _, tp := range tempPositions {
		p := tp.p

		if p.TreasuryType == "SELIC" {
			if totalSelicGross > 10000.00 {
				p.B3Fee = p.GrossValue * ((totalSelicGross - 10000.00) / totalSelicGross) * tp.accruedFeeFactor
			} else {
				p.B3Fee = 0.0
			}
		} else {
			p.B3Fee = p.GrossValue * tp.accruedFeeFactor
		}

		p.IOFTax = tp.grossYield * (getTreasuryIOFRate(tp.holdingDays) / 100.0)
		p.IRTax = (tp.grossYield - p.IOFTax) * (getTreasuryIRRate(tp.holdingDays) / 100.0)
		if p.IRTax < 0 {
			p.IRTax = 0
		}

		p.Taxes = p.IOFTax + p.IRTax
		p.NetValue = p.GrossValue - p.Taxes - p.B3Fee
		p.IsMatured = today.After(p.MaturityDate) || today.Equal(p.MaturityDate)
		p.DaysToMaturity = int(p.MaturityDate.Sub(today).Hours() / 24)
		if p.DaysToMaturity < 0 {
			p.DaysToMaturity = 0
		}

		positions = append(positions, p)
	}

	if positions == nil {
		positions = []TreasuryPosition{}
	}
	return positions, nil
}

func (s *service) GetTreasuryTransactions(ctx context.Context, portfolioID string) ([]TreasuryTxRequest, error) {
	reqs, err := s.repo.GetTreasuryTransactionsList(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	if reqs == nil {
		reqs = []TreasuryTxRequest{}
	}
	return reqs, nil
}

func (s *service) CreateTreasuryTransaction(ctx context.Context, portfolioID string, req *TreasuryTxRequest) (interface{}, error) {
	maturityDate, err := time.Parse("2006-01-02", req.MaturityDate)
	if err != nil {
		return nil, fmt.Errorf("invalid maturity date: %w", err)
	}
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction date: %w", err)
	}

	var assetID string
	err = s.repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
		var err error
		assetID, err = s.repo.GetTreasuryAssetByTicker(ctx, tx, req.Ticker)
		if err == pgx.ErrNoRows {
			assetID, err = s.repo.CreateTreasuryAsset(ctx, tx, req.Ticker, req.Ticker, req.TreasuryType, maturityDate, req.HasCoupons)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if req.Type == "SUBSCRIPTION" {
		var txID string
		err = s.repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			var err error
			txID, err = s.repo.CreateTreasurySubscription(ctx, tx, portfolioID, assetID, req.Quantity, req.UnitPrice, req.ContractedRate, transactionDate)
			return err
		})
		if err != nil {
			return nil, err
		}
		return map[string]string{"id": txID, "status": "subscribed"}, nil
	}

	if req.Type == "REDEMPTION" {
		var result map[string]interface{}
		err = s.repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			lots, err := s.repo.GetActiveLotsForAsset(ctx, tx, portfolioID, assetID)
			if err != nil {
				return err
			}

			holidays, err := s.repo.GetAnbimaHolidays(ctx)
			if err != nil {
				return err
			}

			selicRates, err := s.repo.GetSelicRates(ctx)
			if err != nil {
				return err
			}

			totalSelicInvested, err := s.repo.GetTotalSelicInvested(ctx, tx, portfolioID)
			if err != nil {
				return err
			}

			remainingToRedeem := req.Quantity
			var totalGross, totalIOF, totalIR, totalB3, totalNet float64

			redemptionTxID, err := s.repo.CreateTreasuryRedemptionPlaceholder(ctx, tx, portfolioID, assetID, req.Quantity, req.UnitPrice, req.ContractedRate, transactionDate)
			if err != nil {
				return err
			}

			for _, l := range lots {
				if remainingToRedeem <= 0 {
					break
				}
				depleteQty := l.RemainingQuantity
				if depleteQty > remainingToRedeem {
					depleteQty = remainingToRedeem
				}

				holdingDays := int(transactionDate.Sub(l.TransactionDate).Hours() / 24)
				busDays := countTreasuryBusinessDays(l.TransactionDate, transactionDate, holidays)

				var valAtRedemption float64
				if req.TreasuryType == "SELIC" {
					factor := 1.0
					currDate := l.TransactionDate
					for currDate.Before(transactionDate) {
						currDate = currDate.AddDate(0, 0, 1)
						if currDate.Weekday() != time.Saturday && currDate.Weekday() != time.Sunday && !holidays[currDate.Format("2006-01-02")] {
							rate := 10.75
							if rVal, exists := selicRates[currDate.Format("2006-01-02")]; exists {
								rate = rVal
							}
							dailyRate := math.Pow(1.0+rate/100.0, 1.0/252.0) - 1.0
							dailySpread := math.Pow(1.0+l.ContractedRate/100.0, 1.0/252.0) - 1.0
							factor *= (1.0 + dailyRate + dailySpread)
						}
					}
					valAtRedemption = depleteQty * l.UnitPrice * factor
				} else {
					rate := l.ContractedRate
					dailyRate := math.Pow(1.0+rate/100.0, 1.0/252.0) - 1.0
					factor := math.Pow(1.0+dailyRate, float64(busDays))
					valAtRedemption = depleteQty * l.UnitPrice * factor
				}

				costBasis := depleteQty * l.UnitPrice
				grossYield := valAtRedemption - costBasis
				if grossYield < 0 {
					grossYield = 0
				}

				var b3Fee float64
				dailyB3Rate := math.Pow(1.0+0.0020, 1.0/252.0) - 1.0

				if req.TreasuryType == "SELIC" {
					exemptFraction := 1.0
					if totalSelicInvested > 10000.0 {
						exemptFraction = 10000.0 / totalSelicInvested
					}
					if exemptFraction > 1.0 {
						exemptFraction = 1.0
					}
					taxablePortion := 1.0 - exemptFraction
					accruedFeeFactor := math.Pow(1.0+dailyB3Rate*taxablePortion, float64(busDays)) - 1.0
					b3Fee = valAtRedemption * accruedFeeFactor
				} else {
					accruedFeeFactor := math.Pow(1.0+dailyB3Rate, float64(busDays)) - 1.0
					b3Fee = valAtRedemption * accruedFeeFactor
				}

				iofRate := getTreasuryIOFRate(holdingDays)
				iofTax := grossYield * (iofRate / 100.0)

				irRate := getTreasuryIRRate(holdingDays)
				irTax := (grossYield - iofTax) * (irRate / 100.0)
				if irTax < 0 {
					irTax = 0
				}

				netYield := grossYield - iofTax - irTax - b3Fee
				netVal := costBasis + netYield

				totalGross += valAtRedemption
				totalIOF += iofTax
				totalIR += irTax
				totalB3 += b3Fee
				totalNet += netVal

				newRemaining := l.RemainingQuantity - depleteQty
				err = s.repo.UpdateLotRemainingQuantity(ctx, tx, l.ID, newRemaining)
				if err != nil {
					return err
				}

				err = s.repo.CreateDepletionLink(ctx, tx, l.ID, redemptionTxID, depleteQty)
				if err != nil {
					return err
				}

				remainingToRedeem -= depleteQty
			}

			err = s.repo.UpdateRedemptionFinancials(ctx, tx, redemptionTxID, totalGross, totalIOF, totalIR, totalB3, totalNet)
			if err != nil {
				return err
			}

			result = map[string]interface{}{
				"id":           redemptionTxID,
				"gross_amount": totalGross,
				"iof_tax":      totalIOF,
				"ir_tax":       totalIR,
				"b3_fee":       totalB3,
				"net_amount":   totalNet,
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, fmt.Errorf("invalid transaction type: %s", req.Type)
}

func (s *service) GetTreasuryPerformance(ctx context.Context, portfolioID string) ([]TreasuryPerfPoint, error) {
	txs, err := s.repo.GetTreasuryTransactionsList(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	if len(txs) == 0 {
		return []TreasuryPerfPoint{}, nil
	}

	holidays, err := s.repo.GetAnbimaHolidays(ctx)
	if err != nil {
		holidays = make(map[string]bool)
	}

	selicRates, err := s.repo.GetSelicRates(ctx)
	if err != nil {
		selicRates = make(map[string]float64)
	}

	startDate, err := time.Parse("2006-01-02", txs[0].TransactionDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start date: %w", err)
	}

	txsByDate := make(map[string][]TreasuryTxRequest)
	for _, tx := range txs {
		txsByDate[tx.TransactionDate] = append(txsByDate[tx.TransactionDate], tx)
	}

	type activeLot struct {
		ticker         string
		treasuryType   string
		maturityDate   time.Time
		contractedRate float64
		unitPrice      float64
		quantity       float64
		grossValue     float64
	}

	var activeLots []*activeLot
	var points []TreasuryPerfPoint

	today := time.Now()
	// Normalizar data de hoje para UTC meia-noite para consistência de comparação
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	currDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	for !currDate.After(today) {
		dateStr := currDate.Format("2006-01-02")
		isBusDay := currDate.Weekday() != time.Saturday && currDate.Weekday() != time.Sunday && !holidays[dateStr]

		// 1. Rendimento diário para lotes ativos existentes
		if isBusDay {
			for _, lot := range activeLots {
				if !lot.maturityDate.IsZero() && currDate.After(lot.maturityDate) {
					continue
				}

				var dailyFactor float64
				if lot.treasuryType == "SELIC" {
					rate := 10.75
					if rVal, exists := selicRates[dateStr]; exists {
						rate = rVal
					}
					dailyRate := math.Pow(1.0+rate/100.0, 1.0/252.0) - 1.0
					dailySpread := math.Pow(1.0+lot.contractedRate/100.0, 1.0/252.0) - 1.0
					dailyFactor = 1.0 + dailyRate + dailySpread
				} else {
					dailyFactor = math.Pow(1.0+lot.contractedRate/100.0, 1.0/252.0)
				}
				lot.grossValue *= dailyFactor
			}
		}

		// 2. Processar transações do dia
		if dayTxs, exists := txsByDate[dateStr]; exists {
			for _, tx := range dayTxs {
				matDate, _ := time.Parse("2006-01-02", tx.MaturityDate)
				if tx.Type == "SUBSCRIPTION" {
					lotVal := tx.Quantity * tx.UnitPrice
					activeLots = append(activeLots, &activeLot{
						ticker:         tx.Ticker,
						treasuryType:   tx.TreasuryType,
						maturityDate:   matDate,
						contractedRate: tx.ContractedRate,
						unitPrice:      tx.UnitPrice,
						quantity:       tx.Quantity,
						grossValue:     lotVal,
					})
				} else if tx.Type == "REDEMPTION" {
					qtyToRedeem := tx.Quantity
					for i := 0; i < len(activeLots) && qtyToRedeem > 1e-6; i++ {
						lot := activeLots[i]
						if lot.ticker == tx.Ticker && lot.quantity > 1e-6 {
							if lot.quantity >= qtyToRedeem-1e-6 {
								ratio := (lot.quantity - qtyToRedeem) / lot.quantity
								lot.quantity -= qtyToRedeem
								lot.grossValue *= ratio
								qtyToRedeem = 0.0
							} else {
								qtyToRedeem -= lot.quantity
								lot.quantity = 0.0
								lot.grossValue = 0.0
							}
						}
					}
				}
			}
		}

		// Filtrar lotes zerados
		var nextActive []*activeLot
		for _, lot := range activeLots {
			if lot.quantity > 1e-6 {
				nextActive = append(nextActive, lot)
			}
		}
		activeLots = nextActive

		// 3. Consolidar valores do dia
		var dayTotalInvested float64
		var dayGrossValue float64
		for _, lot := range activeLots {
			dayTotalInvested += lot.quantity * lot.unitPrice
			dayGrossValue += lot.grossValue
		}

		points = append(points, TreasuryPerfPoint{
			Date:          dateStr,
			Value:         dayGrossValue,
			TotalInvested: dayTotalInvested,
		})

		currDate = currDate.AddDate(0, 0, 1)
	}

	return points, nil
}

func (s *service) GetIndexRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	return s.repo.GetIndexRates(ctx, indexer, startDate, endDate)
}

func (s *service) UpdateTreasuryTransaction(ctx context.Context, portfolioID, txID string, req *TreasuryTxRequest) error {
	maturityDate, err := time.Parse("2006-01-02", req.MaturityDate)
	if err != nil {
		return fmt.Errorf("invalid maturity date: %w", err)
	}
	transactionDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		return fmt.Errorf("invalid transaction date: %w", err)
	}

	return s.repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
		existingTx, err := s.repo.GetTreasuryTransactionByID(ctx, tx, txID)
		if err != nil {
			return fmt.Errorf("transaction not found: %w", err)
		}

		if existingTx.PortfolioID != portfolioID {
			return fmt.Errorf("unauthorized: transaction does not belong to the portfolio")
		}

		oldAssetID := existingTx.AssetID

		assetID, err := s.repo.GetTreasuryAssetByTicker(ctx, tx, req.Ticker)
		if err == pgx.ErrNoRows {
			assetID, err = s.repo.CreateTreasuryAsset(ctx, tx, req.Ticker, req.Ticker, req.TreasuryType, maturityDate, req.HasCoupons)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		existingTx.AssetID = assetID
		existingTx.Type = req.Type
		existingTx.Quantity = req.Quantity
		existingTx.UnitPrice = req.UnitPrice
		existingTx.ContractedRate = req.ContractedRate
		existingTx.TransactionDate = transactionDate

		if req.Type == "SUBSCRIPTION" {
			existingTx.RemainingQuantity = req.Quantity
		} else {
			existingTx.RemainingQuantity = 0.0
		}

		err = s.repo.UpdateTreasuryTransaction(ctx, tx, existingTx)
		if err != nil {
			return err
		}

		if oldAssetID != assetID {
			err = s.rebuildTreasuryFIFO(ctx, tx, portfolioID, oldAssetID)
			if err != nil {
				return err
			}
		}

		return s.rebuildTreasuryFIFO(ctx, tx, portfolioID, assetID)
	})
}

func (s *service) DeleteTreasuryTransaction(ctx context.Context, portfolioID, txID string) error {
	return s.repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
		existingTx, err := s.repo.GetTreasuryTransactionByID(ctx, tx, txID)
		if err != nil {
			return fmt.Errorf("transaction not found: %w", err)
		}

		if existingTx.PortfolioID != portfolioID {
			return fmt.Errorf("unauthorized: transaction does not belong to the portfolio")
		}

		assetID := existingTx.AssetID

		err = s.repo.DeleteTreasuryTransactionByID(ctx, tx, txID)
		if err != nil {
			return err
		}

		return s.rebuildTreasuryFIFO(ctx, tx, portfolioID, assetID)
	})
}

func (s *service) rebuildTreasuryFIFO(ctx context.Context, tx pgx.Tx, portfolioID, assetID string) error {
	err := s.repo.DeleteDepletionsByAsset(ctx, tx, portfolioID, assetID)
	if err != nil {
		return err
	}

	err = s.repo.ResetSubscriptionsRemainingQuantity(ctx, tx, portfolioID, assetID)
	if err != nil {
		return err
	}

	err = s.repo.ResetRedemptionFinancials(ctx, tx, portfolioID, assetID)
	if err != nil {
		return err
	}

	redemptions, err := s.repo.GetRedemptionsForAsset(ctx, tx, portfolioID, assetID)
	if err != nil {
		return err
	}

	if len(redemptions) == 0 {
		return nil
	}

	holidays, err := s.repo.GetAnbimaHolidays(ctx)
	if err != nil {
		return err
	}

	selicRates, err := s.repo.GetSelicRates(ctx)
	if err != nil {
		return err
	}

	totalSelicInvested, err := s.repo.GetTotalSelicInvested(ctx, tx, portfolioID)
	if err != nil {
		return err
	}

	_, treasuryType, _, _, err := s.repo.GetTreasuryAssetDetails(ctx, assetID)
	if err != nil {
		return err
	}

	for _, redemption := range redemptions {
		lots, err := s.repo.GetActiveLotsForAsset(ctx, tx, portfolioID, assetID)
		if err != nil {
			return err
		}

		remainingToRedeem := redemption.Quantity
		var totalGross, totalIOF, totalIR, totalB3, totalNet float64

		for _, l := range lots {
			if remainingToRedeem <= 0 {
				break
			}
			depleteQty := l.RemainingQuantity
			if depleteQty > remainingToRedeem {
				depleteQty = remainingToRedeem
			}

			holdingDays := int(redemption.TransactionDate.Sub(l.TransactionDate).Hours() / 24)
			busDays := countTreasuryBusinessDays(l.TransactionDate, redemption.TransactionDate, holidays)

			var valAtRedemption float64
			if treasuryType == "SELIC" {
				factor := 1.0
				currDate := l.TransactionDate
				for currDate.Before(redemption.TransactionDate) {
					currDate = currDate.AddDate(0, 0, 1)
					if currDate.Weekday() != time.Saturday && currDate.Weekday() != time.Sunday && !holidays[currDate.Format("2006-01-02")] {
						rate := 10.75
						if rVal, exists := selicRates[currDate.Format("2006-01-02")]; exists {
							rate = rVal
						}
						dailyRate := math.Pow(1.0+rate/100.0, 1.0/252.0) - 1.0
						dailySpread := math.Pow(1.0+l.ContractedRate/100.0, 1.0/252.0) - 1.0
						factor *= (1.0 + dailyRate + dailySpread)
					}
				}
				valAtRedemption = depleteQty * l.UnitPrice * factor
			} else {
				rate := l.ContractedRate
				dailyRate := math.Pow(1.0+rate/100.0, 1.0/252.0) - 1.0
				factor := math.Pow(1.0+dailyRate, float64(busDays))
				valAtRedemption = depleteQty * l.UnitPrice * factor
			}

			costBasis := depleteQty * l.UnitPrice
			grossYield := valAtRedemption - costBasis
			if grossYield < 0 {
				grossYield = 0
			}

			var b3Fee float64
			dailyB3Rate := math.Pow(1.0+0.0020, 1.0/252.0) - 1.0

			if treasuryType == "SELIC" {
				exemptFraction := 1.0
				if totalSelicInvested > 10000.0 {
					exemptFraction = 10000.0 / totalSelicInvested
				}
				if exemptFraction > 1.0 {
					exemptFraction = 1.0
				}
				taxablePortion := 1.0 - exemptFraction
				accruedFeeFactor := math.Pow(1.0+dailyB3Rate*taxablePortion, float64(busDays)) - 1.0
				b3Fee = valAtRedemption * accruedFeeFactor
			} else {
				accruedFeeFactor := math.Pow(1.0+dailyB3Rate, float64(busDays)) - 1.0
				b3Fee = valAtRedemption * accruedFeeFactor
			}

			iofRate := getTreasuryIOFRate(holdingDays)
			iofTax := grossYield * (iofRate / 100.0)

			irRate := getTreasuryIRRate(holdingDays)
			irTax := (grossYield - iofTax) * (irRate / 100.0)
			if irTax < 0 {
				irTax = 0
			}

			netYield := grossYield - iofTax - irTax - b3Fee
			netVal := costBasis + netYield

			totalGross += valAtRedemption
			totalIOF += iofTax
			totalIR += irTax
			totalB3 += b3Fee
			totalNet += netVal

			newRemaining := l.RemainingQuantity - depleteQty
			err = s.repo.UpdateLotRemainingQuantity(ctx, tx, l.ID, newRemaining)
			if err != nil {
				return err
			}

			err = s.repo.CreateDepletionLink(ctx, tx, l.ID, redemption.ID, depleteQty)
			if err != nil {
				return err
			}

			remainingToRedeem -= depleteQty
		}

		err = s.repo.UpdateRedemptionFinancials(ctx, tx, redemption.ID, totalGross, totalIOF, totalIR, totalB3, totalNet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) GetTreasuryMonthlyYields(ctx context.Context, portfolioID string) ([]MonthlyYield, error) {
	lots, err := s.repo.GetActiveSubscriptionLots(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	holidays, err := s.repo.GetAnbimaHolidays(ctx)
	if err != nil {
		holidays = make(map[string]bool)
	}

	selicRates, err := s.repo.GetSelicRates(ctx)
	if err != nil {
		selicRates = make(map[string]float64)
	}

	var allYields []MonthlyYield

	for _, lot := range lots {
		ticker, treasuryType, maturityDate, _, err := s.repo.GetTreasuryAssetDetails(ctx, lot.AssetID)
		if err != nil {
			continue
		}

		startDate := lot.TransactionDate
		today := time.Now()
		limitDate := today
		if !maturityDate.IsZero() && today.After(maturityDate) {
			limitDate = maturityDate
		}

		grossValue := lot.RemainingQuantity * lot.UnitPrice
		monthlyGross := make(map[string]float64)
		monthlyLastDay := make(map[string]time.Time)

		currDate := startDate
		for !currDate.After(limitDate) {
			if currDate.Weekday() != time.Saturday &&
				currDate.Weekday() != time.Sunday &&
				!holidays[currDate.Format("2006-01-02")] {

				var dailyFactor float64
				if treasuryType == "SELIC" {
					rate := 10.75
					if rVal, exists := selicRates[currDate.Format("2006-01-02")]; exists {
						rate = rVal
					}
					dailyRate := math.Pow(1.0+rate/100.0, 1.0/252.0) - 1.0
					dailySpread := math.Pow(1.0+lot.ContractedRate/100.0, 1.0/252.0) - 1.0
					dailyFactor = 1.0 + dailyRate + dailySpread
				} else {
					dailyFactor = math.Pow(1.0+lot.ContractedRate/100.0, 1.0/252.0)
				}

				monthStr := currDate.Format("2006-01")
				monthlyGross[monthStr] += grossValue * (dailyFactor - 1)
				monthlyLastDay[monthStr] = currDate
				grossValue *= dailyFactor
			}
			currDate = currDate.AddDate(0, 0, 1)
		}

		for monthStr, grossYield := range monthlyGross {
			if grossYield <= 0 {
				continue
			}
			lastDay := monthlyLastDay[monthStr]
			daysHeld := int(lastDay.Sub(startDate).Hours() / 24)
			if daysHeld < 0 {
				daysHeld = 0
			}
			irRate := calculateIRRate(daysHeld)
			allYields = append(allYields, MonthlyYield{
				AssetID:     lot.AssetID,
				AssetName:   ticker,
				AssetType:   "TESOURO",
				Month:       monthStr,
				GrossAmount: grossYield,
				NetAmount:   grossYield * (1 - irRate),
				IsAccrued:   true,
			})
		}
	}

	if allYields == nil {
		allYields = []MonthlyYield{}
	}

	return allYields, nil
}

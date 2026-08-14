package fixedincome

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTreasury_HelperFunctions(t *testing.T) {
	// countTreasuryBusinessDays
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // Thursday (New Year)
	t2 := time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC) // Thursday

	holidays := map[string]bool{"2026-01-01": true}
	days := countTreasuryBusinessDays(t1, t2, holidays)
	// Fri Jan 2, Mon Jan 5, Tue Jan 6, Wed Jan 7, Thu Jan 8 = 5 business days
	assert.Equal(t, 5, days)

	assert.Equal(t, 0, countTreasuryBusinessDays(t2, t1, holidays))

	// getTreasuryIOFRate
	assert.Equal(t, 96.0, getTreasuryIOFRate(-1))
	assert.Equal(t, 96.0, getTreasuryIOFRate(0))
	assert.Equal(t, 96.0, getTreasuryIOFRate(1))
	assert.Equal(t, 93.0, getTreasuryIOFRate(2))
	assert.Equal(t, 3.0, getTreasuryIOFRate(29))
	assert.Equal(t, 0.0, getTreasuryIOFRate(30))
	assert.Equal(t, 0.0, getTreasuryIOFRate(50))

	// getTreasuryIRRate
	assert.Equal(t, 22.5, getTreasuryIRRate(100))
	assert.Equal(t, 22.5, getTreasuryIRRate(180))
	assert.Equal(t, 20.0, getTreasuryIRRate(200))
	assert.Equal(t, 20.0, getTreasuryIRRate(360))
	assert.Equal(t, 17.5, getTreasuryIRRate(500))
	assert.Equal(t, 17.5, getTreasuryIRRate(720))
	assert.Equal(t, 15.0, getTreasuryIRRate(721))
}

func TestTreasury_GetTreasuryPositions(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Error on GetActiveSubscriptionLots
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p1").Return(nil, errors.New("db error")).Once()
	pos, err := svc.GetTreasuryPositions(ctx, "p1")
	assert.Error(t, err)
	assert.Nil(t, pos)

	// 2. Empty lots
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p1").Return([]TreasuryTransaction{}, nil).Once()
	pos, err = svc.GetTreasuryPositions(ctx, "p1")
	assert.NoError(t, err)
	assert.Empty(t, pos)

	// 3. Active lots with SELIC asset
	matDate := time.Now().AddDate(2, 0, 0)
	subDate := time.Now().AddDate(-1, 0, 0)
	lots := []TreasuryTransaction{
		{
			ID:                "lot1",
			AssetID:           "a1",
			Type:              "SUBSCRIPTION",
			RemainingQuantity: 10.0,
			UnitPrice:         100.0,
			ContractedRate:    0.0,
			TransactionDate:   subDate,
		},
	}

	mockRepo.On("GetActiveSubscriptionLots", ctx, "p1").Return(lots, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a1").Return("LFT", "SELIC", matDate, false, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{
		subDate.Format("2006-01-02"): 0.05,
	}, nil).Once()

	pos, err = svc.GetTreasuryPositions(ctx, "p1")
	assert.NoError(t, err)
	assert.Len(t, pos, 1)
	assert.Equal(t, "LFT", pos[0].Ticker)
	assert.Equal(t, "SELIC", pos[0].TreasuryType)
}

func TestTreasury_GetTreasuryTransactions(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	mockRepo.On("GetTreasuryTransactionsList", ctx, "p1").Return([]TreasuryTxRequest{
		{ID: "tx1", Ticker: "LFT"},
	}, nil).Once()

	txs, err := svc.GetTreasuryTransactions(ctx, "p1")
	assert.NoError(t, err)
	assert.Len(t, txs, 1)
}

func TestTreasury_CreateTreasuryTransaction(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	req := &TreasuryTxRequest{
		Ticker:          "LFT",
		Type:            "SUBSCRIPTION",
		Quantity:        5.0,
		UnitPrice:       100.0,
		ContractedRate:  10.0,
		MaturityDate:    "2028-12-31",
		TransactionDate: time.Now().Format("2006-01-02"),
	}

	// 1. Success subscription
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a1", nil).Once()
	mockRepo.On("CreateTreasurySubscription", ctx, mock.Anything, "p1", "a1", 5.0, 100.0, 10.0, mock.Anything).Return("sub1", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Once()

	res, err := svc.CreateTreasuryTransaction(ctx, "p1", req)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"id": "sub1", "status": "subscribed"}, res)
}

func TestTreasury_GetTreasuryPerformance(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	mockRepo.On("GetTreasuryTransactionsList", ctx, "p1").Return([]TreasuryTxRequest{
		{ID: "tx1", Ticker: "LFT", TransactionDate: "2026-01-01"},
	}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Maybe()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Maybe()
	mockRepo.On("GetTreasuryPerformancePoints", ctx, "p1").Return([]TreasuryPerfPoint{
		{Date: "2026-01-01", Value: 1000},
	}, nil).Once()

	pts, err := svc.GetTreasuryPerformance(ctx, "p1")
	assert.NoError(t, err)
	assert.NotEmpty(t, pts)
}

func TestTreasury_GetIndexRates(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()
	now := time.Now()

	mockRepo.On("GetIndexRates", ctx, "CDI", now, now).Return([]IndexRate{
		{Indexer: "CDI", Rate: 0.05},
	}, nil).Once()

	rates, err := svc.GetIndexRates(ctx, "CDI", now, now)
	assert.NoError(t, err)
	assert.Len(t, rates, 1)
}

func TestTreasury_UpdateDeleteTransaction(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// Update
	req := &TreasuryTxRequest{
		Ticker:          "LFT",
		Type:            "SUBSCRIPTION",
		Quantity:        5.0,
		UnitPrice:       100.0,
		MaturityDate:    "2028-12-31",
		TransactionDate: time.Now().Format("2006-01-02"),
	}
	txObj := &TreasuryTransaction{ID: "tx1", AssetID: "a1", PortfolioID: "p1"}

	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx1").Return(txObj, nil).Once()
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a1", nil).Once()
	mockRepo.On("UpdateTreasuryTransaction", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Once()

	err := svc.UpdateTreasuryTransaction(ctx, "p1", "tx1", req)
	assert.NoError(t, err)

	// Delete
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx1").Return(txObj, nil).Once()
	mockRepo.On("DeleteTreasuryTransactionByID", ctx, mock.Anything, "tx1").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{}, nil).Once()

	err = svc.DeleteTreasuryTransaction(ctx, "p1", "tx1")
	assert.NoError(t, err)
}

func TestTreasury_FIFORebuild(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	subTx := TreasuryTransaction{
		ID:                "sub1",
		AssetID:           "a1",
		PortfolioID:       "p1",
		Type:              "SUBSCRIPTION",
		Quantity:          10.0,
		RemainingQuantity: 10.0,
		UnitPrice:         100.0,
		TransactionDate:   time.Now().AddDate(-1, 0, 0),
	}
	redTx := TreasuryTransaction{
		ID:              "red1",
		AssetID:         "a1",
		PortfolioID:     "p1",
		Type:            "REDEMPTION",
		Quantity:        5.0,
		UnitPrice:       120.0,
		TransactionDate: time.Now(),
	}

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a1").Return(nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{subTx}, nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a1").Return([]TreasuryTransaction{redTx}, nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "sub1", 5.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "sub1", "red1", 5.0).Return(nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(1000.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a1").Return("LFT", "SELIC", time.Now().AddDate(2, 0, 0), false, nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "red1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	err := svc.(*service).rebuildTreasuryFIFO(ctx, nil, "p1", "a1")
	assert.NoError(t, err)
}

func TestTreasury_GetTreasuryMonthlyYields(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p1").Return([]TreasuryTransaction{}, nil).Once()
	yields, err := svc.GetTreasuryMonthlyYields(ctx, "p1")
	assert.NoError(t, err)
	assert.Empty(t, yields)
}

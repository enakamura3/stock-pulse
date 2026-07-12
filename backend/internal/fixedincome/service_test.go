package fixedincome

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	Repository
	mock.Mock
}

func (m *MockRepository) GetTransactionsByPortfolio(ctx context.Context, portfolioID string) ([]Transaction, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Transaction), args.Error(1)
}

func (m *MockRepository) GetAssetsByPortfolio(ctx context.Context, portfolioID string) ([]Asset, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Asset), args.Error(1)
}

func (m *MockRepository) GetTreasuryTransactionsList(ctx context.Context, portfolioID string) ([]TreasuryTxRequest, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryTxRequest), args.Error(1)
}

func TestGetUnifiedTransactions_IncludesTreasury(t *testing.T) {
	mockRepo := &MockRepository{}
	svc := NewService(mockRepo, nil)

	portfolioID := "test-portfolio-123"
	ctx := context.Background()

	// 1. Mock standard fixed income transactions & assets
	maturityDate := time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)
	mockRepo.On("GetTransactionsByPortfolio", ctx, portfolioID).Return([]Transaction{
		{
			ID:       "tx-fi-1",
			AssetID:  "asset-fi-1",
			Type:     "SUBSCRIPTION",
			Amount:   5000.0,
			Date:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}, nil)

	mockRepo.On("GetAssetsByPortfolio", ctx, portfolioID).Return([]Asset{
		{
			ID:           "asset-fi-1",
			PortfolioID:  portfolioID,
			Institution:  "Banco do Brasil",
			Type:         "CDB",
			DebtType:     "PREFIXADO",
			Rate:         12.5,
			MaturityDate: maturityDate,
		},
	}, nil)

	// 2. Mock Treasury transactions
	mockRepo.On("GetTreasuryTransactionsList", ctx, portfolioID).Return([]TreasuryTxRequest{
		{
			ID:              "tx-td-1",
			Ticker:          "TESOURO SELIC 2029",
			TreasuryType:    "SELIC",
			MaturityDate:    "2029-03-01",
			HasCoupons:      false,
			Type:            "SUBSCRIPTION",
			Quantity:        1.5,
			UnitPrice:       14000.0,
			ContractedRate:  0.0,
			TransactionDate: "2026-02-10",
		},
		{
			ID:              "tx-td-2",
			Ticker:          "TESOURO IPCA+ 2035",
			TreasuryType:    "IPCA+",
			MaturityDate:    "2035-05-15",
			HasCoupons:      true,
			Type:            "REDEMPTION",
			Quantity:        0.5,
			UnitPrice:       3200.0,
			ContractedRate:  6.2,
			TransactionDate: "2026-03-20",
		},
	}, nil)

	// Call service
	unified, err := svc.GetUnifiedTransactions(ctx, portfolioID, "user-123")
	assert.NoError(t, err)
	assert.Len(t, unified, 3)

	// Check standard fixed income tx
	tx1 := unified[0]
	assert.Equal(t, "tx-fi-1", tx1.ID)
	assert.Equal(t, "RF", tx1.Module)
	assert.Equal(t, "CDB", tx1.AssetType)
	assert.Equal(t, "CDB 12.50% a.a. - Banco do Brasil", tx1.AssetName)
	assert.Nil(t, tx1.Quantity)
	assert.Nil(t, tx1.UnitPrice)
	assert.Equal(t, 5000.0, tx1.TotalValue)
	assert.Equal(t, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), tx1.Date)

	// Check Treasury SUBSCRIPTION tx
	tx2 := unified[1]
	assert.Equal(t, "tx-td-1", tx2.ID)
	assert.Equal(t, "RF", tx2.Module)
	assert.Equal(t, "TESOURO", tx2.AssetType)
	assert.Equal(t, "TESOURO SELIC 2029", tx2.AssetName)
	assert.Equal(t, 1.5, *tx2.Quantity)
	assert.Equal(t, 14000.0, *tx2.UnitPrice)
	assert.Equal(t, 21000.0, tx2.TotalValue)
	assert.Equal(t, "SUBSCRIPTION", tx2.Type)
	assert.Equal(t, time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), tx2.Date)

	// Check Treasury REDEMPTION tx
	tx3 := unified[2]
	assert.Equal(t, "tx-td-2", tx3.ID)
	assert.Equal(t, "RF", tx3.Module)
	assert.Equal(t, "TESOURO", tx3.AssetType)
	assert.Equal(t, "TESOURO IPCA+ 2035", tx3.AssetName)
	assert.Equal(t, 0.5, *tx3.Quantity)
	assert.Equal(t, 3200.0, *tx3.UnitPrice)
	assert.Equal(t, 1600.0, tx3.TotalValue)
	assert.Equal(t, "REDEMPTION", tx3.Type)
	assert.Equal(t, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), tx3.Date)

	mockRepo.AssertExpectations(t)
}

func TestGetUnifiedTransactions_Hibrido_OrMissingAsset(t *testing.T) {
	mockRepo := &MockRepository{}
	svc := NewService(mockRepo, nil)

	portfolioID := "test-portfolio-hibrido"
	ctx := context.Background()

	maturityDate := time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)
	mockRepo.On("GetTransactionsByPortfolio", ctx, portfolioID).Return([]Transaction{
		{
			ID:       "tx-fi-1",
			AssetID:  "asset-fi-1",
			Type:     "SUBSCRIPTION",
			Amount:   5000.0,
			Date:     time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:       "tx-fi-2",
			AssetID:  "asset-fi-missing",
			Type:     "SUBSCRIPTION",
			Amount:   3000.0,
			Date:     time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
		},
	}, nil)

	mockRepo.On("GetAssetsByPortfolio", ctx, portfolioID).Return([]Asset{
		{
			ID:           "asset-fi-1",
			PortfolioID:  portfolioID,
			Institution:  "Banco XP",
			Type:         "IPCA",
			DebtType:     "HIBRIDO",
			Indexer:      "IPCA",
			Rate:         6.5,
			MaturityDate: maturityDate,
		},
	}, nil)

	mockRepo.On("GetTreasuryTransactionsList", ctx, portfolioID).Return([]TreasuryTxRequest{}, nil)

	unified, err := svc.GetUnifiedTransactions(ctx, portfolioID, "user-123")
	assert.NoError(t, err)
	assert.Len(t, unified, 1)

	tx1 := unified[0]
	assert.Equal(t, "tx-fi-1", tx1.ID)
	assert.Equal(t, "IPCA IPCA + 6.50% - Banco XP", tx1.AssetName)
}

func TestGetUnifiedTransactions_Errors(t *testing.T) {
	ctx := context.Background()
	portfolioID := "test-portfolio-errors"

	t.Run("GetTransactionsByPortfolio error", func(t *testing.T) {
		mockRepo := &MockRepository{}
		svc := NewService(mockRepo, nil)
		mockRepo.On("GetTransactionsByPortfolio", ctx, portfolioID).Return([]Transaction(nil), assert.AnError)

		_, err := svc.GetUnifiedTransactions(ctx, portfolioID, "user-123")
		assert.Error(t, err)
	})

	t.Run("GetAssetsByPortfolio error", func(t *testing.T) {
		mockRepo := &MockRepository{}
		svc := NewService(mockRepo, nil)
		mockRepo.On("GetTransactionsByPortfolio", ctx, portfolioID).Return([]Transaction{}, nil)
		mockRepo.On("GetAssetsByPortfolio", ctx, portfolioID).Return([]Asset(nil), assert.AnError)

		_, err := svc.GetUnifiedTransactions(ctx, portfolioID, "user-123")
		assert.Error(t, err)
	})

	t.Run("GetTreasuryTransactionsList error", func(t *testing.T) {
		mockRepo := &MockRepository{}
		svc := NewService(mockRepo, nil)
		mockRepo.On("GetTransactionsByPortfolio", ctx, portfolioID).Return([]Transaction{}, nil)
		mockRepo.On("GetAssetsByPortfolio", ctx, portfolioID).Return([]Asset{}, nil)
		mockRepo.On("GetTreasuryTransactionsList", ctx, portfolioID).Return([]TreasuryTxRequest(nil), assert.AnError)

		_, err := svc.GetUnifiedTransactions(ctx, portfolioID, "user-123")
		assert.Error(t, err)
	})
}

func TestGetUnifiedTransactions_InvalidDates(t *testing.T) {
	mockRepo := &MockRepository{}
	svc := NewService(mockRepo, nil)

	portfolioID := "test-portfolio-invalid-dates"
	ctx := context.Background()

	mockRepo.On("GetTransactionsByPortfolio", ctx, portfolioID).Return([]Transaction{}, nil)
	mockRepo.On("GetAssetsByPortfolio", ctx, portfolioID).Return([]Asset{}, nil)

	mockRepo.On("GetTreasuryTransactionsList", ctx, portfolioID).Return([]TreasuryTxRequest{
		{
			ID:              "tx-td-1",
			Ticker:          "TESOURO SELIC 2029",
			TreasuryType:    "SELIC",
			MaturityDate:    "invalid-date",
			HasCoupons:      false,
			Type:            "SUBSCRIPTION",
			Quantity:        1.0,
			UnitPrice:       1000.0,
			ContractedRate:  0.0,
			TransactionDate: "invalid-date",
		},
	}, nil)

	unified, err := svc.GetUnifiedTransactions(ctx, portfolioID, "user-123")
	assert.NoError(t, err)
	assert.Len(t, unified, 1)

	tx := unified[0]
	assert.WithinDuration(t, time.Now(), tx.Date, 5*time.Second)
	assert.Nil(t, tx.MaturityDate)
}

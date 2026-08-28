package fixedincome

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"
)

type MockFullRepo struct {
	mock.Mock
}

func (m *MockFullRepo) CreateAsset(ctx context.Context, asset *Asset) (*Asset, error) {
	args := m.Called(ctx, asset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Asset), args.Error(1)
}

func (m *MockFullRepo) GetAssetsByPortfolio(ctx context.Context, portfolioID string) ([]Asset, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Asset), args.Error(1)
}

func (m *MockFullRepo) GetAssetByID(ctx context.Context, assetID string) (*Asset, error) {
	args := m.Called(ctx, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Asset), args.Error(1)
}

func (m *MockFullRepo) UpdateAsset(ctx context.Context, asset *Asset) error {
	args := m.Called(ctx, asset)
	return args.Error(0)
}

func (m *MockFullRepo) DeleteAsset(ctx context.Context, assetID string) error {
	args := m.Called(ctx, assetID)
	return args.Error(0)
}

func (m *MockFullRepo) CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	args := m.Called(ctx, tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Transaction), args.Error(1)
}

func (m *MockFullRepo) GetTransactionsByAsset(ctx context.Context, assetID string) ([]Transaction, error) {
	args := m.Called(ctx, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Transaction), args.Error(1)
}

func (m *MockFullRepo) GetTransactionsByPortfolio(ctx context.Context, portfolioID string) ([]Transaction, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Transaction), args.Error(1)
}

func (m *MockFullRepo) GetTransactionByID(ctx context.Context, txID string) (*Transaction, error) {
	args := m.Called(ctx, txID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Transaction), args.Error(1)
}

func (m *MockFullRepo) UpdateTransaction(ctx context.Context, txID string, tx *Transaction) error {
	args := m.Called(ctx, txID, tx)
	return args.Error(0)
}

func (m *MockFullRepo) DeleteTransaction(ctx context.Context, txID string) error {
	args := m.Called(ctx, txID)
	return args.Error(0)
}

func (m *MockFullRepo) SaveIndexRates(ctx context.Context, rates []IndexRate) error {
	args := m.Called(ctx, rates)
	return args.Error(0)
}

func (m *MockFullRepo) GetIndexRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	args := m.Called(ctx, indexer, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]IndexRate), args.Error(1)
}

func (m *MockFullRepo) GetLatestIndexRate(ctx context.Context, indexer string) (*IndexRate, error) {
	args := m.Called(ctx, indexer)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*IndexRate), args.Error(1)
}

func (m *MockFullRepo) ExecuteInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func (m *MockFullRepo) GetTreasuryAssetByTicker(ctx context.Context, tx pgx.Tx, ticker string) (string, error) {
	args := m.Called(ctx, tx, ticker)
	return args.String(0), args.Error(1)
}

func (m *MockFullRepo) CreateTreasuryAsset(ctx context.Context, tx pgx.Tx, ticker string, name string, treasuryType string, maturityDate time.Time, hasCoupons bool) (string, error) {
	args := m.Called(ctx, tx, ticker, name, treasuryType, maturityDate, hasCoupons)
	return args.String(0), args.Error(1)
}

func (m *MockFullRepo) CreateTreasurySubscription(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string, quantity float64, unitPrice float64, contractedRate float64, transactionDate time.Time) (string, error) {
	args := m.Called(ctx, tx, portfolioID, assetID, quantity, unitPrice, contractedRate, transactionDate)
	return args.String(0), args.Error(1)
}

func (m *MockFullRepo) CreateTreasuryRedemptionPlaceholder(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string, quantity float64, unitPrice float64, contractedRate float64, transactionDate time.Time) (string, error) {
	args := m.Called(ctx, tx, portfolioID, assetID, quantity, unitPrice, contractedRate, transactionDate)
	return args.String(0), args.Error(1)
}

func (m *MockFullRepo) GetActiveLotsForAsset(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) ([]TreasuryTransaction, error) {
	args := m.Called(ctx, tx, portfolioID, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryTransaction), args.Error(1)
}

func (m *MockFullRepo) UpdateLotRemainingQuantity(ctx context.Context, tx pgx.Tx, lotID string, remainingQuantity float64) error {
	args := m.Called(ctx, tx, lotID, remainingQuantity)
	return args.Error(0)
}

func (m *MockFullRepo) CreateDepletionLink(ctx context.Context, tx pgx.Tx, subID string, redID string, quantity float64) error {
	args := m.Called(ctx, tx, subID, redID, quantity)
	return args.Error(0)
}

func (m *MockFullRepo) UpdateRedemptionFinancials(ctx context.Context, tx pgx.Tx, redemptionID string, grossAmount float64, iofTax float64, irTax float64, b3Fee float64, netAmount float64) error {
	args := m.Called(ctx, tx, redemptionID, grossAmount, iofTax, irTax, b3Fee, netAmount)
	return args.Error(0)
}

func (m *MockFullRepo) GetAnbimaHolidays(ctx context.Context) (map[string]bool, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockFullRepo) GetSeededHolidayYears(ctx context.Context) ([]int, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockFullRepo) SaveAnbimaHolidays(ctx context.Context, dates []time.Time) error {
	args := m.Called(ctx, dates)
	return args.Error(0)
}

func (m *MockFullRepo) GetSelicRates(ctx context.Context) (map[string]float64, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]float64), args.Error(1)
}

func (m *MockFullRepo) GetTotalSelicInvested(ctx context.Context, tx pgx.Tx, portfolioID string) (float64, error) {
	args := m.Called(ctx, tx, portfolioID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockFullRepo) GetActiveSubscriptionLots(ctx context.Context, portfolioID string) ([]TreasuryTransaction, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryTransaction), args.Error(1)
}

func (m *MockFullRepo) GetTreasuryTransactionsList(ctx context.Context, portfolioID string) ([]TreasuryTxRequest, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryTxRequest), args.Error(1)
}

func (m *MockFullRepo) GetTreasuryPerformancePoints(ctx context.Context, portfolioID string) ([]TreasuryPerfPoint, error) {
	args := m.Called(ctx, portfolioID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryPerfPoint), args.Error(1)
}

func (m *MockFullRepo) GetTreasuryAssetDetails(ctx context.Context, assetID string) (string, string, time.Time, bool, error) {
	args := m.Called(ctx, assetID)
	return args.String(0), args.String(1), args.Get(2).(time.Time), args.Bool(3), args.Error(4)
}

func (m *MockFullRepo) GetTreasuryTransactionByID(ctx context.Context, tx pgx.Tx, id string) (*TreasuryTransaction, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TreasuryTransaction), args.Error(1)
}

func (m *MockFullRepo) UpdateTreasuryTransaction(ctx context.Context, tx pgx.Tx, t *TreasuryTransaction) error {
	args := m.Called(ctx, tx, t)
	return args.Error(0)
}

func (m *MockFullRepo) DeleteTreasuryTransactionByID(ctx context.Context, tx pgx.Tx, id string) error {
	args := m.Called(ctx, tx, id)
	return args.Error(0)
}

func (m *MockFullRepo) DeleteDepletionsByAsset(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) error {
	args := m.Called(ctx, tx, portfolioID, assetID)
	return args.Error(0)
}

func (m *MockFullRepo) ResetSubscriptionsRemainingQuantity(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) error {
	args := m.Called(ctx, tx, portfolioID, assetID)
	return args.Error(0)
}

func (m *MockFullRepo) ResetRedemptionFinancials(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) error {
	args := m.Called(ctx, tx, portfolioID, assetID)
	return args.Error(0)
}

func (m *MockFullRepo) GetRedemptionsForAsset(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) ([]TreasuryTransaction, error) {
	args := m.Called(ctx, tx, portfolioID, assetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TreasuryTransaction), args.Error(1)
}

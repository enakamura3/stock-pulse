package fixedincome

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockBCB struct {
	mock.Mock
}

func (m *mockBCB) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	args := m.Called(ctx, indexer, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]IndexRate), args.Error(1)
}

func TestService_CreateAsset(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	ctx := context.Background()
	a := &Asset{Institution: "Itaú", PortfolioID: "p1"}

	mockRepo.On("CreateAsset", ctx, a).Return(a, nil).Once()
	res, err := svc.CreateAsset(ctx, a)
	assert.NoError(t, err)
	assert.Equal(t, a, res)

	mockRepo.On("CreateAsset", ctx, a).Return(nil, errors.New("db error")).Once()
	res, err = svc.CreateAsset(ctx, a)
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestService_CreateTransaction(t *testing.T) {
	mockRepo := &MockFullRepo{}
	mockBcb := &mockBCB{}
	mockBcb.On("FetchRates", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]IndexRate{}, nil).Maybe()
	svc := NewService(mockRepo, mockBcb)

	ctx := context.Background()
	tx := &Transaction{AssetID: "a1", Type: "APLICACAO", Amount: 1000, Date: time.Now().AddDate(-1, 0, 0)}

	// 1. Error on CreateTransaction
	mockRepo.On("CreateTransaction", ctx, tx).Return(nil, errors.New("db error")).Once()
	_, err := svc.CreateTransaction(ctx, tx)
	assert.Error(t, err)

	// 2. Success with APLICACAO and POS asset (triggers backfill)
	mockRepo.On("CreateTransaction", ctx, tx).Return(tx, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(&Asset{ID: "a1", DebtType: "POS", Indexer: "CDI"}, nil).Once()
	mockRepo.On("GetLatestIndexRate", mock.Anything, "CDI").Return(&IndexRate{Date: time.Now()}, nil).Once()

	res, err := svc.CreateTransaction(ctx, tx)
	assert.NoError(t, err)
	assert.Equal(t, tx, res)
}

func TestService_UpdateTransaction(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Transaction Not Found
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(nil, errors.New("not found")).Once()
	err := svc.UpdateTransaction(ctx, "p1", "tx1", &Transaction{}, nil)
	assert.ErrorContains(t, err, "transaction not found")

	// 2. Asset Not Found
	tx1 := &Transaction{ID: "tx1", AssetID: "a1"}
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(nil, errors.New("asset not found")).Once()
	err = svc.UpdateTransaction(ctx, "p1", "tx1", &Transaction{}, nil)
	assert.ErrorContains(t, err, "failed to get asset")

	// 3. Unauthorized (different portfolio)
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(&Asset{PortfolioID: "other-p"}, nil).Once()
	err = svc.UpdateTransaction(ctx, "p1", "tx1", &Transaction{}, nil)
	assert.ErrorContains(t, err, "unauthorized")

	// 4. Success with maturity date update
	oldMat := time.Now().AddDate(1, 0, 0)
	newMat := time.Now().AddDate(2, 0, 0)
	asset := &Asset{ID: "a1", PortfolioID: "p1", MaturityDate: oldMat}
	updateTx := &Transaction{Type: "APLICACAO", Amount: 2000, Date: time.Now()}

	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(asset, nil).Once()
	mockRepo.On("UpdateTransaction", ctx, "tx1", mock.Anything).Return(nil).Once()
	mockRepo.On("UpdateAsset", ctx, mock.Anything).Return(nil).Once()

	err = svc.UpdateTransaction(ctx, "p1", "tx1", updateTx, &newMat)
	assert.NoError(t, err)
}

func TestService_DeleteTransaction(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Transaction Not Found
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(nil, errors.New("not found")).Once()
	err := svc.DeleteTransaction(ctx, "p1", "tx1")
	assert.ErrorContains(t, err, "transaction not found")

	// 2. Asset Not Found
	tx1 := &Transaction{ID: "tx1", AssetID: "a1"}
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(nil, errors.New("asset not found")).Once()
	err = svc.DeleteTransaction(ctx, "p1", "tx1")
	assert.ErrorContains(t, err, "failed to get asset")

	// 3. Unauthorized
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(&Asset{PortfolioID: "other-p"}, nil).Once()
	err = svc.DeleteTransaction(ctx, "p1", "tx1")
	assert.ErrorContains(t, err, "unauthorized")

	// 4. Success
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(&Asset{PortfolioID: "p1"}, nil).Once()
	mockRepo.On("DeleteTransaction", ctx, "tx1").Return(nil).Once()
	err = svc.DeleteTransaction(ctx, "p1", "tx1")
	assert.NoError(t, err)
}

func TestService_TaxCalculations(t *testing.T) {
	// IOF tests
	assert.Equal(t, 100.0, calculateIOF(-5))
	assert.Equal(t, 1.0, calculateIOF(0))
	assert.Equal(t, 0.96, calculateIOF(1))
	assert.Equal(t, 0.03, calculateIOF(29))
	assert.Equal(t, 0.0, calculateIOF(30))
	assert.Equal(t, 0.0, calculateIOF(100))

	// IR tests
	assert.Equal(t, 0.225, calculateIRRate(100))
	assert.Equal(t, 0.225, calculateIRRate(180))
	assert.Equal(t, 0.20, calculateIRRate(181))
	assert.Equal(t, 0.20, calculateIRRate(360))
	assert.Equal(t, 0.175, calculateIRRate(361))
	assert.Equal(t, 0.175, calculateIRRate(720))
	assert.Equal(t, 0.15, calculateIRRate(721))
}

func TestService_GetPortfolioPositions(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Error fetching assets
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return(nil, errors.New("db error")).Once()
	pos, err := svc.GetPortfolioPositions(ctx, "p1")
	assert.Error(t, err)
	assert.Nil(t, pos)

	// 2. Success with PRE and LCI (Tax Exempt) assets
	startDate := time.Now().AddDate(-1, 0, 0)
	matDate := time.Now().AddDate(1, 0, 0)
	assetPre := Asset{
		ID:           "a1",
		PortfolioID:  "p1",
		Institution:  "Itaú",
		Type:         "CDB",
		DebtType:     "PRE",
		Rate:         12.0,
		MaturityDate: matDate,
	}
	assetLci := Asset{
		ID:           "a2",
		PortfolioID:  "p1",
		Institution:  "Bradesco",
		Type:         "LCI",
		DebtType:     "POS",
		Indexer:      "CDI",
		Rate:         100.0,
		MaturityDate: matDate,
	}

	txPre := Transaction{ID: "t1", AssetID: "a1", Type: "SUBSCRIPTION", Amount: 1000, Date: startDate}
	txLci := Transaction{ID: "t2", AssetID: "a2", Type: "SUBSCRIPTION", Amount: 2000, Date: startDate}

	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetPre, assetLci}, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(&assetPre, nil)
	mockRepo.On("GetAssetByID", ctx, "a2").Return(&assetLci, nil)
	mockRepo.On("GetTransactionsByAsset", ctx, "a1").Return([]Transaction{txPre}, nil)
	mockRepo.On("GetTransactionsByAsset", ctx, "a2").Return([]Transaction{txLci}, nil)
	mockRepo.On("GetIndexRates", ctx, "CDI", mock.Anything, mock.Anything).Return([]IndexRate{
		{Date: startDate, Rate: 0.05},
	}, nil)

	positions, err := svc.GetPortfolioPositions(ctx, "p1")
	assert.NoError(t, err)
	assert.Len(t, positions, 2)
}

func TestService_GetPortfolioPerformance(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// 1. Error fetching assets
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return(nil, errors.New("db error")).Once()
	pts, err := svc.GetPortfolioPerformance(ctx, "p1", "1M")
	assert.Error(t, err)
	assert.Nil(t, pts)

	// 2. Empty assets
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{}, nil).Once()
	pts, err = svc.GetPortfolioPerformance(ctx, "p1", "1M")
	assert.NoError(t, err)
	assert.Empty(t, pts)
}

func TestService_GetRawTransactionsAndAssets(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	mockRepo.On("GetTransactionsByPortfolio", ctx, "p1").Return([]Transaction{{ID: "t1"}}, nil).Once()
	txs, err := svc.GetRawTransactions(ctx, "p1")
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{{ID: "a1"}}, nil).Once()
	assets, err := svc.GetAssetsByPortfolio(ctx, "p1")
	assert.NoError(t, err)
	assert.Len(t, assets, 1)
}

func TestService_CalculateMonthlyYields(t *testing.T) {
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	ctx := context.Background()

	// Error getting assets
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return(nil, errors.New("db error")).Once()
	yields, err := svc.CalculateMonthlyYields(ctx, "p1")
	assert.Error(t, err)
	assert.Nil(t, yields)

	// Success empty
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{}, nil).Once()
	yields, err = svc.CalculateMonthlyYields(ctx, "p1")
	assert.NoError(t, err)
	assert.Empty(t, yields)
}

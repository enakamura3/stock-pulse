package fixedincome

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := f(req)
	if resp == nil {
		return nil, errors.New("network error")
	}
	return resp, nil
}

func TestService_CoverageExtra_UpdateTransaction(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	tx1 := &Transaction{ID: "tx1", AssetID: "a1"}
	asset := &Asset{ID: "a1", PortfolioID: "p1", MaturityDate: time.Now().AddDate(1, 0, 0)}
	newMat := time.Now().AddDate(2, 0, 0)

	// UpdateTransaction repo error
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(asset, nil).Once()
	mockRepo.On("UpdateTransaction", ctx, "tx1", mock.Anything).Return(errors.New("repo update err")).Once()

	err := svc.UpdateTransaction(ctx, "p1", "tx1", tx1, nil)
	assert.ErrorContains(t, err, "repo update err")

	// UpdateAsset repo error
	mockRepo.On("GetTransactionByID", ctx, "tx1").Return(tx1, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a1").Return(asset, nil).Once()
	mockRepo.On("UpdateTransaction", ctx, "tx1", mock.Anything).Return(nil).Once()
	mockRepo.On("UpdateAsset", ctx, mock.Anything).Return(errors.New("asset update err")).Once()

	err = svc.UpdateTransaction(ctx, "p1", "tx1", tx1, &newMat)
	assert.ErrorContains(t, err, "asset update err")
}

func TestService_CoverageExtra_PositionWithHistory(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	sImpl := svc.(*service)

	// 1. GetAssetByID error
	mockRepo.On("GetAssetByID", ctx, "a_err").Return(nil, errors.New("asset not found")).Once()
	_, _, _, err := sImpl.getAssetPositionWithHistory(ctx, "a_err")
	assert.Error(t, err)

	// 2. GetTransactionsByAsset error
	mockRepo.On("GetAssetByID", ctx, "a_tx_err").Return(&Asset{ID: "a_tx_err"}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_tx_err").Return(nil, errors.New("tx err")).Once()
	_, _, _, err = sImpl.getAssetPositionWithHistory(ctx, "a_tx_err")
	assert.Error(t, err)

	// 3. len(txs) == 0
	mockRepo.On("GetAssetByID", ctx, "a_empty").Return(&Asset{ID: "a_empty"}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_empty").Return([]Transaction{}, nil).Once()
	pos, _, _, err := sImpl.getAssetPositionWithHistory(ctx, "a_empty")
	assert.NoError(t, err)
	assert.Nil(t, pos)

	// 4. Matured asset, POS with rate found, REDEMPTION reducing grossValue
	startDate := time.Now().AddDate(-2, 0, 0)
	matDate := time.Now().AddDate(-1, 0, 0) // Already matured
	assetPos := &Asset{
		ID:           "a_pos",
		PortfolioID:  "p1",
		DebtType:     "POS",
		Indexer:      "CDI",
		Rate:         100.0,
		MaturityDate: matDate,
	}
	txSub := Transaction{ID: "t1", AssetID: "a_pos", Type: "SUBSCRIPTION", Amount: 1000.0, Date: startDate}
	txRed := Transaction{ID: "t2", AssetID: "a_pos", Type: "REDEMPTION", Amount: 200.0, Date: startDate.AddDate(0, 1, 0)}

	mockRepo.On("GetAssetByID", ctx, "a_pos").Return(assetPos, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_pos").Return([]Transaction{txSub, txRed}, nil).Once()
	mockRepo.On("GetIndexRates", ctx, "CDI", mock.Anything, mock.Anything).Return([]IndexRate{
		{Date: startDate, Rate: 0.05},
	}, nil).Once()

	pos, histNet, histInv, err := sImpl.getAssetPositionWithHistory(ctx, "a_pos")
	assert.NoError(t, err)
	assert.NotNil(t, pos)
	assert.NotEmpty(t, histNet)
	assert.NotEmpty(t, histInv)

	// 5. HIBRIDO asset with REDEMPTION withdrawalRatio > 1, currentQty <= 0
	assetHib := &Asset{
		ID:           "a_hib",
		PortfolioID:  "p1",
		DebtType:     "HIBRIDO",
		Indexer:      "IPCA",
		Rate:         6.0,
		MaturityDate: time.Now().AddDate(1, 0, 0),
	}
	txSubH := Transaction{ID: "t3", AssetID: "a_hib", Type: "SUBSCRIPTION", Amount: 500.0, Date: startDate}
	txRedH := Transaction{ID: "t4", AssetID: "a_hib", Type: "REDEMPTION", Amount: 2000.0, Date: startDate.AddDate(0, 1, 0)} // Exceeds grossValue

	mockRepo.On("GetAssetByID", ctx, "a_hib").Return(assetHib, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_hib").Return([]Transaction{txSubH, txRedH}, nil).Once()
	mockRepo.On("GetIndexRates", ctx, "IPCA", mock.Anything, mock.Anything).Return([]IndexRate{}, nil).Once() // Fallback rate

	posH, _, _, err := sImpl.getAssetPositionWithHistory(ctx, "a_hib")
	assert.NoError(t, err)
	assert.NotNil(t, posH)
	assert.Equal(t, 0.0, posH.GrossValue)
}

func TestService_CoverageExtra_PerformanceAndYields(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	subDate := time.Now().AddDate(0, -5, 0)
	assetPre := Asset{
		ID:           "a_perf_pre",
		PortfolioID:  "p1",
		Institution:  "BB",
		Type:         "CDB",
		DebtType:     "PRE",
		Rate:         12.0,
		MaturityDate: time.Now().AddDate(1, 0, 0),
	}
	assetLca := Asset{
		ID:           "a_perf_lca",
		PortfolioID:  "p1",
		Institution:  "Sicredi",
		Type:         "LCA",
		DebtType:     "POS",
		Indexer:      "CDI",
		Rate:         95.0,
		MaturityDate: time.Now().AddDate(1, 0, 0),
	}
	tx1 := Transaction{ID: "tx1", AssetID: "a_perf_pre", Type: "SUBSCRIPTION", Amount: 1000.0, Date: subDate}
	tx2 := Transaction{ID: "tx2", AssetID: "a_perf_lca", Type: "SUBSCRIPTION", Amount: 2000.0, Date: subDate}

	// Performance across periods: 3M, 6M, 1Y, ALL
	periods := []string{"3M", "6M", "1Y", "ALL"}
	for _, p := range periods {
		mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetPre, assetLca}, nil).Once()
		mockRepo.On("GetTransactionsByAsset", ctx, "a_perf_pre").Return([]Transaction{tx1}, nil)
		mockRepo.On("GetTransactionsByAsset", ctx, "a_perf_lca").Return([]Transaction{tx2}, nil)
		mockRepo.On("GetAssetByID", ctx, "a_perf_pre").Return(&assetPre, nil)
		mockRepo.On("GetAssetByID", ctx, "a_perf_lca").Return(&assetLca, nil)
		mockRepo.On("GetIndexRates", ctx, "CDI", mock.Anything, mock.Anything).Return([]IndexRate{
			{Date: subDate, Rate: 0.05},
		}, nil)

		pts, err := svc.GetPortfolioPerformance(ctx, "p1", p)
		assert.NoError(t, err)
		assert.NotEmpty(t, pts)
	}

	// CalculateMonthlyYields full test (PRE, POS, HIBRIDO, tax exempt)
	assetHib := Asset{
		ID:           "a_perf_hib",
		PortfolioID:  "p1",
		Institution:  "Caixa",
		Type:         "CRI",
		DebtType:     "HIBRIDO",
		Indexer:      "IPCA",
		Rate:         7.0,
		MaturityDate: time.Now().AddDate(2, 0, 0),
	}
	tx3 := Transaction{ID: "tx3", AssetID: "a_perf_hib", Type: "SUBSCRIPTION", Amount: 1500.0, Date: subDate}

	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetPre, assetLca, assetHib}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_perf_hib").Return([]Transaction{tx3}, nil)
	mockRepo.On("GetAssetByID", ctx, "a_perf_hib").Return(&assetHib, nil)
	mockRepo.On("GetIndexRates", ctx, "IPCA", mock.Anything, mock.Anything).Return([]IndexRate{
		{Date: subDate, Rate: 0.02},
	}, nil)

	yields, err := svc.CalculateMonthlyYields(ctx, "p1")
	assert.NoError(t, err)
	assert.NotEmpty(t, yields)

	// calculateAssetMonthlyYields with empty transactions or transactions error
	sImpl := svc.(*service)
	mockRepo.On("GetTransactionsByAsset", ctx, "a_no_tx").Return([]Transaction{}, nil).Once()
	y, err := sImpl.calculateAssetMonthlyYields(ctx, Asset{ID: "a_no_tx"})
	assert.NoError(t, err)
	assert.Nil(t, y)

	mockRepo.On("GetTransactionsByAsset", ctx, "a_tx_err").Return(nil, errors.New("tx err")).Once()
	y, err = sImpl.calculateAssetMonthlyYields(ctx, Asset{ID: "a_tx_err"})
	assert.Error(t, err)
	assert.Nil(t, y)
}

func TestTreasury_CoverageExtra_CreateTransaction(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	// 1. Invalid maturity date
	reqBadMat := &TreasuryTxRequest{MaturityDate: "invalid-date", TransactionDate: "2026-01-01"}
	_, err := svc.CreateTreasuryTransaction(ctx, "p1", reqBadMat)
	assert.ErrorContains(t, err, "invalid maturity date")

	// 2. Invalid transaction date
	reqBadTxDate := &TreasuryTxRequest{MaturityDate: "2026-12-31", TransactionDate: "invalid-date"}
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqBadTxDate)
	assert.ErrorContains(t, err, "invalid transaction date")

	validReq := &TreasuryTxRequest{
		Ticker:          "NTNB",
		TreasuryType:    "IPCA",
		MaturityDate:    "2029-05-15",
		TransactionDate: "2026-01-10",
		Type:            "UNKNOWN",
		Quantity:        2.0,
		UnitPrice:       3000.0,
	}

	// 3. Asset lookup error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "NTNB").Return("", errors.New("tx exec err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", validReq)
	assert.ErrorContains(t, err, "tx exec err")

	// 4. Invalid transaction type
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "NTNB").Return("", pgx.ErrNoRows).Once()
	mockRepo.On("CreateTreasuryAsset", ctx, mock.Anything, "NTNB", "NTNB", "IPCA", mock.Anything, false).Return("as_ntnb", nil).Once()

	_, err = svc.CreateTreasuryTransaction(ctx, "p1", validReq)
	assert.ErrorContains(t, err, "invalid transaction type")


	// 5. Redemption Success (SELIC and PREFIXADO, with depletion and financials)
	reqRedSelic := &TreasuryTxRequest{
		Ticker:          "LFT",
		TreasuryType:    "SELIC",
		MaturityDate:    "2029-03-01",
		TransactionDate: "2026-03-01",
		Type:            "REDEMPTION",
		Quantity:        5.0,
		UnitPrice:       1000.0,
		ContractedRate:  0.0,
	}

	subDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lot1 := TreasuryTransaction{
		ID:                "sub_lot1",
		AssetID:           "a_lft",
		Type:              "SUBSCRIPTION",
		Quantity:          10.0,
		RemainingQuantity: 10.0,
		UnitPrice:         1000.0,
		ContractedRate:    0.0,
		TransactionDate:   subDate,
	}

	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil)
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{lot1}, nil)
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{"2026-01-01": true}, nil)
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{"2026-01-02": 11.0}, nil)
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(15000.0, nil) // > 10000 taxable portion
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_lft", 5.0, 1000.0, 0.0, mock.Anything).Return("red_tx1", nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "sub_lot1", 5.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "sub_lot1", "red_tx1", 5.0).Return(nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "red_tx1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	res, err := svc.CreateTreasuryTransaction(ctx, "p1", reqRedSelic)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 6. Redemption error (e.g. UpdateLotRemainingQuantity fails)
	reqRedErr := &TreasuryTxRequest{
		Ticker:          "LFT",
		TreasuryType:    "SELIC",
		MaturityDate:    "2029-03-01",
		TransactionDate: "2026-03-01",
		Type:            "REDEMPTION",
		Quantity:        5.0,
		UnitPrice:       1000.0,
	}
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_lft", 5.0, 1000.0, 0.0, mock.Anything).Return("red_tx2", nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "sub_lot1", 5.0).Return(errors.New("db error updating lot")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRedErr)
	assert.ErrorContains(t, err, "db error updating lot")
}

func TestTreasury_CoverageExtra_UpdateAndDeleteAndRebuildFIFO(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	sImpl := svc.(*service)

	// 1. UpdateTreasuryTransaction date parse errors
	err := svc.UpdateTreasuryTransaction(ctx, "p1", "tx1", &TreasuryTxRequest{MaturityDate: "bad", TransactionDate: "2026-01-01"})
	assert.ErrorContains(t, err, "invalid maturity date")
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx1", &TreasuryTxRequest{MaturityDate: "2026-01-01", TransactionDate: "bad"})
	assert.ErrorContains(t, err, "invalid transaction date")

	reqUp := &TreasuryTxRequest{
		Ticker:          "LTN",
		TreasuryType:    "PREFIXADO",
		MaturityDate:    "2027-01-01",
		TransactionDate: "2026-01-01",
		Type:            "SUBSCRIPTION",
		Quantity:        5.0,
		UnitPrice:       800.0,
	}

	// 2. Transaction not found
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_not_found").Return(nil, errors.New("not found")).Once()
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx_not_found", reqUp)
	assert.ErrorContains(t, err, "transaction not found")

	// 3. Unauthorized portfolio
	existingTx := &TreasuryTransaction{ID: "tx1", PortfolioID: "p_other", AssetID: "a_old"}
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_auth").Return(existingTx, nil).Once()
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx_auth", reqUp)
	assert.ErrorContains(t, err, "unauthorized")

	// 4. Update changing asset ID (triggers rebuild for old and new asset)
	existingTxOwn := &TreasuryTransaction{ID: "tx1", PortfolioID: "p1", AssetID: "a_old"}
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx1").Return(existingTxOwn, nil).Once()
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LTN").Return("a_new", nil).Once()
	mockRepo.On("UpdateTreasuryTransaction", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", mock.Anything).Return(nil).Times(2)
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", mock.Anything).Return(nil).Times(2)
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", mock.Anything).Return(nil).Times(2)
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", mock.Anything).Return([]TreasuryTransaction{}, nil).Times(2)

	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx1", reqUp)
	assert.NoError(t, err)

	// 5. DeleteTreasuryTransaction not found and unauthorized
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_del_nf").Return(nil, errors.New("del not found")).Once()
	err = svc.DeleteTreasuryTransaction(ctx, "p1", "tx_del_nf")
	assert.ErrorContains(t, err, "transaction not found")

	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_del_unauth").Return(&TreasuryTransaction{PortfolioID: "p_other"}, nil).Once()
	err = svc.DeleteTreasuryTransaction(ctx, "p1", "tx_del_unauth")
	assert.ErrorContains(t, err, "unauthorized")

	// 6. rebuildTreasuryFIFO error branches
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_err1").Return(errors.New("del depletions err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_err1")
	assert.ErrorContains(t, err, "del depletions err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_err2").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_err2").Return(errors.New("reset subs err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_err2")
	assert.ErrorContains(t, err, "reset subs err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_err3").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_err3").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_err3").Return(errors.New("reset reds err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_err3")
	assert.ErrorContains(t, err, "reset reds err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_err4").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_err4").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_err4").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_err4").Return(nil, errors.New("get reds err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_err4")
	assert.ErrorContains(t, err, "get reds err")
}

func TestTreasury_CoverageExtra_MonthlyYields(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	// 1. GetActiveSubscriptionLots error
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p1").Return(nil, errors.New("lots err")).Once()
	_, err := svc.GetTreasuryMonthlyYields(ctx, "p1")
	assert.Error(t, err)

	// 2. Lots with PREFIXADO, SELIC with fallback rate and maturity passed
	subDate := time.Now().AddDate(0, -3, 0)
	lotPre := TreasuryTransaction{
		ID:                "lot_pre",
		AssetID:           "a_pre",
		RemainingQuantity: 2.0,
		UnitPrice:         900.0,
		ContractedRate:    11.5,
		TransactionDate:   subDate,
	}
	lotSelic := TreasuryTransaction{
		ID:                "lot_selic",
		AssetID:           "a_selic",
		RemainingQuantity: 1.0,
		UnitPrice:         1000.0,
		ContractedRate:    0.5,
		TransactionDate:   subDate,
	}

	mockRepo.On("GetActiveSubscriptionLots", ctx, "p1").Return([]TreasuryTransaction{lotPre, lotSelic}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(nil, errors.New("holidays err")).Once() // Fallback to empty map
	mockRepo.On("GetSelicRates", ctx).Return(nil, errors.New("selic err")).Once()       // Fallback to empty map
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_pre").Return("LTN", "PREFIXADO", time.Now().AddDate(1, 0, 0), false, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_selic").Return("LFT", "SELIC", time.Now().AddDate(2, 0, 0), false, nil).Once()

	yields, err := svc.GetTreasuryMonthlyYields(ctx, "p1")
	assert.NoError(t, err)
	assert.NotEmpty(t, yields)
}

func TestProviders_CoverageExtra(t *testing.T) {
	ctx := context.Background()

	// 1. BCBProvider
	mockBcb := &mockBCB{}
	mockBcb.On("FetchRates", ctx, "CDI", mock.Anything, mock.Anything).Return([]IndexRate{{Rate: 0.05}}, nil).Once()
	bcbProv := NewBCBProvider(mockBcb)
	rates, err := bcbProv.FetchRates(ctx, "CDI", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Len(t, rates, 1)

	// 2. YahooFinanceIndexProvider
	yp := NewYahooFinanceIndexProvider()
	assert.NotNil(t, yp)

	// Unsupported indexer
	_, err = yp.FetchRates(ctx, "UNSUPPORTED", time.Now(), time.Now())
	assert.ErrorContains(t, err, "nao suportado")

	// Yahoo HTTP error
	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return nil
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	// Yahoo Status != 200
	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("err"))}
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	// Yahoo Decode Error
	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad-json"))}
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	// Yahoo Empty Results
	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"chart":{"result":[]}}`))}
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.ErrorContains(t, err, "resultado historico vazio")

	// Yahoo Success
	ts := time.Now().AddDate(0, 0, -1).Unix()
	chartJSON := fmt.Sprintf(`{"chart":{"result":[{"timestamp":[%d],"indicators":{"quote":[{"close":[125000.0]}]}}]}}`, ts)
	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(chartJSON))}
	})
	rates, err = yp.FetchRates(ctx, "IBOV", time.Now().AddDate(0, 0, -2), time.Now())
	assert.NoError(t, err)
	assert.Len(t, rates, 1)

	// 3. BrapiProvider
	bp := NewBrapiProvider()
	assert.NotNil(t, bp)

	// Period > 3 months limit error
	_, err = bp.FetchRates(ctx, "IBOV", time.Now().AddDate(-1, 0, 0), time.Now())
	assert.ErrorContains(t, err, "limite de 3 meses")

	// Unsupported indexer
	_, err = bp.FetchRates(ctx, "INVALID_IDX", time.Now(), time.Now())
	assert.ErrorContains(t, err, "nao suportado")

	// Network error
	bp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return nil
	})
	_, err = bp.FetchRates(ctx, "IBOV", time.Now().AddDate(0, -1, 0), time.Now())
	assert.Error(t, err)

	// Status != 200
	bp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader("forbidden"))}
	})
	_, err = bp.FetchRates(ctx, "IBOV", time.Now().AddDate(0, -1, 0), time.Now())
	assert.Error(t, err)

	// Success
	brapiTS := time.Now().AddDate(0, 0, -5).Unix()
	brapiJSON := fmt.Sprintf(`{"results":[{"symbol":"IBOV","historicalDataPrice":[{"date":%d,"close":125000.0}]}]}`, brapiTS)
	bp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(brapiJSON))}
	})
	rates, err = bp.FetchRates(ctx, "IBOV", time.Now().AddDate(0, -1, 0), time.Now())
	assert.NoError(t, err)
	assert.Len(t, rates, 1)


	// 4. AnbimaClient and BCBClient HTTP error cases
	ac := NewAnbimaClient().(*anbimaClient)
	_, err = ac.FetchHolidays(nil, 2026) // nolint
	assert.Error(t, err)

	ac.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return nil
	})
	_, err = ac.FetchHolidays(ctx, 2026)
	assert.Error(t, err)

	ac.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("err"))}
	})
	_, err = ac.FetchHolidays(ctx, 2026)
	assert.Error(t, err)

	ac.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad-json"))}
	})
	_, err = ac.FetchHolidays(ctx, 2026)
	assert.Error(t, err)

	bc := NewBCBClient().(*bcbClient)
	_, err = bc.FetchRates(nil, "CDI", time.Now(), time.Now()) // nolint
	assert.Error(t, err)

	bc.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return nil
	})
	_, err = bc.FetchRates(ctx, "CDI", time.Now(), time.Now())
	assert.Error(t, err)

	bc.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("err"))}
	})
	_, err = bc.FetchRates(ctx, "CDI", time.Now(), time.Now())
	assert.Error(t, err)

	bc.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad-json"))}
	})
	_, err = bc.FetchRates(ctx, "CDI", time.Now(), time.Now())
	assert.Error(t, err)

	// BCBClient parse errors in data items
	bc.httpClient.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"data":"bad","valor":"1.0"},{"data":"01/01/2026","valor":"bad"}]`))}
	})
	rates, err = bc.FetchRates(ctx, "CDI", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Empty(t, rates)

	// BrapiProvider additional edge cases
	bpKey := &BrapiProvider{apiKey: "my-key", client: &http.Client{}}
	_, err = bpKey.FetchRates(nil, "IBOV", time.Now(), time.Now()) // nolint
	assert.Error(t, err)

	bpKey.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bad-json"))}
	})
	_, err = bpKey.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	bpKey.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":[{"symbol":"OTHER","historicalDataPrice":[]}]}`))}
	})
	_, err = bpKey.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.NoError(t, err)

	outDate := time.Now().AddDate(-1, 0, 0).Unix()
	bpKey.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"results":[{"symbol":"IBOV","historicalDataPrice":[{"date":%d,"close":100}]}]}`, outDate)))}
	})
	_, err = bpKey.FetchRates(ctx, "IBOV", time.Now().AddDate(0, -1, 0), time.Now())
	assert.NoError(t, err)

	// Yahoo additional edge cases
	_, err = yp.FetchRates(nil, "IBOV", time.Now(), time.Now()) // nolint
	assert.Error(t, err)

	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"chart":{"error":"some error"}}`))}
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"chart":{"result":[{"timestamp":[],"indicators":{"quote":[]}}]}}`))}
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	yp.client.Transport = RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"chart":{"result":[{"timestamp":[123,456],"indicators":{"quote":[{"close":[1.0]}]}}]}}`))}
	})
	_, err = yp.FetchRates(ctx, "IBOV", time.Now(), time.Now())
	assert.Error(t, err)

	// 5. IndexRegistry
	reg := NewIndexRegistry()
	_, err = reg.Fetch(ctx, "NOT_FOUND", time.Now(), time.Now())
	assert.ErrorContains(t, err, "não configurado")

	// Primary fails, fallback nil
	pFail := &mockIndexProvider{err: errors.New("p error")}
	reg.Register(IndexerConfig{Name: "P_FAIL", PrimaryProvider: pFail})
	_, err = reg.Fetch(ctx, "P_FAIL", time.Now(), time.Now())
	assert.Error(t, err)

	// Primary returns empty, fallback nil
	pEmpty := &mockIndexProvider{rates: []IndexRate{}}
	reg.Register(IndexerConfig{Name: "P_EMPTY", PrimaryProvider: pEmpty})
	_, err = reg.Fetch(ctx, "P_EMPTY", time.Now(), time.Now())
	assert.ErrorContains(t, err, "nenhum dado retornado")

	// Primary fails, fallback fails
	fbFail := &mockIndexProvider{err: errors.New("fb error")}
	reg.Register(IndexerConfig{Name: "ALL_FAIL", PrimaryProvider: pFail, FallbackProvider: fbFail})
	_, err = reg.Fetch(ctx, "ALL_FAIL", time.Now(), time.Now())
	assert.ErrorContains(t, err, "fallback também falhou")

	// Primary fails, fallback succeeds
	fbSucc := &mockIndexProvider{rates: []IndexRate{{Rate: 0.10}}}
	reg.Register(IndexerConfig{Name: "FB_SUCC", PrimaryProvider: pFail, FallbackProvider: fbSucc})
	rates, err = reg.Fetch(ctx, "FB_SUCC", time.Now(), time.Now())
	assert.NoError(t, err)
	assert.Len(t, rates, 1)

	// 6. AnbimaHolidayWorker syncYear edge cases
	mockRepoWorker := &MockFullRepo{}
	mockAnbima := &mockAnbimaClient{}
	ahWorker := NewAnbimaHolidayWorker(mockRepoWorker, mockAnbima)

	// Fetch error
	mockAnbima.On("FetchHolidays", ctx, 2024).Return(nil, errors.New("fetch err")).Once()
	err = ahWorker.syncYear(ctx, 2024)
	assert.Error(t, err)

	// Empty holidays
	mockAnbima.On("FetchHolidays", ctx, 2025).Return([]brasilAPIHoliday{}, nil).Once()
	err = ahWorker.syncYear(ctx, 2025)
	assert.NoError(t, err)

	// Non-national type, invalid date -> empty dates
	mockAnbima.On("FetchHolidays", ctx, 2026).Return([]brasilAPIHoliday{
		{Date: "2026-01-01", Type: "state"},
		{Date: "invalid-date", Type: "national"},
	}, nil).Once()
	err = ahWorker.syncYear(ctx, 2026)
	assert.NoError(t, err)
}

func TestService_CoverageExtra_Deep(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	// 1. getAssetPositionWithHistory with negative profit (PRE asset with Rate: -50)
	negAsset := Asset{
		ID:           "a_neg",
		PortfolioID:  "p1",
		Institution:  "Neg Bank",
		Type:         "CDB",
		DebtType:     "PRE",
		Rate:         -50.0,
		MaturityDate: time.Now().AddDate(1, 0, 0),
	}
	negTx := Transaction{
		ID:      "tx_neg",
		AssetID: "a_neg",
		Type:    "SUBSCRIPTION",
		Amount:  1000.0,
		Date:    time.Now().AddDate(0, -1, 0),
	}
	mockRepo.On("GetAssetByID", ctx, "a_neg").Return(&negAsset, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_neg").Return([]Transaction{negTx}, nil).Once()
	pos, err := svc.GetAssetPosition(ctx, "a_neg")
	assert.NoError(t, err)
	assert.NotNil(t, pos)
	assert.True(t, pos.GrossValue < pos.TotalInvested)

	// 2. GetPortfolioPerformance with period "1M"
	perfAsset := Asset{
		ID:          "a_perf",
		PortfolioID: "p1",
		Institution: "Perf Bank",
		Type:        "LCI",
		DebtType:    "PRE",
		Rate:        12.0,
	}
	perfTx := Transaction{
		ID:      "tx_perf",
		AssetID: "a_perf",
		Type:    "SUBSCRIPTION",
		Amount:  5000.0,
		Date:    time.Now().AddDate(0, -2, 0),
	}
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{perfAsset}, nil).Once()
	mockRepo.On("GetAssetByID", ctx, "a_perf").Return(&perfAsset, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_perf").Return([]Transaction{perfTx}, nil).Twice()
	perfRes, err := svc.GetPortfolioPerformance(ctx, "p1", "1M")
	assert.NoError(t, err)
	assert.NotNil(t, perfRes)

	// 3. calculateAssetMonthlyYields with past maturity, redemption, negative rate, and future tx
	subDate := time.Now().AddDate(0, -3, 0)
	redDate := time.Now().AddDate(0, -2, 0)
	futDate := time.Now().AddDate(0, 1, 0)

	assetMatured := Asset{
		ID:           "a_mat",
		PortfolioID:  "p1",
		Institution:  "Mat Bank",
		Type:         "CDB",
		DebtType:     "PRE",
		Rate:         10.0,
		MaturityDate: time.Now().AddDate(0, -1, 0), // past maturity
	}
	txsMat := []Transaction{
		{ID: "tx1", AssetID: "a_mat", Type: "SUBSCRIPTION", Amount: 2000.0, Date: subDate},
	}
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetMatured}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_mat").Return(txsMat, nil).Once()
	_, err = svc.CalculateMonthlyYields(ctx, "p1")
	assert.NoError(t, err)

	// Asset with redemption where withdrawalRatio > 1
	assetRedRatio := Asset{
		ID:           "a_red_ratio",
		PortfolioID:  "p1",
		Institution:  "Red Bank",
		Type:         "CDB",
		DebtType:     "PRE",
		Rate:         10.0,
		MaturityDate: time.Now().AddDate(1, 0, 0),
	}
	txsRedRatio := []Transaction{
		{ID: "tx_sub", AssetID: "a_red_ratio", Type: "SUBSCRIPTION", Amount: 1000.0, Date: subDate},
		{ID: "tx_red", AssetID: "a_red_ratio", Type: "REDEMPTION", Amount: 5000.0, Date: redDate}, // 5000 > 1000 => withdrawalRatio > 1
	}
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetRedRatio}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_red_ratio").Return(txsRedRatio, nil).Once()
	_, err = svc.CalculateMonthlyYields(ctx, "p1")
	assert.NoError(t, err)

	// Asset with negative rate yields (grossYield <= 0)
	assetNegRate := Asset{
		ID:          "a_neg_rate",
		PortfolioID: "p1",
		Type:        "CDB",
		DebtType:    "PRE",
		Rate:        -50.0,
	}
	txsNeg := []Transaction{
		{ID: "tx_neg1", AssetID: "a_neg_rate", Type: "SUBSCRIPTION", Amount: 1000.0, Date: subDate},
	}
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetNegRate}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_neg_rate").Return(txsNeg, nil).Once()
	_, err = svc.CalculateMonthlyYields(ctx, "p1")
	assert.NoError(t, err)

	// Asset with future start date (daysHeld < 0)
	assetFut := Asset{
		ID:          "a_fut",
		PortfolioID: "p1",
		Type:        "CDB",
		DebtType:    "PRE",
		Rate:        10.0,
	}
	txsFut := []Transaction{
		{ID: "tx_fut", AssetID: "a_fut", Type: "SUBSCRIPTION", Amount: 1000.0, Date: futDate},
	}
	mockRepo.On("GetAssetsByPortfolio", ctx, "p1").Return([]Asset{assetFut}, nil).Once()
	mockRepo.On("GetTransactionsByAsset", ctx, "a_fut").Return(txsFut, nil).Once()
	_, err = svc.CalculateMonthlyYields(ctx, "p1")
	assert.NoError(t, err)
}

func TestTreasury_PositionsAndPerformance_Deep(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)

	// 1. GetTreasuryPositions error from GetAnbimaHolidays
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p_pos_err1").Return([]TreasuryTransaction{{AssetID: "a1"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(nil, errors.New("holidays err")).Once()
	_, err := svc.GetTreasuryPositions(ctx, "p_pos_err1")
	assert.ErrorContains(t, err, "holidays err")

	// 2. GetTreasuryPositions error from GetTreasuryAssetDetails
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p_pos_err2").Return([]TreasuryTransaction{{AssetID: "a_bad"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_bad").Return("", "", time.Time{}, false, errors.New("asset details err")).Once()
	_, err = svc.GetTreasuryPositions(ctx, "p_pos_err2")
	assert.ErrorContains(t, err, "asset details err")

	// 3. GetTreasuryPositions empty lots (positions == nil)
	mockRepo.On("GetActiveSubscriptionLots", ctx, "p_pos_empty").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	posEmpty, err := svc.GetTreasuryPositions(ctx, "p_pos_empty")
	assert.NoError(t, err)
	assert.Empty(t, posEmpty)

	// 4. GetTreasuryPositions branches
	maturedDate := time.Now().AddDate(0, -1, 0)
	futureDate := time.Now().AddDate(2, 0, 0)
	lotSelicBig := TreasuryTransaction{
		ID:                "lot_s_big",
		AssetID:           "a_s_big",
		RemainingQuantity: 20.0,
		UnitPrice:         1000.0,
		ContractedRate:    0.0,
		TransactionDate:   time.Now().AddDate(0, -2, 0),
	}
	lotPreNeg := TreasuryTransaction{
		ID:                "lot_p_neg",
		AssetID:           "a_p_neg",
		RemainingQuantity: 5.0,
		UnitPrice:         1000.0,
		ContractedRate:    -100.0, // negative gross yield
		TransactionDate:   time.Now().AddDate(0, -1, 0),
	}
	lotMatured := TreasuryTransaction{
		ID:                "lot_mat",
		AssetID:           "a_mat",
		RemainingQuantity: 1.0,
		UnitPrice:         1000.0,
		ContractedRate:    10.0,
		TransactionDate:   time.Now().AddDate(-1, 0, 0),
	}

	mockRepo.On("GetActiveSubscriptionLots", ctx, "p_pos_multi").Return([]TreasuryTransaction{lotSelicBig, lotPreNeg, lotMatured}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_s_big").Return("LFT", "SELIC", futureDate, false, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_p_neg").Return("LTN", "PREFIXADO", futureDate, false, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_mat").Return("NTN-F", "PREFIXADO", maturedDate, true, nil).Once()

	positions, err := svc.GetTreasuryPositions(ctx, "p_pos_multi")
	assert.NoError(t, err)
	assert.Len(t, positions, 3)

	// 5. GetTreasuryTransactions nil list -> empty slice
	mockRepo.On("GetTreasuryTransactionsList", ctx, "p_tx_nil").Return(nil, nil).Once()
	txsNil, err := svc.GetTreasuryTransactions(ctx, "p_tx_nil")
	assert.NoError(t, err)
	assert.Empty(t, txsNil)

	// 6. GetTreasuryPerformance error branches
	mockRepo.On("GetTreasuryTransactionsList", ctx, "p_perf_err").Return(nil, errors.New("perf tx list err")).Once()
	_, err = svc.GetTreasuryPerformance(ctx, "p_perf_err")
	assert.ErrorContains(t, err, "perf tx list err")

	mockRepo.On("GetTreasuryTransactionsList", ctx, "p_perf_empty").Return([]TreasuryTxRequest{}, nil).Once()
	ptsEmpty, err := svc.GetTreasuryPerformance(ctx, "p_perf_empty")
	assert.NoError(t, err)
	assert.Empty(t, ptsEmpty)

	mockRepo.On("GetTreasuryTransactionsList", ctx, "p_perf_date_err").Return([]TreasuryTxRequest{
		{TransactionDate: "bad-date"},
	}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(nil, errors.New("hol err")).Once()
	mockRepo.On("GetSelicRates", ctx).Return(nil, errors.New("selic err")).Once()
	_, err = svc.GetTreasuryPerformance(ctx, "p_perf_date_err")
	assert.ErrorContains(t, err, "failed to parse start date")

	// Full GetTreasuryPerformance simulation using fixed past business days (2024-01-08 was Monday, 09 Tuesday, 10 Wednesday)
	day1 := "2024-01-08"
	day2 := "2024-01-09"
	day3 := "2024-01-10"
	matPast := "2024-01-09"

	perfTxs := []TreasuryTxRequest{
		{
			Ticker:          "LFT",
			TreasuryType:    "SELIC",
			MaturityDate:    futureDate.Format("2006-01-02"),
			TransactionDate: day1,
			Type:            "SUBSCRIPTION",
			Quantity:        10.0,
			UnitPrice:       1000.0,
			ContractedRate:  0.0,
		},
		{
			Ticker:          "LTN_MAT",
			TreasuryType:    "PREFIXADO",
			MaturityDate:    matPast,
			TransactionDate: day1,
			Type:            "SUBSCRIPTION",
			Quantity:        5.0,
			UnitPrice:       800.0,
			ContractedRate:  10.0,
		},
		{
			Ticker:          "LFT",
			TreasuryType:    "SELIC",
			TransactionDate: day2,
			Type:            "REDEMPTION",
			Quantity:        4.0,
			UnitPrice:       1000.0,
		},
		{
			Ticker:          "LTN_MAT",
			TreasuryType:    "PREFIXADO",
			TransactionDate: day3,
			Type:            "REDEMPTION",
			Quantity:        10.0,
			UnitPrice:       800.0,
		},
	}
	mockRepo.On("GetTreasuryTransactionsList", ctx, "p_perf_full").Return(perfTxs, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{day1: 11.25, day2: 11.25, day3: 11.25}, nil).Once()

	points, err := svc.GetTreasuryPerformance(ctx, "p_perf_full")
	assert.NoError(t, err)
	assert.NotEmpty(t, points)
}

func TestTreasury_UpdateDeleteRebuild_Deep(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockFullRepo{}
	svc := NewService(mockRepo, nil)
	sImpl := svc.(*service)

	// 1. CreateTreasuryTransaction: subscription error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("CreateTreasurySubscription", ctx, mock.Anything, "p1", "a_lft", 1.0, 1000.0, 0.0, mock.Anything).Return("", errors.New("sub err")).Once()
	_, err := svc.CreateTreasuryTransaction(ctx, "p1", &TreasuryTxRequest{Ticker: "LFT", Type: "SUBSCRIPTION", MaturityDate: "2029-01-01", TransactionDate: "2026-01-01", Quantity: 1.0, UnitPrice: 1000.0})
	assert.ErrorContains(t, err, "sub err")

	// CreateTreasuryTransaction: new asset creation error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "NEW_T").Return("", pgx.ErrNoRows).Once()
	mockRepo.On("CreateTreasuryAsset", ctx, mock.Anything, "NEW_T", "NEW_T", "SELIC", mock.Anything, false).Return("", errors.New("asset create err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", &TreasuryTxRequest{Ticker: "NEW_T", TreasuryType: "SELIC", Type: "SUBSCRIPTION", MaturityDate: "2029-01-01", TransactionDate: "2026-01-01"})
	assert.ErrorContains(t, err, "asset create err")

	// CreateTreasuryTransaction: redemption with 2 lots where first lot covers quantity (triggers remainingToRedeem <= 0 break)
	lotA := TreasuryTransaction{ID: "lot_a", RemainingQuantity: 10.0, UnitPrice: 1000.0, ContractedRate: 0.0, TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	lotB := TreasuryTransaction{ID: "lot_b", RemainingQuantity: 10.0, UnitPrice: 1000.0, ContractedRate: 0.0, TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{lotA, lotB}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_lft", 5.0, 1000.0, 0.0, mock.Anything).Return("red_brk", nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "lot_a", 5.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "lot_a", "red_brk", 5.0).Return(nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "red_brk", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", &TreasuryTxRequest{Ticker: "LFT", TreasuryType: "SELIC", Type: "REDEMPTION", Quantity: 5.0, UnitPrice: 1000.0, MaturityDate: "2029-01-01", TransactionDate: "2026-01-01"})
	assert.NoError(t, err)

	// Redemption error branches
	reqRed := &TreasuryTxRequest{
		Ticker:          "LFT",
		TreasuryType:    "SELIC",
		MaturityDate:    "2029-01-01",
		TransactionDate: "2026-01-01",
		Type:            "REDEMPTION",
		Quantity:        2.0,
		UnitPrice:       1000.0,
	}

	// 1a. GetActiveLotsForAsset error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return(nil, errors.New("active lots err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "active lots err")

	// 1b. GetAnbimaHolidays error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(nil, errors.New("hol err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "hol err")

	// 1c. GetSelicRates error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(nil, errors.New("selic err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "selic err")

	// 1d. GetTotalSelicInvested error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, errors.New("tot selic err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "tot selic err")

	// 1e. CreateTreasuryRedemptionPlaceholder error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_lft", 2.0, 1000.0, 0.0, mock.Anything).Return("", errors.New("placeholder err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "placeholder err")

	// 1f. CreateDepletionLink error
	lotItem := TreasuryTransaction{
		ID:                "lot1",
		AssetID:           "a_lft",
		RemainingQuantity: 10.0,
		UnitPrice:         1000.0,
		ContractedRate:    0.0,
		TransactionDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{lotItem}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_lft", 2.0, 1000.0, 0.0, mock.Anything).Return("red1", nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "lot1", 8.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "lot1", "red1", 2.0).Return(errors.New("depletion err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "depletion err")

	// 1g. UpdateRedemptionFinancials error
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_lft", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_lft").Return([]TreasuryTransaction{lotItem}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_lft", 2.0, 1000.0, 0.0, mock.Anything).Return("red1", nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "lot1", 8.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "lot1", "red1", 2.0).Return(nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "red1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("financials err")).Once()
	_, err = svc.CreateTreasuryTransaction(ctx, "p1", reqRed)
	assert.ErrorContains(t, err, "financials err")

	// 1h. Non-SELIC redemption (PREFIXADO) with grossYield < 0
	reqRedPre := &TreasuryTxRequest{
		Ticker:          "LTN",
		TreasuryType:    "PREFIXADO",
		MaturityDate:    "2029-01-01",
		TransactionDate: "2026-01-01",
		Type:            "REDEMPTION",
		Quantity:        2.0,
		UnitPrice:       800.0,
		ContractedRate:  10.0,
	}
	lotPre := TreasuryTransaction{
		ID:                "lot_pre",
		AssetID:           "a_pre",
		RemainingQuantity: 1.0,
		UnitPrice:         800.0,
		ContractedRate:    -100.0, // negative gross yield
		TransactionDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LTN").Return("a_pre", nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_pre").Return([]TreasuryTransaction{lotPre}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("CreateTreasuryRedemptionPlaceholder", ctx, mock.Anything, "p1", "a_pre", 2.0, 800.0, 10.0, mock.Anything).Return("red_pre", nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "lot_pre", 0.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "lot_pre", "red_pre", 1.0).Return(nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "red_pre", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	resPre, err := svc.CreateTreasuryTransaction(ctx, "p1", reqRedPre)
	assert.NoError(t, err)
	assert.NotNil(t, resPre)

	// 2. UpdateTreasuryTransaction branches
	existingTx := &TreasuryTransaction{ID: "tx_up1", PortfolioID: "p1", AssetID: "a_old"}
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_up1").Return(existingTx, nil).Once()
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "NEW_TICKER").Return("", pgx.ErrNoRows).Once()
	mockRepo.On("CreateTreasuryAsset", ctx, mock.Anything, "NEW_TICKER", "NEW_TICKER", "SELIC", mock.Anything, false).Return("", errors.New("create asset err")).Once()
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx_up1", &TreasuryTxRequest{
		Ticker:          "NEW_TICKER",
		TreasuryType:    "SELIC",
		MaturityDate:    "2029-01-01",
		TransactionDate: "2026-01-01",
		Type:            "REDEMPTION",
	})
	assert.ErrorContains(t, err, "create asset err")

	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_up2").Return(existingTx, nil).Once()
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "ERR_TICKER").Return("", errors.New("ticker lookup err")).Once()
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx_up2", &TreasuryTxRequest{
		Ticker:          "ERR_TICKER",
		TreasuryType:    "SELIC",
		MaturityDate:    "2029-01-01",
		TransactionDate: "2026-01-01",
		Type:            "REDEMPTION",
	})
	assert.ErrorContains(t, err, "ticker lookup err")

	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_up3").Return(existingTx, nil).Once()
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LFT").Return("a_old", nil).Once()
	mockRepo.On("UpdateTreasuryTransaction", ctx, mock.Anything, mock.Anything).Return(errors.New("update tx err")).Once()
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx_up3", &TreasuryTxRequest{
		Ticker:          "LFT",
		TreasuryType:    "SELIC",
		MaturityDate:    "2029-01-01",
		TransactionDate: "2026-01-01",
		Type:            "REDEMPTION",
	})
	assert.ErrorContains(t, err, "update tx err")

	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_up4").Return(existingTx, nil).Once()
	mockRepo.On("GetTreasuryAssetByTicker", ctx, mock.Anything, "LTN").Return("a_new", nil).Once()
	mockRepo.On("UpdateTreasuryTransaction", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_old").Return(errors.New("rebuild old err")).Once()
	err = svc.UpdateTreasuryTransaction(ctx, "p1", "tx_up4", &TreasuryTxRequest{
		Ticker:          "LTN",
		TreasuryType:    "PREFIXADO",
		MaturityDate:    "2029-01-01",
		TransactionDate: "2026-01-01",
		Type:            "SUBSCRIPTION",
	})
	assert.ErrorContains(t, err, "rebuild old err")

	// 3. DeleteTreasuryTransaction DeleteTreasuryTransactionByID error
	mockRepo.On("GetTreasuryTransactionByID", ctx, mock.Anything, "tx_del_err").Return(existingTx, nil).Once()
	mockRepo.On("DeleteTreasuryTransactionByID", ctx, mock.Anything, "tx_del_err").Return(errors.New("del id err")).Once()
	err = svc.DeleteTreasuryTransaction(ctx, "p1", "tx_del_err")
	assert.ErrorContains(t, err, "del id err")

	// 4. rebuildTreasuryFIFO error branches
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo1").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo1").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo1").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo1").Return([]TreasuryTransaction{{ID: "r1"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(nil, errors.New("fifo hol err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo1")
	assert.ErrorContains(t, err, "fifo hol err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo2").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo2").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo2").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo2").Return([]TreasuryTransaction{{ID: "r1"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(nil, errors.New("fifo selic err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo2")
	assert.ErrorContains(t, err, "fifo selic err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo3").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo3").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo3").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo3").Return([]TreasuryTransaction{{ID: "r1"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, errors.New("fifo total selic err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo3")
	assert.ErrorContains(t, err, "fifo total selic err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo4").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo4").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo4").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo4").Return([]TreasuryTransaction{{ID: "r1"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_fifo4").Return("", "", time.Time{}, false, errors.New("fifo asset details err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo4")
	assert.ErrorContains(t, err, "fifo asset details err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo5").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo5").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo5").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo5").Return([]TreasuryTransaction{{ID: "r1"}}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_fifo5").Return("LFT", "SELIC", time.Now(), false, nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_fifo5").Return(nil, errors.New("fifo active lots err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo5")
	assert.ErrorContains(t, err, "fifo active lots err")

	redTx := TreasuryTransaction{ID: "r1", Quantity: 2.0, TransactionDate: time.Date(2025, 6, 6, 0, 0, 0, 0, time.UTC)}
	lotTx := TreasuryTransaction{ID: "l1", RemainingQuantity: 5.0, UnitPrice: 1000.0, ContractedRate: 0.0, TransactionDate: time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)}
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo6").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo6").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo6").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo6").Return([]TreasuryTransaction{redTx}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{"2025-06-03": 11.0}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(15000.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_fifo6").Return("LFT", "SELIC", time.Now().AddDate(1, 0, 0), false, nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_fifo6").Return([]TreasuryTransaction{lotTx}, nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "l1", 3.0).Return(errors.New("update lot rem err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo6")
	assert.ErrorContains(t, err, "update lot rem err")

	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo7").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo7").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo7").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo7").Return([]TreasuryTransaction{redTx}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{"2025-06-03": 11.25}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_fifo7").Return("LFT", "SELIC", time.Now().AddDate(1, 0, 0), false, nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_fifo7").Return([]TreasuryTransaction{lotTx}, nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "l1", 3.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "l1", "r1", 2.0).Return(errors.New("create depletion err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo7")
	assert.ErrorContains(t, err, "create depletion err")

	// Non-SELIC full rebuild with UpdateRedemptionFinancials error
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo8").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo8").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo8").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo8").Return([]TreasuryTransaction{redTx}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_fifo8").Return("LTN", "PREFIXADO", time.Now().AddDate(1, 0, 0), false, nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_fifo8").Return([]TreasuryTransaction{lotTx}, nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "l1", 3.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "l1", "r1", 2.0).Return(nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "r1", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("update fin err")).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo8")
	assert.ErrorContains(t, err, "update fin err")

	// Non-SELIC rebuild with negative contracted rate (triggers grossYield < 0) and 2 lots (triggers remainingToRedeem <= 0 break)
	redTxBrk := TreasuryTransaction{ID: "r_brk", Quantity: 2.0, TransactionDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	lot1Brk := TreasuryTransaction{ID: "l1_b", RemainingQuantity: 5.0, UnitPrice: 800.0, ContractedRate: -100.0, TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	lot2Brk := TreasuryTransaction{ID: "l2_b", RemainingQuantity: 5.0, UnitPrice: 800.0, ContractedRate: 0.0, TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	mockRepo.On("DeleteDepletionsByAsset", ctx, mock.Anything, "p1", "a_fifo_brk").Return(nil).Once()
	mockRepo.On("ResetSubscriptionsRemainingQuantity", ctx, mock.Anything, "p1", "a_fifo_brk").Return(nil).Once()
	mockRepo.On("ResetRedemptionFinancials", ctx, mock.Anything, "p1", "a_fifo_brk").Return(nil).Once()
	mockRepo.On("GetRedemptionsForAsset", ctx, mock.Anything, "p1", "a_fifo_brk").Return([]TreasuryTransaction{redTxBrk}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{}, nil).Once()
	mockRepo.On("GetTotalSelicInvested", ctx, mock.Anything, "p1").Return(0.0, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_fifo_brk").Return("LTN", "PREFIXADO", time.Now().AddDate(1, 0, 0), false, nil).Once()
	mockRepo.On("GetActiveLotsForAsset", ctx, mock.Anything, "p1", "a_fifo_brk").Return([]TreasuryTransaction{lot1Brk, lot2Brk}, nil).Once()
	mockRepo.On("UpdateLotRemainingQuantity", ctx, mock.Anything, "l1_b", 3.0).Return(nil).Once()
	mockRepo.On("CreateDepletionLink", ctx, mock.Anything, "l1_b", "r_brk", 2.0).Return(nil).Once()
	mockRepo.On("UpdateRedemptionFinancials", ctx, mock.Anything, "r_brk", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	err = sImpl.rebuildTreasuryFIFO(ctx, nil, "p1", "a_fifo_brk")
	assert.NoError(t, err)

	// GetTreasuryMonthlyYields deep branches:
	// - lot 1: GetTreasuryAssetDetails error (triggers continue)
	// - lot 2: past maturity date (triggers limitDate = maturityDate)
	// - lot 3: SELIC lot with rate in selicRates (triggers rate = rVal)
	// - lot 4: negative rate (triggers grossYield <= 0 { continue })
	lotErrDet := TreasuryTransaction{ID: "l_err", AssetID: "a_err_det", RemainingQuantity: 1.0, UnitPrice: 1000.0, TransactionDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lotPastMat := TreasuryTransaction{ID: "l_mat", AssetID: "a_mat_det", RemainingQuantity: 1.0, UnitPrice: 1000.0, ContractedRate: 10.0, TransactionDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lotSelicRate := TreasuryTransaction{ID: "l_sel", AssetID: "a_sel_det", RemainingQuantity: 1.0, UnitPrice: 1000.0, ContractedRate: 0.0, TransactionDate: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)}
	lotNegRate := TreasuryTransaction{ID: "l_neg", AssetID: "a_neg_det", RemainingQuantity: 1.0, UnitPrice: 1000.0, ContractedRate: -100.0, TransactionDate: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)}

	mockRepo.On("GetActiveSubscriptionLots", ctx, "p_yields_deep").Return([]TreasuryTransaction{lotErrDet, lotPastMat, lotSelicRate, lotNegRate}, nil).Once()
	mockRepo.On("GetAnbimaHolidays", ctx).Return(map[string]bool{}, nil).Once()
	mockRepo.On("GetSelicRates", ctx).Return(map[string]float64{"2024-01-09": 11.5}, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_err_det").Return("", "", time.Time{}, false, errors.New("det err")).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_mat_det").Return("LTN", "PREFIXADO", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), false, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_sel_det").Return("LFT", "SELIC", time.Now().AddDate(2, 0, 0), false, nil).Once()
	mockRepo.On("GetTreasuryAssetDetails", ctx, "a_neg_det").Return("LTN", "PREFIXADO", time.Now().AddDate(2, 0, 0), false, nil).Once()

	yRes, err := svc.GetTreasuryMonthlyYields(ctx, "p_yields_deep")
	assert.NoError(t, err)
	assert.NotEmpty(t, yRes)
}

type mockIndexProviderRoundTrip struct {
	onFetch func()
}

func (m *mockIndexProviderRoundTrip) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	if m.onFetch != nil {
		m.onFetch()
	}
	return []IndexRate{{Rate: 0.10, Date: startDate}}, nil
}

func TestWorker_CoverageExtra_Deep(t *testing.T) {
	ctx := context.Background()

	// 1. Worker.SyncRates with weekend rate filtering
	mockRepoWorkerRates := &MockFullRepo{}
	regWorker := NewIndexRegistry()
	satDate := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) // Saturday
	pSat := &mockIndexProvider{rates: []IndexRate{{Indexer: "CDI", Rate: 0.10, Date: satDate}}}
	regWorker.Register(IndexerConfig{Name: "CDI", PrimaryProvider: pSat})
	wRates := NewWorker(mockRepoWorkerRates, regWorker)
	mockRepoWorkerRates.On("GetLatestIndexRate", mock.Anything, "CDI").Return(&IndexRate{Date: time.Now().AddDate(0, -1, 0)}, nil).Once()
	mockRepoWorkerRates.On("GetLatestIndexRate", mock.Anything, mock.Anything).Return(&IndexRate{Date: time.Now().AddDate(0, 1, 0)}, nil)
	wRates.SyncRates(ctx)

	// 2. AnbimaHolidayWorker SyncHolidays syncYear error and context done in sleep
	mockRepoAnb := &MockFullRepo{}
	mockClientAnb := &mockAnbimaClient{}
	ahw := NewAnbimaHolidayWorker(mockRepoAnb, mockClientAnb)
	currYear := time.Now().Year()
	ahw.startYear = currYear
	mockRepoAnb.On("GetSeededHolidayYears", mock.Anything).Return([]int{}, nil).Once()
	mockClientAnb.On("FetchHolidays", mock.Anything, currYear).Return(nil, errors.New("sync year err")).Once()
	mockClientAnb.On("FetchHolidays", mock.Anything, currYear+1).Return([]brasilAPIHoliday{}, nil).Once()
	ctxDone, cancelDone := context.WithCancel(context.Background())
	cancelDone()
	ahw.SyncHolidays(ctxDone)

	// 3. Worker.SyncRates inner loop cancellation via Done()
	mockRepoWorkerCancel := &MockFullRepo{}
	regWorkerCancel := NewIndexRegistry()
	ctxCancelable, cancelFunc := context.WithCancel(context.Background())
	pCancel := &mockIndexProviderRoundTrip{onFetch: func() { cancelFunc() }}
	regWorkerCancel.Register(IndexerConfig{Name: "CDI", PrimaryProvider: pCancel})
	wCancel := NewWorker(mockRepoWorkerCancel, regWorkerCancel)
	mockRepoWorkerCancel.On("GetLatestIndexRate", mock.Anything, "CDI").Return(&IndexRate{Date: time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)}, nil).Once()
	mockRepoWorkerCancel.On("SaveIndexRates", mock.Anything, mock.Anything).Return(nil).Maybe()
	wCancel.SyncRates(ctxCancelable)
}


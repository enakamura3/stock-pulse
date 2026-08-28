package portfolio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockHTTPTransport struct {
	Err error
}

func (m *MockHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, m.Err
}

func TestServiceCoverage_DetermineAssetType(t *testing.T) {
	assert.Equal(t, "ETF_BR", determineAssetType("SPYI11.SA", "SPYI", "BRL"))
}

func TestServiceCoverage_GetPortfolioDividends_ErrorAssetEvents(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, ExecutedAt: time.Now().AddDate(-1, 0, 0), Currency: "USD", AssetType: "STOCK_US"},
	}, nil)
	repo.On("GetAssetEvents", mock.Anything, "a1").Return(([]AssetEvent)(nil), errors.New("events err"))

	divs, err := s.GetPortfolioDividends(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.Empty(t, divs)
}

func TestServiceCoverage_GetPortfolioDividends_Calculation(t *testing.T) {
	s, repo, ms, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, ExecutedAt: time.Now().AddDate(0, 0, -10), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "SELL", Quantity: 2, ExecutedAt: time.Now().AddDate(0, 0, -9), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "SPLIT", Quantity: 2, ExecutedAt: time.Now().AddDate(0, 0, -8), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "REVERSE_SPLIT", Quantity: 2, ExecutedAt: time.Now().AddDate(0, 0, -7), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "BONUS", Quantity: 1, ExecutedAt: time.Now().AddDate(0, 0, -6), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 5, ExecutedAt: time.Now().AddDate(0, 0, 5), Currency: "USD"}, // After CumDate
	}, nil)

	repo.On("GetAssetEvents", mock.Anything, "a1").Return([]AssetEvent{
		{Type: "DIVIDEND", GrossAmount: 2, PaymentDate: time.Now(), CumDate: time.Now()},
	}, nil)

	ms.On("GetHistoricalExchangeRate", mock.Anything, mock.Anything).Return(0.0, errors.New("err"))
	ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(&market.Quote{Price: 5.5}, nil).Maybe()
	repo.On("GetExchangeRateByDate", mock.Anything, "USDBRL", mock.Anything).Return(1.0, nil)

	divs, err := s.GetPortfolioDividends(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.NotEmpty(t, divs)
}

func TestServiceCoverage_GetPortfolioDividends_BRL_Taxes(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "BOVA11", Type: "BUY", Quantity: 10, ExecutedAt: time.Now().AddDate(0, 0, -10), Currency: "BRL", AssetType: "ETF_BR"},
		{AssetID: "a2", Ticker: "PETR4", Type: "BUY", Quantity: 10, ExecutedAt: time.Now().AddDate(0, 0, -10), Currency: "BRL", AssetType: "STOCK_BR"},
	}, nil)

	repo.On("GetAssetEvents", mock.Anything, "a1").Return([]AssetEvent{
		{Type: "DIVIDEND", GrossAmount: 2, PaymentDate: time.Now(), CumDate: time.Now()},
	}, nil)
	repo.On("GetAssetEvents", mock.Anything, "a2").Return([]AssetEvent{
		{Type: "JCP", GrossAmount: 1, PaymentDate: time.Now(), CumDate: time.Now()},
		{Type: "DIVIDEND", GrossAmount: 2, PaymentDate: time.Now(), CumDate: time.Now()},
	}, nil)

	divs, err := s.GetPortfolioDividends(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.NotEmpty(t, divs)
}

func TestServiceCoverage_GetPortfolioDividends_FallbackError(t *testing.T) {
	s, repo, ms, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, ExecutedAt: time.Now().AddDate(0, 0, -10), Currency: "USD"},
	}, nil)
	repo.On("GetAssetEvents", mock.Anything, "a1").Return([]AssetEvent{
		{Type: "DIVIDEND", GrossAmount: 2, PaymentDate: time.Now(), CumDate: time.Now()},
	}, nil)
	ms.On("GetHistoricalExchangeRate", mock.Anything, mock.Anything).Return(0.0, errors.New("err"))
	ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(nil, errors.New("quote err")).Maybe()
	repo.On("GetExchangeRateByDate", mock.Anything, "USDBRL", mock.Anything).Return(0.0, errors.New("err"))

	_, _ = s.GetPortfolioDividends(context.Background(), "p1", "u1")
}

func TestServiceCoverage_GetPortfolioPerformance_SplitsAndExchange(t *testing.T) {
	s, repo, ms, _ := setupServiceTest()
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 100, ExecutedAt: time.Now().AddDate(0, 0, -20), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "SPLIT", Quantity: 2, ExecutedAt: time.Now().AddDate(0, 0, -15), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "REVERSE_SPLIT", Quantity: 2, ExecutedAt: time.Now().AddDate(0, 0, -10), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "SELL", Quantity: 50, ExecutedAt: time.Now().AddDate(0, 0, -5), Currency: "USD"},
	}, nil)
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Now().AddDate(0, 0, -30), nil)

	// One point inside 1M, one point outside 1M
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
		{PriceDate: time.Now().AddDate(0, -2, 0), ClosePrice: 110.0},
	}, nil)

	// Exchange success
	ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(&market.Quote{Price: 5.5}, nil)
	ms.On("GetHistoricalExchangeRate", mock.Anything, mock.Anything).Return(0.0, errors.New("fx err"))
	repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usdbrl-id", nil)
	repo.On("GetDailyPrices", mock.Anything, "usdbrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)

	perf, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", []string{"AAPL", "MSFT"})
	assert.NoError(t, err)
	assert.NotNil(t, perf)
}

// test removed as it cannot be trivially empty

func TestServiceCoverage_GetPortfolioDetails_Transactions(t *testing.T) {
	s, repo, ms, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 100, ExchangeRate: 1, ExecutedAt: time.Now(), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 5, UnitPrice: 110, ExchangeRate: 1, ExecutedAt: time.Now().Add(time.Minute), Currency: "USD"}, // second buy!
		{AssetID: "a1", Ticker: "AAPL", Type: "SPLIT", Quantity: 2, UnitPrice: 0, ExchangeRate: 1, ExecutedAt: time.Now().Add(time.Hour), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "REVERSE_SPLIT", Quantity: 2, UnitPrice: 0, ExchangeRate: 1, ExecutedAt: time.Now().Add(2 * time.Hour), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "BONUS", Quantity: 1, UnitPrice: 0, ExchangeRate: 1, ExecutedAt: time.Now().Add(3 * time.Hour), Currency: "USD"},
	}, nil)
	repo.On("GetAssetEvents", mock.Anything, "a1").Return([]AssetEvent{
		{Type: "JCP", GrossAmount: 1, PaymentDate: time.Now(), CumDate: time.Now().AddDate(0, 1, 0)},
		{Type: "DIVIDEND", GrossAmount: 2, PaymentDate: time.Now(), CumDate: time.Now().AddDate(0, 1, 0)},
		{Type: "AMORTIZATION", GrossAmount: 3, PaymentDate: time.Now(), CumDate: time.Now().AddDate(0, 1, 0)},
		{Type: "YIELD", GrossAmount: 4, PaymentDate: time.Now(), CumDate: time.Now().AddDate(0, 1, 0)},
	}, nil)
	repo.On("GetLatestPrices", mock.Anything, []string{"a1"}).Return(map[string]float64{"a1": 150.0}, nil)
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return(nil, errors.New("ignored"))
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Time{}, errors.New("err"))
	ms.On("GetFundamentals", mock.Anything, "AAPL").Return(&market.Fundamentals{BookValue: 10, EPS: 5}, nil)
	ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 150.0}, nil)

	// To hit `if div.Type == "JCP"` branch we need GetPortfolioDetails to calculate dividends.
	repo.On("GetPortfolioByID", mock.Anything, "p2", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p2", "u1").Return([]Transaction{
		{AssetID: "a2", Ticker: "SPY", Type: "BUY", Quantity: 10, UnitPrice: 100, AssetType: "ETF_US", ExchangeRate: 1.0, Currency: "USD"},
		{AssetID: "a2", Ticker: "SPY", Type: "SELL", Quantity: 20, UnitPrice: 100, AssetType: "ETF_US", ExchangeRate: 1.0, Currency: "USD"},
	}, nil)
	repo.On("GetAssetEvents", mock.Anything, "a2").Return([]AssetEvent{}, nil)
	repo.On("GetLatestPrices", mock.Anything, []string{"a2"}).Return(map[string]float64{"a2": 150.0}, nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a2").Return(time.Time{}, errors.New("err"))
	repo.On("GetDailyPrices", mock.Anything, "a2", mock.Anything, mock.Anything).Return(nil, errors.New("ignored"))

	_, _, err := s.GetPortfolioDetails(context.Background(), "p1", "u1")
	assert.NoError(t, err)

	_, _, err = s.GetPortfolioDetails(context.Background(), "p2", "u1")
	assert.NoError(t, err)

	ms.AssertExpectations(t)
}

func TestServiceCoverage_AddTransaction_Fallback(t *testing.T) {
	s, repo, _, mp := setupServiceTest()

	// Simulate HTTP failure for BackfillHistoricalPrices
	s.httpClient = &http.Client{
		Transport: &MockHTTPTransport{Err: errors.New("http err")},
	}
	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "NEW-USD").Return("", "", pgx.ErrNoRows)

	mp.On("SearchAssets", mock.Anything, "NEW-USD").Return([]market.SearchResult{}, nil)
	mp.On("GetQuote", mock.Anything, "NEW-USD").Return(&market.Quote{Currency: "USD", Name: "New Coin"}, nil)
	repo.On("CreateAsset", mock.Anything, "NEW-USD", "New Coin", "CRYPTO", "USD").Return("new-a", nil)

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("CreateTransaction", mock.Anything, mock.Anything).Return(&Transaction{ID: "tx1"}, nil)
	// Return oldest date in future so executedAt is BEFORE oldestDate!
	repo.On("GetOldestPriceDate", mock.Anything, "new-a").Return(time.Now().Add(24*time.Hour), nil)
	repo.On("GetDailyPrices", mock.Anything, "new-a", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "new-a", mock.Anything).Return(nil)
	repo.On("GetExchangeRateByDate", mock.Anything, mock.Anything, mock.Anything).Return(0.0, errors.New("err")).Once()
	repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("", errors.New("err"))
	repo.On("CreateAsset", mock.Anything, "USDBRL=X", mock.Anything, "CURRENCY", "BRL").Return("usd-brl-id", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "usd-brl-id").Return(time.Time{}, errors.New("err"))
	repo.On("GetDailyPrices", mock.Anything, "usd-brl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "usd-brl-id", mock.Anything).Return(nil)
	repo.On("GetExchangeRateByDate", mock.Anything, mock.Anything, mock.Anything).Return(1.5, nil).Once()

	tx := &Transaction{PortfolioID: "p1", Ticker: "NEW-USD", Type: "BUY", Quantity: 10, UnitPrice: 100, ExecutedAt: time.Now()}
	tx, err := s.AddTransaction(context.Background(), "u1", tx)
	time.Sleep(200 * time.Millisecond)
	assert.NoError(t, err)
}

func TestServiceCoverage_BackfillPrices(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("SaveDailyPrices", mock.Anything, "a1", mock.Anything).Return(errors.New("save err"))
	_ = s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"chart": {
				"result": [{
					"timestamp": [1609459200, 1609545600],
					"indicators": {
						"quote": [{
							"close": [150.0, null]
						}]
					}
				}]
			}
		}`))
	}))
	defer server.Close()

	s.httpClient.Transport = &mockTransport{serverURL: server.URL}
	repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("a1", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Now(), nil)

	_ = s.BackfillGap(context.Background(), "AAPL", time.Now().AddDate(0, 0, -10))
	_ = s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
}

func TestServiceCoverage_BackfillGap_Errors(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetAssetByTicker", mock.Anything, "INVALID").Return("", errors.New("err"))
	repo.On("CreateAsset", mock.Anything, "INVALID", "INVALID", "CURRENCY", "BRL").Return("", errors.New("create err"))
	_ = s.BackfillGap(context.Background(), "INVALID", time.Now())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()
	s.httpClient.Transport = &mockTransport{serverURL: server.URL}
	repo.On("GetAssetByTicker", mock.Anything, "TEST1").Return("t1", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "t1").Return(time.Now(), nil)
	_ = s.BackfillGap(context.Background(), "TEST1", time.Now().AddDate(0, 0, -10))

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"chart": {"error": {"description": "err"}}}`))
	}))
	defer server2.Close()
	s.httpClient.Transport = &mockTransport{serverURL: server2.URL}
	_ = s.BackfillGap(context.Background(), "TEST1", time.Now().AddDate(0, 0, -10))

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"chart": {"result": []}}`))
	}))
	defer server3.Close()
	s.httpClient.Transport = &mockTransport{serverURL: server3.URL}
	_ = s.BackfillGap(context.Background(), "TEST1", time.Now().AddDate(0, 0, -10))

	server4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"chart": {"result": [{"timestamp": []}]}}`))
	}))
	defer server4.Close()
	s.httpClient.Transport = &mockTransport{serverURL: server4.URL}
	_ = s.BackfillGap(context.Background(), "TEST1", time.Now().AddDate(0, 0, -10))

	server5 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"chart": {
				"result": [{
					"timestamp": [1609459200, 1609545600],
					"indicators": {
						"quote": [{
							"close": [150.0]
						}]
					}
				}]
			}
		}`))
	}))
	defer server5.Close()
	s.httpClient.Transport = &mockTransport{serverURL: server5.URL}
	_ = s.BackfillGap(context.Background(), "TEST1", time.Now().AddDate(0, 0, -10))
}

func TestServiceCoverage_GetCurrencyRate(t *testing.T) {
	s, _, ms, _ := setupServiceTest()
	ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(nil, errors.New("err")).Once()
	ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(&market.Quote{Price: 5.5}, nil).Once()

	rate := s.getCurrencyRate(context.Background(), "USD", "BRL")
	assert.Equal(t, 5.5, rate)
}

func TestServiceCoverage_UpdateTransaction_BackfillError(t *testing.T) {
	s, repo, _, _ := setupServiceTest()

	s.httpClient = &http.Client{
		Transport: &MockHTTPTransport{Err: errors.New("http err")},
	}

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("a1", "BRL", nil)
	repo.On("UpdateTransaction", mock.Anything, mock.Anything).Return(nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Now().Add(24*time.Hour), nil)
	repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("a1", nil)
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "a1", mock.Anything).Return(nil)

	// Add BackfillGap failure and exchange rate fallback for coverage
	repo.On("GetExchangeRateByDate", mock.Anything, "BRLUSD=X", mock.Anything).Return(0.0, errors.New("err")).Once()
	repo.On("GetAssetByTicker", mock.Anything, "BRLUSD=X").Return("brlusd-id", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "brlusd-id").Return(time.Time{}, errors.New("err"))
	repo.On("GetDailyPrices", mock.Anything, "brlusd-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "brlusd-id", mock.Anything).Return(nil)
	repo.On("GetExchangeRateByDate", mock.Anything, "BRLUSD=X", mock.Anything).Return(1.5, nil).Once() // Fallback mock

	tx := &Transaction{Ticker: "AAPL", Type: "BUY", ExecutedAt: time.Now(), Currency: "BRL"}
	err := s.UpdateTransaction(context.Background(), "u1", "p1", "t1", tx)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
}

func TestServiceCoverage_UpdateTransaction_DBError(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("a1", "USD", nil)
	repo.On("UpdateTransaction", mock.Anything, mock.Anything).Return(errors.New("db error"))

	tx := &Transaction{Ticker: "AAPL", Type: "BUY", ExecutedAt: time.Now(), Currency: "USD"}
	err := s.UpdateTransaction(context.Background(), "u1", "p1", "t1", tx)
	assert.ErrorContains(t, err, "falha ao atualizar transação")
}

func TestServiceCoverage_GetPortfolioPerformance_NoTxs(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10},
	}, nil)
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)

	// Will filter out AAPL, resulting in 0 txs
	perf, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", []string{"MSFT"})
	assert.NoError(t, err)
	assert.Empty(t, perf)
}

func TestServiceCoverage_GetPortfolioPerformance_FutureTx(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, ExecutedAt: time.Now().AddDate(1, 0, 0), Currency: "USD"},
	}, nil)
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Now().AddDate(1, 0, 0), nil)
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)

	perf, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", []string{})
	assert.NoError(t, err)
	assert.NotEmpty(t, perf)
}

func TestServiceCoverage_AddTransaction_TotalFail(t *testing.T) {
	s, repo, _, mp := setupServiceTest()

	s.httpClient = &http.Client{
		Transport: &MockHTTPTransport{Err: errors.New("http err")},
	}

	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "NEW-USD").Return("", "", pgx.ErrNoRows)
	mp.On("SearchAssets", mock.Anything, "NEW-USD").Return([]market.SearchResult{}, nil)
	mp.On("GetQuote", mock.Anything, "NEW-USD").Return(&market.Quote{Currency: "USD", Name: "New Coin"}, nil)
	repo.On("CreateAsset", mock.Anything, "NEW-USD", "New Coin", "CRYPTO", "USD").Return("new-a", nil)

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("CreateTransaction", mock.Anything, mock.Anything).Return(&Transaction{ID: "tx1"}, nil)
	repo.On("GetOldestPriceDate", mock.Anything, "new-a").Return(time.Time{}, errors.New("no price"))
	repo.On("GetDailyPrices", mock.Anything, "new-a", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "new-a", mock.Anything).Return(errors.New("db err"))

	// Fail both times for Exchange Rate
	repo.On("GetExchangeRateByDate", mock.Anything, mock.Anything, mock.Anything).Return(0.0, errors.New("err"))
	repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usd-brl-id", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "usd-brl-id").Return(time.Time{}, errors.New("err"))
	repo.On("GetDailyPrices", mock.Anything, "usd-brl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "usd-brl-id", mock.Anything).Return(nil)
	repo.On("UpdateTransaction", mock.Anything, mock.Anything).Return(nil) // Goroutine updates tx

	tx := &Transaction{PortfolioID: "p1", Ticker: "NEW-USD", Type: "BUY", Quantity: 10, UnitPrice: 100, ExecutedAt: time.Now()}
	tx, err := s.AddTransaction(context.Background(), "u1", tx)
	time.Sleep(200 * time.Millisecond)
	assert.NoError(t, err)
}

func TestServiceCoverage_BackfillHistoricalPrices_BadURL(t *testing.T) {
	s, _, _, _ := setupServiceTest()
	// nil context makes NewRequestWithContext fail
	err := s.BackfillHistoricalPrices(nil, "a1", "AAPL")
	assert.Error(t, err)
}

func TestServiceCoverage_BackfillGap_BadURL(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetAssetByTicker", mock.Anything, mock.Anything).Return("a1", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Now().AddDate(0, 0, 1), nil)

	err := s.BackfillGap(nil, "AAPL", time.Now())
	assert.Error(t, err)
}

func TestServiceCoverage_AddTransaction_BackfillGap(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("a1", "USD", nil)
	repo.On("CreateTransaction", mock.Anything, mock.Anything).Return(&Transaction{ID: "tx1"}, nil)

	// To enter the if block:
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{{}}, nil)
	oldestDate := time.Now().AddDate(0, -1, 0)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(oldestDate, nil)

	// Fail the BackfillGap
	repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("a1", nil)

	s.httpClient = &http.Client{
		Transport: &MockHTTPTransport{Err: errors.New("http err")},
	}

	tx := &Transaction{PortfolioID: "p1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, Currency: "USD", ExecutedAt: oldestDate.AddDate(0, 0, -1)}
	_, err := s.AddTransaction(context.Background(), "u1", tx)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
}

func TestServiceCoverage_UpdateTransaction_ExchangeFallback(t *testing.T) {
	s, repo, _, _ := setupServiceTest()

	s.httpClient = &http.Client{
		Transport: &MockHTTPTransport{Err: errors.New("http err")},
	}

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("a1", "BRL", nil)
	repo.On("UpdateTransaction", mock.Anything, mock.Anything).Return(nil)

	repo.On("GetExchangeRateByDate", mock.Anything, "BRLUSD=X", mock.Anything).Return(0.0, errors.New("err")).Once()
	repo.On("GetAssetByTicker", mock.Anything, "BRLUSD=X").Return("brlusd-id", nil)
	repo.On("GetOldestPriceDate", mock.Anything, "brlusd-id").Return(time.Time{}, errors.New("err"))
	repo.On("GetDailyPrices", mock.Anything, "brlusd-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("SaveDailyPrices", mock.Anything, "brlusd-id", mock.Anything).Return(nil)
	// Fail the second time too!
	repo.On("GetExchangeRateByDate", mock.Anything, "BRLUSD=X", mock.Anything).Return(0.0, errors.New("err2")).Once()

	// Add mocks for the background backfill check on the asset "a1"
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)
	repo.On("GetOldestPriceDate", mock.Anything, "a1").Return(time.Time{}, errors.New("ignored"))

	tx := &Transaction{Ticker: "AAPL", Type: "BUY", ExecutedAt: time.Now(), Currency: "BRL"}
	err := s.UpdateTransaction(context.Background(), "u1", "p1", "t1", tx)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
}

func TestServiceCoverage_GetPortfolioPerformance_Bonus(t *testing.T) {
	t.Run("Standard BRL Bonus", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		now := time.Now().Truncate(24 * time.Hour)
		startDate := now.AddDate(0, 0, -10)
		bonusDate := now.AddDate(0, 0, -5)

		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usdbrl-id", nil).Maybe()
		repo.On("GetDailyPrices", mock.Anything, "usdbrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil).Maybe()
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
			{AssetID: "a1", Ticker: "ITUB4", Type: "BUY", Quantity: 100, UnitPrice: 20.0, TotalCost: 2000.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: startDate},
			{AssetID: "a1", Ticker: "ITUB4", Type: "BONUS", Quantity: 10, UnitPrice: 15.0, TotalCost: 150.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: bonusDate},
		}, nil)

		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "a1", PriceDate: startDate, ClosePrice: 20.0},
			{AssetID: "a1", PriceDate: bonusDate, ClosePrice: 25.0},
			{AssetID: "a1", PriceDate: now, ClosePrice: 30.0},
		}, nil)

		pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, pts)

		// O último ponto deve refletir 110 ações a R$ 30,00 e TotalInvested = R$ 2150,00
		lastPt := pts[len(pts)-1]
		assert.InDelta(t, 2150.0, lastPt.TotalInvested, 1e-4)
		assert.InDelta(t, 110.0*30.0, lastPt.Value, 1e-4)
	})

	t.Run("Bonus Followed By Sell", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		now := time.Now().Truncate(24 * time.Hour)
		startDate := now.AddDate(0, 0, -10)
		bonusDate := now.AddDate(0, 0, -5)
		sellDate := now.AddDate(0, 0, -2)

		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usdbrl-id", nil).Maybe()
		repo.On("GetDailyPrices", mock.Anything, "usdbrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil).Maybe()
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
			{AssetID: "a1", Ticker: "ITUB4", Type: "BUY", Quantity: 100, UnitPrice: 20.0, TotalCost: 2000.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: startDate},
			{AssetID: "a1", Ticker: "ITUB4", Type: "BONUS", Quantity: 10, UnitPrice: 15.0, TotalCost: 150.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: bonusDate},
			{AssetID: "a1", Ticker: "ITUB4", Type: "SELL", Quantity: 20, UnitPrice: 35.0, TotalCost: 700.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: sellDate},
		}, nil)

		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "a1", PriceDate: startDate, ClosePrice: 20.0},
			{AssetID: "a1", PriceDate: bonusDate, ClosePrice: 25.0},
			{AssetID: "a1", PriceDate: sellDate, ClosePrice: 35.0},
			{AssetID: "a1", PriceDate: now, ClosePrice: 30.0},
		}, nil)

		pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, pts)

		// 100 + 10 - 20 = 90 ações
		// Custo após bonificação: 2150. Custo proporcional após venda de 20/110: 90 * (2150 / 110) = 1759.0909
		lastPt := pts[len(pts)-1]
		assert.InDelta(t, 90.0*(2150.0/110.0), lastPt.TotalInvested, 1e-4)
		assert.InDelta(t, 90.0*30.0, lastPt.Value, 1e-4)
	})

	t.Run("Foreign Currency Bonus With Exchange Rate", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		now := time.Now().Truncate(24 * time.Hour)
		startDate := now.AddDate(0, 0, -10)
		bonusDate := now.AddDate(0, 0, -5)

		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usdbrl-id", nil).Maybe()
		repo.On("GetDailyPrices", mock.Anything, "usdbrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "usdbrl-id", PriceDate: startDate, ClosePrice: 5.0},
			{AssetID: "usdbrl-id", PriceDate: bonusDate, ClosePrice: 5.0},
			{AssetID: "usdbrl-id", PriceDate: now, ClosePrice: 5.0},
		}, nil).Maybe()

		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
			{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 100.0, TotalCost: 1000.0, ExchangeRate: 5.0, Currency: "USD", ExecutedAt: startDate},
			{AssetID: "a1", Ticker: "AAPL", Type: "BONUS", Quantity: 2, UnitPrice: 50.0, TotalCost: 100.0, ExchangeRate: 5.0, Currency: "USD", ExecutedAt: bonusDate},
		}, nil)

		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "a1", PriceDate: startDate, ClosePrice: 100.0},
			{AssetID: "a1", PriceDate: bonusDate, ClosePrice: 120.0},
			{AssetID: "a1", PriceDate: now, ClosePrice: 150.0},
		}, nil)

		pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, pts)

		// Custo: (10 * 100 * 5.0) + (2 * 50 * 5.0) = 5000 + 500 = 5500 BRL
		// Valor: 12 * 150 * 5.0 = 9000 BRL
		lastPt := pts[len(pts)-1]
		assert.InDelta(t, 5500.0, lastPt.TotalInvested, 1e-4)
		assert.InDelta(t, 12.0*150.0*5.0, lastPt.Value, 1e-4)
	})
}

type MockFixedIncomeService struct {
	mock.Mock
	fixedincome.Service
}

func (m *MockFixedIncomeService) GetIndexRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]fixedincome.IndexRate, error) {
	args := m.Called(ctx, indexer, startDate, endDate)
	if args.Get(0) != nil {
		return args.Get(0).([]fixedincome.IndexRate), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestServiceCoverage_GetPortfolioPerformance_Benchmarks_BRL(t *testing.T) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)
	mp := new(MockMarketProvider)
	fiService := new(MockFixedIncomeService)
	uow := &dummyUOW{}
	s := NewService(repo, ms, mp, fiService, uow)

	now := time.Now().Truncate(24 * time.Hour)
	startDate := now.AddDate(0, 0, -10)

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usdbrl-id", nil).Maybe()
	repo.On("GetDailyPrices", mock.Anything, "usdbrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{
		{AssetID: "usdbrl-id", PriceDate: startDate, ClosePrice: 5.0},
		{AssetID: "usdbrl-id", PriceDate: now, ClosePrice: 5.2},
	}, nil).Maybe()

	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "PETR4", Type: "BUY", Quantity: 100, UnitPrice: 30.0, TotalCost: 3000.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: startDate},
	}, nil)

	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
		{AssetID: "a1", PriceDate: startDate, ClosePrice: 30.0},
		{AssetID: "a1", PriceDate: now, ClosePrice: 35.0},
	}, nil)

	// Mock benchmark rates
	fiService.On("GetIndexRates", mock.Anything, "CDI", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{
		{Date: startDate, Rate: 0.05},
		{Date: now, Rate: 0.05},
	}, nil)
	fiService.On("GetIndexRates", mock.Anything, "IPCA", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{
		{Date: startDate, Rate: 0.40},
		{Date: now, Rate: 0.45},
	}, nil)
	fiService.On("GetIndexRates", mock.Anything, "IFIX", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{
		{Date: startDate, Rate: 3000.0},
		{Date: now, Rate: 3050.0},
	}, nil)
	fiService.On("GetIndexRates", mock.Anything, "IBOV", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{
		{Date: startDate, Rate: 120000.0},
		{Date: now, Rate: 125000.0},
	}, nil)
	fiService.On("GetIndexRates", mock.Anything, "SP500", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{
		{Date: startDate, Rate: 5000.0},
		{Date: now, Rate: 5200.0},
	}, nil)

	pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, pts)

	lastPt := pts[len(pts)-1]
	assert.True(t, lastPt.CdiReturnPct > 0)
	assert.True(t, lastPt.IpcaReturnPct > 0)
	assert.True(t, lastPt.IfixReturnPct > 0)
	assert.True(t, lastPt.IbovReturnPct > 0)
	assert.True(t, lastPt.Sp500ReturnPct > 0)
}

func TestServiceCoverage_GetPortfolioPerformance_Benchmarks_USD(t *testing.T) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)
	mp := new(MockMarketProvider)
	fiService := new(MockFixedIncomeService)
	uow := &dummyUOW{}
	s := NewService(repo, ms, mp, fiService, uow)

	now := time.Now().Truncate(24 * time.Hour)
	startDate := now.AddDate(0, 0, -5)

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150.0, TotalCost: 1500.0, ExchangeRate: 1.0, Currency: "USD", ExecutedAt: startDate},
	}, nil)

	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
		{AssetID: "a1", PriceDate: startDate, ClosePrice: 150.0},
		{AssetID: "a1", PriceDate: now, ClosePrice: 160.0},
	}, nil)

	fiService.On("GetIndexRates", mock.Anything, "CDI", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{}, nil)
	fiService.On("GetIndexRates", mock.Anything, "IPCA", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{}, nil)
	fiService.On("GetIndexRates", mock.Anything, "IFIX", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{}, nil)
	fiService.On("GetIndexRates", mock.Anything, "IBOV", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{}, nil)
	fiService.On("GetIndexRates", mock.Anything, "SP500", mock.Anything, mock.Anything).Return([]fixedincome.IndexRate{
		{Date: startDate, Rate: 5000.0},
		{Date: now, Rate: 5100.0},
	}, nil)

	pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, pts)

	lastPt := pts[len(pts)-1]
	assert.InDelta(t, ((5100.0-5000.0)/5000.0)*100.0, lastPt.Sp500ReturnPct, 1e-4)
}

func TestServiceCoverage_GetPortfolioPerformance_Benchmarks_Errors(t *testing.T) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)
	mp := new(MockMarketProvider)
	fiService := new(MockFixedIncomeService)
	uow := &dummyUOW{}
	s := NewService(repo, ms, mp, fiService, uow)

	now := time.Now().Truncate(24 * time.Hour)
	startDate := now.AddDate(0, 0, -5)

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("", errors.New("not found")).Maybe()
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "PETR4", Type: "BUY", Quantity: 100, UnitPrice: 30.0, TotalCost: 3000.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: startDate},
	}, nil)

	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
		{AssetID: "a1", PriceDate: startDate, ClosePrice: 30.0},
		{AssetID: "a1", PriceDate: now, ClosePrice: 35.0},
	}, nil)

	fiService.On("GetIndexRates", mock.Anything, "CDI", mock.Anything, mock.Anything).Return(([]fixedincome.IndexRate)(nil), errors.New("cdi err"))
	fiService.On("GetIndexRates", mock.Anything, "IPCA", mock.Anything, mock.Anything).Return(([]fixedincome.IndexRate)(nil), errors.New("ipca err"))
	fiService.On("GetIndexRates", mock.Anything, "IFIX", mock.Anything, mock.Anything).Return(([]fixedincome.IndexRate)(nil), errors.New("ifix err"))
	fiService.On("GetIndexRates", mock.Anything, "IBOV", mock.Anything, mock.Anything).Return(([]fixedincome.IndexRate)(nil), errors.New("ibov err"))
	fiService.On("GetIndexRates", mock.Anything, "SP500", mock.Anything, mock.Anything).Return(([]fixedincome.IndexRate)(nil), errors.New("sp500 err"))

	pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, pts)
}

func TestServiceCoverage_GetPortfolioPerformance_FutureTxAllPeriod(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	futureDate := time.Now().AddDate(0, 0, 5)

	repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
	repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{
		{AssetID: "a1", Ticker: "PETR4", Type: "BUY", Quantity: 100, UnitPrice: 30.0, TotalCost: 3000.0, ExchangeRate: 1.0, Currency: "BRL", ExecutedAt: futureDate},
	}, nil)
	repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)

	pts, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "ALL", nil)
	assert.NoError(t, err)
	assert.Empty(t, pts)
}

func TestServiceCoverage_ResolveTransactionExchangeRate(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	now := time.Now()

	t.Run("existing rate > 0", func(t *testing.T) {
		rate := s.resolveTransactionExchangeRate(context.Background(), "USD", "BRL", now, 5.45)
		assert.Equal(t, 5.45, rate)
	})

	t.Run("same currency or empty", func(t *testing.T) {
		assert.Equal(t, 1.0, s.resolveTransactionExchangeRate(context.Background(), "BRL", "BRL", now, 0))
		assert.Equal(t, 1.0, s.resolveTransactionExchangeRate(context.Background(), "", "BRL", now, 0))
		assert.Equal(t, 1.0, s.resolveTransactionExchangeRate(context.Background(), "USD", "", now, 0))
	})

	t.Run("first fetch success", func(t *testing.T) {
		repo.On("GetExchangeRateByDate", mock.Anything, "EURBRL=X", now).Return(6.10, nil).Once()

		rate := s.resolveTransactionExchangeRate(context.Background(), "EUR", "BRL", now, 0)
		assert.Equal(t, 6.10, rate)
	})

	t.Run("second fetch success after backfill", func(t *testing.T) {
		repo.On("GetExchangeRateByDate", mock.Anything, "GBPBRL=X", now).Return(0.0, errors.New("not found")).Once()
		repo.On("GetAssetByTicker", mock.Anything, "GBPBRL=X").Return("gbpbrl-id", nil).Once()
		repo.On("GetOldestPriceDate", mock.Anything, "gbpbrl-id").Return(time.Time{}, errors.New("err")).Once()
		repo.On("GetDailyPrices", mock.Anything, "gbpbrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil).Once()
		repo.On("SaveDailyPrices", mock.Anything, "gbpbrl-id", mock.Anything).Return(nil).Once()
		repo.On("GetExchangeRateByDate", mock.Anything, "GBPBRL=X", now).Return(7.25, nil).Once()

		rate := s.resolveTransactionExchangeRate(context.Background(), "GBP", "BRL", now, 0)
		assert.Equal(t, 7.25, rate)
	})

	t.Run("both fetch fail fallback to 1.0", func(t *testing.T) {
		repo.On("GetExchangeRateByDate", mock.Anything, "JPYBRL=X", now).Return(0.0, errors.New("not found")).Once()
		repo.On("GetAssetByTicker", mock.Anything, "JPYBRL=X").Return("jpybrl-id", nil).Once()
		repo.On("GetOldestPriceDate", mock.Anything, "jpybrl-id").Return(time.Time{}, errors.New("err")).Once()
		repo.On("GetDailyPrices", mock.Anything, "jpybrl-id", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil).Once()
		repo.On("SaveDailyPrices", mock.Anything, "jpybrl-id", mock.Anything).Return(nil).Once()
		repo.On("GetExchangeRateByDate", mock.Anything, "JPYBRL=X", now).Return(0.0, errors.New("still not found")).Once()

		rate := s.resolveTransactionExchangeRate(context.Background(), "JPY", "BRL", now, 0)
		assert.Equal(t, 1.0, rate)
	})
}





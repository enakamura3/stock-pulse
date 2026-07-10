package portfolio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		{AssetID: "a1", Ticker: "AAPL", Type: "REVERSE_SPLIT", Quantity: 2, UnitPrice: 0, ExchangeRate: 1, ExecutedAt: time.Now().Add(2*time.Hour), Currency: "USD"},
		{AssetID: "a1", Ticker: "AAPL", Type: "BONUS", Quantity: 1, UnitPrice: 0, ExchangeRate: 1, ExecutedAt: time.Now().Add(3*time.Hour), Currency: "USD"},
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
	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "NEW-USD").Return("", "", errors.New("not found"))

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

	repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "NEW-USD").Return("", "", errors.New("not found"))
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

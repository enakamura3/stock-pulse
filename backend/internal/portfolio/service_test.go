package portfolio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPortfolioRepo struct {
	mock.Mock
}

func (m *MockPortfolioRepo) CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error) {
	args := m.Called(ctx, userID, name, baseCurrency)
	if args.Get(0) != nil {
		return args.Get(0).(*Portfolio), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) GetPortfoliosByUserID(ctx context.Context, userID string) ([]Portfolio, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]Portfolio), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) GetPortfolioByID(ctx context.Context, id, userID string) (*Portfolio, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*Portfolio), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) DeletePortfolio(ctx context.Context, id, userID string) error {
	return m.Called(ctx, id, userID).Error(0)
}

func (m *MockPortfolioRepo) CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	args := m.Called(ctx, tx)
	if args.Get(0) != nil {
		return args.Get(0).(*Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) UpdateTransaction(ctx context.Context, tx Transaction) error {
	return m.Called(ctx, tx).Error(0)
}

func (m *MockPortfolioRepo) GetTransactionsByPortfolioID(ctx context.Context, portfolioID, userID string) ([]Transaction, error) {
	args := m.Called(ctx, portfolioID, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error {
	return m.Called(ctx, txID, portfolioID, userID).Error(0)
}

func (m *MockPortfolioRepo) SaveDailyPrices(ctx context.Context, assetID string, prices []DailyPrice) error {
	return m.Called(ctx, assetID, prices).Error(0)
}

func (m *MockPortfolioRepo) GetDailyPrices(ctx context.Context, assetID string, startDate, endDate time.Time) ([]DailyPrice, error) {
	args := m.Called(ctx, assetID, startDate, endDate)
	if args.Get(0) != nil {
		return args.Get(0).([]DailyPrice), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) GetAssetByTicker(ctx context.Context, ticker string) (string, error) {
	args := m.Called(ctx, ticker)
	return args.String(0), args.Error(1)
}

func (m *MockPortfolioRepo) GetAssetAndCurrencyByTicker(ctx context.Context, ticker string) (string, string, error) {
	args := m.Called(ctx, ticker)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockPortfolioRepo) CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error) {
	args := m.Called(ctx, ticker, name, assetType, currency)
	return args.String(0), args.Error(1)
}

func (m *MockPortfolioRepo) GetAllAssets(ctx context.Context) ([]AssetCompact, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]AssetCompact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepo) UpsertAssetEvent(ctx context.Context, event AssetEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockPortfolioRepo) GetAssetEvents(ctx context.Context, assetID string) ([]AssetEvent, error) {
	args := m.Called(ctx, assetID)
	if args.Get(0) != nil {
		return args.Get(0).([]AssetEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockMarketService struct {
	mock.Mock
}

func (m *MockMarketService) GetQuote(ctx context.Context, ticker string) (*market.Quote, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*market.Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMarketService) GetFundamentals(ctx context.Context, ticker string) (*market.Fundamentals, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*market.Fundamentals), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMarketService) SearchAssets(ctx context.Context, query string) ([]market.SearchResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]market.SearchResult), args.Error(1)
}

func (m *MockMarketService) GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]market.DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockMarketProvider struct {
	mock.Mock
}

func (m *MockMarketProvider) GetQuote(ctx context.Context, ticker string) (*market.Quote, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).(*market.Quote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMarketProvider) SearchAssets(ctx context.Context, query string) ([]market.SearchResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]market.SearchResult), args.Error(1)
}

func (m *MockMarketProvider) GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error) {
	args := m.Called(ctx, ticker)
	if args.Get(0) != nil {
		return args.Get(0).([]market.DividendEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupServiceTest() (*Service, *MockPortfolioRepo, *MockMarketService, *MockMarketProvider) {
	repo := new(MockPortfolioRepo)
	ms := new(MockMarketService)
	mp := new(MockMarketProvider)
	s := NewService(repo, ms, mp)
	return s, repo, ms, mp
}

func TestService_CreatePortfolio(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("CreatePortfolio", mock.Anything, "u1", "My Port", "USD").Return(&Portfolio{ID: "p1"}, nil)

	p, err := s.CreatePortfolio(context.Background(), "u1", "My Port", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "p1", p.ID)

	_, err = s.CreatePortfolio(context.Background(), "u1", "  ", "USD")
	assert.ErrorContains(t, err, "não pode ser vazio")
	
	repo.On("CreatePortfolio", mock.Anything, "u2", "My Port", "BRL").Return(&Portfolio{ID: "p2"}, nil)
	p, err = s.CreatePortfolio(context.Background(), "u2", "My Port", "")
	assert.NoError(t, err)
	assert.Equal(t, "p2", p.ID)
}

func TestService_GetPortfolios(t *testing.T) {
	t.Run("Existing lists", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfoliosByUserID", mock.Anything, "u1").Return([]Portfolio{{ID: "p1"}}, nil)

		lists, err := s.GetPortfolios(context.Background(), "u1")
		assert.NoError(t, err)
		assert.Len(t, lists, 1)
	})

	t.Run("Onboarding create default", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfoliosByUserID", mock.Anything, "u2").Return([]Portfolio{}, nil)
		repo.On("CreatePortfolio", mock.Anything, "u2", "Principal", "BRL").Return(&Portfolio{ID: "p2"}, nil)

		lists, err := s.GetPortfolios(context.Background(), "u2")
		assert.NoError(t, err)
		assert.Len(t, lists, 1)
		assert.Equal(t, "p2", lists[0].ID)
	})

	t.Run("Repo Error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfoliosByUserID", mock.Anything, "u3").Return(nil, errors.New("db error"))

		_, err := s.GetPortfolios(context.Background(), "u3")
		assert.ErrorContains(t, err, "db error")
	})

	t.Run("Onboarding Error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfoliosByUserID", mock.Anything, "u4").Return([]Portfolio{}, nil)
		repo.On("CreatePortfolio", mock.Anything, "u4", "Principal", "BRL").Return(nil, errors.New("err"))

		_, err := s.GetPortfolios(context.Background(), "u4")
		assert.ErrorContains(t, err, "falha ao criar portfólio")
	})
}

func TestService_GetPortfolioDetails(t *testing.T) {
	t.Run("Not found", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(nil, errors.New("err"))

		_, _, err := s.GetPortfolioDetails(context.Background(), "p1", "u1")
		assert.ErrorContains(t, err, "não encontrada ou acesso")
	})

	t.Run("Tx error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(nil, errors.New("err"))

		_, _, err := s.GetPortfolioDetails(context.Background(), "p1", "u1")
		assert.ErrorContains(t, err, "erro ao carregar")
	})

	t.Run("Success with conversion and missing quote", func(t *testing.T) {
		s, repo, ms, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
		
		now := time.Now()
		txs := []Transaction{
			{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, ExchangeRate: 5.0, ExecutedAt: now, CreatedAt: now},
			{AssetID: "a1", Ticker: "AAPL", Type: "SELL", Quantity: 5, UnitPrice: 160, ExchangeRate: 5.0, ExecutedAt: now.Add(time.Hour), CreatedAt: now.Add(time.Hour)},
			{AssetID: "a2", Ticker: "INVALID", Type: "BUY", Quantity: 10, UnitPrice: 10, ExchangeRate: 1.0, ExecutedAt: now},
		}
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(txs, nil)

		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 170.0, Currency: "USD"}, nil)
		ms.On("GetFundamentals", mock.Anything, "AAPL").Return(&market.Fundamentals{}, nil)
		ms.On("GetQuote", mock.Anything, "INVALID").Return(nil, errors.New("err"))
		ms.On("GetFundamentals", mock.Anything, "INVALID").Return(nil, errors.New("err"))
		ms.On("GetQuote", mock.Anything, "BRL=X").Return(nil, errors.New("err"))
		ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(&market.Quote{Price: 5.2}, nil)

		_, pos, err := s.GetPortfolioDetails(context.Background(), "p1", "u1")
		assert.NoError(t, err)
		assert.Len(t, pos, 2)
	})
	
	t.Run("Sell more than holding", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
		
		now := time.Now()
		txs := []Transaction{
			{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 5, UnitPrice: 150, ExchangeRate: 5.0, ExecutedAt: now, CreatedAt: now},
			{AssetID: "a1", Ticker: "AAPL", Type: "SELL", Quantity: 10, UnitPrice: 160, ExchangeRate: 5.0, ExecutedAt: now.Add(time.Hour), CreatedAt: now.Add(time.Hour)},
		}
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(txs, nil)

		_, pos, err := s.GetPortfolioDetails(context.Background(), "p1", "u1")
		assert.NoError(t, err)
		assert.Len(t, pos, 0) // because qty becomes 0
	})
}

func TestService_GetCurrencyRate(t *testing.T) {
	s, _, ms, _ := setupServiceTest()
	assert.Equal(t, 1.0, s.getCurrencyRate(context.Background(), "USD", "USD"))
	
	ms.On("GetQuote", mock.Anything, "USDEUR=X").Return(&market.Quote{Price: 0.85}, nil)
	assert.Equal(t, 0.85, s.getCurrencyRate(context.Background(), "USD", "EUR"))
	
	ms.On("GetQuote", mock.Anything, "CADBRL=X").Return(nil, errors.New("err"))
	assert.Equal(t, 1.0, s.getCurrencyRate(context.Background(), "CAD", "BRL"))
	
	ms.On("GetQuote", mock.Anything, "USDBRL=X").Return(nil, errors.New("err"))
	assert.Equal(t, 5.20, s.getCurrencyRate(context.Background(), "USD", "BRL")) // fallback
}

func TestService_DeleteMethods(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("DeletePortfolio", mock.Anything, "p1", "u1").Return(nil)

		err := s.DeletePortfolio(context.Background(), "p1", "u1")
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	s, repo, _, _ := setupServiceTest()
	repo.On("DeleteTransaction", mock.Anything, "tx1", "p1", "u1").Return(nil)
	assert.NoError(t, s.DeleteTransaction(context.Background(), "tx1", "p1", "u1"))
}

func TestService_GetPortfolioTransactions(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		txs := []Transaction{{ID: "tx1"}}
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(txs, nil)

		res, err := s.GetPortfolioTransactions(context.Background(), "p1", "u1")
		assert.NoError(t, err)
		assert.Equal(t, txs, res)
		repo.AssertExpectations(t)
	})
}

func TestService_AddTransaction(t *testing.T) {
	t.Run("Portfolio Not Found", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(nil, errors.New("err"))
		_, err := s.AddTransaction(context.Background(), "u1", &Transaction{PortfolioID: "p1"})
		assert.ErrorContains(t, err, "carteira não encontrada")
	})

	t.Run("Invalid Ticker", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		_, err := s.AddTransaction(context.Background(), "u1", &Transaction{PortfolioID: "p1", Ticker: "  "})
		assert.ErrorContains(t, err, "ticker do ativo inválido")
	})

	t.Run("Existing Asset", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
		repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("a1", "USD", nil)
		
		tx := &Transaction{PortfolioID: "p1", Ticker: "AAPL", Quantity: 10, UnitPrice: 150}
		repo.On("CreateTransaction", mock.Anything, tx).Return(&Transaction{ID: "tx1"}, nil)

		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{{}}, nil) // backfill skip

		res, err := s.AddTransaction(context.Background(), "u1", tx)
		assert.NoError(t, err)
		assert.Equal(t, "tx1", res.ID)
	})


	t.Run("New Asset - Provider Error", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("", "", errors.New("err"))
		mp.On("GetQuote", mock.Anything, "AAPL").Return(nil, errors.New("err"))

		_, err := s.AddTransaction(context.Background(), "u1", &Transaction{PortfolioID: "p1", Ticker: "AAPL"})
		assert.ErrorContains(t, err, "não encontrado no mercado")
	})

	t.Run("New Crypto Asset - Repo Error", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "BTC-USD").Return("", "", errors.New("err"))
		mp.On("GetQuote", mock.Anything, "BTC-USD").Return(&market.Quote{Currency: "USD", Name: "Bitcoin"}, nil)
		repo.On("CreateAsset", mock.Anything, "BTC-USD", "Bitcoin", "CRYPTO", "USD").Return("", errors.New("err"))

		_, err := s.AddTransaction(context.Background(), "u1", &Transaction{PortfolioID: "p1", Ticker: "BTC-USD"})
		assert.ErrorContains(t, err, "erro ao registrar ativo")
	})
	
	t.Run("New US Equity Asset - Repo Error", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("", "", errors.New("err"))
		mp.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Currency: "USD", Name: "Apple"}, nil)
		repo.On("CreateAsset", mock.Anything, "AAPL", "Apple", "STOCK_US", "USD").Return("", errors.New("err"))

		_, err := s.AddTransaction(context.Background(), "u1", &Transaction{PortfolioID: "p1", Ticker: "AAPL"})
		assert.ErrorContains(t, err, "erro ao registrar ativo")
	})

	t.Run("Create Tx Error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
		repo.On("GetAssetAndCurrencyByTicker", mock.Anything, "AAPL").Return("a1", "USD", nil)
		
		tx := &Transaction{PortfolioID: "p1", Ticker: "AAPL", Quantity: 10, UnitPrice: 150}
		repo.On("CreateTransaction", mock.Anything, tx).Return((*Transaction)(nil), errors.New("db error"))

		_, err := s.AddTransaction(context.Background(), "u1", tx)
		assert.ErrorContains(t, err, "db error")
	})
}

func TestService_GetPortfolioPerformance(t *testing.T) {
	t.Run("Not Found", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(nil, errors.New("err"))

		_, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", nil)
		assert.ErrorContains(t, err, "carteira não encontrada")
	})

	t.Run("Tx Error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(nil, errors.New("err"))

		_, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", nil)
		assert.ErrorContains(t, err, "erro ao carregar transações")
	})

	t.Run("Empty Tx", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{}, nil)
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return([]Transaction{}, nil)

		res, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", nil)
		assert.NoError(t, err)
		assert.Len(t, res, 0)
	})

	t.Run("Success 1M", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
		now := time.Now()
		start := now.AddDate(0, -1, 0)

		txs := []Transaction{
			{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, ExchangeRate: 1.0, Currency: "USD", ExecutedAt: start.Add(time.Hour)},
			{AssetID: "a1", Ticker: "AAPL", Type: "SELL", Quantity: 5, UnitPrice: 160, ExchangeRate: 1.0, Currency: "USD", ExecutedAt: start.Add(24 * time.Hour)},
		}
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(txs, nil)
		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "a1", PriceDate: start, ClosePrice: 150},
			{AssetID: "a1", PriceDate: start.Add(24 * time.Hour), ClosePrice: 160},
		}, nil)

		res, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", nil)
		assert.NoError(t, err)
		assert.True(t, len(res) > 0)
	})
	
	t.Run("Periods", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "USD"}, nil)
		now := time.Now()
		start := now.AddDate(-2, 0, 0)

		txs := []Transaction{
			{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, ExchangeRate: 1.0, Currency: "USD", ExecutedAt: start},
		}
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(txs, nil)
		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{}, nil)

		// Test various periods
		for _, period := range []string{"1M", "3M", "6M", "1Y", "ALL"} {
			res, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", period, nil)
			assert.NoError(t, err)
			assert.True(t, len(res) >= 0)
		}
	})
	
	t.Run("USD to BRL Conversion", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetPortfolioByID", mock.Anything, "p1", "u1").Return(&Portfolio{BaseCurrency: "BRL"}, nil)
		now := time.Now()
		start := now.AddDate(0, -1, 0)

		txs := []Transaction{
			{AssetID: "a1", Ticker: "AAPL", Type: "BUY", Quantity: 10, UnitPrice: 150, ExchangeRate: 5.0, Currency: "USD", ExecutedAt: start.Add(time.Hour)},
		}
		repo.On("GetTransactionsByPortfolioID", mock.Anything, "p1", "u1").Return(txs, nil)
		repo.On("GetAssetByTicker", mock.Anything, "USDBRL=X").Return("usd1", nil)
		repo.On("GetDailyPrices", mock.Anything, "usd1", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "usd1", PriceDate: start, ClosePrice: 5.2},
		}, nil)
		repo.On("GetDailyPrices", mock.Anything, "a1", mock.Anything, mock.Anything).Return([]DailyPrice{
			{AssetID: "a1", PriceDate: start, ClosePrice: 150},
		}, nil)

		res, err := s.GetPortfolioPerformance(context.Background(), "p1", "u1", "1M", nil)
		assert.NoError(t, err)
		assert.True(t, len(res) > 0)
	})
}

// Backfill Test - We need httptest
func TestService_BackfillHistoricalPrices(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"chart": {
					"result": [{
						"timestamp": [1609459200],
						"indicators": {
							"quote": [{
								"close": [150.0]
							}]
						}
					}]
				}
			}`))
		}))
		defer server.Close()

		s, repo, _, _ := setupServiceTest()
		
		// To mock the URL call, we intercept transport
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}
		
		repo.On("SaveDailyPrices", mock.Anything, "a1", mock.Anything).Return(nil)

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.NoError(t, err)
	})
	
	t.Run("HTTP Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		s, _, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.ErrorContains(t, err, "status 500")
	})
	
	t.Run("JSON Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid`))
		}))
		defer server.Close()

		s, _, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.Error(t, err)
	})
	
	t.Run("Provider Error Message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"error": "not found"}}`))
		}))
		defer server.Close()

		s, _, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("Empty Result", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": []}}`))
		}))
		defer server.Close()

		s, _, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.ErrorContains(t, err, "vazio")
	})
	
	t.Run("Missing Indicators", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"timestamp": []}]}}`))
		}))
		defer server.Close()

		s, _, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.ErrorContains(t, err, "sem timestamps")
	})
	
	t.Run("Inconsistent Lengths", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"timestamp": [1, 2], "indicators": {"quote": [{"close": [1]}]}}]}}`))
		}))
		defer server.Close()

		s, _, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.ErrorContains(t, err, "inconsistência")
	})
	
	t.Run("Save Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"chart": {"result": [{"timestamp": [1], "indicators": {"quote": [{"close": [1]}]}}]}}`))
		}))
		defer server.Close()

		s, repo, _, _ := setupServiceTest()
		s.httpClient.Transport = &mockTransport{serverURL: server.URL}
		
		repo.On("SaveDailyPrices", mock.Anything, "a1", mock.Anything).Return(errors.New("db error"))

		err := s.BackfillHistoricalPrices(context.Background(), "a1", "AAPL")
		assert.ErrorContains(t, err, "falha ao gravar")
	})
}

// Mock transport to reroute requests
type mockTransport struct {
	serverURL string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(m.serverURL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func (m *MockMarketService) GetHistoricalExchangeRate(ctx context.Context, date time.Time) (float64, error) {
	args := m.Called(ctx, date)
	return args.Get(0).(float64), args.Error(1)
}


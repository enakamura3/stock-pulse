package watchlist

import (
	"context"
	"errors"
	"testing"

	"github.com/onigiri/stockpulse/backend/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWatchlistRepo struct {
	mock.Mock
}

func (m *MockWatchlistRepo) CreateWatchlist(ctx context.Context, userID, name string) (*Watchlist, error) {
	args := m.Called(ctx, userID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*Watchlist), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWatchlistRepo) GetWatchlistsByUserID(ctx context.Context, userID string) ([]Watchlist, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]Watchlist), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWatchlistRepo) GetWatchlistByID(ctx context.Context, id, userID string) (*Watchlist, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*Watchlist), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWatchlistRepo) DeleteWatchlist(ctx context.Context, id, userID string) error {
	return m.Called(ctx, id, userID).Error(0)
}
func (m *MockWatchlistRepo) GetAssetByTicker(ctx context.Context, ticker string) (string, error) {
	args := m.Called(ctx, ticker)
	return args.String(0), args.Error(1)
}
func (m *MockWatchlistRepo) CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error) {
	args := m.Called(ctx, ticker, name, assetType, currency)
	return args.String(0), args.Error(1)
}
func (m *MockWatchlistRepo) AddWatchlistItem(ctx context.Context, watchlistID, assetID string) (*Item, error) {
	args := m.Called(ctx, watchlistID, assetID)
	if args.Get(0) != nil {
		return args.Get(0).(*Item), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockWatchlistRepo) RemoveWatchlistItem(ctx context.Context, watchlistID, ticker string) error {
	return m.Called(ctx, watchlistID, ticker).Error(0)
}
func (m *MockWatchlistRepo) GetWatchlistItems(ctx context.Context, watchlistID string) ([]Item, error) {
	args := m.Called(ctx, watchlistID)
	if args.Get(0) != nil {
		return args.Get(0).([]Item), args.Error(1)
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
	if args.Get(0) != nil {
		return args.Get(0).([]market.SearchResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func setupServiceTest() (*Service, *MockWatchlistRepo, *MockMarketService, *MockMarketProvider) {
	repo := new(MockWatchlistRepo)
	ms := new(MockMarketService)
	mp := new(MockMarketProvider)
	s := NewService(repo, ms, mp)
	return s, repo, ms, mp
}

func TestService_CreateWatchlist(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("CreateWatchlist", mock.Anything, "u1", "My List").Return(&Watchlist{ID: "w1"}, nil)

	w, err := s.CreateWatchlist(context.Background(), "u1", "My List")
	assert.NoError(t, err)
	assert.Equal(t, "w1", w.ID)

	_, err = s.CreateWatchlist(context.Background(), "u1", "   ")
	assert.ErrorContains(t, err, "não pode ser vazio")
}

func TestService_GetWatchlists(t *testing.T) {
	t.Run("Existing lists", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistsByUserID", mock.Anything, "u1").Return([]Watchlist{{ID: "w1"}}, nil)

		lists, err := s.GetWatchlists(context.Background(), "u1")
		assert.NoError(t, err)
		assert.Len(t, lists, 1)
	})

	t.Run("Onboarding create default", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistsByUserID", mock.Anything, "u2").Return([]Watchlist{}, nil)
		repo.On("CreateWatchlist", mock.Anything, "u2", "Favoritos").Return(&Watchlist{ID: "w2"}, nil)

		lists, err := s.GetWatchlists(context.Background(), "u2")
		assert.NoError(t, err)
		assert.Len(t, lists, 1)
		assert.Equal(t, "w2", lists[0].ID)
	})
	
	t.Run("Repo Error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistsByUserID", mock.Anything, "u3").Return(nil, errors.New("db error"))

		_, err := s.GetWatchlists(context.Background(), "u3")
		assert.ErrorContains(t, err, "db error")
	})

	t.Run("Onboarding error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistsByUserID", mock.Anything, "u4").Return([]Watchlist{}, nil)
		repo.On("CreateWatchlist", mock.Anything, "u4", "Favoritos").Return(nil, errors.New("create err"))

		_, err := s.GetWatchlists(context.Background(), "u4")
		assert.ErrorContains(t, err, "falha ao criar watchlist de onboarding: create err")
	})
}

func TestService_GetWatchlist(t *testing.T) {
	t.Run("Not found", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(nil, errors.New("not found"))

		_, err := s.GetWatchlist(context.Background(), "w1", "u1")
		assert.ErrorContains(t, err, "não encontrada")
	})

	t.Run("Items error", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{ID: "w1"}, nil)
		repo.On("GetWatchlistItems", mock.Anything, "w1").Return(nil, errors.New("db error"))

		_, err := s.GetWatchlist(context.Background(), "w1", "u1")
		assert.ErrorContains(t, err, "erro ao carregar itens")
	})

	t.Run("Success with Quotes", func(t *testing.T) {
		s, repo, ms, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{ID: "w1"}, nil)
		
		items := []Item{{Ticker: "AAPL"}, {Ticker: "INVALID"}}
		repo.On("GetWatchlistItems", mock.Anything, "w1").Return(items, nil)

		ms.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Price: 150.0}, nil)
		ms.On("GetFundamentals", mock.Anything, "AAPL").Return(&market.Fundamentals{}, nil)
		ms.On("GetQuote", mock.Anything, "INVALID").Return(nil, errors.New("not found"))

		w, err := s.GetWatchlist(context.Background(), "w1", "u1")
		assert.NoError(t, err)
		assert.Equal(t, 150.0, w.Items[0].Price)
		assert.Equal(t, 0.0, w.Items[1].Price) // Silently continues
	})
}

func TestService_DeleteWatchlist(t *testing.T) {
	s, repo, _, _ := setupServiceTest()
	repo.On("DeleteWatchlist", mock.Anything, "w1", "u1").Return(nil)

	err := s.DeleteWatchlist(context.Background(), "w1", "u1")
	assert.NoError(t, err)
}

func TestService_AddAssetToWatchlist(t *testing.T) {
	t.Run("Invalid ticker", func(t *testing.T) {
		s, _, _, _ := setupServiceTest()
		_, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "  ")
		assert.ErrorContains(t, err, "ticker inválido")
	})

	t.Run("Watchlist Not found", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(nil, errors.New("not found"))
		_, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "AAPL")
		assert.ErrorContains(t, err, "não encontrada")
	})

	t.Run("Existing Asset", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("a1", nil)
		repo.On("AddWatchlistItem", mock.Anything, "w1", "a1").Return(&Item{ID: "i1"}, nil)

		item, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "aapl")
		assert.NoError(t, err)
		assert.Equal(t, "i1", item.ID)
	})

	t.Run("New Asset - Provider Error", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "INVALID").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "INVALID").Return(nil, errors.New("not found"))

		_, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "INVALID")
		assert.ErrorContains(t, err, "não é suportado")
	})

	t.Run("New Asset - Creation Error", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "AAPL").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "AAPL").Return(&market.Quote{Name: "Apple", Currency: "USD"}, nil)
		repo.On("CreateAsset", mock.Anything, "AAPL", "Apple", "EQUITY_US", "USD").Return("", errors.New("db error"))

		_, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "AAPL")
		assert.ErrorContains(t, err, "erro ao registrar novo")
	})

	t.Run("New Crypto Asset - Success", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "BTC-USD").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "BTC-USD").Return(&market.Quote{Name: "Bitcoin", Currency: "USD"}, nil)
		repo.On("CreateAsset", mock.Anything, "BTC-USD", "Bitcoin", "EQUITY_US", "USD").Return("a2", nil)
		repo.On("AddWatchlistItem", mock.Anything, "w1", "a2").Return(&Item{ID: "i2"}, nil)

		item, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "BTC-USD")
		assert.NoError(t, err)
		assert.Equal(t, "i2", item.ID)
	})
	
	t.Run("New Default Asset - Success", func(t *testing.T) {
		s, repo, _, mp := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{}, nil)
		repo.On("GetAssetByTicker", mock.Anything, "PETR4.SA").Return("", errors.New("not found"))
		mp.On("GetQuote", mock.Anything, "PETR4.SA").Return(&market.Quote{Name: "Petrobras", Currency: "BRL"}, nil)
		repo.On("CreateAsset", mock.Anything, "PETR4.SA", "Petrobras", "EQUITY", "BRL").Return("a3", nil)
		repo.On("AddWatchlistItem", mock.Anything, "w1", "a3").Return(&Item{ID: "i3"}, nil)

		item, err := s.AddAssetToWatchlist(context.Background(), "w1", "u1", "PETR4.SA")
		assert.NoError(t, err)
		assert.Equal(t, "i3", item.ID)
	})
}

func TestService_RemoveAssetFromWatchlist(t *testing.T) {
	t.Run("Not found", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(nil, errors.New("not found"))
		err := s.RemoveAssetFromWatchlist(context.Background(), "w1", "u1", "AAPL")
		assert.ErrorContains(t, err, "não encontrada")
	})

	t.Run("Success", func(t *testing.T) {
		s, repo, _, _ := setupServiceTest()
		repo.On("GetWatchlistByID", mock.Anything, "w1", "u1").Return(&Watchlist{}, nil)
		repo.On("RemoveWatchlistItem", mock.Anything, "w1", "AAPL").Return(nil)
		err := s.RemoveAssetFromWatchlist(context.Background(), "w1", "u1", "AAPL")
		assert.NoError(t, err)
	})
}

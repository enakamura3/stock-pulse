package fixedincome

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasuryService_EditDelete(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL is empty, skipping service integration tests")
	}

	pool := getTestDB(t)
	ctx := context.Background()

	err := cleanupDB(ctx, pool)
	require.NoError(t, err)

	// Initialize repository and service
	repo := NewRepository(pool)
	bcbClient := &mockBCBClient{}
	s := NewService(repo, bcbClient)

	// Create test user and portfolio
	var userID string
	err = pool.QueryRow(ctx, "INSERT INTO \"user\" (email, password_hash, name) VALUES ('service_test@test.com', 'hash', 'Service User') RETURNING id").Scan(&userID)
	require.NoError(t, err)

	var portfolioID string
	err = pool.QueryRow(ctx, "INSERT INTO portfolio (user_id, name) VALUES ($1, 'Service Portfolio') RETURNING id", userID).Scan(&portfolioID)
	require.NoError(t, err)

	// Seed holidays so calculations don't fail or return default
	_, err = pool.Exec(ctx, "INSERT INTO anbima_holidays (holiday_date, description) VALUES ('2026-12-25', 'Natal') ON CONFLICT DO NOTHING")
	require.NoError(t, err)

	var subTxID string

	t.Run("Create subscription via service", func(t *testing.T) {
		req := &TreasuryTxRequest{
			Ticker:          "TESOURO PREFIXADO 2031",
			TreasuryType:    "PREFIXADO",
			MaturityDate:    "2031-01-01",
			HasCoupons:      false,
			Type:            "SUBSCRIPTION",
			Quantity:        2.0,
			UnitPrice:       1000.0,
			ContractedRate:  10.0,
			TransactionDate: "2026-06-01",
		}

		res, err := s.CreateTreasuryTransaction(ctx, portfolioID, req)
		require.NoError(t, err)
		
		resMap, ok := res.(map[string]string)
		require.True(t, ok)
		subTxID = resMap["id"]
		assert.NotEmpty(t, subTxID)
	})

	t.Run("Create redemption via service (triggers FIFO)", func(t *testing.T) {
		req := &TreasuryTxRequest{
			Ticker:          "TESOURO PREFIXADO 2031",
			TreasuryType:    "PREFIXADO",
			MaturityDate:    "2031-01-01",
			HasCoupons:      false,
			Type:            "REDEMPTION",
			Quantity:        1.0,
			UnitPrice:       1050.0,
			ContractedRate:  10.0,
			TransactionDate: "2026-07-01",
		}

		res, err := s.CreateTreasuryTransaction(ctx, portfolioID, req)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Edit subscription via service (triggers FIFO rebuild)", func(t *testing.T) {
		// Change the unit price of the subscription from 1000 to 950
		req := &TreasuryTxRequest{
			Ticker:          "TESOURO PREFIXADO 2031",
			TreasuryType:    "PREFIXADO",
			MaturityDate:    "2031-01-01",
			HasCoupons:      false,
			Type:            "SUBSCRIPTION",
			Quantity:        2.0,
			UnitPrice:       950.0,
			ContractedRate:  10.0,
			TransactionDate: "2026-06-01",
		}

		err := s.UpdateTreasuryTransaction(ctx, portfolioID, subTxID, req)
		require.NoError(t, err)

		// Verify that subscription unit price is updated
		positions, err := s.GetTreasuryPositions(ctx, portfolioID)
		require.NoError(t, err)
		require.Len(t, positions, 1)
		assert.Equal(t, 950.0, positions[0].UnitPrice)
	})

	t.Run("Delete subscription via service", func(t *testing.T) {
		err := s.DeleteTreasuryTransaction(ctx, portfolioID, subTxID)
		require.NoError(t, err)

		// Verify position is deleted
		positions, err := s.GetTreasuryPositions(ctx, portfolioID)
		require.NoError(t, err)
		assert.Empty(t, positions)
	})
}

// Mock BCB Client for service setup
type mockBCBClient struct{}

func (m *mockBCBClient) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	return nil, nil
}

func TestTreasuryService_MonthlyYields(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL is empty, skipping service integration tests")
	}

	pool := getTestDB(t)
	ctx := context.Background()

	err := cleanupDB(ctx, pool)
	require.NoError(t, err)

	repo := NewRepository(pool)
	bcbClient := &mockBCBClient{}
	s := NewService(repo, bcbClient)

	// Create test user and portfolio
	var userID string
	err = pool.QueryRow(ctx, "INSERT INTO \"user\" (email, password_hash, name) VALUES ('yield_test@test.com', 'hash', 'Yield User') RETURNING id").Scan(&userID)
	require.NoError(t, err)

	var portfolioID string
	err = pool.QueryRow(ctx, "INSERT INTO portfolio (user_id, name) VALUES ($1, 'Yield Portfolio') RETURNING id", userID).Scan(&portfolioID)
	require.NoError(t, err)

	// Seed holidays so calculations don't fail
	_, err = pool.Exec(ctx, "INSERT INTO anbima_holidays (holiday_date, description) VALUES ('2026-12-25', 'Natal') ON CONFLICT DO NOTHING")
	require.NoError(t, err)

	// Create subscription in the past
	req := &TreasuryTxRequest{
		Ticker:          "TESOURO PREFIXADO 2031",
		TreasuryType:    "PREFIXADO",
		MaturityDate:    "2031-01-01",
		HasCoupons:      false,
		Type:            "SUBSCRIPTION",
		Quantity:        2.0,
		UnitPrice:       1000.0,
		ContractedRate:  10.0,
		TransactionDate: "2026-03-01", // Past date to have monthly yields
	}

	_, err = s.CreateTreasuryTransaction(ctx, portfolioID, req)
	require.NoError(t, err)

	yields, err := s.GetTreasuryMonthlyYields(ctx, portfolioID)
	require.NoError(t, err)

	// We expect at least some monthly yields since March 2026
	assert.NotEmpty(t, yields)
	for _, y := range yields {
		assert.Equal(t, "TESOURO PREFIXADO 2031", y.AssetName)
		assert.Equal(t, "TESOURO", y.AssetType)
		assert.True(t, y.GrossAmount > 0)
		assert.True(t, y.NetAmount > 0)
		assert.NotEmpty(t, y.Month)
	}
}


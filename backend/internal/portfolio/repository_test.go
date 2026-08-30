package portfolio

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func setupRepoTest(t *testing.T) (pgxmock.PgxPoolIface, *Repository) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("erro ao criar mock: %v", err)
	}
	return mock, NewRepository(mock)
}

func TestRepository_CreatePortfolio(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()

	mock.ExpectBegin()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM portfolio`).WithArgs("u1").WillReturnRows(countRows)

	rows := pgxmock.NewRows([]string{"id", "user_id", "name", "base_currency", "is_default", "created_at"}).
		AddRow("p1", "u1", "Main", "USD", true, now)

	mock.ExpectQuery(`INSERT INTO portfolio`).
		WithArgs("u1", "Main", "USD", true).
		WillReturnRows(rows)

	mock.ExpectCommit()

	p, err := repo.CreatePortfolio(context.Background(), "u1", "Main", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "p1", p.ID)
	assert.True(t, p.IsDefault)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreatePortfolio_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectBegin()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM portfolio`).WithArgs("u1").WillReturnRows(countRows)

	mock.ExpectQuery(`INSERT INTO portfolio`).
		WithArgs("u1", "Main", "USD", true).
		WillReturnError(errors.New("db error"))

	mock.ExpectRollback()

	_, err := repo.CreatePortfolio(context.Background(), "u1", "Main", "USD")
	assert.ErrorContains(t, err, "erro ao criar portfolio")
}

func TestRepository_GetPortfoliosByUserID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "user_id", "name", "base_currency", "is_default", "created_at"}).
		AddRow("p1", "u1", "Main", "USD", true, now)

	mock.ExpectQuery(`SELECT id, user_id, name, base_currency, is_default, created_at`).
		WithArgs("u1").
		WillReturnRows(rows)

	list, err := repo.GetPortfoliosByUserID(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.True(t, list[0].IsDefault)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetPortfoliosByUserID_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, user_id, name, base_currency, is_default, created_at`).
		WithArgs("u1").
		WillReturnError(errors.New("db error"))

	_, err := repo.GetPortfoliosByUserID(context.Background(), "u1")
	assert.ErrorContains(t, err, "db error")
}

func TestRepository_GetPortfolioByID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "user_id", "name", "base_currency", "is_default", "created_at"}).
		AddRow("p1", "u1", "Main", "USD", true, now)

	mock.ExpectQuery(`SELECT id, user_id, name, base_currency, is_default, created_at`).
		WithArgs("p1", "u1").
		WillReturnRows(rows)

	p, err := repo.GetPortfolioByID(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.Equal(t, "p1", p.ID)
	assert.True(t, p.IsDefault)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_SetDefaultPortfolio(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()


	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM portfolio WHERE id = \$1 AND user_id = \$2\)`).
		WithArgs("p1", "u1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`UPDATE portfolio SET is_default = false WHERE user_id = \$1`).
		WithArgs("u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE portfolio SET is_default = true WHERE id = \$1 AND user_id = \$2`).
		WithArgs("p1", "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))


	err := repo.SetDefaultPortfolio(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeletePortfolio(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()



	mock.ExpectQuery(`SELECT is_default FROM portfolio WHERE id = \$1 AND user_id = \$2`).
		WithArgs("p1", "u1").
		WillReturnRows(pgxmock.NewRows([]string{"is_default"}).AddRow(false))

	mock.ExpectExec(`DELETE FROM portfolio`).
		WithArgs("p1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))



	err := repo.DeletePortfolio(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeletePortfolio_Error(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()



	mock.ExpectQuery(`SELECT is_default FROM portfolio WHERE id = \$1 AND user_id = \$2`).
		WithArgs("p1", "u1").
		WillReturnRows(pgxmock.NewRows([]string{"is_default"}).AddRow(false))

	mock.ExpectExec(`DELETE FROM portfolio`).
		WithArgs("p1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))



	err := repo.DeletePortfolio(context.Background(), "p1", "u1")
	assert.ErrorContains(t, err, "não encontrado ou permissão")
}

func TestRepository_CreateTransaction(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	tx := &Transaction{
		PortfolioID:  "p1",
		AssetID:      "a1",
		Type:         "BUY",
		Quantity:     10,
		UnitPrice:    150,
		TotalCost:    1500,
		ExchangeRate: 1,
		ExecutedAt:   now,
	}

	rows := pgxmock.NewRows([]string{"id", "created_at"}).
		AddRow("tx1", now)

	mock.ExpectQuery(`INSERT INTO transaction`).
		WithArgs(tx.PortfolioID, tx.AssetID, tx.Type, tx.Quantity, tx.UnitPrice, tx.TotalCost, tx.Fee, tx.ExchangeRate, tx.ExecutedAt).
		WillReturnRows(rows)

	res, err := repo.CreateTransaction(context.Background(), tx)
	assert.NoError(t, err)
	assert.Equal(t, "tx1", res.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetTransactionsByPortfolioID(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "portfolio_id", "asset_id", "type", "quantity", "unit_price", "total_cost", "fee", "exchange_rate", "executed_at", "created_at",
		"ticker", "name", "asset_type", "currency",
	}).
		AddRow("tx1", "p1", "a1", "BUY", 10.0, 150.0, 1500.0, 0.0, 1.0, now, now, "AAPL", "Apple", "EQUITY_US", "USD")

	mock.ExpectQuery(`SELECT t.id`).
		WithArgs("p1", "u1").
		WillReturnRows(rows)

	list, err := repo.GetTransactionsByPortfolioID(context.Background(), "p1", "u1")
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "AAPL", list[0].Ticker)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteTransaction(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM transaction t`).
		WithArgs("tx1", "p1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.DeleteTransaction(context.Background(), "tx1", "p1", "u1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteTransaction_NotFound(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM transaction t`).
		WithArgs("tx1", "p1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.DeleteTransaction(context.Background(), "tx1", "p1", "u1")
	assert.ErrorContains(t, err, "não encontrada")
}

func TestRepository_GetAssetByTicker(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow("a1")
	mock.ExpectQuery(`SELECT id FROM asset`).
		WithArgs("AAPL").
		WillReturnRows(rows)

	id, err := repo.GetAssetByTicker(context.Background(), "AAPL")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)
}

func TestRepository_CreateAsset(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id"}).AddRow("a1")
	mock.ExpectQuery(`INSERT INTO asset`).
		WithArgs("AAPL", "Apple", "EQUITY_US", "USD").
		WillReturnRows(rows)

	id, err := repo.CreateAsset(context.Background(), "AAPL", "Apple", "EQUITY_US", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)
}

func TestRepository_SaveDailyPrices(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		_, repo := setupRepoTest(t)
		err := repo.SaveDailyPrices(context.Background(), "a1", []DailyPrice{})
		assert.NoError(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		now := time.Now()
		prices := []DailyPrice{{AssetID: "a1", PriceDate: now, ClosePrice: 150.0}}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO asset_daily_price`).
			WithArgs("a1", now, 150.0).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		err := repo.SaveDailyPrices(context.Background(), "a1", prices)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Exec Error", func(t *testing.T) {
		mock, repo := setupRepoTest(t)
		defer mock.Close()

		now := time.Now()
		prices := []DailyPrice{{AssetID: "a1", PriceDate: now, ClosePrice: 150.0}}

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO asset_daily_price`).
			WithArgs("a1", now, 150.0).
			WillReturnError(errors.New("db error"))
		mock.ExpectRollback()

		err := repo.SaveDailyPrices(context.Background(), "a1", prices)
		assert.ErrorContains(t, err, "erro ao inserir preço")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetDailyPrices(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"asset_id", "price_date", "close_price"}).
		AddRow("a1", now, 150.0)

	start := now.Add(-24 * time.Hour)
	mock.ExpectQuery(`SELECT asset_id, price_date, close_price`).
		WithArgs("a1", start, now).
		WillReturnRows(rows)

	list, err := repo.GetDailyPrices(context.Background(), "a1", start, now)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllAssets(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "ticker", "currency", "asset_type"}).
		AddRow("a1", "AAPL", "USD", "stock")

	mock.ExpectQuery(`SELECT id, ticker, currency, asset_type FROM asset WHERE is_active = true`).
		WillReturnRows(rows)

	list, err := repo.GetAllAssets(context.Background())
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "AAPL", list[0].Ticker)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Additional errors for coverage
func TestRepository_Errors(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, user_id`).WillReturnError(errors.New("err"))
	_, err := repo.GetPortfoliosByUserID(context.Background(), "u1")
	assert.Error(t, err)

	mock.ExpectQuery(`SELECT id, user_id`).WillReturnError(errors.New("err"))
	_, err = repo.GetPortfolioByID(context.Background(), "p1", "u1")
	assert.Error(t, err)

	mock.ExpectExec(`DELETE FROM portfolio`).WillReturnError(errors.New("err"))
	err = repo.DeletePortfolio(context.Background(), "p1", "u1")
	assert.Error(t, err)

	tx := &Transaction{}
	mock.ExpectQuery(`INSERT INTO transaction`).WillReturnError(errors.New("err"))
	_, err = repo.CreateTransaction(context.Background(), tx)
	assert.Error(t, err)

	mock.ExpectQuery(`SELECT t.id`).WillReturnError(errors.New("err"))
	_, err = repo.GetTransactionsByPortfolioID(context.Background(), "p1", "u1")
	assert.Error(t, err)

	mock.ExpectExec(`DELETE FROM transaction t`).WillReturnError(errors.New("err"))
	err = repo.DeleteTransaction(context.Background(), "tx1", "p1", "u1")
	assert.Error(t, err)

	mock.ExpectBegin().WillReturnError(errors.New("err"))
	err = repo.SaveDailyPrices(context.Background(), "a1", []DailyPrice{{}})
	assert.Error(t, err)

	mock.ExpectQuery(`SELECT asset_id, price_date, close_price`).WillReturnError(errors.New("err"))
	_, err = repo.GetDailyPrices(context.Background(), "a1", time.Now(), time.Now())
	assert.Error(t, err)

	mock.ExpectQuery(`SELECT id FROM asset`).WillReturnError(errors.New("err"))
	_, err = repo.GetAssetByTicker(context.Background(), "AAPL")
	assert.Error(t, err)

	mock.ExpectQuery(`INSERT INTO asset`).WillReturnError(errors.New("err"))
	_, err = repo.CreateAsset(context.Background(), "AAPL", "Apple", "EQUITY_US", "USD")
	assert.Error(t, err)

	mock.ExpectQuery(`SELECT id, ticker, currency FROM asset`).WillReturnError(errors.New("err"))
	_, err = repo.GetAllAssets(context.Background())
	assert.Error(t, err)
}

func TestRepository_ScanErrors(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	// For GetPortfoliosByUserID
	rows1 := pgxmock.NewRows([]string{"id", "user_id", "name", "base_currency", "created_at"}).AddRow(nil, nil, nil, nil, "badtime")
	mock.ExpectQuery(`SELECT id, user_id, name, base_currency, created_at`).
		WithArgs("u1").
		WillReturnRows(rows1)
	_, err := repo.GetPortfoliosByUserID(context.Background(), "u1")
	assert.Error(t, err)

	// For GetTransactionsByPortfolioID
	rows2 := pgxmock.NewRows([]string{"id", "portfolio_id", "user_id", "asset_id", "transaction_type", "quantity", "price_per_unit", "transaction_date"}).AddRow(nil, nil, nil, nil, nil, nil, nil, "badtime")
	mock.ExpectQuery(`SELECT`).
		WithArgs("p1", "u1").
		WillReturnRows(rows2)
	_, err = repo.GetTransactionsByPortfolioID(context.Background(), "p1", "u1")
	assert.Error(t, err)

	// For GetDailyPrices
	rows3 := pgxmock.NewRows([]string{"asset_id", "price_date", "close_price"}).AddRow(nil, "badtime", nil)
	mock.ExpectQuery(`SELECT asset_id`).
		WithArgs("a1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows3)
	_, err = repo.GetDailyPrices(context.Background(), "a1", time.Now(), time.Now())
	assert.Error(t, err)

	// For GetAllAssets
	rows4 := pgxmock.NewRows([]string{"id", "ticker", "currency", "asset_type"}).AddRow("id", "ticker", "currency", "asset_type").RowError(0, errors.New("scan error"))
	mock.ExpectQuery(`SELECT id, ticker, currency, asset_type FROM asset WHERE is_active = true`).
		WillReturnRows(rows4)
	_, err = repo.GetAllAssets(context.Background())
	assert.Error(t, err)
}

func TestRepository_UpdateTransaction(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	tx := Transaction{
		ID:           "tx1",
		PortfolioID:  "p1",
		Type:         "BUY",
		Quantity:     10,
		UnitPrice:    100,
		TotalCost:    1010,
		Fee:          10,
		ExchangeRate: 1.0,
		ExecutedAt:   now,
	}

	mock.ExpectExec(`UPDATE transaction`).
		WithArgs(tx.Type, tx.Quantity, tx.UnitPrice, tx.TotalCost, tx.Fee, tx.ExchangeRate, tx.ExecutedAt, tx.ID, tx.PortfolioID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateTransaction(context.Background(), tx)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateTransaction_NotFound(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	now := time.Now()
	tx := Transaction{
		ID:           "tx1",
		PortfolioID:  "p1",
		Type:         "BUY",
		Quantity:     10,
		UnitPrice:    100,
		TotalCost:    1010,
		Fee:          10,
		ExchangeRate: 1.0,
		ExecutedAt:   now,
	}

	mock.ExpectExec(`UPDATE transaction`).
		WithArgs(tx.Type, tx.Quantity, tx.UnitPrice, tx.TotalCost, tx.Fee, tx.ExchangeRate, tx.ExecutedAt, tx.ID, tx.PortfolioID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.UpdateTransaction(context.Background(), tx)
	assert.ErrorContains(t, err, "não encontrada")
}

func TestRepository_GetDailyPricesBatch(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

	t.Run("Empty AssetIDs", func(t *testing.T) {
		res, err := repo.GetDailyPricesBatch(context.Background(), []string{}, start, end)
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"asset_id", "price_date", "close_price"}).
			AddRow("a1", start, 30.0)
		mock.ExpectQuery(`SELECT asset_id, price_date, close_price FROM asset_daily_price`).
			WithArgs([]string{"a1"}, start, end).
			WillReturnRows(rows)

		res, err := repo.GetDailyPricesBatch(context.Background(), []string{"a1"}, start, end)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetAssetAndCurrencyByTicker(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "currency"}).AddRow("a1", "BRL")
	mock.ExpectQuery(`SELECT id, currency FROM asset WHERE ticker = UPPER\(\$1\)`).
		WithArgs("PETR4.SA").
		WillReturnRows(rows)

	id, curr, err := repo.GetAssetAndCurrencyByTicker(context.Background(), "PETR4.SA")
	assert.NoError(t, err)
	assert.Equal(t, "a1", id)
	assert.Equal(t, "BRL", curr)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetExchangeRateByDate(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	dt := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"close_price"}).AddRow(5.15)
		mock.ExpectQuery(`SELECT p\.close_price FROM asset_daily_price p JOIN asset a ON p\.asset_id = a\.id WHERE a\.ticker = \$1 AND p\.price_date <= \$2 ORDER BY p\.price_date DESC LIMIT 1`).
			WithArgs("USDBRL=X", dt).
			WillReturnRows(rows)

		rate, err := repo.GetExchangeRateByDate(context.Background(), "USDBRL=X", dt)
		assert.NoError(t, err)
		assert.Equal(t, 5.15, rate)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetOldestPriceDate(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	dt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"oldest_date"}).AddRow(dt)
		mock.ExpectQuery(`SELECT MIN\(price_date\) FROM asset_daily_price WHERE asset_id = \$1`).
			WithArgs("a1").
			WillReturnRows(rows)

		oldest, err := repo.GetOldestPriceDate(context.Background(), "a1")
		assert.NoError(t, err)
		assert.Equal(t, dt, oldest)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_AssetEvents(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	cum := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	pay := time.Date(2024, 4, 20, 0, 0, 0, 0, time.UTC)
	ev := AssetEvent{
		AssetID:     "a1",
		Type:        "Dividendo",
		GrossAmount: 1.5,
		NetAmount:   1.5,
		CumDate:     cum,
		PaymentDate: pay,
	}

	t.Run("Upsert", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO asset_event`).
			WithArgs(ev.AssetID, ev.Type, ev.GrossAmount, ev.NetAmount, ev.CumDate, ev.PaymentDate).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.UpsertAssetEvent(context.Background(), ev)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetAssetEvents", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"id", "asset_id", "type", "gross_amount", "net_amount", "cum_date", "payment_date", "updated_at"}).
			AddRow("e1", "a1", "Dividendo", 1.5, 1.5, cum, pay, time.Now())
		mock.ExpectQuery(`SELECT id, asset_id, type, gross_amount, net_amount, cum_date, payment_date, updated_at FROM asset_event WHERE asset_id = \$1`).
			WithArgs("a1").
			WillReturnRows(rows)

		list, err := repo.GetAssetEvents(context.Background(), "a1")
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetAssetEventsByDate", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"id", "asset_id", "type", "gross_amount", "net_amount", "cum_date", "payment_date", "updated_at"}).
			AddRow("e1", "a1", "Dividendo", 1.5, 1.5, cum, pay, time.Now())
		mock.ExpectQuery(`SELECT id, asset_id, type, gross_amount, net_amount, cum_date, payment_date, updated_at FROM asset_event WHERE asset_id = \$1 AND cum_date = \$2`).
			WithArgs("a1", cum).
			WillReturnRows(rows)

		list, err := repo.GetAssetEventsByDate(context.Background(), "a1", cum)
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateAssetEventValueByID", func(t *testing.T) {
		mock.ExpectExec(`UPDATE asset_event SET gross_amount = \$1, net_amount = \$2, payment_date = \$3`).
			WithArgs(2.0, 2.0, pay, "e1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateAssetEventValueByID(context.Background(), "e1", 2.0, 2.0, pay)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetAssetMetadataByTicker(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	t.Run("Success", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"id", "ticker", "name", "asset_type", "currency"}).
			AddRow("a1", "PETR4.SA", "Petrobras", "STOCK_BR", "BRL")
		mock.ExpectQuery(`SELECT id, ticker, name, asset_type, currency FROM asset WHERE ticker = UPPER\(\$1\)`).
			WithArgs("PETR4.SA").
			WillReturnRows(rows)

		meta, err := repo.GetAssetMetadataByTicker(context.Background(), "PETR4.SA")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "a1", meta.ID)
		assert.Equal(t, "STOCK_BR", meta.AssetType)
		assert.Equal(t, "BRL", meta.Currency)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, ticker, name, asset_type, currency FROM asset WHERE ticker = UPPER\(\$1\)`).
			WithArgs("INVALID").
			WillReturnError(pgx.ErrNoRows)

		meta, err := repo.GetAssetMetadataByTicker(context.Background(), "INVALID")
		assert.Error(t, err)
		assert.Nil(t, meta)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_UpdateAssetType(t *testing.T) {
	mock, repo := setupRepoTest(t)
	defer mock.Close()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE asset SET asset_type = \$1, updated_at = NOW\(\) WHERE id = \$2`).
			WithArgs("FII", "a1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateAssetType(context.Background(), "a1", "FII")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Error", func(t *testing.T) {
		mock.ExpectExec(`UPDATE asset SET asset_type = \$1, updated_at = NOW\(\) WHERE id = \$2`).
			WithArgs("FII", "a1").
			WillReturnError(errors.New("db error"))

		err := repo.UpdateAssetType(context.Background(), "a1", "FII")
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

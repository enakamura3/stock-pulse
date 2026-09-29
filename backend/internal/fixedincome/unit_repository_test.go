package fixedincome

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func setupUnitRepoTest(t *testing.T) (pgxmock.PgxPoolIface, Repository) {
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	return mock, NewRepository(mock)
}

func TestRepository_AssetCRUD(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("CreateAsset Success", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		a := &Asset{
			PortfolioID:  "p1",
			Institution:  "Nubank",
			Type:         "CDB",
			DebtType:     "POS_FIXADO",
			Indexer:      "CDI",
			Rate:         110.0,
			MaturityDate: now.AddDate(1, 0, 0),
		}

		rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("a1", now, now)
		mock.ExpectQuery(`INSERT INTO fixed_income_assets`).
			WithArgs(a.PortfolioID, a.Institution, a.Type, a.DebtType, a.Indexer, a.Rate, a.MaturityDate).
			WillReturnRows(rows)

		res, err := repo.CreateAsset(ctx, a)
		assert.NoError(t, err)
		assert.Equal(t, "a1", res.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateAsset Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		a := &Asset{PortfolioID: "p1"}
		mock.ExpectQuery(`INSERT INTO fixed_income_assets`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("db error"))

		res, err := repo.CreateAsset(ctx, a)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetAssetsByPortfolio Success", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "institution", "type", "debt_type", "indexer", "rate", "maturity_date", "created_at", "updated_at"}
		rows := pgxmock.NewRows(cols).
			AddRow("a1", "p1", "Nubank", "CDB", "POS_FIXADO", "CDI", 110.0, now, now, now).
			AddRow("a2", "p1", "Itaú", "LCI", "PRE_FIXADO", "PRE", 12.0, now, now, now)

		mock.ExpectQuery(`SELECT id, portfolio_id, institution, type, debt_type, indexer, rate, maturity_date, created_at, updated_at FROM fixed_income_assets WHERE portfolio_id = \$1`).
			WithArgs("p1").
			WillReturnRows(rows)

		assets, err := repo.GetAssetsByPortfolio(ctx, "p1")
		assert.NoError(t, err)
		assert.Len(t, assets, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetAssetsByPortfolio Query Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1").WillReturnError(errors.New("query err"))
		assets, err := repo.GetAssetsByPortfolio(ctx, "p1")
		assert.Error(t, err)
		assert.Nil(t, assets)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetAssetsByPortfolio Scan Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "institution", "type", "debt_type", "indexer", "rate", "maturity_date", "created_at", "updated_at"}
		rows := pgxmock.NewRows(cols).AddRow("a1", "p1", "Nubank", "CDB", "POS_FIXADO", "CDI", 110.0, now, now, now)
		rows.RowError(0, errors.New("scan err"))

		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1").WillReturnRows(rows)
		assets, err := repo.GetAssetsByPortfolio(ctx, "p1")
		assert.Error(t, err)
		assert.Nil(t, assets)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetAssetByID Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "institution", "type", "debt_type", "indexer", "rate", "maturity_date", "created_at", "updated_at"}
		rows := pgxmock.NewRows(cols).AddRow("a1", "p1", "Nubank", "CDB", "POS_FIXADO", "CDI", 110.0, now, now, now)
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("a1").WillReturnRows(rows)

		a, err := repo.GetAssetByID(ctx, "a1")
		assert.NoError(t, err)
		assert.Equal(t, "a1", a.ID)

		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("a2").WillReturnError(errors.New("not found"))
		a2, err := repo.GetAssetByID(ctx, "a2")
		assert.Error(t, err)
		assert.Nil(t, a2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateAsset Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		a := &Asset{
			ID:           "a1",
			Institution:  "Nubank",
			Type:         "CDB",
			DebtType:     "POS_FIXADO",
			Indexer:      "CDI",
			Rate:         115.0,
			MaturityDate: now,
		}

		mock.ExpectExec(`UPDATE fixed_income_assets`).
			WithArgs(a.Institution, a.Type, a.DebtType, a.Indexer, a.Rate, a.MaturityDate, a.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateAsset(ctx, a)
		assert.NoError(t, err)

		mock.ExpectExec(`UPDATE fixed_income_assets`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("update err"))
		err = repo.UpdateAsset(ctx, a)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteAsset Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM fixed_income_assets WHERE id = \$1`).
			WithArgs("a1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.DeleteAsset(ctx, "a1")
		assert.NoError(t, err)

		mock.ExpectExec(`DELETE FROM fixed_income_assets WHERE id = \$1`).
			WithArgs("a2").
			WillReturnError(errors.New("delete err"))

		err = repo.DeleteAsset(ctx, "a2")
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_TransactionCRUD(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("CreateTransaction Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		tx := &Transaction{
			AssetID: "a1",
			Type:    "APORTE",
			Amount:  1000.0,
			Date:    now,
		}

		mock.ExpectQuery(`INSERT INTO fixed_income_transactions`).
			WithArgs(tx.AssetID, tx.Type, tx.Amount, tx.Date).
			WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow("tx1", now))

		res, err := repo.CreateTransaction(ctx, tx)
		assert.NoError(t, err)
		assert.Equal(t, "tx1", res.ID)

		mock.ExpectQuery(`INSERT INTO fixed_income_transactions`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("tx err"))
		_, err = repo.CreateTransaction(ctx, tx)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTransactionsByAsset Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "asset_id", "type", "amount", "date", "created_at"}
		rows := pgxmock.NewRows(cols).AddRow("tx1", "a1", "APORTE", 1000.0, now, now)

		mock.ExpectQuery(`SELECT id, asset_id, type, amount, date, created_at FROM fixed_income_transactions WHERE asset_id = \$1`).
			WithArgs("a1").
			WillReturnRows(rows)

		txs, err := repo.GetTransactionsByAsset(ctx, "a1")
		assert.NoError(t, err)
		assert.Len(t, txs, 1)

		mock.ExpectQuery(`SELECT id, asset_id`).WithArgs("a1").WillReturnError(errors.New("q err"))
		_, err = repo.GetTransactionsByAsset(ctx, "a1")
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows(cols).AddRow("tx1", "a1", "APORTE", 1000.0, now, now)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT id, asset_id`).WithArgs("a1").WillReturnRows(rowsScanErr)
		_, err = repo.GetTransactionsByAsset(ctx, "a1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTransactionsByPortfolio Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "asset_id", "type", "amount", "date", "created_at"}
		rows := pgxmock.NewRows(cols).AddRow("tx1", "a1", "APORTE", 1000.0, now, now)

		mock.ExpectQuery(`SELECT t.id, t.asset_id, t.type, t.amount, t.date, t.created_at FROM fixed_income_transactions t JOIN fixed_income_assets a`).
			WithArgs("p1").
			WillReturnRows(rows)

		txs, err := repo.GetTransactionsByPortfolio(ctx, "p1")
		assert.NoError(t, err)
		assert.Len(t, txs, 1)

		mock.ExpectQuery(`SELECT t.id, t.asset_id`).WithArgs("p1").WillReturnError(errors.New("q err"))
		_, err = repo.GetTransactionsByPortfolio(ctx, "p1")
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows(cols).AddRow("tx1", "a1", "APORTE", 1000.0, now, now)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT t.id, t.asset_id`).WithArgs("p1").WillReturnRows(rowsScanErr)
		_, err = repo.GetTransactionsByPortfolio(ctx, "p1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTransactionByID Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "asset_id", "type", "amount", "date", "created_at"}
		rows := pgxmock.NewRows(cols).AddRow("tx1", "a1", "APORTE", 1000.0, now, now)

		mock.ExpectQuery(`SELECT id, asset_id, type, amount, date, created_at FROM fixed_income_transactions WHERE id = \$1`).
			WithArgs("tx1").
			WillReturnRows(rows)

		tx, err := repo.GetTransactionByID(ctx, "tx1")
		assert.NoError(t, err)
		assert.Equal(t, "tx1", tx.ID)

		mock.ExpectQuery(`SELECT id, asset_id`).WithArgs("tx2").WillReturnError(errors.New("not found"))
		_, err = repo.GetTransactionByID(ctx, "tx2")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateTransaction Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		tx := &Transaction{
			Type:   "RESGATE",
			Amount: 500.0,
			Date:   now,
		}

		mock.ExpectExec(`UPDATE fixed_income_transactions`).
			WithArgs(tx.Type, tx.Amount, tx.Date, "tx1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdateTransaction(ctx, "tx1", tx)
		assert.NoError(t, err)

		mock.ExpectExec(`UPDATE fixed_income_transactions`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("update err"))
		err = repo.UpdateTransaction(ctx, "tx1", tx)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteTransaction Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM fixed_income_transactions WHERE id = \$1`).
			WithArgs("tx1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.DeleteTransaction(ctx, "tx1")
		assert.NoError(t, err)

		mock.ExpectExec(`DELETE FROM fixed_income_transactions WHERE id = \$1`).
			WithArgs("tx1").
			WillReturnError(errors.New("del err"))

		err = repo.DeleteTransaction(ctx, "tx1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_IndexRatesAndExecuteInTx(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("SaveIndexRates Empty, Success, BeginErr, ExecErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// Empty
		err := repo.SaveIndexRates(ctx, nil)
		assert.NoError(t, err)

		// Success
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO index_rates`).
			WithArgs("CDI", now, 10.5).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		rates := []IndexRate{{Indexer: "CDI", Date: now, Rate: 10.5}}
		err = repo.SaveIndexRates(ctx, rates)
		assert.NoError(t, err)

		// Begin error
		mock.ExpectBegin().WillReturnError(errors.New("begin err"))
		err = repo.SaveIndexRates(ctx, rates)
		assert.Error(t, err)

		// Exec error
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO index_rates`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("exec err"))
		mock.ExpectRollback()
		err = repo.SaveIndexRates(ctx, rates)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetIndexRates Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"indexer", "date", "rate"}
		rows := pgxmock.NewRows(cols).AddRow("CDI", now, 10.5)

		mock.ExpectQuery(`SELECT indexer, date, rate FROM index_rates WHERE indexer = \$1`).
			WithArgs("CDI", now, now).
			WillReturnRows(rows)

		res, err := repo.GetIndexRates(ctx, "CDI", now, now)
		assert.NoError(t, err)
		assert.Len(t, res, 1)

		mock.ExpectQuery(`SELECT indexer, date`).WithArgs("CDI", now, now).WillReturnError(errors.New("q err"))
		_, err = repo.GetIndexRates(ctx, "CDI", now, now)
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows(cols).AddRow("CDI", now, 10.5)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT indexer, date`).WithArgs("CDI", now, now).WillReturnRows(rowsScanErr)
		_, err = repo.GetIndexRates(ctx, "CDI", now, now)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetLatestIndexRate Success, ErrNoRows, OtherErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"indexer", "date", "rate"}
		rows := pgxmock.NewRows(cols).AddRow("SELIC", now, 10.75)

		mock.ExpectQuery(`SELECT indexer, date, rate FROM index_rates WHERE indexer = \$1 ORDER BY date DESC LIMIT 1`).
			WithArgs("SELIC").
			WillReturnRows(rows)

		rt, err := repo.GetLatestIndexRate(ctx, "SELIC")
		assert.NoError(t, err)
		assert.Equal(t, 10.75, rt.Rate)

		// ErrNoRows
		mock.ExpectQuery(`SELECT indexer, date`).WithArgs("SELIC").WillReturnError(pgx.ErrNoRows)
		rt, err = repo.GetLatestIndexRate(ctx, "SELIC")
		assert.NoError(t, err)
		assert.Nil(t, rt)

		// Other error
		mock.ExpectQuery(`SELECT indexer, date`).WithArgs("SELIC").WillReturnError(errors.New("other err"))
		_, err = repo.GetLatestIndexRate(ctx, "SELIC")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ExecuteInTx BeginErr, FnErr, Success", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectBegin().WillReturnError(errors.New("begin err"))
		err := repo.ExecuteInTx(ctx, func(tx pgx.Tx) error { return nil })
		assert.Error(t, err)

		mock.ExpectBegin()
		mock.ExpectRollback()
		err = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error { return errors.New("fn err") })
		assert.Error(t, err)

		mock.ExpectBegin()
		mock.ExpectCommit()
		err = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error { return nil })
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_TreasuryAssetsAndTransactions(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("GetTreasuryAssetByTicker tx vs nil", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// tx != nil
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM asset WHERE ticker = \$1`).WithArgs("TD2026").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("as1"))
		mock.ExpectCommit()

		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			id, err := repo.GetTreasuryAssetByTicker(ctx, tx, "TD2026")
			assert.NoError(t, err)
			assert.Equal(t, "as1", id)
			return nil
		})

		// tx == nil
		mock.ExpectQuery(`SELECT id FROM asset WHERE ticker = \$1`).WithArgs("TD2026").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("as1"))
		id, err := repo.GetTreasuryAssetByTicker(ctx, nil, "TD2026")
		assert.NoError(t, err)
		assert.Equal(t, "as1", id)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateTreasuryAsset tx vs nil, asset err, treasury err", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// tx != nil success
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO asset`).WithArgs("TD2029", "Tesouro 2029").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("as29"))
		mock.ExpectExec(`INSERT INTO treasury_assets`).WithArgs("as29", "SELIC", now, false).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			id, err := repo.CreateTreasuryAsset(ctx, tx, "TD2029", "Tesouro 2029", "SELIC", now, false)
			assert.NoError(t, err)
			assert.Equal(t, "as29", id)
			return nil
		})

		// tx == nil success
		mock.ExpectQuery(`INSERT INTO asset`).WithArgs("TD2029", "Tesouro 2029").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("as29"))
		mock.ExpectExec(`INSERT INTO treasury_assets`).WithArgs("as29", "SELIC", now, false).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		id, err := repo.CreateTreasuryAsset(ctx, nil, "TD2029", "Tesouro 2029", "SELIC", now, false)
		assert.NoError(t, err)
		assert.Equal(t, "as29", id)

		// Asset query err
		mock.ExpectQuery(`INSERT INTO asset`).WithArgs("TD2029", "Tesouro 2029").WillReturnError(errors.New("asset err"))
		_, err = repo.CreateTreasuryAsset(ctx, nil, "TD2029", "Tesouro 2029", "SELIC", now, false)
		assert.Error(t, err)

		// Treasury exec err
		mock.ExpectQuery(`INSERT INTO asset`).WithArgs("TD2029", "Tesouro 2029").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("as29"))
		mock.ExpectExec(`INSERT INTO treasury_assets`).WithArgs("as29", "SELIC", now, false).WillReturnError(errors.New("treasury err"))
		_, err = repo.CreateTreasuryAsset(ctx, nil, "TD2029", "Tesouro 2029", "SELIC", now, false)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateTreasurySubscription and RedemptionPlaceholder", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// Subscription tx != nil
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO treasury_transactions`).
			WithArgs("p1", "a1", 10.0, 100.0, 5.5, 10.0, now).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("sub1"))
		mock.ExpectCommit()

		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			id, err := repo.CreateTreasurySubscription(ctx, tx, "p1", "a1", 10.0, 100.0, 5.5, now)
			assert.NoError(t, err)
			assert.Equal(t, "sub1", id)
			return nil
		})

		// Subscription tx == nil
		mock.ExpectQuery(`INSERT INTO treasury_transactions`).
			WithArgs("p1", "a1", 10.0, 100.0, 5.5, 10.0, now).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("sub1"))
		id, err := repo.CreateTreasurySubscription(ctx, nil, "p1", "a1", 10.0, 100.0, 5.5, now)
		assert.NoError(t, err)
		assert.Equal(t, "sub1", id)

		// Subscription Error
		mock.ExpectQuery(`INSERT INTO treasury_transactions`).WithArgs("p1", "a1", 10.0, 100.0, 5.5, 10.0, now).WillReturnError(errors.New("sub err"))
		_, err = repo.CreateTreasurySubscription(ctx, nil, "p1", "a1", 10.0, 100.0, 5.5, now)
		assert.Error(t, err)

		// Redemption tx != nil
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO treasury_transactions`).
			WithArgs("p1", "a1", 5.0, 105.0, 5.5, now).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("red1"))
		mock.ExpectCommit()

		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			id, err := repo.CreateTreasuryRedemptionPlaceholder(ctx, tx, "p1", "a1", 5.0, 105.0, 5.5, now)
			assert.NoError(t, err)
			assert.Equal(t, "red1", id)
			return nil
		})

		// Redemption tx == nil
		mock.ExpectQuery(`INSERT INTO treasury_transactions`).
			WithArgs("p1", "a1", 5.0, 105.0, 5.5, now).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("red1"))
		id, err = repo.CreateTreasuryRedemptionPlaceholder(ctx, nil, "p1", "a1", 5.0, 105.0, 5.5, now)
		assert.NoError(t, err)
		assert.Equal(t, "red1", id)

		// Redemption Error
		mock.ExpectQuery(`INSERT INTO treasury_transactions`).WithArgs("p1", "a1", 5.0, 105.0, 5.5, now).WillReturnError(errors.New("red err"))
		_, err = repo.CreateTreasuryRedemptionPlaceholder(ctx, nil, "p1", "a1", 5.0, 105.0, 5.5, now)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetActiveLotsForAsset tx vs nil, query err, scan err", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "asset_id", "type", "quantity", "unit_price", "contracted_rate", "remaining_quantity", "transaction_date"}
		rows := pgxmock.NewRows(cols).AddRow("sub1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now)

		// tx == nil success
		mock.ExpectQuery(`SELECT id, portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date FROM treasury_transactions`).
			WithArgs("p1", "a1").
			WillReturnRows(rows)
		lots, err := repo.GetActiveLotsForAsset(ctx, nil, "p1", "a1")
		assert.NoError(t, err)
		assert.Len(t, lots, 1)

		// tx != nil success
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1", "a1").WillReturnRows(pgxmock.NewRows(cols).AddRow("sub1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			res, err := repo.GetActiveLotsForAsset(ctx, tx, "p1", "a1")
			assert.NoError(t, err)
			assert.Len(t, res, 1)
			return nil
		})

		// Query error
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1", "a1").WillReturnError(errors.New("lot err"))
		_, err = repo.GetActiveLotsForAsset(ctx, nil, "p1", "a1")
		assert.Error(t, err)

		// Scan error
		rowsErr := pgxmock.NewRows(cols).AddRow("sub1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now)
		rowsErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1", "a1").WillReturnRows(rowsErr)
		_, err = repo.GetActiveLotsForAsset(ctx, nil, "p1", "a1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateLotRemainingQuantity, CreateDepletionLink, UpdateRedemptionFinancials", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// UpdateLotRemainingQuantity tx vs nil
		mock.ExpectExec(`UPDATE treasury_transactions SET remaining_quantity = \$1`).WithArgs(5.0, "l1").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err := repo.UpdateLotRemainingQuantity(ctx, nil, "l1", 5.0)
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE treasury_transactions SET remaining_quantity = \$1`).WithArgs(5.0, "l1").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.UpdateLotRemainingQuantity(ctx, tx, "l1", 5.0)
		})

		// CreateDepletionLink tx vs nil
		mock.ExpectExec(`INSERT INTO treasury_depletions`).WithArgs("sub1", "red1", 2.0).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		err = repo.CreateDepletionLink(ctx, nil, "sub1", "red1", 2.0)
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO treasury_depletions`).WithArgs("sub1", "red1", 2.0).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.CreateDepletionLink(ctx, tx, "sub1", "red1", 2.0)
		})

		// UpdateRedemptionFinancials tx vs nil
		mock.ExpectExec(`UPDATE treasury_transactions SET gross_amount = \$1`).
			WithArgs(100.0, 1.0, 15.0, 0.5, 83.5, "red1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err = repo.UpdateRedemptionFinancials(ctx, nil, "red1", 100.0, 1.0, 15.0, 0.5, 83.5)
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE treasury_transactions SET gross_amount = \$1`).
			WithArgs(100.0, 1.0, 15.0, 0.5, 83.5, "red1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.UpdateRedemptionFinancials(ctx, tx, "red1", 100.0, 1.0, 15.0, 0.5, 83.5)
		})

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_HolidaysAndSelic(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("GetAnbimaHolidays Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`SELECT holiday_date FROM anbima_holidays`).
			WillReturnRows(pgxmock.NewRows([]string{"holiday_date"}).AddRow(now))

		hols, err := repo.GetAnbimaHolidays(ctx)
		assert.NoError(t, err)
		assert.True(t, hols[now.Format("2006-01-02")])

		mock.ExpectQuery(`SELECT holiday_date`).WillReturnError(errors.New("hol err"))
		_, err = repo.GetAnbimaHolidays(ctx)
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows([]string{"holiday_date"}).AddRow(now)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT holiday_date`).WillReturnRows(rowsScanErr)
		_, err = repo.GetAnbimaHolidays(ctx)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetSeededHolidayYears Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`SELECT DISTINCT EXTRACT\(YEAR FROM holiday_date\)::int FROM anbima_holidays ORDER BY 1`).
			WillReturnRows(pgxmock.NewRows([]string{"year"}).AddRow(2025).AddRow(2026))

		years, err := repo.GetSeededHolidayYears(ctx)
		assert.NoError(t, err)
		assert.Equal(t, []int{2025, 2026}, years)

		mock.ExpectQuery(`SELECT DISTINCT`).WillReturnError(errors.New("yr err"))
		_, err = repo.GetSeededHolidayYears(ctx)
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows([]string{"year"}).AddRow(2025)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT DISTINCT`).WillReturnRows(rowsScanErr)
		_, err = repo.GetSeededHolidayYears(ctx)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SaveAnbimaHolidays Empty, BeginErr, ExecErr, Success", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// Empty
		err := repo.SaveAnbimaHolidays(ctx, nil)
		assert.NoError(t, err)

		dates := []time.Time{now}

		// Begin error
		mock.ExpectBegin().WillReturnError(errors.New("begin err"))
		err = repo.SaveAnbimaHolidays(ctx, dates)
		assert.Error(t, err)

		// Exec error
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO anbima_holidays`).WithArgs(now).WillReturnError(errors.New("exec err"))
		mock.ExpectRollback()
		err = repo.SaveAnbimaHolidays(ctx, dates)
		assert.Error(t, err)

		// Success
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO anbima_holidays`).WithArgs(now).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()
		err = repo.SaveAnbimaHolidays(ctx, dates)
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetSelicRates Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`SELECT date, rate FROM index_rates WHERE indexer = 'SELIC'`).
			WillReturnRows(pgxmock.NewRows([]string{"date", "rate"}).AddRow(now, 10.75))

		rates, err := repo.GetSelicRates(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 10.75, rates[now.Format("2006-01-02")])

		mock.ExpectQuery(`SELECT date, rate`).WillReturnError(errors.New("selic err"))
		_, err = repo.GetSelicRates(ctx)
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows([]string{"date", "rate"}).AddRow(now, 10.75)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT date, rate`).WillReturnRows(rowsScanErr)
		_, err = repo.GetSelicRates(ctx)
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTotalSelicInvested tx vs nil", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// tx == nil
		mock.ExpectQuery(`SELECT COALESCE\(SUM\(remaining_quantity \* unit_price\), 0\) FROM treasury_transactions`).
			WithArgs("p1").
			WillReturnRows(pgxmock.NewRows([]string{"sum"}).AddRow(5000.0))

		tot, err := repo.GetTotalSelicInvested(ctx, nil, "p1")
		assert.NoError(t, err)
		assert.Equal(t, 5000.0, tot)

		// tx != nil
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT COALESCE\(SUM\(remaining_quantity \* unit_price\), 0\)`).
			WithArgs("p1").
			WillReturnRows(pgxmock.NewRows([]string{"sum"}).AddRow(5000.0))
		mock.ExpectCommit()

		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			res, err := repo.GetTotalSelicInvested(ctx, tx, "p1")
			assert.NoError(t, err)
			assert.Equal(t, 5000.0, res)
			return nil
		})

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_PositionsPerformanceAndReset(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("GetActiveSubscriptionLots Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "asset_id", "type", "quantity", "unit_price", "contracted_rate", "remaining_quantity", "transaction_date"}
		rows := pgxmock.NewRows(cols).AddRow("sub1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now)

		mock.ExpectQuery(`SELECT id, portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date FROM treasury_transactions WHERE portfolio_id = \$1`).
			WithArgs("p1").
			WillReturnRows(rows)

		lots, err := repo.GetActiveSubscriptionLots(ctx, "p1")
		assert.NoError(t, err)
		assert.Len(t, lots, 1)

		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1").WillReturnError(errors.New("q err"))
		_, err = repo.GetActiveSubscriptionLots(ctx, "p1")
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows(cols).AddRow("sub1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1").WillReturnRows(rowsScanErr)
		_, err = repo.GetActiveSubscriptionLots(ctx, "p1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTreasuryPerformancePoints Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"price_date", "value", "theoretical"}
		rows := pgxmock.NewRows(cols).AddRow("2026-01-01", 1050.0, 1000.0)

		mock.ExpectQuery(`SELECT price_date, SUM\(selling_price\) as value, SUM\(theoretical_price\) as theoretical FROM treasury_prices`).
			WithArgs("p1").
			WillReturnRows(rows)

		pts, err := repo.GetTreasuryPerformancePoints(ctx, "p1")
		assert.NoError(t, err)
		assert.Len(t, pts, 1)
		assert.Equal(t, 1050.0, pts[0].Value)

		mock.ExpectQuery(`SELECT price_date`).WithArgs("p1").WillReturnError(errors.New("q err"))
		_, err = repo.GetTreasuryPerformancePoints(ctx, "p1")
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows(cols).AddRow("2026-01-01", 1050.0, 1000.0)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT price_date`).WithArgs("p1").WillReturnRows(rowsScanErr)
		_, err = repo.GetTreasuryPerformancePoints(ctx, "p1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTreasuryTransactionsList Success, QueryErr, ScanErr", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "ticker", "treasury_type", "maturity_date", "has_coupons", "type", "quantity", "unit_price", "contracted_rate", "transaction_date"}
		rows := pgxmock.NewRows(cols).AddRow("tx1", "TD2026", "PREFIXADO", now, false, "SUBSCRIPTION", 1.0, 100.0, 10.0, now)

		mock.ExpectQuery(`SELECT t.id, a.ticker, ta.treasury_type, ta.maturity_date, ta.has_coupons`).
			WithArgs("p1").
			WillReturnRows(rows)

		list, err := repo.GetTreasuryTransactionsList(ctx, "p1")
		assert.NoError(t, err)
		assert.Len(t, list, 1)

		mock.ExpectQuery(`SELECT t.id, a.ticker`).WithArgs("p1").WillReturnError(errors.New("q err"))
		_, err = repo.GetTreasuryTransactionsList(ctx, "p1")
		assert.Error(t, err)

		rowsScanErr := pgxmock.NewRows(cols).AddRow("tx1", "TD2026", "PREFIXADO", now, false, "SUBSCRIPTION", 1.0, 100.0, 10.0, now)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT t.id, a.ticker`).WithArgs("p1").WillReturnRows(rowsScanErr)
		_, err = repo.GetTreasuryTransactionsList(ctx, "p1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTreasuryAssetDetails Success and Error", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		mock.ExpectQuery(`SELECT a.ticker, ta.treasury_type, ta.maturity_date, ta.has_coupons FROM treasury_assets ta`).
			WithArgs("as1").
			WillReturnRows(pgxmock.NewRows([]string{"ticker", "treasury_type", "maturity_date", "has_coupons"}).AddRow("TD2026", "PREFIXADO", now, false))

		ticker, tType, mat, coupons, err := repo.GetTreasuryAssetDetails(ctx, "as1")
		assert.NoError(t, err)
		assert.Equal(t, "TD2026", ticker)
		assert.Equal(t, "PREFIXADO", tType)
		assert.False(t, coupons)
		assert.Equal(t, now, mat)

		mock.ExpectQuery(`SELECT a.ticker`).WithArgs("as2").WillReturnError(errors.New("details err"))
		_, _, _, _, err = repo.GetTreasuryAssetDetails(ctx, "as2")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetTreasuryTransactionByID tx vs nil", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "asset_id", "type", "quantity", "unit_price", "contracted_rate", "remaining_quantity", "transaction_date", "gross_amount", "iof_tax", "ir_tax", "b3_fee", "net_amount", "created_at", "updated_at"}
		gross := 1000.0
		iof := 0.0
		ir := 0.0
		b3 := 0.0
		net := 1000.0
		rows := pgxmock.NewRows(cols).AddRow("tx1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now, &gross, &iof, &ir, &b3, &net, now, now)

		// tx == nil
		mock.ExpectQuery(`SELECT id, portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date, gross_amount, iof_tax, ir_tax, b3_fee, net_amount, created_at, updated_at FROM treasury_transactions`).
			WithArgs("tx1").
			WillReturnRows(rows)
		tx, err := repo.GetTreasuryTransactionByID(ctx, nil, "tx1")
		assert.NoError(t, err)
		assert.Equal(t, "tx1", tx.ID)

		// tx != nil
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("tx1").WillReturnRows(pgxmock.NewRows(cols).AddRow("tx1", "p1", "a1", "SUBSCRIPTION", 10.0, 100.0, 5.5, 10.0, now, &gross, &iof, &ir, &b3, &net, now, now))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(txItem pgx.Tx) error {
			res, err := repo.GetTreasuryTransactionByID(ctx, txItem, "tx1")
			assert.NoError(t, err)
			assert.Equal(t, "tx1", res.ID)
			return nil
		})

		// Error
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("tx2").WillReturnError(errors.New("not found"))
		_, err = repo.GetTreasuryTransactionByID(ctx, nil, "tx2")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateTreasuryTransaction and DeleteTreasuryTransactionByID tx vs nil", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		tt := &TreasuryTransaction{
			ID:                "tx1",
			AssetID:           "a1",
			Type:              "SUBSCRIPTION",
			Quantity:          10.0,
			UnitPrice:         100.0,
			ContractedRate:    5.5,
			RemainingQuantity: 10.0,
			TransactionDate:   now,
		}

		// Update tx == nil
		mock.ExpectExec(`UPDATE treasury_transactions SET asset_id = \$1`).
			WithArgs(tt.AssetID, tt.Type, tt.Quantity, tt.UnitPrice, tt.ContractedRate, tt.RemainingQuantity, tt.TransactionDate, tt.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err := repo.UpdateTreasuryTransaction(ctx, nil, tt)
		assert.NoError(t, err)

		// Update tx != nil
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE treasury_transactions SET asset_id = \$1`).
			WithArgs(tt.AssetID, tt.Type, tt.Quantity, tt.UnitPrice, tt.ContractedRate, tt.RemainingQuantity, tt.TransactionDate, tt.ID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.UpdateTreasuryTransaction(ctx, tx, tt)
		})

		// Delete tx == nil
		mock.ExpectExec(`DELETE FROM treasury_transactions WHERE id = \$1`).WithArgs("tx1").WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err = repo.DeleteTreasuryTransactionByID(ctx, nil, "tx1")
		assert.NoError(t, err)

		// Delete tx != nil
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM treasury_transactions WHERE id = \$1`).WithArgs("tx1").WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.DeleteTreasuryTransactionByID(ctx, tx, "tx1")
		})

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteDepletionsByAsset, ResetSubscriptionsRemainingQuantity, ResetRedemptionFinancials tx vs nil", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		// DeleteDepletionsByAsset tx == nil and tx != nil
		mock.ExpectExec(`DELETE FROM treasury_depletions WHERE subscription_transaction_id IN`).
			WithArgs("p1", "a1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repo.DeleteDepletionsByAsset(ctx, nil, "p1", "a1")
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM treasury_depletions WHERE subscription_transaction_id IN`).
			WithArgs("p1", "a1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.DeleteDepletionsByAsset(ctx, tx, "p1", "a1")
		})

		// ResetSubscriptionsRemainingQuantity tx == nil and tx != nil
		mock.ExpectExec(`UPDATE treasury_transactions SET remaining_quantity = quantity WHERE portfolio_id = \$1 AND asset_id = \$2 AND type = 'SUBSCRIPTION'`).
			WithArgs("p1", "a1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err = repo.ResetSubscriptionsRemainingQuantity(ctx, nil, "p1", "a1")
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE treasury_transactions SET remaining_quantity = quantity WHERE portfolio_id = \$1 AND asset_id = \$2 AND type = 'SUBSCRIPTION'`).
			WithArgs("p1", "a1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.ResetSubscriptionsRemainingQuantity(ctx, tx, "p1", "a1")
		})

		// ResetRedemptionFinancials tx == nil and tx != nil
		mock.ExpectExec(`UPDATE treasury_transactions SET gross_amount = NULL, iof_tax = NULL, ir_tax = NULL, b3_fee = NULL, net_amount = NULL WHERE portfolio_id = \$1 AND asset_id = \$2 AND type = 'REDEMPTION'`).
			WithArgs("p1", "a1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err = repo.ResetRedemptionFinancials(ctx, nil, "p1", "a1")
		assert.NoError(t, err)

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE treasury_transactions SET gross_amount = NULL, iof_tax = NULL, ir_tax = NULL, b3_fee = NULL, net_amount = NULL WHERE portfolio_id = \$1 AND asset_id = \$2 AND type = 'REDEMPTION'`).
			WithArgs("p1", "a1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			return repo.ResetRedemptionFinancials(ctx, tx, "p1", "a1")
		})

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetRedemptionsForAsset tx vs nil, query err, scan err", func(t *testing.T) {
		mock, repo := setupUnitRepoTest(t)
		defer mock.Close()

		cols := []string{"id", "portfolio_id", "asset_id", "type", "quantity", "unit_price", "contracted_rate", "remaining_quantity", "transaction_date"}
		rows := pgxmock.NewRows(cols).AddRow("red1", "p1", "a1", "REDEMPTION", 5.0, 100.0, 5.5, 0.0, now)

		// tx == nil
		mock.ExpectQuery(`SELECT id, portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date FROM treasury_transactions WHERE portfolio_id = \$1 AND asset_id = \$2 AND type = 'REDEMPTION'`).
			WithArgs("p1", "a1").
			WillReturnRows(rows)
		reds, err := repo.GetRedemptionsForAsset(ctx, nil, "p1", "a1")
		assert.NoError(t, err)
		assert.Len(t, reds, 1)

		// tx != nil
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1", "a1").WillReturnRows(pgxmock.NewRows(cols).AddRow("red1", "p1", "a1", "REDEMPTION", 5.0, 100.0, 5.5, 0.0, now))
		mock.ExpectCommit()
		_ = repo.ExecuteInTx(ctx, func(tx pgx.Tx) error {
			res, err := repo.GetRedemptionsForAsset(ctx, tx, "p1", "a1")
			assert.NoError(t, err)
			assert.Len(t, res, 1)
			return nil
		})

		// Query error
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1", "a1").WillReturnError(errors.New("q err"))
		_, err = repo.GetRedemptionsForAsset(ctx, nil, "p1", "a1")
		assert.Error(t, err)

		// Scan error
		rowsScanErr := pgxmock.NewRows(cols).AddRow("red1", "p1", "a1", "REDEMPTION", 5.0, 100.0, 5.5, 0.0, now)
		rowsScanErr.RowError(0, errors.New("scan err"))
		mock.ExpectQuery(`SELECT id, portfolio_id`).WithArgs("p1", "a1").WillReturnRows(rowsScanErr)
		_, err = repo.GetRedemptionsForAsset(ctx, nil, "p1", "a1")
		assert.Error(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

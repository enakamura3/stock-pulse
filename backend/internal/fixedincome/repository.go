package fixedincome

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateAsset(ctx context.Context, asset *Asset) (*Asset, error)
	GetAssetsByPortfolio(ctx context.Context, portfolioID string) ([]Asset, error)
	GetAssetByID(ctx context.Context, assetID string) (*Asset, error)
	UpdateAsset(ctx context.Context, asset *Asset) error
	DeleteAsset(ctx context.Context, assetID string) error

	CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error)
	GetTransactionsByAsset(ctx context.Context, assetID string) ([]Transaction, error)
	GetTransactionsByPortfolio(ctx context.Context, portfolioID string) ([]Transaction, error)
	GetTransactionByID(ctx context.Context, txID string) (*Transaction, error)
	UpdateTransaction(ctx context.Context, txID string, tx *Transaction) error
	DeleteTransaction(ctx context.Context, txID string) error

	SaveIndexRates(ctx context.Context, rates []IndexRate) error
	GetIndexRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error)
	GetLatestIndexRate(ctx context.Context, indexer string) (*IndexRate, error)
}

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db}
}

func (r *repository) CreateAsset(ctx context.Context, a *Asset) (*Asset, error) {
	query := `
		INSERT INTO fixed_income_assets (portfolio_id, institution, type, debt_type, indexer, rate, maturity_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		a.PortfolioID, a.Institution, a.Type, a.DebtType, a.Indexer, a.Rate, a.MaturityDate,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset: %w", err)
	}
	return a, nil
}

func (r *repository) GetAssetsByPortfolio(ctx context.Context, portfolioID string) ([]Asset, error) {
	query := `
		SELECT id, portfolio_id, institution, type, debt_type, indexer, rate, maturity_date, created_at, updated_at
		FROM fixed_income_assets
		WHERE portfolio_id = $1
	`
	rows, err := r.db.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.PortfolioID, &a.Institution, &a.Type, &a.DebtType, &a.Indexer, &a.Rate, &a.MaturityDate, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

func (r *repository) GetAssetByID(ctx context.Context, assetID string) (*Asset, error) {
	query := `
		SELECT id, portfolio_id, institution, type, debt_type, indexer, rate, maturity_date, created_at, updated_at
		FROM fixed_income_assets
		WHERE id = $1
	`
	var a Asset
	err := r.db.QueryRow(ctx, query, assetID).Scan(&a.ID, &a.PortfolioID, &a.Institution, &a.Type, &a.DebtType, &a.Indexer, &a.Rate, &a.MaturityDate, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *repository) UpdateAsset(ctx context.Context, a *Asset) error {
	query := `
		UPDATE fixed_income_assets
		SET institution = $1, type = $2, debt_type = $3, indexer = $4, rate = $5, maturity_date = $6, updated_at = NOW()
		WHERE id = $7
	`
	_, err := r.db.Exec(ctx, query, a.Institution, a.Type, a.DebtType, a.Indexer, a.Rate, a.MaturityDate, a.ID)
	return err
}

func (r *repository) DeleteAsset(ctx context.Context, assetID string) error {
	query := `DELETE FROM fixed_income_assets WHERE id = $1`
	_, err := r.db.Exec(ctx, query, assetID)
	return err
}

func (r *repository) CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	query := `
		INSERT INTO fixed_income_transactions (asset_id, type, amount, date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, tx.AssetID, tx.Type, tx.Amount, tx.Date).Scan(&tx.ID, &tx.CreatedAt)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *repository) GetTransactionsByAsset(ctx context.Context, assetID string) ([]Transaction, error) {
	query := `
		SELECT id, asset_id, type, amount, date, created_at
		FROM fixed_income_transactions
		WHERE asset_id = $1
		ORDER BY date ASC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.AssetID, &tx.Type, &tx.Amount, &tx.Date, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *repository) GetTransactionsByPortfolio(ctx context.Context, portfolioID string) ([]Transaction, error) {
	query := `
		SELECT t.id, t.asset_id, t.type, t.amount, t.date, t.created_at
		FROM fixed_income_transactions t
		JOIN fixed_income_assets a ON t.asset_id = a.id
		WHERE a.portfolio_id = $1
		ORDER BY t.date ASC, t.created_at ASC
	`
	rows, err := r.db.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.AssetID, &tx.Type, &tx.Amount, &tx.Date, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *repository) GetTransactionByID(ctx context.Context, txID string) (*Transaction, error) {
	query := `
		SELECT id, asset_id, type, amount, date, created_at
		FROM fixed_income_transactions
		WHERE id = $1
	`
	var tx Transaction
	err := r.db.QueryRow(ctx, query, txID).Scan(&tx.ID, &tx.AssetID, &tx.Type, &tx.Amount, &tx.Date, &tx.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *repository) UpdateTransaction(ctx context.Context, txID string, tx *Transaction) error {
	query := `
		UPDATE fixed_income_transactions
		SET type = $1, amount = $2, date = $3
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, tx.Type, tx.Amount, tx.Date, txID)
	return err
}

func (r *repository) DeleteTransaction(ctx context.Context, txID string) error {
	query := `DELETE FROM fixed_income_transactions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, txID)
	return err
}

func (r *repository) SaveIndexRates(ctx context.Context, rates []IndexRate) error {
	if len(rates) == 0 {
		return nil
	}

	query := `
		INSERT INTO index_rates (indexer, date, rate)
		VALUES ($1, $2, $3)
		ON CONFLICT (indexer, date) DO UPDATE SET rate = EXCLUDED.rate
	`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, rt := range rates {
		if _, err := tx.Exec(ctx, query, rt.Indexer, rt.Date, rt.Rate); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *repository) GetIndexRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	query := `
		SELECT indexer, date, rate
		FROM index_rates
		WHERE indexer = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`
	rows, err := r.db.Query(ctx, query, indexer, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []IndexRate
	for rows.Next() {
		var rt IndexRate
		if err := rows.Scan(&rt.Indexer, &rt.Date, &rt.Rate); err != nil {
			return nil, err
		}
		rates = append(rates, rt)
	}
	return rates, nil
}

func (r *repository) GetLatestIndexRate(ctx context.Context, indexer string) (*IndexRate, error) {
	query := `
		SELECT indexer, date, rate
		FROM index_rates
		WHERE indexer = $1
		ORDER BY date DESC
		LIMIT 1
	`
	var rt IndexRate
	err := r.db.QueryRow(ctx, query, indexer).Scan(&rt.Indexer, &rt.Date, &rt.Rate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

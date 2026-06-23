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

	ExecuteInTx(ctx context.Context, fn func(tx pgx.Tx) error) error

	// Treasury Asset Operations
	GetTreasuryAssetByTicker(ctx context.Context, tx pgx.Tx, ticker string) (string, error)
	CreateTreasuryAsset(ctx context.Context, tx pgx.Tx, ticker string, name string, treasuryType string, maturityDate time.Time, hasCoupons bool) (string, error)

	// Treasury Transaction Operations
	CreateTreasurySubscription(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string, quantity float64, unitPrice float64, contractedRate float64, transactionDate time.Time) (string, error)
	CreateTreasuryRedemptionPlaceholder(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string, quantity float64, unitPrice float64, contractedRate float64, transactionDate time.Time) (string, error)
	GetActiveLotsForAsset(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) ([]TreasuryTransaction, error)
	UpdateLotRemainingQuantity(ctx context.Context, tx pgx.Tx, lotID string, remainingQuantity float64) error
	CreateDepletionLink(ctx context.Context, tx pgx.Tx, subID string, redID string, quantity float64) error
	UpdateRedemptionFinancials(ctx context.Context, tx pgx.Tx, redemptionID string, grossAmount float64, iofTax float64, irTax float64, b3Fee float64, netAmount float64) error

	// Holiday & Exemption Queries
	GetAnbimaHolidays(ctx context.Context) (map[string]bool, error)
	GetSeededHolidayYears(ctx context.Context) ([]int, error)
	SaveAnbimaHolidays(ctx context.Context, dates []time.Time) error
	GetSelicRates(ctx context.Context) (map[string]float64, error)
	GetTotalSelicInvested(ctx context.Context, tx pgx.Tx, portfolioID string) (float64, error)

	// Positions & Performance
	GetActiveSubscriptionLots(ctx context.Context, portfolioID string) ([]TreasuryTransaction, error)
	GetTreasuryPerformancePoints(ctx context.Context, portfolioID string) ([]TreasuryPerfPoint, error)
	GetTreasuryAssetDetails(ctx context.Context, assetID string) (ticker string, treasuryType string, maturityDate time.Time, hasCoupons bool, err error)
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
	defer func() { _ = tx.Rollback(ctx) }()

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

func (r *repository) ExecuteInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *repository) GetTreasuryAssetByTicker(ctx context.Context, tx pgx.Tx, ticker string) (string, error) {
	var id string
	query := "SELECT id FROM asset WHERE ticker = $1"
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, ticker).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx, query, ticker).Scan(&id)
	}
	return id, err
}

func (r *repository) CreateTreasuryAsset(ctx context.Context, tx pgx.Tx, ticker string, name string, treasuryType string, maturityDate time.Time, hasCoupons bool) (string, error) {
	var id string
	insertAssetQuery := "INSERT INTO asset (ticker, name, asset_type, currency) VALUES ($1, $2, 'TREASURY', 'BRL') RETURNING id"
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, insertAssetQuery, ticker, name).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx, insertAssetQuery, ticker, name).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	insertTreasuryQuery := "INSERT INTO treasury_assets (id, treasury_type, maturity_date, has_coupons) VALUES ($1, $2, $3, $4)"
	if tx != nil {
		_, err = tx.Exec(ctx, insertTreasuryQuery, id, treasuryType, maturityDate, hasCoupons)
	} else {
		_, err = r.db.Exec(ctx, insertTreasuryQuery, id, treasuryType, maturityDate, hasCoupons)
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *repository) CreateTreasurySubscription(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string, quantity float64, unitPrice float64, contractedRate float64, transactionDate time.Time) (string, error) {
	var id string
	query := `
		INSERT INTO treasury_transactions 
		(portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date)
		VALUES ($1, $2, 'SUBSCRIPTION', $3, $4, $5, $6, $7) RETURNING id`
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, portfolioID, assetID, quantity, unitPrice, contractedRate, quantity, transactionDate).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx, query, portfolioID, assetID, quantity, unitPrice, contractedRate, quantity, transactionDate).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *repository) CreateTreasuryRedemptionPlaceholder(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string, quantity float64, unitPrice float64, contractedRate float64, transactionDate time.Time) (string, error) {
	var id string
	query := `
		INSERT INTO treasury_transactions 
		(portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date)
		VALUES ($1, $2, 'REDEMPTION', $3, $4, $5, 0.0, $6) RETURNING id`
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, portfolioID, assetID, quantity, unitPrice, contractedRate, transactionDate).Scan(&id)
	} else {
		err = r.db.QueryRow(ctx, query, portfolioID, assetID, quantity, unitPrice, contractedRate, transactionDate).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *repository) GetActiveLotsForAsset(ctx context.Context, tx pgx.Tx, portfolioID string, assetID string) ([]TreasuryTransaction, error) {
	query := `
		SELECT id, portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date
		FROM treasury_transactions 
		WHERE portfolio_id = $1 AND asset_id = $2 AND type = 'SUBSCRIPTION' AND remaining_quantity > 0
		ORDER BY transaction_date ASC, created_at ASC`
	var rows pgx.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(ctx, query, portfolioID, assetID)
	} else {
		rows, err = r.db.Query(ctx, query, portfolioID, assetID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lots []TreasuryTransaction
	for rows.Next() {
		var t TreasuryTransaction
		err = rows.Scan(&t.ID, &t.PortfolioID, &t.AssetID, &t.Type, &t.Quantity, &t.UnitPrice, &t.ContractedRate, &t.RemainingQuantity, &t.TransactionDate)
		if err != nil {
			return nil, err
		}
		lots = append(lots, t)
	}
	return lots, nil
}

func (r *repository) UpdateLotRemainingQuantity(ctx context.Context, tx pgx.Tx, lotID string, remainingQuantity float64) error {
	query := "UPDATE treasury_transactions SET remaining_quantity = $1, updated_at = NOW() WHERE id = $2"
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, remainingQuantity, lotID)
	} else {
		_, err = r.db.Exec(ctx, query, remainingQuantity, lotID)
	}
	return err
}

func (r *repository) CreateDepletionLink(ctx context.Context, tx pgx.Tx, subID string, redID string, quantity float64) error {
	query := `
		INSERT INTO treasury_depletions (subscription_transaction_id, redemption_transaction_id, quantity)
		VALUES ($1, $2, $3)`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, subID, redID, quantity)
	} else {
		_, err = r.db.Exec(ctx, query, subID, redID, quantity)
	}
	return err
}

func (r *repository) UpdateRedemptionFinancials(ctx context.Context, tx pgx.Tx, redemptionID string, grossAmount float64, iofTax float64, irTax float64, b3Fee float64, netAmount float64) error {
	query := `
		UPDATE treasury_transactions 
		SET gross_amount = $1, iof_tax = $2, ir_tax = $3, b3_fee = $4, net_amount = $5, updated_at = NOW()
		WHERE id = $6`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, grossAmount, iofTax, irTax, b3Fee, netAmount, redemptionID)
	} else {
		_, err = r.db.Exec(ctx, query, grossAmount, iofTax, irTax, b3Fee, netAmount, redemptionID)
	}
	return err
}

func (r *repository) GetAnbimaHolidays(ctx context.Context) (map[string]bool, error) {
	query := "SELECT holiday_date FROM anbima_holidays"
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	holidays := make(map[string]bool)
	for rows.Next() {
		var hd time.Time
		if err := rows.Scan(&hd); err != nil {
			return nil, err
		}
		holidays[hd.Format("2006-01-02")] = true
	}
	return holidays, nil
}

func (r *repository) GetSeededHolidayYears(ctx context.Context) ([]int, error) {
	query := "SELECT DISTINCT EXTRACT(YEAR FROM holiday_date)::int FROM anbima_holidays ORDER BY 1"
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, nil
}

func (r *repository) SaveAnbimaHolidays(ctx context.Context, dates []time.Time) error {
	if len(dates) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		INSERT INTO anbima_holidays (holiday_date, description)
		VALUES ($1, 'Feriado Nacional')
		ON CONFLICT (holiday_date) DO NOTHING`
	for _, d := range dates {
		if _, err := tx.Exec(ctx, query, d); err != nil {
			return fmt.Errorf("SaveAnbimaHolidays: failed to insert %s: %w", d.Format("2006-01-02"), err)
		}
	}
	return tx.Commit(ctx)
}

func (r *repository) GetSelicRates(ctx context.Context) (map[string]float64, error) {
	query := "SELECT date, rate FROM index_rates WHERE indexer = 'SELIC'"
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rates := make(map[string]float64)
	for rows.Next() {
		var sd time.Time
		var sr float64
		if err := rows.Scan(&sd, &sr); err != nil {
			return nil, err
		}
		rates[sd.Format("2006-01-02")] = sr
	}
	return rates, nil
}

func (r *repository) GetTotalSelicInvested(ctx context.Context, tx pgx.Tx, portfolioID string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(remaining_quantity * unit_price), 0)
		FROM treasury_transactions t
		JOIN treasury_assets ta ON t.asset_id = ta.id
		WHERE t.portfolio_id = $1 AND ta.treasury_type = 'SELIC' AND t.type = 'SUBSCRIPTION'`
	var total float64
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, portfolioID).Scan(&total)
	} else {
		err = r.db.QueryRow(ctx, query, portfolioID).Scan(&total)
	}
	return total, err
}

func (r *repository) GetActiveSubscriptionLots(ctx context.Context, portfolioID string) ([]TreasuryTransaction, error) {
	query := `
		SELECT id, portfolio_id, asset_id, type, quantity, unit_price, contracted_rate, remaining_quantity, transaction_date
		FROM treasury_transactions
		WHERE portfolio_id = $1 AND type = 'SUBSCRIPTION' AND remaining_quantity > 0`
	rows, err := r.db.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lots []TreasuryTransaction
	for rows.Next() {
		var t TreasuryTransaction
		err = rows.Scan(&t.ID, &t.PortfolioID, &t.AssetID, &t.Type, &t.Quantity, &t.UnitPrice, &t.ContractedRate, &t.RemainingQuantity, &t.TransactionDate)
		if err != nil {
			return nil, err
		}
		lots = append(lots, t)
	}
	return lots, nil
}

func (r *repository) GetTreasuryPerformancePoints(ctx context.Context, portfolioID string) ([]TreasuryPerfPoint, error) {
	query := `
		SELECT price_date, SUM(selling_price) as value, SUM(theoretical_price) as theoretical
		FROM treasury_prices p
		JOIN treasury_transactions t ON p.asset_id = t.asset_id
		WHERE t.portfolio_id = $1
		GROUP BY price_date ORDER BY price_date ASC`
	rows, err := r.db.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []TreasuryPerfPoint
	for rows.Next() {
		var date string
		var val, th float64
		err = rows.Scan(&date, &val, &th)
		if err != nil {
			return nil, err
		}
		points = append(points, TreasuryPerfPoint{
			Date:          date,
			Value:         val,
			TotalInvested: th,
		})
	}
	return points, nil
}

func (r *repository) GetTreasuryAssetDetails(ctx context.Context, assetID string) (ticker string, treasuryType string, maturityDate time.Time, hasCoupons bool, err error) {
	query := `
		SELECT a.ticker, ta.treasury_type, ta.maturity_date, ta.has_coupons
		FROM treasury_assets ta
		JOIN asset a ON ta.id = a.id
		WHERE ta.id = $1`
	err = r.db.QueryRow(ctx, query, assetID).Scan(&ticker, &treasuryType, &maturityDate, &hasCoupons)
	return
}

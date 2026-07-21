package portfolio

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Portfolio representa o agrupamento de ativos pertencente a um usuário.
type Portfolio struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	BaseCurrency string    `json:"base_currency"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
}

// Transaction representa uma operação de COMPRA ou VENDA de um ativo.
type Transaction struct {
	ID           string    `json:"id"`
	PortfolioID  string    `json:"portfolio_id"`
	AssetID      string    `json:"asset_id"`
	Ticker       string    `json:"ticker,omitempty"`
	AssetName    string    `json:"asset_name,omitempty"`
	AssetType    string    `json:"asset_type,omitempty"`
	Currency     string    `json:"currency,omitempty"`
	Type         string    `json:"type"` // "BUY" ou "SELL"
	Quantity     float64   `json:"quantity"`
	UnitPrice    float64   `json:"unit_price"`
	TotalCost    float64   `json:"total_cost"`
	ExchangeRate float64   `json:"exchange_rate"`
	ExecutedAt   time.Time `json:"executed_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// Position representa a consolidação de um ativo em uma carteira (Preço Médio).
type Position struct {
	AssetID      string  `json:"asset_id"`
	Ticker       string  `json:"ticker"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Currency     string  `json:"currency"`
	Quantity     float64 `json:"quantity"`
	AveragePrice float64 `json:"average_price"`
	TotalCost    float64 `json:"total_cost"`

	// Preenchidos dinamicamente no serviço
	CurrentPrice       float64 `json:"current_price,omitempty"`
	CurrentValue       float64 `json:"current_value,omitempty"`
	ProfitLoss         float64 `json:"profit_loss,omitempty"`
	ReturnPercent      float64 `json:"return_percent,omitempty"`
	DailyChange        float64 `json:"daily_change,omitempty"`
	DailyChangePercent float64 `json:"daily_change_percent,omitempty"`
	GrahamValue        float64 `json:"graham_value,omitempty"`
	BazinValue         float64 `json:"bazin_value,omitempty"`
	PVP                float64 `json:"pvp,omitempty"`
	PE                 float64 `json:"pe,omitempty"`
	DividendYield      float64 `json:"dividend_yield,omitempty"`
}

// DailyPrice representa a cotação histórica diária de um ativo.
type DailyPrice struct {
	AssetID    string    `json:"asset_id"`
	PriceDate  time.Time `json:"price_date"`
	ClosePrice float64   `json:"close_price"`
}

// DBTX define a interface necessária para realizar queries e abstrair pgxpool.Pool para testes.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Repository lida com as interações de banco de dados do módulo de Portfólio.
type Repository struct {
	db DBTX
}

// NewRepository cria uma nova instância de Repository.
func NewRepository(db DBTX) *Repository {
	return &Repository{db: db}
}

// CreatePortfolio insere um novo portfólio no banco de dados.
func (r *Repository) CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM portfolio WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		count = 0
	}
	isDefault := (count == 0)

	query := `
		INSERT INTO portfolio (user_id, name, base_currency, is_default, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, user_id, name, base_currency, is_default, created_at
	`
	p := &Portfolio{}
	err = r.db.QueryRow(ctx, query, userID, name, baseCurrency, isDefault).Scan(
		&p.ID,
		&p.UserID,
		&p.Name,
		&p.BaseCurrency,
		&p.IsDefault,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar portfolio: %w", err)
	}
	return p, nil
}

// GetPortfoliosByUserID lista todos os portfólios pertencentes a um usuário.
func (r *Repository) GetPortfoliosByUserID(ctx context.Context, userID string) ([]Portfolio, error) {
	query := `
		SELECT id, user_id, name, base_currency, is_default, created_at
		FROM portfolio
		WHERE user_id = $1
		ORDER BY is_default DESC, name ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Portfolio
	for rows.Next() {
		var p Portfolio
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.BaseCurrency, &p.IsDefault, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// GetPortfolioByID recupera um portfólio validando o proprietário (Anti-IDOR).
func (r *Repository) GetPortfolioByID(ctx context.Context, id, userID string) (*Portfolio, error) {
	query := `
		SELECT id, user_id, name, base_currency, is_default, created_at
		FROM portfolio
		WHERE id = $1 AND user_id = $2
	`
	p := &Portfolio{}
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&p.ID,
		&p.UserID,
		&p.Name,
		&p.BaseCurrency,
		&p.IsDefault,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// SetDefaultPortfolio marca uma carteira como padrão e desmarca todas as outras do mesmo usuário.
func (r *Repository) SetDefaultPortfolio(ctx context.Context, portfolioID, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portfolio WHERE id = $1 AND user_id = $2)`, portfolioID, userID).Scan(&exists)
	if err != nil || !exists {
		return fmt.Errorf("portfólio não encontrado ou permissão negada")
	}

	_, err = tx.Exec(ctx, `UPDATE portfolio SET is_default = false WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("erro ao resetar carteiras padrão: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE portfolio SET is_default = true WHERE id = $1 AND user_id = $2`, portfolioID, userID)
	if err != nil {
		return fmt.Errorf("erro ao definir carteira padrão: %w", err)
	}

	return tx.Commit(ctx)
}

// DeletePortfolio apaga um portfólio do banco de dados (cascading apaga transações).
func (r *Repository) DeletePortfolio(ctx context.Context, id, userID string) error {
	var isDefault bool
	_ = r.db.QueryRow(ctx, `SELECT is_default FROM portfolio WHERE id = $1 AND user_id = $2`, id, userID).Scan(&isDefault)

	query := `
		DELETE FROM portfolio
		WHERE id = $1 AND user_id = $2
	`
	cmd, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("portfólio não encontrado ou permissão negada")
	}

	if isDefault {
		_, _ = r.db.Exec(ctx, `
			UPDATE portfolio
			SET is_default = true
			WHERE id = (
				SELECT id FROM portfolio WHERE user_id = $1 ORDER BY created_at ASC LIMIT 1
			)
		`, userID)
	}

	return nil
}

// CreateTransaction registra uma operação de compra/venda na carteira.
func (r *Repository) CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error) {
	query := `
		INSERT INTO transaction (portfolio_id, asset_id, type, quantity, unit_price, total_cost, exchange_rate, executed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query,
		tx.PortfolioID,
		tx.AssetID,
		tx.Type,
		tx.Quantity,
		tx.UnitPrice,
		tx.TotalCost,
		tx.ExchangeRate,
		tx.ExecutedAt,
	).Scan(&tx.ID, &tx.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("erro ao inserir transação: %w", err)
	}
	return tx, nil
}

// GetTransactionsByPortfolioID lista as transações de um portfólio validando a posse (Anti-IDOR).
func (r *Repository) GetTransactionsByPortfolioID(ctx context.Context, portfolioID, userID string) ([]Transaction, error) {
	query := `
		SELECT t.id, t.portfolio_id, t.asset_id, t.type, t.quantity, t.unit_price, t.total_cost, t.exchange_rate, t.executed_at, t.created_at,
		       a.ticker, a.name, a.asset_type, a.currency
		FROM transaction t
		INNER JOIN asset a ON t.asset_id = a.id
		INNER JOIN portfolio p ON t.portfolio_id = p.id
		WHERE t.portfolio_id = $1 AND p.user_id = $2
		ORDER BY t.executed_at DESC, t.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, portfolioID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Transaction
	for rows.Next() {
		var tx Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.PortfolioID,
			&tx.AssetID,
			&tx.Type,
			&tx.Quantity,
			&tx.UnitPrice,
			&tx.TotalCost,
			&tx.ExchangeRate,
			&tx.ExecutedAt,
			&tx.CreatedAt,
			&tx.Ticker,
			&tx.AssetName,
			&tx.AssetType,
			&tx.Currency,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, tx)
	}
	return list, nil
}

// DeleteTransaction apaga uma transação verificando o isolamento de tenant (Anti-IDOR).
func (r *Repository) DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error {
	query := `
		DELETE FROM transaction t
		USING portfolio p
		WHERE t.portfolio_id = p.id 
		AND t.id = $1 
		AND t.portfolio_id = $2 
		AND p.user_id = $3
	`
	cmd, err := r.db.Exec(ctx, query, txID, portfolioID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("transação não encontrada ou permissão negada")
	}
	return nil
}

// SaveDailyPrices realiza inserção em lote na tabela asset_daily_price.
func (r *Repository) SaveDailyPrices(ctx context.Context, assetID string, prices []DailyPrice) error {
	if len(prices) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO asset_daily_price (asset_id, price_date, close_price, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (asset_id, price_date) DO NOTHING
	`

	for _, p := range prices {
		_, err := tx.Exec(ctx, query, assetID, p.PriceDate, p.ClosePrice)
		if err != nil {
			return fmt.Errorf("erro ao inserir preço diário histórico: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetDailyPrices busca a série temporal de preços históricos de um ativo.
func (r *Repository) GetDailyPrices(ctx context.Context, assetID string, startDate, endDate time.Time) ([]DailyPrice, error) {
	query := `
		SELECT asset_id, price_date, close_price
		FROM asset_daily_price
		WHERE asset_id = $1 AND price_date BETWEEN $2 AND $3
		ORDER BY price_date ASC
	`
	rows, err := r.db.Query(ctx, query, assetID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DailyPrice
	for rows.Next() {
		var p DailyPrice
		if err := rows.Scan(&p.AssetID, &p.PriceDate, &p.ClosePrice); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// GetAssetByTicker busca o ID de um ativo pelo ticker.
func (r *Repository) GetAssetByTicker(ctx context.Context, ticker string) (string, error) {
	var id string
	query := `SELECT id FROM asset WHERE UPPER(ticker) = UPPER($1)`
	err := r.db.QueryRow(ctx, query, ticker).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetAssetAndCurrencyByTicker busca o ID e a moeda de um ativo pelo ticker.
func (r *Repository) GetAssetAndCurrencyByTicker(ctx context.Context, ticker string) (string, string, error) {
	var id, currency string
	query := `SELECT id, currency FROM asset WHERE UPPER(ticker) = UPPER($1)`
	err := r.db.QueryRow(ctx, query, ticker).Scan(&id, &currency)
	if err != nil {
		return "", "", err
	}
	return id, currency, nil
}

// CreateAsset cadastra localmente um novo ativo.
func (r *Repository) CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error) {
	query := `
		INSERT INTO asset (ticker, name, asset_type, currency, created_at, updated_at)
		VALUES (UPPER($1), $2, $3, $4, NOW(), NOW())
		RETURNING id
	`
	var id string
	err := r.db.QueryRow(ctx, query, ticker, name, assetType, currency).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetAllAssets retorna todos os ativos cadastrados no banco (útil para o Daily Worker).
type AssetCompact struct {
	ID        string `json:"id"`
	Ticker    string `json:"ticker"`
	Currency  string `json:"currency"`
	AssetType string `json:"asset_type"`
}

func (r *Repository) GetAllAssets(ctx context.Context) ([]AssetCompact, error) {
	query := `SELECT id, ticker, currency, asset_type FROM asset WHERE is_active = true`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AssetCompact
	for rows.Next() {
		var a AssetCompact
		if err := rows.Scan(&a.ID, &a.Ticker, &a.Currency, &a.AssetType); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *Repository) UpdateTransaction(ctx context.Context, tx Transaction) error {
	query := `
		UPDATE transaction
		SET type = $1, quantity = $2, unit_price = $3, total_cost = $4, exchange_rate = $5, executed_at = $6
		WHERE id = $7 AND portfolio_id = $8
	`
	tag, err := r.db.Exec(ctx, query,
		tx.Type, tx.Quantity, tx.UnitPrice, tx.TotalCost, tx.ExchangeRate, tx.ExecutedAt,
		tx.ID, tx.PortfolioID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("transação não encontrada ou acesso negado")
	}
	return nil
}

// GetExchangeRateByDate obtém a taxa de câmbio histórica usando LOCF (Last Observation Carried Forward).
func (r *Repository) GetExchangeRateByDate(ctx context.Context, currencyPairTicker string, date time.Time) (float64, error) {
	query := `
		SELECT p.close_price
		FROM asset_daily_price p
		JOIN asset a ON p.asset_id = a.id
		WHERE a.ticker = $1 AND p.price_date <= $2
		ORDER BY p.price_date DESC
		LIMIT 1
	`
	var rate float64
	err := r.db.QueryRow(ctx, query, currencyPairTicker, date).Scan(&rate)
	if err != nil {
		return 0, err
	}
	return rate, nil
}

// GetOldestPriceDate retorna a data mais antiga registrada para um ativo na tabela asset_daily_price.
func (r *Repository) GetOldestPriceDate(ctx context.Context, assetID string) (time.Time, error) {
	query := `
		SELECT MIN(price_date)
		FROM asset_daily_price
		WHERE asset_id = $1
	`
	var oldestDate time.Time
	err := r.db.QueryRow(ctx, query, assetID).Scan(&oldestDate)
	if err != nil {
		return time.Time{}, err
	}
	return oldestDate, nil
}

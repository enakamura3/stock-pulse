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
	AssetID       string  `json:"asset_id"`
	Ticker        string  `json:"ticker"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Currency      string  `json:"currency"`
	Quantity      float64 `json:"quantity"`
	AveragePrice  float64 `json:"average_price"`
	TotalCost     float64 `json:"total_cost"`
	
	// Preenchidos dinamicamente no serviço
	CurrentPrice  float64 `json:"current_price,omitempty"`
	CurrentValue  float64 `json:"current_value,omitempty"`
	ProfitLoss    float64 `json:"profit_loss,omitempty"`
	ReturnPercent float64 `json:"return_percent,omitempty"`
	GrahamValue   float64 `json:"graham_value,omitempty"`
	BazinValue    float64 `json:"bazin_value,omitempty"`
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
	query := `
		INSERT INTO portfolio (user_id, name, base_currency, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, user_id, name, base_currency, created_at
	`
	p := &Portfolio{}
	err := r.db.QueryRow(ctx, query, userID, name, baseCurrency).Scan(
		&p.ID,
		&p.UserID,
		&p.Name,
		&p.BaseCurrency,
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
		SELECT id, user_id, name, base_currency, created_at
		FROM portfolio
		WHERE user_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Portfolio
	for rows.Next() {
		var p Portfolio
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.BaseCurrency, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// GetPortfolioByID recupera um portfólio validando o proprietário (Anti-IDOR).
func (r *Repository) GetPortfolioByID(ctx context.Context, id, userID string) (*Portfolio, error) {
	query := `
		SELECT id, user_id, name, base_currency, created_at
		FROM portfolio
		WHERE id = $1 AND user_id = $2
	`
	p := &Portfolio{}
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&p.ID,
		&p.UserID,
		&p.Name,
		&p.BaseCurrency,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// DeletePortfolio apaga um portfólio do banco de dados (cascading apaga transações).
func (r *Repository) DeletePortfolio(ctx context.Context, id, userID string) error {
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
	ID       string `json:"id"`
	Ticker   string `json:"ticker"`
	Currency string `json:"currency"`
}

func (r *Repository) GetAllAssets(ctx context.Context) ([]AssetCompact, error) {
	query := `SELECT id, ticker, currency FROM asset WHERE is_active = true`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AssetCompact
	for rows.Next() {
		var a AssetCompact
		if err := rows.Scan(&a.ID, &a.Ticker, &a.Currency); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

package alert

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX define a interface necessária para realizar queries e abstrair pgxpool.Pool para testes.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Alert representa o modelo de dados de um alerta de preço.
type Alert struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	AssetID     string     `json:"asset_id"`
	Ticker      string     `json:"ticker,omitempty"`
	AssetName   string     `json:"asset_name,omitempty"`
	Currency    string     `json:"currency,omitempty"`
	TargetPrice float64    `json:"target_price"`
	Condition   string     `json:"condition"` // "ABOVE" ou "BELOW"
	Status      string     `json:"status"`    // "ACTIVE", "TRIGGERED", "DISABLED"
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`

	// Dados do usuário injetados na busca do Worker
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

// Repository gerencia a persistência das regras de alertas no PostgreSQL.
type Repository struct {
	db DBTX
}

// NewRepository inicializa o repositório de Alertas.
func NewRepository(db DBTX) *Repository {
	return &Repository{
		db: db,
	}
}

// CreateAlert insere um novo alerta de preço no banco de dados.
func (r *Repository) CreateAlert(ctx context.Context, a *Alert) error {
	query := `
		INSERT INTO alert (user_id, asset_id, target_price, condition, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query, a.UserID, a.AssetID, a.TargetPrice, a.Condition, a.Status).Scan(&a.ID, &a.CreatedAt)
	return err
}

// GetAlertsByUserID retorna todos os alertas de um usuário (Anti-IDOR nativo).
func (r *Repository) GetAlertsByUserID(ctx context.Context, userID string) ([]*Alert, error) {
	query := `
		SELECT a.id, a.user_id, a.asset_id, ast.ticker, ast.name, ast.currency, a.target_price, a.condition, a.status, a.triggered_at, a.created_at
		FROM alert a
		INNER JOIN asset ast ON a.asset_id = ast.id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var a Alert
		err := rows.Scan(
			&a.ID, &a.UserID, &a.AssetID, &a.Ticker, &a.AssetName, &a.Currency,
			&a.TargetPrice, &a.Condition, &a.Status, &a.TriggeredAt, &a.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, &a)
	}

	return alerts, nil
}

// GetAlertByID busca um alerta pelo ID validando a posse do usuário (Anti-IDOR).
func (r *Repository) GetAlertByID(ctx context.Context, id string, userID string) (*Alert, error) {
	query := `
		SELECT a.id, a.user_id, a.asset_id, ast.ticker, ast.name, ast.currency, a.target_price, a.condition, a.status, a.triggered_at, a.created_at
		FROM alert a
		INNER JOIN asset ast ON a.asset_id = ast.id
		WHERE a.id = $1 AND a.user_id = $2
	`
	var a Alert
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&a.ID, &a.UserID, &a.AssetID, &a.Ticker, &a.AssetName, &a.Currency,
		&a.TargetPrice, &a.Condition, &a.Status, &a.TriggeredAt, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// DeleteAlert exclui fisicamente o alerta de preço do usuário (Anti-IDOR).
func (r *Repository) DeleteAlert(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM alert WHERE id = $1 AND user_id = $2`
	res, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("alerta não encontrado ou não pertence a este usuário")
	}
	return nil
}

// ToggleAlertStatus alterna o status entre ACTIVE e DISABLED de forma segura (Anti-IDOR).
func (r *Repository) ToggleAlertStatus(ctx context.Context, id string, userID string) (string, error) {
	query := `
		UPDATE alert
		SET status = CASE WHEN status = 'ACTIVE' THEN 'DISABLED' ELSE 'ACTIVE' END
		WHERE id = $1 AND user_id = $2
		RETURNING status
	`
	var nextStatus string
	err := r.db.QueryRow(ctx, query, id, userID).Scan(&nextStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("alerta não encontrado ou não pertence a este usuário")
		}
		return "", err
	}
	return nextStatus, nil
}

// GetActiveAlerts retorna todos os alertas ativos globalmente junto com dados de contato do usuário.
func (r *Repository) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	query := `
		SELECT a.id, a.user_id, a.asset_id, ast.ticker, ast.name, ast.currency, a.target_price, a.condition, a.status, a.triggered_at, a.created_at, u.name, u.email
		FROM alert a
		INNER JOIN asset ast ON a.asset_id = ast.id
		INNER JOIN "user" u ON a.user_id = u.id
		WHERE a.status = 'ACTIVE'
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var a Alert
		err := rows.Scan(
			&a.ID, &a.UserID, &a.AssetID, &a.Ticker, &a.AssetName, &a.Currency,
			&a.TargetPrice, &a.Condition, &a.Status, &a.TriggeredAt, &a.CreatedAt,
			&a.UserName, &a.UserEmail,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, &a)
	}

	return alerts, nil
}

// MarkAlertTriggered altera o status do alerta para TRIGGERED no momento do disparo.
func (r *Repository) MarkAlertTriggered(ctx context.Context, id string) error {
	query := `
		UPDATE alert
		SET status = 'TRIGGERED', triggered_at = NOW()
		WHERE id = $1 AND status = 'ACTIVE'
	`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("alerta não pôde ser disparado (não está ativo ou inexistente)")
	}
	return nil
}

// GetAssetByTicker verifica se o ativo com o ticker especificado existe no banco de dados.
func (r *Repository) GetAssetByTicker(ctx context.Context, ticker string) (string, error) {
	query := `SELECT id FROM asset WHERE UPPER(ticker) = UPPER($1)`
	var id string
	err := r.db.QueryRow(ctx, query, ticker).Scan(&id)
	return id, err
}

// CreateAsset cria um registro inédito de ativo no banco de dados.
func (r *Repository) CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error) {
	query := `
		INSERT INTO asset (ticker, name, asset_type, currency, created_at, updated_at)
		VALUES (UPPER($1), $2, $3, $4, NOW(), NOW())
		RETURNING id
	`
	var id string
	err := r.db.QueryRow(ctx, query, ticker, name, assetType, currency).Scan(&id)
	return id, err
}


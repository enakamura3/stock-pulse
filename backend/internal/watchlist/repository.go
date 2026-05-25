package watchlist

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Watchlist representa o agrupamento de favoritos pertencente a um usuário.
type Watchlist struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Items     []Item    `json:"items,omitempty"`
}

// Item representa um ativo vinculado a uma lista de favoritos.
type Item struct {
	ID          string    `json:"id"`
	WatchlistID string    `json:"watchlist_id"`
	AssetID     string    `json:"asset_id"`
	Ticker      string    `json:"ticker"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Currency    string    `json:"currency"`
	AddedAt     time.Time `json:"added_at"`
	
	// Campos estendidos para cotações dinâmicas injetadas pelo serviço
	Price         float64 `json:"price,omitempty"`
	Change        float64 `json:"change,omitempty"`
	ChangePercent float64 `json:"change_percent,omitempty"`
}

// Repository lida com a persistência das tabelas watchlist, watchlist_item e asset.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository cria uma nova instância de Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateWatchlist insere uma nova lista de favoritos vinculada ao usuário.
func (r *Repository) CreateWatchlist(ctx context.Context, userID, name string) (*Watchlist, error) {
	query := `
		INSERT INTO watchlist (user_id, name, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, user_id, name, created_at
	`
	w := &Watchlist{}
	err := r.db.QueryRow(ctx, query, userID, name).Scan(
		&w.ID,
		&w.UserID,
		&w.Name,
		&w.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar watchlist: %w", err)
	}
	return w, nil
}

// GetWatchlistsByUserID lista todas as listas de favoritos pertencentes a um usuário.
func (r *Repository) GetWatchlistsByUserID(ctx context.Context, userID string) ([]Watchlist, error) {
	query := `
		SELECT id, user_id, name, created_at
		FROM watchlist
		WHERE user_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []Watchlist
	for rows.Next() {
		var w Watchlist
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.CreatedAt); err != nil {
			return nil, err
		}
		lists = append(lists, w)
	}
	return lists, nil
}

// GetWatchlistByID resgata uma watchlist validando a posse (Anti-IDOR).
func (r *Repository) GetWatchlistByID(ctx context.Context, id, userID string) (*Watchlist, error) {
	query := `
		SELECT id, user_id, name, created_at
		FROM watchlist
		WHERE id = $1 AND user_id = $2
	`
	w := &Watchlist{}
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&w.ID,
		&w.UserID,
		&w.Name,
		&w.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// DeleteWatchlist apaga uma lista de favoritos validando a posse (Anti-IDOR).
func (r *Repository) DeleteWatchlist(ctx context.Context, id, userID string) error {
	query := `
		DELETE FROM watchlist
		WHERE id = $1 AND user_id = $2
	`
	cmd, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("watchlist não encontrada ou permissão negada")
	}
	return nil
}

// GetAssetByTicker verifica se o ativo com o ticker especificado existe e retorna seu ID.
func (r *Repository) GetAssetByTicker(ctx context.Context, ticker string) (string, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	query := `SELECT id FROM asset WHERE UPPER(ticker) = $1`
	var id string
	err := r.db.QueryRow(ctx, query, ticker).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// CreateAsset cria um registro inédito de ativo no banco de dados.
func (r *Repository) CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	query := `
		INSERT INTO asset (ticker, name, asset_type, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`
	var id string
	err := r.db.QueryRow(ctx, query, ticker, name, assetType, currency).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("erro ao inserir ativo: %w", err)
	}
	return id, nil
}

// AddWatchlistItem vincula um ativo a uma lista de favoritos.
func (r *Repository) AddWatchlistItem(ctx context.Context, watchlistID, assetID string) (*Item, error) {
	query := `
		INSERT INTO watchlist_item (watchlist_id, asset_id, added_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (watchlist_id, asset_id) DO UPDATE SET added_at = NOW()
		RETURNING id, watchlist_id, asset_id, added_at
	`
	item := &Item{}
	err := r.db.QueryRow(ctx, query, watchlistID, assetID).Scan(
		&item.ID,
		&item.WatchlistID,
		&item.AssetID,
		&item.AddedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao vincular ativo na watchlist: %w", err)
	}
	return item, nil
}

// RemoveWatchlistItem remove um ativo de uma lista de favoritos pelo seu ticker.
func (r *Repository) RemoveWatchlistItem(ctx context.Context, watchlistID, ticker string) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	query := `
		DELETE FROM watchlist_item
		WHERE watchlist_id = $1 
		AND asset_id IN (SELECT id FROM asset WHERE UPPER(ticker) = $2)
	`
	_, err := r.db.Exec(ctx, query, watchlistID, ticker)
	return err
}

// GetWatchlistItems lista todos os itens de uma watchlist com os detalhes do ativo mapeados.
func (r *Repository) GetWatchlistItems(ctx context.Context, watchlistID string) ([]Item, error) {
	query := `
		SELECT wi.id, wi.watchlist_id, wi.asset_id, wi.added_at, a.ticker, a.name, a.asset_type, a.currency
		FROM watchlist_item wi
		INNER JOIN asset a ON wi.asset_id = a.id
		WHERE wi.watchlist_id = $1
		ORDER BY wi.added_at DESC
	`
	rows, err := r.db.Query(ctx, query, watchlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ID,
			&item.WatchlistID,
			&item.AssetID,
			&item.AddedAt,
			&item.Ticker,
			&item.Name,
			&item.Type,
			&item.Currency,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

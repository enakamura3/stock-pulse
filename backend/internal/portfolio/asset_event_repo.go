package portfolio

import (
	"context"
	"database/sql"
	"time"
)

type AssetEvent struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	Type        string    `json:"type"`
	GrossAmount float64   `json:"gross_amount"`
	NetAmount   float64   `json:"net_amount"`
	ExDate      time.Time `json:"ex_date"`
	PaymentDate time.Time `json:"payment_date"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *Repository) UpsertAssetEvent(ctx context.Context, event AssetEvent) error {
	query := `
		INSERT INTO asset_event (asset_id, type, gross_amount, net_amount, ex_date, payment_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (asset_id, ex_date, type, gross_amount) 
		DO UPDATE SET 
			payment_date = EXCLUDED.payment_date,
			net_amount = EXCLUDED.net_amount,
			updated_at = CURRENT_TIMESTAMP
		WHERE
			asset_event.payment_date IS DISTINCT FROM EXCLUDED.payment_date OR
			asset_event.net_amount IS DISTINCT FROM EXCLUDED.net_amount
	`
	var paymentDate interface{} = event.PaymentDate
	if event.PaymentDate.IsZero() {
		paymentDate = nil
	}
	_, err := r.db.Exec(ctx, query,
		event.AssetID, event.Type, event.GrossAmount, event.NetAmount, event.ExDate, paymentDate,
	)
	return err
}

func (r *Repository) GetAssetEvents(ctx context.Context, assetID string) ([]AssetEvent, error) {
	query := `
		SELECT id, asset_id, type, gross_amount, net_amount, ex_date, payment_date, updated_at
		FROM asset_event
		WHERE asset_id = $1
		ORDER BY ex_date DESC
	`
	rows, err := r.db.Query(ctx, query, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AssetEvent
	for rows.Next() {
		var e AssetEvent
		var paymentDate sql.NullTime
		if err := rows.Scan(&e.ID, &e.AssetID, &e.Type, &e.GrossAmount, &e.NetAmount, &e.ExDate, &paymentDate, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if paymentDate.Valid {
			e.PaymentDate = paymentDate.Time
		}
		list = append(list, e)
	}
	return list, nil
}

func (r *Repository) GetAssetEventsByDate(ctx context.Context, assetID string, exDate time.Time) ([]AssetEvent, error) {
	query := `
		SELECT id, asset_id, type, gross_amount, net_amount, ex_date, payment_date, updated_at
		FROM asset_event
		WHERE asset_id = $1 AND ex_date = $2
	`
	rows, err := r.db.Query(ctx, query, assetID, exDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AssetEvent
	for rows.Next() {
		var e AssetEvent
		var paymentDate sql.NullTime
		if err := rows.Scan(&e.ID, &e.AssetID, &e.Type, &e.GrossAmount, &e.NetAmount, &e.ExDate, &paymentDate, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if paymentDate.Valid {
			e.PaymentDate = paymentDate.Time
		}
		list = append(list, e)
	}
	return list, nil
}

func (r *Repository) UpdateAssetEventValueByID(ctx context.Context, eventID string, newGross, newNet float64, newPayment time.Time) error {
	query := `
		UPDATE asset_event
		SET gross_amount = $1, net_amount = $2, payment_date = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`
	var paymentDate interface{} = newPayment
	if newPayment.IsZero() {
		paymentDate = nil
	}
	
	_, err := r.db.Exec(ctx, query, newGross, newNet, paymentDate, eventID)
	return err
}

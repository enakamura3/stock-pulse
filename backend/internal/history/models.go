package history

import (
	"context"
	"time"
)

// UnifiedTransaction represents a generic financial transaction spanning any module (RV, RF, etc.)
type UnifiedTransaction struct {
	ID           string    `json:"id"`
	PortfolioID  string    `json:"portfolio_id"`
	Module       string    `json:"module"` // "RV" or "RF"
	Date         time.Time `json:"date"`
	AssetName    string    `json:"asset_name"` // Ticker (RV) or Asset Name (RF)
	AssetType    string    `json:"asset_type"` // STOCK, FII, CDB, LCI, etc.
	Type         string    `json:"type"`       // BUY, SELL, BONUS, SPLIT, SUBSCRIPTION, REDEMPTION
	Quantity     *float64  `json:"quantity"`   // nil for RF
	UnitPrice    *float64  `json:"unit_price"` // nil for RF
	ExchangeRate *float64  `json:"exchange_rate"` // nil for RF
	TotalValue   float64   `json:"total_value"`
	Currency     string    `json:"currency"`
	MaturityDate *time.Time `json:"maturity_date,omitempty"` // For RF
}

// TransactionSource is the interface that specialized modules (RV, RF) must implement
// to provide their transactions in the unified format.
type TransactionSource interface {
	GetUnifiedTransactions(ctx context.Context, portfolioID, userID string) ([]UnifiedTransaction, error)
}

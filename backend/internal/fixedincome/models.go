package fixedincome

import (
	"time"
)

// Asset representa um título de renda fixa na carteira (ex: um CDB, um Tesouro Direto).
type Asset struct {
	ID           string    `json:"id"`
	PortfolioID  string    `json:"portfolio_id"`
	Institution  string    `json:"institution"` // ex: Itaú, Tesouro Nacional
	Type         string    `json:"type"`        // ex: CDB, LCI, LCA, TESOURO
	DebtType     string    `json:"debt_type"`   // ex: PRE, POS, HIBRIDO
	Indexer      string    `json:"indexer"`     // ex: CDI, SELIC, IPCA, PRE
	Rate         float64   `json:"rate"`        // ex: 1.10 (110%), 12.0 (12% a.a.)
	MaturityDate time.Time `json:"maturity_date"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Transaction representa um aporte ou resgate em um ativo de renda fixa.
type Transaction struct {
	ID        string    `json:"id"`
	AssetID   string    `json:"asset_id"`
	Type      string    `json:"type"` // "SUBSCRIPTION" ou "REDEMPTION"
	Amount    float64   `json:"amount"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

// Position representa a consolidação e os cálculos em tempo real de um ativo.
type Position struct {
	Asset             Asset     `json:"asset"`
	StartDate         time.Time `json:"start_date"`
	TotalInvested     float64   `json:"total_invested"`
	GrossValue        float64   `json:"gross_value"`
	NetValue          float64   `json:"net_value"`
	NetReturnPercent  float64   `json:"net_return_percent"`
	IsMatured         bool      `json:"is_matured"`
	DaysToMaturity    int       `json:"days_to_maturity"`
	TaxesCalculated   float64   `json:"taxes_calculated"` // IR + IOF deduzidos
}

// IndexRate representa o valor do índice (fator diário ou percentual) numa data.
type IndexRate struct {
	Indexer string    `json:"indexer"`
	Date    time.Time `json:"date"`
	Rate    float64   `json:"rate"`
}

// PerformancePoint represents a daily historical value for the fixed income portfolio.
type PerformancePoint struct {
	Date          string  `json:"date"`
	Value         float64 `json:"value"`
	TotalInvested float64 `json:"total_invested"`
}

// MonthlyYield represents the accrued interest for a fixed income asset in a specific month.
type MonthlyYield struct {
	AssetID     string  `json:"asset_id"`
	AssetName   string  `json:"asset_name"`
	AssetType   string  `json:"asset_type"`
	Month       string  `json:"month"` // YYYY-MM
	GrossAmount float64 `json:"gross_amount"`
	NetAmount   float64 `json:"net_amount"`
	IsAccrued   bool    `json:"is_accrued"` // Always true for fixed income yields
}

// TreasuryTxRequest represents the incoming payload for subscriptions/redemptions.
type TreasuryTxRequest struct {
	ID              string  `json:"id,omitempty"`
	Ticker          string  `json:"ticker"`
	TreasuryType    string  `json:"treasury_type"` // SELIC, PREFIXADO, IPCA+
	MaturityDate    string  `json:"maturity_date"`  // YYYY-MM-DD
	HasCoupons      bool    `json:"has_coupons"`
	Type            string  `json:"type"`            // SUBSCRIPTION, REDEMPTION
	Quantity        float64 `json:"quantity"`
	UnitPrice       float64 `json:"unit_price"`
	ContractedRate  float64 `json:"contracted_rate"`
	TransactionDate string  `json:"transaction_date"` // YYYY-MM-DD
}

// TreasuryPosition represents an active asset position in the portfolio calculated in real-time.
type TreasuryPosition struct {
	TransactionID  string    `json:"transaction_id"`
	AssetID        string    `json:"asset_id"`
	Ticker         string    `json:"ticker"`
	TreasuryType   string    `json:"treasury_type"`
	MaturityDate   time.Time `json:"maturity_date"`
	HasCoupons     bool      `json:"has_coupons"`
	StartDate      time.Time `json:"start_date"`
	Quantity       float64   `json:"quantity"`
	UnitPrice      float64   `json:"unit_price"`
	ContractedRate float64   `json:"contracted_rate"`
	TotalInvested  float64   `json:"total_invested"`
	GrossValue     float64   `json:"gross_value"`
	NetValue       float64   `json:"net_value"`
	IsMatured      bool      `json:"is_matured"`
	DaysToMaturity int       `json:"days_to_maturity"`
	Taxes          float64   `json:"taxes_calculated"` // IOF + IR
	B3Fee          float64   `json:"b3_fee"`
	IRTax          float64   `json:"ir_tax"`
	IOFTax         float64   `json:"iof_tax"`
}

// TreasuryPerfPoint represents a historical performance metric point.
type TreasuryPerfPoint struct {
	Date          string  `json:"date"`
	Value         float64 `json:"value"`
	TotalInvested float64 `json:"total_invested"`
}

// TreasuryTransaction represents a row in the treasury_transactions table.
type TreasuryTransaction struct {
	ID                string     `json:"id"`
	PortfolioID       string     `json:"portfolio_id"`
	AssetID           string     `json:"asset_id"`
	Type              string     `json:"type"` // SUBSCRIPTION, REDEMPTION
	Quantity          float64    `json:"quantity"`
	UnitPrice         float64    `json:"unit_price"`
	ContractedRate    float64    `json:"contracted_rate"`
	RemainingQuantity float64    `json:"remaining_quantity"`
	TransactionDate   time.Time  `json:"transaction_date"`
	GrossAmount       *float64   `json:"gross_amount,omitempty"`
	IOFTax            *float64   `json:"iof_tax,omitempty"`
	IRTax             *float64   `json:"ir_tax,omitempty"`
	B3Fee             *float64   `json:"b3_fee,omitempty"`
	NetAmount         *float64   `json:"net_amount,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

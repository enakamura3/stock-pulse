package market

import (
	"context"
	"time"
)

// DividendEvent representa um evento de provento normalizado.
// Todas as DividendSources produzem este modelo canônico.
type DividendEvent struct {
	Date        time.Time `json:"date"`         // Data Com (cum_date)
	PaymentDate time.Time `json:"payment_date"` // Data de Pagamento
	Amount      float64   `json:"amount"`       // Valor bruto por cota/ação
	Type        string    `json:"type"`         // Dividendo | JCP | Rendimento | Amortização
}

// DividendSource é a interface que toda fonte de proventos deve implementar.
// Cada implementação recebe dados crus de seu Client correspondente
// e os traduz para o modelo canônico DividendEvent.
type DividendSource interface {
	// GetDividends busca proventos de um ticker e os retorna no formato canônico.
	GetDividends(ctx context.Context, ticker string, assetType string) ([]DividendEvent, error)

	// Name retorna o identificador da fonte (ex: "fundamentus", "b3", "yahoo").
	Name() string

	// SupportedAssetTypes retorna os tipos de ativo que esta fonte suporta.
	// Valores possíveis: STOCK_BR, FII, FIAGRO, ETF_BR, BDR, STOCK_US, ETF_US, CRYPTO
	SupportedAssetTypes() []string
}

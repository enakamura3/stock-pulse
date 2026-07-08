package fixedincome

import (
	"context"
	"time"
)

// BCBProvider é um adaptador formal que implementa a interface IndexProvider
// delegando para o BCBClient (API SGS do Banco Central do Brasil).
// Cobre os índices: CDI, SELIC, IPCA.
type BCBProvider struct {
	client BCBClient
}

// NewBCBProvider inicializa o provedor do Banco Central.
func NewBCBProvider(client BCBClient) *BCBProvider {
	return &BCBProvider{client: client}
}

// FetchRates implementa IndexProvider delegando para o BCBClient.
func (p *BCBProvider) FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error) {
	return p.client.FetchRates(ctx, indexer, startDate, endDate)
}

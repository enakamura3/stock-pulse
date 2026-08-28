package fixedincome

import (
	"context"
	"fmt"
	"time"
)

// IndexProvider define o contrato para buscar dados históricos de um índice.
type IndexProvider interface {
	FetchRates(ctx context.Context, indexer string, startDate, endDate time.Time) ([]IndexRate, error)
}

// IndexerConfig define o pipeline de busca de um indicador específico.
type IndexerConfig struct {
	Name             string
	PrimaryProvider  IndexProvider
	FallbackProvider IndexProvider // Opcional (nil se não houver fallback)
}

// IndexRegistry gerencia os pipelines de busca de indicadores cadastrados de forma dinâmica.
type IndexRegistry struct {
	indexers map[string]IndexerConfig
}

// NewIndexRegistry inicializa o mapa de indexadores.
func NewIndexRegistry() *IndexRegistry {
	return &IndexRegistry{
		indexers: make(map[string]IndexerConfig),
	}
}

// Register adiciona ou updates a configuração de um índice no registro.
func (r *IndexRegistry) Register(config IndexerConfig) {
	r.indexers[config.Name] = config
}

// Fetch executa a busca do índice usando o pipeline de provedores configurado.
func (r *IndexRegistry) Fetch(ctx context.Context, indexerName string, startDate, endDate time.Time) ([]IndexRate, error) {
	config, ok := r.indexers[indexerName]
	if !ok {
		return nil, fmt.Errorf("indexador não configurado: %s", indexerName)
	}

	// 1. Tenta buscar no provedor primário
	rates, err := config.PrimaryProvider.FetchRates(ctx, indexerName, startDate, endDate)
	if err == nil && len(rates) > 0 {
		return rates, nil
	}

	// 2. Se falhar ou vier vazio e houver provedor de fallback, executa o fallback
	if config.FallbackProvider != nil {
		fallbackRates, fallbackErr := config.FallbackProvider.FetchRates(ctx, indexerName, startDate, endDate)
		if fallbackErr == nil && len(fallbackRates) > 0 {
			return fallbackRates, nil
		}
		if fallbackErr != nil {
			return nil, fmt.Errorf("provedor primário falhou (%w) e fallback também falhou: %v", err, fallbackErr)
		}
	}

	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("nenhum dado retornado para o índice %s", indexerName)
}

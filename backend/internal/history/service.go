package history

import (
	"context"
	"sort"
)

type Service interface {
	GetPortfolioHistory(ctx context.Context, portfolioID, userID string) ([]UnifiedTransaction, error)
}

type service struct {
	sources []TransactionSource
}

func NewService(sources ...TransactionSource) Service {
	return &service{
		sources: sources,
	}
}

func (s *service) GetPortfolioHistory(ctx context.Context, portfolioID, userID string) ([]UnifiedTransaction, error) {
	var allTransactions []UnifiedTransaction

	// Aggregate transactions from all injected sources
	for _, source := range s.sources {
		txs, err := source.GetUnifiedTransactions(ctx, portfolioID, userID)
		if err != nil {
			return nil, err
		}
		allTransactions = append(allTransactions, txs...)
	}

	// Sort by date descending (newest first)
	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].Date.After(allTransactions[j].Date)
	})

	return allTransactions, nil
}

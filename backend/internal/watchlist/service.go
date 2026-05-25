package watchlist

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/onigiri/stockpulse/backend/internal/market"
)

// Service implementa as regras de negócio de favoritos e orquestração de cotações.
type Service struct {
	repo           *Repository
	marketService  *market.Service
	marketProvider market.QuoteProvider
}

// NewService cria uma nova instância de Service.
func NewService(repo *Repository, marketService *market.Service, marketProvider market.QuoteProvider) *Service {
	return &Service{
		repo:           repo,
		marketService:  marketService,
		marketProvider: marketProvider,
	}
}

// CreateWatchlist cria uma nova lista de favoritos para o usuário.
func (s *Service) CreateWatchlist(ctx context.Context, userID, name string) (*Watchlist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("o nome da watchlist não pode ser vazio")
	}
	return s.repo.CreateWatchlist(ctx, userID, name)
}

// GetWatchlists lista as listas do usuário. Cria a padrão "Favoritos" se nenhuma existir.
func (s *Service) GetWatchlists(ctx context.Context, userID string) ([]Watchlist, error) {
	lists, err := s.repo.GetWatchlistsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// UX Onboarding: Cria lista "Favoritos" se o usuário acabou de criar a conta
	if len(lists) == 0 {
		log.Printf("[Watchlist] Usuário %s não possui listas. Criando padrão 'Favoritos'...", userID)
		w, err := s.repo.CreateWatchlist(ctx, userID, "Favoritos")
		if err != nil {
			return nil, fmt.Errorf("falha ao criar watchlist de onboarding: %w", err)
		}
		lists = append(lists, *w)
	}

	return lists, nil
}

// GetWatchlist resgata os detalhes da lista de favoritos com todas as cotações em tempo real agregadas.
func (s *Service) GetWatchlist(ctx context.Context, id, userID string) (*Watchlist, error) {
	// Anti-IDOR: Valida se a lista pertence ao usuário solicitante
	w, err := s.repo.GetWatchlistByID(ctx, id, userID)
	if err != nil {
		return nil, errors.New("lista de favoritos não encontrada ou permissão negada")
	}

	items, err := s.repo.GetWatchlistItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar itens da lista: %w", err)
	}

	// Agrega cotações em tempo real para cada item de forma sequencial ou paralela.
	// Sequencial é suficiente e seguro para o MVP dado o cache do Redis.
	for i := range items {
		quote, err := s.marketService.GetQuote(ctx, items[i].Ticker)
		if err != nil {
			log.Printf("[Watchlist] Falha ao injetar cotação para %s: %v", items[i].Ticker, err)
			continue
		}
		items[i].Price = quote.Price
		items[i].Change = quote.Change
		items[i].ChangePercent = quote.ChangePercent
	}

	w.Items = items
	return w, nil
}

// DeleteWatchlist remove a lista do banco de dados (Criação de Cascading apaga os itens automaticamente).
func (s *Service) DeleteWatchlist(ctx context.Context, id, userID string) error {
	return s.repo.DeleteWatchlist(ctx, id, userID)
}

// AddAssetToWatchlist adiciona o ativo à lista, importando metadados dinamicamente do Yahoo se inédito.
func (s *Service) AddAssetToWatchlist(ctx context.Context, watchlistID, userID, ticker string) (*Item, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return nil, errors.New("ticker inválido")
	}

	// Anti-IDOR: Valida se a watchlist pertence de fato ao usuário logado
	_, err := s.repo.GetWatchlistByID(ctx, watchlistID, userID)
	if err != nil {
		return nil, errors.New("lista de favoritos não encontrada ou acesso não autorizado")
	}

	// Passo 1: Verifica se o ativo já existe no banco de dados local
	assetID, err := s.repo.GetAssetByTicker(ctx, ticker)
	if err != nil {
		// Passo 2: Se não existir, busca metadados ricos do Yahoo Finance (Auto-Import/Onboarding)
		log.Printf("[Watchlist] Ativo %s não existe localmente. Buscando metadados no Yahoo...", ticker)
		quote, err := s.marketProvider.GetQuote(ctx, ticker)
		if err != nil {
			return nil, fmt.Errorf("ativo '%s' não foi encontrado ou não é suportado pelo provedor de mercado: %w", ticker, err)
		}

		// Passo 3: Cria o ativo localmente salvando Ticker, Nome, Tipo e Moeda
		assetType := "EQUITY" // Padrão genérico se desconhecido
		if quote.Currency == "USD" && !strings.Contains(ticker, ".") {
			assetType = "EQUITY_US"
		} else if strings.Contains(ticker, "-") {
			assetType = "CRYPTO"
		}

		assetID, err = s.repo.CreateAsset(ctx, ticker, quote.Name, assetType, quote.Currency)
		if err != nil {
			return nil, fmt.Errorf("erro ao registrar novo ativo no banco: %w", err)
		}
		log.Printf("[Watchlist] Ativo %s cadastrado com sucesso sob ID %s", ticker, assetID)
	}

	// Passo 4: Vincula o ativo na Watchlist do usuário
	return s.repo.AddWatchlistItem(ctx, watchlistID, assetID)
}

// RemoveAssetFromWatchlist desvincula o ativo da lista de favoritos do usuário.
func (s *Service) RemoveAssetFromWatchlist(ctx context.Context, watchlistID, userID, ticker string) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	// Anti-IDOR: Valida se a lista pertence de fato ao usuário solicitante
	_, err := s.repo.GetWatchlistByID(ctx, watchlistID, userID)
	if err != nil {
		return errors.New("lista de favoritos não encontrada ou permissão negada")
	}

	return s.repo.RemoveWatchlistItem(ctx, watchlistID, ticker)
}

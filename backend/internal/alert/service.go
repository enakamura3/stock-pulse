package alert

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/onigiri/stock-pulse/backend/internal/market"
)

// AlertRepository define as operações de banco de dados necessárias para alertas.
type AlertRepository interface {
	GetAssetByTicker(ctx context.Context, ticker string) (string, error)
	CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error)
	CreateAlert(ctx context.Context, a *Alert) error
	GetAlertsByUserID(ctx context.Context, userID string) ([]*Alert, error)
	DeleteAlert(ctx context.Context, id string, userID string) error
	GetActiveAlerts(ctx context.Context) ([]*Alert, error)
	MarkAlertTriggered(ctx context.Context, id string) error
	ToggleAlertStatus(ctx context.Context, id string, userID string) (string, error)
}

// Service implementa as regras de negócio para a gestão de alertas de preço.
type Service struct {
	repo           AlertRepository
	marketProvider market.QuoteProvider
}

// NewService inicializa o Alert Service.
func NewService(repo AlertRepository, marketProvider market.QuoteProvider) *Service {
	return &Service{
		repo:           repo,
		marketProvider: marketProvider,
	}
}

// CreateAlert valida e cria um novo alerta de preço, importando o ativo caso seja inédito localmente.
func (s *Service) CreateAlert(ctx context.Context, userID string, ticker string, targetPrice float64, condition string) (*Alert, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	condition = strings.ToUpper(strings.TrimSpace(condition))

	if ticker == "" {
		return nil, errors.New("o ticker do ativo não pode ser vazio")
	}
	if targetPrice <= 0 {
		return nil, errors.New("o preço alvo deve ser maior do que zero")
	}
	if condition != "ABOVE" && condition != "BELOW" {
		return nil, errors.New("a condição de alerta deve ser 'ABOVE' (acima de) ou 'BELOW' (abaixo de)")
	}

	// 1. Tenta recuperar o ID do ativo no banco de dados local
	assetID, err := s.repo.GetAssetByTicker(ctx, ticker)
	if err != nil {
		// 2. Se o ativo for inédito, consulta no provedor Yahoo Finance para validar sua existência
		log.Printf("[Alerts] Ativo %s não encontrado localmente. Validando no Yahoo Finance...", ticker)
		quote, err := s.marketProvider.GetQuote(ctx, ticker)
		if err != nil {
			return nil, fmt.Errorf("o ativo '%s' não existe ou não é suportado pelo provedor de mercado: %w", ticker, err)
		}

		// 3. Cadastra o ativo de forma inteligente no banco local
		assetType := "EQUITY"
		if strings.Contains(ticker, "-") {
			assetType = "CRYPTO"
		} else if quote.Currency == "USD" && !strings.Contains(ticker, ".") {
			assetType = "EQUITY_US"
		}

		assetID, err = s.repo.CreateAsset(ctx, ticker, quote.Name, assetType, quote.Currency)
		if err != nil {
			return nil, fmt.Errorf("erro ao registrar o novo ativo no banco de dados: %w", err)
		}
		log.Printf("[Alerts] Novo ativo %s cadastrado com ID %s", ticker, assetID)
	}

	// 4. Cria e salva o alerta como ACTIVE
	alert := &Alert{
		UserID:      userID,
		AssetID:     assetID,
		TargetPrice: targetPrice,
		Condition:   condition,
		Status:      "ACTIVE",
	}

	err = s.repo.CreateAlert(ctx, alert)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar alerta no banco: %w", err)
	}

	// Preenche campos auxiliares para retornar à API
	alert.Ticker = ticker
	return alert, nil
}

// GetAlerts resgata a lista de todos os alertas do usuário (Anti-IDOR).
func (s *Service) GetAlerts(ctx context.Context, userID string) ([]*Alert, error) {
	if userID == "" {
		return nil, errors.New("ID de usuário inválido")
	}
	return s.repo.GetAlertsByUserID(ctx, userID)
}

// DeleteAlert remove o alerta de preço de forma segura (Anti-IDOR).
func (s *Service) DeleteAlert(ctx context.Context, id string, userID string) error {
	if id == "" || userID == "" {
		return errors.New("parâmetros inválidos para deleção")
	}
	return s.repo.DeleteAlert(ctx, id, userID)
}

// ToggleAlert alterna o status de ativação do alerta ('ACTIVE' <-> 'DISABLED') com segurança (Anti-IDOR).
func (s *Service) ToggleAlert(ctx context.Context, id string, userID string) (string, error) {
	if id == "" || userID == "" {
		return "", errors.New("parâmetros inválidos para toggle")
	}
	return s.repo.ToggleAlertStatus(ctx, id, userID)
}

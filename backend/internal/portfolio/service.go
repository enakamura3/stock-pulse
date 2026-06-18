package portfolio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/history"
	"github.com/onigiri/stock-pulse/backend/internal/market"
)

// determineAssetType define a categoria oficial do ativo no banco
func determineAssetType(ticker, name, currency string) string {
	if strings.Contains(ticker, "-") {
		return "CRYPTO"
	}

	if !strings.HasSuffix(ticker, ".SA") {
		// Internacional
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "etf") || strings.Contains(lowerName, "trust") || strings.Contains(lowerName, "fund") {
			return "ETF_US"
		}
		return "STOCK_US"
	}

	// É do Brasil (.SA)
	if strings.HasSuffix(ticker, "34.SA") || strings.HasSuffix(ticker, "35.SA") || strings.HasSuffix(ticker, "39.SA") {
		return "BDR"
	}

	if strings.HasSuffix(ticker, "11.SA") {
		lowerName := strings.ToLower(name)
		isEtf := strings.Contains(lowerName, "etf") || strings.Contains(lowerName, "ishares") || strings.Contains(lowerName, "índice") || strings.Contains(lowerName, "indice")
		isFiagro := strings.Contains(lowerName, "fiagro") || strings.Contains(lowerName, "agro")
		isFii := strings.Contains(lowerName, "fii") || strings.Contains(lowerName, "fundo") || strings.Contains(lowerName, "fdo") || strings.Contains(lowerName, "imob") || strings.Contains(lowerName, "lajes") || strings.Contains(lowerName, "shopping")

		if isEtf {
			return "ETF_BR"
		}
		if isFiagro {
			return "FIAGRO"
		}
		// Hardcoded common FIIs if name misses
		tickerUpper := strings.ToUpper(ticker)
		if isFii || tickerUpper == "MXRF11.SA" || tickerUpper == "HGLG11.SA" || tickerUpper == "KNRI11.SA" || tickerUpper == "BTLG11.SA" || tickerUpper == "XPML11.SA" || tickerUpper == "VISC11.SA" {
			return "FII"
		}
		if tickerUpper == "SPYI11.SA" || tickerUpper == "QQQI11.SA" || tickerUpper == "IVVB11.SA" || tickerUpper == "NASD11.SA" || tickerUpper == "BOVA11.SA" {
			return "ETF_BR"
		}
	}

	return "STOCK_BR"
}

// PortfolioRepository define as operações de banco de dados para a carteira.
type PortfolioRepository interface {
	CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error)
	GetPortfoliosByUserID(ctx context.Context, userID string) ([]Portfolio, error)
	GetPortfolioByID(ctx context.Context, id, userID string) (*Portfolio, error)
	DeletePortfolio(ctx context.Context, id, userID string) error
	CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error)
	UpdateTransaction(ctx context.Context, tx Transaction) error
	GetTransactionsByPortfolioID(ctx context.Context, portfolioID, userID string) ([]Transaction, error)
	DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error
	SaveDailyPrices(ctx context.Context, assetID string, prices []DailyPrice) error
	GetDailyPrices(ctx context.Context, assetID string, startDate, endDate time.Time) ([]DailyPrice, error)
	GetAssetByTicker(ctx context.Context, ticker string) (string, error)
	GetAssetAndCurrencyByTicker(ctx context.Context, ticker string) (string, string, error)
	CreateAsset(ctx context.Context, ticker, name, assetType, currency string) (string, error)
	GetAllAssets(ctx context.Context) ([]AssetCompact, error)
	UpsertAssetEvent(ctx context.Context, event AssetEvent) error
	GetAssetEvents(ctx context.Context, assetID string) ([]AssetEvent, error)
	GetAssetEventsByDate(ctx context.Context, assetID string, exDate time.Time) ([]AssetEvent, error)
	UpdateAssetEventValueByID(ctx context.Context, eventID string, newGross, newNet float64, newPayment time.Time) error
	GetExchangeRateByDate(ctx context.Context, currencyPairTicker string, date time.Time) (float64, error)
	GetOldestPriceDate(ctx context.Context, assetID string) (time.Time, error)
}

// MarketService define as operações de mercado suportadas.
type MarketService interface {
	GetQuote(ctx context.Context, ticker string) (*market.Quote, error)
	GetFundamentals(ctx context.Context, ticker string) (*market.Fundamentals, error)
	GetDividends(ctx context.Context, ticker string, assetType string) ([]market.DividendEvent, error)
	GetHistoricalExchangeRate(ctx context.Context, date time.Time) (float64, error)
}

// Service gerencia as regras de negócio de carteiras, transações e histórico.
type Service struct {
	repo           PortfolioRepository
	marketService  MarketService
	marketProvider market.QuoteProvider
	fiService      fixedincome.Service
	httpClient     *http.Client
}

// NewService cria uma nova instância de Service.
func NewService(repo PortfolioRepository, marketService MarketService, marketProvider market.QuoteProvider, fiService fixedincome.Service) *Service {
	return &Service{
		repo:           repo,
		marketService:  marketService,
		marketProvider: marketProvider,
		fiService:      fiService,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *Service) GetFixedIncomeService() fixedincome.Service {
	return s.fiService
}

// CreatePortfolio cria uma nova carteira de investimentos para o usuário.
func (s *Service) CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("o nome da carteira não pode ser vazio")
	}
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if baseCurrency == "" {
		baseCurrency = "BRL"
	}
	return s.repo.CreatePortfolio(ctx, userID, name, baseCurrency)
}

// GetPortfolios lista os portfólios do usuário (cria "Principal" padrão se vazio).
func (s *Service) GetPortfolios(ctx context.Context, userID string) ([]Portfolio, error) {
	lists, err := s.repo.GetPortfoliosByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// UX Onboarding: Cria portfólio "Principal" se o usuário acabou de criar a conta
	if len(lists) == 0 {
		log.Printf("[Portfolio] Usuário %s não possui carteiras. Criando padrão 'Principal' BRL...", userID)
		p, err := s.repo.CreatePortfolio(ctx, userID, "Principal", "BRL")
		if err != nil {
			return nil, fmt.Errorf("falha ao criar portfólio de onboarding: %w", err)
		}
		lists = append(lists, *p)
	}

	return lists, nil
}

// CalculatedDividend representa o dividendo calculado para o usuário.
type CalculatedDividend struct {
	AssetID        string    `json:"asset_id"`
	Ticker         string    `json:"ticker"`
	ExDate         time.Time `json:"ex_date"`
	PaymentDate    time.Time `json:"payment_date"`
	GrossAmount    float64   `json:"gross_amount"`
	NetAmount      float64   `json:"net_amount"`
	Currency       string    `json:"currency"`
	OriginalGross  float64   `json:"original_gross_amount,omitempty"`
	OriginalNet    float64   `json:"original_net_amount,omitempty"`
	Type           string    `json:"type"`
	Quantity       float64   `json:"quantity"`
	PerShareAmount float64   `json:"per_share_amount"`
	AssetType      string    `json:"asset_type"`
	AssetName      string    `json:"asset_name"`
}

// GetPortfolioDividends calcula todos os dividendos (históricos e futuros) com base na posição da carteira na data ex-dividendo.
func (s *Service) GetPortfolioDividends(ctx context.Context, portfolioID, userID string) ([]CalculatedDividend, error) {
	// Verifica a existência do portfólio para garantir autorização
	_, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return nil, err
	}

	// Busca todas as transações para calcular as posições no tempo
	transactions, err := s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
	if err != nil {
		return nil, err
	}

	// Agrupa transações por ticker para facilitar processamento cronológico
	txByTicker := make(map[string][]Transaction)
	for _, tx := range transactions {
		txByTicker[tx.Ticker] = append(txByTicker[tx.Ticker], tx)
	}

	var results []CalculatedDividend

	// Para cada ativo, buscamos os proventos e iteramos para calcular
	for ticker, txs := range txByTicker {
		// Ordena transações cronologicamente
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].ExecutedAt.Before(txs[j].ExecutedAt)
		})

		// A moeda base do ativo (BRL ou USD)
		currency := "BRL"
		if len(txs) > 0 && txs[0].Currency != "" {
			currency = txs[0].Currency
		}

		divs, err := s.repo.GetAssetEvents(ctx, txs[0].AssetID)
		if err != nil {
			log.Printf("Aviso: falha ao buscar dividendos locais para %s: %v", ticker, err)
			continue
		}

		for _, div := range divs {
			exDate := div.ExDate

			// Calcula a quantidade na carteira no fechamento do dia anterior à ex-date,
			// Ou seja, compras efetuadas ATÉ a ex-date (pois na ex-date a ação já é negociada sem o dividendo,
			// mas o comprador do dia D-1 recebe o dividendo).
			// Simplificando: compras onde ExecutedAt < exDate (começo do dia).
			var quantity float64 = 0
			for _, tx := range txs {
				// Se a transação ocorreu depois do início do exDate, interrompe a soma (lista tá ordenada)
				if tx.ExecutedAt.After(exDate) || tx.ExecutedAt.Equal(exDate) {
					break
				}

				if tx.Type == "BUY" {
					quantity += tx.Quantity
				} else if tx.Type == "SELL" {
					quantity -= tx.Quantity
				} else if tx.Type == "SPLIT" {
					quantity = quantity * tx.Quantity
				} else if tx.Type == "REVERSE_SPLIT" {
					if tx.Quantity > 0 {
						quantity = math.Floor(quantity / tx.Quantity)
					}
				} else if tx.Type == "BONUS" {
					quantity += tx.Quantity
				}
			}

			// Se a quantidade resultante é > 0, o usuário tem direito ao dividendo!
			if quantity > 0 {
				grossAmount := quantity * div.GrossAmount
				netAmount := grossAmount
				divCurrency := currency // cópia local para não afetar as próximas iterações

				// Regras de Impostos
				if divCurrency == "USD" {
					// EUA: 30% retido na fonte
					netAmount = grossAmount * 0.70
				} else if divCurrency == "BRL" {
					if div.Type == "JCP" {
						// JCP: 15% de imposto retido na fonte
						netAmount = grossAmount * 0.85
					} else if strings.HasPrefix(txs[0].AssetType, "ETF") {
						// Exceção: ETFs na B3 que sofrem tributação de 15% nos dividendos retidos na fonte
						netAmount = grossAmount * 0.85
					} else {
						// Dividendos, Rendimentos (FII), Amortização: 0% de imposto
						netAmount = grossAmount
					}
				}

				exchangeRate := 1.0
				var originalGross float64 = 0
				var originalNet float64 = 0

				// Conversão Cambial (Apenas para USD -> BRL)
				if divCurrency == "USD" {
					originalGross = grossAmount
					originalNet = netAmount

					fx, err := s.marketService.GetHistoricalExchangeRate(ctx, exDate)
					if err == nil {
						exchangeRate = fx
					} else {
						exchangeRate = 5.0 // fallback
					}

					// Converte Gross e Net para a moeda base do Portfolio (assumindo BRL)
					grossAmount = grossAmount * exchangeRate
					netAmount = netAmount * exchangeRate
					divCurrency = "BRL"
				}

				results = append(results, CalculatedDividend{
					AssetID:        txs[0].AssetID,
					Ticker:         ticker,
					ExDate:         exDate,
					PaymentDate:    div.PaymentDate,
					GrossAmount:    grossAmount,
					NetAmount:      netAmount,
					Currency:       divCurrency,
					OriginalGross:  originalGross,
					OriginalNet:    originalNet,
					Type:           div.Type,
					Quantity:       quantity,
					PerShareAmount: div.GrossAmount * exchangeRate,
					AssetType:      txs[0].AssetType,
					AssetName:      txs[0].AssetName,
				})
			}
		}
	}

	// Ordena do mais recente para o mais antigo
	sort.Slice(results, func(i, j int) bool {
		return results[i].PaymentDate.After(results[j].PaymentDate)
	})

	return results, nil
}

// GetPortfolioDetails calcula o consolidado da carteira (posições ativas, custo e lucro médio).
func (s *Service) GetPortfolioDetails(ctx context.Context, portfolioID, userID string) (*Portfolio, []Position, error) {
	// Anti-IDOR: Valida se a carteira pertence ao usuário logado
	p, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return nil, nil, errors.New("carteira não encontrada ou acesso não autorizado")
	}

	// Recupera todas as transações
	txs, err := s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao carregar transações da carteira: %w", err)
	}

	// Ordena transações cronologicamente (mais antiga para mais recente) para calcular preço médio
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].ExecutedAt.Equal(txs[j].ExecutedAt) {
			return txs[i].CreatedAt.Before(txs[j].CreatedAt)
		}
		return txs[i].ExecutedAt.Before(txs[j].ExecutedAt)
	})

	// Agrupa e calcula as posições
	posMap := make(map[string]*Position)
	for _, tx := range txs {
		pos, ok := posMap[tx.AssetID]
		if !ok {
			pos = &Position{
				AssetID:  tx.AssetID,
				Ticker:   tx.Ticker,
				Name:     tx.AssetName,
				Type:     tx.AssetType,
				Currency: tx.Currency,
			}
			posMap[tx.AssetID] = pos
		}

		if tx.Type == "BUY" {
			if pos.Quantity == 0 {
				pos.Quantity = tx.Quantity
				pos.TotalCost = tx.Quantity * tx.UnitPrice * tx.ExchangeRate
				pos.AveragePrice = tx.UnitPrice
			} else {
				pos.Quantity += tx.Quantity
				pos.TotalCost += tx.Quantity * tx.UnitPrice * tx.ExchangeRate
				pos.AveragePrice = pos.TotalCost / (pos.Quantity * tx.ExchangeRate) // Preço na moeda original
			}
		} else if tx.Type == "SELL" {
			if pos.Quantity >= tx.Quantity {
				pos.Quantity -= tx.Quantity
				pos.TotalCost = pos.Quantity * pos.AveragePrice * tx.ExchangeRate
			} else {
				// Venda acima do saldo zerará a posição
				pos.Quantity = 0
				pos.TotalCost = 0
				pos.AveragePrice = 0
			}
		} else if tx.Type == "SPLIT" {
			if pos.Quantity > 0 && tx.Quantity > 0 {
				pos.Quantity = pos.Quantity * tx.Quantity
				pos.AveragePrice = pos.AveragePrice / tx.Quantity
			}
		} else if tx.Type == "REVERSE_SPLIT" {
			if pos.Quantity > 0 && tx.Quantity > 0 {
				pos.Quantity = math.Floor(pos.Quantity / tx.Quantity)
				pos.AveragePrice = pos.AveragePrice * tx.Quantity
			}
		} else if tx.Type == "BONUS" {
			pos.Quantity += tx.Quantity
			pos.TotalCost += tx.Quantity * tx.UnitPrice * tx.ExchangeRate
			if pos.Quantity > 0 {
				pos.AveragePrice = pos.TotalCost / (pos.Quantity * tx.ExchangeRate)
			}
		}
	}

	// Filtra apenas posições ativas (quantidade > 0)
	var activePositions []Position
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, pos := range posMap {
		if pos.Quantity > 0 {
			wg.Add(1)
			go func(pos *Position) {
				defer wg.Done()

				// Injeta cotações em tempo real e calcula rentabilidade
				quote, err := s.marketService.GetQuote(ctx, pos.Ticker)
				if err != nil {
					log.Printf("[Portfolio] Erro ao recuperar cotação atual para %s: %v", pos.Ticker, err)
					mu.Lock()
					activePositions = append(activePositions, *pos)
					mu.Unlock()
					return
				}

				// Conversão cambial em tempo real (se ativo for USD e carteira for BRL)
				rate := 1.0
				if pos.Currency != p.BaseCurrency {
					rate = s.getCurrencyRate(ctx, pos.Currency, p.BaseCurrency)
				}

				pos.CurrentPrice = quote.Price
				pos.CurrentValue = pos.Quantity * quote.Price * rate
				pos.DailyChange = quote.Change
				pos.DailyChangePercent = quote.ChangePercent
				pos.ProfitLoss = pos.CurrentValue - pos.TotalCost
				if pos.TotalCost > 0 {
					pos.ReturnPercent = (pos.ProfitLoss / pos.TotalCost) * 100
				}

				// Injeta fundamentos (Graham, Bazin, P/VP, P/L)
				if f, errF := s.marketService.GetFundamentals(ctx, pos.Ticker); errF == nil && f != nil {
					pos.GrahamValue = f.GrahamValue
					pos.BazinValue = f.BazinValue
					pos.DividendYield = f.DividendYield

					if f.BookValue > 0 {
						pos.PVP = pos.CurrentPrice / f.BookValue
					}
					if f.EPS > 0 {
						pos.PE = pos.CurrentPrice / f.EPS
					}
				}

				mu.Lock()
				activePositions = append(activePositions, *pos)
				mu.Unlock()
			}(pos)
		}
	}
	wg.Wait()

	// Re-ordena as posições por Ticker alfabético para exibição elegante
	sort.Slice(activePositions, func(i, j int) bool {
		return activePositions[i].Ticker < activePositions[j].Ticker
	})

	return p, activePositions, nil
}

// AddTransaction registra uma nova transação, importando o ativo e disparando backfill se necessário.
func (s *Service) AddTransaction(ctx context.Context, userID string, tx *Transaction) (*Transaction, error) {
	// Anti-IDOR: Valida se a carteira pertence ao usuário logado
	p, err := s.repo.GetPortfolioByID(ctx, tx.PortfolioID, userID)
	if err != nil {
		return nil, errors.New("carteira não encontrada ou acesso não autorizado")
	}

	tx.Ticker = strings.ToUpper(strings.TrimSpace(tx.Ticker))
	if tx.Ticker == "" {
		return nil, errors.New("ticker do ativo inválido")
	}

	// Busca ou cria o ativo na base local
	assetID, currency, err := s.repo.GetAssetAndCurrencyByTicker(ctx, tx.Ticker)
	if err != nil {
		// Importa metadados do Yahoo Finance
		log.Printf("[Portfolio] Ativo %s não existe na base. Importando...", tx.Ticker)
		quote, err := s.marketProvider.GetQuote(ctx, tx.Ticker)
		if err != nil {
			return nil, fmt.Errorf("ativo '%s' não encontrado no mercado: %w", tx.Ticker, err)
		}

		assetType := determineAssetType(tx.Ticker, quote.Name, quote.Currency)

		assetID, err = s.repo.CreateAsset(ctx, tx.Ticker, quote.Name, assetType, quote.Currency)
		if err != nil {
			return nil, fmt.Errorf("erro ao registrar ativo localmente: %w", err)
		}
		currency = quote.Currency
	}

	tx.AssetID = assetID

	// Correção Cambial: Se a taxa não foi fornecida, busca automaticamente
	if tx.ExchangeRate <= 0 {
		if currency != p.BaseCurrency {
			currencyPair := fmt.Sprintf("%s%s=X", currency, p.BaseCurrency)
			log.Printf("[Portfolio] Buscando câmbio histórico para %s na data %s no banco de dados...", currencyPair, tx.ExecutedAt)
			
			rate, err := s.repo.GetExchangeRateByDate(ctx, currencyPair, tx.ExecutedAt)
			if err != nil || rate <= 0 {
				log.Printf("[Portfolio] Taxa não encontrada na base. Disparando Micro-Backfill para tapar o buraco...")
				s.BackfillGap(ctx, currencyPair, tx.ExecutedAt)
				
				// Tenta buscar novamente
				rate, err = s.repo.GetExchangeRateByDate(ctx, currencyPair, tx.ExecutedAt)
			}
			
			if err == nil && rate > 0 {
				tx.ExchangeRate = rate
				log.Printf("[Portfolio] Câmbio encontrado na base: %.4f", rate)
			} else {
				log.Printf("[Portfolio] Aviso: Falha ao buscar câmbio histórico após backfill (%v). Usando fallback de 1.0", err)
				tx.ExchangeRate = 1.0
			}
		} else {
			tx.ExchangeRate = 1.0
		}
	}

	tx.TotalCost = tx.Quantity * tx.UnitPrice
	savedTx, err := s.repo.CreateTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Dispara o Auto-Backfill de 5 anos de forma assíncrona (Goroutine controlada)
	go func(id, ticker, curr string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		// Verifica se o histórico já existe no banco
		existing, err := s.repo.GetDailyPrices(bgCtx, id, time.Now().AddDate(0, 0, -7), time.Now())
		if err == nil && len(existing) > 0 {
			log.Printf("[Backfill] Ativo %s já possui histórico recente.", ticker)
			
			// Se possui histórico recente, vamos verificar se a transação é mais antiga que o nosso buraco
			oldestDate, err := s.repo.GetOldestPriceDate(bgCtx, id)
			if err == nil && !oldestDate.IsZero() && tx.ExecutedAt.Before(oldestDate) {
				log.Printf("[Backfill] Transação antiga detectada. Disparando BackfillGap para o ativo %s tapar o buraco até %s", ticker, oldestDate.Format("2006-01-02"))
				if err := s.BackfillGap(bgCtx, ticker, tx.ExecutedAt); err != nil {
					log.Printf("[Backfill] Falha ao rodar BackfillGap de %s: %v", ticker, err)
				}
			}
		} else {
			log.Printf("[Backfill] Iniciando preenchimento histórico máximo (max) para %s...", ticker)
			if err := s.BackfillHistoricalPrices(bgCtx, id, ticker); err != nil {
				log.Printf("[Backfill] Falha ao rodar backfill histórico de %s: %v", ticker, err)
			}
		}

		// Se o ativo for em USD e a carteira estiver em BRL, preenche também o histórico cambial de USDBRL=X!
		if curr == "USD" && p.BaseCurrency == "BRL" {
			usdBrlID, err := s.repo.GetAssetByTicker(bgCtx, "USDBRL=X")
			if err != nil {
				usdBrlID, err = s.repo.CreateAsset(bgCtx, "USDBRL=X", "USD/BRL Currency Pair", "CURRENCY", "BRL")
			}
			if err == nil {
				log.Printf("[Backfill] Iniciando preenchimento cambial de USDBRL=X para carteira BRL...")
				if err := s.BackfillHistoricalPrices(bgCtx, usdBrlID, "USDBRL=X"); err != nil {
					log.Printf("[Backfill] Falha ao rodar backfill histórico de USDBRL=X: %v", err)
				}
			}
		}
	}(assetID, tx.Ticker, currency)

	return savedTx, nil
}

// DeleteTransaction apaga uma transação da carteira.
func (s *Service) DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error {
	return s.repo.DeleteTransaction(ctx, txID, portfolioID, userID)
}

func (s *Service) GetPortfolioTransactions(ctx context.Context, portfolioID, userID string) ([]Transaction, error) {
	return s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
}

// DeletePortfolio remove a carteira do banco de dados.
func (s *Service) DeletePortfolio(ctx context.Context, id, userID string) error {
	return s.repo.DeletePortfolio(ctx, id, userID)
}

// PerformancePoint representa o saldo consolidado de um portfólio em uma data histórica.
type PerformancePoint struct {
	Date          string  `json:"date"`
	Value         float64 `json:"value"`
	TotalInvested float64 `json:"total_invested"`
}

// GetPortfolioPerformance constrói a série histórica de rentabilidade consolidada.
func (s *Service) GetPortfolioPerformance(ctx context.Context, portfolioID string, userID string, period string, filterTickers []string) ([]PerformancePoint, error) {
	// Anti-IDOR: Valida a posse
	p, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return nil, errors.New("carteira não encontrada ou acesso não autorizado")
	}

	// Carrega todas as transações
	txs, err := s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar transações: %w", err)
	}
	if len(txs) == 0 {
		return []PerformancePoint{}, nil
	}

	// Filtra as transações caso o usuário tenha selecionado uma categoria específica
	if len(filterTickers) > 0 {
		tickerMap := make(map[string]bool)
		for _, t := range filterTickers {
			tickerMap[t] = true
		}
		filteredTxs := make([]Transaction, 0)
		for _, tx := range txs {
			if tickerMap[tx.Ticker] {
				filteredTxs = append(filteredTxs, tx)
			}
		}
		txs = filteredTxs
	}

	if len(txs) == 0 {
		return []PerformancePoint{}, nil
	}

	// Ordena cronologicamente do mais antigo para o mais novo
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].ExecutedAt.Before(txs[j].ExecutedAt)
	})

	// Determina janela temporal da consulta
	endDate := time.Now()
	var startDate time.Time
	switch strings.ToUpper(period) {
	case "1M":
		startDate = endDate.AddDate(0, -1, 0)
	case "3M":
		startDate = endDate.AddDate(0, -3, 0)
	case "6M":
		startDate = endDate.AddDate(0, -6, 0)
	case "1Y":
		startDate = endDate.AddDate(-1, 0, 0)
	default: // "ALL" ou padrão
		startDate = txs[0].ExecutedAt
	}

	// Cobre o caso de a data inicial ser posterior à primeira transação
	if startDate.After(txs[0].ExecutedAt) {
		startDate = txs[0].ExecutedAt
	}

	// Busca histórico de cotações diárias para cada ativo envolvido na carteira
	assetIDsMap := make(map[string]bool)
	hasUSDAsset := false
	for _, tx := range txs {
		assetIDsMap[tx.AssetID] = true
		if tx.Currency == "USD" {
			hasUSDAsset = true
		}
	}

	// Injeta USDBRL=X se houver ativos em USD e carteira BRL
	var usdBrlID string
	if hasUSDAsset && p.BaseCurrency == "BRL" {
		id, err := s.repo.GetAssetByTicker(ctx, "USDBRL=X")
		if err == nil {
			usdBrlID = id
			assetIDsMap[id] = true
		}
	}

	// Mapeia preços históricos no formato: pricesMap[asset_id][date_string] = close_price
	pricesMap := make(map[string]map[string]float64)
	for assetID := range assetIDsMap {
		pricesMap[assetID] = make(map[string]float64)
		hist, err := s.repo.GetDailyPrices(ctx, assetID, txs[0].ExecutedAt, endDate)
		if err == nil {
			for _, dp := range hist {
				pricesMap[assetID][dp.PriceDate.Format("2006-01-02")] = dp.ClosePrice
			}
		}
	}

	// Reconstrói a linha do tempo dia a dia aplicando LOCF
	var points []PerformancePoint
	currDate := startDate

	// LOCF helper
	getPriceLOCF := func(assetID string, d time.Time) float64 {
		chk := d
		for i := 0; i < 100; i++ {
			dateStr := chk.Format("2006-01-02")
			if val, ok := pricesMap[assetID][dateStr]; ok && val > 0 {
				return val
			}
			chk = chk.AddDate(0, 0, -1)
		}
		return 0.0
	}

	for !currDate.After(endDate) {
		dayStr := currDate.Format("2006-01-02")

		// Consolida as quantidades de ativos e custos acumulados até este dia específico
		dailyQuantities := make(map[string]float64)
		dailyCosts := make(map[string]float64)
		dailyCurrencies := make(map[string]string)
		dailyTickers := make(map[string]string)

		dailySplitAdjustments := make(map[string]float64)

		for _, tx := range txs {
			// Ignora transações ocorridas após a data analisada, mas acumula o fator de split futuro
			if tx.ExecutedAt.After(currDate) {
				if tx.Type == "SPLIT" && tx.Quantity > 0 {
					if dailySplitAdjustments[tx.AssetID] == 0 {
						dailySplitAdjustments[tx.AssetID] = 1.0
					}
					dailySplitAdjustments[tx.AssetID] *= tx.Quantity
				} else if tx.Type == "REVERSE_SPLIT" && tx.Quantity > 0 {
					if dailySplitAdjustments[tx.AssetID] == 0 {
						dailySplitAdjustments[tx.AssetID] = 1.0
					}
					dailySplitAdjustments[tx.AssetID] /= tx.Quantity
				}
				continue
			}

			dailyCurrencies[tx.AssetID] = tx.Currency
			dailyTickers[tx.AssetID] = tx.Ticker

			if tx.Type == "BUY" {
				dailyQuantities[tx.AssetID] += tx.Quantity
				dailyCosts[tx.AssetID] += tx.Quantity * tx.UnitPrice * tx.ExchangeRate
			} else if tx.Type == "SELL" {
				if dailyQuantities[tx.AssetID] >= tx.Quantity {
					dailyQuantities[tx.AssetID] -= tx.Quantity
					// Reduz o custo proporcionalmente
					dailyCosts[tx.AssetID] = dailyQuantities[tx.AssetID] * (dailyCosts[tx.AssetID] / (dailyQuantities[tx.AssetID] + tx.Quantity))
				} else {
					dailyQuantities[tx.AssetID] = 0
					dailyCosts[tx.AssetID] = 0
				}
			} else if tx.Type == "SPLIT" {
				if dailyQuantities[tx.AssetID] > 0 && tx.Quantity > 0 {
					dailyQuantities[tx.AssetID] = dailyQuantities[tx.AssetID] * tx.Quantity
				}
			} else if tx.Type == "REVERSE_SPLIT" {
				if dailyQuantities[tx.AssetID] > 0 && tx.Quantity > 0 {
					dailyQuantities[tx.AssetID] = math.Floor(dailyQuantities[tx.AssetID] / tx.Quantity)
				}
			}
		}

		// Calcula valor total de mercado e custo investido para a data analisada
		var totalMarketValue float64
		var totalInvested float64

		for assetID, qty := range dailyQuantities {
			if qty > 0 {
				price := getPriceLOCF(assetID, currDate)
				cost := dailyCosts[assetID]

				// Se o preço não for encontrado, usa o custo médio de aquisição como fallback temporário
				if price == 0 && qty > 0 {
					price = cost / qty
				}

				// Taxa cambial do dia
				rate := 1.0
				if dailyCurrencies[assetID] != p.BaseCurrency && p.BaseCurrency == "BRL" && usdBrlID != "" {
					rate = getPriceLOCF(usdBrlID, currDate)
					if rate == 0 {
						rate = 5.0 // Fallback seguro
					}
				}

				adjFactor := dailySplitAdjustments[assetID]
				if adjFactor == 0 {
					adjFactor = 1.0
				}
				adjustedQty := qty * adjFactor

				totalMarketValue += adjustedQty * price * rate
				totalInvested += cost
			}
		}

		points = append(points, PerformancePoint{
			Date:          dayStr,
			Value:         totalMarketValue,
			TotalInvested: totalInvested,
		})

		currDate = currDate.AddDate(0, 0, 1)
	}

	// Filtra a janela final de exibição com base no período solicitado pelo usuário
	var finalPoints []PerformancePoint
	limitDate := time.Now()
	switch strings.ToUpper(period) {
	case "1M":
		limitDate = time.Now().AddDate(0, -1, 0)
	case "3M":
		limitDate = time.Now().AddDate(0, -3, 0)
	case "6M":
		limitDate = time.Now().AddDate(0, -6, 0)
	case "1Y":
		limitDate = time.Now().AddDate(-1, 0, 0)
	default:
		limitDate = startDate
	}

	for _, pt := range points {
		ptDate, err := time.Parse("2006-01-02", pt.Date)
		if err == nil && !ptDate.Before(limitDate) {
			finalPoints = append(finalPoints, pt)
		}
	}

	if len(finalPoints) == 0 {
		return points, nil
	}

	return finalPoints, nil
}

// BackfillHistoricalPrices realiza a chamada histórica ao Yahoo e grava os dados usando 10 anos de histórico diário.
func (s *Service) BackfillHistoricalPrices(ctx context.Context, assetID, ticker string) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=10y", url.PathEscape(ticker))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("provedor yahoo retornou status %d", resp.StatusCode)
	}

	var data struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error interface{} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if data.Chart.Error != nil {
		return fmt.Errorf("erro no provedor: %v", data.Chart.Error)
	}

	if len(data.Chart.Result) == 0 {
		return errors.New("resultado histórico vazio")
	}

	res := data.Chart.Result[0]
	if len(res.Timestamp) == 0 || len(res.Indicators.Quote) == 0 {
		return errors.New("série histórica sem timestamps ou quotes")
	}

	closes := res.Indicators.Quote[0].Close
	if len(res.Timestamp) != len(closes) {
		return errors.New("inconsistência de tamanho nos dados históricos do provedor")
	}

	var prices []DailyPrice
	for i := range res.Timestamp {
		if closes[i] == nil {
			continue // Ignora dias sem cotação
		}
		prices = append(prices, DailyPrice{
			AssetID:    assetID,
			PriceDate:  time.Unix(res.Timestamp[i], 0).UTC(),
			ClosePrice: *closes[i],
		})
	}

	if len(prices) > 0 {
		err = s.repo.SaveDailyPrices(ctx, assetID, prices)
		if err != nil {
			return fmt.Errorf("falha ao gravar histórico no banco: %w", err)
		}
		log.Printf("[Backfill] Sincronizados %d preços históricos para o ativo %s", len(prices), ticker)
	}

	return nil
}

func (s *Service) getCurrencyRate(ctx context.Context, fromCurrency, toCurrency string) float64 {
	if fromCurrency == toCurrency {
		return 1.0
	}
	ticker := fmt.Sprintf("%s%s=X", fromCurrency, toCurrency)
	quote, err := s.marketService.GetQuote(ctx, ticker)
	if err == nil && quote.Price > 0 {
		return quote.Price
	}

	if fromCurrency == "USD" && toCurrency == "BRL" {
		quote, err = s.marketService.GetQuote(ctx, "USDBRL=X")
		if err == nil && quote.Price > 0 {
			return quote.Price
		}
		return 5.20 // Fallback seguro e condizente com a média histórica recente
	}
	return 1.0
}

// UpdateTransaction edita uma transação existente de um portfólio.
func (s *Service) UpdateTransaction(ctx context.Context, userID, portfolioID, txID string, tx *Transaction) error {
	// Anti-IDOR: Valida se a carteira pertence ao usuário logado
	p, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return errors.New("carteira não encontrada ou acesso não autorizado")
	}

	tx.Ticker = strings.ToUpper(strings.TrimSpace(tx.Ticker))
	if tx.Ticker == "" {
		return errors.New("ticker do ativo inválido")
	}

	assetID, currency, err := s.repo.GetAssetAndCurrencyByTicker(ctx, tx.Ticker)
	if err != nil {
		return errors.New("ativo não encontrado na base")
	}
	tx.AssetID = assetID

	// Correção Cambial: Se a taxa não foi fornecida, busca automaticamente
	if tx.ExchangeRate <= 0 {
		if currency != p.BaseCurrency {
			currencyPair := fmt.Sprintf("%s%s=X", currency, p.BaseCurrency)
			log.Printf("[Portfolio-Update] Buscando câmbio histórico para %s na data %s no banco de dados...", currencyPair, tx.ExecutedAt)
			
			rate, err := s.repo.GetExchangeRateByDate(ctx, currencyPair, tx.ExecutedAt)
			if err != nil || rate <= 0 {
				log.Printf("[Portfolio-Update] Taxa não encontrada na base. Disparando Micro-Backfill para tapar o buraco...")
				s.BackfillGap(ctx, currencyPair, tx.ExecutedAt)
				
				// Tenta buscar novamente
				rate, err = s.repo.GetExchangeRateByDate(ctx, currencyPair, tx.ExecutedAt)
			}
			
			if err == nil && rate > 0 {
				tx.ExchangeRate = rate
				log.Printf("[Portfolio-Update] Câmbio encontrado na base: %.4f", rate)
			} else {
				log.Printf("[Portfolio-Update] Aviso: Falha ao buscar câmbio histórico após backfill (%v). Usando fallback de 1.0", err)
				tx.ExchangeRate = 1.0
			}
		} else {
			tx.ExchangeRate = 1.0
		}
	}

	tx.ID = txID
	tx.PortfolioID = portfolioID
	tx.TotalCost = tx.Quantity * tx.UnitPrice

	// Executa atualização no banco
	err = s.repo.UpdateTransaction(ctx, *tx)
	if err != nil {
		return fmt.Errorf("falha ao atualizar transação: %w", err)
	}

	// Dispara verificação de Auto-Backfill de forma assíncrona para não travar o Update
	go func(id, ticker, curr string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		oldestDate, err := s.repo.GetOldestPriceDate(bgCtx, id)
		if err == nil && !oldestDate.IsZero() && tx.ExecutedAt.Before(oldestDate) {
			log.Printf("[Backfill-Update] Transação antiga detectada. Disparando BackfillGap para o ativo %s", ticker)
			if err := s.BackfillGap(bgCtx, ticker, tx.ExecutedAt); err != nil {
				log.Printf("[Backfill-Update] Falha ao rodar BackfillGap de %s: %v", ticker, err)
			}
		}
	}(tx.AssetID, tx.Ticker, currency)

	return nil
}

func (s *Service) GetUnifiedTransactions(ctx context.Context, portfolioID, userID string) ([]history.UnifiedTransaction, error) {
	txs, err := s.repo.GetTransactionsByPortfolioID(ctx, portfolioID, userID)
	if err != nil {
		return nil, err
	}

	var unified []history.UnifiedTransaction
	for _, tx := range txs {
		assetName := tx.Ticker
		assetType := tx.AssetType

		qty := tx.Quantity
		price := tx.UnitPrice
		exch := tx.ExchangeRate

		total := qty * price
		if tx.Type == "BONUS" || tx.Type == "SPLIT" || tx.Type == "REVERSE_SPLIT" {
			total = 0
		}

		unified = append(unified, history.UnifiedTransaction{
			ID:           tx.ID,
			PortfolioID:  tx.PortfolioID,
			Module:       "RV",
			Date:         tx.ExecutedAt,
			AssetName:    assetName,
			AssetType:    assetType,
			Type:         tx.Type,
			Quantity:     &qty,
			UnitPrice:    &price,
			ExchangeRate: &exch,
			TotalValue:   total,
			Currency:     tx.Currency,
		})
	}
	return unified, nil
}

// BackfillGap realiza uma chamada histórica direcionada ao provedor para preencher buracos no histórico diário.
// Ele baixa desde missingDate-5 dias até a data mais antiga registrada no banco.
func (s *Service) BackfillGap(ctx context.Context, ticker string, missingDate time.Time) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	assetID, err := s.repo.GetAssetByTicker(ctx, ticker)
	if err != nil {
		// Se a moeda nem existe, criamos como ativo silenciosamente
		assetID, err = s.repo.CreateAsset(ctx, ticker, ticker, "CURRENCY", "BRL")
		if err != nil {
			return fmt.Errorf("falha ao criar ativo cambial para backfill: %w", err)
		}
	}

	oldestDate, err := s.repo.GetOldestPriceDate(ctx, assetID)
	if err != nil || oldestDate.IsZero() {
		// Se não há histórico, preenchemos de missingDate até hoje
		oldestDate = time.Now()
	}

	// Queremos de (missingDate - 5 dias) até oldestDate
	period1 := missingDate.AddDate(0, 0, -5).Unix()
	period2 := oldestDate.Unix()

	if period1 >= period2 {
		return nil // Sem gap para baixar
	}

	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d", url.PathEscape(ticker), period1, period2)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("provedor yahoo retornou status %d", resp.StatusCode)
	}

	var data struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error interface{} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if data.Chart.Error != nil {
		return fmt.Errorf("erro no provedor: %v", data.Chart.Error)
	}

	if len(data.Chart.Result) == 0 {
		return errors.New("resultado histórico vazio")
	}

	res := data.Chart.Result[0]
	if len(res.Timestamp) == 0 || len(res.Indicators.Quote) == 0 {
		return errors.New("série histórica sem timestamps ou quotes")
	}

	closes := res.Indicators.Quote[0].Close
	if len(res.Timestamp) != len(closes) {
		return errors.New("inconsistência de tamanho nos dados históricos do provedor")
	}

	var prices []DailyPrice
	for i := range res.Timestamp {
		if closes[i] == nil {
			continue
		}
		prices = append(prices, DailyPrice{
			AssetID:    assetID,
			PriceDate:  time.Unix(res.Timestamp[i], 0).UTC(),
			ClosePrice: *closes[i],
		})
	}

	if len(prices) > 0 {
		err = s.repo.SaveDailyPrices(ctx, assetID, prices)
		if err != nil {
			return fmt.Errorf("falha ao gravar histórico no banco: %w", err)
		}
		log.Printf("[Micro-Backfill] Sincronizados %d preços para cobrir o buraco de %s", len(prices), ticker)
	}

	return nil
}

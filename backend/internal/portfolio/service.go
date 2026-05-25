package portfolio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/onigiri/stockpulse/backend/internal/market"
)

// Service gerencia as regras de negócio de carteiras, transações e histórico.
type Service struct {
	repo           *Repository
	marketService  *market.Service
	marketProvider market.QuoteProvider
	httpClient     *http.Client
}

// NewService cria uma nova instância de Service.
func NewService(repo *Repository, marketService *market.Service, marketProvider market.QuoteProvider) *Service {
	return &Service{
		repo:           repo,
		marketService:  marketService,
		marketProvider: marketProvider,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
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
		}
	}

	// Filtra apenas posições ativas (quantidade > 0)
	var activePositions []Position
	for _, pos := range posMap {
		if pos.Quantity > 0 {
			// Injeta cotações em tempo real e calcula rentabilidade
			quote, err := s.marketService.GetQuote(ctx, pos.Ticker)
			if err != nil {
				log.Printf("[Portfolio] Erro ao recuperar cotação atual para %s: %v", pos.Ticker, err)
				activePositions = append(activePositions, *pos)
				continue
			}

			// Conversão cambial em tempo real (se ativo for USD e carteira for BRL)
			rate := 1.0
			if pos.Currency != p.BaseCurrency {
				rate = s.getCurrencyRate(ctx, pos.Currency, p.BaseCurrency)
			}

			pos.CurrentPrice = quote.Price
			pos.CurrentValue = pos.Quantity * quote.Price * rate
			pos.ProfitLoss = pos.CurrentValue - pos.TotalCost
			if pos.TotalCost > 0 {
				pos.ReturnPercent = (pos.ProfitLoss / pos.TotalCost) * 100
			}

			activePositions = append(activePositions, *pos)
		}
	}

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
	assetID, err := s.repo.GetAssetByTicker(ctx, tx.Ticker)
	var currency string
	if err != nil {
		// Importa metadados do Yahoo Finance
		log.Printf("[Portfolio] Ativo %s não existe na base. Importando...", tx.Ticker)
		quote, err := s.marketProvider.GetQuote(ctx, tx.Ticker)
		if err != nil {
			return nil, fmt.Errorf("ativo '%s' não encontrado no mercado: %w", tx.Ticker, err)
		}

		assetType := "EQUITY"
		if quote.Currency == "USD" && !strings.Contains(tx.Ticker, ".") {
			assetType = "EQUITY_US"
		} else if strings.Contains(tx.Ticker, "-") {
			assetType = "CRYPTO"
		}

		assetID, err = s.repo.CreateAsset(ctx, tx.Ticker, quote.Name, assetType, quote.Currency)
		if err != nil {
			return nil, fmt.Errorf("erro ao registrar ativo localmente: %w", err)
		}
		currency = quote.Currency
	} else {
		// Recupera cotação para identificar a moeda do ativo
		quote, err := s.marketService.GetQuote(ctx, tx.Ticker)
		if err == nil {
			currency = quote.Currency
		} else {
			currency = "BRL"
		}
	}

	tx.AssetID = assetID
	tx.TotalCost = tx.Quantity * tx.UnitPrice

	// Executa inserção no banco
	savedTx, err := s.repo.CreateTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Dispara o Auto-Backfill de 5 anos de forma assíncrona (Goroutine controlada)
	go func(id, ticker, curr string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		// Verifica se o histórico já existe no banco para evitar bypasss inuteis
		existing, err := s.repo.GetDailyPrices(bgCtx, id, time.Now().AddDate(0, 0, -7), time.Now())
		if err == nil && len(existing) > 0 {
			log.Printf("[Backfill] Ativo %s já possui histórico recente. Pulando backfill.", ticker)
			return
		}

		log.Printf("[Backfill] Iniciando preenchimento histórico de 5 anos para %s...", ticker)
		if err := s.BackfillHistoricalPrices(bgCtx, id, ticker); err != nil {
			log.Printf("[Backfill] Falha ao rodar backfill histórico de %s: %v", ticker, err)
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

// GetPortfolioPerformance reconstrói a série histórica diária de evolução patrimonial aplicando LOCF.
func (s *Service) GetPortfolioPerformance(ctx context.Context, portfolioID, userID, period string) ([]PerformancePoint, error) {
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

	// Ordena cronologicamente
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
		for i := 0; i < 30; i++ {
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

		for _, tx := range txs {
			// Ignora transações ocorridas após a data analisada
			if tx.ExecutedAt.After(currDate) {
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

				totalMarketValue += qty * price * rate
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

// BackfillHistoricalPrices realiza a chamada de 5 anos histórica ao Yahoo e grava os dados.
func (s *Service) BackfillHistoricalPrices(ctx context.Context, assetID, ticker string) error {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=5y", url.PathEscape(ticker))

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

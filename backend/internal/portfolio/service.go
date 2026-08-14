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
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/onigiri/stock-pulse/backend/internal/calculator"
	"github.com/onigiri/stock-pulse/backend/internal/database"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/history"
	"github.com/onigiri/stock-pulse/backend/internal/market"
)

// determineAssetType define a categoria oficial do ativo no banco
func determineAssetType(ticker, name, currency string) string {
	return calculator.DetermineAssetType(ticker, name, currency)
}

// PortfolioRepository define as operações de banco de dados para a carteira.
type PortfolioRepository interface {
	CreatePortfolio(ctx context.Context, userID, name, baseCurrency string) (*Portfolio, error)
	GetPortfoliosByUserID(ctx context.Context, userID string) ([]Portfolio, error)
	GetPortfolioByID(ctx context.Context, id, userID string) (*Portfolio, error)
	SetDefaultPortfolio(ctx context.Context, portfolioID, userID string) error
	DeletePortfolio(ctx context.Context, id, userID string) error
	CreateTransaction(ctx context.Context, tx *Transaction) (*Transaction, error)
	UpdateTransaction(ctx context.Context, tx Transaction) error
	GetTransactionsByPortfolioID(ctx context.Context, portfolioID, userID string) ([]Transaction, error)
	DeleteTransaction(ctx context.Context, txID, portfolioID, userID string) error
	SaveDailyPrices(ctx context.Context, assetID string, prices []DailyPrice) error
	GetDailyPrices(ctx context.Context, assetID string, startDate, endDate time.Time) ([]DailyPrice, error)
	GetDailyPricesBatch(ctx context.Context, assetIDs []string, startDate, endDate time.Time) ([]DailyPrice, error)
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
	uow            database.UnitOfWork
}

// NewService cria uma nova instância de Service.
func NewService(repo PortfolioRepository, marketService MarketService, marketProvider market.QuoteProvider, fiService fixedincome.Service, uow database.UnitOfWork) *Service {
	return &Service{
		repo:           repo,
		marketService:  marketService,
		marketProvider: marketProvider,
		fiService:      fiService,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		uow: uow,
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

// SetDefaultPortfolio define uma carteira como padrão para o usuário.
func (s *Service) SetDefaultPortfolio(ctx context.Context, portfolioID, userID string) error {
	if portfolioID == "" || userID == "" {
		return errors.New("IDs inválidos")
	}
	return s.uow.Do(ctx, func(txCtx context.Context) error {
		return s.repo.SetDefaultPortfolio(txCtx, portfolioID, userID)
	})
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
	} else {
		hasDefault := false
		for _, p := range lists {
			if p.IsDefault {
				hasDefault = true
				break
			}
		}
		if !hasDefault {
			_ = s.SetDefaultPortfolio(ctx, lists[0].ID, userID)
			lists[0].IsDefault = true
		}
	}

	return lists, nil
}

// CalculatedDividend representa o dividendo calculado para o usuário.
type CalculatedDividend struct {
	AssetID        string    `json:"asset_id"`
	Ticker         string    `json:"ticker"`
	CumDate        time.Time `json:"cum_date"`
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

		pos.Quantity, pos.TotalCost, pos.AveragePrice = calculator.UpdatePositionOnTransaction(
			pos.Quantity, pos.TotalCost, pos.AveragePrice,
			tx.Type, tx.Quantity, tx.UnitPrice, tx.ExchangeRate, tx.Fee,
		)
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
				pos.DailyChange = quote.Change
				pos.DailyChangePercent = quote.ChangePercent
				pos.CurrentValue, pos.ProfitLoss, pos.ReturnPercent = calculator.CalculatePositionMetrics(
					pos.Quantity, quote.Price, pos.TotalCost, rate,
				)

				// Injeta fundamentos (Graham, Bazin, P/VP, P/L)
				if f, errF := s.marketService.GetFundamentals(ctx, pos.Ticker); errF == nil && f != nil {
					pos.GrahamValue = f.GrahamValue
					pos.BazinValue = f.BazinValue
					pos.DividendYield = f.DividendYield
					pos.PVP, pos.PE = calculator.CalculateValuationRatios(pos.CurrentPrice, f.BookValue, f.EPS)
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
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("erro ao consultar ativo no banco: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
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

	if tx.Type == "SELL" {
		tx.TotalCost = (tx.Quantity * tx.UnitPrice) - tx.Fee
	} else {
		tx.TotalCost = (tx.Quantity * tx.UnitPrice) + tx.Fee
	}
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
	return s.uow.Do(ctx, func(txCtx context.Context) error {
		return s.repo.DeletePortfolio(txCtx, id, userID)
	})
}

// PerformancePoint representa o saldo consolidado de um portfólio em uma data histórica com retornos e benchmarks.
type PerformancePoint struct {
	Date           string  `json:"date"`
	Value          float64 `json:"value"`
	TotalInvested  float64 `json:"total_invested"`
	ReturnPct      float64 `json:"return_pct"`
	CdiReturnPct   float64 `json:"cdi_return_pct"`
	IpcaReturnPct  float64 `json:"ipca_return_pct"`
	IfixReturnPct  float64 `json:"ifix_return_pct"`
	IbovReturnPct  float64 `json:"ibov_return_pct"`
	Sp500ReturnPct float64 `json:"sp500_return_pct"`
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
	if tx.Type == "SELL" {
		tx.TotalCost = (tx.Quantity * tx.UnitPrice) - tx.Fee
	} else {
		tx.TotalCost = (tx.Quantity * tx.UnitPrice) + tx.Fee
	}

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

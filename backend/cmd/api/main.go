package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/onigiri/stock-pulse/backend/internal/alert"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
	"github.com/onigiri/stock-pulse/backend/internal/database"
	"github.com/onigiri/stock-pulse/backend/internal/docs"

	"github.com/onigiri/stock-pulse/backend/internal/history"
	"github.com/onigiri/stock-pulse/backend/internal/market"
	customMiddleware "github.com/onigiri/stock-pulse/backend/internal/middleware"
	"github.com/onigiri/stock-pulse/backend/internal/fixedincome"
	"github.com/onigiri/stock-pulse/backend/internal/portfolio"
	"github.com/onigiri/stock-pulse/backend/internal/telegram"
	"github.com/onigiri/stock-pulse/backend/internal/watchlist"
	"github.com/onigiri/stock-pulse/backend/internal/websocket"
	"github.com/onigiri/stock-pulse/backend/internal/worker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Inicialização do Logger Estruturado JSON (slog) - Fase 4
	logLevelStr := os.Getenv("LOG_LEVEL")
	var level slog.Level
	switch strings.ToLower(logLevelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelWarn // Opção 1B (Padrão Silencioso)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Inicialização do Banco de Dados
	dbPool, err := database.NewPool()
	if err != nil {
		log.Fatalf("Falha ao conectar no banco de dados: %v", err)
	}
	defer dbPool.Close()

	// Inicialização do Redis (Cache & Session)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379" // Fallback local de dev
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()
	if err := rdb.Ping(redisCtx).Err(); err != nil {
		log.Fatalf("Falha ao conectar no Redis: %v", err)
	}
	defer rdb.Close()

	// Configuração de Segredos
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Variável de ambiente JWT_SECRET é obrigatória e não foi configurada.")
	}

	// Inicialização de Camadas de Autenticação
	authRepo := auth.NewRepository(dbPool)
	authService := auth.NewService(authRepo, rdb, jwtSecret)
	authHandler := auth.NewHandler(authService)

	// Inicialização de Camadas de Market Data
	var marketProvider market.QuoteProvider
	if os.Getenv("MOCK_EXTERNAL_APIS") == "true" {
		marketProvider = market.NewMockProvider()
		fmt.Println("Aviso: Inicializando Market Data usando MockProvider (MOCK_EXTERNAL_APIS=true)")
	} else {
		fmt.Println("Aviso: Inicializando Yahoo Finance Provider (MOCK_EXTERNAL_APIS=false)")
		marketProvider = market.NewYahooFinanceProvider()
	}
	marketService := market.NewService(marketProvider, rdb)
	marketHandler := market.NewHandler(marketService)

	// Inicialização de Camadas de Watchlist
	watchlistRepo := watchlist.NewRepository(dbPool)
	watchlistService := watchlist.NewService(watchlistRepo, marketService, marketProvider)
	watchlistHandler := watchlist.NewHandler(watchlistService)

	// Inicialização de Camadas de Renda Fixa (Fase 5)
	fiRepo := fixedincome.NewRepository(dbPool)
	fiBcbClient := fixedincome.NewBCBClient()
	fiService := fixedincome.NewService(fiRepo, fiBcbClient)
	fiHandler := fixedincome.NewHandler(fiService, fiRepo)
	fiWorker := fixedincome.NewWorker(fiRepo, fiBcbClient)

	// Inicialização de Camadas de Portfólio & Daily Worker
	portfolioRepo := portfolio.NewRepository(dbPool)
	portfolioService := portfolio.NewService(portfolioRepo, marketService, marketProvider, fiService)
	portfolioHandler := portfolio.NewHandler(portfolioService)
	portfolioWorker := portfolio.NewDailyWorker(portfolioRepo, marketProvider)
	dividendWorker := portfolio.NewDividendWorker(portfolioRepo, marketService)

	// Inicialização das Camadas da Fase 3 (Alertas & Tempo Real)

	wsHub := websocket.NewHub(marketService)
	wsHandler := websocket.NewHandler(wsHub)

	alertRepo := alert.NewRepository(dbPool)
	alertService := alert.NewService(alertRepo, marketProvider)
	alertHandler := alert.NewHandler(alertService)
	
	historyService := history.NewService(portfolioService, fiService)
	historyHandler := history.NewHandler(historyService)

	// Telegram Bot
	telegramRepo := telegram.NewRepository(dbPool)
	telegramService := telegram.NewService(telegramRepo, rdb)
	telegramHandlers := telegram.NewHandlers(telegramService, portfolioService, marketService, fiService)
	telegramBot, err := telegram.NewBotRunner(os.Getenv("TELEGRAM_BOT_TOKEN"), telegramHandlers)
	if err != nil {
		slog.Error("Failed to start telegram bot", "err", err)
	}
	telegramHttpHandler := telegram.NewHTTPHandler(telegramService, telegramBot.GetUsername())

	alertWorker := alert.NewAlertWorker(alertRepo, marketService, telegramBot)

	// Inicialização da Documentação API Swagger (Fase 4)
	docsHandler := docs.NewHandler("docs/openapi.yaml")

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	workerManager := worker.NewManager()
	workerManager.Register(worker.NewWorker("DividendWorker", 24*time.Hour, dividendWorker.SyncAllDividends))
	workerManager.Register(worker.NewWorker("DailyWorker", 24*time.Hour, portfolioWorker.Run))
	workerManager.Register(worker.NewWorker("FixedIncomeWorker", 24*time.Hour, fiWorker.SyncRates))
	workerManager.Register(worker.NewWorker("AlertWorker", alertWorker.Interval(), alertWorker.CheckActiveAlerts))
	
	workerManager.StartAll(workerCtx)
	workerHandler := worker.NewHandler(workerManager)

	go wsHub.Start(workerCtx)
	if telegramBot != nil {
		go telegramBot.Start()
	}

	// Configuração das Rotas (Chi)
	r := chi.NewRouter()
	r.Use(customMiddleware.CORS())    // CORS seguro com credenciais
	r.Use(customMiddleware.Metrics()) // Coleta de métricas Prometheus (Fase 4)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// Exposição de Métricas Prometheus (Fase 4)
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		err := dbPool.Ping(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Database is down"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - Database Connected"))
	})

	// Rotas da API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Documentação interativa da API (Fase 4)
		r.Get("/swagger", docsHandler.ServeUI)
		r.Get("/swagger/openapi.yaml", docsHandler.ServeYAML)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.Refresh)
			r.With(customMiddleware.AuthRequired([]byte(jwtSecret))).Get("/me", authHandler.Me)
		})

		// Rotas Protegidas (Exige Sessão)
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.AuthRequired([]byte(jwtSecret)))

			// Cotações e Busca
			r.Get("/quotes/{ticker}", marketHandler.GetQuote)
			r.Get("/assets/search", marketHandler.Search)

			// Favoritos / Watchlists
			r.Get("/watchlists", watchlistHandler.GetWatchlists)
			r.Post("/watchlists", watchlistHandler.CreateWatchlist)
			r.Get("/watchlists/{id}", watchlistHandler.GetWatchlist)
			r.Delete("/watchlists/{id}", watchlistHandler.DeleteWatchlist)
			r.Post("/watchlists/{id}/items", watchlistHandler.AddAsset)
			r.Delete("/watchlists/{id}/items/{ticker}", watchlistHandler.RemoveAsset)

			// Carteiras / Portfólios
			r.Get("/portfolios", portfolioHandler.GetPortfolios)
			r.Post("/portfolios", portfolioHandler.CreatePortfolio)
			r.Get("/portfolios/{id}", portfolioHandler.GetPortfolio)
			r.Delete("/portfolios/{id}", portfolioHandler.DeletePortfolio)
			r.Get("/portfolios/{id}/transactions", portfolioHandler.GetTransactions)
			r.Post("/portfolios/{id}/transactions", portfolioHandler.AddTransaction)
			r.Post("/portfolios/{id}/transactions/bulk", portfolioHandler.BulkImportTransactions)
			r.Put("/portfolios/{id}/transactions/{txId}", portfolioHandler.UpdateTransaction)
			r.Delete("/portfolios/{id}/transactions/{txId}", portfolioHandler.DeleteTransaction)
			r.Get("/portfolios/{id}/performance", portfolioHandler.GetPerformance)
			r.Get("/portfolios/{id}/dividends", portfolioHandler.GetDividends)
			r.Get("/portfolios/{id}/export", portfolioHandler.ExportPortfolio)

			// Renda Fixa
			fiHandler.RegisterRoutes(r)
			historyHandler.RegisterRoutes(r)

			// Conexão WebSocket em Tempo Real (Fase 3)
			r.Get("/ws", wsHandler.ServeWS)

			// Gestão de Alertas (Fase 3)
			r.Get("/alerts", alertHandler.GetAlerts)
			r.Post("/alerts", alertHandler.CreateAlert)
			r.Delete("/alerts/{id}", alertHandler.DeleteAlert)
			r.Put("/alerts/{id}/toggle", alertHandler.ToggleAlert)

			// Integração Telegram
			r.Route("/telegram", func(r chi.Router) {
				r.Post("/link", telegramHttpHandler.GenerateLinkToken)
			})

			// Workers / System Management
			r.Route("/workers", func(r chi.Router) {
				workerHandler.RegisterRoutes(r)
			})
		})
	})

	// Configuração do Servidor HTTP
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful Shutdown (RNF09)
	go func() {
		fmt.Printf("Servidor stock-pulse rodando na porta %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro crítico no servidor: %v\n", err)
		}
	}()

	// Aguarda sinal de interrupção (SIGINT/SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Sinal de desligamento recebido. Encerrando servidor com Graceful Shutdown...")
	workerCancel() // Encerra o Daily Worker em background imediatamente

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Erro durante o Graceful Shutdown: %v", err)
	}

	fmt.Println("Servidor encerrado com segurança.")
}

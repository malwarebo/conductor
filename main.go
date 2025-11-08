package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/malwarebo/conductor/analytics"
	"github.com/malwarebo/conductor/api"
	"github.com/malwarebo/conductor/cache"
	"github.com/malwarebo/conductor/config"
	"github.com/malwarebo/conductor/db"
	"github.com/malwarebo/conductor/middleware"
	"github.com/malwarebo/conductor/monitoring"
	"github.com/malwarebo/conductor/observability"
	"github.com/malwarebo/conductor/providers"
	"github.com/malwarebo/conductor/stores"
	"github.com/malwarebo/conductor/security"
	"github.com/malwarebo/conductor/services"
	"github.com/malwarebo/conductor/webhooks"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func printBanner() {
	fmt.Printf("%s%s", colorCyan, colorBold)
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                              ║")
	fmt.Println("║  Conductor - Payment Orchestration Platform              ║")
	fmt.Println("║                                                              ║")
	fmt.Println("║  Multi-provider payment processing made simple              ║")
	fmt.Println("║                                                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("%s", colorReset)
}

func printStep(step, message string) {
	fmt.Printf("%s[%s]%s %s%s%s\n", colorBlue, step, colorReset, colorBold, message, colorReset)
}

func printSuccess(message string) {
	fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, message)
}

func printWarning(message string) {
	fmt.Printf("%s⚠%s %s\n", colorYellow, colorReset, message)
}

func printError(message string) {
	fmt.Printf("%s✗%s %s\n", colorRed, colorReset, message)
}

func printInfo(message string) {
	fmt.Printf("%sℹ%s %s\n", colorCyan, colorReset, message)
}

func main() {
	printBanner()
	fmt.Println()

	printStep("1/10", "Loading configuration...")
	cfg, err := config.CreateLoadConfig()
	if err != nil {
		printError(fmt.Sprintf("Failed to load configuration: %v", err))
		os.Exit(1)
	}
	printSuccess("Configuration loaded successfully")

	printStep("2/10", "Validating configuration...")
	if err := cfg.Validate(); err != nil {
		printError(fmt.Sprintf("Configuration validation failed: %v", err))
		os.Exit(1)
	}
	printSuccess("Configuration validation passed")

	printStep("3/10", "Connecting to database...")
	poolConfig := db.PoolConfig{
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.MaxLifetime,
		ConnMaxIdleTime: cfg.Database.MaxIdleTime,
		MaxRetries:      3,
		RetryDelay:      time.Second,
	}

	connectionPool, err := db.CreateNewConnectionPool(cfg.GetDatabaseURL(), cfg.Database.ReplicaDSNs, poolConfig)
	if err != nil {
		printError(fmt.Sprintf("Failed to create connection pool: %v", err))
		os.Exit(1)
	}
	defer connectionPool.Close()

	database := connectionPool.GetPrimary()
	printSuccess(fmt.Sprintf("Connected to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port))

	printStep("3.1/10", "Running database migrations...")
	migrator := db.CreateNewMigrator(database)
	if err := migrator.LoadMigrationsFromDir("db/migrations"); err != nil {
		printWarning(fmt.Sprintf("Failed to load migrations: %v", err))
	} else {
		if err := migrator.Up(); err != nil {
			printWarning(fmt.Sprintf("Failed to run migrations: %v", err))
		} else {
			printSuccess("Database migrations completed")
		}
	}

	printStep("4/10", "Connecting to Redis...")
	redisCache, err := cache.CreateRedisCache(cache.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		TTL:      cfg.Redis.TTL,
	})
	if err != nil {
		printWarning(fmt.Sprintf("Failed to connect to Redis: %v (continuing without cache)", err))
	} else {
		defer redisCache.Close()
		printSuccess(fmt.Sprintf("Connected to Redis at %s:%d", cfg.Redis.Host, cfg.Redis.Port))
	}

	_ = cache.CacheConfig{
		Strategy:       cache.WriteThrough,
		TTL:            cfg.Redis.TTL,
		MaxSize:        1000,
		EvictionPolicy: "lru",
	}

	printStep("5/10", "Initializing security components...")
	encryptionKey, err := security.CreateGenerateEncryptionKey()
	if err != nil {
		printError(fmt.Sprintf("Failed to generate encryption key: %v", err))
		os.Exit(1)
	}

	encryption, err := security.CreateEncryptionManager(encryptionKey)
	if err != nil {
		printError(fmt.Sprintf("Failed to initialize encryption: %v", err))
		os.Exit(1)
	}

	jwtManager := security.CreateJWTManager(cfg.Security.JWTSecret, "conductor", "conductor-api")
	_ = security.CreateAPIKeyManager()
	_ = security.CreateAuditLogger()
	_ = security.CreateValidator()

	rateLimiter := security.CreateTieredRateLimiter(map[string]security.RateLimitConfig{
		"default":  {RequestsPerSecond: 10, Burst: 20, Window: time.Minute},
		"premium":  {RequestsPerSecond: 100, Burst: 200, Window: time.Minute},
		"standard": {RequestsPerSecond: 50, Burst: 100, Window: time.Minute},
	})
	printSuccess("Security components initialized")

	printStep("6/10", "Initializing monitoring and alerting...")
	alertManager := monitoring.CreateAlertManager()
	_ = monitoring.CreateMetricsCollector()
	healthService := monitoring.CreateHealthService("1.0.0")

	alertManager.AddRule(&monitoring.AlertRule{
		ID:   "high_error_rate",
		Name: "High Error Rate",
		Condition: func(metrics map[string]float64) bool {
			return metrics["error_rate"] > 0.05
		},
		Level:    monitoring.Critical,
		Cooldown: 5 * time.Minute,
		Enabled:  true,
	})

	alertManager.AddRule(&monitoring.AlertRule{
		ID:   "high_response_time",
		Name: "High Response Time",
		Condition: func(metrics map[string]float64) bool {
			return metrics["response_time"] > 2000
		},
		Level:    monitoring.Warning,
		Cooldown: 2 * time.Minute,
		Enabled:  true,
	})

	healthService.AddCheck("database", func(ctx context.Context) error {
		sqlDB, err := database.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	})

	healthService.AddCheck("redis", func(ctx context.Context) error {
		if redisCache == nil {
			return fmt.Errorf("redis not available")
		}
		return nil
	})

	printSuccess("Monitoring and alerting initialized")

	printStep("7/10", "Initializing stores...")
	paymentRepo := stores.CreatePaymentRepository(database)
	planRepo := stores.CreatePlanRepository(database)
	subscriptionRepo := stores.CreateSubscriptionRepository(database)
	disputeRepo := stores.CreateDisputeRepository(database)
	fraudRepo := stores.CreateFraudRepository(database)
	customerStore := stores.CreateCustomerStore(database)
	paymentMethodStore := stores.CreatePaymentMethodStore(database)
	providerMappingStore := stores.CreateProviderMappingStore(database)
	printSuccess("Stores initialized")

	printStep("8/10", "Initializing payment providers...")
	stripeProvider := providers.CreateStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.CreateXenditProvider(cfg.Xendit.Secret)

	providerSelector := providers.CreateMultiProviderSelector([]providers.PaymentProvider{stripeProvider, xenditProvider}, providerMappingStore)
	printSuccess("Payment providers initialized")
	printInfo("  • Stripe: Ready for USD, EUR, GBP")
	printInfo("  • Xendit: Ready for IDR, SGD, MYR, PHP, THB, VND")
	_ = customerStore
	_ = paymentMethodStore

	printStep("9/10", "Initializing services...")
	paymentService := services.CreatePaymentService(paymentRepo, providerSelector)
	subscriptionService := services.CreateSubscriptionService(planRepo, subscriptionRepo, providerSelector)
	disputeService := services.CreateDisputeService(disputeRepo, providerSelector)
	fraudService := services.CreateFraudService(fraudRepo, cfg.OpenAI.APIKey)
	routingService := services.CreateRoutingService(cfg.OpenAI.APIKey)

	_ = webhooks.CreateWebhookManager()
	_ = analytics.CreateAnalyticsReporter()

	printSuccess("Services initialized")

	printStep("10/10", "Initializing observability...")
	healthChecker := observability.CreateHealthChecker()

	healthChecker.AddCheck("database", func(ctx context.Context) error {
		sqlDB, err := database.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	})

	healthChecker.AddCheck("redis", func(ctx context.Context) error {
		if redisCache == nil {
			return fmt.Errorf("redis not available")
		}
		return nil
	})

	printSuccess("Observability initialized")

	printStep("11/11", "Setting up HTTP server...")
	paymentHandler := api.CreatePaymentHandler(paymentService)
	subscriptionHandler := api.CreateSubscriptionHandler(subscriptionService)
	disputeHandler := api.CreateDisputeHandler(disputeService)
	fraudHandler := api.CreateFraudHandler(fraudService)
	routingHandler := api.CreateRoutingHandler(routingService)

	router := mux.NewRouter()

	authMiddleware := middleware.CreateAuthMiddleware(jwtManager, rateLimiter, encryption, cfg.Security.WebhookSecret)

	router.Use(middleware.CreateLoggingMiddleware)
	router.Use(authMiddleware.HeadersMiddleware)
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	router.Use(middleware.CreateCORSMiddleware(allowedOrigins))
	router.Use(middleware.CreateRecoveryMiddleware)

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(authMiddleware.RateLimitMiddleware)
	apiRouter.Use(authMiddleware.JWTMiddleware)
	apiRouter.Use(authMiddleware.EncryptionMiddleware)

	apiRouter.HandleFunc("/health", api.CreateHealthCheckHandler).Methods("GET")
	apiRouter.HandleFunc("/metrics", api.CreateMetricsHandler).Methods("GET")

	apiRouter.HandleFunc("/charges", paymentHandler.HandleCharge).Methods("POST")
	apiRouter.HandleFunc("/refunds", paymentHandler.HandleRefund).Methods("POST")
	apiRouter.HandleFunc("/plans", subscriptionHandler.HandlePlans).Methods("POST", "GET")
	apiRouter.HandleFunc("/plans/{id}", subscriptionHandler.HandlePlans).Methods("GET", "PUT", "DELETE")

	apiRouter.HandleFunc("/subscriptions", subscriptionHandler.HandleSubscriptions).Methods("POST", "GET")
	apiRouter.HandleFunc("/subscriptions/{id}", subscriptionHandler.HandleSubscriptions).Methods("GET", "PUT", "DELETE")

	apiRouter.HandleFunc("/disputes", disputeHandler.HandleDisputes).Methods("POST", "GET")
	apiRouter.HandleFunc("/disputes/{id}", disputeHandler.HandleDisputes).Methods("GET", "PUT")
	apiRouter.HandleFunc("/disputes/{id}/evidence", disputeHandler.HandleDisputes).Methods("POST")
	apiRouter.HandleFunc("/disputes/stats", disputeHandler.HandleDisputes).Methods("GET")

	apiRouter.HandleFunc("/fraud/analyze", fraudHandler.AnalyzeTransaction).Methods("POST")
	apiRouter.HandleFunc("/fraud/stats", fraudHandler.GetStats).Methods("GET")

	apiRouter.HandleFunc("/routing/select", routingHandler.HandleRouting).Methods("POST")
	apiRouter.HandleFunc("/routing/stats", routingHandler.HandleProviderStats).Methods("GET")
	apiRouter.HandleFunc("/routing/metrics", routingHandler.HandleRoutingMetrics).Methods("GET")
	apiRouter.HandleFunc("/routing/config", routingHandler.HandleRoutingConfig).Methods("GET", "PUT")

	webhookRouter := router.PathPrefix("/api/v1/webhooks").Subrouter()
	webhookRouter.Use(authMiddleware.WebhookMiddleware)
	webhookRouter.HandleFunc("/stripe", paymentHandler.HandleStripeWebhook).Methods("POST")
	webhookRouter.HandleFunc("/xendit", paymentHandler.HandleXenditWebhook).Methods("POST")

	server := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	printSuccess("HTTP server configured")

	fmt.Println()
	fmt.Printf("%s%s Conductor is ready!%s\n", colorGreen, colorBold, colorReset)
	fmt.Println()
	fmt.Printf("%s%sAPI Endpoints:%s\n", colorPurple, colorBold, colorReset)
	fmt.Printf("  %s•%s Health Check: %shttp://localhost:%s/api/v1/health%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s•%s Metrics:      %shttp://localhost:%s/api/v1/metrics%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s•%s Payments:     %shttp://localhost:%s/api/v1/charges%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s•%s Subscriptions: %shttp://localhost:%s/api/v1/subscriptions%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s•%s Disputes:     %shttp://localhost:%s/api/v1/disputes%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s•%s Fraud Detection: %shttp://localhost:%s/api/v1/fraud/analyze%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s•%s AI Routing:     %shttp://localhost:%s/api/v1/routing/select%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Println()
	fmt.Printf("%s%sEnvironment:%s %s%s%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Environment, colorReset)
	fmt.Printf("%s%sServer Port:%s %s%s%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("%s%sDatabase:%s %s%s:%d%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Database.Host, cfg.Database.Port, colorReset)
	if redisCache != nil {
		fmt.Printf("%s%sRedis:%s %s%s:%d%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Redis.Host, cfg.Redis.Port, colorReset)
	}
	fmt.Printf("%s%sSecurity:%s %sJWT + Encryption + Rate Limiting%s\n", colorPurple, colorBold, colorReset, colorYellow, "", colorReset)
	fmt.Printf("%s%sMonitoring:%s %sAlerts + Metrics + Health Checks%s\n", colorPurple, colorBold, colorReset, colorYellow, "", colorReset)
	fmt.Println()
	fmt.Printf("%s%sPress Ctrl+C to stop the server%s\n", colorYellow, colorBold, colorReset)
	fmt.Println()

	go func() {
		printInfo(fmt.Sprintf("Starting HTTP server on port %s...", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			printError(fmt.Sprintf("Server failed to start: %v", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println()
	printWarning("Shutting down Conductor server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		printError(fmt.Sprintf("Server forced to shutdown: %v", err))
		os.Exit(1)
	}

	alertManager.Close()
	rateLimiter.Close()

	printSuccess("Conductor server stopped gracefully")
	fmt.Println()
	fmt.Printf("%s%s👋 Thanks for using Conductor!%s\n", colorCyan, colorBold, colorReset)
}

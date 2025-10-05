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
	"github.com/malwarebo/gopay/api"
	"github.com/malwarebo/gopay/cache"
	"github.com/malwarebo/gopay/config"
	"github.com/malwarebo/gopay/middleware"
	"github.com/malwarebo/gopay/monitoring"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
	"github.com/malwarebo/gopay/security"
	"github.com/malwarebo/gopay/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

func printBanner() {
	fmt.Printf("%s%s", colorCyan, colorBold)
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                              ║")
	fmt.Println("║  🚀 GoPay Payment Orchestration System                      ║")
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
	cfg, err := config.LoadConfig()
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
	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseURL()), &gorm.Config{})
	if err != nil {
		printError(fmt.Sprintf("Failed to connect to database: %v", err))
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		printError(fmt.Sprintf("Failed to get database instance: %v", err))
		os.Exit(1)
	}
	defer sqlDB.Close()

	printSuccess(fmt.Sprintf("Connected to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port))

	printStep("4/10", "Connecting to Redis...")
	redisCache, err := cache.NewRedisCache(cache.RedisConfig{
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

	printStep("5/10", "Initializing security components...")
	encryptionKey, err := security.GenerateEncryptionKey()
	if err != nil {
		printError(fmt.Sprintf("Failed to generate encryption key: %v", err))
		os.Exit(1)
	}

	encryption, err := security.NewEncryptionManager(encryptionKey)
	if err != nil {
		printError(fmt.Sprintf("Failed to initialize encryption: %v", err))
		os.Exit(1)
	}

	jwtManager := security.NewJWTManager(cfg.Security.JWTSecret, "gopay", "gopay-api")

	rateLimiter := security.NewTieredRateLimiter(map[string]security.RateLimitConfig{
		"default":  {RequestsPerSecond: 10, Burst: 20, Window: time.Minute},
		"premium":  {RequestsPerSecond: 100, Burst: 200, Window: time.Minute},
		"standard": {RequestsPerSecond: 50, Burst: 100, Window: time.Minute},
	})
	printSuccess("Security components initialized")

	printStep("6/10", "Initializing monitoring and alerting...")
	alertManager := monitoring.NewAlertManager()

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
	printSuccess("Monitoring and alerting initialized")

	printStep("7/10", "Initializing payment providers...")
	stripeProvider := providers.NewStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.NewXenditProvider(cfg.Xendit.Secret)

	providerSelector := providers.NewMultiProviderSelector([]providers.PaymentProvider{stripeProvider, xenditProvider})
	printSuccess("Payment providers initialized")
	printInfo("  • Stripe: Ready for USD, EUR, GBP")
	printInfo("  • Xendit: Ready for IDR, SGD, MYR, PHP, THB, VND")

	printStep("8/10", "Initializing repositories...")
	paymentRepo := repositories.NewPaymentRepository(db)
	planRepo := repositories.NewPlanRepository(db)
	subscriptionRepo := repositories.NewSubscriptionRepository(db)
	disputeRepo := repositories.NewDisputeRepository(db)
	fraudRepo := repositories.NewFraudRepository(db)
	printSuccess("Repositories initialized")

	printStep("9/10", "Initializing services...")
	paymentService := services.NewPaymentService(paymentRepo, providerSelector)
	subscriptionService := services.NewSubscriptionService(planRepo, subscriptionRepo, providerSelector)
	disputeService := services.NewDisputeService(disputeRepo, providerSelector)
	fraudService := services.NewFraudService(fraudRepo, cfg.OpenAI.APIKey)
	routingService := services.NewRoutingService(cfg.OpenAI.APIKey)
	printSuccess("Services initialized")

	printStep("10/10", "Setting up HTTP server...")
	paymentHandler := api.NewPaymentHandler(paymentService)
	subscriptionHandler := api.NewSubscriptionHandler(subscriptionService)
	disputeHandler := api.NewDisputeHandler(disputeService)
	fraudHandler := api.NewFraudHandler(fraudService)
	routingHandler := api.NewRoutingHandler(routingService)

	router := mux.NewRouter()

	authMiddleware := middleware.NewAuthMiddleware(jwtManager, rateLimiter, encryption, cfg.Security.WebhookSecret)

	router.Use(middleware.LoggingMiddleware)
	router.Use(authMiddleware.HeadersMiddleware)
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	router.Use(middleware.CORSMiddleware(allowedOrigins))
	router.Use(middleware.RecoveryMiddleware)

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(authMiddleware.RateLimitMiddleware)
	apiRouter.Use(authMiddleware.JWTMiddleware)
	apiRouter.Use(authMiddleware.EncryptionMiddleware)

	apiRouter.HandleFunc("/health", api.HealthCheckHandler).Methods("GET")
	apiRouter.HandleFunc("/metrics", api.MetricsHandler).Methods("GET")

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
	fmt.Printf("%s%s🎉 GoPay is ready!%s\n", colorGreen, colorBold, colorReset)
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
	printWarning("Shutting down GoPay server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		printError(fmt.Sprintf("Server forced to shutdown: %v", err))
		os.Exit(1)
	}

	alertManager.Close()
	rateLimiter.Close()

	printSuccess("GoPay server stopped gracefully")
	fmt.Println()
	fmt.Printf("%s%s👋 Thanks for using GoPay!%s\n", colorCyan, colorBold, colorReset)
}

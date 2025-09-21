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
	"github.com/malwarebo/gopay/db"
	"github.com/malwarebo/gopay/middleware"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
	"github.com/malwarebo/gopay/services"
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

	// Configuration
	printStep("1/8", "Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		printError(fmt.Sprintf("Failed to load configuration: %v", err))
		os.Exit(1)
	}
	printSuccess("Configuration loaded successfully")

	// Validate configuration
	printStep("2/8", "Validating configuration...")
	if err := cfg.Validate(); err != nil {
		printError(fmt.Sprintf("Configuration validation failed: %v", err))
		os.Exit(1)
	}
	printSuccess("Configuration validation passed")

	// Database connection
	printStep("3/8", "Connecting to database...")
	db, err := db.NewDB(cfg.GetDatabaseURL())
	if err != nil {
		printError(fmt.Sprintf("Failed to connect to database: %v", err))
		os.Exit(1)
	}
	defer db.Close()
	printSuccess(fmt.Sprintf("Connected to PostgreSQL at %s:%d", cfg.Database.Host, cfg.Database.Port))

	// Redis connection
	printStep("4/8", "Connecting to Redis...")
	redisCache, err := cache.NewRedisCache(cache.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		TTL:      time.Duration(cfg.Redis.TTL) * time.Second,
	})
	if err != nil {
		printWarning(fmt.Sprintf("Failed to connect to Redis: %v (continuing without cache)", err))
	} else {
		defer redisCache.Close()
		printSuccess(fmt.Sprintf("Connected to Redis at %s:%d", cfg.Redis.Host, cfg.Redis.Port))
	}

	// Initialize payment providers
	printStep("5/8", "Initializing payment providers...")
	stripeProvider := providers.NewStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.NewXenditProvider(cfg.Xendit.Secret)

	providerSelector := providers.NewMultiProviderSelector([]providers.PaymentProvider{stripeProvider, xenditProvider})
	printSuccess("Payment providers initialized")
	printInfo("  • Stripe: Ready for USD, EUR, GBP")
	printInfo("  • Xendit: Ready for IDR, SGD, MYR, PHP, THB, VND")

	// Initialize repositories
	printStep("6/8", "Initializing repositories...")
	paymentRepo := repositories.NewPaymentRepository(db)
	planRepo := repositories.NewPlanRepository(db)
	subscriptionRepo := repositories.NewSubscriptionRepository(db)
	disputeRepo := repositories.NewDisputeRepository(db.DB)
	fraudRepo := repositories.NewFraudRepository(db.DB)
	printSuccess("Repositories initialized")

	// Initialize services
	printStep("7/8", "Initializing services...")
	paymentService := services.NewPaymentService(paymentRepo, providerSelector)
	subscriptionService := services.NewSubscriptionService(planRepo, subscriptionRepo, providerSelector)
	disputeService := services.NewDisputeService(disputeRepo, providerSelector)
	fraudService := services.NewFraudService(fraudRepo, cfg.OpenAI.APIKey)
	printSuccess("Services initialized")

	// Initialize handlers and router
	printStep("8/8", "Setting up HTTP server...")
	paymentHandler := api.NewPaymentHandler(paymentService)
	subscriptionHandler := api.NewSubscriptionHandler(subscriptionService)
	disputeHandler := api.NewDisputeHandler(disputeService)
	fraudHandler := api.NewFraudHandler(fraudService)

	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.LoggingMiddleware)
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	router.Use(middleware.CORSMiddleware(allowedOrigins))
	router.Use(middleware.RecoveryMiddleware)

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.RateLimitMiddleware)
	apiRouter.Use(middleware.AuthMiddleware)

	// Register routes
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

	apiRouter.HandleFunc("/webhooks/stripe", paymentHandler.HandleStripeWebhook).Methods("POST")
	apiRouter.HandleFunc("/webhooks/xendit", paymentHandler.HandleXenditWebhook).Methods("POST")

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	printSuccess("HTTP server configured")

	// Startup complete
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
	fmt.Printf("  %s•%s Fraud Stats:    %shttp://localhost:%s/api/v1/fraud/stats%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Println()
	fmt.Printf("%s%sEnvironment:%s %s%s%s\n", colorPurple, colorBold, colorReset, colorYellow, "development", colorReset)
	fmt.Printf("%s%sServer Port:%s %s%s%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("%s%sDatabase:%s %s%s:%d%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Database.Host, cfg.Database.Port, colorReset)
	if redisCache != nil {
		fmt.Printf("%s%sRedis:%s %s%s:%d%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Redis.Host, cfg.Redis.Port, colorReset)
	}
	fmt.Println()
	fmt.Printf("%s%sPress Ctrl+C to stop the server%s\n", colorYellow, colorBold, colorReset)
	fmt.Println()

	// Start server
	go func() {
		printInfo(fmt.Sprintf("Starting HTTP server on port %s...", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			printError(fmt.Sprintf("Server failed to start: %v", err))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
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

	printSuccess("GoPay server stopped gracefully")
	fmt.Println()
	fmt.Printf("%s%s👋 Thanks for using GoPay!%s\n", colorCyan, colorBold, colorReset)
}

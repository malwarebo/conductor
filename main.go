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
	"github.com/malwarebo/conductor/api"
	"github.com/malwarebo/conductor/cache"
	"github.com/malwarebo/conductor/config"
	"github.com/malwarebo/conductor/db"
	"github.com/malwarebo/conductor/middleware"
	"github.com/malwarebo/conductor/providers"
	"github.com/malwarebo/conductor/security"
	"github.com/malwarebo/conductor/services"
	"github.com/malwarebo/conductor/stores"
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
	fmt.Println("║  Conductor - Payment Orchestration Platform                  ║")
	fmt.Println("║                                                              ║")
	fmt.Println("║  Multi-provider payment processing                           ║")
	fmt.Println("║                                                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("%s", colorReset)
}

func printStep(step, message string) {
	fmt.Printf("%s[%s]%s %s%s%s\n", colorBlue, step, colorReset, colorBold, message, colorReset)
}

func printSuccess(message string) {
	fmt.Printf("%s%s %s\n", colorGreen, colorReset, message)
}

func printWarning(message string) {
	fmt.Printf("%s%s %s\n", colorYellow, colorReset, message)
}

func printError(message string) {
	fmt.Printf("%s%s %s\n", colorRed, colorReset, message)
}

func printInfo(message string) {
	fmt.Printf("%s%s %s\n", colorCyan, colorReset, message)
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

	rateLimiter := security.CreateTieredRateLimiter(map[string]security.RateLimitConfig{
		"default":  {RequestsPerSecond: 10, Burst: 20, Window: time.Minute},
		"premium":  {RequestsPerSecond: 100, Burst: 200, Window: time.Minute},
		"standard": {RequestsPerSecond: 50, Burst: 100, Window: time.Minute},
	})
	printSuccess("Security components initialized")

	printStep("6/8", "Initializing stores...")
	paymentRepo := stores.CreatePaymentRepository(database)
	planRepo := stores.CreatePlanRepository(database)
	subscriptionRepo := stores.CreateSubscriptionRepository(database)
	disputeRepo := stores.CreateDisputeRepository(database)
	fraudRepo := stores.CreateFraudRepository(database)
	providerMappingStore := stores.CreateProviderMappingStore(database)
	idempotencyStore := stores.CreateIdempotencyStore(database)
	auditStore := stores.CreateAuditStore(database)
	tenantStore := stores.CreateTenantStore(database)
	webhookStore := stores.CreateWebhookStore(database)
	customerStore := stores.CreateCustomerStore(database)
	paymentMethodStore := stores.CreatePaymentMethodStore(database)
	printSuccess("Stores initialized")

	printStep("7/8", "Initializing payment providers...")
	stripeProvider := providers.CreateStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.CreateXenditProvider(cfg.Xendit.Secret)

	providerSelector := providers.CreateMultiProviderSelector([]providers.PaymentProvider{stripeProvider, xenditProvider}, providerMappingStore)
	printSuccess("Payment providers initialized")
	printInfo("  • Stripe: Ready for USD, EUR, GBP")
	printInfo("  • Xendit: Ready for IDR, SGD, MYR, PHP, THB, VND")

	printStep("8/8", "Initializing services...")
	fraudService := services.CreateFraudService(fraudRepo, cfg.OpenAI.APIKey)
	paymentService := services.CreatePaymentServiceFull(paymentRepo, idempotencyStore, auditStore, providerSelector, fraudService)
	subscriptionService := services.CreateSubscriptionService(planRepo, subscriptionRepo, providerSelector)
	disputeService := services.CreateDisputeService(disputeRepo, providerSelector)
	routingService := services.CreateRoutingService(cfg.OpenAI.APIKey)
	auditService := services.CreateAuditService(auditStore)
	tenantService := services.CreateTenantService(tenantStore)
	webhookService := services.CreateWebhookService(webhookStore, paymentRepo, tenantStore, auditStore)
	invoiceService := services.CreateInvoiceService(providerSelector)
	payoutService := services.CreatePayoutService(providerSelector)
	customerService := services.CreateCustomerService(customerStore, providerSelector)
	paymentMethodService := services.CreatePaymentMethodService(paymentMethodStore, providerSelector)
	balanceService := services.CreateBalanceService(providerSelector)

	printSuccess("Services initialized")

	printStep("8/8", "Setting up HTTP server...")
	paymentHandler := api.CreatePaymentHandlerWithWebhook(paymentService, webhookService)
	subscriptionHandler := api.CreateSubscriptionHandler(subscriptionService)
	disputeHandler := api.CreateDisputeHandler(disputeService)
	fraudHandler := api.CreateFraudHandler(fraudService)
	routingHandler := api.CreateRoutingHandler(routingService)
	tenantHandler := api.CreateTenantHandler(tenantService)
	auditHandler := api.CreateAuditHandler(auditService)
	invoiceHandler := api.CreateInvoiceHandler(invoiceService)
	payoutHandler := api.CreatePayoutHandler(payoutService)
	customerHandler := api.CreateCustomerHandler(customerService)
	paymentMethodHandler := api.CreatePaymentMethodHandler(paymentMethodService)
	balanceHandler := api.CreateBalanceHandler(balanceService)

	router := mux.NewRouter()

	authMiddleware := middleware.CreateAuthMiddleware(jwtManager, rateLimiter, encryption, cfg.Security.WebhookSecret)
	tenantMiddleware := middleware.CreateTenantMiddleware(tenantService, auditService)

	router.Use(middleware.CreateLoggingMiddleware)
	router.Use(authMiddleware.HeadersMiddleware)
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	router.Use(middleware.CreateCORSMiddleware(allowedOrigins))
	router.Use(middleware.CreateRecoveryMiddleware)

	apiRouter := router.PathPrefix("/v1").Subrouter()
	apiRouter.Use(authMiddleware.RateLimitMiddleware)
	apiRouter.Use(authMiddleware.JWTMiddleware)
	apiRouter.Use(tenantMiddleware.TenantContextMiddleware)
	apiRouter.Use(tenantMiddleware.IdempotencyMiddleware)
	apiRouter.Use(tenantMiddleware.AuditMiddleware)
	apiRouter.Use(authMiddleware.EncryptionMiddleware)

	apiRouter.HandleFunc("/health", api.CreateHealthCheckHandler).Methods("GET")

	apiRouter.HandleFunc("/charges", paymentHandler.HandleCharge).Methods("POST")
	apiRouter.HandleFunc("/authorize", paymentHandler.HandleAuthorize).Methods("POST")
	apiRouter.HandleFunc("/payments/{id}", paymentHandler.HandleGetPayment).Methods("GET")
	apiRouter.HandleFunc("/payments/{id}/capture", paymentHandler.HandleCapture).Methods("POST")
	apiRouter.HandleFunc("/payments/{id}/void", paymentHandler.HandleVoid).Methods("POST")
	apiRouter.HandleFunc("/payments/{id}/confirm", paymentHandler.HandleConfirm3DS).Methods("POST")
	apiRouter.HandleFunc("/refunds", paymentHandler.HandleRefund).Methods("POST")

	apiRouter.HandleFunc("/payment-sessions", paymentHandler.HandleCreatePaymentSession).Methods("POST")
	apiRouter.HandleFunc("/payment-sessions", paymentHandler.HandleListPaymentSessions).Methods("GET")
	apiRouter.HandleFunc("/payment-sessions/{id}", paymentHandler.HandleGetPaymentSession).Methods("GET")
	apiRouter.HandleFunc("/payment-sessions/{id}", paymentHandler.HandleUpdatePaymentSession).Methods("PATCH")
	apiRouter.HandleFunc("/payment-sessions/{id}/confirm", paymentHandler.HandleConfirmPaymentSession).Methods("POST")
	apiRouter.HandleFunc("/payment-sessions/{id}/capture", paymentHandler.HandleCapturePaymentSession).Methods("POST")
	apiRouter.HandleFunc("/payment-sessions/{id}/cancel", paymentHandler.HandleCancelPaymentSession).Methods("POST")

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
	apiRouter.HandleFunc("/routing/config", routingHandler.HandleRoutingConfig).Methods("GET", "PUT")

	apiRouter.HandleFunc("/tenants", tenantHandler.HandleCreate).Methods("POST")
	apiRouter.HandleFunc("/tenants", tenantHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/tenants/{id}", tenantHandler.HandleGet).Methods("GET")
	apiRouter.HandleFunc("/tenants/{id}", tenantHandler.HandleUpdate).Methods("PUT")
	apiRouter.HandleFunc("/tenants/{id}", tenantHandler.HandleDelete).Methods("DELETE")
	apiRouter.HandleFunc("/tenants/{id}/deactivate", tenantHandler.HandleDeactivate).Methods("POST")
	apiRouter.HandleFunc("/tenants/{id}/regenerate-secret", tenantHandler.HandleRegenerateSecret).Methods("POST")

	apiRouter.HandleFunc("/audit-logs", auditHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/audit-logs/{resource_type}/{resource_id}", auditHandler.HandleGetResourceHistory).Methods("GET")

	apiRouter.HandleFunc("/invoices", invoiceHandler.HandleCreate).Methods("POST")
	apiRouter.HandleFunc("/invoices", invoiceHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/invoices/{id}", invoiceHandler.HandleGet).Methods("GET")
	apiRouter.HandleFunc("/invoices/{id}/cancel", invoiceHandler.HandleCancel).Methods("POST")

	apiRouter.HandleFunc("/payouts", payoutHandler.HandleCreate).Methods("POST")
	apiRouter.HandleFunc("/payouts", payoutHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/payouts/{id}", payoutHandler.HandleGet).Methods("GET")
	apiRouter.HandleFunc("/payouts/{id}/cancel", payoutHandler.HandleCancel).Methods("POST")
	apiRouter.HandleFunc("/payout-channels", payoutHandler.HandleGetChannels).Methods("GET")

	apiRouter.HandleFunc("/customers", customerHandler.HandleCreate).Methods("POST")
	apiRouter.HandleFunc("/customers/{id}", customerHandler.HandleGet).Methods("GET")
	apiRouter.HandleFunc("/customers/{id}", customerHandler.HandleUpdate).Methods("PUT")
	apiRouter.HandleFunc("/customers/{id}", customerHandler.HandleDelete).Methods("DELETE")

	apiRouter.HandleFunc("/payment-methods", paymentMethodHandler.HandleCreate).Methods("POST")
	apiRouter.HandleFunc("/payment-methods", paymentMethodHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/payment-methods/{id}", paymentMethodHandler.HandleGet).Methods("GET")
	apiRouter.HandleFunc("/payment-methods/{id}/attach", paymentMethodHandler.HandleAttach).Methods("POST")
	apiRouter.HandleFunc("/payment-methods/{id}/detach", paymentMethodHandler.HandleDetach).Methods("POST")
	apiRouter.HandleFunc("/payment-methods/{id}/expire", paymentMethodHandler.HandleExpire).Methods("POST")

	apiRouter.HandleFunc("/balance", balanceHandler.HandleGet).Methods("GET")

	webhookRouter := router.PathPrefix("/v1/webhooks").Subrouter()
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
	fmt.Printf("  %s-%s Health Check: %shttp://localhost:%s/v1/health%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s-%s Payments:     %shttp://localhost:%s/v1/charges%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s-%s Subscriptions: %shttp://localhost:%s/v1/subscriptions%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s-%s Disputes:     %shttp://localhost:%s/v1/disputes%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s-%s Fraud Detection: %shttp://localhost:%s/v1/fraud/analyze%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("  %s-%s AI Routing:     %shttp://localhost:%s/v1/routing/select%s\n", colorCyan, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Println()
	fmt.Printf("%s%sEnvironment:%s %s%s%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Environment, colorReset)
	fmt.Printf("%s%sServer Port:%s %s%s%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Server.Port, colorReset)
	fmt.Printf("%s%sDatabase:%s %s%s:%d%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Database.Host, cfg.Database.Port, colorReset)
	if redisCache != nil {
		fmt.Printf("%s%sRedis:%s %s%s:%d%s\n", colorPurple, colorBold, colorReset, colorYellow, cfg.Redis.Host, cfg.Redis.Port, colorReset)
	}
	fmt.Printf("%s%sSecurity:%s %sJWT + Encryption + Rate Limiting%s\n", colorPurple, colorBold, colorReset, colorYellow, colorReset)
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

	rateLimiter.Close()

	printSuccess("Conductor server stopped gracefully")
	fmt.Println()
	fmt.Printf("%s%sThanks for using Conductor!%s\n", colorCyan, colorBold, colorReset)
}

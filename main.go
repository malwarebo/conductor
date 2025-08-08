package main

import (
	"context"
	"log"
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

func main() {
	log.Println("Starting gopay payment orchestration system...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	db, err := db.NewDB(cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	redisCache, err := cache.NewRedisCache(cache.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		TTL:      time.Duration(cfg.Redis.TTL) * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	stripeProvider := providers.NewStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.NewXenditProvider(cfg.Xendit.Secret)

	providerSelector := providers.NewMultiProviderSelector([]providers.PaymentProvider{stripeProvider, xenditProvider})

	paymentRepo := repositories.NewPaymentRepository(db)
	planRepo := repositories.NewPlanRepository(db)
	subscriptionRepo := repositories.NewSubscriptionRepository(db)
	disputeRepo := repositories.NewDisputeRepository(db.DB)

	paymentService := services.NewPaymentService(paymentRepo, providerSelector)
	subscriptionService := services.NewSubscriptionService(planRepo, subscriptionRepo, providerSelector)
	disputeService := services.NewDisputeService(disputeRepo, providerSelector)

	paymentHandler := api.NewPaymentHandler(paymentService)
	subscriptionHandler := api.NewSubscriptionHandler(subscriptionService)
	disputeHandler := api.NewDisputeHandler(disputeService)

	router := mux.NewRouter()

	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.CORSMiddleware)
	router.Use(middleware.RecoveryMiddleware)

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(middleware.RateLimitMiddleware)

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

	apiRouter.HandleFunc("/webhooks/stripe", paymentHandler.HandleStripeWebhook).Methods("POST")
	apiRouter.HandleFunc("/webhooks/xendit", paymentHandler.HandleXenditWebhook).Methods("POST")

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

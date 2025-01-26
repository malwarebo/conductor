package main

import (
	"log"
	"net/http"

	"github.com/malwarebo/gopay/api"
	"github.com/malwarebo/gopay/config"
	"github.com/malwarebo/gopay/db"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
	"github.com/malwarebo/gopay/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	database, err := db.NewDB(cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize repositories
	planRepo := repositories.NewPlanRepository(database)
	subscriptionRepo := repositories.NewSubscriptionRepository(database)
	disputeRepo := repositories.NewDisputeRepository(database)

	// Initialize payment providers
	stripeProvider := providers.NewStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.NewXenditProvider(cfg.Xendit.Secret)

	// Initialize services
	paymentService := services.NewPaymentService(stripeProvider, xenditProvider)
	subscriptionService := services.NewSubscriptionService(planRepo, subscriptionRepo, stripeProvider, xenditProvider)
	disputeService := services.NewDisputeService(disputeRepo, stripeProvider, xenditProvider)

	// Initialize handlers
	paymentHandler := api.NewPaymentHandler(paymentService)
	subscriptionHandler := api.NewSubscriptionHandler(subscriptionService)
	disputeHandler := api.NewDisputeHandler(disputeService)

	// Setup payment routes
	http.HandleFunc("/charge", paymentHandler.HandleCharge)
	http.HandleFunc("/refund", paymentHandler.HandleRefund)

	// Setup subscription routes
	http.HandleFunc("/plans", subscriptionHandler.HandlePlans)
	http.HandleFunc("/plans/", subscriptionHandler.HandlePlans)
	http.HandleFunc("/subscriptions", subscriptionHandler.HandleSubscriptions)
	http.HandleFunc("/subscriptions/", subscriptionHandler.HandleSubscriptions)

	// Setup dispute routes
	http.HandleFunc("/disputes", disputeHandler.HandleDisputes)
	http.HandleFunc("/disputes/", disputeHandler.HandleDisputes)
	http.HandleFunc("/disputes/stats", disputeHandler.HandleDisputes)

	// Start server
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

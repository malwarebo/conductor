package main

import (
	"log"
	"net/http"

	"github.com/malwarebo/gopay/api"
	"github.com/malwarebo/gopay/config"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize payment providers
	stripeProvider := providers.NewStripeProvider(cfg.Stripe.Secret)
	xenditProvider := providers.NewXenditProvider(cfg.Xendit.Secret)

	// Initialize payment service with both providers
	paymentService := services.NewPaymentService(stripeProvider, xenditProvider)

	// Initialize payment handler
	paymentHandler := api.NewPaymentHandler(paymentService)

	// Setup routes
	http.HandleFunc("/charge", paymentHandler.HandleCharge)
	http.HandleFunc("/refund", paymentHandler.HandleRefund)

	// Start server
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

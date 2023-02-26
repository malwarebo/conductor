package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gopay/config"
	"gopay/payments"
)

func main() {
    cfg := config.NewConfig()
    paymentService := payments.NewPaymentService(cfg.XenditSecretKey)

    http.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        var req payments.CreatePaymentRequest
        err := json.NewDecoder(r.Body).Decode(&req)
        if err != nil {
            http.Error(w, "Bad request", http.StatusBadRequest)
            return
        }

        resp, err := paymentService.CreatePayment(&req)
        if err != nil {
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    })

    fmt.Println("Server listening on port 8080")
    http.ListenAndServe(":8080", nil)
}

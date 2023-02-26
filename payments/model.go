package payments

type CreatePaymentRequest struct {
    ExternalID string  `json:"external_id"`
    Amount     float64 `json:"amount"`
    Email      string  `json:"email"`
}

type CreatePaymentResponse struct {
    ID string `json:"id"`
}

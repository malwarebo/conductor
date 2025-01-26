package stripe

import (
	"github.com/malwarebo/gopay/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

func NewStripeService(apiKey string) *StripeService {
	stripe.Key = apiKey
	return &StripeService{}
}

type StripeService struct{}

func (s *StripeService) CreateCharge(params *models.ChargeRequest) (*models.ChargeResponse, error) {
	intent, err := paymentintent.New(&stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(params.Amount * 100)), // Convert to cents
		Currency: stripe.String(string(params.Currency)),
	})
	if err != nil {
		return nil, err
	}

	return &models.ChargeResponse{
		ID:            intent.ID,
		Amount:        params.Amount,
		Currency:      params.Currency,
		Status:        intent.Status,
		PaymentMethod: params.PaymentMethod,
		ProviderName:  "stripe",
		Metadata:      params.Metadata,
	}, nil
}

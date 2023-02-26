package payments

import (
	"github.com/xendit/xendit-go/client"
	"github.com/xendit/xendit-go/invoice"
)

type PaymentService struct {
    xenditClient *client.XenditClient
}

func NewPaymentService(secretKey string) *PaymentService {
    return &PaymentService{
        xenditClient: &client.XenditClient{
            SecretKey: secretKey,
        },
    }
}

func (s *PaymentService) CreatePayment(req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
    iReq := &invoice.CreateInvoiceParams{
        ExternalID: req.ExternalID,
        Amount:     req.Amount,
        PayerEmail: req.Email,
    }

    i, err := invoice.Create(iReq, s.xenditClient)
    if err != nil {
        return nil, err
    }

    return &CreatePaymentResponse{
        ID: i.ID,
    }, nil
}

package services

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/providers"
)

type InvoiceService struct {
	provider providers.PaymentProvider
}

func CreateInvoiceService(provider providers.PaymentProvider) *InvoiceService {
	return &InvoiceService{
		provider: provider,
	}
}

func (s *InvoiceService) CreateInvoice(ctx context.Context, req *models.CreateInvoiceRequest) (*models.Invoice, error) {
	if invProvider, ok := s.provider.(providers.InvoiceProvider); ok {
		return invProvider.CreateInvoice(ctx, req)
	}
	return nil, providers.ErrNotSupported
}

func (s *InvoiceService) GetInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	if invProvider, ok := s.provider.(providers.InvoiceProvider); ok {
		return invProvider.GetInvoice(ctx, invoiceID)
	}
	return nil, providers.ErrNotSupported
}

func (s *InvoiceService) ListInvoices(ctx context.Context, req *models.ListInvoicesRequest) ([]*models.Invoice, error) {
	if invProvider, ok := s.provider.(providers.InvoiceProvider); ok {
		return invProvider.ListInvoices(ctx, req)
	}
	return nil, providers.ErrNotSupported
}

func (s *InvoiceService) CancelInvoice(ctx context.Context, invoiceID string) (*models.Invoice, error) {
	if invProvider, ok := s.provider.(providers.InvoiceProvider); ok {
		return invProvider.CancelInvoice(ctx, invoiceID)
	}
	return nil, providers.ErrNotSupported
}


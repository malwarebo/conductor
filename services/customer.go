package services

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/providers"
	"github.com/malwarebo/conductor/stores"
)

type CustomerService struct {
	customerStore *stores.CustomerStore
	provider      providers.PaymentProvider
}

func CreateCustomerService(customerStore *stores.CustomerStore, provider providers.PaymentProvider) *CustomerService {
	return &CustomerService{
		customerStore: customerStore,
		provider:      provider,
	}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, req *models.CreateCustomerRequest) (*models.Customer, error) {
	providerID, err := s.provider.CreateCustomer(ctx, req)
	if err != nil {
		return nil, err
	}

	customer := &models.Customer{
		ExternalID: providerID,
		Email:      req.Email,
		Name:       req.Name,
		Phone:      req.Phone,
		Metadata:   req.Metadata,
	}

	if s.customerStore != nil {
		if err := s.customerStore.Create(ctx, customer); err != nil {
			return nil, err
		}
	}

	return customer, nil
}

func (s *CustomerService) GetCustomer(ctx context.Context, customerID string) (*models.Customer, error) {
	return s.provider.GetCustomer(ctx, customerID)
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, customerID string, req *models.UpdateCustomerRequest) error {
	return s.provider.UpdateCustomer(ctx, customerID, req)
}

func (s *CustomerService) DeleteCustomer(ctx context.Context, customerID string) error {
	return s.provider.DeleteCustomer(ctx, customerID)
}


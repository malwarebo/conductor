package services

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/providers"
)

type BalanceService struct {
	provider providers.PaymentProvider
}

func CreateBalanceService(provider providers.PaymentProvider) *BalanceService {
	return &BalanceService{
		provider: provider,
	}
}

func (s *BalanceService) GetBalance(ctx context.Context, currency string) (*models.Balance, error) {
	if balanceProvider, ok := s.provider.(providers.BalanceProvider); ok {
		return balanceProvider.GetBalance(ctx, currency)
	}
	return nil, providers.ErrNotSupported
}

func (s *BalanceService) GetAllBalances(ctx context.Context) ([]*models.Balance, error) {
	currencies := []string{"USD", "EUR", "GBP", "IDR", "SGD", "PHP"}
	var balances []*models.Balance

	if balanceProvider, ok := s.provider.(providers.BalanceProvider); ok {
		for _, currency := range currencies {
			balance, err := balanceProvider.GetBalance(ctx, currency)
			if err == nil && balance != nil {
				balances = append(balances, balance)
			}
		}
	}

	return balances, nil
}


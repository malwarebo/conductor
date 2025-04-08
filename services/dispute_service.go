package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
)

var (
	ErrDisputeNotFound = errors.New("dispute not found")
	ErrInvalidStatus   = errors.New("invalid status")
)

type DisputeService struct {
	disputeRepo *repositories.DisputeRepository
	provider    providers.PaymentProvider
}

func NewDisputeService(disputeRepo *repositories.DisputeRepository, provider providers.PaymentProvider) *DisputeService {
	return &DisputeService{
		disputeRepo: disputeRepo,
		provider:    provider,
	}
}

func (s *DisputeService) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.DisputeResponse, error) {
	dispute := &models.Dispute{
		CustomerID:    req.CustomerID,
		TransactionID: req.TransactionID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Reason:        req.Reason,
		Status:        models.DisputeStatusOpen,
		Evidence:      req.Evidence,
		DueBy:         req.DueBy,
		Metadata:      req.Metadata,
	}

	if err := s.disputeRepo.Create(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to create dispute: %w", err)
	}

	return &models.DisputeResponse{Dispute: dispute}, nil
}

func (s *DisputeService) GetDispute(ctx context.Context, id string) (*models.DisputeResponse, error) {
	dispute, err := s.disputeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute: %w", err)
	}
	return &models.DisputeResponse{Dispute: dispute}, nil
}

func (s *DisputeService) ListDisputes(ctx context.Context, customerID string) ([]models.Dispute, error) {
	disputes, err := s.disputeRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list disputes: %w", err)
	}
	return disputes, nil
}

func (s *DisputeService) UpdateDispute(ctx context.Context, id string, req *models.UpdateDisputeRequest) (*models.DisputeResponse, error) {
	dispute, err := s.disputeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute: %w", err)
	}

	if req.Status != "" {
		switch req.Status {
		case models.DisputeStatusOpen, models.DisputeStatusWon, models.DisputeStatusLost, models.DisputeStatusCanceled:
			dispute.Status = req.Status
		default:
			return nil, ErrInvalidStatus
		}
	}

	if req.Metadata != nil {
		dispute.Metadata = req.Metadata
	}

	if err := s.disputeRepo.Update(ctx, dispute); err != nil {
		return nil, fmt.Errorf("failed to update dispute: %w", err)
	}

	return &models.DisputeResponse{Dispute: dispute}, nil
}

func (s *DisputeService) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	stats, err := s.disputeRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute stats: %w", err)
	}
	return stats, nil
}

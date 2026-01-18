package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/malwarebo/conductor/models"
	"github.com/malwarebo/conductor/providers"
	"github.com/malwarebo/conductor/stores"
)

var (
	ErrDisputeNotFound = errors.New("dispute not found")
	ErrInvalidStatus   = errors.New("invalid status")
)

type DisputeService struct {
	disputeRepo *stores.DisputeRepository
	provider    providers.PaymentProvider
}

func CreateDisputeService(disputeRepo *stores.DisputeRepository, provider providers.PaymentProvider) *DisputeService {
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
	if s.provider != nil {
		providerDispute, err := s.provider.GetDispute(ctx, id)
		if err == nil && providerDispute != nil {
			return &models.DisputeResponse{Dispute: providerDispute}, nil
		}
	}

	dispute, err := s.disputeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrDisputeNotFound
	}
	return &models.DisputeResponse{Dispute: dispute}, nil
}

func (s *DisputeService) ListDisputes(ctx context.Context, customerID string) ([]models.Dispute, error) {
	if s.provider != nil {
		providerDisputes, err := s.provider.ListDisputes(ctx, customerID)
		if err == nil && len(providerDisputes) > 0 {
			result := make([]models.Dispute, len(providerDisputes))
			for i, d := range providerDisputes {
				result[i] = *d
			}
			return result, nil
		}
	}

	disputes, err := s.disputeRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list disputes: %w", err)
	}
	return disputes, nil
}

func (s *DisputeService) UpdateDispute(ctx context.Context, id string, req *models.UpdateDisputeRequest) (*models.DisputeResponse, error) {
	if s.provider != nil {
		providerDispute, err := s.provider.UpdateDispute(ctx, id, req)
		if err == nil && providerDispute != nil {
			return &models.DisputeResponse{Dispute: providerDispute}, nil
		}
	}

	dispute, err := s.disputeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrDisputeNotFound
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

func (s *DisputeService) AcceptDispute(ctx context.Context, id string) (*models.DisputeResponse, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	dispute, err := s.provider.AcceptDispute(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to accept dispute: %w", err)
	}

	return &models.DisputeResponse{Dispute: dispute}, nil
}

func (s *DisputeService) ContestDispute(ctx context.Context, id string, evidence map[string]interface{}) (*models.DisputeResponse, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	dispute, err := s.provider.ContestDispute(ctx, id, evidence)
	if err != nil {
		return nil, fmt.Errorf("failed to contest dispute: %w", err)
	}

	return &models.DisputeResponse{Dispute: dispute}, nil
}

func (s *DisputeService) SubmitEvidence(ctx context.Context, id string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	evidence, err := s.provider.SubmitDisputeEvidence(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit evidence: %w", err)
	}

	return evidence, nil
}

func (s *DisputeService) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	if s.provider != nil {
		providerStats, err := s.provider.GetDisputeStats(ctx)
		if err == nil && providerStats != nil {
			return providerStats, nil
		}
	}

	stats, err := s.disputeRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute stats: %w", err)
	}
	return stats, nil
}

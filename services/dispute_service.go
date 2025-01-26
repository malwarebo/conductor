package services

import (
	"context"
	"errors"
	"sync"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
)

var (
	ErrDisputeNotFound = errors.New("dispute not found")
	ErrInvalidEvidence = errors.New("invalid evidence")
)

type DisputeService struct {
	providers []providers.PaymentProvider
	mu        sync.RWMutex
}

func NewDisputeService(providers ...providers.PaymentProvider) *DisputeService {
	return &DisputeService{
		providers: providers,
	}
}

func (s *DisputeService) AddProvider(provider providers.PaymentProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = append(s.providers, provider)
}

func (s *DisputeService) getAvailableProvider(ctx context.Context) providers.PaymentProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, provider := range s.providers {
		if provider.IsAvailable(ctx) {
			return provider
		}
	}
	return nil
}

// Dispute Management
func (s *DisputeService) CreateDispute(ctx context.Context, req *models.CreateDisputeRequest) (*models.Dispute, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate request
	if req.TransactionID == "" {
		return nil, errors.New("transaction ID is required")
	}
	if req.Reason == "" {
		return nil, errors.New("dispute reason is required")
	}

	return provider.CreateDispute(ctx, req)
}

func (s *DisputeService) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate dispute exists
	dispute, err := provider.GetDispute(ctx, disputeID)
	if err != nil {
		return nil, ErrDisputeNotFound
	}

	// Validate status transition
	if req.Status != "" && !isValidStatusTransition(dispute.Status, req.Status) {
		return nil, errors.New("invalid status transition")
	}

	return provider.UpdateDispute(ctx, disputeID, req)
}

func (s *DisputeService) SubmitEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate dispute exists and is in a valid state for evidence submission
	dispute, err := provider.GetDispute(ctx, disputeID)
	if err != nil {
		return nil, ErrDisputeNotFound
	}

	if !canSubmitEvidence(dispute.Status) {
		return nil, errors.New("dispute is not in a valid state for evidence submission")
	}

	// Validate evidence
	if err := validateEvidence(req); err != nil {
		return nil, err
	}

	return provider.SubmitDisputeEvidence(ctx, disputeID, req)
}

func (s *DisputeService) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.GetDispute(ctx, disputeID)
}

func (s *DisputeService) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.ListDisputes(ctx, customerID)
}

func (s *DisputeService) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}
	return provider.GetDisputeStats(ctx)
}

// Helper functions
func isValidStatusTransition(current, new models.DisputeStatus) bool {
	transitions := map[models.DisputeStatus][]models.DisputeStatus{
		models.DisputeStatusOpen: {
			models.DisputeStatusUnderReview,
			models.DisputeStatusCanceled,
		},
		models.DisputeStatusUnderReview: {
			models.DisputeStatusWon,
			models.DisputeStatusLost,
		},
	}

	allowedTransitions, exists := transitions[current]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if new == allowed {
			return true
		}
	}
	return false
}

func canSubmitEvidence(status models.DisputeStatus) bool {
	return status == models.DisputeStatusOpen || status == models.DisputeStatusUnderReview
}

func validateEvidence(req *models.SubmitEvidenceRequest) error {
	if req.Type == "" {
		return errors.New("evidence type is required")
	}
	if req.Description == "" {
		return errors.New("evidence description is required")
	}
	return nil
}

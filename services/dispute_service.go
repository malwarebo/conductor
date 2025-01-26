package services

import (
	"context"
	"errors"
	"sync"

	"github.com/malwarebo/gopay/models"
	"github.com/malwarebo/gopay/providers"
	"github.com/malwarebo/gopay/repositories"
)

var (
	ErrDisputeNotFound = errors.New("dispute not found")
	ErrInvalidEvidence = errors.New("invalid evidence")
	ErrNoAvailableProvider = errors.New("no available provider")
)

type DisputeService struct {
	providers     []providers.PaymentProvider
	disputeRepo   *repositories.DisputeRepository
	mu           sync.RWMutex
}

func NewDisputeService(disputeRepo *repositories.DisputeRepository, providers ...providers.PaymentProvider) *DisputeService {
	return &DisputeService{
		providers:   providers,
		disputeRepo: disputeRepo,
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

	// Create dispute in payment provider
	dispute, err := provider.CreateDispute(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store dispute in database
	if err := s.disputeRepo.Create(ctx, dispute); err != nil {
		return nil, err
	}

	return dispute, nil
}

func (s *DisputeService) UpdateDispute(ctx context.Context, disputeID string, req *models.UpdateDisputeRequest) (*models.Dispute, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate dispute exists
	dispute, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, ErrDisputeNotFound
	}

	// Validate status transition
	if req.Status != "" && !isValidStatusTransition(dispute.Status, req.Status) {
		return nil, errors.New("invalid status transition")
	}

	// Update dispute in payment provider
	updatedDispute, err := provider.UpdateDispute(ctx, disputeID, req)
	if err != nil {
		return nil, err
	}

	// Update dispute in database
	if err := s.disputeRepo.Update(ctx, updatedDispute); err != nil {
		return nil, err
	}

	return updatedDispute, nil
}

func (s *DisputeService) SubmitEvidence(ctx context.Context, disputeID string, req *models.SubmitEvidenceRequest) (*models.Evidence, error) {
	provider := s.getAvailableProvider(ctx)
	if provider == nil {
		return nil, ErrNoAvailableProvider
	}

	// Validate dispute exists and is in a valid state for evidence submission
	dispute, err := s.disputeRepo.GetByID(ctx, disputeID)
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

	// Submit evidence to payment provider
	evidence, err := provider.SubmitDisputeEvidence(ctx, disputeID, req)
	if err != nil {
		return nil, err
	}

	// Store evidence in database
	if err := s.disputeRepo.AddEvidence(ctx, evidence); err != nil {
		return nil, err
	}

	return evidence, nil
}

func (s *DisputeService) GetDispute(ctx context.Context, disputeID string) (*models.Dispute, error) {
	// Get dispute from database
	dispute, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}

	// Get evidence for the dispute
	evidence, err := s.disputeRepo.GetEvidence(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	dispute.Evidence = evidence

	return dispute, nil
}

func (s *DisputeService) ListDisputes(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	// Get disputes from database
	disputes, err := s.disputeRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// Get evidence for each dispute
	for _, dispute := range disputes {
		evidence, err := s.disputeRepo.GetEvidence(ctx, dispute.ID)
		if err != nil {
			return nil, err
		}
		dispute.Evidence = evidence
	}

	return disputes, nil
}

func (s *DisputeService) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	return s.disputeRepo.GetStats(ctx)
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

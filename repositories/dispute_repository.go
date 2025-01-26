package repositories

import (
	"context"

	"github.com/malwarebo/gopay/db"
	"github.com/malwarebo/gopay/models"
)

type DisputeRepository struct {
	db *db.DB
}

func NewDisputeRepository(db *db.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

func (r *DisputeRepository) Create(ctx context.Context, dispute *models.Dispute) error {
	return r.db.WithContext(ctx).Create(dispute).Error
}

func (r *DisputeRepository) Update(ctx context.Context, dispute *models.Dispute) error {
	return r.db.WithContext(ctx).Save(dispute).Error
}

func (r *DisputeRepository) GetByID(ctx context.Context, id string) (*models.Dispute, error) {
	var dispute models.Dispute
	if err := r.db.WithContext(ctx).Preload("Evidence").First(&dispute, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &dispute, nil
}

func (r *DisputeRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	var disputes []*models.Dispute
	if err := r.db.WithContext(ctx).Preload("Evidence").Where("customer_id = ?", customerID).Find(&disputes).Error; err != nil {
		return nil, err
	}
	return disputes, nil
}

func (r *DisputeRepository) AddEvidence(ctx context.Context, evidence *models.Evidence) error {
	return r.db.WithContext(ctx).Create(evidence).Error
}

func (r *DisputeRepository) GetEvidence(ctx context.Context, disputeID string) ([]models.Evidence, error) {
	var evidence []models.Evidence
	if err := r.db.WithContext(ctx).Where("dispute_id = ?", disputeID).Find(&evidence).Error; err != nil {
		return nil, err
	}
	return evidence, nil
}

func (r *DisputeRepository) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	var stats models.DisputeStats
	
	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Dispute{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// Get counts by status
	if err := r.db.WithContext(ctx).Model(&models.Dispute{}).Where("status = ?", models.DisputeStatusOpen).Count(&stats.Open).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Dispute{}).Where("status = ?", models.DisputeStatusUnderReview).Count(&stats.UnderReview).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Dispute{}).Where("status = ?", models.DisputeStatusWon).Count(&stats.Won).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Dispute{}).Where("status = ?", models.DisputeStatusLost).Count(&stats.Lost).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Dispute{}).Where("status = ?", models.DisputeStatusCanceled).Count(&stats.Canceled).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

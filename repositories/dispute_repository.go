package repositories

import (
	"context"

	"github.com/malwarebo/gopay/models"
	"gorm.io/gorm"
)

type DisputeRepository struct {
	db *gorm.DB
}

func CreateDisputeRepository(db *gorm.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

func (r *DisputeRepository) Create(ctx context.Context, dispute *models.Dispute) error {
	return r.db.WithContext(ctx).Create(dispute).Error
}

func (r *DisputeRepository) GetByID(ctx context.Context, id string) (*models.Dispute, error) {
	var dispute models.Dispute
	err := r.db.WithContext(ctx).First(&dispute, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &dispute, nil
}

func (r *DisputeRepository) Update(ctx context.Context, dispute *models.Dispute) error {
	return r.db.WithContext(ctx).Save(dispute).Error
}

func (r *DisputeRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Dispute{}, "id = ?", id).Error
}

func (r *DisputeRepository) ListByCustomer(ctx context.Context, customerID string) ([]models.Dispute, error) {
	var disputes []models.Dispute
	query := r.db.WithContext(ctx)
	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	err := query.Find(&disputes).Error
	return disputes, err
}

func (r *DisputeRepository) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	var stats models.DisputeStats
	err := r.db.WithContext(ctx).Model(&models.Dispute{}).
		Select(`
			COUNT(*) as total,
			COUNT(CASE WHEN status = ? THEN 1 END) as open,
			COUNT(CASE WHEN status = ? THEN 1 END) as won,
			COUNT(CASE WHEN status = ? THEN 1 END) as lost,
			COUNT(CASE WHEN status = ? THEN 1 END) as canceled
		`, models.DisputeStatusOpen, models.DisputeStatusWon, models.DisputeStatusLost, models.DisputeStatusCanceled).
		Scan(&stats).Error
	return &stats, err
}

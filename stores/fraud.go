package stores

import (
	"time"

	"github.com/malwarebo/gopay/models"
	"gorm.io/gorm"
)

type FraudRepository interface {
	SaveAnalysisResult(result *models.FraudAnalysisResult) error
	GetStatsByDateRange(startDate, endDate time.Time) (*models.FraudStatsResponse, error)
}

type fraudRepository struct {
	db *gorm.DB
}

func CreateFraudRepository(db *gorm.DB) FraudRepository {
	return &fraudRepository{
		db: db,
	}
}

func (r *fraudRepository) SaveAnalysisResult(result *models.FraudAnalysisResult) error {
	return r.db.Create(result).Error
}

func (r *fraudRepository) GetStatsByDateRange(startDate, endDate time.Time) (*models.FraudStatsResponse, error) {
	var stats struct {
		TotalTransactions           int64
		TotalFraudulentTransactions int64
		AverageFraudScore           float64
	}

	err := r.db.Model(&models.FraudAnalysisResult{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Count(&stats.TotalTransactions).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.FraudAnalysisResult{}).
		Where("created_at >= ? AND created_at <= ? AND is_fraudulent = ?", startDate, endDate, true).
		Count(&stats.TotalFraudulentTransactions).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&models.FraudAnalysisResult{}).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Select("COALESCE(AVG(fraud_score), 0)").
		Scan(&stats.AverageFraudScore).Error
	if err != nil {
		return nil, err
	}

	var fraudulentPercentage float64
	if stats.TotalTransactions > 0 {
		fraudulentPercentage = (float64(stats.TotalFraudulentTransactions) / float64(stats.TotalTransactions)) * 100
	}

	return &models.FraudStatsResponse{
		TotalTransactions:               int(stats.TotalTransactions),
		TotalFraudulentTransactions:     int(stats.TotalFraudulentTransactions),
		AverageFraudScore:               stats.AverageFraudScore,
		FraudulentTransactionPercentage: fraudulentPercentage,
	}, nil
}

package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type PlanRepository struct {
	BaseStore
}

func CreatePlanRepository(db *gorm.DB) *PlanRepository {
	return &PlanRepository{BaseStore: BaseStore{db: db}}
}

func (r *PlanRepository) Create(ctx context.Context, plan *models.Plan) error {
	return r.GetDB(ctx).Create(plan).Error
}

func (r *PlanRepository) Update(ctx context.Context, plan *models.Plan) error {
	return r.GetDB(ctx).Save(plan).Error
}

func (r *PlanRepository) GetByID(ctx context.Context, id string) (*models.Plan, error) {
	var plan models.Plan
	if err := r.GetDB(ctx).First(&plan, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *PlanRepository) List(ctx context.Context) ([]*models.Plan, error) {
	var plans []*models.Plan
	if err := r.GetDB(ctx).Where("active = ?", true).Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func (r *PlanRepository) Delete(ctx context.Context, id string) error {
	return r.GetDB(ctx).Model(&models.Plan{}).Where("id = ?", id).Update("active", false).Error
}

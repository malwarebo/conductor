package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/malwarebo/gopay/db"
	"github.com/malwarebo/gopay/models"
)

type PlanRepository struct {
	db *db.DB
}

func NewPlanRepository(db *db.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

func (r *PlanRepository) Create(ctx context.Context, plan *models.Plan) error {
	metadata, err := json.Marshal(plan.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO plans (name, amount, currency, interval, trial_days, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		plan.Name,
		plan.Amount,
		plan.Currency,
		plan.Interval,
		plan.TrialDays,
		metadata,
	).Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt)
}

func (r *PlanRepository) Update(ctx context.Context, plan *models.Plan) error {
	metadata, err := json.Marshal(plan.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE plans
		SET name = $1, amount = $2, currency = $3, interval = $4, trial_days = $5, metadata = $6
		WHERE id = $7
		RETURNING updated_at`

	return r.db.QueryRowContext(ctx, query,
		plan.Name,
		plan.Amount,
		plan.Currency,
		plan.Interval,
		plan.TrialDays,
		metadata,
		plan.ID,
	).Scan(&plan.UpdatedAt)
}

func (r *PlanRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM plans WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PlanRepository) GetByID(ctx context.Context, id string) (*models.Plan, error) {
	plan := &models.Plan{}
	var metadata []byte

	query := `
		SELECT id, name, amount, currency, interval, trial_days, metadata, created_at, updated_at
		FROM plans
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&plan.ID,
		&plan.Name,
		&plan.Amount,
		&plan.Currency,
		&plan.Interval,
		&plan.TrialDays,
		&metadata,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &plan.Metadata); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

func (r *PlanRepository) List(ctx context.Context) ([]*models.Plan, error) {
	query := `
		SELECT id, name, amount, currency, interval, trial_days, metadata, created_at, updated_at
		FROM plans
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*models.Plan
	for rows.Next() {
		plan := &models.Plan{}
		var metadata []byte

		err := rows.Scan(
			&plan.ID,
			&plan.Name,
			&plan.Amount,
			&plan.Currency,
			&plan.Interval,
			&plan.TrialDays,
			&metadata,
			&plan.CreatedAt,
			&plan.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &plan.Metadata); err != nil {
				return nil, err
			}
		}

		plans = append(plans, plan)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return plans, nil
}

package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/malwarebo/gopay/db"
	"github.com/malwarebo/gopay/models"
)

type SubscriptionRepository struct {
	db *db.DB
}

func NewSubscriptionRepository(db *db.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, sub *models.Subscription) error {
	metadata, err := json.Marshal(sub.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO subscriptions (
			customer_id, plan_id, status, current_period_start,
			current_period_end, trial_end, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		sub.CustomerID,
		sub.PlanID,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.TrialEnd,
		metadata,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

func (r *SubscriptionRepository) Update(ctx context.Context, sub *models.Subscription) error {
	metadata, err := json.Marshal(sub.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE subscriptions
		SET customer_id = $1, plan_id = $2, status = $3,
			current_period_start = $4, current_period_end = $5,
			trial_end = $6, canceled_at = $7, metadata = $8
		WHERE id = $9
		RETURNING updated_at`

	return r.db.QueryRowContext(ctx, query,
		sub.CustomerID,
		sub.PlanID,
		sub.Status,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.TrialEnd,
		sub.CanceledAt,
		metadata,
		sub.ID,
	).Scan(&sub.UpdatedAt)
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id string) (*models.Subscription, error) {
	sub := &models.Subscription{}
	var metadata []byte

	query := `
		SELECT id, customer_id, plan_id, status, current_period_start,
			current_period_end, trial_end, canceled_at, metadata,
			created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sub.ID,
		&sub.CustomerID,
		&sub.PlanID,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.TrialEnd,
		&sub.CanceledAt,
		&metadata,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &sub.Metadata); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

func (r *SubscriptionRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Subscription, error) {
	query := `
		SELECT id, customer_id, plan_id, status, current_period_start,
			current_period_end, trial_end, canceled_at, metadata,
			created_at, updated_at
		FROM subscriptions
		WHERE customer_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.Subscription
	for rows.Next() {
		sub := &models.Subscription{}
		var metadata []byte

		err := rows.Scan(
			&sub.ID,
			&sub.CustomerID,
			&sub.PlanID,
			&sub.Status,
			&sub.CurrentPeriodStart,
			&sub.CurrentPeriodEnd,
			&sub.TrialEnd,
			&sub.CanceledAt,
			&metadata,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &sub.Metadata); err != nil {
				return nil, err
			}
		}

		subs = append(subs, sub)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

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
	metadata, err := json.Marshal(dispute.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO disputes (
			payment_id, customer_id, amount, currency, status,
			reason, provider, provider_dispute_id, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		dispute.PaymentID,
		dispute.CustomerID,
		dispute.Amount,
		dispute.Currency,
		dispute.Status,
		dispute.Reason,
		dispute.Provider,
		dispute.ProviderDisputeID,
		metadata,
	).Scan(&dispute.ID, &dispute.CreatedAt, &dispute.UpdatedAt)
}

func (r *DisputeRepository) Update(ctx context.Context, dispute *models.Dispute) error {
	metadata, err := json.Marshal(dispute.Metadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE disputes
		SET payment_id = $1, customer_id = $2, amount = $3,
			currency = $4, status = $5, reason = $6,
			provider = $7, provider_dispute_id = $8, metadata = $9
		WHERE id = $10
		RETURNING updated_at`

	return r.db.QueryRowContext(ctx, query,
		dispute.PaymentID,
		dispute.CustomerID,
		dispute.Amount,
		dispute.Currency,
		dispute.Status,
		dispute.Reason,
		dispute.Provider,
		dispute.ProviderDisputeID,
		metadata,
		dispute.ID,
	).Scan(&dispute.UpdatedAt)
}

func (r *DisputeRepository) GetByID(ctx context.Context, id string) (*models.Dispute, error) {
	dispute := &models.Dispute{}
	var metadata []byte

	query := `
		SELECT id, payment_id, customer_id, amount, currency,
			status, reason, provider, provider_dispute_id,
			metadata, created_at, updated_at
		FROM disputes
		WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&dispute.ID,
		&dispute.PaymentID,
		&dispute.CustomerID,
		&dispute.Amount,
		&dispute.Currency,
		&dispute.Status,
		&dispute.Reason,
		&dispute.Provider,
		&dispute.ProviderDisputeID,
		&metadata,
		&dispute.CreatedAt,
		&dispute.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &dispute.Metadata); err != nil {
			return nil, err
		}
	}

	return dispute, nil
}

func (r *DisputeRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Dispute, error) {
	query := `
		SELECT id, payment_id, customer_id, amount, currency,
			status, reason, provider, provider_dispute_id,
			metadata, created_at, updated_at
		FROM disputes
		WHERE customer_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disputes []*models.Dispute
	for rows.Next() {
		dispute := &models.Dispute{}
		var metadata []byte

		err := rows.Scan(
			&dispute.ID,
			&dispute.PaymentID,
			&dispute.CustomerID,
			&dispute.Amount,
			&dispute.Currency,
			&dispute.Status,
			&dispute.Reason,
			&dispute.Provider,
			&dispute.ProviderDisputeID,
			&metadata,
			&dispute.CreatedAt,
			&dispute.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &dispute.Metadata); err != nil {
				return nil, err
			}
		}

		disputes = append(disputes, dispute)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return disputes, nil
}

func (r *DisputeRepository) GetStats(ctx context.Context) (*models.DisputeStats, error) {
	stats := &models.DisputeStats{}

	query := `
		SELECT 
			COUNT(*) as total_disputes,
			COUNT(CASE WHEN status = 'open' THEN 1 END) as open_disputes,
			COUNT(CASE WHEN status = 'won' THEN 1 END) as won_disputes,
			COUNT(CASE WHEN status = 'lost' THEN 1 END) as lost_disputes,
			COALESCE(SUM(CASE WHEN status = 'open' THEN amount ELSE 0 END), 0) as amount_at_risk
		FROM disputes`

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalDisputes,
		&stats.OpenDisputes,
		&stats.WonDisputes,
		&stats.LostDisputes,
		&stats.AmountAtRisk,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (r *DisputeRepository) AddEvidence(ctx context.Context, evidence *models.Evidence) error {
	query := `
		INSERT INTO dispute_evidence (
			dispute_id, evidence_type, file_url, description
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		evidence.DisputeID,
		evidence.Type,
		evidence.FileURL,
		evidence.Description,
	).Scan(&evidence.ID, &evidence.CreatedAt, &evidence.UpdatedAt)
}

func (r *DisputeRepository) GetEvidence(ctx context.Context, disputeID string) ([]*models.Evidence, error) {
	query := `
		SELECT id, dispute_id, evidence_type, file_url,
			description, created_at, updated_at
		FROM dispute_evidence
		WHERE dispute_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, disputeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evidences []*models.Evidence
	for rows.Next() {
		evidence := &models.Evidence{}

		err := rows.Scan(
			&evidence.ID,
			&evidence.DisputeID,
			&evidence.Type,
			&evidence.FileURL,
			&evidence.Description,
			&evidence.CreatedAt,
			&evidence.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		evidences = append(evidences, evidence)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return evidences, nil
}

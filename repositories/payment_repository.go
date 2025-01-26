package repositories

import (
	"context"

	"github.com/malwarebo/gopay/db"
	"github.com/malwarebo/gopay/models"
)

type PaymentRepository struct {
	db *db.DB
}

func NewPaymentRepository(db *db.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *PaymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.WithContext(ctx).Preload("Refunds").First(&payment, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Payment, error) {
	var payments []*models.Payment
	if err := r.db.WithContext(ctx).Preload("Refunds").Where("customer_id = ?", customerID).Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) CreateRefund(ctx context.Context, refund *models.Refund) error {
	return r.db.WithContext(ctx).Create(refund).Error
}

func (r *PaymentRepository) GetRefundByID(ctx context.Context, id string) (*models.Refund, error) {
	var refund models.Refund
	if err := r.db.WithContext(ctx).First(&refund, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *PaymentRepository) ListRefundsByPayment(ctx context.Context, paymentID string) ([]*models.Refund, error) {
	var refunds []*models.Refund
	if err := r.db.WithContext(ctx).Where("payment_id = ?", paymentID).Find(&refunds).Error; err != nil {
		return nil, err
	}
	return refunds, nil
}

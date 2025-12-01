package stores

import (
	"context"

	"github.com/malwarebo/conductor/models"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	BaseStore
}

func CreatePaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{BaseStore: BaseStore{db: db}}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	return r.GetDB(ctx).Create(payment).Error
}

func (r *PaymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	return r.GetDB(ctx).Save(payment).Error
}

func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
	var payment models.Payment
	if err := r.GetDB(ctx).Preload("Refunds").First(&payment, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Payment, error) {
	var payments []*models.Payment
	if err := r.GetDB(ctx).Preload("Refunds").Where("customer_id = ?", customerID).Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) CreateRefund(ctx context.Context, refund *models.Refund) error {
	return r.GetDB(ctx).Create(refund).Error
}

func (r *PaymentRepository) GetRefundByID(ctx context.Context, id string) (*models.Refund, error) {
	var refund models.Refund
	if err := r.GetDB(ctx).First(&refund, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *PaymentRepository) ListRefundsByPayment(ctx context.Context, paymentID string) ([]*models.Refund, error) {
	var refunds []*models.Refund
	if err := r.GetDB(ctx).Where("payment_id = ?", paymentID).Find(&refunds).Error; err != nil {
		return nil, err
	}
	return refunds, nil
}

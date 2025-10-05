package repositories

import (
	"context"

	"github.com/malwarebo/gopay/models"
	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	db := r.getDB(ctx)
	return db.Create(payment).Error
}

func (r *PaymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	db := r.getDB(ctx)
	return db.Save(payment).Error
}

func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
	var payment models.Payment
	db := r.getDB(ctx)
	if err := db.Preload("Refunds").First(&payment, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (r *PaymentRepository) ListByCustomer(ctx context.Context, customerID string) ([]*models.Payment, error) {
	var payments []*models.Payment
	db := r.getDB(ctx)
	if err := db.Preload("Refunds").Where("customer_id = ?", customerID).Find(&payments).Error; err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *PaymentRepository) CreateRefund(ctx context.Context, refund *models.Refund) error {
	db := r.getDB(ctx)
	return db.Create(refund).Error
}

func (r *PaymentRepository) GetRefundByID(ctx context.Context, id string) (*models.Refund, error) {
	var refund models.Refund
	db := r.getDB(ctx)
	if err := db.First(&refund, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &refund, nil
}

func (r *PaymentRepository) ListRefundsByPayment(ctx context.Context, paymentID string) ([]*models.Refund, error) {
	var refunds []*models.Refund
	db := r.getDB(ctx)
	if err := db.Where("payment_id = ?", paymentID).Find(&refunds).Error; err != nil {
		return nil, err
	}
	return refunds, nil
}

func (r *PaymentRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey, tx)
		return fn(txCtx)
	})
}

type contextKey string

const txKey contextKey = "tx"

func (r *PaymentRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return r.db.WithContext(ctx)
}

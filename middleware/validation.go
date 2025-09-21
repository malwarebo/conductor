package middleware

import (
	"net/http"
	"strconv"

	"github.com/malwarebo/gopay/utils"
)

func RequestSizeLimitMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := utils.ValidateRequestSize(r, maxSize); err != nil {
				utils.WriteValidationError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ValidateChargeRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	if err := utils.ValidateJSONRequest(w, r, 1024*1024); err != nil {
		return err
	}

	var validationErrors utils.ValidationErrors

	if chargeReq, ok := req.(*struct {
		CustomerID    string `json:"customer_id"`
		Amount        int64  `json:"amount"`
		Currency      string `json:"currency"`
		PaymentMethod string `json:"payment_method"`
		Description   string `json:"description"`
	}); ok {
		if err := utils.ValidateString(chargeReq.CustomerID, "customer_id", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateAmount(chargeReq.Amount, "amount"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateCurrency(chargeReq.Currency, "currency"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateString(chargeReq.PaymentMethod, "payment_method", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateString(chargeReq.Description, "description", 0, 500, false); err != nil {
			validationErrors = append(validationErrors, *err)
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func ValidateRefundRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	if err := utils.ValidateJSONRequest(w, r, 1024*1024); err != nil {
		return err
	}

	var validationErrors utils.ValidationErrors

	if refundReq, ok := req.(*struct {
		PaymentID string `json:"payment_id"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
		Reason    string `json:"reason"`
	}); ok {
		if err := utils.ValidateUUID(refundReq.PaymentID, "payment_id"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateAmount(refundReq.Amount, "amount"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateCurrency(refundReq.Currency, "currency"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateString(refundReq.Reason, "reason", 0, 500, false); err != nil {
			validationErrors = append(validationErrors, *err)
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func ValidateFraudRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	if err := utils.ValidateJSONRequest(w, r, 1024*1024); err != nil {
		return err
	}

	var validationErrors utils.ValidationErrors

	if fraudReq, ok := req.(*struct {
		TransactionID       string  `json:"transaction_id"`
		UserID              string  `json:"user_id"`
		TransactionAmount   float64 `json:"transaction_amount"`
		BillingCountry      string  `json:"billing_country"`
		ShippingCountry     string  `json:"shipping_country"`
		IPAddress           string  `json:"ip_address"`
		TransactionVelocity int     `json:"transaction_velocity"`
	}); ok {
		if err := utils.ValidateString(fraudReq.TransactionID, "transaction_id", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateString(fraudReq.UserID, "user_id", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if fraudReq.TransactionAmount <= 0 {
			validationErrors = append(validationErrors, utils.ValidationError{
				Field:   "transaction_amount",
				Message: "must be greater than 0",
			})
		}
		if err := utils.ValidateCountryCode(fraudReq.BillingCountry, "billing_country"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateCountryCode(fraudReq.ShippingCountry, "shipping_country"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.ValidateIPAddress(fraudReq.IPAddress, "ip_address"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if fraudReq.TransactionVelocity < 0 {
			validationErrors = append(validationErrors, utils.ValidationError{
				Field:   "transaction_velocity",
				Message: "must be non-negative",
			})
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func ParseIntParam(r *http.Request, param string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}

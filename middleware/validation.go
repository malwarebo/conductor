package middleware

import (
	"net/http"
	"strconv"

	"github.com/malwarebo/conductor/utils"
)

func CreateRequestSizeLimitMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := utils.CreateValidateRequestSize(r, maxSize); err != nil {
				utils.CreateWriteValidationError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CreateValidateChargeRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	if err := utils.CreateValidateJSONRequest(w, r, 1024*1024); err != nil {
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
		if err := utils.CreateValidateString(chargeReq.CustomerID, "customer_id", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateAmount(chargeReq.Amount, "amount"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateCurrency(chargeReq.Currency, "currency"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateString(chargeReq.PaymentMethod, "payment_method", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateString(chargeReq.Description, "description", 0, 500, false); err != nil {
			validationErrors = append(validationErrors, *err)
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func CreateValidateRefundRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	if err := utils.CreateValidateJSONRequest(w, r, 1024*1024); err != nil {
		return err
	}

	var validationErrors utils.ValidationErrors

	if refundReq, ok := req.(*struct {
		PaymentID string `json:"payment_id"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
		Reason    string `json:"reason"`
	}); ok {
		if err := utils.CreateValidateUUID(refundReq.PaymentID, "payment_id"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateAmount(refundReq.Amount, "amount"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateCurrency(refundReq.Currency, "currency"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateString(refundReq.Reason, "reason", 0, 500, false); err != nil {
			validationErrors = append(validationErrors, *err)
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func CreateValidateFraudRequest(w http.ResponseWriter, r *http.Request, req interface{}) error {
	if err := utils.CreateValidateJSONRequest(w, r, 1024*1024); err != nil {
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
		if err := utils.CreateValidateString(fraudReq.TransactionID, "transaction_id", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateString(fraudReq.UserID, "user_id", 1, 255, true); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if fraudReq.TransactionAmount <= 0 {
			validationErrors = append(validationErrors, utils.ValidationError{
				Field:   "transaction_amount",
				Message: "must be greater than 0",
			})
		}
		if err := utils.CreateValidateCountryCode(fraudReq.BillingCountry, "billing_country"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateCountryCode(fraudReq.ShippingCountry, "shipping_country"); err != nil {
			validationErrors = append(validationErrors, *err)
		}
		if err := utils.CreateValidateIPAddress(fraudReq.IPAddress, "ip_address"); err != nil {
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

func CreateParseIntParam(r *http.Request, param string, defaultValue int) (int, error) {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}

# Fraud Detection System - Testing Guide

This document provides examples and testing instructions for the GoPay fraud detection system.

## Setup

1. **Update Configuration**: Add your OpenAI API key to `config/config.json`:
```json
{
  "openai": {
    "api_key": "sk-your-openai-api-key-here"
  }
}
```

2. **Run Database Migration**: Apply the fraud detection table migration:
```bash
psql -h localhost -U gopay_user -d gopay -f db/migrations/001_add_fraud_analysis_table.sql
```

3. **Start the Server**:
```bash
go run main.go
```

## Test Cases

### 1. Low Risk Transaction (Should Allow)

```bash
curl -X POST http://localhost:8080/api/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_low_risk_001",
    "user_id": "user_12345",
    "transaction_amount": 50.00,
    "billing_country": "US",
    "shipping_country": "US",
    "ip_address": "192.168.1.100",
    "transaction_velocity": 1
  }'
```

Expected Response:
```json
{
  "allow": true,
  "reason": "Low risk transaction"
}
```

### 2. High Risk Transaction - Country Mismatch (Should Deny)

```bash
curl -X POST http://localhost:8080/api/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_high_risk_001",
    "user_id": "user_67890",
    "transaction_amount": 1500.00,
    "billing_country": "US",
    "shipping_country": "NG",
    "ip_address": "41.58.0.1",
    "transaction_velocity": 8
  }'
```

Expected Response:
```json
{
  "allow": false,
  "reason": "High risk: billing and shipping countries don't match, high transaction amount, and high transaction velocity"
}
```

### 3. Medium Risk Transaction (Edge Case)

```bash
curl -X POST http://localhost:8080/api/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_medium_risk_001",
    "user_id": "user_11111",
    "transaction_amount": 800.00,
    "billing_country": "CA",
    "shipping_country": "US",
    "ip_address": "192.168.1.200",
    "transaction_velocity": 3
  }'
```

### 4. Very High Amount Transaction (Should Deny)

```bash
curl -X POST http://localhost:8080/api/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_very_high_001",
    "user_id": "user_22222",
    "transaction_amount": 10000.00,
    "billing_country": "GB",
    "shipping_country": "GB",
    "ip_address": "81.2.69.142",
    "transaction_velocity": 1
  }'
```

## Statistics Testing

After running several fraud analyses, test the statistics endpoint:

```bash
curl "http://localhost:8080/api/v1/fraud/stats?start_date=2025-08-10T00:00:00Z&end_date=2025-08-10T23:59:59Z"
```

Expected Response:
```json
{
  "total_transactions": 4,
  "total_fraudulent_transactions": 2,
  "average_fraud_score": 45.5,
  "fraudulent_transaction_percentage": 50.0
}
```

## Integration Testing with Enhanced Payment Flow

Test the fraud-integrated payment flow using the enhanced handler:

```bash
curl -X POST http://localhost:8080/api/v1/charges/enhanced \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust_12345",
    "amount": 15000,
    "currency": "USD",
    "payment_method": "card",
    "description": "High-value electronics purchase",
    "billing_country": "US",
    "shipping_country": "NG",
    "transaction_velocity": 7
  }'
```

Expected Response (Fraud Denied):
```json
{
  "error": "Transaction denied due to fraud risk",
  "fraud_reason": "High risk: billing and shipping countries don't match, high transaction amount, and high transaction velocity",
  "transaction_id": "txn_1723276800123"
}
```

## Load Testing

Create a simple script to generate multiple transactions for testing statistics:

```bash
#!/bin/bash

# Generate 20 test transactions
for i in {1..20}; do
    # Mix of low and high risk transactions
    if [ $((i % 3)) -eq 0 ]; then
        # High risk transaction
        curl -s -X POST http://localhost:8080/api/v1/fraud/analyze \
          -H "Content-Type: application/json" \
          -d "{
            \"transaction_id\": \"txn_load_test_$i\",
            \"user_id\": \"user_$i\",
            \"transaction_amount\": $((RANDOM % 5000 + 1000)),
            \"billing_country\": \"US\",
            \"shipping_country\": \"NG\",
            \"ip_address\": \"192.168.1.$i\",
            \"transaction_velocity\": $((RANDOM % 10 + 5))
          }" > /dev/null
    else
        # Low risk transaction
        curl -s -X POST http://localhost:8080/api/v1/fraud/analyze \
          -H "Content-Type: application/json" \
          -d "{
            \"transaction_id\": \"txn_load_test_$i\",
            \"user_id\": \"user_$i\",
            \"transaction_amount\": $((RANDOM % 200 + 20)),
            \"billing_country\": \"US\",
            \"shipping_country\": \"US\",
            \"ip_address\": \"192.168.1.$i\",
            \"transaction_velocity\": $((RANDOM % 3 + 1))
          }" > /dev/null
    fi
    echo "Processed transaction $i"
done

echo "Load test complete. Checking statistics..."
curl "http://localhost:8080/api/v1/fraud/stats?start_date=$(date -u +%Y-%m-%dT00:00:00Z)&end_date=$(date -u +%Y-%m-%dT23:59:59Z)"
```

## Error Scenarios

### 1. Invalid Request Data

```bash
curl -X POST http://localhost:8080/api/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "",
    "user_id": "user_12345"
  }'
```

Expected: `400 Bad Request` with validation error

### 2. OpenAI API Failure Simulation

Temporarily use an invalid API key to test fallback logic:

```json
{
  "openai": {
    "api_key": "invalid_key_test"
  }
}
```

The system should fall back to rule-based detection and still provide results.

### 3. Invalid Date Range for Statistics

```bash
curl "http://localhost:8080/api/v1/fraud/stats?start_date=invalid&end_date=2025-08-10T23:59:59Z"
```

Expected: `400 Bad Request` with date format error

## Monitoring Checklist

- [ ] All fraud analyses are being logged to the database
- [ ] Statistics endpoint returns accurate aggregations
- [ ] OpenAI integration is working (check logs for API calls)
- [ ] Fallback logic activates when OpenAI is unavailable
- [ ] Payment flow integration prevents fraudulent transactions
- [ ] Performance is acceptable (fraud check <2 seconds)

## Production Considerations

1. **Rate Limiting**: Implement rate limiting for fraud analysis endpoints
2. **Caching**: Cache fraud scores for repeated transaction patterns
3. **Monitoring**: Set up alerts for high fraud rates or API failures
4. **Tuning**: Adjust fraud score thresholds based on false positive rates
5. **Data Retention**: Implement data retention policies for fraud analysis results

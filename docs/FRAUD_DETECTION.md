# Fraud Detection

AI-powered fraud analysis using OpenAI GPT-4o with rule-based fallback.

## API

### POST /v1/fraud/analyze

```json
{
  "transaction_id": "txn_123",
  "user_id": "user_456",
  "transaction_amount": 150.00,
  "billing_country": "US",
  "shipping_country": "US",
  "ip_address": "192.168.1.1",
  "transaction_velocity": 2
}
```

**Response:**
```json
{
  "allow": true,
  "reason": "Low risk transaction"
}
```

### GET /v1/fraud/stats

Query params: `start_date`, `end_date` (ISO 8601)

```json
{
  "total_transactions": 1234,
  "total_fraudulent_transactions": 56,
  "average_fraud_score": 23.5,
  "fraudulent_transaction_percentage": 4.54
}
```

## Configuration

Add to `config.json`:
```json
{
  "openai": {
    "api_key": "sk-..."
  }
}
```

## Scoring Logic

### AI Analysis (Primary)
Uses GPT-4o to evaluate:
- Country mismatches
- Transaction amount patterns
- Transaction velocity
- IP address risk

### Fallback Rules (When AI Unavailable)

| Condition | Score |
|-----------|-------|
| Country mismatch | +25 |
| Amount > $1000 | +30 |
| Velocity > 5 txns | +35 |
| Amount > $5000 | +20 |

**Decision:** Score ≥70 and flagged → Deny

## Privacy

- No PII sent to OpenAI (names, emails, addresses)
- Only country codes, amount categories, risk levels transmitted
- All results stored locally

## Integration

Fraud check runs synchronously before payment processing. Denied transactions are rejected immediately.

## Database

```sql
CREATE TABLE fraud_analysis_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    transaction_amount DECIMAL(10,2) NOT NULL,
    billing_country VARCHAR(3) NOT NULL,
    shipping_country VARCHAR(3) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    transaction_velocity INTEGER NOT NULL,
    is_fraudulent BOOLEAN NOT NULL,
    fraud_score INTEGER NOT NULL CHECK (fraud_score >= 0 AND fraud_score <= 100),
    reason TEXT NOT NULL,
    allow BOOLEAN NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## Test Cases

**Low Risk (Allow):**
```bash
curl -X POST http://localhost:8080/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{"transaction_id":"txn_1","user_id":"user_1","transaction_amount":50,"billing_country":"US","shipping_country":"US","ip_address":"192.168.1.1","transaction_velocity":1}'
```

**High Risk (Deny):**
```bash
curl -X POST http://localhost:8080/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{"transaction_id":"txn_2","user_id":"user_2","transaction_amount":1500,"billing_country":"US","shipping_country":"NG","ip_address":"41.58.0.1","transaction_velocity":8}'
```

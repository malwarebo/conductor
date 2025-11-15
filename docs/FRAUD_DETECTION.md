# Fraud Detection Layer

The fraud detection layer provides intelligent fraud analysis for payment transactions using OpenAI's GPT-4 model as the primary analysis engine with fallback logic for reliability.

## Features

- **AI-Powered Analysis**: Uses OpenAI GPT-4o for sophisticated fraud pattern detection
- **Fallback Logic**: Basic rule-based detection when AI service is unavailable
- **PII Protection**: Anonymizes sensitive data before sending to external services
- **Statistics Dashboard**: Provides fraud analytics and reporting
- **Configurable Thresholds**: Adjustable fraud score thresholds for decision making

## API Endpoints

### 1. Fraud Analysis Endpoint

Analyzes a transaction for fraud risk and returns an allow/deny decision.

```
POST /v1/fraud/analyze
```

**Request Body:**
```json
{
  "transaction_id": "string",
  "user_id": "string",
  "transaction_amount": "number",
  "billing_country": "string",
  "shipping_country": "string",
  "ip_address": "string",
  "transaction_velocity": "integer"
}
```

**Response:**
```json
{
  "allow": boolean,
  "reason": "string"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/fraud/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "txn_123456789",
    "user_id": "user_987654321",
    "transaction_amount": 150.00,
    "billing_country": "US",
    "shipping_country": "US",
    "ip_address": "192.168.1.1",
    "transaction_velocity": 2
  }'
```

### 2. Fraud Statistics Endpoint

Returns aggregated fraud statistics for a date range.

```
GET /v1/fraud/stats?start_date={ISO8601}&end_date={ISO8601}
```

**Query Parameters:**
- `start_date` - Start date in ISO 8601 format (e.g., `2025-08-01T00:00:00Z`)
- `end_date` - End date in ISO 8601 format (e.g., `2025-08-10T23:59:59Z`)

**Response:**
```json
{
  "total_transactions": 1234,
  "total_fraudulent_transactions": 56,
  "average_fraud_score": 23.5,
  "fraudulent_transaction_percentage": 4.54
}
```

**Example:**
```bash
curl "http://localhost:8080/v1/fraud/stats?start_date=2025-08-01T00:00:00Z&end_date=2025-08-10T23:59:59Z"
```

## Configuration

Add OpenAI configuration to your `config.json`:

```json
{
  "openai": {
    "api_key": "your_openai_api_key_here"
  }
}
```

## Database Schema

The fraud detection system uses the `fraud_analysis_results` table to store analysis results:

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

## Fraud Detection Logic

### AI Analysis (Primary)

The system uses OpenAI GPT-4o with a specialized prompt to analyze transaction data. The AI considers:

- Country mismatches between billing and shipping
- Transaction amounts relative to typical patterns
- Transaction velocity (frequency)
- IP address risk indicators
- User behavior patterns

### Fallback Logic (Secondary)

When the AI service is unavailable, the system uses rule-based detection:

- **Country Mismatch**: +25 fraud score
- **High Amount** (>$1000): +30 fraud score
- **High Velocity** (>5 transactions): +35 fraud score
- **Very High Amount** (>$5000): +20 fraud score

Transactions with fraud score ≥50 are flagged as fraudulent.

### Decision Threshold

- **Allow**: Fraud score <70 or not flagged as fraudulent
- **Deny**: Fraud score ≥70 and flagged as fraudulent

## Privacy and Security

- **No PII Transmission**: User names, emails, and detailed addresses are never sent to OpenAI
- **Data Anonymization**: Only country codes, amount categories, and risk levels are shared
- **Local Logging**: All analysis results are stored locally for audit and statistics

## Integration with Payment Flow

The fraud detection layer integrates with the payment orchestrator:

1. Payment request received
2. Fraud analysis performed (synchronously)
3. If denied, payment is rejected immediately
4. If allowed, payment proceeds to provider
5. All decisions are logged

## Statistics and Tracking

Track fraud detection performance with the statistics endpoint:

- Fraud detection rates over time
- Average fraud scores
- Transaction volume trends
- False positive/negative rates

## Migration

Run the migration to add the fraud analysis table:

```bash
psql -h localhost -U conductor_user -d conductor -f db/migrations/001_add_fraud_analysis_table.sql
```

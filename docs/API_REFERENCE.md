# Conductor API Reference

## Overview

The Conductor API provides a unified interface for payment processing across multiple providers. This document covers all available endpoints, request/response formats, and authentication requirements.

## Base URL

```
http://localhost:8080/v1
```

## Authentication

Conductor uses JWT-based authentication. Include the JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### Getting a JWT Token

For development and testing, you can generate JWT tokens using the application's JWT manager directly:

```bash
# Using curl to get a token (if you have a token endpoint)
curl -X POST http://localhost:8080/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "your_password"
  }'
```

Or programmatically in your application:

```go
// Example: Generate a JWT token programmatically
jwtManager := security.CreateJWTManager("your-jwt-secret", "conductor", "conductor-api")
token, err := jwtManager.GenerateToken("user123", "test@example.com", []string{"admin"}, "api-key-123", 24*time.Hour)
```

## Common Response Formats

### Success Response
```json
{
  "status": "success",
  "data": { ... },
  "timestamp": "2025-01-26T10:30:00Z"
}
```

### Error Response
```json
{
  "error": "Error message",
  "status": "400",
  "timestamp": "2025-01-26T10:30:00Z"
}
```

## Endpoints

### Health Check

#### GET /health
Check system health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-26T10:30:00Z",
  "uptime": "2h15m30s"
}
```

### Payments

#### POST /charges
Create a new payment charge.

**Request Body:**
```json
{
  "amount": 1000,
  "currency": "USD",
  "description": "Payment for order #123",
  "customer_email": "customer@example.com",
  "customer_name": "John Doe",
  "metadata": {
    "order_id": "123",
    "product": "premium_plan"
  }
}
```

**Response:**
```json
{
  "id": "ch_1234567890",
  "amount": 1000,
  "currency": "USD",
  "status": "pending",
  "provider": "stripe",
  "provider_transaction_id": "pi_1234567890",
  "created_at": "2025-01-26T10:30:00Z"
}
```

#### GET /charges/{id}
Retrieve a specific charge.

**Response:**
```json
{
  "id": "ch_1234567890",
  "amount": 1000,
  "currency": "USD",
  "status": "succeeded",
  "provider": "stripe",
  "provider_transaction_id": "pi_1234567890",
  "created_at": "2025-01-26T10:30:00Z",
  "updated_at": "2025-01-26T10:31:00Z"
}
```

#### POST /charges/{id}/refund
Refund a charge.

**Request Body:**
```json
{
  "amount": 500,
  "reason": "customer_request"
}
```

**Response:**
```json
{
  "id": "rf_1234567890",
  "charge_id": "ch_1234567890",
  "amount": 500,
  "status": "pending",
  "created_at": "2025-01-26T10:30:00Z"
}
```

### Subscriptions

#### POST /subscriptions
Create a new subscription.

**Request Body:**
```json
{
  "plan_id": "plan_premium_monthly",
  "customer_email": "customer@example.com",
  "customer_name": "John Doe",
  "payment_method": "card",
  "billing_cycle": "monthly"
}
```

**Response:**
```json
{
  "id": "sub_1234567890",
  "plan_id": "plan_premium_monthly",
  "status": "active",
  "current_period_start": "2025-01-26T10:30:00Z",
  "current_period_end": "2025-02-26T10:30:00Z",
  "created_at": "2025-01-26T10:30:00Z"
}
```

#### GET /subscriptions/{id}
Retrieve a specific subscription.

**Response:**
```json
{
  "id": "sub_1234567890",
  "plan_id": "plan_premium_monthly",
  "status": "active",
  "current_period_start": "2025-01-26T10:30:00Z",
  "current_period_end": "2025-02-26T10:30:00Z",
  "created_at": "2025-01-26T10:30:00Z"
}
```

#### POST /subscriptions/{id}/cancel
Cancel a subscription.

**Response:**
```json
{
  "id": "sub_1234567890",
  "status": "cancelled",
  "cancelled_at": "2025-01-26T10:30:00Z"
}
```

### Plans

#### GET /plans
List all available plans.

**Response:**
```json
{
  "plans": [
    {
      "id": "plan_basic_monthly",
      "name": "Basic Plan",
      "description": "Basic monthly subscription",
      "amount": 999,
      "currency": "USD",
      "interval": "month",
      "active": true
    }
  ]
}
```

#### POST /plans
Create a new plan.

**Request Body:**
```json
{
  "name": "Premium Plan",
  "description": "Premium monthly subscription",
  "amount": 2999,
  "currency": "USD",
  "interval": "month"
}
```

**Response:**
```json
{
  "id": "plan_premium_monthly",
  "name": "Premium Plan",
  "description": "Premium monthly subscription",
  "amount": 2999,
  "currency": "USD",
  "interval": "month",
  "active": true,
  "created_at": "2025-01-26T10:30:00Z"
}
```

### Fraud Detection

#### POST /fraud/analyze
Analyze a transaction for fraud.

**Request Body:**
```json
{
  "amount": 1000,
  "currency": "USD",
  "customer_email": "customer@example.com",
  "customer_ip": "192.168.1.1",
  "customer_country": "US",
  "payment_method": "card",
  "transaction_type": "payment"
}
```

**Response:**
```json
{
  "fraud_score": 0.15,
  "risk_level": "low",
  "recommendation": "approve",
  "reasons": [
    "Low risk transaction pattern",
    "Valid customer information"
  ],
  "analyzed_at": "2025-01-26T10:30:00Z"
}
```

#### GET /fraud/stats
Get fraud detection statistics.

**Query Parameters:**
- `start_date` (optional): Start date for statistics (ISO 8601)
- `end_date` (optional): End date for statistics (ISO 8601)

**Response:**
```json
{
  "total_transactions": 1000,
  "fraudulent_transactions": 15,
  "fraud_rate": 0.015,
  "avg_fraud_score": 0.25,
  "period": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-01-26T23:59:59Z"
  }
}
```

### AI Routing

#### POST /routing/select
Get AI-powered provider selection.

**Request Body:**
```json
{
  "amount": 1000,
  "currency": "USD",
  "customer_country": "US",
  "payment_method": "card",
  "transaction_type": "payment"
}
```

**Response:**
```json
{
  "selected_provider": "stripe",
  "confidence": 0.95,
  "reasoning": "Stripe has 99.2% success rate for USD transactions in US",
  "alternatives": [
    {
      "provider": "xendit",
      "confidence": 0.85,
      "reasoning": "Good for international transactions"
    }
  ]
}
```

#### GET /routing/stats
Get routing statistics.

**Response:**
```json
{
  "total_routes": 1000,
  "success_rate": 0.987,
  "avg_response_time": 150,
  "provider_breakdown": {
    "stripe": {
      "routes": 750,
      "success_rate": 0.992,
      "avg_response_time": 120
    },
    "xendit": {
      "routes": 250,
      "success_rate": 0.976,
      "avg_response_time": 200
    }
  }
}
```

### Disputes

#### GET /disputes
List all disputes.

**Query Parameters:**
- `status` (optional): Filter by status (open, closed, won, lost)
- `limit` (optional): Number of results (default: 20)
- `offset` (optional): Number of results to skip (default: 0)

**Response:**
```json
{
  "disputes": [
    {
      "id": "dp_1234567890",
      "charge_id": "ch_1234567890",
      "amount": 1000,
      "currency": "USD",
      "status": "open",
      "reason": "fraudulent",
      "created_at": "2025-01-26T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

#### GET /disputes/{id}
Retrieve a specific dispute.

**Response:**
```json
{
  "id": "dp_1234567890",
  "charge_id": "ch_1234567890",
  "amount": 1000,
  "currency": "USD",
  "status": "open",
  "reason": "fraudulent",
  "evidence": {
    "customer_communication": "Customer claims unauthorized charge",
    "duplicate_charge": false,
    "product_description": "Premium subscription"
  },
  "created_at": "2025-01-26T10:30:00Z"
}
```

### Performance Monitoring

#### POST /performance/benchmark
Run performance benchmarks.

**Request Body:**
```json
{
  "name": "payment_benchmark",
  "duration": "10s",
  "concurrency": 5,
  "operation": "payment"
}
```

**Response:**
```json
{
  "name": "payment_benchmark",
  "duration": "10s",
  "concurrency": 5,
  "operations": 500,
  "ops_per_second": 50,
  "avg_duration": "20ms",
  "p95_duration": "45ms",
  "p99_duration": "80ms",
  "errors": 0,
  "error_rate": 0.0
}
```

#### POST /performance/load-test
Run load tests.

**Request Body:**
```json
{
  "concurrency": 10,
  "duration": "30s",
  "ramp_up_duration": "5s",
  "target_rps": 50,
  "endpoint": "http://localhost:8080/v1/health"
}
```

**Response:**
```json
{
  "concurrency": 10,
  "duration": "30s",
  "total_requests": 1500,
  "successful_requests": 1485,
  "failed_requests": 15,
  "avg_response_time": "25ms",
  "p95_response_time": "60ms",
  "p99_response_time": "120ms",
  "requests_per_second": 50.0
}
```

#### GET /performance/metrics
Get performance metrics.

**Response:**
```json
{
  "metrics": {
    "response_time": "25ms",
    "throughput": 50.0,
    "error_rate": 0.01,
    "memory_usage": "128MB",
    "cpu_usage": "15%"
  },
  "summary": {
    "by_type": {
      "payment": {
        "count": 1000,
        "avg_duration": "20ms"
      }
    },
    "total_metrics": 1
  }
}
```

#### GET /performance/health
Get enhanced health status.

**Response:**
```json
{
  "status": "healthy",
  "checks": {
    "database": {
      "status": "healthy",
      "response_time": "5ms"
    },
    "redis": {
      "status": "healthy",
      "response_time": "2ms"
    }
  },
  "summary": {
    "healthy": 2,
    "unhealthy": 0,
    "degraded": 0
  }
}
```

#### GET /performance/optimization
Get optimization recommendations.

**Response:**
```json
{
  "metrics": {
    "avg_response_time": "25ms",
    "cache_hit_rate": 0.85,
    "error_rate": 0.01,
    "memory_usage": "128MB"
  },
  "recommendations": [
    "Consider increasing cache TTL for better hit rate",
    "Database query optimization recommended"
  ],
  "optimizations": {
    "cache_ttl": "2h",
    "cache_size": 2000,
    "recommendation": "Increase cache TTL and size"
  }
}
```

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid request data |
| 401 | Unauthorized - Invalid or missing authentication |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource not found |
| 409 | Conflict - Resource already exists |
| 422 | Unprocessable Entity - Validation error |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error - Server error |
| 502 | Bad Gateway - Provider error |
| 503 | Service Unavailable - Service temporarily unavailable |

## Rate Limits

| Tier | Requests per Second | Burst |
|------|---------------------|-------|
| Default | 10 | 20 |
| Standard | 50 | 100 |
| Premium | 200 | 400 |
| Admin | 1000 | 2000 |

## Webhooks

Conductor sends webhooks for important events. Configure webhook endpoints in your provider settings.

### Webhook Events

- `payment.succeeded` - Payment completed successfully
- `payment.failed` - Payment failed
- `payment.refunded` - Payment refunded
- `subscription.created` - Subscription created
- `subscription.cancelled` - Subscription cancelled
- `dispute.created` - Dispute created
- `dispute.updated` - Dispute updated

### Webhook Security

All webhooks include a signature header for verification:

```
X-Conductor-Signature: sha256=abc123...
```

Verify the signature using your webhook secret to ensure the webhook is from Conductor.

---

## API Endpoints

Note: This section need to be migrated to it is own separate page.

### Payments

- `POST /v1/charges` - Create a new charge
- `POST /v1/refunds` - Create a refund

### Subscriptions

- `POST /v1/plans` - Create a subscription plan
- `GET /v1/plans` - List all plans
- `GET /v1/plans/:id` - Get plan details
- `PUT /v1/plans/:id` - Update plan
- `DELETE /v1/plans/:id` - Delete plan
- `POST /v1/subscriptions` - Create a subscription
- `GET /v1/subscriptions` - List subscriptions (requires customer_id parameter)
- `GET /v1/subscriptions/:id` - Get subscription details
- `PUT /v1/subscriptions/:id` - Update subscription
- `DELETE /v1/subscriptions/:id` - Cancel subscription

### Disputes

- `POST /v1/disputes` - Create a dispute
- `GET /v1/disputes` - List disputes (requires customer_id parameter)
- `GET /v1/disputes/:id` - Get dispute details
- `PUT /v1/disputes/:id` - Update dispute
- `POST /v1/disputes/:id/evidence` - Submit evidence
- `GET /v1/disputes/stats` - Get dispute statistics

### Fraud Detection

- `POST /v1/fraud/analyze` - Analyze transaction for fraud risk
- `GET /v1/fraud/stats` - Get fraud detection statistics

### System

- `GET /v1/health` - Health check

## API Examples

Here are some real examples of how to use the API. The system automatically routes your requests to the right payment provider based on the currency!

### Creating Charges

#### Basic charge with Stripe (USD)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "customer_id": "cus_123456789",
    "amount": 2500,
    "currency": "USD",
    "payment_method": "pm_123456789",
    "description": "Payment for order #12345"
  }'
```

#### Charge with metadata using Xendit (IDR)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "customer_id": "customer_123",
    "amount": 500000,
    "currency": "IDR",
    "payment_method": "pm_xendit_123",
    "description": "Premium subscription payment",
    "metadata": {
      "order_id": "ORD-2024-001",
      "user_id": "user_456",
      "product_type": "subscription",
      "billing_cycle": "monthly"
    }
  }'
```

#### Charge using Razorpay (INR)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "customer_id": "customer_india_123",
    "amount": 100000,
    "currency": "INR",
    "description": "Payment for order #67890",
    "metadata": {
      "order_id": "ORD-2024-IN-001",
      "payment_method_preference": "upi"
    }
  }'
```

Note: Razorpay uses an Order → Payment flow. The response will include `requires_action: true` with the order ID in `client_secret`. Use Razorpay.js on the frontend to complete the payment.

#### High-value charge with Stripe (EUR)

```bash
curl -X POST http://localhost:8080/v1/charges \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cus_europe_789",
    "amount": 9999,
    "currency": "EUR",
    "payment_method": "pm_europe_456",
    "description": "Annual enterprise license",
    "metadata": {
      "license_type": "enterprise",
      "duration": "annual",
      "seats": 100,
      "region": "EU"
    }
  }'
```

### Creating Refunds

#### Simple refund

```bash
curl -X POST http://localhost:8080/v1/refunds \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_api_key_here" \
  -d '{
    "payment_id": "ch_123456789",
    "amount": 2500,
    "currency": "USD",
    "reason": "Customer requested refund"
  }'
```

#### Partial refund with metadata

```bash
curl -X POST http://localhost:8080/v1/refunds \
  -H "Content-Type: application/json" \
  -d '{
    "payment_id": "ch_123456789",
    "amount": 1000,
    "currency": "USD",
    "reason": "Partial refund for damaged item",
    "metadata": {
      "refund_type": "partial",
      "damage_reported": true,
      "customer_service_agent": "agent_123"
    }
  }'
```

### Managing Subscriptions

#### Create a subscription plan

```bash
curl -X POST http://localhost:8080/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Premium Plan",
    "description": "Premium features with priority support",
    "amount": 2999,
    "currency": "USD",
    "billing_period": "monthly",
    "pricing_type": "fixed",
    "trial_days": 7,
    "features": ["priority_support", "advanced_analytics", "api_access"]
  }'
```

#### Create a subscription

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cus_123456789",
    "plan_id": "plan_premium_001",
    "quantity": 1,
    "trial_days": 7,
    "payment_method_id": "pm_123456789",
    "metadata": {
      "marketing_source": "website",
      "referral_code": "WELCOME10"
    }
  }'
```

### Handling Disputes

#### Create a dispute

```bash
curl -X POST http://localhost:8080/v1/disputes \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cus_123456789",
    "transaction_id": "ch_123456789",
    "amount": 2500,
    "currency": "USD",
    "reason": "fraudulent",
    "evidence": {
      "customer_communication": "Customer claims unauthorized charge"
    },
    "due_by": "2024-02-15T23:59:59Z"
  }'
```

#### Submit evidence for a dispute

```bash
curl -X POST http://localhost:8080/v1/disputes/disp_123/evidence \
  -H "Content-Type: application/json" \
  -d '{
    "type": "customer_communication",
    "description": "Email from customer confirming receipt",
    "files": ["https://example.com/evidence1.pdf"],
    "metadata": {
      "evidence_type": "email",
      "customer_email": "customer@example.com"
    }
  }'
```

### System Health

#### Check if the system is healthy

```bash
curl -X GET http://localhost:8080/v1/health
```

## How Currency Routing Works

The system is smart about routing your payments to the right provider:

- **Stripe**: USD, EUR, GBP (perfect for international payments)
- **Xendit**: IDR, SGD, MYR, PHP, THB, VND (great for Southeast Asia)
- **Razorpay**: INR (optimized for India with UPI and Netbanking support)

Just specify the currency in your request, and the system automatically picks the best provider.

## Important Notes

### Amount Format

Always use the smallest currency unit:

- **USD/EUR**: cents (1000 = $10.00)
- **IDR**: rupiah (50000 = Rp 50,000)
- **SGD**: cents (1500 = S$15.00)
- **INR**: paise (100000 = ₹1,000.00)

### Payment Methods

Make sure you're using valid payment method IDs from your chosen provider:

- Stripe: `pm_123456789`
- Xendit: `pm_xendit_123`

### Customer IDs

Your customer IDs should match what's in your provider's system.

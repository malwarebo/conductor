# Conductor API Reference

Conductor provides a unified REST API for payment processing across multiple providers (Stripe, Xendit, Razorpay, Airwallex). All endpoints are under the `/v1` prefix.

## Base URL

```
http://localhost:8080/v1
```

## Authentication

All API requests require a Bearer token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

Tokens are generated using the JWT manager with your configured secret. For programmatic access, you can also use the tenant API key flow via the `X-API-Key` header.

```go
jwtManager := security.CreateJWTManager("your-jwt-secret", "conductor", "conductor-api")
token, err := jwtManager.GenerateToken("user123", "test@example.com", []string{"admin"}, "api-key-123", 24*time.Hour)
```

## Error Responses

All errors follow the same format:

```json
{
  "error": "Description of what went wrong"
}
```

## Rate Limits

| Tier | Requests per Second | Burst |
|------|---------------------|-------|
| Default | 10 | 20 |
| Standard | 50 | 100 |
| Premium | 100 | 200 |

All list endpoints enforce a maximum page size of 100 items.

---

## Health

### GET /health

Returns the current server health.

```json
{
  "status": "healthy",
  "timestamp": "2025-01-26T10:30:00Z",
  "uptime": "2h15m30s"
}
```

---

## Payments

### POST /charges

Create a new payment charge. The system automatically routes to the right provider based on currency.

**Headers:**
- `Idempotency-Key` (optional): Prevents duplicate charges.

**Request:**
```json
{
  "customer_id": "cus_123456789",
  "amount": 2500,
  "currency": "USD",
  "payment_method": "pm_123456789",
  "description": "Payment for order #12345",
  "capture_method": "automatic",
  "metadata": {
    "order_id": "123"
  }
}
```

**Response (200):**
```json
{
  "id": "pi_abc123",
  "customer_id": "cus_123456789",
  "amount": 2500,
  "currency": "usd",
  "status": "succeeded",
  "payment_method": "pm_123456789",
  "provider_name": "stripe",
  "provider_charge_id": "pi_abc123",
  "capture_method": "automatic",
  "captured_amount": 2500,
  "created_at": "2025-01-26T10:30:00Z"
}
```

### POST /authorize

Create a payment authorization (manual capture). Accepts the same body as `/charges` with `capture_method` forced to `manual`.

**Headers:**
- `Idempotency-Key` (optional)

### GET /payments/{id}

Retrieve a specific payment by ID.

### POST /payments/{id}/capture

Capture a previously authorized payment.

**Request:**
```json
{
  "amount": 2500
}
```

Amount is optional — omitting it captures the full authorized amount.

### POST /payments/{id}/void

Cancel an authorized (uncaptured) payment.

### POST /payments/{id}/confirm

Confirm a payment that requires 3D Secure authentication.

---

## Refunds

### POST /refunds

Create a refund for an existing payment.

**Request:**
```json
{
  "payment_id": "pi_abc123",
  "amount": 1000,
  "reason": "Customer requested refund",
  "metadata": {
    "refund_type": "partial"
  }
}
```

**Response (200):**
```json
{
  "id": "re_xyz789",
  "payment_id": "pi_abc123",
  "amount": 1000,
  "currency": "usd",
  "status": "succeeded",
  "reason": "Customer requested refund",
  "provider_name": "stripe",
  "provider_refund_id": "re_xyz789",
  "created_at": "2025-01-26T10:30:00Z"
}
```

---

## Payment Sessions

Payment sessions let you create and manage payment intents with a client-side flow.

### POST /payment-sessions

Create a new payment session.

**Request:**
```json
{
  "amount": 5000,
  "currency": "USD",
  "customer_id": "cus_123",
  "description": "Order #456",
  "capture_method": "automatic",
  "return_url": "https://example.com/return",
  "metadata": {}
}
```

### GET /payment-sessions

List payment sessions.

**Query Parameters:**
- `customer_id` (optional)
- `limit` (optional, default: 20, max: 100)

**Response (200):**
```json
{
  "payment_sessions": [ ... ],
}
```

### GET /payment-sessions/{id}

Retrieve a payment session.

### PATCH /payment-sessions/{id}

Update a payment session (amount, currency, description, payment method, metadata).

### POST /payment-sessions/{id}/confirm

Confirm a payment session, optionally attaching a payment method.

### POST /payment-sessions/{id}/capture

Capture a payment session. Optionally pass `{"amount": 5000}` for partial capture.

### POST /payment-sessions/{id}/cancel

Cancel a payment session.

---

## Plans

### POST /plans

Create a subscription plan.

**Request:**
```json
{
  "name": "Premium Plan",
  "description": "Premium features with priority support",
  "amount": 29.99,
  "currency": "USD",
  "billing_period": "month",
  "trial_days": 7,
  "features": ["priority_support", "advanced_analytics"]
}
```

### GET /plans

List all plans.

**Response (200):**
```json
{
  "data": [ ... ],
  "total": 5
}
```

### GET /plans/{id}

Retrieve a specific plan.

### PUT /plans/{id}

Update a plan.

### DELETE /plans/{id}

Delete a plan. Returns `204 No Content`.

---

## Subscriptions

### POST /subscriptions

Create a subscription.

**Request:**
```json
{
  "customer_id": "cus_123456789",
  "plan_id": "plan_premium_001",
  "quantity": 1,
  "trial_days": 7,
  "payment_method_id": "pm_123456789"
}
```

### GET /subscriptions

List subscriptions for a customer.

**Query Parameters:**
- `customer_id` (required)

**Response (200):**
```json
{
  "data": [ ... ],
  "total": 3
}
```

### GET /subscriptions/{id}

Retrieve a subscription.

### PUT /subscriptions/{id}

Update a subscription (plan, quantity, payment method, metadata).

### DELETE /subscriptions/{id}

Cancel a subscription.

**Request:**
```json
{
  "cancel_at_period_end": true,
  "reason": "No longer needed"
}
```

---

## Disputes

### POST /disputes

Create a dispute (availability depends on provider).

### GET /disputes

List disputes for a customer.

**Query Parameters:**
- `customer_id` (required)

**Response (200):**
```json
{
  "data": [ ... ],
  "total": 2
}
```

### GET /disputes/{id}

Retrieve a specific dispute.

### PUT /disputes/{id}

Update dispute metadata.

### POST /disputes/{id}/accept

Accept (close) a dispute.

### POST /disputes/{id}/contest

Contest a dispute with evidence.

**Request:**
```json
{
  "uncategorized_text": "Customer received the product",
  "product_description": "Digital subscription",
  "customer_name": "John Doe",
  "customer_email_address": "john@example.com"
}
```

### POST /disputes/{id}/evidence

Submit structured evidence for a dispute.

**Request:**
```json
{
  "type": "customer_communication",
  "description": "Email from customer confirming receipt",
  "files": ["https://example.com/evidence1.pdf"]
}
```

### GET /disputes/stats

Get aggregate dispute statistics (total, open, won, lost, canceled).

---

## Fraud Detection

### POST /fraud/analyze

Analyze a transaction for fraud risk using AI-powered analysis.

**Request:**
```json
{
  "transaction_id": "txn_001",
  "user_id": "user_123",
  "transaction_amount": 25.00,
  "billing_country": "US",
  "shipping_country": "US",
  "ip_address": "192.168.1.1",
  "transaction_velocity": 1
}
```

### GET /fraud/stats

Get fraud detection statistics for a date range.

**Query Parameters:**
- `start_date` (required, ISO 8601)
- `end_date` (required, ISO 8601)

---

## AI Routing

### POST /routing/select

Get an AI-powered recommendation for which payment provider to use.

**Request:**
```json
{
  "amount": 1000,
  "currency": "USD",
  "country": "US",
  "payment_method": "card"
}
```

### GET /routing/stats

Get provider performance statistics.

### GET /routing/config

Get current routing configuration.

### PUT /routing/config

Update routing configuration (AI routing toggle, cache TTL, confidence threshold, fallback provider).

---

## Invoices

### POST /invoices

Create an invoice.

**Request:**
```json
{
  "customer_id": "cus_123",
  "amount": 5000,
  "currency": "USD",
  "description": "Consulting services",
  "due_date": "2025-02-28T00:00:00Z"
}
```

### GET /invoices

List invoices.

**Query Parameters:**
- `customer_id` (optional)
- `status` (optional)
- `limit` (optional, default: 20, max: 100)
- `offset` (optional)

### GET /invoices/{id}

Retrieve an invoice.

### POST /invoices/{id}/cancel

Cancel (void) an invoice.

---

## Payouts

### POST /payouts

Create a payout.

**Request:**
```json
{
  "amount": 10000,
  "currency": "USD",
  "destination_account": "acct_123",
  "description": "Weekly payout"
}
```

### GET /payouts

List payouts.

**Query Parameters:**
- `reference_id` (optional)
- `status` (optional)
- `limit` (optional, default: 20, max: 100)
- `offset` (optional)

### GET /payouts/{id}

Retrieve a payout.

### POST /payouts/{id}/cancel

Cancel a payout.

### GET /payout-channels

List available payout channels for a currency.

**Query Parameters:**
- `currency` (optional)

---

## Customers

### POST /customers

Create a customer.

**Request:**
```json
{
  "email": "customer@example.com",
  "name": "Jane Doe",
  "phone": "+1234567890"
}
```

### GET /customers/{id}

Retrieve a customer.

### PUT /customers/{id}

Update a customer (email, name, phone, metadata).

### DELETE /customers/{id}

Delete a customer.

---

## Payment Methods

### POST /payment-methods

Create or register a payment method.

**Request:**
```json
{
  "customer_id": "cus_123",
  "card_token": "pm_abc123",
  "type": "card",
  "reusable": true,
  "is_default": true
}
```

### GET /payment-methods

List payment methods.

**Query Parameters:**
- `customer_id` (optional)
- `type` (optional)

### GET /payment-methods/{id}

Retrieve a payment method.

### POST /payment-methods/{id}/attach

Attach a payment method to a customer.

**Request:**
```json
{
  "customer_id": "cus_123"
}
```

### POST /payment-methods/{id}/detach

Detach a payment method from a customer.

### POST /payment-methods/{id}/expire

Mark a payment method as expired.

---

## Balance

### GET /balance

Get account balance. Optionally filter by currency.

**Query Parameters:**
- `currency` (optional): ISO currency code

---

## Tenants

### POST /tenants

Create a new tenant.

### GET /tenants

List tenants.

**Query Parameters:**
- `limit` (optional, default: 20, max: 100)
- `offset` (optional)
- `active_only` (optional, default: true)

### GET /tenants/{id}

Retrieve a tenant.

### PUT /tenants/{id}

Update a tenant.

### DELETE /tenants/{id}

Delete a tenant.

### POST /tenants/{id}/deactivate

Deactivate a tenant.

### POST /tenants/{id}/regenerate-secret

Regenerate a tenant's API secret.

---

## Audit Logs

### GET /audit-logs

List audit logs.

**Query Parameters:**
- `user_id` (optional)
- `action` (optional)
- `resource_type` (optional)
- `resource_id` (optional)
- `limit` (optional, default: 100, max: 100)
- `offset` (optional)
- `start_date` (optional, ISO 8601)
- `end_date` (optional, ISO 8601)

### GET /audit-logs/{resource_type}/{resource_id}

Get the audit history for a specific resource.

**Query Parameters:**
- `limit` (optional, default: 50, max: 100)

---

## Webhooks

Conductor receives webhooks from payment providers at these endpoints. Each handler validates the signature using the provider's own format before processing.

| Provider | Endpoint | Signature Header |
|----------|----------|-----------------|
| Stripe | `POST /v1/webhooks/stripe` | `Stripe-Signature` |
| Xendit | `POST /v1/webhooks/xendit` | `x-callback-token` |
| Razorpay | `POST /v1/webhooks/razorpay` | `X-Razorpay-Signature` |

Conductor also sends outbound webhooks to tenant-configured URLs. Outbound webhooks include an `X-Webhook-Signature` header (HMAC-SHA256) for verification.

### Webhook Events

- `payment_intent.succeeded` / `payment.succeeded` — Payment completed
- `payment_intent.payment_failed` / `payment.failed` — Payment failed
- `charge.refunded` / `refund.succeeded` — Refund processed
- `customer.subscription.created` — Subscription created
- `customer.subscription.deleted` — Subscription canceled
- `charge.dispute.created` — Dispute opened

---

## Currency Routing

Conductor automatically routes payments to the best provider based on currency:

| Provider | Currencies |
|----------|-----------|
| Stripe | USD, EUR, GBP, CAD, AUD, JPY, SGD, HKD |
| Xendit | IDR, SGD, MYR, PHP, THB, VND |
| Razorpay | INR |

Just set the `currency` field in your charge request and Conductor handles the rest.

## Amount Format

All monetary amounts use the smallest currency unit:

- **USD/EUR/GBP**: cents (`1000` = $10.00)
- **IDR**: rupiah (`50000` = Rp 50,000)
- **SGD**: cents (`1500` = S$15.00)
- **INR**: paise (`100000` = ₹1,000.00)

## Error Codes

| Code | Meaning |
|------|---------|
| 400 | Bad request — invalid or missing parameters |
| 401 | Unauthorized — missing or invalid authentication |
| 404 | Not found — resource doesn't exist |
| 405 | Method not allowed |
| 429 | Rate limit exceeded |
| 500 | Internal server error |
| 503 | No payment provider available |

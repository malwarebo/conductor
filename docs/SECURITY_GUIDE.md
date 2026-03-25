# Security Guide

## Authentication

### JWT Tokens
- Use strong secrets (min 32 chars): `openssl rand -base64 32`
- Short-lived access tokens (15 min recommended)
- Validate signature and expiration on every request

### API Keys
- Generated with 32 random bytes
- Stored as hashed values, never plaintext
- Support scoped permissions per key

### Headers
```
Authorization: Bearer <jwt-token>
X-API-Key: <api-key>
```

## Data Protection

### Encryption
- TLS 1.2+ required for all connections
- Database SSL mode: `require`
- Sensitive fields encrypted at application level (AES-256-GCM)

### PII Handling
- Card numbers never stored (tokenized via providers)
- Fraud detection anonymizes data before external API calls
- Audit logs mask sensitive fields

## API Security

### Rate Limiting

| Tier | RPS | Burst |
|------|-----|-------|
| Default | 10 | 20 |
| Standard | 50 | 100 |
| Premium | 100 | 200 |

### Input Validation
- All requests validated before processing
- Amount limits enforced per currency
- GORM parameterized queries prevent SQL injection

### Headers Set
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

## Database

### Access Control
```sql
CREATE USER conductor_app WITH PASSWORD '...';
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO conductor_app;
REVOKE CREATE, DROP ON DATABASE conductor_prod FROM conductor_app;
```

### Connection
- Max 100 open connections
- Connection lifetime: 1 hour
- SSL required in production

## Infrastructure

### Docker
```dockerfile
FROM alpine:latest
RUN adduser -D conductor
USER conductor
```
- Run as non-root user
- Read-only root filesystem
- Drop all capabilities

### Environment Variables
```bash
# Never commit to version control
JWT_SECRET=...
DB_PASSWORD=...
STRIPE_SECRET_KEY=...
```

## Webhooks

### Inbound (from providers)
- Stripe: Verify `Stripe-Signature` header
- Xendit: Verify `x-callback-token` header
- Razorpay: Verify `X-Razorpay-Signature` (HMAC-SHA256)

### Outbound (to tenants)
- Include `X-Webhook-Signature` header (HMAC-SHA256)
- Tenants verify using their webhook secret

## Audit Logging

All sensitive operations logged:
- Payment create/capture/refund
- Subscription changes
- Customer data access
- Configuration changes

Query via `GET /v1/audit-logs`

## Checklist

- [ ] JWT secret is 32+ random characters
- [ ] Database SSL enabled
- [ ] API keys hashed before storage
- [ ] Rate limiting enabled
- [ ] Webhook signatures verified
- [ ] Running as non-root user
- [ ] Environment variables for secrets
- [ ] Audit logging enabled

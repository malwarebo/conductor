# Architecture

## Interactive Diagram

```bash
make diagram
```

Opens visualization at http://localhost:9090 - click components to explore code examples.

## Overview

Conductor is a payment switch providing a unified interface for multiple payment providers while maintaining state and data consistency.

![Architecture](/assets/conductor_arch.png)

## Components

### API
- REST endpoints for payments, subscriptions, disputes, plans
- JWT/API key authentication
- Rate limiting (tiered)

### Services
- **Payment**: Charges, refunds, captures
- **Subscription**: Recurring billing, plan management
- **Dispute**: Evidence handling, resolution tracking
- **Fraud**: AI-powered transaction analysis

### Stores
- GORM-based data access
- Transaction management
- Entity relationships

### Providers
- Abstract `PaymentProvider` interface
- Stripe, Xendit, Razorpay, Airwallex implementations
- Smart routing via `MultiProviderSelector`

### Data
- PostgreSQL (GORM ORM)
- Redis (caching, rate limiting)
- Auto-migrations

## Data Models

| Entity | Purpose |
|--------|---------|
| Payment | Transaction details, status, provider info, refund history |
| Subscription | Recurring billing, plan reference, payment history |
| Dispute | Evidence, resolution tracking, related transaction |
| Plan | Pricing, billing interval, features |
| Customer | Profile, payment methods |

## Request Flow

```
HTTP → Handler → Service → Store → Database
                       ↘ Provider → External API
```

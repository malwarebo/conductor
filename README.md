# Gopay - Payment Orchestration System

Gopay is an open-source payment orchestration system that supports multiple payment providers (Stripe and Xendit) with features for payment processing, subscriptions, and dispute management.

## Features

- **Multi-provider Support**
  - Stripe and Xendit integration
  - Automatic provider failover
  - Easy addition of new providers

- **Payment Processing**
  - Charge processing
  - Refund handling
  - Transaction management

- **Subscription Management**
  - Plan creation and management
  - Subscription lifecycle handling
  - Trial period support
  - Automatic billing

- **Dispute Handling**
  - Dispute creation and management
  - Evidence submission
  - Status tracking
  - Dispute statistics

## Prerequisites

- Go 1.19 or higher
- PostgreSQL (for data storage)
- Stripe account and API key
- Xendit account and API key

## Installation

1. Clone the repository:
```bash
git clone https://github.com/malwarebo/gopay.git
cd gopay
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
export STRIPE_API_KEY="your_stripe_api_key"
export XENDIT_API_KEY="your_xendit_api_key"
export PORT="8080"  # Optional, defaults to 8080
```

## Running the Service

1. Start the server:
```bash
go run main.go
```

The service will start on the configured port (default: 8080).

## API Documentation

### Payment Endpoints

#### Charge Payment
```http
POST /charge
Content-Type: application/json

{
  "amount": 1000,
  "currency": "USD",
  "payment_method": "card",
  "description": "Test charge",
  "customer_id": "cust_123",
  "metadata": {
    "order_id": "ord_123"
  }
}
```

#### Refund Payment
```http
POST /refund
Content-Type: application/json

{
  "charge_id": "ch_123",
  "amount": 1000,
  "reason": "customer_request"
}
```

### Subscription Endpoints

#### Create Plan
```http
POST /plans
Content-Type: application/json

{
  "name": "Premium Plan",
  "amount": 2999,
  "currency": "USD",
  "interval": "month",
  "trial_days": 14,
  "metadata": {
    "features": "all"
  }
}
```

#### Create Subscription
```http
POST /subscriptions
Content-Type: application/json

{
  "customer_id": "cust_123",
  "plan_id": "plan_123",
  "trial_days": 14,
  "metadata": {
    "source": "web"
  }
}
```

#### Update Subscription
```http
PUT /subscriptions/{subscription_id}
Content-Type: application/json

{
  "plan_id": "new_plan_123",
  "prorate": true
}
```

#### Cancel Subscription
```http
DELETE /subscriptions/{subscription_id}
Content-Type: application/json

{
  "at_period_end": true
}
```

### Dispute Endpoints

#### Create Dispute
```http
POST /disputes
Content-Type: application/json

{
  "transaction_id": "ch_123",
  "amount": 1000,
  "currency": "USD",
  "reason": "fraudulent",
  "evidence": {
    "product_description": "Digital product",
    "customer_email_address": "customer@example.com"
  }
}
```

#### Submit Evidence
```http
POST /disputes/{dispute_id}/evidence
Content-Type: application/json

{
  "type": "shipping_documentation",
  "description": "Proof of delivery",
  "files": ["file_123"]
}
```

#### Get Dispute Statistics
```http
GET /disputes/stats
```

## Error Handling

The API returns standard HTTP status codes:

- 200: Success
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 404: Not Found
- 500: Internal Server Error

Error responses include a message:
```json
{
  "error": "Detailed error message"
}
```

## Development

### Adding a New Provider

1. Implement the `PaymentProvider` interface in `providers/provider.go`
2. Add provider configuration in `config/config.go`
3. Initialize the provider in `main.go`

### Running Tests
```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

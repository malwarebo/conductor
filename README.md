# Gopay - Modern Payment Orchestration System

A flexible payment orchestration system that supports multiple payment gateways (Stripe and Xendit) with automatic failover.

## Features

- Single unified API for multiple payment gateways
- Automatic failover between payment providers
- Support for charges and refunds
- Easy to extend with new payment providers
- Built with Go for high performance and reliability
- Configuration via JSON and environment variables

## Configuration

The system can be configured using either a JSON configuration file (`config/config.json`) or environment variables. Environment variables take precedence over the config file.

### Configuration File
```json
{
  "xendit": {
    "secret": "xnd_development_...",
    "public": "xnd_public_development_..."
  },
  "stripe": {
    "secret": "sk_test_...",
    "public": "pk_test_..."
  },
  "server": {
    "port": "8080"
  }
}
```

### Environment Variables
```bash
export STRIPE_API_KEY=your_stripe_secret_key
export XENDIT_API_KEY=your_xendit_secret_key
export PORT=8080  # Optional, defaults to 8080
```

## Setup

1. Clone the repository:
```bash
git clone https://github.com/malwarebo/gopay.git
```

2. Configure the application using either:
   - Update `config/config.json` with your API keys
   - Set environment variables (these will override config.json values)

3. Run the server:
```bash
go run main.go
```

## API Endpoints

### Charge Payment
```http
POST /charge
Content-Type: application/json

{
    "amount": 1000.00,
    "currency": "USD",
    "payment_method": "pm_card_visa",
    "description": "Test charge",
    "customer_id": "cust_123",
    "metadata": {
        "order_id": "ord_123"
    }
}
```

### Process Refund
```http
POST /refund
Content-Type: application/json

{
    "transaction_id": "ch_123",
    "amount": 1000.00,
    "reason": "customer_requested",
    "metadata": {
        "refund_reason": "product_defect"
    }
}
```

## Architecture

The system is designed with the following components:

1. **Provider Interface**: Common interface for all payment providers
2. **Payment Service**: Orchestrates payments across providers
3. **API Handlers**: RESTful endpoints for payment operations
4. **Configuration**: Flexible configuration via JSON and environment variables

## Adding New Providers

To add a new payment provider:

1. Implement the `PaymentProvider` interface in `providers/`
2. Add the provider configuration in `config/config.json`
3. Add the provider to `PaymentService` in `main.go`

## Error Handling

The system handles various error scenarios:
- Provider unavailability
- Invalid requests
- Payment failures
- Configuration errors

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
